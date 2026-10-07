package movie

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GhostofGoes/WOPR/internal/games/catalog"
	"github.com/GhostofGoes/WOPR/internal/games/ending"
	"github.com/GhostofGoes/WOPR/internal/games/gtw"
	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/movie/scenes"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/proto/host"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// play runs a director under the host, as movie mode does: the pinned seed, programs from the
// catalog, and none of them abortable. testkit has no clock, so the scenes' pauses are skipped.
func play(t *testing.T, opts Options) *testkit.Session {
	t.Helper()
	opts.noPauses = true
	reg := catalog.Registry()
	cfg := host.Config{
		Seed: Seed,
		Resolve: func(l proto.Launch) (proto.Program, host.Placement, error) {
			e, ok := reg.Get(l.Slug)
			if !ok || e.New == nil {
				return nil, host.Placement{}, errors.New("no such program")
			}
			return e.New(), host.Placement{Layout: e.Info.Layout, PanelRows: e.Info.PanelRows, NoAbort: true}, nil
		},
	}
	return testkit.Start(t, New(opts), host.Placement{}, cfg)
}

// Two small scenes: every kind of console step.
var twoScenes = []scenes.Scene{
	{Slug: "first", Title: "FIRST", Blurb: "THE FIRST.", Steps: []scenes.Step{
		scenes.Say{Lines: script.Recon("LOGON: "), Pace: proto.PaceInstant},
		scenes.Type{Prompt: script.Recon("NAME: "), Text: script.User(script.Reconstructed, "Joshua")[0]},
		scenes.Wait{D: time.Second, Prov: script.Original},
		scenes.Say{Lines: script.Recon("GREETINGS."), Pace: proto.PaceSpeech},
	}},
	{Slug: "second", Title: "SECOND", Blurb: "THE SECOND.", Steps: []scenes.Step{
		scenes.Type{Text: script.User(script.Reconstructed, "Hello.")[0]},
		scenes.Clear{Prov: script.Reconstructed},
		scenes.Say{Lines: script.Recon("BYE."), Pace: proto.PaceSpeech},
	}},
}

// wopr -m <scene> plays from that scene to the end of the list, then exits 0 (RF-5).
func TestPlaysToTheEndOfTheList(t *testing.T) {
	t.Parallel()
	s := play(t, Options{Scene: "first", Scenes: twoScenes})
	want := "[CLEAR]\nLOGON: \nNAME: Joshua\n\nGREETINGS.\n[CLEAR]\nHello.\n\n[CLEAR]\nBYE.\n"
	if !s.Exited() || s.Transcript() != want {
		t.Errorf("exited %v, transcript:\n%s\nwant:\n%s", s.Exited(), s.Transcript(), want)
	}
	if s := play(t, Options{Scene: "2", Scenes: twoScenes}); !strings.HasPrefix(s.Transcript(), "[CLEAR]\nHello.") || !s.Exited() {
		t.Errorf("from scene 2:\n%s", s.Transcript())
	}
	if s := play(t, Options{Scene: "first", Single: true, Scenes: twoScenes}); strings.Contains(s.Transcript(), "BYE.") || !s.Exited() {
		t.Errorf("Single plays one scene:\n%s", s.Transcript())
	}
}

