package main

import (
	"io"
	"os"
	"path"
	"slices"
	"strings"
)

// launch is how the program was started, as the macOS relaunch in Terminal sees it.
type launch struct {
	exe                 string   // the program's absolute path, symlinks resolved
	ppid                int      // the parent process's ID
	stdinTTY, stdoutTTY bool     // whether standard input and output are terminals
	args                []string // os.Args[1:]
}

// launchdPID is launchd's process ID. LaunchServices has launchd start the apps that Finder, the
// Dock, Spotlight and `open` open, so launchd is their parent.
const launchdPID = 1

// runFunc runs a command and returns its exit status; the error is for a command that could not
// run at all.
type runFunc func(name string, args ...string) (status int, err error)

// openInTerminal opens a file in Terminal through LaunchServices, and Terminal runs an executable
// it is given in a new window, with a terminal. It sends Terminal no Apple events, which would
// need the user's permission to control Terminal.
var openInTerminal = []string{"/usr/bin/open", "-b", "com.apple.Terminal"}

// relaunch reopens wopr in Terminal when Finder (or `open`) started it from a macOS app bundle:
// its program is in <name>.app/Contents/MacOS/, its parent is launchd, neither standard input nor
// output is a terminal, and its only argument, if any, is the -psn_ one older versions of macOS
// pass. Terminal then runs the same program again with a terminal, and that run starts as usual.
// In every other case relaunch does nothing and returns false: a shell pipeline or a test that
// runs the bundle's program without a terminal gets today's "not a terminal" error. Otherwise it
// returns exitOK, or exitError when open fails, after saying why on stderr.
//
// A quarantined app run from Downloads or the disk image runs from a randomised, read-only copy
// (App Translocation), which macOS may remove when this process exits. Terminal is given that
// path, so the install instructions say to drag WOPR to Applications first.
func relaunch(l launch, run runFunc, stderr io.Writer) (code int, relaunched bool) {
	if l.ppid != launchdPID || l.stdinTTY || l.stdoutTTY || !launchServicesArgs(l.args) || !inAppBundle(l.exe) {
		return 0, false
	}
	argv := append(slices.Clone(openInTerminal), l.exe)
	status, err := run(argv[0], argv[1:]...)
	switch {
	case err != nil:
		warn(stderr, "cannot open Terminal: %v", err)
		return exitError, true
	case status != 0:
		warn(stderr, "cannot open Terminal: %s exited with status %d", strings.Join(argv, " "), status)
		return exitError, true
	}
	return exitOK, true
}

// launchServicesArgs reports whether args are what LaunchServices passes an app: nothing, or a
// process serial number (-psn_0_12345) on versions of macOS before 10.9.
func launchServicesArgs(args []string) bool {
	return len(args) == 0 || len(args) == 1 && strings.HasPrefix(args[0], "-psn_")
}

// inAppBundle reports whether exe is a bundle's program: /path/<name>.app/Contents/MacOS/<file>.
// macOS paths use slashes, so this uses package path, not filepath.
func inAppBundle(exe string) bool {
	macOS := path.Dir(exe)
	contents := path.Dir(macOS)
	app := path.Base(path.Dir(contents))
	return path.IsAbs(exe) && path.Base(macOS) == "MacOS" && path.Base(contents) == "Contents" &&
		strings.HasSuffix(app, ".app") && len(app) > len(".app")
}

// isTerminal reports whether f is a terminal, here a character device other than the null
// device. LaunchServices starts an app with /dev/null as its standard input, and a terminal is the
// only other character device a program's standard streams meet in practice. The terminal driver
// would know for sure, but asking it needs golang.org/x/sys or unsafe, which cmd/wopr does not
// import (internal/archtest).
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	null, err := os.Stat(os.DevNull)
	return err != nil || !os.SameFile(fi, null)
}
