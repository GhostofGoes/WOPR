// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/ui"
)

func env(vars map[string]string) func(string) string {
	return func(k string) string { return vars[k] }
}

// run's print-and-exit actions, its usage errors, and its refusals to start the full-screen
// interface. Under go test, standard output is not a terminal, so run never reaches ui.Run here;
// the e2e tests run the program in a pseudo-terminal.
func TestRun(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		args       []string
		env        map[string]string
		code       int
		stdout     string // a substring that stdout must hold, or "" for none at all
		stderr     string // likewise for stderr
		quietError bool   // stderr must be empty
	}{
		{name: "version", args: []string{"--version"}, stdout: "wopr", quietError: true},
		{name: "help", args: []string{"-h"}, stdout: "--seed", quietError: true},
		{name: "games", args: []string{"--games"}, stdout: "TIC-TAC-TOE", quietError: true},
		{name: "scenes", args: []string{"-S"}, stdout: "\n", quietError: true},
		{name: "licenses", args: []string{"-L"}, stdout: "Copyright", quietError: true},
		{name: "unknown flag", args: []string{"--frobnicate"}, code: exitUsage, stderr: "Run 'wopr --help' for usage."},
		{name: "bad seed", args: []string{"--seed", "x"}, code: exitUsage, stderr: "non-negative integer"},
		{name: "dumb terminal", env: map[string]string{"TERM": "dumb"}, code: exitUsage, stderr: "TERM=dumb"},
		{name: "not a terminal", env: map[string]string{"TERM": "xterm"}, code: exitUsage, stderr: "not a terminal"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.name == "not a terminal" && stdoutIsCharDevice() {
				t.Skip("the test binary's standard output is a terminal, so run would start the interface")
			}
			var stdout, stderr strings.Builder
			if code := run(tc.args, &stdout, &stderr, env(tc.env)); code != tc.code {
				t.Errorf("exit %d, want %d; stderr %q", code, tc.code, stderr.String())
			}
			if tc.stdout == "" && stdout.Len() > 0 {
				t.Errorf("stdout %q, want nothing", stdout.String())
			}
			if !strings.Contains(stdout.String(), tc.stdout) {
				t.Errorf("stdout %q lacks %q", stdout.String(), tc.stdout)
			}
			if tc.quietError && stderr.Len() > 0 {
				t.Errorf("stderr %q, want nothing", stderr.String())
			}
			if !strings.Contains(stderr.String(), tc.stderr) {
				t.Errorf("stderr %q lacks %q", stderr.String(), tc.stderr)
			}
		})
	}
}

func stdoutIsCharDevice() bool {
	fi, err := os.Stdout.Stat()
	return err != nil || fi.Mode()&os.ModeCharDevice != 0
}

type failWriter struct{ err error }

func (w failWriter) Write([]byte) (int, error) { return 0, w.err }

// A closed pipe (wopr --games | head) is success; any other write error fails, and says why.
func TestRunPrintFailures(t *testing.T) {
	t.Parallel()
	var stderr strings.Builder
	if code := run([]string{"--games"}, failWriter{fmt.Errorf("write: %w", syscall.EPIPE)}, &stderr, env(nil)); code != exitOK {
		t.Errorf("broken pipe: exit %d, want %d", code, exitOK)
	}
	if stderr.Len() > 0 {
		t.Errorf("broken pipe: stderr %q, want nothing", stderr.String())
	}
	if code := run([]string{"--version"}, failWriter{errors.New("disk full")}, &stderr, env(nil)); code != exitError {
		t.Errorf("write error: exit %d, want %d", code, exitError)
	}
	if !strings.Contains(stderr.String(), "wopr: disk full") {
		t.Errorf("write error: stderr %q lacks the error", stderr.String())
	}
}

func TestPanelSetting(t *testing.T) {
	t.Parallel()
	for v, want := range map[string]ui.Panel{"": ui.PanelDefault, "1": ui.PanelOn, "yes": ui.PanelOn, "0": ui.PanelOff, "off": ui.PanelOff} {
		if got := panelSetting(env(map[string]string{"WOPR_PANEL": v})); got != want {
			t.Errorf("WOPR_PANEL=%q: %v, want %v", v, got, want)
		}
	}
}

func TestSessionSeedDiffers(t *testing.T) {
	t.Parallel()
	if a, b := sessionSeed(), sessionSeed(); a == b {
		t.Error("two session seeds are equal") // one chance in 2^64
	}
}

// openDebugLog stays off without WOPR_DEBUG, opens a log in the cache directory with it, and
// stays off, saying why, when it cannot. It changes the environment, so it does not run in
// parallel.
func TestOpenDebugLog(t *testing.T) {
	var stderr strings.Builder
	if lg := openDebugLog(env(map[string]string{"WOPR_DEBUG": "0"}), &stderr); lg != nil {
		t.Error("WOPR_DEBUG=0 opened a log")
	}

	cache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cache) // Linux and the BSDs
	t.Setenv("HOME", cache)           // macOS: $HOME/Library/Caches
	t.Setenv("LocalAppData", cache)   // Windows
	lg := openDebugLog(env(map[string]string{"WOPR_DEBUG": "1"}), &stderr)
	if lg == nil {
		t.Fatalf("WOPR_DEBUG=1 opened no log; stderr %q", stderr.String())
	}
	if !strings.HasPrefix(lg.Path(), cache) {
		t.Errorf("log %s is outside the cache directory %s", lg.Path(), cache)
	}
	if err := lg.Close(); err != nil {
		t.Fatal(err)
	}

	// A file where the log's directory should be.
	blocked := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", blocked)
	t.Setenv("HOME", blocked)
	t.Setenv("LocalAppData", blocked)
	dir, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "wopr"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	stderr.Reset()
	if lg := openDebugLog(env(map[string]string{"WOPR_DEBUG": "1"}), &stderr); lg != nil {
		_ = lg.Close()
		t.Fatalf("opened a log at %s past a file in the way", lg.Path())
	}
	if !strings.Contains(stderr.String(), "not logging") {
		t.Errorf("stderr %q does not say why logging is off", stderr.String())
	}

	// No cache directory at all.
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("HOME", "")
	t.Setenv("LocalAppData", "")
	if _, err := os.UserCacheDir(); err == nil {
		t.Skip("this system has a cache directory without the environment")
	}
	stderr.Reset()
	if lg := openDebugLog(env(map[string]string{"WOPR_DEBUG": "1"}), &stderr); lg != nil {
		_ = lg.Close()
		t.Fatal("opened a log without a cache directory")
	}
	if !strings.Contains(stderr.String(), "no cache directory") {
		t.Errorf("stderr %q does not say why logging is off", stderr.String())
	}
}
