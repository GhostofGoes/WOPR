// Package biotoxic is Theaterwide Biotoxic and Chemical Warfare on the sim engine.
// Agents released on the front spread with the wind from region to region; units and
// civilians in contaminated ground suffer, decontamination slows it but never stops it,
// and whatever anyone does, nobody wins (docs/PLAN.md §6.1): the game ends WINNER: NONE.
package biotoxic

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
		"THEATERWIDE BIOTOXIC AND CHEMICAL WARFARE.",
		"CHEMICAL TROOPS RELEASE AN AGENT ON A REGION (TYPE RELEASE IN FULL);",
		"IT SPREADS WITH THE WIND. DECON CLEANS A LITTLE.",
		"UNITS AND CIVILIANS IN CONTAMINATED GROUND SUFFER. EIGHT TURNS.",
		"TYPE HELP FOR ORDERS.",
	)
	lineReleased   = script.Orig("# RELEASED ON #.")
	lineDecon      = script.Orig("DECONTAMINATION AT #.")
	lineSpread     = script.Orig("THE CONTAMINATION SPREADS.")
	lineSick       = script.Orig("# SICKENS IN THE CONTAMINATED GROUND.")
	lineEverywhere = script.Orig("THE CONTAMINATION REACHES EVERY REGION.")
	lineNoneLeft   = script.Orig("NO ONE IS LEFT TO FIGHT.")
	lineTime       = script.Orig("TIME. THE GROUND WILL NOT BE SAFE FOR A GENERATION.")
	lineWindEast   = script.Orig("THE WIND TURNS EAST.")
	lineWindWest   = script.Orig("THE WIND TURNS WEST.")
	lineRain       = script.Orig("RAIN WASHES SOME OF IT INTO THE RIVERS.")
	lineStock      = script.Orig("A STOCKPILE IS HIT. THE AGENT LEAKS.")
	lineQuiet      = script.Orig("NOTHING MOVES BUT THE WIND.")
	lineWOPRs      = script.Orig("WOPR'S ")
	lineHelp       = script.Orig("RELEASE <REGION>: CHEMICAL TROOPS ONLY, UP TO TWO REGIONS AWAY.", "DECON: CLEAN THE UNIT'S OWN REGION A LITTLE.")
	lineNoRange    = script.Orig("THAT REGION IS OUT OF RANGE.", "ONLY CHEMICAL TROOPS CARRY AN AGENT.")
	lineClean      = script.Orig("THERE IS NOTHING TO CLEAN HERE.")
	lineLevel      = script.Orig("CONTAMINATED: # OF # REGIONS")
	levelMarks     = script.Orig("", "~", "~~", "~~~")
	title          = script.Orig("BIOTOXIC AND CHEMICAL WARFARE")
	regionNames    = script.Orig("WESTERN CITIES", "FARMLAND", "RIVER DELTA", "INDUSTRIAL BELT", "FOREST", "EASTERN CITIES")
	unitNames      = script.Orig("CHM", "CHEMICAL BATTALION", "INF", "INFANTRY", "CHEMICAL TROOPS", "CIVILIANS")
	civilians      = unitNames[5].Text
)

// Lines is every script block, for the provenance test.
var Lines = []script.Ls{
	lineIntro, lineReleased, lineDecon, lineSpread, lineSick, lineEverywhere, lineNoneLeft, lineTime, lineWindEast,
	lineWindWest, lineRain, lineStock, lineQuiet, lineWOPRs, lineHelp, lineNoRange, lineClean, lineLevel, levelMarks, title,
	regionNames, unitNames,
}

var (
	chemical = &sim.UnitType{Name: unitNames[0].Text, Long: unitNames[1].Text, Attack: 2, Defence: 3, Move: 1, Range: 1, Category: unitNames[4].Text}
	infantry = &sim.UnitType{Name: unitNames[2].Text, Long: unitNames[3].Text, Attack: 4, Defence: 4, Move: 1, Range: 1, Category: unitNames[3].Text}
)

const (
	turns    = 8
	maxLevel = 3
)

func fill(text string, args ...string) string {
	for _, a := range args {
		text = strings.Replace(text, "#", a, 1)
	}
	return text
}

func level(s *sim.State, r int) int { return s.Vars[fmt.Sprintf("c%d", r)] }

func setLevel(s *sim.State, r, v int) { s.Vars[fmt.Sprintf("c%d", r)] = min(max(v, 0), maxLevel) }

// Release puts an agent on a region up to two away; only chemical troops carry one.
var Release = &sim.Verb{
	Name: "RELEASE", Help: lineHelp[0].Text, Target: true, Phase: 1, Drastic: true,
	Check: func(s *sim.State, u *sim.Unit, r int) string {
		if u.Type != chemical {
			return lineNoRange[1].Text
		}
		if r < 0 || r >= len(s.Regions) || abs(r-u.Region) > 2 {
			return lineNoRange[0].Text
		}
		return ""
	},
	Apply: func(s *sim.State, u *sim.Unit, r int) {
		setLevel(s, r, level(s, r)+2)
		who := u.Name()
		if u.Side == sim.WOPR {
			who = lineWOPRs[0].Text + who
		}
		s.Say(fill(lineReleased[0].Text, who, s.Regions[r].Name))
	},
}

