//go:build windows

package main

import (
	"errors"
	"syscall"
)

const windowsHint = " (run wopr in Windows Terminal or another ConPTY console; mintty needs winpty)"

func ignoreSIGPIPE() {}

const (
	errorBrokenPipe syscall.Errno = 109 // ERROR_BROKEN_PIPE
	errorNoData     syscall.Errno = 232 // ERROR_NO_DATA: the pipe is being closed
)

func isBrokenPipe(err error) bool {
	return errors.Is(err, errorBrokenPipe) || errors.Is(err, errorNoData) || errors.Is(err, syscall.EPIPE)
}