// wopr -m opens the menu; a scene chosen there plays to the end of the list, then the menu
// returns; q leaves.
func TestMenu(t *testing.T) {
	t.Parallel()
	s := play(t, Options{Scenes: twoScenes})
	if asking, prompt := s.Asking(); !asking || prompt != "SCENE: " || !s.Contains("  2.  SECOND           THE SECOND.") {
		t.Fatalf("the menu asks for a scene:\n%s", s.Transcript())
	}
	s.Type("third")
	if !s.Contains("NO SUCH SCENE.") {
		t.Errorf("an unknown scene is refused:\n%s", s.Transcript())
	}
	s.Type("2")
	if !s.Contains("BYE.") || s.Exited() || strings.Count(s.Transcript(), "MOVIE MODE") != 2 {
		t.Fatalf("scene 2 plays, then the menu returns:\n%s", s.Transcript())
	}
	s.Type("Q")
	if !s.Exited() {
		t.Error("Q leaves")
	}
	for _, in := range []string{"", "  "} {
		s := play(t, Options{Scenes: twoScenes})
		s.Type(in)
		if asking, _ := s.Asking(); !asking || s.Exited() {
			t.Errorf("%q asks again", in)
		}
	}
	amb := append(append([]scenes.Scene{}, twoScenes...), scenes.Scene{Slug: "seventh", Title: "SEVENTH"})
	s = play(t, Options{Scenes: amb})
	s.Type("se")
	if !s.Contains("THAT NAME FITS MORE THAN ONE SCENE:\n  2.  SECOND\n  3.  SEVENTH") {
		t.Errorf("an ambiguous name lists its scenes:\n%s", s.Transcript())
	}
}

func TestResolve(t *testing.T) {
	t.Parallel()
	list := infos(append(append([]scenes.Scene{}, twoScenes...), scenes.Scene{Slug: "norad-terminal", Title: "NORAD TERMINAL"}))
	for in, want := range map[string]int{"1": 0, "2": 1, "first": 0, "SECOND": 1, "sec": 1, "norad terminal": 2, "Norad_Terminal": 2, "nor": 2} {
		if got, err := resolve(list, in); err != nil || got != want {
			t.Errorf("%q: %d %v, want %d", in, got, err, want)
		}
	}
	for _, in := range []string{"", "0", "4", "x", "-"} {
		if _, err := resolve(list, in); !errors.Is(err, ErrNoScene) {
			t.Errorf("%q: %v, want ErrNoScene", in, err)
		}
	}
	list = append(list, Info{Number: 4, Slug: "fourth"})
	var amb *AmbiguousError
	if _, err := resolve(list, "f"); !errors.As(err, &amb) || len(amb.Candidates) != 2 || err.Error() != "could mean first, fourth" {
		t.Errorf("f: %v", err)
	}
}

// The controls, on the program directly (testkit plays everything at once): Space pauses and
// resumes, n and p change scene, and Esc opens the menu. Each cut reveals what is queued.
func TestControls(t *testing.T) {
	t.Parallel()
	d := New(Options{Scene: "first", Scenes: twoScenes})
	d.Start(proto.Env{})
	key := func(k proto.Key, r rune) []proto.Output { return d.Handle(proto.KeyEvent{Key: k, Rune: r}) }

	if outs := key(proto.KeyRune, ' '); !hasOut[proto.Hold](outs) || !d.paused {
		t.Fatalf("Space pauses: %#v", outs)
	}
	if outs := key(proto.KeyRune, ' '); !hasOut[proto.Hold](outs) || d.paused {
		t.Fatalf("Space resumes: %#v", outs)
	}
	key(proto.KeyRune, ' ')
	outs := key(proto.KeyRune, 'n')
	if d.scene != 1 || d.paused || !hasOut[proto.Skip](outs) || !hasOut[proto.Hold](outs) {
		t.Fatalf("n goes on to the next scene, unpaused, dropping the rest: scene %d %#v", d.scene, outs)
	}
	key(proto.KeyLeft, 0)
	if d.scene != 0 {
		t.Fatal("Left goes back")
	}
	key(proto.KeyRune, 'p')
	if d.scene != 0 || d.phase != playing {
		t.Fatal("p on the first scene starts it again")
	}
	key(proto.KeyRight, 0)
	if outs := key(proto.KeyRune, 'N'); !hasOut[proto.Quit](outs) {
		t.Errorf("next after the last scene ends the list: %#v", outs)
	}

	d = New(Options{Scene: "first", Scenes: twoScenes})
	d.Start(proto.Env{})
	if outs := key(proto.KeyEsc, 0); d.phase != inMenu || !hasOut[proto.Prompt](outs) {
		t.Fatalf("Esc opens the menu: %#v", outs)
	}
	if outs := key(proto.KeyRune, 'n'); outs != nil {
		t.Error("the menu reads lines, not the controls")
	}
	// A Drained left over from the cut scene does nothing in the menu.
	if outs := d.Handle(proto.Drained{}); outs != nil {
		t.Errorf("a stale Drained: %#v", outs)
	}
}

