package tictactoe_test

import (
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/games/tictactoe"
	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// In movie mode the climax plays itself (docs/PLAN.md §7): one player, a game in which both
// sides play perfectly, stalemate, then zero players and the ending in movie mode, with the
// launch code carried on. Every seed ends in stalemate.
func TestMovieModePlaysTheFilmsClimax(t *testing.T) {
	t.Parallel()
	for seed := range uint64(40) {
		g := testkit.Game(t, tictactoe.New(), info, tictactoe.MovieMode+":5", seed)
		res, over := g.Result()
		if !over || res.Outcome != proto.NoWinner {
			t.Fatalf("seed %d: the scene plays to the hand-off unaided: %+v\n%s", seed, res, g.Transcript())
		}
		if l := g.Launches(); len(l) != 2 || l[1] != tictactoe.EndingSlug {
			t.Fatalf("seed %d: launches %v", seed, l)
		}
		tr := g.Transcript()
		for _, want := range []string{"PLEASE LIST NUMBER OF PLAYERS: 1\n\n", "STALEMATE. WANT TO PLAY AGAIN?\nZERO\n\n"} {
			if !strings.Contains(tr, want) {
				t.Fatalf("seed %d: missing %q:\n%s", seed, want, tr)
			}
		}
		if seed == 1 {
			golden.AssertString(t, "movie", tr)
		}
	}
}

// The ending hears the movie mode, launch code included.
func TestMovieModeHandsTheModeOn(t *testing.T) {
	t.Parallel()
	g := tictactoe.New()
	g.Start(proto.Env{Seed: 1, Instant: true, Mode: "movie:4"})
	for _, o := range g.Handle(proto.LineEvent{Text: "ZERO"}) {
		if d, ok := o.(proto.Done); ok {
			if d.Result.Next == nil || d.Result.Next.Mode != "movie:4" {
				t.Errorf("hand-off %+v", d.Result.Next)
			}
			return
		}
	}
	t.Error("zero players must hand off")
}
