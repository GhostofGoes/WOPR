// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

//go:build linux

package sandbox

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// The process tests run the real Restrict in a child process, the test binary itself
// started again with one test selected, since a sandbox cannot be undone. This checks what
// the one-thread tests cannot: that AllThreadsSyscall reaches every thread the runtime has,
// that TSYNC puts the filter on all of them, and that threads made afterwards inherit it all.
// It starts the child with os.StartProcess: the child is this test binary, and archtest
// keeps os/exec, the package for running other programs, out of internal/sandbox.

const (
	childEnv   = "WOPR_SANDBOX_CHILD" // the fixture, as JSON: set only in the child
	readyLine  = "PROBE-READY"
	doneLine   = "PROBE-DONE" // missing if an exec replaced the child
	newThreads = 16
)

// childReport is what the child prints, as one JSON line.
type childReport struct {
	Log        string
	ABI        int
	CapsBefore bool // the child had capabilities before Restrict
	Zone       string
	Threads    []threadReport
	Signal     string // a SIGWINCH sent to the process itself, received through signal.Notify
	GroupStop  string // Ctrl+Z's kill(0, SIGTSTP) to its own process group, caught the same way
	Runtime    string // GC, freeing memory, timers and goroutines after the lockdown
	Kept       map[string]string
	TasksPre   []int // the threads that existed before Restrict
	ExecProbe  string
}

type threadReport struct {
	Tid            int
	NoNewPrivs     int
	Seccomp        int
	CapsZero       bool
	Files, Sockets map[string]string
	TCP            map[string]string
	Abstract, Proc string
	Kept           map[string]string
	Filter         map[string]string // filterProbes: refused with EPERM on every thread
	RuntimeLocked  bool
}

// TestRestrictProcessChild is the child's side: it does nothing in a normal run.
func TestRestrictProcessChild(t *testing.T) {
	cfg := os.Getenv(childEnv)
	if cfg == "" {
		t.Skip("runs only as TestRestrictProcess's child")
	}
	var f fixture
	if err := json.Unmarshal([]byte(cfg), &f); err != nil {
		panic(err)
	}
	// Before the lockdown, as wopr: open the files it will keep using, note the threads.
	var err error
	if f.kept, err = os.OpenFile(f.path("kept"), os.O_WRONLY|os.O_APPEND, 0); err != nil {
		panic(err)
	}
	f.pty = os.NewFile(3, "pty") // the parent passes the terminal side of a pty as fd 3
	rep := childReport{TasksPre: tasks()}
	v, _, _ := unix.Syscall(unix.SYS_LANDLOCK_CREATE_RULESET, 0, 0, unix.LANDLOCK_CREATE_RULESET_VERSION)
	rep.ABI = int(v)
	hdr := unix.CapUserHeader{Version: unix.LINUX_CAPABILITY_VERSION_3}
	var caps [2]unix.CapUserData
	rep.CapsBefore = unix.Capget(&hdr, &caps[0]) == nil && caps != noCaps
	winch, tstp := make(chan os.Signal, 1), make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	signal.Notify(tstp, syscall.SIGTSTP) // caught, so it does not stop the child

	rep.Log = Restrict()

	rep.Zone, _ = time.Now().Zone()
	rep.Kept = run(keptProbes(f))
	// The main thread, then threads made after the lockdown: each goroutine locks a thread
	// of its own and holds it until all have reported, so that there are newThreads of them.
	var wg sync.WaitGroup
	var mu sync.Mutex
	hold := make(chan struct{})
	probe := func(locked bool) {
		r := probeThread(f)
		r.RuntimeLocked = locked
		mu.Lock()
		rep.Threads = append(rep.Threads, r)
		mu.Unlock()
	}
	runtime.LockOSThread()
	probe(false)
	for range newThreads {
		wg.Add(1)
		go func() {
			runtime.LockOSThread()
			probe(true)
			wg.Done()
			<-hold
		}()
	}
	wg.Wait()

	rep.Signal = sendAndCatch(unix.Getpid(), unix.SIGWINCH, winch)
	rep.GroupStop = sendAndCatch(0, unix.SIGTSTP, tstp) // the child has a process group of its own
	rep.Runtime = exerciseRuntime()
	rep.ExecProbe = "not tried"
	if len(rep.Threads) > 0 && refused(rep.Threads[0].Sockets["open a UDP socket"]) {
		rep.ExecProbe = outcome(syscall.Exec("/bin/true", []string{"true"}, nil))
	}
	out, _ := json.Marshal(rep)
	fmt.Printf("%s\n%s\n", out, readyLine)
	_, _ = os.Stdin.Read(make([]byte, 1)) // the parent reads /proc, then closes stdin
	close(hold)
	fmt.Println(doneLine)
	// syscall.Exit, not a return: the test framework would write coverage and gobco's
	// counters at exit, and the sandbox refuses those files.
	syscall.Exit(0)
}

