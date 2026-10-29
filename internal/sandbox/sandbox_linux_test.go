// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

//go:build linux

package sandbox

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

func landlockABI(t *testing.T) int {
	t.Helper()
	v, _, e := unix.Syscall(unix.SYS_LANDLOCK_CREATE_RULESET, 0, 0, unix.LANDLOCK_CREATE_RULESET_VERSION)
	if e != 0 {
		skipOrFail(t, fmt.Sprintf("Landlock is not available here: %v", e))
	}
	return int(v)
}

// skipOrFail skips a test the system cannot run, unless WOPR_SANDBOX_REQUIRE is set: CI sets it
// where every feature must be there, so that a missing one fails instead of passing quietly.
func skipOrFail(t *testing.T, why string) {
	t.Helper()
	if os.Getenv("WOPR_SANDBOX_REQUIRE") != "" {
		t.Fatal(why + " (WOPR_SANDBOX_REQUIRE is set)")
	}
	t.Skip(why)
}

// newFixture makes the files, listeners and pty that the probes use, before any sandbox.
func newFixture(t *testing.T) fixture {
	t.Helper()
	dir := t.TempDir()
	for _, name := range fixtureFiles {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	kept, err := os.Create(filepath.Join(dir, "kept"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = kept.Close() })
	listen := func(network, addr string) string {
		ln, err := net.Listen(network, addr)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = ln.Close() })
		return ln.Addr().String()
	}
	f := fixture{
		Dir:      dir,
		TCP:      listen("tcp4", "127.0.0.1:0"),
		UnixPath: listen("unix", filepath.Join(dir, "sock")),
		Abstract: listen("unix", "@wopr-sandbox-"+strconv.Itoa(os.Getpid())+"-"+t.Name()),
		kept:     kept,
	}
	f.pty = openPty(t)
	return f
}

// openPty opens a pty pair and returns its terminal side, as wopr's own terminal.
func openPty(t *testing.T) *os.File {
	t.Helper()
	ptm, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Skipf("no pty: %v", err)
	}
	t.Cleanup(func() { _ = ptm.Close() })
	fd := int(ptm.Fd())
	if err := unix.IoctlSetPointerInt(fd, unix.TIOCSPTLCK, 0); err != nil {
		t.Fatal(err)
	}
	n, err := unix.IoctlGetInt(fd, unix.TIOCGPTN)
	if err != nil {
		t.Fatal(err)
	}
	pts, err := os.OpenFile("/dev/pts/"+strconv.Itoa(n), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pts.Close() })
	return pts
}

// oneThread runs restrict and then fn on one locked thread. The goroutine never unlocks,
// so the runtime ends the thread with it, and the test process itself stays unrestricted.
func oneThread[T any](c calls, fn func(log string) T) T {
	done := make(chan T)
	go func() {
		runtime.LockOSThread()
		done <- fn(restrict(c))
	}()
	return <-done
}

var thread = calls{all: unix.RawSyscall6, tsync: false, deny: syscall.EPERM}

type oneThreadResult struct {
	log                  string
	files, tcp, sockets  map[string]string
	kept                 map[string]string
	abstract, exec, fork string
	caps                 [2]unix.CapUserData
	seccomp              int
}

