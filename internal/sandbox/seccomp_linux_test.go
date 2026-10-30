// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

//go:build linux

package sandbox

import (
	"fmt"
	"runtime"
	"slices"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

// seccompData is the kernel's struct seccomp_data, as the filter sees it.
type seccompData struct {
	nr   uint32
	arch uint32
	args [6]uint64
}

// word reads the 32-bit word at off, as BPF_LD|BPF_W|BPF_ABS does on a little-endian system.
func (d seccompData) word(t *testing.T, off uint32) uint32 {
	t.Helper()
	switch {
	case off == offNr:
		return d.nr
	case off == offArch:
		return d.arch
	case off >= offArg0 && off < offArg0+48 && off%4 == 0:
		a := d.args[(off-offArg0)/8]
		if (off-offArg0)%8 == 0 {
			return uint32(a)
		}
		return uint32(a >> 32)
	}
	t.Fatalf("load from offset %d", off)
	return 0
}

// interpret runs a classic BPF program the way the kernel's seccomp does, for the
// instructions the filter uses, and fails on any other or on a jump out of the program.
func interpret(t *testing.T, prog []unix.SockFilter, d seccompData) uint32 {
	t.Helper()
	var a uint32
	for pc := 0; pc < len(prog); pc++ {
		in := prog[pc]
		switch in.Code {
		case ldAbs:
			a = d.word(t, in.K)
		case jeq, jge, jset:
			var hit bool
			switch in.Code {
			case jeq:
				hit = a == in.K
			case jge:
				hit = a >= in.K
			default:
				hit = a&in.K != 0
			}
			if hit {
				pc += int(in.Jt)
			} else {
				pc += int(in.Jf)
			}
			if pc+1 >= len(prog) {
				t.Fatalf("jump past the end from %d", pc)
			}
		case ret:
			return in.K
		default:
			t.Fatalf("instruction %d: unexpected code %#x", pc, in.Code)
		}
	}
	t.Fatal("the program ran off its end")
	return 0
}

const testPid = 4242

// mustAllow is every call that wopr makes after the lockdown (strace of real sessions:
// chess with the AI thinking, movie mode, resize, Ctrl+Z and SIGCONT, Ctrl+C, the debug log)
// and that the Go runtime may make at any time.
var mustAllow = map[string]map[string]uint32{
	"amd64": {
		"read": 0, "write": 1, "close": 3, "fstat": 5, "lseek": 8, "mmap": 9, "mprotect": 10, "munmap": 11,
		"brk": 12, "rt_sigaction": 13, "rt_sigprocmask": 14, "rt_sigreturn": 15, "pread64": 17,
		"sched_yield": 24, "mincore": 27, "madvise": 28, "nanosleep": 35, "getpid": 39, "exit": 60,
		"fcntl": 72, "sigaltstack": 131, "prctl": 157, "arch_prctl": 158, "gettid": 186, "futex": 202,
		"sched_getaffinity": 204, "clock_gettime": 228, "exit_group": 231, "epoll_ctl": 233,
		"openat": 257, "epoll_pwait": 281, "eventfd2": 290, "epoll_create1": 291, "pipe2": 293,
		"getrandom": 318, "rseq": 334, "restart_syscall": 219, "uname": 63, "pidfd_open": 434,
	},
	"arm64": {
		"read": 63, "write": 64, "close": 57, "fstat": 80, "lseek": 62, "mmap": 222, "mprotect": 226,
		"munmap": 215, "brk": 214, "rt_sigaction": 134, "rt_sigprocmask": 135, "rt_sigreturn": 139,
		"pread64": 67, "sched_yield": 124, "mincore": 232, "madvise": 233, "nanosleep": 101,
		"getpid": 172, "exit": 93, "fcntl": 25, "sigaltstack": 132, "prctl": 167, "gettid": 178,
		"futex": 98, "sched_getaffinity": 123, "clock_gettime": 113, "exit_group": 94, "epoll_ctl": 21,
		"openat": 56, "epoll_pwait": 22, "eventfd2": 19, "epoll_create1": 20, "pipe2": 59,
		"getrandom": 278, "rseq": 293, "restart_syscall": 128, "uname": 160, "pidfd_open": 434,
	},
}

// runtimeClone is the Go runtime's clone flags for a new thread (runtime/os_linux.go).
const runtimeClone = unix.CLONE_VM | unix.CLONE_FS | unix.CLONE_FILES | unix.CLONE_SIGHAND |
	unix.CLONE_SYSVSEM | unix.CLONE_THREAD

// TestSeccompProgram checks each architecture's filter, on any host, decision by decision.
func TestSeccompProgram(t *testing.T) {
	for _, goarch := range []string{"amd64", "arm64"} {
		t.Run(goarch, func(t *testing.T) {
			arch := seccompArches[goarch]
			prog, err := seccompProgram(arch, testPid, syscall.EPERM)
			if err != nil {
				t.Fatal(err)
			}
			if len(prog) > unix.BPF_MAXINSNS {
				t.Fatalf("%d instructions", len(prog))
			}
			deny := unix.SECCOMP_RET_ERRNO | uint32(syscall.EPERM)
			call := func(name string, args ...uint64) seccompData {
				n, ok := arch.nr[name]
				if !ok {
					n, ok = mustAllow[goarch][name]
				}
				if !ok {
					t.Fatalf("no number for %s", name)
				}
				d := seccompData{nr: n, arch: arch.audit}
				copy(d.args[:], args)
				return d
			}
			check := func(what string, d seccompData, want uint32) {
				t.Helper()
				if got := interpret(t, prog, d); got != want {
					t.Errorf("%s: %#x, want %#x", what, got, want)
				}
			}

			names := 0
			for _, name := range denied {
				if _, ok := arch.nr[name]; !ok {
					continue
				}
				names++
				check(name, call(name), deny)
			}
			if names < 60 {
				t.Errorf("only %d denied calls have a number", names)
			}
			for name := range mustAllow[goarch] {
				check(name, call(name), retAllow)
				if slices.Contains(denied, name) {
					t.Errorf("%s is both allowed and denied", name)
				}
			}

			check("ioctl TCGETS", call("ioctl", 0, unix.TCGETS), retAllow)
			check("ioctl TCSETS", call("ioctl", 0, unix.TCSETS), retAllow)
			check("ioctl TIOCGWINSZ", call("ioctl", 1, unix.TIOCGWINSZ), retAllow)
			check("ioctl TIOCSWINSZ", call("ioctl", 1, unix.TIOCSWINSZ), retAllow)
			check("ioctl TIOCSTI", call("ioctl", 0, unix.TIOCSTI), deny)
			check("ioctl TIOCSTI, high bits set", call("ioctl", 0, 1<<32|unix.TIOCSTI), deny)
			check("ioctl TIOCLINUX", call("ioctl", 0, unix.TIOCLINUX), deny)
			check("ioctl TIOCGPGRP", call("ioctl", 0, unix.TIOCGPGRP), retAllow)
			check("ioctl FIOSETOWN", call("ioctl", 3, fioSetOwn), deny)
			check("ioctl SIOCSPGRP", call("ioctl", 3, unix.SIOCSPGRP), deny)

			check("fcntl F_GETFL", call("fcntl", 0, unix.F_GETFL), retAllow)
			check("fcntl F_SETFL", call("fcntl", 0, unix.F_SETFL, unix.O_NONBLOCK), retAllow)
			check("fcntl F_DUPFD_CLOEXEC", call("fcntl", 0, unix.F_DUPFD_CLOEXEC), retAllow)
			check("fcntl F_GETOWN", call("fcntl", 0, unix.F_GETOWN), retAllow)
			check("fcntl F_SETOWN", call("fcntl", 0, unix.F_SETOWN, 1), deny)
			check("fcntl F_SETOWN_EX", call("fcntl", 0, unix.F_SETOWN_EX), deny)
			check("fcntl F_SETSIG", call("fcntl", 0, unix.F_SETSIG, uint64(unix.SIGKILL)), deny)
			check("fcntl F_SETOWN, high bits set", call("fcntl", 0, 1<<32|unix.F_SETOWN, 1), deny)

			check("prlimit64 self (0)", call("prlimit64", 0, unix.RLIMIT_NOFILE), retAllow)
			check("prlimit64 self (pid)", call("prlimit64", testPid, unix.RLIMIT_NOFILE), retAllow)
			check("prlimit64 another process", call("prlimit64", 1, unix.RLIMIT_NOFILE), deny)

			check("clone a thread", call("clone", runtimeClone|unix.CLONE_SETTLS), retAllow)
			check("clone a process (fork)", call("clone", uint64(unix.SIGCHLD)), deny)
			check("clone a process (vfork)", call("clone", unix.CLONE_VM|unix.CLONE_VFORK|uint64(unix.SIGCHLD)), deny)
			for _, ns := range []uint64{unix.CLONE_NEWUSER, unix.CLONE_NEWNS, unix.CLONE_NEWPID, unix.CLONE_NEWNET, unix.CLONE_NEWTIME} {
				check(fmt.Sprintf("clone a thread in namespace %#x", ns), call("clone", runtimeClone|ns), deny)
			}
			check("clone, flags only in the high bits", call("clone", runtimeClone|1<<40), retAllow)
			check("clone3", call("clone3"), retNosys)

			check("kill self", call("kill", testPid, 0), retAllow)
			check("kill the process group (SIGTSTP)", call("kill", 0, uint64(unix.SIGTSTP)), retAllow)
			check("kill the process group (signal 0)", call("kill", 0, 0), retAllow)
			check("kill the process group (SIGKILL)", call("kill", 0, uint64(unix.SIGKILL)), deny)
			check("kill the process group (SIGCONT)", call("kill", 0, uint64(unix.SIGCONT)), deny)
			check("kill self (SIGKILL)", call("kill", testPid, uint64(unix.SIGKILL)), retAllow)
			check("kill another process", call("kill", 1, uint64(unix.SIGKILL)), deny)
			check("kill every process", call("kill", 0xffffffff, uint64(unix.SIGKILL)), deny)
			check("kill another process group", call("kill", uint64(0xffffffff-testPid+1), uint64(unix.SIGKILL)), deny)
			check("kill self, high bits set", call("kill", 1<<32|testPid, 0), retAllow)
			check("tgkill own thread", call("tgkill", testPid, 77, uint64(unix.SIGURG)), retAllow)
			check("tgkill another process", call("tgkill", 1, 1, uint64(unix.SIGKILL)), deny)
			check("rt_sigqueueinfo self", call("rt_sigqueueinfo", testPid), retAllow)
			check("rt_sigqueueinfo another", call("rt_sigqueueinfo", 1), deny)
			check("rt_tgsigqueueinfo another", call("rt_tgsigqueueinfo", 1, 1), deny)

			foreign := call("read")
			foreign.arch = unix.AUDIT_ARCH_I386
			check("a call from another ABI", foreign, retKill)
			if arch.x32 {
				x32 := call("read")
				x32.nr |= x32Bit
				check("an x32 call", x32, retKill)
			}
			check("a call the kernel does not have", seccompData{nr: 4000, arch: arch.audit}, retAllow)
		})
	}
}

// The filter's errno is the one it is given, for every call it refuses.
func TestSeccompErrno(t *testing.T) {
	prog, err := seccompProgram(seccompArches["amd64"], testPid, unix.EHWPOISON)
	if err != nil {
		t.Fatal(err)
	}
	d := seccompData{nr: seccompArches["amd64"].nr["execve"], arch: unix.AUDIT_ARCH_X86_64}
	if got := interpret(t, prog, d); got != unix.SECCOMP_RET_ERRNO|uint32(unix.EHWPOISON) {
		t.Errorf("execve: %#x", got)
	}
}

func TestAssemble(t *testing.T) {
	if _, err := assemble([]insn{jump(jeq, 0, "nowhere", "")}); err == nil {
		t.Error("a jump to a missing label assembled")
	}
	back := []insn{labelled("top", stmt(ret, 0)), jump(jeq, 0, "top", "")}
	if _, err := assemble(back); err == nil {
		t.Error("a backward jump assembled")
	}
	far := []insn{jump(jeq, 0, "far", "")}
	for range 256 {
		far = append(far, stmt(ret, 0))
	}
	far = append(far, labelled("far", stmt(ret, 0)))
	if _, err := assemble(far); err == nil {
		t.Error("a jump over 256 instructions assembled")
	}
	long := make([]insn, unix.BPF_MAXINSNS+1)
	for i := range long {
		long[i] = stmt(ret, 0)
	}
	if _, err := assemble(long); err == nil {
		t.Error("a program over BPF_MAXINSNS assembled")
	}
}

// TestSeccompOnOneThread installs the real filter on one locked thread, with an errno that
// the kernel never gives for these calls, so that every refusal is the filter's own.
func TestSeccompOnOneThread(t *testing.T) {
	if _, ok := seccompArches[runtime.GOARCH]; !ok {
		t.Skipf("no filter for %s", runtime.GOARCH)
	}
	f := newFixture(t)
	type result struct {
		install          error
		refused, allowed map[string]string
		before, after    int
	}
	got := func() result {
		done := make(chan result)
		go func() {
			runtime.LockOSThread() // never unlocked: the filtered thread ends with the goroutine
			var r result
			r.before, _ = unix.PrctlRetInt(unix.PR_GET_SECCOMP, 0, 0, 0, 0)
			if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
				r.install = err
				done <- r
				return
			}
			r.install = installSeccomp(false, unix.EHWPOISON)
			r.after, _ = unix.PrctlRetInt(unix.PR_GET_SECCOMP, 0, 0, 0, 0)
			r.refused = run(filterProbes(f))
			r.allowed = run(keptProbes(f))
			done <- r
		}()
		return <-done
	}()
	if got.install != nil {
		skipOrFail(t, fmt.Sprintf("seccomp is not available: %v", got.install))
	}
	// A container's runtime can have put a filter on the test before it started, so the
	// mode before is 0 or 2.
	if got.after != 2 {
		t.Errorf("PR_GET_SECCOMP %d before, %d after; want 2 after", got.before, got.after)
	}
	for name, o := range got.refused {
		if want := filterErrno(name, "EHWPOISON"); o != want {
			t.Errorf("%s: %s, want %s from the filter", name, o, want)
		}
	}
	for name, o := range got.allowed {
		if o != "ok" {
			t.Errorf("%s: %s, want it allowed", name, o)
		}
	}
	if s, _ := unix.PrctlRetInt(unix.PR_GET_SECCOMP, 0, 0, 0, 0); s != got.before {
		t.Errorf("the test's own thread has seccomp %d, %d before: the filter leaked", s, got.before)
	}
}