// sendAndCatch sends sig with kill(pid, sig) and waits for it on c.
func sendAndCatch(pid int, sig syscall.Signal, c chan os.Signal) string {
	if err := unix.Kill(pid, sig); err != nil {
		return outcome(err)
	}
	select {
	case <-c:
		return "ok"
	case <-time.After(5 * time.Second):
		return "not received"
	}
}

func probeThread(f fixture) threadReport {
	r := threadReport{Tid: unix.Gettid()}
	r.NoNewPrivs, _ = unix.PrctlRetInt(unix.PR_GET_NO_NEW_PRIVS, 0, 0, 0, 0)
	r.Seccomp, _ = unix.PrctlRetInt(unix.PR_GET_SECCOMP, 0, 0, 0, 0)
	hdr := unix.CapUserHeader{Version: unix.LINUX_CAPABILITY_VERSION_3}
	var caps [2]unix.CapUserData
	r.CapsZero = unix.Capget(&hdr, &caps[0]) == nil && caps == noCaps
	files := fileProbes(f)
	r.Files = run(map[string]func() error{"read a file": files["read a file"], "create a file": files["create a file"], "list a folder": files["list a folder"]})
	r.TCP = run(tcpProbes(f))
	r.Sockets = run(socketProbes(f))
	r.Abstract = outcome(abstractProbe(f))
	r.Proc = outcome(func() error { _, err := os.ReadFile("/proc/self/status"); return err }())
	r.Filter = run(filterProbes(f))
	r.Kept = run(map[string]func() error{"write a file opened before": keptProbes(f)["write a file opened before"], "signal this thread": keptProbes(f)["signal this thread"]})
	return r
}

// exerciseRuntime makes the Go runtime do, after the lockdown, what a long session does.
func exerciseRuntime() string {
	big := make([][]byte, 0, 64)
	for range 64 {
		big = append(big, make([]byte, 1<<20)) // new spans: mmap
	}
	runtime.GC()
	debug.FreeOSMemory() // madvise
	_ = big
	results := make(chan int, 100)
	for i := range 100 {
		go func() { time.Sleep(time.Millisecond); results <- i }() // timers and goroutines
	}
	sum := 0
	for range 100 {
		sum += <-results
	}
	if sum != 4950 {
		return "goroutines lost"
	}
	r, w, err := os.Pipe() // netpoll: the pipe is registered with epoll
	if err != nil {
		return outcome(err)
	}
	go func() { _, _ = w.WriteString("ping"); _ = w.Close() }()
	if err := r.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return outcome(err)
	}
	buf := make([]byte, 4)
	if _, err := r.Read(buf); err != nil || string(buf) != "ping" {
		return fmt.Sprintf("pipe: %q %v", buf, err)
	}
	return "ok"
}

func tasks() []int {
	entries, _ := os.ReadDir("/proc/self/task")
	var ids []int
	for _, e := range entries {
		if n, err := strconv.Atoi(e.Name()); err == nil {
			ids = append(ids, n)
		}
	}
	return ids
}

// TestRestrictProcess starts the child and checks its report and, from outside, the status
// of every one of its threads.
func TestRestrictProcess(t *testing.T) {
	if os.Getenv(childEnv) != "" {
		t.Skip("the child runs only TestRestrictProcessChild")
	}
	abi := landlockABI(t)
	f := newFixture(t)
	t.Run("as this user", func(t *testing.T) { checkProcess(t, f, abi, nil) })
	if os.Geteuid() != 0 {
		// In a user namespace the child holds every capability, so the drop is tested too.
		t.Run("with capabilities", func(t *testing.T) {
			checkProcess(t, f, abi, &syscall.SysProcAttr{
				Cloneflags:  syscall.CLONE_NEWUSER,
				UidMappings: []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getuid(), Size: 1}},
				GidMappings: []syscall.SysProcIDMap{{ContainerID: 0, HostID: os.Getgid(), Size: 1}},
			})
		})
	}
}