// TestRestrict sandboxes one thread exactly as Restrict sandboxes them all, and checks that
// each layer refuses what it should and leaves what wopr needs.
func TestRestrict(t *testing.T) {
	abi := landlockABI(t)
	f := newFixture(t)
	r := oneThread(thread, func(log string) oneThreadResult {
		r := oneThreadResult{log: log}
		r.files = run(fileProbes(f, abi))
		r.tcp = run(tcpProbes(f))
		r.sockets = run(socketProbes(f))
		r.kept = run(keptProbes(f))
		r.abstract = outcome(abstractProbe(f))
		r.seccomp, _ = unix.PrctlRetInt(unix.PR_GET_SECCOMP, 0, 0, 0, 0)
		r.exec, r.fork = "not tried", "not tried"
		if refused(r.files["read a file"]) && r.seccomp == 2 { // an exec or a fork could escape only a sandbox that is not there
			r.exec = outcome(syscall.Exec("/bin/true", []string{"true"}, nil))
			r.fork = outcome(forkProbe())
		}
		hdr := unix.CapUserHeader{Version: unix.LINUX_CAPABILITY_VERSION_3}
		_ = unix.Capget(&hdr, &r.caps[0])
		return r
	})
	want := fmt.Sprintf("landlock ABI %d, seccomp", abi)
	if !strings.HasPrefix(r.log, "sandbox: no_new_privs") || !strings.HasSuffix(r.log, want) {
		t.Fatalf("restrict = %q, want no_new_privs first and %q last", r.log, want)
	}
	for name, o := range r.files {
		if o != "EACCES" {
			t.Errorf("%s: %s, want EACCES from Landlock", name, o)
		}
	}
	for name, o := range r.tcp {
		if !refused(o) {
			t.Errorf("%s: %s, want it refused", name, o)
		}
	}
	for name, o := range r.sockets {
		if o != "EPERM" {
			t.Errorf("%s: %s, want EPERM from the seccomp filter (Landlock does not cover it)", name, o)
		}
	}
	if !refused(r.abstract) {
		t.Errorf("connect an abstract UNIX socket: %s, want it refused", r.abstract)
	}
	if r.exec != "EPERM" || r.fork != "EPERM" {
		t.Errorf("run a program: %s, start a process: %s; want both EPERM", r.exec, r.fork)
	}
	for name, o := range r.kept {
		if o != "ok" {
			t.Errorf("%s: %s, want it to keep working", name, o)
		}
	}
	if r.caps != noCaps {
		t.Errorf("capabilities left: %+v", r.caps)
	}
	if r.seccomp != 2 {
		t.Errorf("PR_GET_SECCOMP = %d, want 2 (a filter)", r.seccomp)
	}
	if _, err := os.ReadFile(f.path("before")); err != nil {
		t.Errorf("only the locked thread was restricted, but the test's own read failed: %v", err)
	}
	if got, err := os.ReadFile(f.kept.Name()); err != nil || !strings.Contains(string(got), "written in the sandbox") {
		t.Errorf("the write through a file opened before is not in the file: %q, %v", got, err)
	}
}

// Each ABI handles every right it knows, and never signal scoping.
func TestRuleset(t *testing.T) {
	for _, tc := range []struct {
		abi             int
		fs, net, scoped uint64
	}{
		{1, fsV1, 0, 0},
		{2, fsV1 | unix.LANDLOCK_ACCESS_FS_REFER, 0, 0},
		{3, fsV1 | unix.LANDLOCK_ACCESS_FS_REFER | unix.LANDLOCK_ACCESS_FS_TRUNCATE, 0, 0},
		{4, fsV1 | unix.LANDLOCK_ACCESS_FS_REFER | unix.LANDLOCK_ACCESS_FS_TRUNCATE, 3, 0},
		{5, 0xffff, 3, 0},
		{6, 0xffff, 3, unix.LANDLOCK_SCOPE_ABSTRACT_UNIX_SOCKET},
		{7, 0xffff, 3, unix.LANDLOCK_SCOPE_ABSTRACT_UNIX_SOCKET},
	} {
		got := ruleset(tc.abi)
		if got.Access_fs != tc.fs || got.Access_net != tc.net || got.Scoped != tc.scoped {
			t.Errorf("ABI %d: %+v, want fs %#x net %#x scoped %#x", tc.abi, got, tc.fs, tc.net, tc.scoped)
		}
	}
}