// filterErrno is what the filter answers a filter probe with, when it refuses with deny.
func filterErrno(probe, deny string) string {
	if probe == "clone3" {
		return "ENOSYS"
	}
	return deny
}

// TestSeccompNumbers checks the host architecture's table and the must-allow list against
// golang.org/x/sys/unix. CI runs it on amd64 and arm64, so both tables are checked.
func TestSeccompNumbers(t *testing.T) {
	arch, ok := seccompArches[runtime.GOARCH]
	if !ok {
		t.Skipf("no table for %s", runtime.GOARCH)
	}
	if len(arch.nr) != len(hostNumbers) {
		t.Errorf("the table names %d calls, x/sys/unix %d", len(arch.nr), len(hostNumbers))
	}
	for name, want := range hostNumbers {
		if got, ok := arch.nr[name]; !ok || uintptr(got) != want {
			t.Errorf("%s: table %d (present %v), x/sys/unix %d", name, got, ok, want)
		}
	}
	for name, want := range hostAllowNumbers {
		if got := mustAllow[runtime.GOARCH][name]; uintptr(got) != want {
			t.Errorf("allowed %s: %d, x/sys/unix %d", name, got, want)
		}
	}
	for _, name := range denied {
		_, inAmd64 := seccompArches["amd64"].nr[name]
		_, inArm64 := seccompArches["arm64"].nr[name]
		if !inAmd64 && !inArm64 {
			t.Errorf("denied call %s has no number on any architecture", name)
		}
	}
}
