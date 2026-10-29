// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

//go:build e2e && !windows

package e2e

import (
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Ctrl+Z suspends wopr and fg continues it: the session carries on where it was.
func TestSuspendResume(t *testing.T) {
	s := start(t, 80, 24, "--instant")
	s.waitFor("LOGON:", 10*time.Second)
	suspend(t, s)
	s.send("Joshua\r")
	s.waitFor("GREETINGS PROFESSOR FALKEN.", 10*time.Second)
	s.send("\x03")
	if code := s.wait(10 * time.Second); code != 130 {
		t.Errorf("exit %d after Ctrl+C, want 130", code)
	}
}

// SIGTERM ends the session as LOGOFF does: the terminal is restored and the log says so.
func TestSIGTERM(t *testing.T) {
	env, cache := debugEnv(t)
	s := startEnv(t, 80, 24, env, "--instant")
	s.waitFor("LOGON:", 10*time.Second)
	if err := s.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	if code := s.wait(10 * time.Second); code != 0 {
		t.Errorf("exit %d after SIGTERM, want 0", code)
	}
	inOrder(t, debugLog(t, cache), ` sandbox: `, ` session ended: outcome 0, `)
}

var sgr = regexp.MustCompile(`\x1b\[([0-9;:]*)m`)

// colours lists the colour parameters of every SGR sequence in out (foreground, background).
func colours(out string) []string {
	var found []string
	for _, m := range sgr.FindAllStringSubmatch(out, -1) {
		for _, p := range strings.FieldsFunc(m[1], func(r rune) bool { return r == ';' || r == ':' }) {
			n, _ := strconv.Atoi(p)
			if n >= 30 && n <= 49 || n >= 90 && n <= 107 {
				found = append(found, p)
			}
		}
	}
	return found
}

// Every theme draws in colour on a colour terminal, and NO_COLOR draws none: the colour
// detection and the themes run before the lockdown, and the screen is drawn after it.
func TestColours(t *testing.T) {
	for _, theme := range []string{"imsai", "green", "amber", "norad"} {
		t.Run(theme, func(t *testing.T) {
			s := start(t, 80, 24, "--instant", "--theme", theme)
			s.waitFor("LOGON:", 10*time.Second)
			s.send("\x03")
			s.wait(10 * time.Second)
			if len(colours(s.output())) == 0 {
				t.Error("no colour on a colour terminal")
			}
		})
	}
	t.Run("NO_COLOR", func(t *testing.T) {
		s := startEnv(t, 80, 24, []string{"NO_COLOR=1"}, "--instant")
		s.waitFor("LOGON:", 10*time.Second)
		s.send("Joshua\r")
		s.waitFor("GREETINGS PROFESSOR FALKEN.", 10*time.Second)
		s.send("\x03")
		s.wait(10 * time.Second)
		if c := colours(s.output()); len(c) > 0 {
			t.Errorf("colour under NO_COLOR: SGR parameters %v", c)
		}
	})
}

// suspend presses Ctrl+Z, waits for wopr to give the terminal back, and continues it, as a
// shell's fg does. wopr leads its own session here, so the kernel may discard the stop, but
// wopr still waits for SIGCONT.
func suspend(t *testing.T, s *session) {
	t.Helper()
	left := strings.Count(s.output(), "\x1b[?1049l")
	s.send("\x1a")
	deadline := time.Now().Add(10 * time.Second)
	for strings.Count(s.output(), "\x1b[?1049l") == left {
		if time.Now().After(deadline) {
			t.Fatalf("Ctrl+Z did not give the terminal back; screen:\n%s", s.screen())
		}
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(200 * time.Millisecond) // let it reach the wait for SIGCONT
	entered := strings.Count(s.output(), "\x1b[?1049h")
	if err := s.cmd.Process.Signal(syscall.SIGCONT); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(10 * time.Second)
	for strings.Count(s.output(), "\x1b[?1049h") < entered+1 {
		if time.Now().After(deadline) {
			t.Fatalf("wopr did not take the terminal back after SIGCONT; screen:\n%s", s.screen())
		}
		time.Sleep(20 * time.Millisecond)
	}
}
