package catalog

import (
	"errors"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games"
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
