// Package fightercombat is Fighter Combat: a turn-based dogfight of energy and aspect
// against WOPR (docs/PLAN.md §6.3, a bespoke game on the sim engine's combat-results and
// kill-ratio tables). Each turn both pilots pick a manoeuvre at once: PRESS closes the
// range, EXTEND opens it, CLIMB builds energy, BREAK turns hard (to shake a pursuer and
// spoil a shot), FIRE shoots (missiles at medium or long range, guns close in, and only
// from behind). Two hits bring an aircraft down.
package fightercombat

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/prompt"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
	"github.com/GhostofGoes/WOPR/internal/sim"
)

// Script text, all original. A # is filled in when the line is shown.
var (
	lineIntro = script.Orig(
		"FIGHTER COMBAT, ONE ON ONE. EACH TURN YOU AND WOPR PICK A MANOEUVRE AT ONCE.",
		"PRESS CLOSES IN. EXTEND OPENS THE RANGE. CLIMB BUILDS ENERGY.",
		"BREAK TURNS HARD, TO SHAKE A PURSUER OR SPOIL A SHOT.",
		"FIRE: MISSILES AT RANGE, GUNS CLOSE IN FROM BEHIND. TWO HITS AND YOU ARE DOWN.",
		"DISENGAGE AT LONG RANGE TO END IT EVEN. THE FUEL LASTS 15 TURNS.",
	)
	promptMove   = script.Orig("MANOEUVRE: ")
	lineHelp     = script.Orig("PRESS, EXTEND, CLIMB, BREAK, FIRE OR DISENGAGE.")
	lineNoShot   = script.Orig("NO SHOT: # ")
	lineWhyShot  = script.Orig("GUNS NEED YOU CLOSE IN AND ON ITS TAIL.", "NOT FROM THERE: GET ON ITS TAIL OR MEET IT HEAD-ON.", "NO MISSILES LEFT; CLOSE IN FOR GUNS.")
	lineNoLeave  = script.Orig("ONLY AT LONG RANGE.")
	lineMoves    = script.Orig("YOU PRESS IN.", "YOU EXTEND.", "YOU CLIMB.", "YOU BREAK HARD.")
	lineWOPRMove = script.Orig("WOPR PRESSES IN.", "WOPR EXTENDS.", "WOPR CLIMBS.", "WOPR BREAKS HARD.")
	lineShot     = script.Orig("YOU FIRE #: #.", "WOPR FIRES #: #.")
	lineWeapons  = script.Orig("A MISSILE", "GUNS")
	lineShotEnds = script.Orig("A HIT", "A NEAR MISS", "A MISS")
	lineDown     = script.Orig("WOPR'S AIRCRAFT GOES DOWN.", "YOUR AIRCRAFT GOES DOWN.", "BOTH AIRCRAFT GO DOWN.")
	lineBingo    = script.Orig("BINGO FUEL. BOTH OF YOU TURN FOR HOME.")
	lineLeave    = script.Orig("YOU DISENGAGE. WOPR LETS YOU GO.")
	lineStatus   = script.Orig("TURN # OF 15. RANGE #.", "YOU:  #. ENERGY #, MISSILES #.", "WOPR: #ENERGY #, MISSILES #.")
	lineRanges   = script.Orig("", "CLOSE", "MEDIUM", "LONG")
	linePos      = script.Orig("ON THE DEFENSIVE", "NEUTRAL", "ON ITS TAIL")
	lineDamage   = script.Orig(", DAMAGED", "DAMAGED. ")
	categories   = script.Orig("AIRCRAFT", "MISSILES FIRED")
)

// Lines is every script block, for the provenance test.
var Lines = []script.Ls{
	lineIntro, promptMove, lineHelp, lineNoShot, lineWhyShot, lineNoLeave, lineMoves, lineWOPRMove, lineShot,
	lineWeapons, lineShotEnds, lineDown, lineBingo, lineLeave, lineStatus, lineRanges, linePos, lineDamage,
	categories,
}

// Manoeuvres.
const (
	press = iota
	extend
	climb
	breakTurn
	fire
)

const (
	turns      = 15
	maxEnergy  = 6
	rangeClose = 1
	rangeLong  = 3
)

func fill(text string, args ...string) string {
	for _, a := range args {
		text = strings.Replace(text, "#", a, 1)
	}
	return text
}

