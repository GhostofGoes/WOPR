// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package gtw_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/GhostofGoes/WOPR/internal/games/gtw"
	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/proto/host"
)

// windowed starts GTW under the host with a launch window of window (0: none), and plays
// side 2 against Las Vegas to the strike 2 prompt. It returns the game too, to draw its board.
func windowed(t *testing.T, window time.Duration) (*testkit.Session, proto.Program) {
	t.Helper()
	g := gtw.New()
	s := testkit.Start(t, g, host.Placement{}, host.Config{Seed: 1, LaunchWindow: window})
	s.Type("2").Type("Las Vegas").Type("")
	if _, p := s.Asking(); p != "STRIKE 2 OF 3 [50 50 100]: " {
		t.Fatalf("asks %q, want strike 2:\n%s", p, s.Transcript())
	}
	return s, g
}

// foot is the board's last row, where the orders hint and the launch window show.
func foot(g proto.Program) string {
	c := proto.NewCanvas(80, 19)
	g.View(c)
	rows := strings.Split(c.String(), "\n")
	return rows[18]
}

// The window counts down on the board in whole seconds. At zero WOPR takes launch authority,
// as AUTO gives it: the rest of the war is the same as AUTO's, with no prompt and no key
// taken until the assessment.
func TestLaunchWindowShuts(t *testing.T) {
	t.Parallel()
	s, g := windowed(t, 5*time.Second)
	for _, want := range []string{"WINDOW 5 SECONDS", "WINDOW 4 SECONDS", "WINDOW 3 SECONDS", "WINDOW 2 SECONDS", "WINDOW 1 SECOND"} {
		if got := foot(g); !strings.HasPrefix(got, "ORDERS: ") || !strings.HasSuffix(got, " "+want) || len(got) != 80 {
			t.Fatalf("the foot reads %q, want the orders hint and %q at the right edge", got, want)
		}
		if asking, _ := s.Asking(); !asking {
			t.Fatalf("at %s the prompt is gone", want)
		}
		s.Advance(time.Second)
	}
	if !s.Contains("LAUNCH WINDOW CLOSED, PROFESSOR.\nWOPR HAS LAUNCH AUTHORITY.\nSTRIKE 2 LAUNCHED.") {
		t.Errorf("WOPR does not take launch authority:\n%s", s.Transcript())
	}
	if s.Contains("STRIKE 3 OF 3") || !s.Contains("STRIKE ASSESSMENT COMPLETE.") {
		t.Errorf("WOPR orders every remaining strike:\n%s", s.Transcript())
	}
	if asking, p := s.Asking(); !asking || p != "" || s.Keys() {
		t.Errorf("the assessment asks for Enter: asking %v %q, keys %v", asking, p, s.Keys())
	}

	auto, _ := windowed(t, 0)
	auto.Type("auto")
	_, after, _ := strings.Cut(s.Transcript(), "WOPR HAS LAUNCH AUTHORITY.")
	_, autoAfter, _ := strings.Cut(auto.Transcript(), "WOPR HAS LAUNCH AUTHORITY.")
	if after != autoAfter {
		t.Errorf("the war after the window shut:\n%s\nis not AUTO's:\n%s", after, autoAfter)
	}
}

// Paced, the strike prompt runs a one-second clock for the window, and when it shuts key mode
// takes the prompt's place, its hint saying no order is taken, while the strike flies.
func TestLaunchWindowPaced(t *testing.T) {
	t.Parallel()
	g := gtw.New()
	g.Start(proto.Env{Seed: 1, Width: 80, Height: 19, LaunchWindow: 2 * time.Second})
	g.Handle(proto.LineEvent{Text: "2"})
	g.Handle(proto.LineEvent{Text: "Las Vegas"})
	g.Handle(proto.LineEvent{Text: ""})
	var outs []proto.Output
	for len(outs) == 0 || !slices.ContainsFunc(outs, isPrompt) {
		outs = g.Handle(proto.TickEvent{Dt: time.Minute})
	}
	if !slices.Contains(outs, proto.Output(proto.Animate{Every: time.Second})) {
		t.Errorf("the strike prompt starts no clock for the window: %#v", outs)
	}
	if outs = g.Handle(proto.TickEvent{Dt: time.Second}); !slices.Equal(outs, []proto.Output{proto.Redraw{}}) {
		t.Errorf("a second into the window: %#v", outs)
	}
	outs = g.Handle(proto.TickEvent{Dt: time.Second})
	if len(outs) == 0 || outs[0] != proto.Output(proto.AwaitKeys{Hint: "NO ORDERS ACCEPTED, PROFESSOR."}) ||
		slices.ContainsFunc(outs, isPrompt) || !slices.Contains(outs, proto.Output(proto.Animate{Every: frameEvery})) {
		t.Errorf("the window shut: %#v", outs)
	}
	if outs := g.Handle(proto.KeyEvent{Key: proto.KeyEnter}); len(outs) != 0 {
		t.Errorf("a key gave an order: %#v", outs)
	}
}

