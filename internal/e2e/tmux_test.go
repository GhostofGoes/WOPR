// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

//go:build e2e && unix

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// tmuxServer is a private tmux server: its own socket, no configuration file, and an
// environment without the developer's tmux or colour settings. Tests drive it as a user
// would, with send-keys, and read the pane with capture-pane, as fzf's and other TUIs'
// tmux tests do. Without tmux, the tests skip, unless WOPR_E2E_TMUX is set (CI's Linux
// smoke jobs), where a missing tmux fails them.
type tmuxServer struct {
	t    *testing.T
	sock string
	env  []string
}

func newTmux(t *testing.T) *tmuxServer {
	t.Helper()
	if _, err := exec.LookPath("tmux"); err != nil {
		if os.Getenv("WOPR_E2E_TMUX") != "" {
			t.Fatal("WOPR_E2E_TMUX is set, but tmux is not on the PATH")
		}
		t.Skip("tmux is not installed")
	}
	// Not t.TempDir: a socket's path must stay short (104 bytes on macOS).
	dir, err := os.MkdirTemp("", "wopr-tmux")
	if err != nil {
		t.Fatal(err)
	}
	var env []string
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		switch {
		case strings.HasPrefix(name, "TMUX"), strings.HasPrefix(name, "WOPR_"),
			name == "NO_COLOR", name == "COLORTERM", name == "TERM":
			continue
		}
		env = append(env, kv)
	}
	s := &tmuxServer{t: t, sock: filepath.Join(dir, "s"), env: env}
	t.Cleanup(func() {
		_ = exec.Command("tmux", "-S", s.sock, "kill-server").Run() //nolint:gosec // the test's own socket
		_ = os.RemoveAll(dir)
	})
	return s
}

// run runs a tmux command against the private server and returns its output.
func (s *tmuxServer) run(args ...string) string {
	s.t.Helper()
	cmd := exec.Command("tmux", append([]string{"-S", s.sock, "-f", os.DevNull}, args...)...) //nolint:gosec // fixed program, test arguments
	cmd.Env = s.env
	out, err := cmd.CombinedOutput()
	if err != nil {
		s.t.Fatalf("tmux %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// start starts wopr in an 80x24 pane of a new session named wopr. tmux's own TERM,
// tmux-256color, gives 256 colours without help, and a dead pane stays to report its status.
func (s *tmuxServer) start(env []string, args ...string) {
	s.t.Helper()
	cmd := []string{
		"start-server", ";",
		"set-option", "-g", "default-terminal", "tmux-256color", ";",
		"set-option", "-g", "remain-on-exit", "on", ";",
		"new-session", "-d", "-s", "wopr", "-x", "80", "-y", "24",
	}
	for _, kv := range env {
		cmd = append(cmd, "-e", kv)
	}
	s.run(append(append(cmd, "--", binary), args...)...)
}

// capture is the pane's screen. With escapes, it keeps the colours and attributes.
func (s *tmuxServer) capture(escapes bool) string {
	s.t.Helper()
	if escapes {
		return s.run("capture-pane", "-p", "-e", "-t", "wopr")
	}
	return s.run("capture-pane", "-p", "-t", "wopr")
}

func (s *tmuxServer) waitFor(text string, timeout time.Duration) {
	s.t.Helper()
	for deadline := time.Now().Add(timeout); ; time.Sleep(50 * time.Millisecond) {
		screen := s.capture(false)
		if strings.Contains(screen, text) {
			return
		}
		if time.Now().After(deadline) {
			s.t.Fatalf("timed out waiting for %q; pane:\n%s", text, screen)
		}
	}
}

// typeLine types text and presses Enter.
func (s *tmuxServer) typeLine(text string) {
	s.t.Helper()
	s.run("send-keys", "-t", "wopr", "-l", text)
	s.run("send-keys", "-t", "wopr", "Enter")
}

// exitStatus waits for wopr to exit and returns its status.
func (s *tmuxServer) exitStatus(timeout time.Duration) string {
	s.t.Helper()
	for deadline := time.Now().Add(timeout); ; time.Sleep(50 * time.Millisecond) {
		out := strings.TrimSpace(s.run("display-message", "-p", "-t", "wopr", "#{pane_dead} #{pane_dead_status}"))
		if dead, status, _ := strings.Cut(out, " "); dead == "1" {
			return status
		}
		if time.Now().After(deadline) {
			s.t.Fatalf("wopr did not exit; pane:\n%s", s.capture(false))
		}
	}
}

// sgrColour matches an SGR sequence that sets a foreground or background colour.
var sgrColour = regexp.MustCompile(`\x1b\[(?:[0-9:]*;)*(?:3[0-8]|4[0-8]|9[0-7]|10[0-7])[;:m]`)

// wopr works inside a real tmux: the opening, a reply, LOGOFF, and the terminal restored.
// tmux's TERM alone would give 256 colours, so 24-bit colour in the pane shows that wopr
// draws in true colour inside tmux. NO_COLOR still turns colour off there.
func TestTmux(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  []string
		want func(string) bool
	}{
		{"true colour", nil, func(p string) bool { return strings.Contains(p, "\x1b[38;2;") }},
		{"NO_COLOR", []string{"NO_COLOR=1"}, func(p string) bool { return !sgrColour.MatchString(p) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newTmux(t)
			s.start(tc.env, "--instant")
			s.waitFor("LOGON:", 10*time.Second)
			if pane := s.capture(true); !tc.want(pane) {
				t.Errorf("wrong colours in the pane:\n%q", pane)
			}
			s.typeLine("Joshua")
			s.waitFor("GREETINGS PROFESSOR FALKEN.", 10*time.Second)
			s.typeLine("Hello.")
			s.waitFor("HOW ARE YOU FEELING TODAY?", 10*time.Second)
			s.typeLine("LOGOFF")
			if status := s.exitStatus(10 * time.Second); status != "0" {
				t.Errorf("exit %s after LOGOFF, want 0", status)
			}
			// The alternate screen is left: the pane and its history hold only the exit line.
			if pane := s.run("capture-pane", "-p", "-S", "-", "-t", "wopr"); !strings.Contains(pane, "--CONNECTION TERMINATED--") || strings.Contains(pane, "LOGON:") {
				t.Errorf("the pane after LOGOFF:\n%s", pane)
			}
		})
	}
}

// An outer terminal without 24-bit colour still gets colour: tmux converts wopr's colours
// to the 256 that the terminal's TERM names (SECURITY.md relies on this). A client attaches
// from a pseudo-terminal with TERM=xterm-256color, which tmux does not take for true colour.
func TestTmuxConvertsForOuterTerminal(t *testing.T) {
	s := newTmux(t)
	s.start(nil, "--instant")
	s.waitFor("LOGON:", 10*time.Second)
	// TMUX= keeps a developer's own tmux from refusing the nested client.
	c := startProgram(t, 80, 25, append(s.env, "TMUX="), "tmux", "-S", s.sock, "attach-session", "-t", "wopr")
	c.waitFor("LOGON:", 10*time.Second)
	out := c.output()
	if !strings.Contains(out, "38;5;") || strings.Contains(out, "38;2;") {
		t.Errorf("the client got no 256-colour codes, or 24-bit ones:\n%q", out)
	}
	s.run("kill-session", "-t", "wopr")
}
