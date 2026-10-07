package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/GhostofGoes/WOPR/internal/games/gamestest"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// hookRoot is a root program that uses the movie-mode hooks (docs/PLAN.md §7): it captures
// keys; Space holds and releases output, d asks for a Drained event, s skips, g launches the
// stub game, and Esc arrives as a key.
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
	d.press("HOLD").send(enter).steps(10)
	if !d.m.frozen || strings.Contains(visible(d), "HELD.") {
		t.Errorf("held before HELD. is revealed: frozen %v", d.m.frozen)
	}
}