func hasOut[T proto.Output](outs []proto.Output) bool {
	for _, o := range outs {
		if _, ok := o.(T); ok {
			return true
		}
	}
	return false
}

// A Board step plays the film's war frame by frame on ticks, or at once under --instant; a
// Run step launches a program over the director, and the scene goes on after it.
func TestBoardAndRun(t *testing.T) {
	t.Parallel()
	scene := []scenes.Scene{{Slug: "war", Steps: []scenes.Step{
		scenes.Board{At: gtw.FilmStrike1, Play: true, Prov: script.Reconstructed},
		scenes.Board{At: gtw.FilmClimax, Prov: script.Reconstructed},
		scenes.Run{Slug: gtw.TicTacToeSlug, Mode: ending.MovieMode, Prov: script.Reconstructed},
		scenes.Say{Lines: script.Orig("AFTER."), Pace: proto.PaceSpeech},
	}}}
	s := play(t, Options{Scene: "war", Scenes: scene})
	for _, want := range []string{"FIRST STRIKE LAUNCHED.", "DEFCON 4.", "PLEASE LIST NUMBER OF PLAYERS: 1", "STALEMATE.", "NOT TO PLAY.", "AFTER."} {
		if !s.Contains(want) {
			t.Errorf("missing %q:\n%s", want, s.Transcript())
		}
	}
	if s.Contains("** GAME ROUTINE RUNNING **") || !s.Exited() {
		t.Errorf("a Board step without Play jumps silently, and the list ends:\n%s", s.Transcript())
	}

	// Paced: the board animates on ticks and the step ends when the strike has landed.
	d := New(Options{Scene: "war", Scenes: scene})
	outs := d.Start(proto.Env{Width: 80, Height: 19})
	if !hasOut[proto.SetLayout](outs) || !hasOut[proto.Animate](outs) || hasOut[proto.Drain](outs) || d.phase != animating {
		t.Fatalf("the board opens and animates: %#v", outs)
	}
	var lines []string
	for range 60 {
		outs = d.Handle(proto.TickEvent{Dt: gtw.FilmFrame})
		for _, o := range outs {
			if say, ok := o.(proto.Say); ok {
				lines = append(lines, say.Lines...)
			}
		}
		if hasOut[proto.Drain](outs) {
			break
		}
	}
	if d.phase != playing || len(lines) != 3 || !strings.Contains(lines[1], "DEFCON 4.") {
		t.Fatalf("the first strike plays out: phase %d, lines %q", d.phase, lines)
	}
	d.Handle(proto.Drained{}) // the climax: the board keeps the clock running for the code search
	outs = d.Handle(proto.Drained{})
	if !hasOut[proto.Launch](outs) || d.phase != running {
		t.Fatalf("Run launches: %#v", outs)
	}
	for _, o := range outs {
		if l, ok := o.(proto.Launch); ok && l.Mode != "movie:0" {
			t.Errorf("the ending carries on from the board's code: %+v", l)
		}
	}
}

