//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/x/xpty"
)

var binaryFlag = flag.String("binary", "", "path to the wopr binary (default: build ./cmd/wopr)")

var binary string

func TestMain(m *testing.M) {
	flag.Parse()
	binary = *binaryFlag
	if binary == "" {
		dir, err := os.MkdirTemp("", "wopr-e2e")
		if err != nil {
			panic(err)
		}
		binary = filepath.Join(dir, "wopr")
		if runtime.GOOS == "windows" {
			binary += ".exe"
		}
		build := exec.Command("go", "build", "-o", binary, "github.com/GhostofGoes/WOPR/cmd/wopr")
		build.Stdout, build.Stderr = os.Stdout, os.Stderr
		if err := build.Run(); err != nil {
			panic("building wopr: " + err.Error())
		}
		code := m.Run()
		_ = os.RemoveAll(dir)
		os.Exit(code)
	}
	os.Exit(m.Run())
}

func exitCode(err error) int {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	if err != nil {
		return -1
	}
	return 0
}

func run(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(binary, args...)
	var o, e bytes.Buffer
	cmd.Stdout, cmd.Stderr = &o, &e
	err := cmd.Run()
	return o.String(), e.String(), exitCode(err)
}

func TestVersion(t *testing.T) {
	for _, flagName := range []string{"-v", "--version"} {
		out, _, code := run(t, flagName)
		if code != 0 || !strings.HasPrefix(out, "wopr ") {
			t.Errorf("%s: exit %d, output %q", flagName, code, out)
		}
	}
}

func TestGamesAndLicenses(t *testing.T) {
	out, _, code := run(t, "--games")
	if code != 0 || !strings.Contains(out, "GLOBAL THERMONUCLEAR WAR") {
		t.Errorf("--games: exit %d, output %q", code, out)
	}
	out, _, code = run(t, "-L")
	if code != 0 || !strings.Contains(out, "THIRD-PARTY NOTICES") || !strings.Contains(out, "MIT License") {
		t.Errorf("--licenses: exit %d", code)
	}
}

// `wopr --games | head -1` must exit 0: the read end is closed before wopr writes, so the
// write fails with a broken pipe on every OS.
func TestClosedPipe(t *testing.T) {
	cmd := exec.Command(binary, "--games")
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stdout = w
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
	if code := exitCode(cmd.Wait()); code != 0 {
		t.Errorf("exit %d writing to a closed pipe, want 0", code)
	}
}

func TestRefusesWithoutTerminal(t *testing.T) {
	_, errOut, code := run(t)
	if code != 2 || !strings.Contains(errOut, "not a terminal") {
		t.Errorf("exit %d, stderr %q; want 2 and an explanation", code, errOut)
	}
	_, _, code = run(t, "gtw", "chess")
	if code != 2 {
		t.Errorf("usage error exit %d, want 2", code)
	}
}

// session is a running wopr in a pseudo-terminal.
type session struct {
	t   *testing.T
	pty xpty.Pty
	cmd *exec.Cmd
	mu  sync.Mutex
	out bytes.Buffer
}

func start(t *testing.T, w, h int, args ...string) *session {
	t.Helper()
	p, err := xpty.NewPty(w, h)
	if err != nil {
		t.Fatalf("pty: %v", err)
	}
	cmd := exec.Command(binary, args...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	if err := p.Start(cmd); err != nil {
		t.Fatalf("start: %v", err)
	}
	s := &session{t: t, pty: p, cmd: cmd}
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := p.Read(buf)
			if n > 0 {
				s.mu.Lock()
				s.out.Write(buf[:n])
				s.mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()
	t.Cleanup(func() { _ = p.Close() })
	return s
}

func (s *session) output() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.out.String()
}

func (s *session) waitFor(text string, timeout time.Duration) {
	s.t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(s.output(), text) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	s.t.Fatalf("timed out waiting for %q; output so far:\n%q", text, s.output())
}

func (s *session) send(keys string) {
	s.t.Helper()
	if _, err := io.WriteString(s.pty, keys); err != nil {
		s.t.Fatalf("write: %v", err)
	}
}

func (s *session) wait(timeout time.Duration) int {
	s.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err := xpty.WaitProcess(ctx, s.cmd)
	if ctx.Err() != nil {
		s.t.Fatalf("wopr did not exit; output:\n%q", s.output())
	}
	if s.cmd.ProcessState != nil {
		return s.cmd.ProcessState.ExitCode()
	}
	return exitCode(err)
}

func TestTUIStartsAndInterrupts(t *testing.T) {
	s := start(t, 80, 24, "--instant")
	s.waitFor("LOGON:", 10*time.Second)
	s.send("\x03") // Ctrl+C
	if code := s.wait(10 * time.Second); code != 130 {
		t.Errorf("exit %d after Ctrl+C, want 130", code)
	}
	time.Sleep(100 * time.Millisecond)
	if !strings.Contains(s.output(), "--CONNECTION TERMINATED--") {
		t.Errorf("no exit line in output:\n%q", s.output())
	}
	if runtime.GOOS != "windows" { // conhost re-renders, so raw escape sequences are Unix-only
		if out := s.output(); !strings.Contains(out, "\x1b[?1049h") || !strings.Contains(out, "\x1b[?1049l") {
			t.Error("alternate screen not entered and left")
		}
	}
}
