// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

//go:build linux

package sandbox

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"unsafe" // seccomp(2) takes a pointer to the filter

	"golang.org/x/sys/unix"
)

// The seccomp filter is a denylist: after the lockdown wopr reads its terminal, writes the
// screen and the debug log, waits, signals itself and starts threads. Nothing else. The filter
// refuses what it never does again: run a program (execve, execveat and memfd_create,
// which Landlock alone does not stop), start a process, open any socket (Landlock covers
// TCP only, and not MPTCP, UDP or UNIX sockets), signal another process, push input
// into the terminal (TIOCSTI, TIOCLINUX), and the kernel interfaces that exploits use and
// wopr never needs (io_uring, bpf, userfaultfd, keyrings, namespaces, mounts, modules).
// A denied call fails with EPERM, so the program carries on: a library that tries something
// optional, such as a program to ask, gets an error it already handles. A call
// from another ABI (i386 int 0x80, x32) is never wopr's own, and kills the process.

// seccompArch is one architecture's numbering: the audit arch the kernel reports in
// seccomp_data, and the number of every system call the filter names.
type seccompArch struct {
	audit uint32
	x32   bool // amd64: x32 calls share the audit arch and set bit 30 of the number
	nr    map[string]uint32
}

// Both tables are data, so that every architecture's filter is built and checked on any
// host. Their numbers are checked against golang.org/x/sys/unix on their own architecture.
var seccompArches = map[string]seccompArch{
	"amd64": {audit: unix.AUDIT_ARCH_X86_64, x32: true, nr: map[string]uint32{
		"_sysctl": 156, "acct": 163, "add_key": 248, "adjtimex": 159, "bpf": 321, "chroot": 161,
		"clock_adjtime": 305, "clock_settime": 227, "clone": 56, "clone3": 435, "delete_module": 176,
		"execve": 59, "execveat": 322, "finit_module": 313, "fork": 57, "fsconfig": 431, "fsmount": 432,
		"fsopen": 430, "fspick": 433, "init_module": 175, "io_uring_enter": 426, "io_uring_register": 427,
		"io_uring_setup": 425, "ioctl": 16, "ioperm": 173, "iopl": 172, "kcmp": 312, "kexec_file_load": 320,
		"kexec_load": 246, "keyctl": 250, "kill": 62, "landlock_add_rule": 445,
		"landlock_create_ruleset": 444, "landlock_restrict_self": 446, "lookup_dcookie": 212,
		"memfd_create": 319, "memfd_secret": 447, "modify_ldt": 154, "mount": 165, "mount_setattr": 442,
		"move_mount": 429, "name_to_handle_at": 303, "nfsservctl": 180, "open_by_handle_at": 304,
		"open_tree": 428, "open_tree_attr": 467, "perf_event_open": 298, "personality": 135,
		"pidfd_getfd": 438, "pidfd_send_signal": 424, "pivot_root": 155, "process_vm_readv": 310,
		"process_vm_writev": 311, "ptrace": 101, "quotactl": 179, "quotactl_fd": 443, "reboot": 169,
		"request_key": 249, "rt_sigqueueinfo": 129, "rt_tgsigqueueinfo": 297, "seccomp": 317,
		"setdomainname": 171, "sethostname": 170, "setns": 308, "settimeofday": 164, "socket": 41,
		"socketpair": 53, "swapoff": 168, "swapon": 167, "syslog": 103, "tgkill": 234, "tkill": 200,
		"umount2": 166, "unshare": 272, "userfaultfd": 323, "uselib": 134, "vfork": 58, "vhangup": 153,
	}},
	"arm64": {audit: unix.AUDIT_ARCH_AARCH64, nr: map[string]uint32{
		"acct": 89, "add_key": 217, "adjtimex": 171, "bpf": 280, "chroot": 51, "clock_adjtime": 266,
		"clock_settime": 112, "clone": 220, "clone3": 435, "delete_module": 106, "execve": 221,
		"execveat": 281, "finit_module": 273, "fsconfig": 431, "fsmount": 432, "fsopen": 430,
		"fspick": 433, "init_module": 105, "io_uring_enter": 426, "io_uring_register": 427,
		"io_uring_setup": 425, "ioctl": 29, "kcmp": 272, "kexec_file_load": 294, "kexec_load": 104,
		"keyctl": 219, "kill": 129, "landlock_add_rule": 445, "landlock_create_ruleset": 444,
		"landlock_restrict_self": 446, "lookup_dcookie": 18, "memfd_create": 279, "memfd_secret": 447,
		"mount": 40, "mount_setattr": 442, "move_mount": 429, "name_to_handle_at": 264,
		"nfsservctl": 42, "open_by_handle_at": 265, "open_tree": 428, "open_tree_attr": 467,
		"perf_event_open": 241, "personality": 92, "pidfd_getfd": 438, "pidfd_send_signal": 424,
		"pivot_root": 41, "process_vm_readv": 270, "process_vm_writev": 271, "ptrace": 117,
		"quotactl": 60, "quotactl_fd": 443, "reboot": 142, "request_key": 218, "rt_sigqueueinfo": 138,
		"rt_tgsigqueueinfo": 240, "seccomp": 277, "setdomainname": 162, "sethostname": 161,
		"setns": 268, "settimeofday": 170, "socket": 198, "socketpair": 199, "swapoff": 225,
		"swapon": 224, "syslog": 116, "tgkill": 131, "tkill": 130, "umount2": 39, "unshare": 97,
		"userfaultfd": 282, "vhangup": 58,
	}},
}

