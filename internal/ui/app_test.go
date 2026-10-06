package ui

import (
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games/catalog"
	"github.com/GhostofGoes/WOPR/internal/games/gamestest"
	"github.com/GhostofGoes/WOPR/internal/golden"
)

func instant() Options {
	return Options{Instant: true, Seed: 1, SeedSet: true, Registry: catalog.Registry()}
}

func TestLogonJoshua(t *testing.T) {
	t.Parallel()
	var s snapshots
	d := newDriver(t, instant(), 80, 24).settle()
	s.add("dial and LOGON", d)
	d.line("Joshua")
	s.add("greeting", d)
	d.line("Hello.")
	s.add("small talk", d)
	if !strings.Contains(d.screen(), "HOW ARE YOU FEELING TODAY?") {
		t.Errorf("no reply to Hello:\n%s", d.screen())
	}
	golden.AssertString(t, "logon_joshua", s.String())
}

func TestLogoffExitsCleanly(t *testing.T) {
	t.Parallel()
	var s snapshots
	d := newDriver(t, instant(), 80, 24).settle().line("Joshua").line("LOGOFF")
	s.add("after LOGOFF", d)
	if d.ended != "quit" {
		t.Errorf("LOGOFF ended with %q, want quit", d.ended)
	}
	golden.AssertString(t, "logoff_exit0", s.String())
}

func TestCtrlCAndCtrlD(t *testing.T) {
	t.Parallel()
	if d := newDriver(t, instant(), 80, 24).settle().key(ctrlC); d.ended != "interrupt" {
		t.Errorf("Ctrl+C ended with %q, want interrupt", d.ended)
	}
	if d := newDriver(t, instant(), 80, 24).settle().typ("x").key(ctrlD); d.ended != "" {
		t.Errorf("Ctrl+D on a non-empty line must not quit (ended %q)", d.ended)
	}
	if d := newDriver(t, instant(), 80, 24).settle().key(ctrlD); d.ended != "quit" {
		t.Errorf("Ctrl+D on an empty line ended with %q, want quit", d.ended)
	}
}

// Shrinking below 80x24 freezes the session behind the TOO SMALL card; growing it resumes
// exactly where it stopped.
func TestTooSmallPausesAndResumes(t *testing.T) {
	t.Parallel()
	var s snapshots
	opts := instant()
	opts.Instant = false
	d := newDriver(t, opts, 80, 24).steps(12)
	before := d.screen()
	s.add("dialling, part revealed", d)

	d.resize(60, 20).settle()
	s.add("too small", d)
	d.typ("JOSHUA")
	if len(d.pending) != 0 {
		t.Errorf("the clock must stop while too small; %d ticks armed", len(d.pending))
	}

	d.resize(80, 24)
	if got := d.screen(); got != before {
		t.Errorf("resume changed the screen:\n%s\nwant:\n%s", got, before)
	}
	d.settle()
	s.add("resumed and settled", d)
	if !strings.Contains(d.screen(), "LOGON:") || strings.Contains(d.screen(), "JOSHUA") {
		t.Errorf("keys typed while too small must be dropped:\n%s", d.screen())
	}
	golden.AssertString(t, "too_small_pause_resume", s.String())
}

// The first key during a reveal skips it; Enter on an empty line is only a skip.
func TestKeySkipsReveal(t *testing.T) {
	t.Parallel()
	opts := instant()
	opts.Instant = false
	d := newDriver(t, opts, 80, 24).steps(3)
	if !d.m.tw.Revealing() {
		t.Fatal("expected text mid-reveal")
	}
	d.send(enter)
	if d.m.tw.Revealing() || d.m.ed.Value() != "" {
		t.Errorf("Enter must skip the reveal and type nothing; editor %q", d.m.ed.Value())
	}
	d.settle().send(enter).settle()
	if strings.Count(d.screen(), "LOGON:") != 2 {
		t.Errorf("Enter at the prompt submits an empty logon:\n%s", d.screen())
	}
}

