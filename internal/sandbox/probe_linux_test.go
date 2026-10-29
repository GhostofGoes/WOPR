// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

//go:build linux

package sandbox

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// The probes try, from the calling thread, what the sandbox must refuse and what it must
// still allow. Each one is harmless if it succeeds by mistake: it works only on its own
// temporary files and on pids and file descriptors that do not exist, or it fails in the
// kernel with an errno other than the filter's. A probe that could replace or copy the
// process (exec, fork) runs only once another probe has shown that the sandbox is in place.

// fixture is what a probe works on. The test makes it before the sandbox, as wopr opens the
// terminal and the debug log before its lockdown.
type fixture struct {
	Dir      string // a temporary folder with the files below
	TCP      string // a TCP listener, host:port
	UnixPath string // a UNIX socket listener in Dir
	Abstract string // an abstract UNIX socket listener ("@name")
	kept     *os.File
	pty      *os.File // the terminal side of a pty pair
}

// files the fixture holds, one per probe that changes a file.
var fixtureFiles = []string{"before", "remove", "rename", "truncate"}

func (f fixture) path(name string) string { return filepath.Join(f.Dir, name) }

// outcome is a probe's result: "ok", or the errno's name (EACCES, EPERM, ...).
func outcome(err error) string {
	if err == nil {
		return "ok"
	}
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return errnoName(errno)
	}
	return err.Error()
}

func errnoName(e syscall.Errno) string {
	if e == 0 {
		return "ok"
	}
	if name := unix.ErrnoName(e); name != "" {
		return name
	}
	return e.Error()
}

// raw makes a system call on the calling thread and returns its errno only.
func raw(trap uintptr, a ...uintptr) error {
	var args [6]uintptr
	copy(args[:], a)
	_, _, e := unix.RawSyscall6(trap, args[0], args[1], args[2], args[3], args[4], args[5])
	if e != 0 {
		return e
	}
	return nil
}

// socketTo makes a socket and connects it to sa; either step may be the one refused.
func socketTo(domain, typ, proto int, sa unix.Sockaddr) error {
	fd, err := unix.Socket(domain, typ|unix.SOCK_CLOEXEC, proto)
	if err != nil {
		return err
	}
	defer func() { _ = unix.Close(fd) }()
	if sa == nil {
		return nil
	}
	return unix.Connect(fd, sa)
}

func inet4(hostport string) unix.Sockaddr {
	addr, err := net.ResolveTCPAddr("tcp4", hostport)
	if err != nil {
		panic(err)
	}
	sa := &unix.SockaddrInet4{Port: addr.Port}
	copy(sa.Addr[:], addr.IP.To4())
	return sa
}

// fileProbes are refused by Landlock (EACCES). abi gates the rights a kernel lacks.
func fileProbes(f fixture, abi int) map[string]func() error {
	p := map[string]func() error{
		"read a file":     func() error { _, err := os.ReadFile(f.path("before")); return err },
		"list a folder":   func() error { _, err := os.ReadDir(f.Dir); return err },
		"remove a file":   func() error { return os.Remove(f.path("remove")) },
		"rename a file":   func() error { return os.Rename(f.path("rename"), f.path("renamed")) },
		"make a folder":   func() error { return os.Mkdir(f.path("folder"), 0o700) },
		"make a symlink":  func() error { return os.Symlink("before", f.path("symlink")) },
		"make a hardlink": func() error { return os.Link(f.path("before"), f.path("hardlink")) },
		"make a fifo":     func() error { return unix.Mkfifo(f.path("fifo"), 0o600) },
		"create a file": func() error {
			c, err := os.Create(f.path("created"))
			if err == nil {
				_ = c.Close()
			}
			return err
		},
		"open a device": func() error {
			c, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
			if err == nil {
				_ = c.Close()
			}
			return err
		},
	}
	if abi >= 3 {
		p["truncate a file"] = func() error { return os.Truncate(f.path("truncate"), 0) }
	}
	return p
}

// tcpProbes are refused by Landlock (ABI 4) and, earlier, by the seccomp filter's socket rule.
func tcpProbes(f fixture) map[string]func() error {
	return map[string]func() error{
		"connect TCP": func() error { return socketTo(unix.AF_INET, unix.SOCK_STREAM, unix.IPPROTO_TCP, inet4(f.TCP)) },
		"bind TCP": func() error {
			fd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, unix.IPPROTO_TCP)
			if err != nil {
				return err
			}
			defer func() { _ = unix.Close(fd) }()
			return unix.Bind(fd, &unix.SockaddrInet4{Addr: [4]byte{127, 0, 0, 1}})
		},
	}
}

