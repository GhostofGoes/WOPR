package biotoxic_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/biotoxic"
	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
	"github.com/GhostofGoes/WOPR/internal/sim"
)

var info = games.Info{Name: "THEATERWIDE BIOTOXIC AND CHEMICAL WARFARE", Slug: "theaterwide-biotoxic-and-chemical-warfare", Layout: proto.LayoutConsole}

func play(t *testing.T, seed uint64, strategy func(*sim.State, *sim.Unit) string) *testkit.GameSession {
	t.Helper()
	g := biotoxic.New().(*sim.Game)
	s := testkit.Game(t, g, info, "", seed)
	for range 300 {
		if asking, _ := s.Asking(); !asking {
			break
		}
		if bad := g.IllegalAIOrders(); len(bad) > 0 {
			t.Fatalf("seed %d: illegal WOPR orders: %v", seed, bad)
		}
		s.Type(strategy(g.State(), g.Next()))
	}
	if _, over := s.Result(); !over {
		t.Fatalf("seed %d: no end:\n%s", seed, s.Transcript())
	}
	return s
}

// releaser poisons the enemy's side at every chance.
func releaser(s *sim.State, u *sim.Unit) string {
	if u.Type.Name == "CHM" {
		return "RELEASE " + string(rune('1'+min(u.Region+2, len(s.Regions)-1)))
	}
	return sim.Simple(s, u)
}

// careful never releases anything and cleans where it can.
func careful(s *sim.State, u *sim.Unit) string {
	if u.Type.Name == "CHM" && biotoxic.Decon.Check(s, u, -1) == "" {
		return "DECON"
	}
	return "HOLD"
}

func TestTranscript(t *testing.T) {
	t.Parallel()
	golden.AssertString(t, "transcript", play(t, 42, releaser).Transcript())
}

// WINNER: NONE, whatever anyone does.
func TestNobodyWins(t *testing.T) {
	t.Parallel()
	for name, strategy := range map[string]func(*sim.State, *sim.Unit) string{"releaser": releaser, "careful": careful, "simple": sim.Simple} {
		for seed := range uint64(25) {
			res, _ := play(t, seed, strategy).Result()
			if res.Outcome != proto.NoWinner {
				t.Fatalf("%s seed %d: outcome %v", name, seed, res.Outcome)
			}
		}
	}
}

func TestEveryLineHasProvenance(t *testing.T) {
	t.Parallel()
	notice, err := os.ReadFile(filepath.Join("..", "..", "..", "NOTICE.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, err := range script.Validate(biotoxic.Lines, string(notice)) {
		t.Error(err)
	}
}
