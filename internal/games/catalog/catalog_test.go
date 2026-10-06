package catalog

import (
	"errors"
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/ending"
	"github.com/GhostofGoes/WOPR/internal/games/gtw"
)

func TestRegistry(t *testing.T) {
	t.Parallel()
	r := Registry()
	if got := len(r.Listed()); got != 15 {
		t.Fatalf("LIST GAMES has %d entries, want 15", got)
	}
	if got := len(r.All()); got != 16 {
		t.Fatalf("registry has %d entries, want 16", got)
	}
	last := r.Listed()[14].Info
	if last.Name != "GLOBAL THERMONUCLEAR WAR" || last.Number != 15 {
		t.Errorf("15th listed game is %q (%d)", last.Name, last.Number)
	}
	for _, e := range r.All() {
		if e.Info.Layout != 0 && e.Info.PanelRows > 12 {
			t.Errorf("%s: PanelRows %d leaves too little console at 80x24", e.Info.Slug, e.Info.PanelRows)
		}
	}
}

func TestResolve(t *testing.T) {
	t.Parallel()
	r := Registry()
	same := []string{"15", "gtw", "Global Thermonuclear War", "global-thermonuclear-war", "global thermo", "GLOBAL"}
	for _, in := range same {
		e, err := r.Resolve(in)
		if err != nil || e.Info.Slug != "global-thermonuclear-war" {
			t.Errorf("Resolve(%q) = %v, %v; want global-thermonuclear-war", in, e.Info.Slug, err)
		}
	}
	for in, want := range map[string]string{
		"7": "chess", "chess": "chess", "che": "", "tic-tac-toe": "tic-tac-toe", "ttt": "tic-tac-toe",
		"Falken's Maze": "falkens-maze", "Falkens-Maze": "falkens-maze", "blackjack": "black-jack",
		"air-to-ground": "air-to-ground-actions",
	} {
		e, err := r.Resolve(in)
		if want == "" {
			var amb *games.AmbiguousError
			if !errors.As(err, &amb) {
				t.Errorf("Resolve(%q) = %v, %v; want ambiguous", in, e.Info.Slug, err)
			}
			continue
		}
		if err != nil || e.Info.Slug != want {
			t.Errorf("Resolve(%q) = %v, %v; want %s", in, e.Info.Slug, err, want)
		}
	}
	var amb *games.AmbiguousError
	if _, err := r.Resolve("theaterwide"); !errors.As(err, &amb) || len(amb.Candidates) != 2 {
		t.Errorf("theaterwide must be ambiguous between two games, got %v", err)
	}
	for _, in := range []string{"16", "0", "", "pong", "  "} {
		if _, err := r.Resolve(in); !errors.Is(err, games.ErrNotFound) {
			t.Errorf("Resolve(%q) = %v, want ErrNotFound", in, err)
		}
	}
}

// FuzzResolve checks Resolve's contract on the real catalog: it never panics, a match
// resolves back from its own slug, a failure is ErrNotFound or a genuine ambiguity, and an
// Exact match is what Resolve returns too.
func FuzzResolve(f *testing.F) {
	for _, s := range []string{"15", "gtw", "chess", "Global Thermonuclear War", "fal", "b", "-1", "fifteen", "", "  ", "black jack"} {
		f.Add(s)
	}
	r := Registry()
	f.Fuzz(func(t *testing.T, in string) {
		e, err := r.Resolve(in)
		if err == nil {
			back, err := r.Resolve(e.Info.Slug)
			if err != nil || back.Info.Slug != e.Info.Slug {
				t.Fatalf("%q resolved to %s, which does not resolve to itself", in, e.Info.Slug)
			}
		} else {
			var amb *games.AmbiguousError
			switch {
			case errors.As(err, &amb):
				if len(amb.Candidates) < 2 {
					t.Fatalf("%q: ambiguity with %d candidates", in, len(amb.Candidates))
				}
			case !errors.Is(err, games.ErrNotFound):
				t.Fatalf("%q: unexpected error %v", in, err)
			}
		}
		if x, ok := r.Exact(in); ok && (err != nil || x.Info.Slug != e.Info.Slug) {
			t.Fatalf("%q: Exact found %s but Resolve gave %v, %v", in, x.Info.Slug, e.Info.Slug, err)
		}
	})
}

// GTW prints the film's game list at the climax; it must match the registry's.
func TestGTWListMatchesTheRegistry(t *testing.T) {
	t.Parallel()
	var names []string
	for _, e := range Registry().Listed() {
		names = append(names, e.Info.Name)
	}
	if got := gtw.FilmList(); strings.Join(got, "|") != strings.Join(names, "|") {
		t.Errorf("GTW's list %v differs from the registry's %v", got, names)
	}
	if _, ok := Registry().Get(ending.Slug); !ok {
		t.Error("the ending must be resolvable by the host")
	}
	if _, err := Registry().Resolve("ending"); err == nil {
		t.Error("but never chosen by a player")
	}
}
