// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

//go:build linux

package sandbox

import (
	"fmt"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe" // the Landlock and capability calls take pointers: the one package allowed unsafe (docs/PLAN.md §4.1)

	"golang.org/x/sys/unix"
)

// syscall6 runs one system call: on every thread in Restrict, on the calling thread in tests.
type syscall6 func(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err syscall.Errno)

// calls is how restrict reaches the kernel. Restrict restricts every thread; the tests
// restrict one locked thread, which the runtime throws away when its goroutine ends.
type calls struct {
	all   syscall6      // a call made on every thread (syscall.AllThreadsSyscall6)
	tsync bool          // the seccomp filter covers every thread
	deny  syscall.Errno // what a call that the filter refuses returns
}

// Restrict sandboxes the whole process and returns a line for the debug log. Call it once
// the terminal is open and the color profile is known: after it, wopr opens no file.
//
// Every thread must be restricted, and the Go runtime starts threads before main, so the
// per-thread calls go through syscall.AllThreadsSyscall6. That works only without cgo. A cgo
// build (go install with a C compiler) gets ENOTSUP, and keeps only no_new_privs and the
// seccomp filter, which the kernel itself copies to every thread. The release builds set
// CGO_ENABLED=0.
func Restrict() string {
	_, _ = time.Now().Zone() // Go reads /etc/localtime on first use; read it while it can
	return restrict(calls{all: syscall.AllThreadsSyscall6, tsync: true, deny: syscall.EPERM})
}

// restrict does each step it can, in order, and says what it did. no_new_privs comes first:
// Landlock and seccomp need it. The seccomp filter comes last: it refuses the Landlock calls.
func restrict(c calls) string {
	if _, _, e := c.all(unix.SYS_PRCTL, unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0, 0); e == syscall.ENOTSUP {
		return restrictCgo(c)
	} else if e != 0 {
		return fmt.Sprintf("sandbox: off (no_new_privs: %v)", e)
	}
	did := []string{"no_new_privs"}
	if dropped, err := dropCaps(c.all); err != nil {
		did = append(did, fmt.Sprintf("capabilities kept (%v)", err))
	} else if dropped {
		did = append(did, "capabilities dropped")
	}
	if abi, err := landlock(c.all); err != nil {
		did = append(did, fmt.Sprintf("no landlock (%v)", err))
	} else {
		did = append(did, fmt.Sprintf("landlock ABI %d", abi))
	}
	did = append(did, seccompStatus(installSeccomp(c.tsync, c.deny)))
	return "sandbox: " + strings.Join(did, ", ")
}

// restrictCgo is restrict for a cgo build, where no call reaches every thread. The seccomp
// filter still does, through the kernel's TSYNC, and brings no_new_privs to every thread
// with it. Capabilities and Landlock are per thread, so they stay as they are.
func restrictCgo(c calls) string {
	runtime.LockOSThread() // no_new_privs and the filter must be set from the same thread
	defer runtime.UnlockOSThread()
	if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
		return fmt.Sprintf("sandbox: off (no_new_privs: %v)", err)
	}
	return "sandbox: cgo build: no_new_privs, " + seccompStatus(installSeccomp(c.tsync, c.deny)) +
		", no capability drop or landlock"
}

func seccompStatus(err error) string {
	if err != nil {
		return fmt.Sprintf("no seccomp (%v)", err)
	}
	return "seccomp"
}

// The capability calls' arguments live in package variables, not on a goroutine stack that
// could move while other threads read them. Restrict runs once, so nothing shares them.
var (
	capHdr  = unix.CapUserHeader{Version: unix.LINUX_CAPABILITY_VERSION_3}
	capData [2]unix.CapUserData // 64 capability bits, as two 32-bit words
	noCaps  [2]unix.CapUserData
)

