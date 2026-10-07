package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/GhostofGoes/WOPR/internal/games/catalog"
	"github.com/GhostofGoes/WOPR/internal/golden"
)

func movieMode(scene string) Options {
	return Options{Movie: true, Scene: scene, Registry: catalog.Registry(), Theme: "norad"}
}

// until runs the clock until the screen satisfies cond.
func (d *driver) until(what string, cond func(screen string) bool) *driver {
	d.t.Helper()
	for i := 0; !cond(d.screen()); i++ {
		if i > maxSteps || !d.step() {
			d.t.Fatalf("never saw %s:\n%s", what, d.screen())
		}
	}
	return d
}

func shows(text string) func(string) bool {
	return func(s string) bool { return strings.Contains(s, text) }
}

// wopr -m: the scene menu; Q leaves with exit 0.
func TestMovieMenu(t *testing.T) {
	t.Parallel()
	var s snapshots
	d := newDriver(t, movieMode(""), 80, 24).settle()
	s.add("wopr -m: the scene menu", d)
	d.line("x")
	if !strings.Contains(d.screen(), "NO SUCH SCENE.") {
		t.Errorf("an unknown scene is refused:\n%s", d.screen())
	}
	d.line("q")
	if d.ended != "quit" {
		t.Errorf("q ends with %q, want quit (exit 0)", d.ended)
	}
	golden.AssertString(t, "movie_menu", s.String())
}

// wopr -m <scene> -i: text appears at once, but the scenes' pauses still hold each page up to
// be read; the list plays to the end and exits 0 by itself.
func TestMovieInstantPlaysThrough(t *testing.T) {
	t.Parallel()
	opts := movieMode("first-contact")
	opts.Instant = true
	d := newDriver(t, opts, 80, 24)
	if s := d.screen(); !strings.Contains(s, "CONNECTING...") || strings.Contains(s, "CONNECTED.") || d.ended != "" {
		t.Fatalf("the dial's pause holds its first line:\n%s", s)
	}
	begun := d.now
	d.until("the refused logon", shows("--CONNECTION TERMINATED--"))
	if d.ended != "" || d.now.Sub(begun) < time.Second {
		t.Fatalf("the first page took %v (ended %q):\n%s", d.now.Sub(begun), d.ended, d.screen())
	}
	d.settle()
	if d.ended != "quit" || !strings.Contains(strings.Join(transcript(d), "\n"), "NOT TO PLAY.") {
		t.Fatalf("ended %q:\n%s", d.ended, d.screen())
	}
	if played := d.now.Sub(begun); played < 40*time.Second {
		t.Errorf("the film took %v: its pauses were skipped", played)
	}
}

// Paced playback, as a viewer sees it: a target being typed, a pause, the first strike on the
// board, Esc to the menu, the climax at DEFCON 1 and its tic-tac-toe, which no key interrupts.
func TestMovieScreens(t *testing.T) {
	t.Parallel()
	var s snapshots
	d := newDriver(t, movieMode("first-strike"), 80, 24)
	d.until("a target half typed", func(sc string) bool { return strings.Contains(sc, "Las V") && !strings.Contains(sc, "Las Vegas") })
	s.add("first-strike: typing a target", d)

	d.send(space)
	paused := d.screen()
	s.add("Space: paused", d)
	d.steps(30)
	if d.screen() != paused || !strings.Contains(paused, "** PAUSED **") {
		t.Fatalf("nothing moves while paused:\n%s", d.screen())
	}
	d.send(space)

	d.until("the second strike's prompt", shows("STRIKE 2 OF 3 [50 50 100]:"))
	s.add("first-strike: the first strike has landed", d)

	d.key(esc)
	if !strings.Contains(d.screen(), "SCENE:") {
		t.Fatalf("Esc opens the menu:\n%s", d.screen())
	}
	d.press("climax").send(enter) // no settling: that would play the whole scene
	d.until("a refused game", shows("** ACCESS DENIED **"))
	s.add("climax: DEFCON 1, the board again", d)

	d.until("tic-tac-toe", shows("ONE OR TWO PLAYERS?"))
	d.send(esc).send(esc).send(keyRune('n'))
	if d.m.runner.Depth() != 2 {
		t.Fatal("no key ends the climax's tic-tac-toe")
	}
	d.until("the first move", shows("YOUR MOVE: "))
	s.add("climax: tic-tac-toe plays itself", d)

	d.settle()
	if d.ended != "" || !strings.Contains(d.screen(), "SCENE:") {
		t.Fatalf("after a scene picked in the menu, the end of the list returns to it (ended %q):\n%s", d.ended, d.screen())
	}
	if !strings.Contains(strings.Join(transcript(d), "\n"), "HOW ABOUT A NICE GAME OF CHESS?") {
		t.Error("the climax played to the end")
	}
	golden.AssertString(t, "movie_screens", s.String())
}

// n skips to the next scene at once; the last scene's n ends the list (exit 0).
func TestMovieNextAndPrevious(t *testing.T) {
	t.Parallel()
	d := newDriver(t, movieMode("call-back"), 80, 24)
	d.until("the call-back", shows("GREETINGS PROFESSOR FALKEN."))
	d.send(keyRune('n')).steps(1) // the new page is the next item out
	if !strings.Contains(d.screen(), "LOGON:") || strings.Contains(d.screen(), "GREETINGS") {
		t.Fatalf("n starts the next scene on a clean page:\n%s", d.screen())
	}
	d.send(keyRune('p')).steps(5)
	if strings.Contains(d.screen(), "LOGON:") {
		t.Fatalf("p goes back to the call-back:\n%s", d.screen())
	}
	d.send(keyRune('n')).send(keyRune('n')).send(keyRune('n'))
	if d.ended != "quit" {
		t.Errorf("past the last scene, the list ends: %q", d.ended)
	}
}
