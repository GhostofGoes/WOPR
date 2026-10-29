// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

//go:build linux && arm64

package sandbox

import "golang.org/x/sys/unix"

// hostNumbers is golang.org/x/sys/unix's number for every call the arm64 table names.
var hostNumbers = map[string]uintptr{
	"acct": unix.SYS_ACCT, "add_key": unix.SYS_ADD_KEY, "adjtimex": unix.SYS_ADJTIMEX,
	"bpf": unix.SYS_BPF, "chroot": unix.SYS_CHROOT, "clock_adjtime": unix.SYS_CLOCK_ADJTIME,
	"clock_settime": unix.SYS_CLOCK_SETTIME, "clone": unix.SYS_CLONE, "clone3": unix.SYS_CLONE3,
	"delete_module": unix.SYS_DELETE_MODULE, "execve": unix.SYS_EXECVE, "execveat": unix.SYS_EXECVEAT,
	"finit_module": unix.SYS_FINIT_MODULE, "fsconfig": unix.SYS_FSCONFIG,
	"fsmount": unix.SYS_FSMOUNT, "fsopen": unix.SYS_FSOPEN, "fspick": unix.SYS_FSPICK,
	"init_module": unix.SYS_INIT_MODULE, "io_uring_enter": unix.SYS_IO_URING_ENTER,
	"io_uring_register": unix.SYS_IO_URING_REGISTER, "io_uring_setup": unix.SYS_IO_URING_SETUP,
	"ioctl": unix.SYS_IOCTL, "kcmp": unix.SYS_KCMP,
	"kexec_file_load": unix.SYS_KEXEC_FILE_LOAD, "kexec_load": unix.SYS_KEXEC_LOAD, "keyctl": unix.SYS_KEYCTL,
	"kill": unix.SYS_KILL, "landlock_add_rule": unix.SYS_LANDLOCK_ADD_RULE,
	"landlock_create_ruleset": unix.SYS_LANDLOCK_CREATE_RULESET,
	"landlock_restrict_self":  unix.SYS_LANDLOCK_RESTRICT_SELF, "lookup_dcookie": unix.SYS_LOOKUP_DCOOKIE,
	"memfd_create": unix.SYS_MEMFD_CREATE, "memfd_secret": unix.SYS_MEMFD_SECRET,
	"mount": unix.SYS_MOUNT, "mount_setattr": unix.SYS_MOUNT_SETATTR,
	"move_mount": unix.SYS_MOVE_MOUNT, "name_to_handle_at": unix.SYS_NAME_TO_HANDLE_AT,
	"nfsservctl": unix.SYS_NFSSERVCTL, "open_by_handle_at": unix.SYS_OPEN_BY_HANDLE_AT,
	"open_tree": unix.SYS_OPEN_TREE, "open_tree_attr": unix.SYS_OPEN_TREE_ATTR,
	"perf_event_open": unix.SYS_PERF_EVENT_OPEN, "personality": unix.SYS_PERSONALITY,
	"pidfd_getfd": unix.SYS_PIDFD_GETFD, "pidfd_send_signal": unix.SYS_PIDFD_SEND_SIGNAL,
	"pivot_root": unix.SYS_PIVOT_ROOT, "process_vm_readv": unix.SYS_PROCESS_VM_READV,
	"process_vm_writev": unix.SYS_PROCESS_VM_WRITEV, "ptrace": unix.SYS_PTRACE, "quotactl": unix.SYS_QUOTACTL,
	"quotactl_fd": unix.SYS_QUOTACTL_FD, "reboot": unix.SYS_REBOOT, "request_key": unix.SYS_REQUEST_KEY,
	"rt_sigqueueinfo": unix.SYS_RT_SIGQUEUEINFO, "rt_tgsigqueueinfo": unix.SYS_RT_TGSIGQUEUEINFO,
	"seccomp": unix.SYS_SECCOMP, "setdomainname": unix.SYS_SETDOMAINNAME, "sethostname": unix.SYS_SETHOSTNAME,
	"setns": unix.SYS_SETNS, "settimeofday": unix.SYS_SETTIMEOFDAY, "socket": unix.SYS_SOCKET,
	"socketpair": unix.SYS_SOCKETPAIR, "swapoff": unix.SYS_SWAPOFF, "swapon": unix.SYS_SWAPON,
	"syslog": unix.SYS_SYSLOG, "tgkill": unix.SYS_TGKILL, "tkill": unix.SYS_TKILL, "umount2": unix.SYS_UMOUNT2,
	"unshare": unix.SYS_UNSHARE, "userfaultfd": unix.SYS_USERFAULTFD, "vhangup": unix.SYS_VHANGUP,
	"fchmod": unix.SYS_FCHMOD, "fchmodat": unix.SYS_FCHMODAT, "fchmodat2": unix.SYS_FCHMODAT2,
	"fchown": unix.SYS_FCHOWN, "fchownat": unix.SYS_FCHOWNAT, "fcntl": unix.SYS_FCNTL,
	"file_setattr": unix.SYS_FILE_SETATTR, "fremovexattr": unix.SYS_FREMOVEXATTR,
	"fsetxattr": unix.SYS_FSETXATTR, "ioprio_set": unix.SYS_IOPRIO_SET, "lremovexattr": unix.SYS_LREMOVEXATTR,
	"lsetxattr": unix.SYS_LSETXATTR, "migrate_pages": unix.SYS_MIGRATE_PAGES, "move_pages": unix.SYS_MOVE_PAGES,
	"prlimit64": unix.SYS_PRLIMIT64, "process_madvise": unix.SYS_PROCESS_MADVISE,
	"process_mrelease": unix.SYS_PROCESS_MRELEASE, "removexattr": unix.SYS_REMOVEXATTR,
	"removexattrat": unix.SYS_REMOVEXATTRAT, "sched_setaffinity": unix.SYS_SCHED_SETAFFINITY,
	"sched_setattr": unix.SYS_SCHED_SETATTR, "sched_setparam": unix.SYS_SCHED_SETPARAM,
	"sched_setscheduler": unix.SYS_SCHED_SETSCHEDULER, "setpriority": unix.SYS_SETPRIORITY,
	"setxattr": unix.SYS_SETXATTR, "setxattrat": unix.SYS_SETXATTRAT, "truncate": unix.SYS_TRUNCATE,
	"utimensat": unix.SYS_UTIMENSAT,
}