// A cgo build gets ENOTSUP from AllThreadsSyscall. It keeps no_new_privs and the seccomp
// filter, which need no call on every thread, and says what it could not do.
func TestRestrictCgoBuild(t *testing.T) {
	landlockABI(t)
	f := newFixture(t)
	notSupported := func(_, _, _, _, _, _, _ uintptr) (uintptr, uintptr, syscall.Errno) {
		return 0, 0, syscall.ENOTSUP
	}
	type result struct {
		log, read, socket string
		nnp, seccomp      int
	}
	r := oneThread(calls{all: notSupported, deny: syscall.EPERM}, func(log string) result {
		r := result{log: log}
		r.read = run(fileProbes(f, 1))["read a file"]
		r.socket = run(socketProbes(f))["open a UDP socket"]
		r.nnp, _ = unix.PrctlRetInt(unix.PR_GET_NO_NEW_PRIVS, 0, 0, 0, 0)
		r.seccomp, _ = unix.PrctlRetInt(unix.PR_GET_SECCOMP, 0, 0, 0, 0)
		return r
	})
	if r.log != "sandbox: cgo build: no_new_privs, seccomp, no capability drop or landlock" {
		t.Errorf("restrict = %q", r.log)
	}
	if r.nnp != 1 || r.seccomp != 2 || r.socket != "EPERM" {
		t.Errorf("no_new_privs %d, seccomp %d, a socket %s; want 1, 2 and EPERM", r.nnp, r.seccomp, r.socket)
	}
	if r.read != "ok" {
		t.Errorf("a cgo build has no Landlock, yet reading a file gave %s", r.read)
	}
}

// Any other failure of the first call turns the sandbox off and says why.
func TestRestrictOff(t *testing.T) {
	broken := func(_, _, _, _, _, _, _ uintptr) (uintptr, uintptr, syscall.Errno) {
		return 0, 0, syscall.EINVAL
	}
	if got := restrict(calls{all: broken}); got != "sandbox: off (no_new_privs: invalid argument)" {
		t.Errorf("restrict = %q", got)
	}
}

// A capability drop that fails is reported, and the other layers still go in.
func TestRestrictCapsKept(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("needs capabilities to drop")
	}
	landlockABI(t)
	capsetFails := func(trap, a1, a2, a3, a4, a5, a6 uintptr) (uintptr, uintptr, syscall.Errno) {
		if trap == unix.SYS_CAPSET {
			return 0, 0, syscall.EPERM
		}
		return unix.RawSyscall6(trap, a1, a2, a3, a4, a5, a6)
	}
	got := oneThread(calls{all: capsetFails, deny: syscall.EPERM}, func(log string) string { return log })
	if !strings.Contains(got, "capabilities kept (operation not permitted)") || !strings.HasSuffix(got, ", seccomp") {
		t.Errorf("restrict = %q", got)
	}
}

// A Landlock that fails is reported, and the other layers still go in.
func TestRestrictNoLandlock(t *testing.T) {
	landlockABI(t)
	f := newFixture(t)
	noLandlock := func(trap, a1, a2, a3, a4, a5, a6 uintptr) (uintptr, uintptr, syscall.Errno) {
		if trap == unix.SYS_LANDLOCK_RESTRICT_SELF {
			return 0, 0, syscall.EOPNOTSUPP
		}
		return unix.RawSyscall6(trap, a1, a2, a3, a4, a5, a6)
	}
	type result struct{ log, read, socket string }
	r := oneThread(calls{all: noLandlock, deny: syscall.EPERM}, func(log string) result {
		return result{log, run(fileProbes(f, 1))["read a file"], run(socketProbes(f))["open a UDP socket"]}
	})
	if !strings.Contains(r.log, ", no landlock (operation not supported), seccomp") {
		t.Errorf("restrict = %q", r.log)
	}
	if r.read != "ok" || r.socket != "EPERM" {
		t.Errorf("without Landlock: reading a file %s, a socket %s; want ok and EPERM", r.read, r.socket)
	}
}