// Typeahead: while WOPR thinks, keys go to the editor and Enter is held until the prompt
// returns; the thinking indicator shows meanwhile.
func TestTypeaheadWhileThinking(t *testing.T) {
	t.Parallel()
	var s snapshots
	d := newDriver(t, Options{Instant: true, Play: "stub", Registry: gamestest.Registry()}, 80, 24).settle()
	d.slowThinks = true
	d.press("THINK").send(enter)
	d.steps(40) // over a second of the indicator
	d.press("WIN").send(enter)
	s.add("thinking, with typeahead", d)
	if !d.m.held || d.m.ed.Value() != "WIN" || !strings.Contains(d.screen(), "PROCESSING") {
		t.Fatalf("Enter while thinking must be held; held %v, editor %q:\n%s", d.m.held, d.m.ed.Value(), d.screen())
	}
	d.finishThinks()
	s.add("thought, then the held line", d)
	if d.m.held || !strings.Contains(d.screen(), "WINNER: PROFESSOR FALKEN") {
		t.Errorf("the held line was not submitted:\n%s", d.screen())
	}
	golden.AssertString(t, "typeahead_thinking", s.String())
}

// Esc in a game asks for confirmation, then ends the game. The stub game uses the panel
// layout, so its board sits above the console.
func TestEscConfirmInGame(t *testing.T) {
	t.Parallel()
	var s snapshots
	d := newDriver(t, Options{Instant: true, Play: "stub", Registry: gamestest.Registry()}, 80, 24).settle()
	s.add("game started", d)
	d.send(esc) // no settle: the confirmation lasts host.EscWindow
	s.add("first Esc", d)
	if !strings.Contains(d.screen(), "PRESS ESC AGAIN") {
		t.Error("no confirmation notice")
	}
	d.send(esc).settle()
	s.add("second Esc", d)
	if !strings.Contains(d.screen(), "GAME TERMINATED BEFORE COMPLETION.") {
		t.Error("the game was not aborted")
	}
	golden.AssertString(t, "esc_confirm", s.String())
}

// The confirmation expires: after the window, one Esc only asks again.
func TestEscWindowExpires(t *testing.T) {
	t.Parallel()
	d := newDriver(t, Options{Instant: true, Play: "stub", Registry: gamestest.Registry()}, 80, 24).settle()
	d.key(esc) // settle runs the clock past the window
	if strings.Contains(d.screen(), "PRESS ESC AGAIN") {
		t.Error("the notice outlived its window")
	}
	d.send(esc)
	if !strings.Contains(d.screen(), "PRESS ESC AGAIN") || d.m.runner.Depth() != 2 {
		t.Errorf("an Esc after the window must ask again, not abort:\n%s", d.screen())
	}
}

// In key mode, keys go to the game instead of the line editor.
func TestKeyMode(t *testing.T) {
	t.Parallel()
	d := newDriver(t, Options{Instant: true, Play: "stub", Registry: gamestest.Registry()}, 80, 24).settle()
	d.line("KEYS").key(up)
	if !strings.Contains(d.screen(), "KEY up") {
		t.Errorf("the arrow did not reach the game:\n%s", d.screen())
	}
}

func TestFrontPanelAndCentring(t *testing.T) {
	t.Parallel()
	var s snapshots
	opts := instant()
	opts.Panel = true
	d := newDriver(t, opts, 100, 30).settle().line("Joshua")
	s.add("panel, 100 columns", d)
	golden.AssertString(t, "front_panel", s.String())
}

func TestScrollback(t *testing.T) {
	t.Parallel()
	d := newDriver(t, instant(), 80, 24).settle().line("Joshua")
	for range 6 {
		d.line("HELP")
	}
	bottom := d.screen()
	d.key(pgUp)
	if d.screen() == bottom {
		t.Fatal("PgUp did not scroll")
	}
	d.key(pgDn)
	if d.screen() != bottom {
		t.Errorf("PgDn did not return to the bottom:\n%s", d.screen())
	}
}
