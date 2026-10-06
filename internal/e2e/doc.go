// Package e2e drives the real wopr binary: as a plain process for the print-and-exit
// commands, and in a pseudo-terminal (a Unix pty, or ConPTY on Windows) for the TUI.
//
// The tests are behind the e2e build tag. Locally:
//
//	go test -tags e2e ./internal/e2e            # builds ./cmd/wopr first
//
// In CI the build job cross-compiles this test per target (go test -c -tags e2e) and the
// smoke job runs it next to the release binary, without a Go toolchain:
//
//	./e2e.test -test.v -binary ./wopr
package e2e
