//go:build e2e

package e2e

import "os/exec"

// controllingTerminal is a no-op: ConPTY delivers resizes as console input events.
func controllingTerminal(*exec.Cmd) {}
