// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

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
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/x/vt"
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
	return runEnv(t, nil, args...)
}

// runEnv runs wopr without a terminal, with extra environment variables.
func runEnv(t *testing.T, env []string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(binary, args...)
	cmd.Env = append(os.Environ(), env...)
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
	if _, errOut, code := run(t, "--movie"); code != 2 || !strings.Contains(errOut, "not a terminal") {
		t.Errorf("--movie without a terminal: exit %d, stderr %q; want 2 and an explanation", code, errOut)
	}
}

// Usage errors exit 2 before wopr looks for a terminal: --only without --movie, and a seed
// that is not a number, from the flag or from WOPR_SEED, each naming its source.
func TestUsageErrors(t *testing.T) {
	if _, errOut, code := run(t, "--only"); code != 2 || !strings.Contains(errOut, "--only needs --movie") {
		t.Errorf("--only alone: exit %d, stderr %q; want 2 and an explanation", code, errOut)
	}
	if _, errOut, code := runEnv(t, []string{"WOPR_SEED=x"}); code != 2 || !strings.Contains(errOut, `WOPR_SEED needs a non-negative integer, got "x"`) {
		t.Errorf("WOPR_SEED=x: exit %d, stderr %q; want 2 naming WOPR_SEED", code, errOut)
	}
	if _, errOut, code := runEnv(t, []string{"WOPR_SEED=1"}, "-s", "x"); code != 2 || !strings.Contains(errOut, "--seed needs") {
		t.Errorf("-s x: exit %d, stderr %q; want 2 naming --seed", code, errOut)
	}
}

// --scenes lists the movie's scenes; an unknown scene is a usage error that lists them.
func TestScenes(t *testing.T) {
	for _, flagName := range []string{"-S", "--scenes"} {
		out, _, code := run(t, flagName)
		if code != 0 || !strings.Contains(out, " 1. first-contact") || !strings.Contains(out, " 6. climax") {
			t.Errorf("%s: exit %d, output %q", flagName, code, out)
		}
	}
	_, errOut, code := run(t, "-m", "nowhere")
	if code != 2 || !strings.Contains(errOut, "no scene matches") || !strings.Contains(errOut, "norad-terminal") {
		t.Errorf("an unknown scene: exit %d, stderr %q; want 2 and the list", code, errOut)
	}
}

// --scenario lists GTW's scenarios; an unknown scenario is a usage error that names the list.
func TestScenarios(t *testing.T) {
	for _, flagName := range []string{"-c", "--scenario"} {
		out, _, code := run(t, flagName)
		if code != 0 || !strings.Contains(out, "  1. U.S. FIRST STRIKE") || !strings.Contains(out, "156. ") {
			t.Errorf("%s: exit %d, output %q", flagName, code, out)
		}
	}
	_, errOut, code := run(t, "-c", "nowhere")
	if code != 2 || !strings.Contains(errOut, "no scenario matches") {
		t.Errorf("an unknown scenario: exit %d, stderr %q; want 2", code, errOut)
	}
}

// session is a running wopr in a pseudo-terminal. Its output feeds a terminal emulator, so
// tests assert on the screen a user would see rather than on the byte stream: the typewriter
// and Bubble Tea's diffing renderer split text across writes. The emulator's answers to
// terminal queries go back to wopr, as a real terminal's would.
type session struct {
	t   *testing.T
	pty xpty.Pty
	cmd *exec.Cmd

	mu  sync.Mutex
	emu *vt.Emulator
	raw bytes.Buffer

	modes any // the pty's terminal modes before wopr started (Unix)
}

func start(t *testing.T, w, h int, args ...string) *session {
	t.Helper()
	return startEnv(t, w, h, nil, args...)
}

// startEnv starts wopr with extra environment variables.
func startEnv(t *testing.T, w, h int, env []string, args ...string) *session {
	t.Helper()
	return startProgram(t, w, h, env, binary, args...)
}

