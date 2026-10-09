//go:build !darwin

package main

// relaunchInTerminal does nothing: only macOS starts wopr from an app bundle without a terminal
// (launch.go). Windows and Linux shortcuts open a terminal themselves.
func relaunchInTerminal() (code int, relaunched bool) { return 0, false }
