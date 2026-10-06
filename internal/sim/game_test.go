package sim

import (
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

var testInf = &UnitType{Name: "INF", Long: "INFANTRY", Attack: 4, Defence: 4, Move: 1, Range: 1, Category: "INFANTRY"}

// testScenario: the player's infantry at 1, WOPR's at 2, and WOPR always pulls back to 3.
func testScenario() *Scenario {
	return &Scenario{
		Title:     "TEST",
		Intro:     script.Orig("TEST."),
		Regions:   []Region{{Name: "EL ALAMEIN"}, {Name: "MERSA MATRUH"}, {Name: "SIDI BARRANI", Terrain: Rough}},
		TurnLimit: 3,
		Verbs: []*Verb{{
			Name: "EXPLODE", Help: "EXPLODE: TEST.", Phase: 2, Drastic: true,
			Apply: func(s *State, _ *Unit, _ int) { s.Vars["boom"]++ },
		}},
		Setup: func(s *State) {
			s.AddUnit(Player, testInf, 0)
			s.AddUnit(WOPR, testInf, 1)
		},
		AI: func(s *State) []Order {
			var orders []Order
			for _, u := range s.Living(WOPR) {
				orders = append(orders, Order{Unit: u, Verb: Move, Target: 2})
			}
			return orders
		},
	}
}

func said(outs []proto.Output) string {
	var b strings.Builder
	for _, o := range outs {
		if s, ok := o.(proto.Say); ok {
			b.WriteString(strings.Join(s.Lines, "\n") + "\n")
		}
	}
	return b.String()
}

func TestParsing(t *testing.T) {
	t.Parallel()
	g := NewGame(testScenario())
	g.Start(proto.Env{Seed: 1, Instant: true})
	regions := map[string]int{"1": 0, "EL ALAMEIN": 0, "ALAMEIN": 0, "MATRUH": 1, "M": 1, "SIDI": 2, "BAR": 2, "BA": -1, "CAIRO": -1, "4": -1}
	for in, want := range regions {
		if got := g.region(in); got != want {
			t.Errorf("region %q = %d, want %d", in, got, want)
		}
	}
	if g.verb("M") != Move || g.verb("H") != Hold || g.verb("EXPLODE") == nil {
		t.Error("verbs by prefix and by name")
	}
	if v := g.verb("E"); v != nil {
		t.Errorf("a drastic verb answers to a prefix: %s", v.Name)
	}
	if out := said(g.Handle(proto.LineEvent{Text: "e"})); !strings.Contains(out, "UNKNOWN ORDER") {
		t.Errorf("E: %q", out)
	}
}

func TestTowardOwnRegion(t *testing.T) {
	t.Parallel()
	g := NewGame(testScenario())
	g.Start(proto.Env{Seed: 1, Instant: true})
	u := g.s.Living(Player)[0]
	if to := g.s.Toward(u, 2); to != -1 {
		t.Errorf("Toward through the enemy = %d, want -1", to)
	}
	g.s.Living(WOPR)[0].Region = 2 // the way east is open now
	if to := g.s.Toward(u, u.Region); to != -1 {
		t.Errorf("Toward its own region = %d, want -1 (stay)", to)
	}
	if to := g.s.Toward(u, 2); to != 1 {
		t.Errorf("Toward the enemy = %d, want 1", to)
	}
}

// An order the other side's moves made impossible is reported, not dropped in silence.
func TestVoidedOrderIsReported(t *testing.T) {
	t.Parallel()
	g := NewGame(testScenario())
	g.Start(proto.Env{Seed: 1, Instant: true})
	out := said(g.Handle(proto.LineEvent{Text: "attack 2"}))
	if !strings.Contains(out, "INF1 STANDS FAST: NO ENEMY THERE IN RANGE.") {
		t.Errorf("the voided attack: %q", out)
	}
}

func TestMapAndRatios(t *testing.T) {
	t.Parallel()
	g := NewGame(testScenario())
	g.Start(proto.Env{Seed: 1, Instant: true})
	s := g.s
	s.AddUnit(WOPR, testInf, 1).Steps = 1
	s.AddUnit(WOPR, testInf, 1)
	if got := unitList(s.In(1, WOPR), true); got != "INF1,2*,3" {
		t.Errorf("stacked units = %q", got)
	}
	for _, l := range g.mapLines() {
		if len(l) > 80 {
			t.Errorf("map line wider than 80: %q", l)
		}
	}
	lines := RatioTable([2]map[string]int{{"INFANTRY": 3}, {"INFANTRY": 12}})
	head, row := lines[1], lines[2]
	if strings.Index(head, "YOU")+len("YOU") != strings.Index(row, "3")+1 ||
		strings.Index(head, "WOPR")+len("WOPR") != len(row) {
		t.Errorf("ratio columns do not line up:\n%s\n%s", head, row)
	}
}

func TestEngineText(t *testing.T) {
	t.Parallel()
	for _, block := range EngineLines() {
		for _, l := range block {
			if l.Text != strings.ToUpper(l.Text) || len(l.Text) > 78 {
				t.Errorf("engine line not upper case or too wide for HELP's indent: %q", l.Text)
			}
		}
	}
	for _, v := range []*Verb{Move, Attack, Hold} {
		if v.Help != strings.ToUpper(v.Help) {
			t.Errorf("%s help is not upper case: %q", v.Name, v.Help)
		}
	}
}
