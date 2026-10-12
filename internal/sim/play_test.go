// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package sim

import (
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

var (
	testArm = &UnitType{Name: "ARM", Long: "ARMOUR", Attack: 6, Defence: 4, Move: 2, Range: 1, Category: "ARMOUR"}
	testArt = &UnitType{Name: "ART", Long: "ARTILLERY", Attack: 5, Defence: 2, Move: 1, Range: 2, Category: "ARTILLERY"}
	testGue = &UnitType{Name: "GUE", Long: "GUERRILLA", Attack: 2, Defence: 2, Move: 1, Range: 1, Category: "INFANTRY"}
)

// battleScenario is a longer strip with every hook set and WOPR on DefaultAI, so that whole
// games reach combat, retreats, losses, the event deck and each ending.
func battleScenario() *Scenario {
	return &Scenario{
		Title: "BATTLE",
		Intro: script.Orig("BATTLE."),
		Art:   script.Orig("ART"),
		Regions: []Region{
			{Name: "ONE"},
			{Name: "TWO", Terrain: Rough},
			{Name: "THREE"},
			{Name: "FOUR", Terrain: City},
			{Name: "FIVE"},
			{Name: "SIX"},
		},
		TurnLimit: 8,
		Events: []Event{
			{Text: "RAIN.", Apply: func(s *State) { s.Vars["rain"]++ }},
			{Text: "QUIET."},
			{Apply: func(s *State) { s.Vars["silent"]++ }},
		},
		Setup: func(s *State) {
			s.AddUnit(Player, testInf, 0)
			s.AddUnit(Player, testArm, 1)
			s.AddUnit(Player, testArt, 0)
			s.AddUnit(WOPR, testInf, 4)
			s.AddUnit(WOPR, testArm, 4)
			s.AddUnit(WOPR, testArt, 5)
			s.AddUnit(WOPR, testGue, 3).Hidden = true
		},
		AttackMod: func(s *State, _ *Unit, a int) int { return a - s.Vars["rain"]%2 },
		Upkeep:    func(s *State) { s.Vars["upkeep"]++ },
		Status:    func(s *State) string { return "RAIN " + strings.Repeat("*", s.Vars["rain"]%4) },
		RegionNote: func(_ *State, r int) string {
			if r == 3 {
				return "(RUINS)"
			}
			return ""
		},
	}
}

// play gives every order Simple would and returns the result, failing on a game that does
// not end within its turn limit.
func play(t *testing.T, sc *Scenario, seed uint64) (proto.Result, *Game) {
	t.Helper()
	g := NewGame(sc)
	outs := g.Start(proto.Env{Seed: seed, Deterministic: true, Instant: true})
	for range 200 {
		for _, o := range outs {
			if d, ok := o.(proto.Done); ok {
				return d.Result, g
			}
		}
		u := g.Next()
		if u == nil {
			t.Fatalf("seed %d: no unit to order and no result", seed)
		}
		outs = g.Handle(proto.LineEvent{Text: Simple(g.State(), u)})
	}
	t.Fatalf("seed %d: no result after 200 orders", seed)
	return proto.Result{}, nil
}

// Whole games on many seeds end within the turn limit, keep every unit on the strip with 0 to
// 2 steps, count every lost step in the ratios, and reach more than one ending.
func TestWholeGames(t *testing.T) {
	t.Parallel()
	outcomes := map[proto.Outcome]int{}
	for seed := range uint64(40) {
		res, g := play(t, battleScenario(), seed)
		outcomes[res.Outcome]++
		s := g.State()
		if s.Turn > g.sc.TurnLimit {
			t.Errorf("seed %d: turn %d past the limit", seed, s.Turn)
		}
		lost := [2]int{}
		for _, u := range s.Units {
			if u.Region < 0 || u.Region >= len(s.Regions) || u.Steps < 0 || u.Steps > 2 {
				t.Errorf("seed %d: %s in region %d with %d steps", seed, u.Name(), u.Region, u.Steps)
			}
			lost[u.Side] += 2 - u.Steps
		}
		for side, want := range lost {
			got := 0
			for _, n := range s.Losses[side] {
				got += n
			}
			if got != want {
				t.Errorf("seed %d: side %d lost %d steps, the ratios count %d", seed, side, want, got)
			}
		}
		if s.Vars["upkeep"] != s.Turn {
			t.Errorf("seed %d: upkeep ran %d times in %d turns", seed, s.Vars["upkeep"], s.Turn)
		}
		if g.Handle(proto.LineEvent{Text: "HOLD"}) != nil {
			t.Errorf("seed %d: an order after the end was answered", seed)
		}
	}
	if len(outcomes) < 2 {
		t.Errorf("40 games all ended the same way: %v", outcomes)
	}
}

// The commands that are not orders, and orders the engine refuses, ask for the same unit
// again; END holds every unit left and resolves the turn.
func TestOrderPrompts(t *testing.T) {
	t.Parallel()
	g := NewGame(battleScenario())
	g.Start(proto.Env{Seed: 1, Instant: true})
	first := g.Next()
	if g.Handle(proto.KeyEvent{Key: proto.KeyEnter}) != nil {
		t.Error("a key event was answered")
	}
	for in, want := range map[string]string{
		"STATUS":       "BATTLE",
		"map":          "(RUINS)",
		"HELP":         "ORDERS: MOVE, ATTACK, HOLD",
		"FROBNICATE":   "UNKNOWN ORDER",
		"MOVE":         "WHICH REGION",
		"MOVE NOWHERE": "WHICH REGION",
		"MOVE 6":       "CANNOT",
		"ATTACK 5":     "NO ENEMY THERE",
	} {
		outs := g.Handle(proto.LineEvent{Text: in})
		if got := said(outs) + promptsOf(outs); !strings.Contains(got, want) {
			t.Errorf("%s: %q lacks %q", in, got, want)
		}
		if g.Next() != first {
			t.Fatalf("%s moved on to the next unit", in)
		}
	}
	g.Handle(proto.LineEvent{Text: ""}) // a blank line holds
	if g.Next() == first {
		t.Fatal("a blank line did not hold the unit")
	}
	if outs := g.Handle(proto.LineEvent{Text: "END"}); g.State().Turn != 2 {
		t.Errorf("END did not resolve the turn: %q", said(outs))
	}
}

func promptsOf(outs []proto.Output) string {
	var b strings.Builder
	for _, o := range outs {
		if p, ok := o.(proto.Prompt); ok {
			b.WriteString(p.Text + "\n")
		}
	}
	return b.String()
}

// The engine's own rule decides a game the scenario leaves open: a side with no units has
// lost, both sides gone is no winner, and the last turn is a draw.
func TestDecide(t *testing.T) {
	t.Parallel()
	g := NewGame(testScenario())
	g.Start(proto.Env{Seed: 1, Instant: true})
	you, wopr := g.s.Living(Player)[0], g.s.Living(WOPR)[0]
	check := func(name string, final bool, want proto.Outcome, wantDone bool) {
		t.Helper()
		if o, _, done := g.decide(final); done != wantDone || (done && o != want) {
			t.Errorf("%s: %v, %v; want %v, %v", name, o, done, want, wantDone)
		}
	}
	check("both alive", false, 0, false)
	check("last turn", true, proto.Draw, true)
	wopr.Steps = 0
	check("WOPR gone", false, proto.Win, true)
	you.Steps = 0
	check("both gone", false, proto.NoWinner, true)
	wopr.Steps = 2
	check("you gone", false, proto.Loss, true)

	g.sc.Victory = func(*State, bool) (proto.Outcome, string, bool) { return proto.Win, "WON.", true }
	if o, why, done := g.decide(false); o != proto.Win || why != "WON." || !done {
		t.Errorf("the scenario's victory was overruled: %v %q %v", o, why, done)
	}
}

// Each combat result does what the table says to both units.
func TestAttackResults(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name            string
		res             Result
		block           bool // a player unit behind the defender, so it cannot retreat
		weak            bool // the defender starts reduced, for better odds
		attSteps, defSt int
		defAt           int
		advanced        bool
	}{
		{name: "attacker loses", res: AttackerLoses, attSteps: 1, defSt: 2, defAt: 1},
		{name: "exchange", res: Exchange, attSteps: 1, defSt: 1, defAt: 1},
		{name: "retreat", res: DefenderRetreats, attSteps: 2, defSt: 2, defAt: 2, advanced: true},
		{name: "retreat blocked", res: DefenderRetreats, block: true, attSteps: 2, defSt: 1, defAt: 1},
		{name: "defender loses", res: DefenderLoses, weak: true, attSteps: 2, defSt: 0, defAt: 1, advanced: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Find a seed whose first die gives this result at these odds.
			for seed := range uint64(200) {
				g := NewGame(testScenario())
				g.Start(proto.Env{Seed: seed, Instant: true})
				s := g.s
				att, def := s.Living(Player)[0], s.Living(WOPR)[0]
				if tc.weak {
					def.Steps = 1
				}
				if tc.block {
					s.AddUnit(Player, testInf, 2)
				}
				roll := proto.NewRand(seed, 0).IntN(6) + 1
				if CRT(s.AttackOf(att), s.DefenceOf(def), s.Regions[1].Terrain, roll) != tc.res {
					continue
				}
				s.Attack(att, 1)
				if att.Steps != tc.attSteps || def.Steps != tc.defSt || def.Region != tc.defAt {
					t.Errorf("attacker %d steps; defender %d steps in region %d", att.Steps, def.Steps, def.Region)
				}
				if advanced := att.Region == 1; advanced != tc.advanced {
					t.Errorf("attacker advanced: %v, want %v", advanced, tc.advanced)
				}
				return
			}
			t.Fatal("no seed under 200 rolls this result")
		})
	}
}

