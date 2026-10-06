// Package desertwarfare is Desert Warfare on the sim engine: armour against armour along
// a coast road, with supply lines. A unit is in supply when no enemy holds a region
// between it and its depot; out of supply it attacks at half. Take the enemy depot to win,
// or hold more of the road after ten turns.
package desertwarfare

import (
	"fmt"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
	"github.com/GhostofGoes/WOPR/internal/sim"
)

// Script text, all original.
var (
	lineIntro = script.Orig(
		"DESERT WARFARE. ARMOUR AGAINST ARMOUR ALONG THE COAST ROAD.",
		"YOUR DEPOT IS AT ALEXANDRIA, WOPR'S AT BENGHAZI. TAKE IT, OR HOLD MORE OF THE ROAD",
		"AFTER TEN TURNS. A UNIT CUT OFF FROM ITS DEPOT ATTACKS AT HALF. TYPE HELP FOR ORDERS.",
	)
	lineTookDepot  = script.Orig("YOUR ARMOUR ROLLS INTO BENGHAZI.")
	lineLostDepot  = script.Orig("WOPR HOLDS ALEXANDRIA.")
	lineHeldMore   = script.Orig("TIME. YOU HOLD MORE OF THE ROAD.")
	lineHeldLess   = script.Orig("TIME. WOPR HOLDS MORE OF THE ROAD.")
	lineHeldEven   = script.Orig("TIME. THE ROAD IS SPLIT EVENLY.")
	lineSandstorm  = script.Orig("A SANDSTORM BLOWS UP. EVERY ATTACK NEXT TURN IS AT HALF.")
	lineConvoy     = script.Orig("A SUPPLY CONVOY GETS THROUGH. ONE OF YOUR REDUCED UNITS IS MADE GOOD.")
	lineWOPRConvoy = script.Orig("WOPR'S CONVOY GETS THROUGH. ONE OF ITS REDUCED UNITS IS MADE GOOD.")
	lineMines      = script.Orig("A MINEFIELD IN THE OPEN COSTS A UNIT A STEP.")
	lineQuiet      = script.Orig("THE FRONT IS QUIET.")
	lineHeat       = script.Orig("THE HEAT SLOWS EVERYONE. NOTHING ELSE HAPPENS.")
	lineAir        = script.Orig("AIR SUPPORT ARRIVES: YOUR ATTACKS NEXT TURN COUNT ONE MORE.")
	lineWOPRAir    = script.Orig("WOPR'S AIR SUPPORT ARRIVES: ITS ATTACKS NEXT TURN COUNT ONE MORE.")
	lineSupply     = script.Orig("SUPPLY: # OF YOUR UNITS CUT OFF")
	unitNames      = script.Orig("ARM", "ARMOUR", "INF", "INFANTRY", "ART", "ARTILLERY")
	title          = script.Orig("DESERT WARFARE")
	regionNames    = script.Orig("ALEXANDRIA", "EL ALAMEIN", "MERSA MATRUH", "SIDI BARRANI", "SOLLUM", "TOBRUK", "BENGHAZI")
)

// Lines is every script block, for the provenance test.
var Lines = []script.Ls{
	lineIntro, lineTookDepot, lineLostDepot, lineHeldMore, lineHeldLess, lineHeldEven, lineSandstorm,
	lineConvoy, lineWOPRConvoy, lineMines, lineQuiet, lineHeat, lineAir, lineWOPRAir, lineSupply, unitNames,
	title, regionNames,
}

var (
	armour    = &sim.UnitType{Name: unitNames[0].Text, Long: unitNames[1].Text, Attack: 6, Defence: 4, Move: 2, Range: 1, Category: unitNames[1].Text}
	infantry  = &sim.UnitType{Name: unitNames[2].Text, Long: unitNames[3].Text, Attack: 3, Defence: 5, Move: 1, Range: 1, Category: unitNames[3].Text}
	artillery = &sim.UnitType{Name: unitNames[4].Text, Long: unitNames[5].Text, Attack: 5, Defence: 2, Move: 1, Range: 2, Category: unitNames[5].Text}
)

const (
	depotPlayer = 0
	depotWOPR   = 6
	turns       = 10
)