// socketProbes get past Landlock: only the seccomp filter refuses them.
func socketProbes(f fixture) map[string]func() error {
	return map[string]func() error{
		"connect MPTCP": func() error {
			return socketTo(unix.AF_INET, unix.SOCK_STREAM, unix.IPPROTO_MPTCP, inet4(f.TCP))
		},
		"open a UDP socket":          func() error { return socketTo(unix.AF_INET, unix.SOCK_DGRAM, 0, nil) },
		"open an IPv6 socket":        func() error { return socketTo(unix.AF_INET6, unix.SOCK_STREAM, 0, nil) },
		"open a netlink socket":      func() error { return socketTo(unix.AF_NETLINK, unix.SOCK_RAW, unix.NETLINK_ROUTE, nil) },
		"connect a UNIX socket file": func() error { return socketTo(unix.AF_UNIX, unix.SOCK_STREAM, 0, &unix.SockaddrUnix{Name: f.UnixPath}) },
		"make a socket pair":         func() error { _, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM, 0); return err },
		"run a program from memory": func() error {
			fd, err := unix.MemfdCreate("probe", unix.MFD_CLOEXEC)
			if err != nil {
				return err
			}
			defer func() { _ = unix.Close(fd) }()
			empty := []byte{0} // "": an empty memfd holds no program, so a working execveat gives ENOEXEC
			return raw(unix.SYS_EXECVEAT, uintptr(fd), uintptr(unsafe.Pointer(&empty[0])), 0, 0, unix.AT_EMPTY_PATH)
		},
	}
}

// abstractProbe is refused by Landlock's scope (ABI 6) and by the filter.
func abstractProbe(f fixture) error {
	return socketTo(unix.AF_UNIX, unix.SOCK_STREAM, 0, &unix.SockaddrUnix{Name: f.Abstract})
}

// keptProbes must keep working: files and the terminal opened before the sandbox, and what the
// runtime and Bubble Tea do after it (pipes and epoll on resume, signals to the process
// itself, memory, randomness).
func keptProbes(f fixture) map[string]func() error {
	p := map[string]func() error{
		"write a file opened before": func() error { _, err := f.kept.WriteString("written in the sandbox\n"); return err },
		"make a pipe": func() error {
			r, w, err := os.Pipe()
			if err == nil {
				_, err = w.WriteString("x")
				_ = r.Close()
				_ = w.Close()
			}
			return err
		},
		"make an epoll": func() error {
			fd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
			if err == nil {
				_ = unix.Close(fd)
			}
			return err
		},
		"map memory": func() error {
			b, err := unix.Mmap(-1, 0, 1<<20, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_ANON|unix.MAP_PRIVATE)
			if err == nil {
				err = unix.Madvise(b, unix.MADV_DONTNEED)
				_ = unix.Munmap(b)
			}
			return err
		},
		"read random bytes":  func() error { _, err := unix.Getrandom(make([]byte, 16), 0); return err },
		"signal the process": func() error { return unix.Kill(unix.Getpid(), 0) },
		"signal the process group": func() error {
			return unix.Kill(0, 0) // Ctrl+Z's SIGTSTP goes to the group
		},
		"signal this thread": func() error { return unix.Tgkill(unix.Getpid(), unix.Gettid(), 0) },
		"read no_new_privs": func() error {
			if v, err := unix.PrctlRetInt(unix.PR_GET_NO_NEW_PRIVS, 0, 0, 0, 0); err != nil || v != 1 {
				return errors.Join(err, errors.New("no_new_privs is not set"))
			}
			return nil
		},
	}
	if f.pty != nil {
		fd := int(f.pty.Fd())
		p["read the terminal's modes"] = func() error { _, err := unix.IoctlGetTermios(fd, unix.TCGETS); return err }
		p["set the terminal's modes"] = func() error {
			t, err := unix.IoctlGetTermios(fd, unix.TCGETS)
			if err == nil {
				err = unix.IoctlSetTermios(fd, unix.TCSETS, t)
			}
			return err
		}
		p["read the terminal's size"] = func() error { _, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ); return err }
		p["set the terminal's size"] = func() error {
			return unix.IoctlSetWinsize(fd, unix.TIOCSWINSZ, &unix.Winsize{Row: 24, Col: 80})
		}
	}
	return p
}