// denied lists the calls that always fail. A name that an architecture lacks (fork on arm64)
// is skipped there.
var denied = []string{
	// running programs and starting processes (clone is checked by its flags below)
	"execve", "execveat", "memfd_create", "memfd_secret", "fork", "vfork",
	// sockets of every family
	"socket", "socketpair",
	// other processes: their memory, their files and their signals (kill and tgkill below)
	"ptrace", "process_vm_readv", "process_vm_writev", "pidfd_getfd", "pidfd_send_signal", "kcmp", "tkill",
	// kernel interfaces that wopr never uses
	"bpf", "perf_event_open", "userfaultfd", "io_uring_setup", "io_uring_enter", "io_uring_register",
	"keyctl", "add_key", "request_key", "personality", "uselib", "iopl", "ioperm", "modify_ldt", "_sysctl",
	// namespaces, mounts and opening files by handle
	"unshare", "setns", "mount", "umount2", "pivot_root", "chroot", "mount_setattr", "open_tree",
	"open_tree_attr", "move_mount", "fsopen", "fsconfig", "fsmount", "fspick",
	"name_to_handle_at", "open_by_handle_at",
	// administering the system
	"kexec_load", "kexec_file_load", "init_module", "finit_module", "delete_module", "reboot",
	"swapon", "swapoff", "acct", "quotactl", "quotactl_fd", "syslog", "vhangup", "nfsservctl",
	"lookup_dcookie", "settimeofday", "clock_settime", "clock_adjtime", "adjtimex", "sethostname",
	"setdomainname",
	// changing the sandbox, which is complete once the filter is in
	"landlock_create_ruleset", "landlock_add_rule", "landlock_restrict_self", "seccomp",
}

// seccomp_data offsets (include/uapi/linux/seccomp.h). Arguments are 64-bit; amd64 and arm64
// are little-endian, so the low 32 bits come first. ioctl's request, clone's flags and a pid
// are 32-bit values to the kernel, which ignores the high bits: compare those only.
const (
	offNr   = 0
	offArch = 4
	offArg0 = 16
	offArg1 = 24
	x32Bit  = 0x40000000

	cloneNewAny = unix.CLONE_NEWNS | unix.CLONE_NEWUTS | unix.CLONE_NEWIPC | unix.CLONE_NEWUSER |
		unix.CLONE_NEWPID | unix.CLONE_NEWNET | unix.CLONE_NEWCGROUP | unix.CLONE_NEWTIME

	retAllow = unix.SECCOMP_RET_ALLOW
	retKill  = unix.SECCOMP_RET_KILL_PROCESS
	retNosys = unix.SECCOMP_RET_ERRNO | uint32(unix.ENOSYS)
)

// insn is one BPF instruction whose jumps name labels; assemble turns them into offsets.
type insn struct {
	f      unix.SockFilter
	jt, jf string // "" falls through
	label  string // a label for the instruction itself
}

func stmt(code uint16, k uint32) insn { return insn{f: unix.SockFilter{Code: code, K: k}} }

func jump(code uint16, k uint32, jt, jf string) insn {
	return insn{f: unix.SockFilter{Code: code, K: k}, jt: jt, jf: jf}
}

func labelled(label string, i insn) insn { i.label = label; return i }

const (
	ldAbs = unix.BPF_LD | unix.BPF_W | unix.BPF_ABS
	jeq   = unix.BPF_JMP | unix.BPF_JEQ | unix.BPF_K
	jge   = unix.BPF_JMP | unix.BPF_JGE | unix.BPF_K
	jset  = unix.BPF_JMP | unix.BPF_JSET | unix.BPF_K
	ret   = unix.BPF_RET | unix.BPF_K
)