// dropCaps clears every capability set when the process has any (run as root, or with
// file capabilities), and reports whether it did. An ordinary user has none to drop.
func dropCaps(all syscall6) (bool, error) {
	if err := unix.Capget(&capHdr, &capData[0]); err != nil {
		return false, err
	}
	if capData == noCaps {
		return false, nil
	}
	_, _, e := all(unix.SYS_CAPSET, uintptr(unsafe.Pointer(&capHdr)), uintptr(unsafe.Pointer(&noCaps[0])), 0, 0, 0, 0) //nolint:gosec // capset reads both while the call runs; they are package variables, so they never move
	runtime.KeepAlive(&capHdr)
	if e != 0 {
		return false, e
	}
	return true, nil
}

// fsV1 is every file system right of Landlock ABI 1. Later ABIs add rights below.
const fsV1 = unix.LANDLOCK_ACCESS_FS_EXECUTE | unix.LANDLOCK_ACCESS_FS_WRITE_FILE |
	unix.LANDLOCK_ACCESS_FS_READ_FILE | unix.LANDLOCK_ACCESS_FS_READ_DIR |
	unix.LANDLOCK_ACCESS_FS_REMOVE_DIR | unix.LANDLOCK_ACCESS_FS_REMOVE_FILE |
	unix.LANDLOCK_ACCESS_FS_MAKE_CHAR | unix.LANDLOCK_ACCESS_FS_MAKE_DIR |
	unix.LANDLOCK_ACCESS_FS_MAKE_REG | unix.LANDLOCK_ACCESS_FS_MAKE_SOCK |
	unix.LANDLOCK_ACCESS_FS_MAKE_FIFO | unix.LANDLOCK_ACCESS_FS_MAKE_BLOCK |
	unix.LANDLOCK_ACCESS_FS_MAKE_SYM

// ruleset handles every right that the kernel's Landlock ABI knows, so a domain with no rules
// denies them all. It leaves out LANDLOCK_SCOPE_SIGNAL: each thread gets its own domain, and
// kernels before the fix that always allows signals inside a process would then refuse the
// Go runtime's signals between its threads.
func ruleset(abi int) unix.LandlockRulesetAttr {
	a := unix.LandlockRulesetAttr{Access_fs: fsV1}
	if abi >= 2 {
		a.Access_fs |= unix.LANDLOCK_ACCESS_FS_REFER
	}
	if abi >= 3 {
		a.Access_fs |= unix.LANDLOCK_ACCESS_FS_TRUNCATE
	}
	if abi >= 4 {
		a.Access_net = unix.LANDLOCK_ACCESS_NET_BIND_TCP | unix.LANDLOCK_ACCESS_NET_CONNECT_TCP
	}
	if abi >= 5 {
		a.Access_fs |= unix.LANDLOCK_ACCESS_FS_IOCTL_DEV // checked at open: the open terminal keeps it
	}
	if abi >= 6 {
		a.Scoped = unix.LANDLOCK_SCOPE_ABSTRACT_UNIX_SOCKET
	}
	return a
}

// landlock enters a Landlock domain with no rules and returns the kernel's ABI version.
// no_new_privs must already be set.
func landlock(all syscall6) (int, error) {
	v, _, e := unix.Syscall(unix.SYS_LANDLOCK_CREATE_RULESET, 0, 0, unix.LANDLOCK_CREATE_RULESET_VERSION)
	if e != 0 {
		return 0, e // ENOSYS: before Linux 5.13; EOPNOTSUPP: Landlock is not enabled
	}
	abi := int(v)
	attr := ruleset(abi)
	fd, _, e := unix.Syscall(unix.SYS_LANDLOCK_CREATE_RULESET, uintptr(unsafe.Pointer(&attr)), unsafe.Sizeof(attr), 0) //nolint:gosec // the kernel reads attr during the call
	if e != 0 {
		return abi, e
	}
	defer func() { _ = unix.Close(int(fd)) }()
	if _, _, e := all(unix.SYS_LANDLOCK_RESTRICT_SELF, fd, 0, 0, 0, 0, 0); e != 0 {
		return abi, e
	}
	return abi, nil
}