var hostAllowNumbers = map[string]uintptr{
	"read": unix.SYS_READ, "write": unix.SYS_WRITE, "close": unix.SYS_CLOSE, "fstat": unix.SYS_FSTAT,
	"lseek": unix.SYS_LSEEK, "mmap": unix.SYS_MMAP, "mprotect": unix.SYS_MPROTECT, "munmap": unix.SYS_MUNMAP,
	"brk": unix.SYS_BRK, "rt_sigaction": unix.SYS_RT_SIGACTION, "rt_sigprocmask": unix.SYS_RT_SIGPROCMASK,
	"rt_sigreturn": unix.SYS_RT_SIGRETURN, "pread64": unix.SYS_PREAD64, "sched_yield": unix.SYS_SCHED_YIELD,
	"mincore": unix.SYS_MINCORE, "madvise": unix.SYS_MADVISE, "nanosleep": unix.SYS_NANOSLEEP,
	"getpid": unix.SYS_GETPID, "exit": unix.SYS_EXIT, "fcntl": unix.SYS_FCNTL, "sigaltstack": unix.SYS_SIGALTSTACK,
	"prctl": unix.SYS_PRCTL, "gettid": unix.SYS_GETTID, "futex": unix.SYS_FUTEX,
	"sched_getaffinity": unix.SYS_SCHED_GETAFFINITY, "clock_gettime": unix.SYS_CLOCK_GETTIME,
	"exit_group": unix.SYS_EXIT_GROUP, "epoll_ctl": unix.SYS_EPOLL_CTL, "openat": unix.SYS_OPENAT,
	"epoll_pwait": unix.SYS_EPOLL_PWAIT, "eventfd2": unix.SYS_EVENTFD2, "epoll_create1": unix.SYS_EPOLL_CREATE1,
	"pipe2": unix.SYS_PIPE2, "getrandom": unix.SYS_GETRANDOM, "rseq": unix.SYS_RSEQ,
	"restart_syscall": unix.SYS_RESTART_SYSCALL, "uname": unix.SYS_UNAME, "pidfd_open": unix.SYS_PIDFD_OPEN,
}
