package guerrilla_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/guerrilla"
	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
	"github.com/GhostofGoes/WOPR/internal/sim"
)

var info = games.Info{Name: "GUERRILLA ENGAGEMENT", Slug: "guerrilla-engagement", Layout: proto.LayoutConsole}

// cunning plays like a guerrilla: hide when exposed, raid only at good odds, recruit when
// support allows, otherwise lie low.
func cunning(s *sim.State, u *sim.Unit) string {
	if !u.Hidden && u.Steps == 1 {
		return "HIDE"
	}
	for at := range s.Regions {
		if s.CanAttack(u, at) {
			for _, e := range s.In(at, sim.WOPR) {
				if col := sim.Column(s.AttackOf(u), s.DefenceOf(e), s.Regions[at].Terrain); col >= 2 || u.Hidden && col >= 1 {
					return fmt.Sprintf("ATTACK %d", at+1)
				}
			}
		}
	}
	if s.Vars["support"] >= 3 && s.Control[u.Region] != sim.WOPR {
		return "RECRUIT"
	}
	if !u.Hidden {
		return "HIDE"
	}
	return "HOLD"
}

func play(t *testing.T, seed uint64, strategy func(*sim.State, *sim.Unit) string) *testkit.GameSession {
	t.Helper()
	g := guerrilla.New().(*sim.Game)
	s := testkit.Game(t, g, info, "", seed)
	for range 300 {
		if asking, _ := s.Asking(); !asking {
			break
		}
		if bad := g.IllegalAIOrders(); len(bad) > 0 {
			t.Fatalf("seed %d: illegal WOPR orders: %v", seed, bad)
		}
		for _, u := range g.State().Units {
			if u.Side == sim.Player && u.Hidden && u.Alive() {
				for at := range g.State().Regions {
					for _, w := range g.State().Living(sim.WOPR) {
						if w.Region == at && g.State().CanAttack(w, u.Region) && !visibleTarget(g.State(), u.Region) {
							t.Fatalf("seed %d: WOPR can attack a region holding only hidden cells", seed)
						}
					}
				}
			}
		}
		s.Type(strategy(g.State(), g.Next()))
	}
	if _, over := s.Result(); !over {
		t.Fatalf("seed %d: no end:\n%s", seed, s.Transcript())
	}
	return s
}

func visibleTarget(s *sim.State, r int) bool {
	for _, u := range s.In(r, sim.Player) {
		if !u.Hidden {
			return true
		}
	}
	return false
}

func TestTranscript(t *testing.T) {
	t.Parallel()
	golden.AssertString(t, "transcript", play(t, 42, cunning).Transcript())
}

func TestSeededGames(t *testing.T) {
	t.Parallel()
	for name, strategy := range map[string]func(*sim.State, *sim.Unit) string{"cunning": cunning, "simple": sim.Simple} {
		results := map[proto.Outcome]int{}
		for seed := range uint64(40) {
			res, _ := play(t, seed, strategy).Result()
			results[res.Outcome]++
		}
		t.Logf("%s: wins %d, losses %d", name, results[proto.Win], results[proto.Loss])
		if name == "cunning" && results[proto.Win] == 0 {
			t.Error("a careful guerrilla should be able to win sometimes")
		}
	}
}

func TestEveryLineHasProvenance(t *testing.T) {
	t.Parallel()
	notice, err := os.ReadFile(filepath.Join("..", "..", "..", "NOTICE.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, err := range script.Validate(guerrilla.Lines, string(notice)) {
		t.Error(err)
	}
}
