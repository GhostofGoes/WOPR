//go:build e2e

package e2e

import (
	"os/exec"
	"testing"

	"github.com/charmbracelet/x/xpty"
)

// controllingTerminal is a no-op: ConPTY delivers resizes as console input events.
func controllingTerminal(*exec.Cmd) {}

// Under ConPTY, conhost owns the console modes, so there is nothing comparable to check;
// the exit code and the final screen are the assertions on Windows.
func terminalModes(*testing.T, xpty.Pty) any { return nil }

func checkModesRestored(*testing.T, xpty.Pty, any) {}
