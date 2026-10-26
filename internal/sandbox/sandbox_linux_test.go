// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

//go:build linux

package sandbox

import (
	"errors"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

func landlockABI(t *testing.T) int {
	t.Helper()
	v, _, e := unix.Syscall(unix.SYS_LANDLOCK_CREATE_RULESET, 0, 0, unix.LANDLOCK_CREATE_RULESET_VERSION)
	if e != 0 {
		t.Skipf("Landlock is not available here: %v", e)
	}
	return int(v)
}

type restricted struct {
	log                     string
	read, create, list, run error
	dial                    error
	kept                    error // a write to a file opened before
	caps                    [2]unix.CapUserData
}

// restrictOneThread runs restrict on one locked thread and tries what the sandbox forbids
// from it. The goroutine never unlocks, so the runtime ends the thread with it: the test
// process itself stays unrestricted.
func restrictOneThread(t *testing.T, dir string, kept *os.File, addr string) restricted {
	t.Helper()
	done := make(chan restricted)
	go func() {
		runtime.LockOSThread()
		var r restricted
		r.log = restrict(unix.RawSyscall6)
		_, r.read = os.ReadFile(filepath.Join(dir, "before"))
		_, r.create = os.Create(filepath.Join(dir, "after"))
		_, r.list = os.ReadDir(dir)
		r.run = fs.ErrPermission
		if errors.Is(r.read, fs.ErrPermission) { // a failed sandbox would let exec replace the test
			r.run = syscall.Exec("/bin/true", []string{"true"}, nil)
		}
		if c, err := net.Dial("tcp", addr); err == nil {
			_ = c.Close()
		} else {
			r.dial = err
		}
		_, r.kept = kept.WriteString("still open\n")
		hdr := unix.CapUserHeader{Version: unix.LINUX_CAPABILITY_VERSION_3}
		_ = unix.Capget(&hdr, &r.caps[0])
		done <- r
	}()
	return <-done
}

func TestRestrict(t *testing.T) {
	abi := landlockABI(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "before"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	kept, err := os.Create(filepath.Join(dir, "kept"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = kept.Close() }()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()

	r := restrictOneThread(t, dir, kept, ln.Addr().String())
	if !strings.Contains(r.log, "no_new_privs") || !strings.Contains(r.log, "landlock ABI") {
		t.Fatalf("restrict = %q, want no_new_privs and Landlock", r.log)
	}
	for name, err := range map[string]error{"read a file": r.read, "create a file": r.create, "list a folder": r.list, "run a program": r.run} {
		if !errors.Is(err, fs.ErrPermission) {
			t.Errorf("%s: err %v, want permission denied", name, err)
		}
	}
	if abi >= 4 && !errors.Is(r.dial, syscall.EACCES) {
		t.Errorf("TCP connect: err %v, want permission denied (ABI %d)", r.dial, abi)
	}
	if r.kept != nil {
		t.Errorf("a file opened before must stay usable: %v", r.kept)
	}
	if r.caps != noCaps {
		t.Errorf("capabilities left: %+v", r.caps)
	}
	if _, err := os.ReadFile(filepath.Join(dir, "before")); err != nil {
		t.Errorf("only the locked thread was restricted, but the test's own read failed: %v", err)
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

// A build with cgo gets ENOTSUP from AllThreadsSyscall: the sandbox is off, and says so.
func TestRestrictWithoutAllThreads(t *testing.T) {
	notSupported := func(_, _, _, _, _, _, _ uintptr) (uintptr, uintptr, syscall.Errno) {
		return 0, 0, syscall.ENOTSUP
	}
	if got := restrict(notSupported); !strings.HasPrefix(got, "sandbox: off (") {
		t.Errorf("restrict = %q, want the sandbox off with a reason", got)
	}
}
