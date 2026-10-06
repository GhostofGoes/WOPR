// Package guerrilla is Guerrilla Engagement on the sim engine: an asymmetric campaign. The
// player's cells start hidden in the hills; a hidden cell cannot be attacked, and patrols
// find cells near them. Raids that hurt WOPR win popular support, losses cost it, and
// support recruits new cells. The player wins with support at ten or the capital taken,
// loses with no cells left; a cell alive after twelve turns is a win too.
package guerrilla

import (
	"fmt"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
	"github.com/GhostofGoes/WOPR/internal/sim"
)

// Script text, all original. A # is filled in when the line is shown.
var (
	lineIntro = script.Orig(
		"GUERRILLA ENGAGEMENT. YOUR CELLS ARE HIDDEN IN THE HILLS (MARKED ~ ON THE MAP).",
		"A HIDDEN CELL CANNOT BE ATTACKED, AND ITS AMBUSH STRIKES TWICE AS HARD.",
		"ATTACKING GIVES IT AWAY, AND SO MAY A PATROL NEARBY.",
		"RAIDS THAT HURT WOPR WIN SUPPORT; LOSSES COST IT. RECRUIT SPENDS 3 SUPPORT.",
		"SUPPORT 10 OR THE CAPITAL WINS, AND SO DOES A CELL ALIVE AFTER TWELVE TURNS.",
		"TYPE HELP FOR ORDERS.",
	)
	lineRise      = script.Orig("THE COUNTRY RISES WITH YOU.")
	lineCapital   = script.Orig("YOUR CELLS HOLD THE CAPITAL.")
	lineCrushed   = script.Orig("THE LAST CELL IS GONE.")
	lineEndured   = script.Orig("TIME. THE INSURGENCY ENDURES.")
	lineHides     = script.Orig("# GOES TO GROUND.")
	lineFound     = script.Orig("A PATROL FINDS #.")
	lineRecruited = script.Orig("A NEW CELL FORMS AT #.")
	lineNoRecruit = script.Orig("RECRUITING NEEDS 3 SUPPORT AND GROUND WOPR DOES NOT HOLD.")
	lineNoHide    = script.Orig("IT IS ALREADY HIDDEN.")
	lineSupport   = script.Orig("SUPPORT #")
	lineShelter   = script.Orig("THE VILLAGES SHELTER YOUR CELLS. ALL GO TO GROUND.")
	lineInformer  = script.Orig("AN INFORMER TALKS.", "# IS EXPOSED.")
	lineCurfew    = script.Orig("A CURFEW IN THE TOWNS.", "SUPPORT FALLS BY ONE.")
	lineSweep     = script.Orig("A HEAVY-HANDED SWEEP. SUPPORT RISES BY ONE.")
	lineAirdrop   = script.Orig("A SUPPLY DROP REACHES THE HILLS.", "# IS MADE GOOD.")
	lineReinforce = script.Orig("FRESH TROOPS REACH THE CAPITAL.")
	lineQuiet     = script.Orig("A QUIET WEEK.")
	lineMonsoon   = script.Orig("THE RAINS COME. PATROLS FIND NOTHING THIS TURN.")
	verbHelp      = script.Orig("HIDE: GO TO GROUND (NO MOVE, NO ATTACK).", "RECRUIT: SPEND 3 SUPPORT ON A NEW CELL HERE.")
	title         = script.Orig("GUERRILLA ENGAGEMENT")
	regionNames   = script.Orig("BORDER CAMPS", "JUNGLE", "HIGHLANDS", "RIVER VALLEY", "PROVINCE TOWN", "CAPITAL")
	unitNames     = script.Orig("GUE", "GUERRILLA CELL", "GAR", "GARRISON", "PAT", "PATROL", "GUERRILLAS", "TROOPS")
)

// Lines is every script block, for the provenance test.
var Lines = []script.Ls{
	lineIntro, lineRise, lineCapital, lineCrushed, lineEndured, lineHides, lineFound,
	lineRecruited, lineNoRecruit, lineNoHide, lineSupport, lineShelter, lineInformer, lineCurfew, lineSweep,
	lineAirdrop, lineReinforce, lineQuiet, lineMonsoon, verbHelp, title, regionNames, unitNames,
}

var (
	cell     = &sim.UnitType{Name: unitNames[0].Text, Long: unitNames[1].Text, Attack: 3, Defence: 2, Move: 2, Range: 1, Category: unitNames[6].Text}
	garrison = &sim.UnitType{Name: unitNames[2].Text, Long: unitNames[3].Text, Attack: 3, Defence: 5, Move: 1, Range: 1, Category: unitNames[7].Text}
	patrol   = &sim.UnitType{Name: unitNames[4].Text, Long: unitNames[5].Text, Attack: 4, Defence: 3, Move: 2, Range: 1, Category: unitNames[7].Text}
)

