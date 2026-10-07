//go:build e2e && !windows

package e2e

import (
	"os/exec"
	"reflect"
	"syscall"
	"testing"

	"github.com/charmbracelet/x/term"
	"github.com/charmbracelet/x/xpty"
)

// controllingTerminal makes the pty's slave end, the child's stdin, its controlling terminal
// in a new session, so a resize reaches it as SIGWINCH. xpty leaves this to the caller.
func controllingTerminal(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
}

// terminalModes snapshots the pty's terminal modes (termios). It reads them through the
// master, which shares the slave's modes: on macOS the slave descriptor is revoked when the
// session leader (wopr) exits, and any ioctl on it then fails with ENOTTY.
func terminalModes(t *testing.T, p xpty.Pty) any {
	t.Helper()
	up, ok := p.(*xpty.UnixPty)
	if !ok {
		t.Fatalf("unexpected pty type %T", p)
	}
	st, err := term.GetState(up.Master().Fd())
	if err != nil {
		t.Fatalf("termios: %v", err)
	}
	return st
}

// checkModesRestored fails if wopr left the terminal in a different mode (raw, no echo).
func checkModesRestored(t *testing.T, p xpty.Pty, before any) {
	t.Helper()
	if after := terminalModes(t, p); !reflect.DeepEqual(before, after) {
		t.Error("terminal modes were not restored on exit")
	}
}
