package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
)

// relaunchInTerminal reopens wopr in Terminal when Finder started it from WOPR.app, and reports
// whether it did, with the exit code (launch.go).
func relaunchInTerminal() (code int, relaunched bool) {
	exe, err := os.Executable()
	if err != nil {
		return 0, false
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		return 0, false
	}
	l := launch{exe: exe, ppid: os.Getppid(), stdinTTY: isTerminal(os.Stdin), stdoutTTY: isTerminal(os.Stdout), args: os.Args[1:]}
	return relaunch(l, runCommand, os.Stderr)
}

// runCommand runs a command with this process's standard output and error, and returns its exit
// status.
func runCommand(name string, args ...string) (int, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	err := cmd.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), nil
	}
	return 0, err
}
