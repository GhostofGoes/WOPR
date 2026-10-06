//go:build e2e && !windows

package e2e

import (
	"os/exec"
	"syscall"
)

// controllingTerminal makes the pty's slave end, the child's stdin, its controlling terminal
// in a new session, so a resize reaches it as SIGWINCH. xpty leaves this to the caller.
func controllingTerminal(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
}