func isPrompt(o proto.Output) bool { _, ok := o.(proto.Prompt); return ok }

// An order in time stops the window, and the next strike's prompt starts a new one. HELP and
// a refused order do not stop it.
func TestLaunchWindowOrderInTime(t *testing.T) {
	t.Parallel()
	s, g := windowed(t, 5*time.Second)
	s.Advance(4 * time.Second).Type("hold")
	if _, p := s.Asking(); p != "STRIKE 3 OF 3 [100 100 100]: " || !s.Contains("NO LAUNCH ORDERED.") {
		t.Fatalf("the order was not taken:\n%s", s.Transcript())
	}
	if got := foot(g); !strings.HasSuffix(got, " WINDOW 5 SECONDS") || !strings.HasPrefix(got, "LAST ORDERS: ") {
		t.Errorf("strike 3's window does not start again: %q", got)
	}
	s.Advance(time.Second).Type("help").Advance(time.Second).Type("50 50")
	if got := foot(g); !strings.HasSuffix(got, " WINDOW 3 SECONDS") {
		t.Errorf("HELP and a refusal stopped the window: %q", got)
	}
	s.Type("")
	s.Advance(time.Hour)
	if s.Contains("LAUNCH WINDOW CLOSED") || s.Contains("WOPR HAS LAUNCH AUTHORITY.") {
		t.Errorf("the window shut after the order:\n%s", s.Transcript())
	}
	if !s.Contains("STRIKE ASSESSMENT COMPLETE.") {
		t.Errorf("the war did not end:\n%s", s.Transcript())
	}
}

// Without --launch-window a strike prompt waits for ever, and the board shows no window.
func TestNoLaunchWindowByDefault(t *testing.T) {
	t.Parallel()
	s, g := windowed(t, 0)
	s.Advance(time.Hour)
	if _, p := s.Asking(); p != "STRIKE 2 OF 3 [50 50 100]: " || s.Contains("WOPR HAS LAUNCH AUTHORITY.") {
		t.Errorf("the prompt did not wait:\n%s", s.Transcript())
	}
	if got := foot(g); strings.Contains(got, "WINDOW") {
		t.Errorf("the foot reads %q", got)
	}
}

// The longest window fits beside both hints, apart from them, and is an alert only in its last
// ten seconds.
func TestLaunchWindowFits(t *testing.T) {
	t.Parallel()
	s, g := windowed(t, 999*time.Second)
	style := func() proto.Style {
		c := proto.NewCanvas(80, 19)
		g.View(c)
		return c.At(79, 18).S
	}
	for _, hint := range []string{"ORDERS: ", "LAST ORDERS: "} {
		got := foot(g)
		left, _, _ := strings.Cut(got, "  ")
		if !strings.HasPrefix(got, hint) || !strings.HasSuffix(got, "  WINDOW 999 SECONDS") || len(got) != 80 ||
			!strings.HasSuffix(left, ".") {
			t.Errorf("the foot reads %q", got)
		}
		if style() != proto.StyleBright {
			t.Errorf("%q: the window is %v with 999 seconds left, want bright", got, style())
		}
		s.Type("")
	}
	s, g = windowed(t, 11*time.Second)
	if style() != proto.StyleBright {
		t.Errorf("the window is an alert with 11 seconds left")
	}
	s.Advance(time.Second)
	if style() != proto.StyleAlert {
		t.Errorf("the window is %v with 10 seconds left, want an alert", style())
	}
}
