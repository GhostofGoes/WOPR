package theaterwide_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/games/theaterwide"
	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
	"github.com/GhostofGoes/WOPR/internal/sim"
)

var info = games.Info{Name: "THEATERWIDE TACTICAL WARFARE", Slug: "theaterwide-tactical-warfare", Layout: proto.LayoutConsole}

func play(t *testing.T, seed uint64, strategy func(*sim.State, *sim.Unit) string) *testkit.GameSession {
	t.Helper()
	g := theaterwide.New().(*sim.Game)
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

// escalator climbs the ladder every turn with its first unit, and otherwise plays simply.
func escalator(s *sim.State, u *sim.Unit) string {
	if units := s.Living(sim.Player); len(units) > 0 && u == firstOf(units) && theaterwide.Escalate.Check(s, u, -1) == "" {
		return "ESCALATE"
	}
	return sim.Simple(s, u)
}

func firstOf(us []*sim.Unit) *sim.Unit {
	best := us[0]
	for _, u := range us[1:] {
		if u.Type.Name < best.Type.Name || u.Type.Name == best.Type.Name && u.ID < best.ID {
			best = u
		}
	}
	return best
}

func TestTranscript(t *testing.T) {
	t.Parallel()
	golden.AssertString(t, "transcript", play(t, 42, sim.Simple).Transcript())
}

// Climbing the ladder ends in a strategic exchange that nobody wins: WOPR answers every
// rung, unless the talks hold it off for a while.
func TestEscalationEndsInNoWinner(t *testing.T) {
	t.Parallel()
	for seed := range uint64(20) {
		s := play(t, seed, escalator)
		res, _ := s.Result()
		if res.Outcome != proto.NoWinner || !s.Contains("STRATEGIC EXCHANGE.") {
			t.Fatalf("seed %d: escalating every turn ended %v:\n%s", seed, res.Outcome, s.Transcript())
		}
		if !strings.Contains(strings.Join(res.Lines, "\n"), "CIVILIANS") {
			t.Fatalf("seed %d: the kill ratios count civilians: %q", seed, res.Lines)
		}
	}
}

func TestConventionalGamesEnd(t *testing.T) {
	t.Parallel()
	outcomes := map[proto.Outcome]int{}
	for seed := range uint64(30) {
		res, _ := play(t, seed, sim.Simple).Result()
		outcomes[res.Outcome]++
	}
	t.Logf("simple play: win %d, loss %d, draw %d, none %d", outcomes[proto.Win], outcomes[proto.Loss], outcomes[proto.Draw], outcomes[proto.NoWinner])
}

func TestEveryLineHasProvenance(t *testing.T) {
	t.Parallel()
	notice, err := os.ReadFile(filepath.Join("..", "..", "..", "NOTICE.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, err := range script.Validate(theaterwide.Lines, string(notice)) {
		t.Error(err)
	}
}