// filterProbes are what only the seccomp filter refuses, each made so that the kernel
// itself would answer with another errno (named in the comment): with a filter whose errno
// is one the kernel never gives for these calls, every answer shows the filter decided it.
func filterProbes(f fixture) map[string]func() error {
	const nobody = 0x7ffffff0 // a pid that does not exist
	p := map[string]func() error{
		"ptrace":              func() error { return raw(unix.SYS_PTRACE, unix.PTRACE_PEEKDATA, nobody, 0, 0) },         // ESRCH
		"io_uring_setup":      func() error { return raw(unix.SYS_IO_URING_SETUP, 0, 0) },                               // EINVAL or EFAULT
		"bpf":                 func() error { return raw(unix.SYS_BPF, ^uintptr(0), 0, 0) },                             // EINVAL
		"perf_event_open":     func() error { return raw(unix.SYS_PERF_EVENT_OPEN, 0, 0, ^uintptr(0), ^uintptr(0), 0) }, // EFAULT
		"userfaultfd":         func() error { return raw(unix.SYS_USERFAULTFD, ^uintptr(0)) },                           // EINVAL
		"keyctl":              func() error { return raw(unix.SYS_KEYCTL, ^uintptr(0)) },                                // EOPNOTSUPP
		"unshare":             func() error { return raw(unix.SYS_UNSHARE, unix.CLONE_NEWUSER) },                        // EINVAL: many threads
		"setns":               func() error { return raw(unix.SYS_SETNS, ^uintptr(0), 0) },                              // EBADF
		"personality":         func() error { return raw(unix.SYS_PERSONALITY, 0xffffffff) },                            // reads it: ok
		"pidfd_send_signal":   func() error { return raw(unix.SYS_PIDFD_SEND_SIGNAL, ^uintptr(0), 0, 0, 0) },            // EBADF
		"pidfd_getfd":         func() error { return raw(unix.SYS_PIDFD_GETFD, ^uintptr(0), 0, 0) },                     // EBADF
		"process_vm_readv":    func() error { return raw(unix.SYS_PROCESS_VM_READV, nobody, 0, 1, 0, 1, 0) },            // ESRCH
		"landlock":            func() error { return raw(unix.SYS_LANDLOCK_CREATE_RULESET, 0, 0, 1) },                   // ok: the ABI
		"seccomp":             func() error { return raw(unix.SYS_SECCOMP, 99, 0, 0) },                                  // EINVAL
		"memfd_create":        func() error { return raw(unix.SYS_MEMFD_CREATE, 0, 0) },                                 // EFAULT
		"execve":              func() error { return raw(unix.SYS_EXECVE, 0, 0, 0) },                                    // EFAULT
		"socket":              func() error { return raw(unix.SYS_SOCKET, ^uintptr(0), 0, 0) },                          // EAFNOSUPPORT
		"tkill":               func() error { return raw(unix.SYS_TKILL, uintptr(unix.Gettid()), 0) },                   // ok
		"kill another":        func() error { return raw(unix.SYS_KILL, nobody, 0) },                                    // ESRCH
		"tgkill another":      func() error { return raw(unix.SYS_TGKILL, nobody, nobody, 0) },                          // ESRCH
		"clone a namespace":   func() error { return raw(unix.SYS_CLONE, unix.CLONE_THREAD|unix.CLONE_NEWUSER) },        // EINVAL
		"clone3":              func() error { return raw(unix.SYS_CLONE3, 0, 0) },                                       // EINVAL; the filter says ENOSYS
		"push terminal input": func() error { return ioctlProbe(f, unix.TIOCSTI) },                                      // ok, or EPERM (legacy_tiocsti=0)
		"push terminal input (high bits)": func() error {
			return ioctlProbe(f, 1<<32|unix.TIOCSTI) // the kernel ignores the high bits
		},
		"select on a console": func() error { return ioctlProbe(f, unix.TIOCLINUX) }, // ENOTTY on a pty
	}
	return p
}

func ioctlProbe(f fixture, req uint64) error {
	if f.pty == nil {
		return errors.New("no terminal")
	}
	c := []byte{'\a'}
	return raw(unix.SYS_IOCTL, f.pty.Fd(), uintptr(req), uintptr(unsafe.Pointer(&c[0])))
}

// forkProbe tries to start a process with clone (fork's flags). Run it only where the filter
// is known to be in place: if a broken filter lets it through, the copy exits at once.
func forkProbe() error {
	pid, _, e := unix.RawSyscall6(unix.SYS_CLONE, uintptr(unix.SIGCHLD), 0, 0, 0, 0, 0)
	if e != 0 {
		return e
	}
	if pid == 0 { // the copy: leave before it runs any Go code
		_, _, _ = unix.RawSyscall(unix.SYS_EXIT_GROUP, 99, 0, 0)
	}
	var ws unix.WaitStatus
	_, _ = unix.Wait4(int(pid), &ws, 0, nil)
	return errors.New("a process was started")
}

// run runs every probe and returns each one's outcome.
func run(probes map[string]func() error) map[string]string {
	out := make(map[string]string, len(probes))
	for name, p := range probes {
		out[name] = outcome(p())
	}
	return out
}

// refused reports whether an outcome is a refusal: EACCES from Landlock, EPERM from the filter.
func refused(o string) bool { return o == "EACCES" || o == "EPERM" }