// seccompProgram builds the filter for arch. pid is the process's own pid, the only target
// its signals may have; deny is the errno a denied call gets (EPERM; tests choose another).
func seccompProgram(arch seccompArch, pid uint32, deny syscall.Errno) ([]unix.SockFilter, error) {
	retDeny := unix.SECCOMP_RET_ERRNO | uint32(deny)&unix.SECCOMP_RET_DATA //nolint:gosec // an errno is below 4096
	nr := arch.nr
	p := []insn{
		stmt(ldAbs, offArch),
		jump(jeq, arch.audit, "native", "kill"),
		labelled("native", stmt(ldAbs, offNr)),
	}
	if arch.x32 {
		p = append(p, jump(jge, x32Bit, "kill", ""))
	}
	p = append(p,
		jump(jeq, nr["ioctl"], "ioctl", ""),
		jump(jeq, nr["clone"], "clone", ""),
		jump(jeq, nr["clone3"], "nosys", ""),
		jump(jeq, nr["kill"], "kill-pid", ""),
		jump(jeq, nr["tgkill"], "own-pid", ""),
		jump(jeq, nr["rt_sigqueueinfo"], "own-pid", ""),
		jump(jeq, nr["rt_tgsigqueueinfo"], "own-pid", ""),
	)
	for _, name := range denied {
		if n, ok := nr[name]; ok {
			p = append(p, jump(jeq, n, "deny", ""))
		}
	}
	p = append(p,
		stmt(ret, retAllow),
		// ioctl: everything but pushing input into a terminal.
		labelled("ioctl", stmt(ldAbs, offArg1)),
		jump(jeq, unix.TIOCSTI, "deny", ""),
		jump(jeq, unix.TIOCLINUX, "deny", "allow"),
		// clone: new threads only (the runtime's flags always include CLONE_THREAD), and no
		// namespaces. A process (fork) fails with deny.
		labelled("clone", stmt(ldAbs, offArg0)),
		jump(jset, unix.CLONE_THREAD, "", "deny"),
		jump(jset, cloneNewAny, "deny", "allow"),
		// kill: only the process itself (0 is its process group, for Ctrl+Z's SIGTSTP).
		labelled("kill-pid", stmt(ldAbs, offArg0)),
		jump(jeq, 0, "allow", ""),
		// tgkill and the sigqueue calls: only threads of this process.
		labelled("own-pid", stmt(ldAbs, offArg0)),
		jump(jeq, pid, "allow", "deny"),
		labelled("allow", stmt(ret, retAllow)),
		labelled("deny", stmt(ret, retDeny)),
		// clone3: its flags are in memory that BPF cannot read. ENOSYS makes callers use clone.
		labelled("nosys", stmt(ret, retNosys)),
		labelled("kill", stmt(ret, retKill)),
	)
	return assemble(p)
}

// assemble resolves the labels into BPF's forward jump offsets (at most 255 instructions).
func assemble(p []insn) ([]unix.SockFilter, error) {
	at := map[string]int{}
	for i, in := range p {
		if in.label != "" {
			at[in.label] = i
		}
	}
	off := func(i int, label string) (uint8, error) {
		if label == "" {
			return 0, nil
		}
		to, ok := at[label]
		if !ok {
			return 0, fmt.Errorf("seccomp: no label %q", label)
		}
		d := to - i - 1
		if d < 0 || d > 255 {
			return 0, fmt.Errorf("seccomp: jump from %d to %q (%d) out of range", i, label, to)
		}
		return uint8(d), nil //nolint:gosec // checked just above
	}
	out := make([]unix.SockFilter, len(p))
	for i, in := range p {
		out[i] = in.f
		var err error
		if out[i].Jt, err = off(i, in.jt); err != nil {
			return nil, err
		}
		if out[i].Jf, err = off(i, in.jf); err != nil {
			return nil, err
		}
	}
	if len(out) > unix.BPF_MAXINSNS {
		return nil, errors.New("seccomp: program too long")
	}
	return out, nil
}

// The filter and its header live in package variables while the kernel reads them.
var (
	filter []unix.SockFilter
	fprog  unix.SockFprog
)

// errNoArch: no table for this architecture (riscv64, say), so no filter.
var errNoArch = errors.New("no filter for " + runtime.GOARCH)

// installSeccomp loads the filter. With tsync it covers every thread of the process (and
// gives them no_new_privs); without, only the calling thread, which tests use. no_new_privs
// must be set on the calling thread first.
func installSeccomp(tsync bool, deny syscall.Errno) error {
	arch, ok := seccompArches[runtime.GOARCH]
	if !ok {
		return errNoArch
	}
	var err error
	if filter, err = seccompProgram(arch, uint32(unix.Getpid()), deny); err != nil { //nolint:gosec // pids are positive 32-bit values
		return err
	}
	fprog = unix.SockFprog{Len: uint16(len(filter)), Filter: &filter[0]} //nolint:gosec // assemble caps the length at 4096
	var flags uintptr
	if tsync {
		flags = unix.SECCOMP_FILTER_FLAG_TSYNC
	}
	r, _, e := unix.Syscall(unix.SYS_SECCOMP, unix.SECCOMP_SET_MODE_FILTER, flags, uintptr(unsafe.Pointer(&fprog))) //nolint:gosec // the kernel copies the filter during the call
	runtime.KeepAlive(&fprog)
	switch {
	case e != 0:
		return e // EINVAL: no CONFIG_SECCOMP_FILTER; ENOSYS: no seccomp at all
	case r != 0:
		return fmt.Errorf("thread %d has another filter", r) // TSYNC reports a thread it could not sync
	}
	return nil
}