// A Run step whose program cannot be built does not stop the show: the host says so, and the
// scene goes on. The controls work while a Run step is out, so nothing can leave the director
// deaf to them (the host gives keys to a program that does run, not to the director).
func TestRunThatCannotStart(t *testing.T) {
	t.Parallel()
	scene := []scenes.Scene{{Slug: "broken", Steps: []scenes.Step{
		scenes.Say{Lines: script.Orig("BEFORE."), Pace: proto.PaceSpeech},
		scenes.Run{Slug: "no-such-program", Prov: script.Original},
		scenes.Say{Lines: script.Orig("AFTER."), Pace: proto.PaceSpeech},
	}}}
	s := play(t, Options{Scene: "broken", Scenes: scene})
	if want := "[CLEAR]\nBEFORE.\n** GAME ROUTINE NOT AVAILABLE **\nAFTER.\n"; s.Transcript() != want || !s.Exited() {
		t.Errorf("exited %v, transcript:\n%s\nwant:\n%s", s.Exited(), s.Transcript(), want)
	}

	d := New(Options{Scene: "broken", Scenes: scene})
	d.Start(proto.Env{})
	if outs := d.Handle(proto.Drained{}); !hasOut[proto.Launch](outs) || !hasOut[proto.Drain](outs) || d.phase != running {
		t.Fatalf("Run launches, then asks to hear back: %#v", outs)
	}
	if outs := d.Handle(proto.KeyEvent{Key: proto.KeyEsc}); d.phase != inMenu || !hasOut[proto.Prompt](outs) {
		t.Errorf("Esc reaches the menu from a Run step: %#v", outs)
	}
}

// Under --instant text appears at once, but the director still waits out a scene's pauses on
// its own ticks (the host drops Wait), so every page stays up to be read; a pause freezes them.
func TestInstantKeepsThePauses(t *testing.T) {
	t.Parallel()
	d := New(Options{Scene: "first", Scenes: twoScenes})
	d.Start(proto.Env{Instant: true})
	d.Handle(proto.Drained{}) // LOGON: is out; NAME: Joshua is typed
	outs := d.Handle(proto.Drained{})
	if d.phase != waiting || hasOut[proto.Drain](outs) || !hasOut[proto.Animate](outs) {
		t.Fatalf("the pause is timed on ticks: phase %d %#v", d.phase, outs)
	}
	tick := func() []proto.Output { return d.Handle(proto.TickEvent{Dt: tickEvery}) }
	for range 9 {
		if outs := tick(); hasOut[proto.Say](outs) {
			t.Fatalf("too early: %#v", outs)
		}
	}
	d.Handle(proto.KeyEvent{Key: proto.KeyRune, Rune: ' '}) // pause: the host stops the ticks
	d.Handle(proto.KeyEvent{Key: proto.KeyRune, Rune: ' '})
	outs = tick()
	if d.phase != playing || !hasOut[proto.Say](outs) || !hasOut[proto.Drain](outs) {
		t.Fatalf("after a second, GREETINGS.: %#v", outs)
	}
	if a, ok := outs[0].(proto.Animate); !ok || a.Every != 0 {
		t.Errorf("the clock stops with the pause: %#v", outs)
	}

	// Next scene during a pause: the wait is dropped with the rest of the scene.
	d = New(Options{Scene: "first", Scenes: twoScenes})
	d.Start(proto.Env{Instant: true})
	d.Handle(proto.Drained{})
	d.Handle(proto.Drained{})
	if d.Handle(proto.KeyEvent{Key: proto.KeyRight}); d.scene != 1 || d.phase != playing {
		t.Fatalf("Right leaves the pause for the next scene: scene %d phase %d", d.scene, d.phase)
	}
	if outs := tick(); outs != nil {
		t.Errorf("a stray tick does nothing: %#v", outs)
	}
}

// The director's text is tagged and fits 80 columns.
func TestDirectorText(t *testing.T) {
	t.Parallel()
	notice, err := os.ReadFile(filepath.Join("..", "..", "NOTICE.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, err := range script.Validate(Lines, string(notice)) {
		t.Error(err)
	}
	for _, ls := range Lines {
		for _, l := range ls {
			if len(l.Text) > 80 {
				t.Errorf("wider than 80: %q", l.Text)
			}
		}
	}
}
