//go:build unix

package main

import (
	"errors"
	"os/signal"
	"syscall"
)

const windowsHint = ""

// ignoreSIGPIPE lets a write to a closed pipe return EPIPE instead of killing the
// process, so `wopr --games | head -1` exits 0.
func ignoreSIGPIPE() { signal.Ignore(syscall.SIGPIPE) }

func isBrokenPipe(err error) bool { return errors.Is(err, syscall.EPIPE) }