// Decon cleans the unit's own region a little.
var Decon = &sim.Verb{
	Name: "DECON", Help: lineHelp[1].Text, Phase: 2,
	Check: func(s *sim.State, u *sim.Unit, _ int) string {
		if level(s, u.Region) == 0 {
			return lineClean[0].Text
		}
		return ""
	},
	Apply: func(s *sim.State, u *sim.Unit, _ int) {
		setLevel(s, u.Region, level(s, u.Region)-1)
		s.Say(fill(lineDecon[0].Text, s.Regions[u.Region].Name))
	},
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Scenario is the scenario data and hooks.
func Scenario() *sim.Scenario {
	return &sim.Scenario{
		Title:     title[0].Text,
		Intro:     lineIntro,
		Regions:   regions(sim.City, sim.Open, sim.Open, sim.City, sim.Rough, sim.City),
		TurnLimit: turns,
		Verbs:     []*sim.Verb{Release, Decon},
		Setup: func(s *sim.State) {
			s.AddUnit(sim.Player, infantry, 1)
			s.AddUnit(sim.Player, infantry, 2)
			s.AddUnit(sim.Player, chemical, 1)
			s.AddUnit(sim.WOPR, infantry, 4)
			s.AddUnit(sim.WOPR, infantry, 3)
			s.AddUnit(sim.WOPR, chemical, 4)
			s.Control[0], s.Control[5] = sim.Player, sim.WOPR
			s.Vars["wind"] = 1 // blowing east, toward WOPR
		},
		Events: []sim.Event{
			{Text: lineWindEast[0].Text, Apply: func(s *sim.State) { s.Vars["wind"] = 1 }},
			{Text: lineWindWest[0].Text, Apply: func(s *sim.State) { s.Vars["wind"] = -1 }},
			{Text: lineRain[0].Text, Apply: func(s *sim.State) {
				for r := range s.Regions {
					setLevel(s, r, level(s, r)-1)
				}
			}},
			{Text: lineStock[0].Text, Apply: func(s *sim.State) {
				r := s.Rand().IntN(len(s.Regions))
				setLevel(s, r, level(s, r)+2)
			}},
			{Text: lineQuiet[0].Text},
		},
		Upkeep: upkeep,
		Status: func(s *sim.State) string {
			n := 0
			for r := range s.Regions {
				if level(s, r) > 0 {
					n++
				}
			}
			return fill(lineLevel[0].Text, fmt.Sprint(n), fmt.Sprint(len(s.Regions)))
		},
		AI:         ai,
		RegionNote: Marks,
		// Nobody wins this one, however it ends.
		Victory: func(s *sim.State, final bool) (proto.Outcome, string, bool) {
			everywhere := true
			for r := range s.Regions {
				everywhere = everywhere && level(s, r) > 0
			}
			switch {
			case everywhere:
				return proto.NoWinner, lineEverywhere[0].Text, true
			case len(s.Living(sim.Player)) == 0 || len(s.Living(sim.WOPR)) == 0:
				return proto.NoWinner, lineNoneLeft[0].Text, true
			case final:
				return proto.NoWinner, lineTime[0].Text, true
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

// upkeep: contamination of 2 or more spreads one region downwind; units in it may lose a
// step, and civilians in it count against the side whose half of the front it is.
func upkeep(s *sim.State) {
	wind := s.Vars["wind"]
	next := make([]int, len(s.Regions))
	for r := range s.Regions {
		next[r] = level(s, r)
	}
	spread := false
	for r := range s.Regions {
		if level(s, r) >= 2 {
			if to := r + wind; to >= 0 && to < len(s.Regions) {
				next[to] = max(next[to], level(s, r)-1)
				spread = true
			}
		}
	}
	for r := range s.Regions {
		setLevel(s, r, next[r])
	}
	if spread {
		s.Say(lineSpread[0].Text)
	}
	for _, u := range s.Units {
		if u.Alive() && s.Rand().IntN(maxLevel+1) < level(s, u.Region) {
			who := u.Name()
			if u.Side == sim.WOPR {
				who = lineWOPRs[0].Text + who
			}
			s.Say(fill(lineSick[0].Text, who))
			s.Lose(u, 1)
		}
	}
	for r := range s.Regions {
		if c := level(s, r); c > 0 {
			side := sim.Player
			if r >= len(s.Regions)/2 {
				side = sim.WOPR
			}
			s.Losses[side][civilians] += c * c
		}
	}
}

// ai: WOPR answers a release with releases of its own toward the player's side, and
// decontaminates where it stands; its infantry fights as usual.
func ai(s *sim.State) []sim.Order {
	var orders []sim.Order
	poisoned := false
	for r := len(s.Regions) / 2; r < len(s.Regions); r++ {
		poisoned = poisoned || level(s, r) > 0
	}
	for _, o := range sim.DefaultAI(s) {
		u := o.Unit
		switch {
		case u.Type == chemical && poisoned:
			target := max(u.Region-2, 0)
			orders = append(orders, sim.Order{Unit: u, Verb: Release, Target: target})
		case u.Type == chemical && level(s, u.Region) > 0:
			orders = append(orders, sim.Order{Unit: u, Verb: Decon, Target: -1})
		default:
			orders = append(orders, o)
		}
	}
	return orders
}

// Marks shows a region's contamination on the map (~ to ~~~).
func Marks(s *sim.State, r int) string { return levelMarks[level(s, r)].Text }