// Scenario is the scenario data and hooks.
func Scenario() *sim.Scenario {
	return &sim.Scenario{
		Title:     title[0].Text,
		Intro:     lineIntro,
		Regions:   regions(sim.City, sim.Rough, sim.Open, sim.Open, sim.Rough, sim.City, sim.City),
		TurnLimit: turns,
		Setup: func(s *sim.State) {
			for _, side := range []int{sim.Player, sim.WOPR} {
				front, back := 1, 0
				if side == sim.WOPR {
					front, back = 5, 6
				}
				s.AddUnit(side, armour, front)
				s.AddUnit(side, armour, front)
				s.AddUnit(side, infantry, front)
				s.AddUnit(side, artillery, back)
			}
			s.Control[depotPlayer], s.Control[depotWOPR] = sim.Player, sim.WOPR
		},
		Events: []sim.Event{
			{Text: lineSandstorm[0].Text, Apply: func(s *sim.State) { s.Vars["sandstorm"] = 2 }},
			{Text: lineConvoy[0].Text, Apply: func(s *sim.State) { repair(s, sim.Player) }},
			{Text: lineWOPRConvoy[0].Text, Apply: func(s *sim.State) { repair(s, sim.WOPR) }},
			{Text: lineMines[0].Text, Apply: mines},
			{Text: lineQuiet[0].Text},
			{Text: lineHeat[0].Text},
			{Text: lineAir[0].Text, Apply: func(s *sim.State) { s.Vars["air0"] = 2 }},
			{Text: lineWOPRAir[0].Text, Apply: func(s *sim.State) { s.Vars["air1"] = 2 }},
		},
		AttackMod: func(s *sim.State, u *sim.Unit, a int) int {
			if !inSupply(s, u) {
				a = (a + 1) / 2
			}
			if s.Vars[fmt.Sprintf("air%d", u.Side)] > 0 {
				a++
			}
			if s.Vars["sandstorm"] > 0 {
				a = (a + 1) / 2
			}
			return a
		},
		Upkeep: func(s *sim.State) {
			for _, k := range []string{"sandstorm", "air0", "air1"} {
				if s.Vars[k] > 0 {
					s.Vars[k]-- // set to 2 by the card: it lasts through the next turn
				}
			}
		},
		Status: func(s *sim.State) string {
			n := 0
			for _, u := range s.Living(sim.Player) {
				if !inSupply(s, u) {
					n++
				}
			}
			return fillN(lineSupply[0].Text, n)
		},
		Victory: victory,
	}
}

// regions names the strip's regions, rear to rear, with the given terrain.
func regions(terrain ...sim.Terrain) []sim.Region {
	out := make([]sim.Region, len(terrain))
	for i, t := range terrain {
		out[i] = sim.Region{Name: regionNames[i].Text, Terrain: t}
	}
	return out
}

// New returns a game.
func New() games.Game { return sim.NewGame(Scenario()) }

func fillN(text string, n int) string {
	for i := range len(text) {
		if text[i] == '#' {
			return text[:i] + fmt.Sprint(n) + text[i+1:]
		}
	}
	return text
}

// inSupply reports whether no enemy holds a region between the unit and its depot.
func inSupply(s *sim.State, u *sim.Unit) bool {
	depot := depotPlayer
	if u.Side == sim.WOPR {
		depot = depotWOPR
	}
	lo, hi := min(depot, u.Region), max(depot, u.Region)
	for r := lo; r <= hi; r++ {
		if s.Control[r] == sim.Enemy(u.Side) {
			return false
		}
	}
	return true
}

// repair makes one of a side's reduced units whole, chosen with the stream.
func repair(s *sim.State, side int) {
	var reduced []*sim.Unit
	for _, u := range s.Living(side) {
		if u.Steps == 1 {
			reduced = append(reduced, u)
		}
	}
	if len(reduced) > 0 {
		reduced[s.Rand().IntN(len(reduced))].Steps = 2
	}
}

// mines cost a step to one unit standing in open ground, if any.
func mines(s *sim.State) {
	var exposed []*sim.Unit
	for _, u := range s.Units {
		if u.Alive() && s.Regions[u.Region].Terrain == sim.Open {
			exposed = append(exposed, u)
		}
	}
	if len(exposed) > 0 {
		s.Lose(exposed[s.Rand().IntN(len(exposed))], 1)
	}
}

func victory(s *sim.State, final bool) (proto.Outcome, string, bool) {
	switch {
	case s.Control[depotWOPR] == sim.Player:
		return proto.Win, lineTookDepot[0].Text, true
	case s.Control[depotPlayer] == sim.WOPR:
		return proto.Loss, lineLostDepot[0].Text, true
	case !final:
		return 0, "", false
	}
	mine, theirs := 0, 0
	for _, c := range s.Control {
		switch c {
		case sim.Player:
			mine++
		case sim.WOPR:
			theirs++
		}
	}
	switch {
	case mine > theirs:
		return proto.Win, lineHeldMore[0].Text, true
	case theirs > mine:
		return proto.Loss, lineHeldLess[0].Text, true
	}
	return proto.Draw, lineHeldEven[0].Text, true
}