func say(lines ...string) proto.Output { return proto.Say{Lines: lines, Pace: proto.PaceSpeech} }

// Game is the dogfight as a proto.Program.
type Game struct {
	rng      *rand.Rand
	turn     int
	dist     int    // range: 1 close, 2 medium, 3 long
	pos      int    // the player's aspect: -1 defensive, 0 neutral, 1 on WOPR's tail
	energy   [2]int // player, WOPR
	missiles [2]int
	hits     [2]int
	losses   [2]map[string]int
	over     bool
}

// New returns a game.
func New() games.Game { return &Game{} }

// Start implements proto.Program.
func (g *Game) Start(env proto.Env) []proto.Output {
	g.rng = proto.NewRand(env.Seed, 0)
	g.turn, g.dist = 1, rangeLong
	g.energy = [2]int{3, 3}
	g.missiles = [2]int{2, 2}
	g.losses = [2]map[string]int{{}, {}}
	return []proto.Output{say(lineIntro.Texts()...), say(g.status()...), proto.Prompt{Text: promptMove[0].Text}}
}

// View implements proto.Program; the dogfight is console text.
func (g *Game) View(*proto.Canvas) {}

func (g *Game) status() []string {
	you := linePos[g.pos+1].Text
	if g.hits[0] > 0 {
		you += lineDamage[0].Text
	}
	hurt := ""
	if g.hits[1] > 0 {
		hurt = lineDamage[1].Text
	}
	return []string{
		fill(lineStatus[0].Text, fmt.Sprint(g.turn), lineRanges[g.dist].Text),
		fill(lineStatus[1].Text, you, fmt.Sprint(g.energy[0]), fmt.Sprint(g.missiles[0])),
		fill(lineStatus[2].Text, hurt, fmt.Sprint(g.energy[1]), fmt.Sprint(g.missiles[1])),
	}
}

// Handle implements proto.Program.
func (g *Game) Handle(ev proto.Event) []proto.Output {
	line, ok := ev.(proto.LineEvent)
	if !ok || g.over {
		return nil
	}
	again := func(text string) []proto.Output {
		return []proto.Output{say(text), proto.Prompt{Text: promptMove[0].Text}}
	}
	words := strings.Fields(prompt.Normalize(line.Text))
	if len(words) == 0 {
		return again(lineHelp[0].Text)
	}
	var m int
	switch words[0] {
	case "P", "PRESS", "CLOSE":
		m = press
	case "E", "EXTEND", "RUN":
		m = extend
	case "C", "CLIMB", "ZOOM":
		m = climb
	case "B", "BREAK", "TURN", "EVADE":
		m = breakTurn
	case "F", "FIRE", "SHOOT", "GUNS", "MISSILE":
		m = fire
	case "D", "DISENGAGE", "LEAVE", "RTB":
		if g.dist != rangeLong {
			return again(lineNoLeave[0].Text)
		}
		g.over = true
		return []proto.Output{say(lineLeave[0].Text), g.done(proto.Draw)}
	default:
		return again(lineHelp[0].Text)
	}
	if m == fire {
		if why := g.cannotFire(0); why >= 0 {
			return again(fill(lineNoShot[0].Text, lineWhyShot[why].Text))
		}
	}
	return g.resolve(m, g.choose())
}

// aspect is side's view of the fight: 1 behind the other, 0 neutral, -1 defensive.
func (g *Game) aspect(side int) int {
	if side == 0 {
		return g.pos
	}
	return -g.pos
}

// cannotFire says why side has no shot (an index into lineWhyShot), or -1 when it has one.
func (g *Game) cannotFire(side int) int {
	switch {
	case g.dist == rangeClose && g.aspect(side) < 1:
		return 0
	case g.dist == rangeClose:
		return -1
	case g.aspect(side) < 0:
		return 1
	case g.missiles[side] == 0:
		return 2
	}
	return -1
}

// choose is WOPR's manoeuvre: shoot when it has a fair shot, shake a pursuer, rebuild
// energy when low, close in when it holds the edge; a little unpredictability from the
// stream.
func (g *Game) choose() int {
	if g.rng.IntN(5) == 0 {
		return g.rng.IntN(4) // press, extend, climb or break: never an impossible shot
	}
	switch {
	case g.cannotFire(1) < 0 && (g.aspect(1) == 1 || g.dist > rangeClose):
		return fire
	case g.aspect(1) < 0:
		return breakTurn
	case g.energy[1] <= 1:
		return climb
	case g.dist > rangeClose && g.aspect(1) >= 0:
		return press
	}
	return climb
}

