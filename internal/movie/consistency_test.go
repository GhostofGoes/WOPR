package movie

import (
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games/catalog"
	"github.com/GhostofGoes/WOPR/internal/games/gtw"
	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/movie/scenes"
	"github.com/GhostofGoes/WOPR/internal/proto/host"
	"github.com/GhostofGoes/WOPR/internal/wopr"
)

// The consistency test (docs/PLAN.md §7): movie mode and interactive play cannot drift apart.

// sceneText is what a console scene shows, as a transcript: WOPR's lines, each typed line
// with its prompt and the blank after it, and [CLEAR] for a new page.
func sceneText(t *testing.T, s scenes.Scene) string {
	t.Helper()
	var lines []string
	for _, st := range s.Steps {
		switch st := st.(type) {
		case scenes.Say:
			lines = append(lines, st.Lines.Texts()...)
		case scenes.Type:
			lines = append(lines, strings.Join(st.Prompt.Texts(), "")+st.Text.Text, "")
		case scenes.Clear:
			lines = append(lines, "[CLEAR]")
		case scenes.Wait:
		case scenes.Board, scenes.Run:
			t.Fatalf("%s: not a console scene", s.Slug)
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func typedLines(s scenes.Scene) []string {
	var out []string
	for _, st := range s.Steps {
		if ty, ok := st.(scenes.Type); ok {
			out = append(out, ty.Text.Text)
		}
	}
	return out
}

// An interactive scene's Type steps, fed through the real persona, print the scene's lines:
// the scenes double as end-to-end persona tests.
func TestInteractiveScenesMatchThePersona(t *testing.T) {
	t.Parallel()
	n := 0
	for _, s := range scenes.All() {
		if !s.Interactive {
			continue
		}
		n++
		t.Run(s.Slug, func(t *testing.T) {
			t.Parallel()
			p := testkit.Start(t, wopr.New(catalog.Registry(), nil, wopr.Options{}), host.Placement{}, host.Config{Seed: 1})
			for _, line := range typedLines(s) {
				p.Type(line)
			}
			if want, got := sceneText(t, s), p.Transcript(); !strings.HasPrefix(got, want) {
				t.Errorf("the scene shows:\n%s\nthe persona prints:\n%s", want, got)
			}
		})
	}
	if n < 2 {
		t.Errorf("%d interactive scenes; first-contact and joshua are the persona's", n)
	}
}

// scene returns a scene by slug.
func scene(t *testing.T, slug string) scenes.Scene {
	t.Helper()
	for _, s := range scenes.All() {
		if s.Slug == slug {
			return s
		}
	}
	t.Fatalf("no scene %q", slug)
	return scenes.Scene{}
}

func transcriptLines(s *testkit.Session) []string {
	return strings.Split(strings.TrimSuffix(s.Transcript(), "\n"), "\n")
}

// gtwGame runs the real Global Thermonuclear War from the catalog.
func gtwGame(t *testing.T) *testkit.GameSession {
	t.Helper()
	e, ok := catalog.Registry().Get(gtw.Slug)
	if !ok {
		t.Fatal("no GTW in the catalog")
	}
	return testkit.Game(t, e.New(), e.Info, "", Seed)
}

// The first-strike scene, board included, is what GTW prints for the film's choices; and the
// choices it types are the ones the board plays.
func TestFirstStrikeIsTheGames(t *testing.T) {
	t.Parallel()
	s := scene(t, "first-strike")
	typed := typedLines(s)
	if want := append(append([]string{gtw.FilmSide}, gtw.FilmTargets...), "", ""); strings.Join(typed, "|") != strings.Join(want, "|") {
		t.Fatalf("the scene types %q; the board plays %q", typed, want)
	}
	g := gtwGame(t)
	for _, line := range typed {
		g.Type(line)
	}
	movie := transcriptLines(play(t, Options{Scene: s.Slug, Single: true}))[1:] // after the scene's own [CLEAR]
	game := transcriptLines(g.Session)
	if len(game) < len(movie) || strings.Join(game[:len(movie)], "\n") != strings.Join(movie, "\n") {
		t.Errorf("the scene:\n%s\nthe game:\n%s", strings.Join(movie, "\n"), strings.Join(game, "\n"))
	}
}

// The climax's NORAD notices are GTW's replies to the same lines, up to tic-tac-toe.
func TestClimaxIsTheGames(t *testing.T) {
	t.Parallel()
	g := gtwGame(t)
	for _, line := range append(append([]string{gtw.FilmSide}, gtw.FilmTargets...), "", "", "", "") {
		g.Type(line) // to the kill ratios
	}
	start := len(transcriptLines(g.Session)) + 2 // after the echo of the Enter that starts the climax
	g.Type("")
	for _, line := range typedLines(scene(t, "climax")) {
		g.Type(line)
	}
	game := transcriptLines(g.Session)[start:]
	game = game[:len(game)-1] // GTW cannot launch tic-tac-toe under testkit.Game; the movie can
	movie := transcriptLines(play(t, Options{Scene: "climax", Single: true}))[1:]
	if len(movie) < len(game) || strings.Join(movie[:len(game)], "\n") != strings.Join(game, "\n") {
		t.Errorf("the scene:\n%s\nthe game:\n%s", strings.Join(movie, "\n"), strings.Join(game, "\n"))
	}
	if !strings.Contains(strings.Join(movie, "\n"), "STALEMATE. WANT TO PLAY AGAIN?\nZERO") {
		t.Errorf("the climax plays tic-tac-toe to zero players:\n%s", strings.Join(movie, "\n"))
	}
}

// The whole film, as wopr -m 1 -i plays it: every scene, then exit 0. The same every time.
func TestTheWholeFilm(t *testing.T) {
	t.Parallel()
	s := play(t, Options{Scene: "1"})
	if !s.Exited() {
		t.Fatalf("the movie ends by itself:\n%s", s.Transcript())
	}
	if again := play(t, Options{Scene: "1"}); again.Transcript() != s.Transcript() {
		t.Error("two replays differ")
	}
	for _, want := range []string{"HELP NOT AVAILABLE", "FINE.", "DEFCON 3.", "TO WIN THE GAME.", "PROJECTED", "GOOSE ISLAND", "NOT TO PLAY."} {
		if !s.Contains(want) {
			t.Errorf("missing %q", want)
		}
	}
	for _, l := range s.Wide(80) {
		t.Errorf("wider than 80 columns: %q", l)
	}
	golden.AssertString(t, "film", s.Transcript())
}