// startProgram starts any program, such as a tmux client, as startEnv starts wopr.
func startProgram(t *testing.T, w, h int, env []string, name string, args ...string) *session {
	t.Helper()
	p, err := xpty.NewPty(w, h)
	if err != nil {
		t.Fatalf("pty: %v", err)
	}
	cmd := exec.Command(name, args...)
	cmd.Env = append(append(os.Environ(), "TERM=xterm-256color"), env...)
	controllingTerminal(cmd)
	modes := terminalModes(t, p)
	if err := p.Start(cmd); err != nil {
		t.Fatalf("start: %v", err)
	}
	s := &session{t: t, pty: p, cmd: cmd, emu: vt.NewEmulator(w, h), modes: modes}
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := p.Read(buf)
			if n > 0 {
				s.mu.Lock()
				s.raw.Write(buf[:n])
				_, _ = s.emu.Write(buf[:n])
				s.mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()
	// Forward the emulator's query replies. The goroutine outlives the test, parked in Read:
	// Emulator.Close would end it, but Close races with that Read inside x/vt.
	go func() { _, _ = io.Copy(p, s.emu) }()
	t.Cleanup(func() { _ = p.Close() })
	return s
}

// output is everything wopr has written so far.
func (s *session) output() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.raw.String()
}

// screen is the emulator's current screen, one line per row.
func (s *session) screen() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.emu.String()
}

func (s *session) waitFor(text string, timeout time.Duration) {
	s.t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(s.screen(), text) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	s.t.Fatalf("timed out waiting for %q; screen:\n%s", text, s.screen())
}

func (s *session) send(keys string) {
	s.t.Helper()
	if _, err := io.WriteString(s.pty, keys); err != nil {
		s.t.Fatalf("write: %v", err)
	}
}

func (s *session) resize(w, h int) {
	s.t.Helper()
	s.mu.Lock()
	s.emu.Resize(w, h)
	s.mu.Unlock()
	if err := s.pty.Resize(w, h); err != nil {
		s.t.Fatalf("resize: %v", err)
	}
}

func (s *session) wait(timeout time.Duration) int {
	s.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err := xpty.WaitProcess(ctx, s.cmd)
	if ctx.Err() != nil {
		s.t.Fatalf("wopr did not exit; screen:\n%s", s.screen())
	}
	// Let the reader drain the last writes before the caller looks at the screen.
	time.Sleep(100 * time.Millisecond)
	checkModesRestored(s.t, s.pty, s.modes)
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
	if !strings.Contains(s.screen(), "--CONNECTION TERMINATED--") {
		t.Errorf("no exit line on the screen:\n%s", s.screen())
	}
	if runtime.GOOS != "windows" { // conhost re-renders, so raw escape sequences are Unix-only
		if out := s.output(); !strings.Contains(out, "\x1b[?1049h") || !strings.Contains(out, "\x1b[?1049l") {
			t.Error("alternate screen not entered and left")
		}
	}
}

// The film's opening: the backdoor password, the greeting, small talk, and a clean logoff.
func TestLogonConversationLogoff(t *testing.T) {
	s := start(t, 80, 24, "--instant")
	s.waitFor("LOGON:", 10*time.Second)
	s.send("Joshua\r")
	s.waitFor("GREETINGS PROFESSOR FALKEN.", 10*time.Second)
	s.send("Hello.\r")
	s.waitFor("HOW ARE YOU FEELING TODAY?", 10*time.Second)
	s.send("LOGOFF\r")
	if code := s.wait(10 * time.Second); code != 0 {
		t.Errorf("exit %d after LOGOFF, want 0", code)
	}
	if !strings.Contains(s.screen(), "--CONNECTION TERMINATED--") {
		t.Errorf("no exit line on the screen:\n%s", s.screen())
	}
}

// Movie mode (M6): the last scene, played with --instant, shows its text at once but keeps
// its pauses, draws the board, plays tic-tac-toe and the ending, and exits 0 without a key.
func TestMoviePlaysToTheEnd(t *testing.T) {
	s := start(t, 80, 24, "--movie", "climax", "--instant")
	s.waitFor("** GAME ROUTINE RUNNING **", 10*time.Second)
	if !strings.Contains(s.screen(), "LAUNCH CODE:") {
		t.Errorf("the climax opens on the big board:\n%s", s.screen())
	}
	s.waitFor("** ACCESS DENIED **", 30*time.Second)
	s.waitFor("NOT TO PLAY.", 30*time.Second) // the scene's last pause holds the film's last words
	if code := s.wait(30 * time.Second); code != 0 {
		t.Errorf("exit %d at the end of the movie, want 0", code)
	}
	if !strings.Contains(s.screen(), "--CONNECTION TERMINATED--") {
		t.Errorf("no exit line on the screen:\n%s", s.screen())
	}
}

