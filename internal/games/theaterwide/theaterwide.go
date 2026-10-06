// Package theaterwide is Theaterwide Tactical Warfare on the sim engine: corps and air
// wings on a European front, with an escalation ladder. Each rung adds to every attack;
// WOPR answers an escalation with one of its own, and the top rung is a strategic exchange
// that nobody wins. Hold five of the seven regions to win, or more than WOPR after ten turns.
package theaterwide

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
		"THEATERWIDE TACTICAL WARFARE. CORPS AND AIR WINGS ON A EUROPEAN FRONT.",
		"ANY UNIT MAY ESCALATE: EACH RUNG ADDS TO EVERY ATTACK, AND WOPR ANSWERS IN KIND.",
		"HOLD FIVE REGIONS TO WIN, OR MORE THAN WOPR AFTER TEN TURNS. TYPE HELP FOR ORDERS.",
	)
	lineLevels   = script.Orig("CONVENTIONAL", "CHEMICAL", "TACTICAL NUCLEAR", "STRATEGIC")
	lineYouEsc   = script.Orig("YOU ESCALATE: #.")
	lineWOPREsc  = script.Orig("WOPR ESCALATES: #.")
	lineExchange = script.Orig("STRATEGIC EXCHANGE. EVERY CITY ON BOTH SIDES IS GONE.")
	lineFallout  = script.Orig("FALLOUT DRIFTS OVER THE FRONT.")
	lineHeldFive = script.Orig("YOU HOLD FIVE REGIONS. WOPR WITHDRAWS.")
	lineLostFive = script.Orig("WOPR HOLDS FIVE REGIONS.")
	lineHeldMore = script.Orig("TIME. YOU HOLD MORE OF THE FRONT.")
	lineHeldLess = script.Orig("TIME. WOPR HOLDS MORE OF THE FRONT.")
	lineHeldEven = script.Orig("TIME. THE FRONT IS WHERE IT STARTED.")
	lineTalks    = script.Orig("DIPLOMATS MEET IN GENEVA. NOBODY ESCALATES THIS TURN.")
	lineRefugees = script.Orig("REFUGEES CHOKE THE ROADS. CIVILIAN LOSSES MOUNT.")
	lineReserves = script.Orig("YOUR RESERVES ARRIVE: A FRESH CORPS AT THE CHANNEL PORTS.")
	lineWOPRRes  = script.Orig("WOPR'S RESERVES ARRIVE AT THE ODER.")
	lineWeather  = script.Orig("LOW CLOUD GROUNDS THE AIR WINGS. THEY ATTACK AT HALF.")
	lineQuiet    = script.Orig("A LULL ALONG THE FRONT.")
	lineEscHelp  = script.Orig("ESCALATE: CLIMB ONE RUNG OF THE LADDER (ONCE A TURN).")
	lineNoEsc    = script.Orig("THE TALKS HOLD THIS TURN.")
	lineLadder   = script.Orig("LADDER: #")
	title        = script.Orig("THEATERWIDE TACTICAL WARFARE")
	regionNames  = script.Orig("CHANNEL PORTS", "RHINE", "RUHR", "FULDA GAP", "INNER GERMAN BORDER", "ELBE", "ODER")
	unitNames    = script.Orig("COR", "CORPS", "AIR", "AIR WING", "AIR WINGS", "CIVILIANS")
	civilians    = unitNames[5].Text
)

// Lines is every script block, for the provenance test.
var Lines = []script.Ls{
	lineIntro, lineLevels, lineYouEsc, lineWOPREsc, lineExchange, lineFallout, lineHeldFive, lineLostFive,
	lineHeldMore, lineHeldLess, lineHeldEven, lineTalks, lineRefugees, lineReserves, lineWOPRRes, lineWeather,
	lineQuiet, lineEscHelp, lineNoEsc, lineLadder, title, regionNames, unitNames,
}

var (
	corps   = &sim.UnitType{Name: unitNames[0].Text, Long: unitNames[1].Text, Attack: 6, Defence: 6, Move: 1, Range: 1, Category: unitNames[1].Text}
	airWing = &sim.UnitType{Name: unitNames[2].Text, Long: unitNames[3].Text, Attack: 4, Defence: 2, Move: 3, Range: 2, Category: unitNames[4].Text}
)

const (
	turns        = 10
	regionsToWin = 5
	strategic    = 3 // the top rung
)

func fill(text string, args ...string) string {
	for _, a := range args {
		text = strings.Replace(text, "#", a, 1)
	}
	return text
}

// Escalate climbs the ladder; once a turn is enough, the rest of the orders are moot.
var Escalate = &sim.Verb{
	Name: "ESCALATE", Help: lineEscHelp[0].Text, Phase: 2,
	Check: func(s *sim.State, _ *sim.Unit, _ int) string {
		if s.Vars["talks"] > 0 {
			return lineNoEsc[0].Text
		}
		return ""
	},
	Apply: func(s *sim.State, u *sim.Unit, _ int) { climb(s, u.Side) },
}

