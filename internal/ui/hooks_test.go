package ui

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/GhostofGoes/WOPR/internal/games/gamestest"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// hookRoot is a root program that uses the movie-mode hooks (docs/PLAN.md §7): it captures
// keys; Space holds and releases output, d asks for a Drained event, s skips, g launches the
// stub game, m asks for a line (which ends the capture), and Esc arrives as a key.
type hookRoot struct{ held bool }

const hookLine = "A LINE THAT TAKES A WHILE TO TYPE OUT."

func (h *hookRoot) Start(proto.Env) []proto.Output {
	return []proto.Output{proto.Say{Lines: []string{hookLine}}, proto.AwaitKeys{Capture: true}}
}

func (h *hookRoot) Handle(ev proto.Event) []proto.Output {
	say := func(s string) []proto.Output { return []proto.Output{proto.Say{Lines: []string{s}}} }
	switch ev := ev.(type) {
	case proto.KeyEvent:
		switch {
		case ev.Key == proto.KeyEsc:
			return say("ESC CAPTURED.")
		case ev.Rune == ' ':
			h.held = !h.held
			return []proto.Output{proto.Hold{On: h.held}}
		case ev.Rune == 'd':
			return append(say("DRAINING."), proto.Drain{})
		case ev.Rune == 's':
			return append(say("SKIPPED STRAIGHT TO THE END OF THIS LINE."), proto.Skip{})
		case ev.Rune == 'g':
			return []proto.Output{proto.Launch{Slug: "stub"}}
		case ev.Rune == 'm':
			return []proto.Output{proto.Prompt{Text: "MENU: "}}
		}
	case proto.Drained:
		return say("DRAINED.")
	case proto.GameOver:
		return say("BACK.")
	}
	return nil
}

func (h *hookRoot) View(*proto.Canvas) {}

func hookDriver(t *testing.T) *driver {
	t.Helper()
	return newDriverRoot(t, Options{Registry: gamestest.Registry()}, 80, 24, &hookRoot{})
}

func visible(d *driver) string { return strings.Join(transcript(d), "\n") }

var space = tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}

// A capturing root gets Space as a key, not a skip; Hold freezes the typewriter and the clock
// until released.
func TestCaptureAndHold(t *testing.T) {
	t.Parallel()
	d := hookDriver(t).steps(5)
	before := visible(d)
	if !d.m.tw.Revealing() || before == hookLine {
		t.Fatalf("expected the line mid-reveal: %q", before)
	}
	d.send(space)
	if !d.m.frozen || visible(d) != before {
		t.Fatalf("Space must hold, not skip: frozen %v, %q", d.m.frozen, visible(d))
	}
	d.steps(20)
	if visible(d) != before || len(d.pending) != 0 {
		t.Fatalf("nothing moves while held: %q, %d ticks armed", visible(d), len(d.pending))
	}
	if !strings.Contains(d.screen(), "cursor: col") {
		t.Errorf("movie mode keeps the cursor after the text:\n%s", d.screen())
	}
	d.send(space).settle()
	if d.m.frozen || !strings.Contains(visible(d), hookLine) {
		t.Fatalf("released, the line finishes: %q", visible(d))
	}
	d.key(esc)
	if !strings.Contains(visible(d), "ESC CAPTURED.") {
		t.Errorf("Esc reaches a capturing root as a key: %q", visible(d))
	}
}

// Drained arrives once the output before the Drain is out; Skip reveals at once.
func TestDrainAndSkip(t *testing.T) {
	t.Parallel()
	d := hookDriver(t).settle()
	d.send(keyRune('d')).steps(2)
	if strings.Contains(visible(d), "DRAINED.") {
		t.Fatal("Drained must wait for DRAINING. to be revealed")
	}
	d.settle()
	if !strings.HasSuffix(visible(d), "DRAINING.\nDRAINED.") {
		t.Fatalf("Drained after the reveal: %q", visible(d))
	}
	d.send(keyRune('s'))
	if !strings.HasSuffix(visible(d), "SKIPPED STRAIGHT TO THE END OF THIS LINE.") || d.m.tw.Busy() {
		t.Fatalf("Skip reveals at once: %q", visible(d))
	}
}