// An attack with no visible defender, or by a destroyed unit, does nothing.
func TestAttackWithoutDefender(t *testing.T) {
	t.Parallel()
	g := NewGame(testScenario())
	g.Start(proto.Env{Seed: 1, Instant: true})
	s := g.s
	att, def := s.Living(Player)[0], s.Living(WOPR)[0]
	def.Hidden = true
	s.Attack(att, 1)
	def.Hidden = false
	att.Steps = 0
	s.Attack(att, 1)
	if att.Attacked || def.Steps != 2 || len(s.said) > 0 {
		t.Errorf("an empty attack had effects: %v %d %q", att.Attacked, def.Steps, s.said)
	}
}

// DefaultAI attacks at good odds, pulls a reduced unit back when threatened, and otherwise
// advances or holds.
func TestDefaultAI(t *testing.T) {
	t.Parallel()
	sc := testScenario()
	sc.AI = nil
	g := NewGame(sc)
	g.Start(proto.Env{Seed: 1, Instant: true})
	s := g.s
	you, wopr := s.Living(Player)[0], s.Living(WOPR)[0]
	orders := g.AIOrders()
	if len(orders) != 1 || orders[0].Unit != wopr || orders[0].Verb != Hold {
		t.Fatalf("at even odds with the way west blocked: %+v, want HOLD", orders)
	}
	you.Steps = 1 // weak now: 2:1 or better
	if o := DefaultAI(s)[0]; o.Verb != Attack || o.Target != 0 {
		t.Errorf("against a weak unit: %s %d, want ATTACK 1", o.Verb.Name, o.Target+1)
	}
	you.Steps, wopr.Steps = 2, 1 // reduced and threatened: fall back
	if o := DefaultAI(s)[0]; o.Verb != Move || o.Target != 2 {
		t.Errorf("reduced and threatened: %s %d, want MOVE 3", o.Verb.Name, o.Target+1)
	}
	wopr.Region, wopr.Steps = 2, 2 // no enemy next door: press on toward the player's rear
	if o := DefaultAI(s)[0]; o.Verb != Move || o.Target != 1 {
		t.Errorf("advancing: %s %d, want MOVE 2", o.Verb.Name, o.Target+1)
	}
}

func TestNearest(t *testing.T) {
	t.Parallel()
	even := func(r int) bool { return r%2 == 0 }
	for _, tc := range []struct{ from, want int }{{0, 0}, {1, 0}, {3, 2}, {5, 4}} {
		if got := Nearest(6, tc.from, even); got != tc.want {
			t.Errorf("Nearest(6, %d, even) = %d, want %d", tc.from, got, tc.want)
		}
	}
	if got := Nearest(6, 2, func(int) bool { return false }); got != -1 {
		t.Errorf("Nearest with no match = %d, want -1", got)
	}
}

func TestSortUnits(t *testing.T) {
	t.Parallel()
	us := []*Unit{{ID: 2, Type: testInf}, {ID: 1, Type: testArt}, {ID: 1, Type: testInf}, {ID: 1, Type: testArm}}
	sortUnits(us)
	var got []string
	for _, u := range us {
		got = append(got, u.Name())
	}
	if strings.Join(got, " ") != "ARM1 ART1 INF1 INF2" {
		t.Errorf("sorted: %v", got)
	}
}