const (
	capital    = 5
	turns      = 12
	winSupport = 10
	recruitFee = 3
)

func fill(text string, args ...string) string {
	for _, a := range args {
		text = strings.Replace(text, "#", a, 1)
	}
	return text
}

// Hide is the order to go to ground.
var Hide = &sim.Verb{
	Name: "HIDE", Help: verbHelp[0].Text, Phase: 2,
	Check: func(_ *sim.State, u *sim.Unit, _ int) string {
		if u.Hidden {
			return lineNoHide[0].Text
		}
		return ""
	},
	Apply: func(s *sim.State, u *sim.Unit, _ int) {
		u.Hidden = true
		s.Say(fill(lineHides[0].Text, s.Label(u)))
	},
}

// Recruit spends support on a new hidden cell in the unit's region. Recruits already
// ordered this turn count against the support.
var Recruit = &sim.Verb{
	Name: "RECRUIT", Help: verbHelp[1].Text, Phase: 2,
	Check: func(s *sim.State, u *sim.Unit, _ int) string {
		queued := 0
		for _, o := range s.Pending {
			if o.Verb.Name == "RECRUIT" {
				queued++
			}
		}
		if s.Vars["support"] < recruitFee*(queued+1) || s.Control[u.Region] == sim.WOPR {
			return lineNoRecruit[0].Text
		}
		return ""
	},
	Apply: func(s *sim.State, u *sim.Unit, _ int) {
		s.Vars["support"] -= recruitFee
		n := s.AddUnit(sim.Player, cell, u.Region)
		n.Hidden = true
		s.Say(fill(lineRecruited[0].Text, s.Regions[u.Region].Name))
	},
}

// Scenario is the scenario data and hooks.
func Scenario() *sim.Scenario {
	return &sim.Scenario{
		Title:     title[0].Text,
		Intro:     lineIntro,
		Regions:   regions(sim.Rough, sim.Rough, sim.Rough, sim.Open, sim.City, sim.City),
		TurnLimit: turns,
		Verbs:     []*sim.Verb{Hide, Recruit},
		Setup: func(s *sim.State) {
			for _, r := range []int{0, 1, 1} {
				s.AddUnit(sim.Player, cell, r).Hidden = true
			}
			s.AddUnit(sim.WOPR, garrison, capital)
			s.AddUnit(sim.WOPR, garrison, 4)
			s.AddUnit(sim.WOPR, patrol, 3)
			s.AddUnit(sim.WOPR, patrol, 3)
			s.Vars["support"] = 3
			s.Control[capital], s.Control[4] = sim.WOPR, sim.WOPR
		},
		Events: []sim.Event{
			{Text: lineShelter[0].Text, Apply: func(s *sim.State) {
				for _, u := range s.Living(sim.Player) {
					u.Hidden = true
				}
			}},
			{Text: lineInformer[0].Text, Apply: func(s *sim.State) {
				var hidden []*sim.Unit
				for _, u := range s.Living(sim.Player) {
					if u.Hidden {
						hidden = append(hidden, u)
					}
				}
				if len(hidden) > 0 {
					u := hidden[s.Rand().IntN(len(hidden))]
					u.Hidden = false
					s.Say(fill(lineInformer[1].Text, s.Label(u)))
				}
			}},
			{Text: lineCurfew[0].Text, Apply: func(s *sim.State) {
				if s.Vars["support"] > 0 {
					s.Vars["support"]--
					s.Say(lineCurfew[1].Text)
				}
			}},
			{Text: lineSweep[0].Text, Apply: func(s *sim.State) { s.Vars["support"]++ }},
			{Text: lineAirdrop[0].Text, Apply: func(s *sim.State) {
				for _, u := range s.Living(sim.Player) {
					if u.Steps == 1 {
						u.Steps = 2
						s.Say(fill(lineAirdrop[1].Text, s.Label(u)))
						return
					}
				}
			}},
			{Text: lineReinforce[0].Text, Apply: func(s *sim.State) { s.AddUnit(sim.WOPR, patrol, capital) }},
			{Text: lineQuiet[0].Text},
			{Text: lineMonsoon[0].Text, Apply: func(s *sim.State) { s.Vars["monsoon"] = 1 }},
		},
		Upkeep: upkeep,
		// An ambush from hiding hits twice as hard.
		AttackMod: func(_ *sim.State, u *sim.Unit, a int) int {
			if u.Side == sim.Player && u.Hidden {
				return 2 * a
			}
			return a
		},
		Status: func(s *sim.State) string { return fill(lineSupport[0].Text, fmt.Sprint(s.Vars["support"])) },
		AI:     ai,
		Victory: func(s *sim.State, final bool) (proto.Outcome, string, bool) {
			switch {
			case s.Vars["support"] >= winSupport:
				return proto.Win, lineRise[0].Text, true
			case s.Control[capital] == sim.Player:
				return proto.Win, lineCapital[0].Text, true
			case len(s.Living(sim.Player)) == 0:
				return proto.Loss, lineCrushed[0].Text, true
			case final: // surviving is the insurgent's victory
				return proto.Win, lineEndured[0].Text, true
			}
			return 0, "", false
		},
	}
}