// While a program the capturing root launched runs, keys other than Ctrl+C do nothing: no
// typing, no Esc machine.
func TestShieldedLaunch(t *testing.T) {
	t.Parallel()
	d := hookDriver(t).settle()
	d.key(keyRune('g'))
	if d.m.runner.Depth() != 2 {
		t.Fatal("g launches the stub")
	}
	d.typ("WIN").key(enter).key(esc).key(esc)
	if d.m.runner.Depth() != 2 || d.m.ed.Value() != "" || strings.Contains(d.screen(), "PRESS ESC AGAIN") {
		t.Fatalf("keys must do nothing while shielded:\n%s", d.screen())
	}
	if d.send(esc); !strings.Contains(d.screen(), "** THE GAME PLAYS TO THE END. CTRL+C QUITS. **") {
		t.Fatalf("a dropped key says why:\n%s", d.screen())
	}
	if d.settle(); strings.Contains(d.screen(), "PLAYS TO THE END") {
		t.Fatalf("the notice clears after the Esc window:\n%s", d.screen())
	}
	if d.send(ctrlC); d.ended != "interrupt" {
		t.Error("Ctrl+C still quits")
	}
}

// The stub's movie-mode cases as an ordinary game: a typed line on one row, Drained after
// its line, and Hold.
func TestStubHooks(t *testing.T) {
	t.Parallel()
	d := newDriver(t, Options{Play: "stub", Registry: gamestest.Registry()}, 80, 24).settle()
	d.line("TYPE")
	if lines := transcript(d); len(lines) < 2 || lines[len(lines)-2] != "TYPED: Hello." || lines[len(lines)-1] != "" {
		t.Errorf("a typed line is one row and a blank: %q", lines)
	}
	d.line("DRAIN")
	if !strings.HasSuffix(strings.TrimRight(visible(d), "\n"), "DRAINING.\nDRAINED.") {
		t.Errorf("the stub hears Drained: %q", visible(d))
	}
	d.line("SKIP")
	if !strings.Contains(visible(d), "SKIPPED AT ONCE.") {
		t.Errorf("skip: %q", visible(d))
	}
}

// Held output stays held: no key reveals it, not even once the holder has asked for a line
// and keys reach the line editor, and neither does --instant.
func TestHoldIsNotSkipped(t *testing.T) {
	t.Parallel()
	d := hookDriver(t).steps(5)
	before := visible(d)
	d.send(space).send(keyRune('m')).send(keyRune('x')).send(enter).steps(10)
	if !d.m.frozen || visible(d) != before || d.m.runner.Captures() {
		t.Fatalf("a key while held must not reveal (frozen %v): %q, was %q", d.m.frozen, visible(d), before)
	}

	opts := Options{Registry: gamestest.Registry(), Instant: true}
	d = newDriverRoot(t, opts, 80, 24, &hookRoot{}).settle()
	d.send(space).send(keyRune('d'))
	if !d.m.frozen || strings.Contains(visible(d), "DRAINING.") {
		t.Fatalf("--instant must not reveal held output: %q", visible(d))
	}
	d.send(space)
	if d.m.frozen || !strings.HasSuffix(visible(d), "DRAINING.\nDRAINED.") {
		t.Fatalf("released, the held output comes out and its marker is answered: %q", visible(d))
	}
}

// thinker prints A, asks for Drained and starts a Think at once.
type thinker struct{}

func (thinker) Start(proto.Env) []proto.Output {
	return []proto.Output{
		proto.Say{Lines: []string{"A"}},
		proto.Drain{},
		proto.Think{Fn: func(context.Context) (any, error) { return nil, nil }},
	}
}

func (thinker) Handle(ev proto.Event) []proto.Output {
	switch ev.(type) {
	case proto.Drained:
		return []proto.Output{proto.Say{Lines: []string{"DRAINED"}}}
	case proto.ThinkDone:
		return []proto.Output{proto.Say{Lines: []string{"THOUGHT"}}}
	}
	return nil
}

func (thinker) View(*proto.Canvas) {}

// Drained comes after ThinkDone in the UI, paced or instant, as it does under testkit.
func TestDrainedAfterThinkDone(t *testing.T) {
	t.Parallel()
	for _, instant := range []bool{false, true} {
		d := newDriverRoot(t, Options{Instant: instant}, 80, 24, thinker{}).settle()
		if got := visible(d); got != "A\nTHOUGHT\nDRAINED" {
			t.Errorf("instant %v: %q", instant, got)
		}
	}
}