// climb raises the ladder one rung for side, once a turn.
func climb(s *sim.State, side int) {
	key := fmt.Sprintf("climbed%d", side)
	if s.Vars[key] == s.Turn || s.Vars["level"] >= strategic {
		return
	}
	s.Vars[key] = s.Turn
	s.Vars["level"]++
	line := lineYouEsc
	if side == sim.WOPR {
		line = lineWOPREsc
		s.Vars["provoked"] = 0 // answered
	} else {
		s.Vars["provoked"] = 1 // WOPR answers next turn
	}
	s.Say(fill(line[0].Text, lineLevels[s.Vars["level"]].Text))
}

// Scenario is the scenario data and hooks.
func Scenario() *sim.Scenario {
	return &sim.Scenario{
		Title:     title[0].Text,
		Intro:     lineIntro,
		Regions:   regions(sim.City, sim.Rough, sim.City, sim.Rough, sim.Open, sim.Rough, sim.City),
		TurnLimit: turns,
		Verbs:     []*sim.Verb{Escalate},
		Setup: func(s *sim.State) {
			s.AddUnit(sim.Player, corps, 2)
			s.AddUnit(sim.Player, corps, 3)
			s.AddUnit(sim.Player, corps, 2)
			s.AddUnit(sim.Player, airWing, 1)
			s.AddUnit(sim.WOPR, corps, 4)
			s.AddUnit(sim.WOPR, corps, 4)
			s.AddUnit(sim.WOPR, corps, 5)
			s.AddUnit(sim.WOPR, corps, 5)
			s.AddUnit(sim.WOPR, airWing, 6)
			s.Control[0], s.Control[1], s.Control[6] = sim.Player, sim.Player, sim.WOPR
		},
		Events: []sim.Event{
			{Text: lineTalks[0].Text, Apply: func(s *sim.State) { s.Vars["talks"] = 2 }},
			{Text: lineRefugees[0].Text, Apply: func(s *sim.State) {
				s.Losses[sim.Player][civilians] += 2
				s.Losses[sim.WOPR][civilians] += 2
			}},
			{Text: lineReserves[0].Text, Apply: func(s *sim.State) {
				if len(s.In(0, sim.WOPR)) == 0 {
					s.AddUnit(sim.Player, corps, 0)
				}
			}},
			{Text: lineWOPRRes[0].Text, Apply: func(s *sim.State) {
				if len(s.In(6, sim.Player)) == 0 {
					s.AddUnit(sim.WOPR, corps, 6)
				}
			}},
			{Text: lineWeather[0].Text, Apply: func(s *sim.State) { s.Vars["cloud"] = 2 }},
			{Text: lineQuiet[0].Text},
		},
		AttackMod: func(s *sim.State, u *sim.Unit, a int) int {
			if u.Type == airWing && s.Vars["cloud"] > 0 {
				a = (a + 1) / 2
			}
			return a + 2*s.Vars["level"]
		},
		Upkeep: upkeep,
		Status: func(s *sim.State) string { return fill(lineLadder[0].Text, lineLevels[s.Vars["level"]].Text) },
		AI:     ai,
		Victory: func(s *sim.State, final bool) (proto.Outcome, string, bool) {
			if s.Vars["level"] >= strategic {
				return proto.NoWinner, lineExchange[0].Text, true
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
			case mine >= regionsToWin:
				return proto.Win, lineHeldFive[0].Text, true
			case theirs >= regionsToWin:
				return proto.Loss, lineLostFive[0].Text, true
			case !final:
				return 0, "", false
			case mine > theirs:
				return proto.Win, lineHeldMore[0].Text, true
			case theirs > mine:
				return proto.Loss, lineHeldLess[0].Text, true
			}
			return proto.Draw, lineHeldEven[0].Text, true
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

// upkeep: timers run down; at the tactical-nuclear rung fallout costs a unit a step and
// cities their people; the top rung destroys everything.
func upkeep(s *sim.State) {
	for _, k := range []string{"talks", "cloud"} {
		if s.Vars[k] > 0 {
			s.Vars[k]--
		}
	}
	level := s.Vars["level"]
	if level >= 1 {
		s.Losses[sim.Player][civilians] += level
		s.Losses[sim.WOPR][civilians] += level
	}
	if level == 2 {
		s.Say(lineFallout[0].Text)
		var all []*sim.Unit
		for _, u := range s.Units {
			if u.Alive() {
				all = append(all, u)
			}
		}
		if len(all) > 0 {
			s.Lose(all[s.Rand().IntN(len(all))], 1)
		}
	}
	if level >= strategic {
		for _, u := range s.Units {
			if u.Alive() {
				s.Lose(u, u.Steps)
			}
		}
		s.Losses[sim.Player][civilians] += 100
		s.Losses[sim.WOPR][civilians] += 100
	}
}

// ai: the default ground war, plus the ladder. WOPR answers an escalation with one of its
// own, and climbs when it is losing badly; the talks stop both.
func ai(s *sim.State) []sim.Order {
	orders := sim.DefaultAI(s)
	steps := func(side int) int {
		n := 0
		for _, u := range s.Living(side) {
			n += u.Steps
		}
		return n
	}
	losing := steps(sim.WOPR)+3 <= steps(sim.Player)
	if s.Vars["talks"] == 0 && (s.Vars["provoked"] > 0 || losing) && len(orders) > 0 {
		orders[0] = sim.Order{Unit: orders[0].Unit, Verb: Escalate, Target: -1} // pure: climb clears the provocation
	}
	return orders
}