// wopr -m joshua --only plays that scene alone, its pauses included, and exits 0 at its end
// without a key, instead of going on to the next scene (first-strike, which asks for a side).
func TestMovieOnlyPlaysOneScene(t *testing.T) {
	s := start(t, 80, 24, "-m", "joshua", "--only", "--instant")
	s.waitFor("GREETINGS PROFESSOR FALKEN.", 15*time.Second)
	s.waitFor("FINE.", 30*time.Second)
	deadline := time.Now().Add(30 * time.Second)
	for sc := s.screen(); !strings.Contains(sc, "--CONNECTION TERMINATED--"); sc = s.screen() {
		if strings.Contains(sc, "WHICH SIDE DO YOU WANT?") {
			t.Fatalf("the next scene started:\n%s", sc)
		}
		if time.Now().After(deadline) {
			t.Fatalf("wopr did not end after the scene; screen:\n%s", sc)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if code := s.wait(10 * time.Second); code != 0 {
		t.Errorf("exit %d at the end of the scene, want 0", code)
	}
}

// Movie mode's keys from a real terminal, paced: Space pauses (nothing moves), Right skips to
// the next scene, a lone Esc opens the menu, and Ctrl+C exits 130.
func TestMovieKeys(t *testing.T) {
	s := start(t, 80, 24, "-m", "2")
	s.waitFor("LOGON:", 15*time.Second)
	s.send(" ")
	s.waitFor("** PAUSED **", 5*time.Second)
	paused := s.screen()
	time.Sleep(time.Second)
	if s.screen() != paused {
		t.Errorf("the screen moved while paused:\n%s\nthen:\n%s", paused, s.screen())
	}
	s.send("\x1b[C") // Right
	s.waitFor("WHICH SIDE DO YOU WANT?", 15*time.Second)
	if strings.Contains(s.screen(), "LOGON:") || strings.Contains(s.screen(), "** PAUSED **") {
		t.Errorf("the next scene starts unpaused on a new page:\n%s", s.screen())
	}
	s.send("\x1b") // Esc
	s.waitFor("SCENE:", 10*time.Second)
	s.send("\x03")
	if code := s.wait(10 * time.Second); code != 130 {
		t.Errorf("exit %d after Ctrl+C, want 130", code)
	}
}

// wopr -c 2 plays USSR FIRST STRIKE on the big board with no orders asked, once Enter opens it
// after the briefing (--instant has no pause). Enter shows the kill ratios and the verdict, and
// Enter again opens the scenario menu.
func TestScenarioPlaysItself(t *testing.T) {
	s := start(t, 80, 24, "-c", "2", "--instant")
	s.waitFor("PRESS ENTER TO OPEN THE BOARD.", 15*time.Second) // --instant has no pause to read the briefing in
	s.send("\r")
	s.waitFor("GLOBAL THERMONUCLEAR WAR: USSR FIRST STRIKE", 15*time.Second)
	s.waitFor("STRIKE ASSESSMENT COMPLETE.", 30*time.Second)
	s.send("\r")
	s.waitFor("WINNER: NONE", 15*time.Second)
	s.send("\r")
	s.waitFor("SCENARIOS ON FILE: PAGE 1 OF 3", 15*time.Second)
	s.send("\x03")
	if code := s.wait(10 * time.Second); code != 130 {
		t.Errorf("exit %d after Ctrl+C, want 130", code)
	}
}

// wopr gtw -n asks for another war after the kill ratios, in place of the climax. YES starts
// one at the side choice, and NO returns to WOPR with no winner.
func TestNoClimax(t *testing.T) {
	s := start(t, 80, 24, "gtw", "-n", "--instant")
	war := func(side, target string) {
		t.Helper()
		s.waitFor("PLEASE CHOOSE ONE:", 15*time.Second)
		s.send(side + "\r")
		s.waitFor("PLEASE LIST PRIMARY TARGETS BY", 15*time.Second)
		s.send(target + "\r")
		s.waitFor("OF 8.", 15*time.Second)
		s.send("\r")
		for _, want := range []string{"STRIKE 2 OF 3", "STRIKE 3 OF 3", "STRIKE ASSESSMENT COMPLETE.", "PRESS ENTER TO CONTINUE."} {
			s.waitFor(want, 30*time.Second)
			s.send("\r")
		}
		s.waitFor("ANOTHER WAR, PROFESSOR?", 15*time.Second)
	}
	war("2", "Las Vegas")
	s.send("yes\r")
	war("1", "Moscow")
	s.send("no\r")
	s.waitFor("WINNER: NONE", 15*time.Second)
	s.send("\x03")
	if code := s.wait(10 * time.Second); code != 130 {
		t.Errorf("exit %d after Ctrl+C, want 130", code)
	}
}

// wopr gtw -a plays the film's war: a side and a target, then the board, the kill ratios and
// the climax by themselves, with no PRESS ENTER, even under --instant. TIC-TAC-TOE gets
// through, as in the film.
func TestAuthentic(t *testing.T) {
	s := start(t, 80, 24, "gtw", "-a", "--instant")
	s.waitFor("PLEASE CHOOSE ONE:", 15*time.Second)
	s.send("2\r")
	s.waitFor("PLEASE LIST PRIMARY TARGETS BY", 15*time.Second)
	s.send("Las Vegas\r")
	s.send("\r")
	s.waitFor("POPULATION DEATHS", 30*time.Second)
	s.waitFor("** GAME ROUTINE RUNNING **", 30*time.Second)
	s.send("TIC-TAC-TOE\r")
	s.waitFor("OF PLAYERS:", 15*time.Second)
	s.send("\x03")
	if code := s.wait(10 * time.Second); code != 130 {
		t.Errorf("exit %d after Ctrl+C, want 130", code)
	}
}

// wopr -m opens the scene menu; q leaves with exit 0.
func TestMovieMenuQuits(t *testing.T) {
	s := start(t, 80, 24, "-m", "--instant")
	s.waitFor("SCENE:", 10*time.Second)
	if !strings.Contains(s.screen(), "6.  CLIMAX") {
		t.Errorf("the menu lists the scenes:\n%s", s.screen())
	}
	s.send("q\r")
	if code := s.wait(10 * time.Second); code != 0 {
		t.Errorf("exit %d after q, want 0", code)
	}
}

// A terminal below 80x24 gets the TOO SMALL card; growing it resumes the session.
func TestTooSmallThenResize(t *testing.T) {
	s := start(t, 60, 20, "--instant")
	s.waitFor("TERMINAL TOO SMALL", 10*time.Second)
	s.resize(80, 24)
	s.waitFor("LOGON:", 10*time.Second)
	s.send("\x03")
	if code := s.wait(10 * time.Second); code != 130 {
		t.Errorf("exit %d after Ctrl+C, want 130", code)
	}
}

// The debug log records the session's seed, says where it is, and never holds typed input.
// Without a pinned seed it records the random one, so a bug report can replay the session
// (docs/PLAN.md §4.6); WOPR_SEED pins it. The unpinned run sets WOPR_SEED empty, which means
// unset, so a value in the test's own environment cannot leak in.
func TestDebugLog(t *testing.T) {
	for _, tc := range []struct {
		name, seedEnv string
		want          *regexp.Regexp
	}{
		{"random", "WOPR_SEED=", regexp.MustCompile(`seed [0-9]+ \(pinned: false\)`)},
		{"WOPR_SEED", "WOPR_SEED=1983", regexp.MustCompile(`seed 1983 \(pinned: true\)`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cache := t.TempDir()
			env := []string{"WOPR_DEBUG=1", tc.seedEnv, "XDG_CACHE_HOME=" + cache, "HOME=" + cache, "LocalAppData=" + cache}
			s := startEnv(t, 80, 24, env, "--instant")
			s.waitFor("LOGON:", 10*time.Second)
			s.send("Joshua\r")
			s.waitFor("GREETINGS PROFESSOR FALKEN.", 10*time.Second)
			s.send("\x03")
			s.wait(10 * time.Second)
			if !strings.Contains(s.screen(), "debug log:") {
				t.Errorf("the log's path is printed on exit:\n%s", s.screen())
			}
			var found string
			_ = filepath.WalkDir(cache, func(path string, d os.DirEntry, err error) error {
				if err == nil && d.Name() == "debug.log" {
					found = path
				}
				return nil
			})
			data, err := os.ReadFile(found)
			if err != nil {
				t.Fatalf("no debug log under %s: %v", cache, err)
			}
			if !tc.want.Match(data) {
				t.Errorf("the log must hold %q:\n%s", tc.want, data)
			}
			if strings.Contains(string(data), "Joshua") {
				t.Errorf("the log must hold no typed input:\n%s", data)
			}
		})
	}
}

// The libraries' own switches stay off (SECURITY.md): inside tmux, colour detection would
// run `tmux info` from the PATH, and TEA_TRACE would append raw input and output to any path.
// A fake tmux first on the PATH records any run. TEA_DEBUG writes only on a panic, and this
// Bubble Tea never reads UV_DEBUG, so for those two the test checks only that nothing appears;
// scrubenv's own test checks that all four are removed. Each variable that ends colour
// detection before it asks tmux is unset. wopr still draws in true colour there, as its debug
// log says. The fake is a shell script, so the test is Unix-only. test-snap.sh skips it: a snap
// has its own /tmp.
func TestLibrarySwitchesStayOff(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake tmux is a shell script")
	}
	for _, k := range []string{"COLORTERM", "NO_COLOR", "CLICOLOR_FORCE", "TTY_FORCE", "WT_SESSION", "GOOGLE_CLOUD_SHELL"} {
		t.Setenv(k, "") // restored after the test
		if err := os.Unsetenv(k); err != nil {
			t.Fatal(err)
		}
	}
	dir := t.TempDir()
	ran := filepath.Join(dir, "tmux-ran")
	fake := "#!/bin/sh\necho \"$@\" >> '" + ran + "'\n"
	if err := os.WriteFile(filepath.Join(dir, "tmux"), []byte(fake), 0o755); err != nil { //nolint:gosec // the fake must be executable
		t.Fatal(err)
	}
	trace, uvDebug := filepath.Join(dir, "tea-trace.log"), filepath.Join(dir, "uv-debug.log")
	cache := filepath.Join(dir, "cache")
	env := []string{
		"PATH=" + dir + string(os.PathListSeparator) + os.Getenv("PATH"),
		"TMUX=/tmp/tmux-1000/default,1,0",
		"TEA_TRACE=" + trace, "TEA_DEBUG=1", "UV_DEBUG=" + uvDebug,
		"WOPR_DEBUG=1", "XDG_CACHE_HOME=" + cache, "HOME=" + cache,
	}
	s := startEnv(t, 80, 24, env, "--instant")
	s.waitFor("LOGON:", 10*time.Second)
	s.send("\x03")
	s.wait(10 * time.Second)
	for _, f := range []string{ran, trace, uvDebug} {
		if data, err := os.ReadFile(f); err == nil {
			t.Errorf("%s exists:\n%s", filepath.Base(f), data)
		}
	}
	// TERM=xterm-256color alone gives 256 colours. Inside tmux, wopr draws in true colour.
	var log []byte
	_ = filepath.WalkDir(cache, func(path string, d os.DirEntry, err error) error {
		if err == nil && d.Name() == "debug.log" {
			log, _ = os.ReadFile(path)
		}
		return nil
	})
	if !strings.Contains(string(log), "colour profile TrueColor") {
		t.Errorf("inside tmux, the debug log names no true colour profile:\n%s", log)
	}
}

// A TERM that is a path does not make the terminfo loader read outside its folder
// (internal/scrubenv): /dev/zero would never end. --version still prints and exits.
func TestTermPathIgnored(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no /dev/zero")
	}
	s := startEnv(t, 80, 24, []string{"TERM=../../../../../../../../../dev/zero"}, "--version")
	if code := s.wait(10 * time.Second); code != 0 {
		t.Errorf("exit %d, want 0; screen:\n%s", code, s.screen())
	}
	if !strings.Contains(s.screen(), "wopr") {
		t.Errorf("no version line:\n%s", s.screen())
	}
}

// scrubenv must clear the environment before Lip Gloss reads it in a package-level variable
// (internal/scrubenv). Go's own init trace shows the order the binary really runs inits in.
// test-snap.sh skips it, as snapd may not pass GODEBUG through.
func TestScrubenvInitsBeforeLipGloss(t *testing.T) {
	_, stderr, code := runEnv(t, []string{"GODEBUG=inittrace=1"}, "--version")
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, stderr)
	}
	scrub := strings.Index(stderr, "init github.com/GhostofGoes/WOPR/internal/scrubenv ")
	for _, pkg := range []string{"github.com/charmbracelet/colorprofile", "charm.land/lipgloss/v2"} {
		i := strings.Index(stderr, "init "+pkg+" ")
		switch {
		case scrub < 0 || i < 0:
			t.Fatalf("the init trace lacks scrubenv or %s:\n%s", pkg, stderr)
		case scrub > i:
			t.Errorf("%s initialises before scrubenv", pkg)
		}
	}
}