func checkProcess(t *testing.T, f fixture, abi int, sys *syscall.SysProcAttr) {
	t.Helper()
	cfg, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if sys == nil {
		sys = &syscall.SysProcAttr{}
	}
	sys.Setpgid = true // its own process group: Ctrl+Z's kill(0, ...) must stay inside it
	args := []string{os.Args[0], "-test.run=^TestRestrictProcessChild$", "-test.count=1", "-test.timeout=60s"}
	p, err := os.StartProcess(os.Args[0], args, &os.ProcAttr{
		Env:   append(os.Environ(), childEnv+"="+string(cfg), "TZ=America/New_York"),
		Files: []*os.File{stdinR, stdoutW, os.Stderr, f.pty},
		Sys:   sys,
	})
	_ = stdinR.Close()
	_ = stdoutW.Close()
	if err != nil {
		if sys.Cloneflags != 0 {
			t.Skipf("no user namespace here: %v", err)
		}
		t.Fatal(err)
	}
	killed := time.AfterFunc(60*time.Second, func() { _ = p.Kill() })
	defer killed.Stop()

	lines := bufio.NewScanner(stdoutR)
	lines.Buffer(make([]byte, 1<<20), 1<<20)
	var rep childReport
	ready := false
	for lines.Scan() {
		line := lines.Text()
		if strings.HasPrefix(line, "{") {
			if err := json.Unmarshal([]byte(line), &rep); err != nil {
				t.Fatalf("the child's report: %v", err)
			}
		}
		if line == readyLine {
			ready = true
			break
		}
	}
	if !ready {
		_ = stdinW.Close()
		st, _ := p.Wait()
		t.Fatalf("the child never reported (%v)", st)
	}
	statuses := taskStatuses(t, p.Pid)
	_ = stdinW.Close()
	done := false
	for lines.Scan() {
		if lines.Text() == doneLine {
			done = true
		}
	}
	st, err := p.Wait()
	if err != nil || !st.Success() || !done {
		t.Fatalf("the child: %v, %v, done line %v: an exec may have replaced it", st, err, done)
	}
	checkReport(t, rep, abi)
	checkStatuses(t, statuses, rep)
	if got, err := os.ReadFile(f.path("kept")); err != nil || !strings.Contains(string(got), "written in the sandbox") {
		t.Errorf("the child's writes through a file opened before are missing: %q, %v", got, err)
	}
	if sys.Cloneflags != 0 && !rep.CapsBefore {
		// Ubuntu's AppArmor gives an unprivileged user namespace no capabilities.
		t.Skip("the user namespace gave the child no capabilities, so none were dropped")
	}
}

var statusField = regexp.MustCompile(`(?m)^(NoNewPrivs|Seccomp|Seccomp_filters|CapEff|CapPrm|CapInh|CapAmb):\s+(\S+)$`)

// taskStatuses reads /proc/<pid>/task/*/status for every thread of the child.
func taskStatuses(t *testing.T, pid int) map[string]map[string]string {
	t.Helper()
	dir := filepath.Join("/proc", strconv.Itoa(pid), "task")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]map[string]string{}
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(dir, e.Name(), "status"))
		if err != nil {
			continue // the thread ended
		}
		fields := map[string]string{}
		for _, m := range statusField.FindAllStringSubmatch(string(data), -1) {
			fields[m[1]] = m[2]
		}
		out[e.Name()] = fields
	}
	return out
}