// resolve plays one turn: shots first (from where everyone stood), then the manoeuvres.
func (g *Game) resolve(mine, theirs int) []proto.Output {
	var lines []string
	moves := [2]int{mine, theirs}
	said := [2]script.Ls{lineMoves, lineWOPRMove}
	for side := range 2 {
		if moves[side] == fire {
			lines = append(lines, g.shoot(side, moves[1-side] == breakTurn))
		} else {
			lines = append(lines, said[side][moves[side]].Text)
		}
	}
	g.manoeuvre(moves)
	outs := []proto.Output{say(lines...)}
	switch {
	case g.hits[0] >= 2 && g.hits[1] >= 2:
		g.over = true
		return append(outs, say(lineDown[2].Text), g.done(proto.NoWinner))
	case g.hits[1] >= 2:
		g.over = true
		return append(outs, say(lineDown[0].Text), g.done(proto.Win))
	case g.hits[0] >= 2:
		g.over = true
		return append(outs, say(lineDown[1].Text), g.done(proto.Loss))
	case g.turn >= turns:
		g.over = true
		return append(outs, say(lineBingo[0].Text), g.done(proto.Draw))
	}
	g.turn++
	return append(outs, say(g.status()...), proto.Prompt{Text: promptMove[0].Text})
}

// shoot resolves side's shot on the combat-results table: a missile at range, guns close in;
// being behind helps, a hard break and energy spoil it.
func (g *Game) shoot(side int, evading bool) string {
	weapon, attack := 0, 6
	if g.dist == rangeClose {
		weapon, attack = 1, 8
	} else {
		g.missiles[side]--
		g.losses[side][categories[1].Text]++
	}
	if g.aspect(side) == 1 {
		attack += 2
	}
	defence := 3
	if evading {
		defence += 3
	}
	if g.energy[1-side] >= 4 {
		defence++
	}
	end := 2 // a miss
	switch sim.CRT(attack, defence, sim.Open, g.rng.IntN(6)+1) {
	case sim.DefenderLoses, sim.Exchange:
		end = 0
		g.hits[1-side]++
		if g.hits[1-side] == 2 {
			g.losses[1-side][categories[0].Text]++
		}
	case sim.DefenderRetreats:
		end = 1
		g.energy[1-side] = max(g.energy[1-side]-1, 0)
	}
	return fill(lineShot[side].Text, lineWeapons[weapon].Text, lineShotEnds[end].Text)
}

// manoeuvre applies both manoeuvres to the range, the energy and the aspect.
func (g *Game) manoeuvre(m [2]int) {
	cost := [...]int{press: -1, extend: 1, climb: 2, breakTurn: -2, fire: 0}
	for side := range 2 {
		g.energy[side] = min(max(g.energy[side]+cost[m[side]], 0), maxEnergy)
	}
	closing := 0
	for side := range 2 {
		switch m[side] {
		case press:
			closing--
		case extend:
			closing++
		}
	}
	g.dist = min(max(g.dist+closing, rangeClose), rangeLong)

	// The aspect: a hard break may shake a pursuer; pressing in with more energy, or
	// climbing well above the other, takes the edge.
	switch {
	case m[0] == breakTurn && g.pos < 0 && g.rng.IntN(6) < 2+g.energy[0]/2:
		g.pos = 0
	case m[1] == breakTurn && g.pos > 0 && g.rng.IntN(6) < 2+g.energy[1]/2:
		g.pos = 0
	case m[0] == press && m[1] != breakTurn && g.energy[0] > g.energy[1]:
		g.pos = min(g.pos+1, 1)
	case m[1] == press && m[0] != breakTurn && g.energy[1] > g.energy[0]:
		g.pos = max(g.pos-1, -1)
	case m[0] == climb && g.energy[0] >= g.energy[1]+3:
		g.pos = min(g.pos+1, 1)
	case m[1] == climb && g.energy[1] >= g.energy[0]+3:
		g.pos = max(g.pos-1, -1)
	}
}

func (g *Game) done(o proto.Outcome) proto.Output {
	return proto.Done{Result: proto.Result{Outcome: o, Lines: sim.RatioTable(g.losses)}}
}
