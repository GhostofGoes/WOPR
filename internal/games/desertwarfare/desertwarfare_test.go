package desertwarfare_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/desertwarfare"
	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
	"github.com/GhostofGoes/WOPR/internal/sim"
)

var info = games.Info{Name: "DESERT WARFARE", Slug: "desert-warfare", Layout: proto.LayoutConsole}

// play runs a seeded game with the simple strategy, checking WOPR's orders before every
// player order, and returns the session.
func play(t *testing.T, seed uint64) *testkit.GameSession {
	t.Helper()
	g := desertwarfare.New().(*sim.Game)
	s := testkit.Game(t, g, info, "", seed)
	for step := 0; step < 200; step++ {
		asking, _ := s.Asking()
		if !asking {
			break
		}
		if bad := g.IllegalAIOrders(); len(bad) > 0 {
			t.Fatalf("seed %d: WOPR would give illegal orders: %v", seed, bad)
		}
		s.Type(sim.Simple(g.State(), g.Next()))
	}
	if _, over := s.Result(); !over {
		t.Fatalf("seed %d: the game did not end:\n%s", seed, s.Transcript())
	}
	return s
}

func TestTranscript(t *testing.T) {
	t.Parallel()
	s := play(t, 42)
	golden.AssertString(t, "transcript", s.Transcript())
}

func TestSeededGamesEnd(t *testing.T) {
	t.Parallel()
	outcomes := map[proto.Outcome]int{}
	for seed := range uint64(40) {
		res, _ := play(t, seed).Result()
		outcomes[res.Outcome]++
		if len(res.Lines) < 2 || res.Lines[0] != "KILL RATIOS" {
			t.Fatalf("seed %d: the result carries the kill ratios: %q", seed, res.Lines)
		}
	}
	t.Logf("outcomes over 40 games (win, loss, draw): %d %d %d", outcomes[proto.Win], outcomes[proto.Loss], outcomes[proto.Draw])
	if outcomes[proto.Win] == 40 || outcomes[proto.Loss] == 40 {
		t.Errorf("one side always wins: %v", outcomes)
	}
}

func TestOrdersAreChecked(t *testing.T) {
	t.Parallel()
	g := desertwarfare.New().(*sim.Game)
	s := testkit.Game(t, g, info, "", 1)
	s.Type("fly 3").Type("move 7").Type("attack 7").Type("move").Type("help").Type("status")
	for _, want := range []string{"UNKNOWN ORDER. TYPE HELP.", "IT CANNOT GET THERE THIS TURN.", "NO ENEMY THERE IN RANGE.", "WHICH REGION?", "ORDERS: MOVE, ATTACK, HOLD", "REGION"} {
		if !s.Contains(want) {
			t.Errorf("missing %q:\n%s", want, s.Transcript())
		}
	}
	if g.Next() == nil || g.Next().Name() != "ARM1" {
		t.Fatal("refused orders leave the same unit to order")
	}
	s.Type("m mersa").Type("end")
	if g.State().Turn != 2 {
		t.Fatalf("END holds the rest and plays the turn: turn %d", g.State().Turn)
	}
}

func TestEveryLineHasProvenance(t *testing.T) {
	t.Parallel()
	notice, err := os.ReadFile(filepath.Join("..", "..", "..", "NOTICE.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, err := range script.Validate(append(desertwarfare.Lines, sim.EngineLines()...), string(notice)) {
		t.Error(err)
	}
}