func checkReport(t *testing.T, rep childReport, abi int) {
	t.Helper()
	cgo := strings.HasPrefix(rep.Log, "sandbox: cgo build:")
	switch {
	case cgo:
		if os.Getenv("WOPR_SANDBOX_REQUIRE") != "" {
			t.Errorf("the test binary uses cgo, so Landlock is off: %q", rep.Log)
		}
		t.Logf("cgo build: checking no_new_privs and seccomp only: %q", rep.Log)
	case !strings.HasPrefix(rep.Log, "sandbox: no_new_privs") || !strings.HasSuffix(rep.Log, fmt.Sprintf("landlock ABI %d, seccomp", abi)):
		t.Fatalf("Restrict = %q", rep.Log)
	}
	if rep.CapsBefore && !cgo && !strings.Contains(rep.Log, "capabilities dropped") {
		t.Errorf("the child had capabilities, but Restrict = %q", rep.Log)
	}
	if rep.Zone != "EDT" && rep.Zone != "EST" {
		t.Errorf("time zone %q: /etc/localtime was not loaded before the lockdown", rep.Zone)
	}
	if rep.Signal != "ok" || rep.GroupStop != "ok" || rep.Runtime != "ok" {
		t.Errorf("after the lockdown: signal %s, Ctrl+Z's stop %s, runtime %s", rep.Signal, rep.GroupStop, rep.Runtime)
	}
	if rep.ExecProbe != "EPERM" {
		t.Errorf("exec after the lockdown: %s, want EPERM", rep.ExecProbe)
	}
	for name, o := range rep.Kept {
		if o != "ok" {
			t.Errorf("%s: %s, want it to keep working", name, o)
		}
	}
	pre := map[int]bool{}
	for _, id := range rep.TasksPre {
		pre[id] = true
	}
	fresh := 0
	for _, th := range rep.Threads {
		if !pre[th.Tid] {
			fresh++
		}
		where := fmt.Sprintf("thread %d (made after the lockdown: %v)", th.Tid, !pre[th.Tid])
		if th.NoNewPrivs != 1 || th.Seccomp != 2 {
			t.Errorf("%s: no_new_privs %d, seccomp %d", where, th.NoNewPrivs, th.Seccomp)
		}
		if !cgo && !th.CapsZero {
			t.Errorf("%s: capabilities left", where)
		}
		for name, o := range th.Sockets {
			if o != "EPERM" {
				t.Errorf("%s: %s: %s, want EPERM", where, name, o)
			}
		}
		for name, o := range th.Kept {
			if o != "ok" {
				t.Errorf("%s: %s: %s", where, name, o)
			}
		}
		for name, o := range th.Filter {
			if want := filterErrno(name, "EPERM"); o != want {
				t.Errorf("%s: %s: %s, want %s", where, name, o, want)
			}
		}
		if len(th.Filter) < 30 {
			t.Errorf("%s: only %d filter probes ran", where, len(th.Filter))
		}
		if cgo {
			continue
		}
		for name, o := range th.Files {
			if o != "EACCES" {
				t.Errorf("%s: %s: %s, want EACCES", where, name, o)
			}
		}
		for name, o := range th.TCP {
			if !refused(o) {
				t.Errorf("%s: %s: %s, want it refused", where, name, o)
			}
		}
		if !refused(th.Abstract) || th.Proc != "EACCES" {
			t.Errorf("%s: abstract socket %s, /proc %s; want both refused", where, th.Abstract, th.Proc)
		}
	}
	if len(rep.Threads) != newThreads+1 {
		t.Errorf("%d threads reported, want %d", len(rep.Threads), newThreads+1)
	}
	if fresh == 0 {
		t.Error("no thread was made after the lockdown, so inheritance was not tested")
	}
}

// checkStatuses checks, from outside, every thread the child had: the runtime's own too.
func checkStatuses(t *testing.T, statuses map[string]map[string]string, rep childReport) {
	t.Helper()
	cgo := strings.HasPrefix(rep.Log, "sandbox: cgo build:")
	if len(statuses) < newThreads+1 {
		t.Errorf("the child has %d threads, want at least %d", len(statuses), newThreads+1)
	}
	// The child inherits this process's filters (a container's, say), and adds its own.
	want := strconv.Itoa(ownFilters(t) + 1)
	for tid, st := range statuses {
		if st["NoNewPrivs"] != "1" || st["Seccomp"] != "2" || st["Seccomp_filters"] != want {
			t.Errorf("thread %s: NoNewPrivs %s, Seccomp %s, Seccomp_filters %s; want 1, 2, %s",
				tid, st["NoNewPrivs"], st["Seccomp"], st["Seccomp_filters"], want)
		}
		if cgo {
			continue
		}
		for _, c := range []string{"CapEff", "CapPrm", "CapInh", "CapAmb"} {
			if strings.Trim(st[c], "0") != "" {
				t.Errorf("thread %s: %s %s, want none", tid, c, st[c])
			}
		}
	}
}

// ownFilters is the number of seccomp filters on this process.
func ownFilters(t *testing.T) int {
	t.Helper()
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range statusField.FindAllStringSubmatch(string(data), -1) {
		if m[1] == "Seccomp_filters" {
			n, err := strconv.Atoi(m[2])
			if err != nil {
				t.Fatal(err)
			}
			return n
		}
	}
	return 0 // a kernel before 5.9 does not show the count
}