func regions(terrain ...sim.Terrain) []sim.Region {
	out := make([]sim.Region, len(terrain))
	for i, t := range terrain {
		out[i] = sim.Region{Name: regionNames[i].Text, Terrain: t}
	}
	return out
}

// New returns a game.
func New() games.Game { return sim.NewGame(Scenario()) }

// upkeep: support moves with the turn's losses (WOPR's steps lost win it, cells' cost it),
// a cell that attacked is exposed, and patrols find hidden cells near them.
func upkeep(s *sim.State) {
	lostW, lostP := total(s.Losses[sim.WOPR]), total(s.Losses[sim.Player])
	s.Vars["support"] = max(s.Vars["support"]+(lostW-s.Vars["lostW"])-(lostP-s.Vars["lostP"]), 0)
	s.Vars["lostW"], s.Vars["lostP"] = lostW, lostP

	for _, u := range s.Living(sim.Player) {
		if u.Attacked { // a raid gives the cell away
			u.Hidden = false
		}
	}
	if s.Vars["monsoon"] > 0 {
		s.Vars["monsoon"] = 0
		return
	}
	for _, u := range s.Living(sim.Player) {
		if !u.Hidden {
			continue
		}
		for _, p := range s.Living(sim.WOPR) {
			if p.Type != patrol {
				continue
			}
			d := p.Region - u.Region
			// Half the time in the same region, one time in six next door.
			if d == 0 && s.Rand().IntN(6) >= 3 || (d == 1 || d == -1) && s.Rand().IntN(6) >= 5 {
				u.Hidden = false
				s.Say(fill(lineFound[0].Text, s.Label(u)))
				break
			}
		}
	}
}

func total(m map[string]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}

// ai: patrols attack visible cells at even odds or better and otherwise hunt the nearest
// visible cell, or sweep toward the border; garrisons hold their towns and strike only
// at cells next door.
func ai(s *sim.State) []sim.Order {
	var orders []sim.Order
	cells := s.Living(sim.Player)
	for _, u := range s.Living(sim.WOPR) {
		at, col := -1, -1
		for r := range s.Regions {
			if !s.CanAttack(u, r) {
				continue
			}
			if c := sim.Column(s.AttackOf(u), s.DefenceOf(toughest(s, r)), s.Regions[r].Terrain); c > col {
				at, col = r, c
			}
		}
		if at >= 0 && col >= 1 { // even odds or better
			orders = append(orders, sim.Order{Unit: u, Verb: sim.Attack, Target: at})
			continue
		}
		if u.Type == garrison {
			orders = append(orders, sim.Order{Unit: u, Verb: sim.Hold, Target: -1})
			continue
		}
		goal := sim.Nearest(len(s.Regions), u.Region, func(r int) bool {
			for _, c := range cells {
				if c.Region == r && !c.Hidden {
					return true
				}
			}
			return false
		})
		if goal < 0 {
			goal = 0 // sweep toward the border camps
		}
		if to := s.Toward(u, goal); to >= 0 && to != u.Region {
			orders = append(orders, sim.Order{Unit: u, Verb: sim.Move, Target: to})
		} else {
			orders = append(orders, sim.Order{Unit: u, Verb: sim.Hold, Target: -1})
		}
	}
	return orders
}

// toughest is the visible cell in r with the best defence.
func toughest(s *sim.State, r int) *sim.Unit {
	var best *sim.Unit
	for _, c := range s.In(r, sim.Player) {
		if !c.Hidden && (best == nil || s.DefenceOf(c) > s.DefenceOf(best)) {
			best = c
		}
	}
	return best
}
