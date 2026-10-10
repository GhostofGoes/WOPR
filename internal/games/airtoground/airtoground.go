// Package airtoground is Air-to-Ground Actions: a short air campaign against WOPR's
// defended targets (docs/PLAN.md §6.3, a bespoke game on the sim engine's combat-results
// and kill-ratio tables). Each sortie sends one package at one target: strike aircraft bomb
// it, SEAD aircraft fight its surface-to-air missile sites, and escorts tie up WOPR's
// interceptors. WOPR hides a mobile SAM battery where it expects the next raid, and keeps
// its interceptors on the ground when it expects a strong escort.
package airtoground

import (
	"fmt"
	"math/rand/v2"
	"strconv"
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
		"AIR-TO-GROUND ACTIONS. SIX SORTIES, TWELVE AIRCRAFT, SIX DEFENDED TARGETS.",
		"EACH SORTIE SENDS ONE PACKAGE AT ONE TARGET: STRIKE AIRCRAFT BOMB IT, SEAD",
		"AIRCRAFT FIGHT ITS SAM SITES, ESCORTS TIE UP WOPR'S INTERCEPTORS.",
		"DESTROY TARGETS WORTH 10 POINTS TO WIN. LOST AIRCRAFT ARE NOT REPLACED.",
		"SET A PACKAGE (TARGET 5 STRIKE 4 SEAD 2 ESCORT 2), THEN GO. TYPE HELP FOR MORE.",
	)
	promptPackage = script.Orig("PACKAGE: ")
	lineHelp      = script.Orig(
		"TARGET <NUMBER OR NAME>, STRIKE <N>, SEAD <N>, ESCORT <N> SET THE PACKAGE;",
		"GO FLIES IT. STATUS SHOWS THE TARGETS. END CALLS OFF THE CAMPAIGN.",
		"DESTROYING THE RADAR BLINDS THE SAMS, THE AIRFIELD GROUNDS THE INTERCEPTORS,",
		"AND THE FUEL DEPOT STOPS WOPR REPLACING THEM.",
	)
	linePackage   = script.Orig("PACKAGE: TARGET #. STRIKE #, SEAD #, ESCORT #.")
	lineNone      = script.Orig("NONE")
	lineWhich     = script.Orig("WHICH TARGET? 1 TO 6, OR ITS NAME.")
	lineGone      = script.Orig("THE # IS ALREADY DESTROYED.")
	lineTooMany   = script.Orig("THAT IS # AIRCRAFT; YOU HAVE #.")
	lineNoStrike  = script.Orig("A PACKAGE NEEDS AT LEAST ONE STRIKE AIRCRAFT.")
	lineStatus    = script.Orig("SORTIE # OF 6. AIRCRAFT #. POINTS #; 10 TO WIN.")
	lineTableHead = script.Orig(" #  TARGET           VALUE  DAMAGE  SAMS")
	lineDestroyed = script.Orig("DESTROYED")
	lineWOPRAir   = script.Orig("WOPR: INTERCEPTORS #. ", "WOPR: INTERCEPTORS GROUNDED. ")
	lineMobile    = script.Orig("A MOBILE SAM BATTERY IS SOMEWHERE.", "THE MOBILE SAM BATTERY IS GONE.")
	lineSortie    = script.Orig("SORTIE #: THE #. STRIKE #, SEAD #, ESCORT #.")
	lineScramble  = script.Orig("WOPR SCRAMBLES INTERCEPTORS: #.", "WOPR KEEPS ITS INTERCEPTORS ON THE GROUND.")
	lineEscorts   = script.Orig("ESCORTS: INTERCEPTORS DOWN #, DRIVEN OFF #; ESCORTS LOST #.")
	lineIntercept = script.Orig("INTERCEPTORS: SHOT DOWN #, TURNED BACK #; INTERCEPTORS LOST #.")
	lineWaiting   = script.Orig("THE MOBILE SAM BATTERY WAS WAITING AT THE #.")
	lineSEAD      = script.Orig("SEAD: SITES DESTROYED #, SUPPRESSED #; AIRCRAFT LOST #.")
	lineMobileHit = script.Orig("THE MOBILE SAM BATTERY IS DESTROYED.")
	lineSAMs      = script.Orig("SAMS: SHOT DOWN #, TURNED BACK #.")
	lineBombs     = script.Orig("OVER THE #: AIRCRAFT #, HITS #, LOST TO FLAK #.", "NO AIRCRAFT REACH THE #.")
	lineDown      = script.Orig("THE # IS DESTROYED. # POINTS.")
	lineEffects   = script.Orig(
		"WOPR CAN NO LONGER REPLACE ITS INTERCEPTORS.",
		"WOPR'S INTERCEPTORS ARE GROUNDED.",
		"WOPR'S SAMS ARE FIRING BLIND.",
	)
	lineAll      = script.Orig("EVERY TARGET IS DESTROYED.")
	lineNoPlanes = script.Orig("THE SQUADRON IS GONE.")
	lineCallOff  = script.Orig("YOU CALL OFF THE CAMPAIGN.")
	lineScore    = script.Orig("THE CAMPAIGN ENDS WITH # POINTS.")
	targetNames  = script.Orig("BRIDGE", "FUEL DEPOT", "MARSHALLING YARD", "AIRFIELD", "RADAR", "COMMAND BUNKER")
	categories   = script.Orig("AIRCRAFT", "INTERCEPTORS", "SAM SITES", "TARGETS")
)

// Lines is every script block, for the provenance test.
var Lines = []script.Ls{
	lineIntro, promptPackage, lineHelp, linePackage, lineNone, lineWhich, lineGone, lineTooMany, lineNoStrike,
	lineStatus, lineTableHead, lineDestroyed, lineWOPRAir, lineMobile, lineSortie, lineScramble, lineEscorts,
	lineIntercept, lineWaiting, lineSEAD, lineMobileHit, lineSAMs, lineBombs, lineDown, lineEffects, lineAll,
	lineNoPlanes, lineCallOff, lineScore, targetNames, categories, artTitle, areaMap, areaIcons, areaWrecks,
	areaHit, burstsAbove, burstsBelow,
}

// The targets, in table order; the effects of losing three of them.
const (
	bridge = iota
	fuelDepot
	yard
	airfield
	radar
	bunker
	numTargets
)

const (
	sorties      = 6
	squadron     = 12
	winPoints    = 10
	drawPoints   = 6
	maxFighters  = 6
	strikeAttack = 6 // a bomb run against the target's defence and hardness
	escortAttack = 8 // against an interceptor's defence of 4
	interceptor  = 4 // an interceptor's attack and defence
	planeDefence = 2 // a strike or SEAD aircraft under attack
	seadAttack   = 6 // against a SAM site's defence of 3
	siteDefence  = 3
	samAttack    = 4 // halved once the radar is gone
)

// target is one defended target: its points, the hits that destroy it, its defence and
// hardness on the combat-results table, and its fixed SAM sites.
type target struct {
	value, toughness, defence int
	hardness                  sim.Terrain
	sams, damage              int
}

func (t *target) destroyed() bool { return t.damage >= t.toughness }

func newTargets() [numTargets]target {
	return [numTargets]target{
		bridge:    {value: 3, toughness: 2, defence: 2, hardness: sim.Rough, sams: 1},
		fuelDepot: {value: 2, toughness: 2, defence: 2, sams: 1},
		yard:      {value: 2, toughness: 3, defence: 2},
		airfield:  {value: 3, toughness: 3, defence: 3, sams: 2},
		radar:     {value: 2, toughness: 1, defence: 2, sams: 2},
		bunker:    {value: 4, toughness: 2, defence: 1, hardness: sim.City, sams: 3},
	}
}

// maxGroup caps a typed aircraft count, far above any squadron, so that a package's size
// cannot overflow and fly a huge or negative number of aircraft.
const maxGroup = 999

// pack is a strike package; target is -1 until one is chosen.
type pack struct{ target, strike, sead, escort int }

func (p pack) size() int { return p.strike + p.sead + p.escort }

func fill(text string, args ...any) string {
	for _, a := range args {
		text = strings.Replace(text, "#", fmt.Sprint(a), 1)
	}
	return text
}

func say(lines ...string) proto.Output { return proto.Say{Lines: lines, Pace: proto.PaceSpeech} }

func table(lines ...string) proto.Output { return proto.Say{Lines: lines, Pace: proto.PaceTable} }

// Game is the campaign as a proto.Program.
type Game struct {
	rng        *rand.Rand
	sortie     int
	aircraft   int
	fighters   int
	targets    [numTargets]target
	pkg        pack
	mobile     int // where WOPR's mobile SAM battery waits this sortie; -1 once destroyed
	lastTarget int // what WOPR remembers of the last raid
	lastEscort int
	losses     [2]map[string]int
	over       bool
}

// New returns a game.
func New() games.Game { return &Game{} }

// Start implements proto.Program.
func (g *Game) Start(env proto.Env) []proto.Output {
	g.rng = proto.NewRand(env.Seed, 0)
	g.sortie, g.aircraft, g.fighters = 1, squadron, 4
	g.targets = newTargets()
	g.pkg = pack{target: -1, strike: 4, sead: 2, escort: 2}
	g.lastTarget, g.lastEscort = -1, 0
	g.losses = [2]map[string]int{{}, {}}
	g.placeMobile()
	outs := []proto.Output{table(artTitle.Texts()...), say(lineIntro.Texts()...)}
	return append(append(outs, g.status(-1)...), g.ask())
}

// View implements proto.Program; the campaign is console text and art.
func (g *Game) View(*proto.Canvas) {}

func (g *Game) ask() proto.Output { return proto.Prompt{Text: promptPackage[0].Text} }

func (g *Game) again(lines ...string) []proto.Output { return []proto.Output{say(lines...), g.ask()} }

func (g *Game) points() int {
	n := 0
	for i := range g.targets {
		if g.targets[i].destroyed() {
			n += g.targets[i].value
		}
	}
	return n
}

// status is the campaign as it stands: the score, the targets table with the target-area
// map beside it, WOPR's defences and the package being planned. hit is the target the
// sortie just flown destroyed, which the map shows going up, or -1.
func (g *Game) status(hit int) []proto.Output {
	lines := []string{lineTableHead[0].Text}
	for i := range g.targets {
		t := &g.targets[i]
		if t.destroyed() {
			lines = append(lines, fmt.Sprintf("%2d  %-16s %5d  %s", i+1, targetNames[i].Text, t.value, lineDestroyed[0].Text))
			continue
		}
		lines = append(lines, fmt.Sprintf("%2d  %-16s %5d  %4d/%d  %4d", i+1, targetNames[i].Text, t.value, t.damage, t.toughness, t.sams))
	}
	air := fill(lineWOPRAir[0].Text, g.fighters)
	if g.targets[airfield].destroyed() {
		air = lineWOPRAir[1].Text
	}
	mobile := lineMobile[0].Text
	if g.mobile < 0 {
		mobile = lineMobile[1].Text
	}
	return []proto.Output{
		say(fill(lineStatus[0].Text, g.sortie, g.aircraft, g.points())),
		table(beside(lines, g.area(hit))...),
		say(air+mobile, g.packageLine()),
	}
}

func (g *Game) packageLine() string {
	name := lineNone[0].Text
	if g.pkg.target >= 0 {
		name = targetNames[g.pkg.target].Text
	}
	return fill(linePackage[0].Text, name, g.pkg.strike, g.pkg.sead, g.pkg.escort)
}

// findTarget reads a target by number or by a word of its name (or the start of one).
func findTarget(w string) int {
	if n, err := strconv.Atoi(w); err == nil {
		if n >= 1 && n <= numTargets {
			return n - 1
		}
		return -1
	}
	found := -1
	for i, name := range targetNames {
		for _, part := range strings.Fields(name.Text) {
			if part == w || (len(w) >= 3 && strings.HasPrefix(part, w)) {
				if found >= 0 && found != i {
					return -1
				}
				found = i
			}
		}
	}
	return found
}

var filler = map[string]bool{"THE": true, "AT": true, "AND": true, "WITH": true, "ON": true, "A": true, "OF": true}

// Handle implements proto.Program. A line may set any part of the package and fly it:
// "TARGET 5 STRIKE 4 SEAD 2 GO".
func (g *Game) Handle(ev proto.Event) []proto.Output {
	line, ok := ev.(proto.LineEvent)
	if !ok || g.over {
		return nil
	}
	words := strings.Fields(prompt.Normalize(line.Text))
	if len(words) == 0 {
		return g.again(lineHelp.Texts()...)
	}
	p, launch := g.pkg, false
	for i := 0; i < len(words); i++ {
		w := words[i]
		count := func() (int, bool) {
			if i+1 < len(words) {
				if n, err := strconv.Atoi(words[i+1]); err == nil {
					i++
					return min(n, maxGroup), true
				}
			}
			return 0, false
		}
		switch w {
		case "HELP":
			return g.again(lineHelp.Texts()...)
		case "STATUS", "MAP", "TARGETS":
			return append(g.status(-1), g.ask())
		case "END", "STOP", "RTB":
			g.over = true
			return []proto.Output{say(lineCallOff[0].Text), g.finish()}
		case "GO", "FLY", "LAUNCH":
			launch = true
		case "TARGET", "HIT":
			for i+1 < len(words) && filler[words[i+1]] {
				i++
			}
			if i+1 >= len(words) || findTarget(words[i+1]) < 0 {
				return g.again(lineWhich[0].Text)
			}
			i++
			p.target = findTarget(words[i])
		case "STRIKE", "BOMB", "BOMBERS":
			if n, ok := count(); ok {
				p.strike = n
			}
		case "SEAD":
			if n, ok := count(); ok {
				p.sead = n
			}
		case "ESCORT", "ESCORTS", "COVER":
			if n, ok := count(); ok {
				p.escort = n
			}
		default:
			if filler[w] {
				continue
			}
			t := findTarget(w)
			if t < 0 {
				return g.again(lineHelp.Texts()...)
			}
			p.target = t
		}
	}
	p.strike, p.sead, p.escort = max(p.strike, 0), max(p.sead, 0), max(p.escort, 0)
	g.pkg = p
	if !launch {
		return g.again(g.packageLine())
	}
	switch {
	case p.target < 0:
		return g.again(lineWhich[0].Text)
	case g.targets[p.target].destroyed():
		return g.again(fill(lineGone[0].Text, targetNames[p.target].Text))
	case p.strike == 0:
		return g.again(lineNoStrike[0].Text)
	case p.size() > g.aircraft:
		return g.again(fill(lineTooMany[0].Text, p.size(), g.aircraft))
	}
	return g.fly()
}

// placeMobile is WOPR's guess at the next raid: usually wherever the last one went, if
// it still stands; otherwise the most valuable target still standing; now and then
// somewhere else entirely.
func (g *Game) placeMobile() {
	if g.mobile < 0 {
		return
	}
	var standing []int
	best := -1
	for i := range g.targets {
		if g.targets[i].destroyed() {
			continue
		}
		standing = append(standing, i)
		if best < 0 || g.targets[i].value > g.targets[best].value {
			best = i
		}
	}
	switch {
	case len(standing) == 0:
	case g.rng.IntN(4) == 0:
		g.mobile = standing[g.rng.IntN(len(standing))]
	case g.lastTarget >= 0 && !g.targets[g.lastTarget].destroyed():
		g.mobile = g.lastTarget
	default:
		g.mobile = best
	}
}

// scramble is how many interceptors WOPR sends: none with the airfield gone, and usually
// none when it expects the escort to outnumber them.
func (g *Game) scramble() int {
	if g.targets[airfield].destroyed() || g.fighters == 0 {
		return 0
	}
	if g.lastEscort > g.fighters && g.rng.IntN(3) < 2 {
		return 0
	}
	return g.fighters
}

func (g *Game) roll(attack, defence int, t sim.Terrain) sim.Result {
	return sim.CRT(attack, defence, t, g.rng.IntN(6)+1)
}

// fly resolves one sortie: interceptors, then the SAM sites, then the bomb run.
func (g *Game) fly() []proto.Output {
	p := g.pkg
	t := &g.targets[p.target]
	name := targetNames[p.target].Text
	lines := []string{fill(lineSortie[0].Text, g.sortie, name, p.strike, p.sead, p.escort)}
	lost, hit := 0, -1 // hit: the target, if this sortie destroyed it
	strike, sead, escort := p.strike, p.sead, p.escort

	// Interceptors: each escort ties one up; the rest go for the strike and SEAD aircraft.
	up := g.scramble()
	if up == 0 && !g.targets[airfield].destroyed() && g.fighters > 0 {
		lines = append(lines, lineScramble[1].Text)
	}
	if up > 0 {
		lines = append(lines, fill(lineScramble[0].Text, up))
		down, off, escLost := 0, 0, 0
		for range min(escort, up) {
			switch g.roll(escortAttack, interceptor, sim.Open) {
			case sim.DefenderLoses:
				down++
			case sim.Exchange:
				down++
				escLost++
			case sim.DefenderRetreats:
				off++
			case sim.AttackerLoses:
				escLost++
			}
		}
		if escort > 0 {
			lines = append(lines, fill(lineEscorts[0].Text, down, off, escLost))
		}
		free := up - min(escort, up)
		g.fighters -= down
		lost += escLost
		shot, back, intLost := 0, 0, 0
		for range free {
			if strike+sead == 0 {
				break
			}
			hitStrike := g.rng.IntN(strike+sead) < strike
			res := g.roll(interceptor, planeDefence, sim.Open)
			switch res {
			case sim.DefenderLoses, sim.Exchange, sim.DefenderRetreats:
				if hitStrike {
					strike--
				} else {
					sead--
				}
				if res == sim.DefenderRetreats {
					back++
				} else {
					shot++
				}
			}
			if res == sim.Exchange || res == sim.AttackerLoses {
				intLost++
			}
		}
		if free > 0 {
			lines = append(lines, fill(lineIntercept[0].Text, shot, back, intLost))
		}
		g.fighters -= intLost
		lost += shot
		g.losses[sim.WOPR][categories[1].Text] += down + intLost
	}

	// SAM sites: the fixed ones, then the mobile battery if it waited here. SEAD aircraft
	// take them in turn; whatever is left fires at the strike aircraft.
	sites := make([]bool, t.sams) // true: the mobile battery
	if g.mobile == p.target {
		sites = append(sites, true)
		lines = append(lines, fill(lineWaiting[0].Text, name))
	}
	next, destroyed, suppressed, seadLost := 0, 0, 0, 0
	mobileHit := false
	for range sead {
		if next == len(sites) {
			break
		}
		res := g.roll(seadAttack, siteDefence, sim.Open)
		switch res {
		case sim.DefenderLoses, sim.Exchange:
			destroyed++
			if sites[next] {
				g.mobile, mobileHit = -1, true
			} else {
				t.sams--
			}
			next++
		case sim.DefenderRetreats:
			suppressed++
			next++
		}
		if res == sim.Exchange || res == sim.AttackerLoses {
			seadLost++
		}
	}
	if sead > 0 && len(sites) > 0 {
		lines = append(lines, fill(lineSEAD[0].Text, destroyed, suppressed, seadLost))
		if mobileHit {
			lines = append(lines, lineMobileHit[0].Text)
		}
	}
	lost += seadLost
	g.losses[sim.WOPR][categories[2].Text] += destroyed
	attack := samAttack
	if g.targets[radar].destroyed() {
		attack /= 2
	}
	shot, back := 0, 0
	for range len(sites) - next {
		if strike == 0 {
			break
		}
		switch g.roll(attack, planeDefence, sim.Open) {
		case sim.DefenderLoses, sim.Exchange:
			shot++
			strike--
		case sim.DefenderRetreats:
			back++
			strike--
		}
	}
	if len(sites) > next && shot+back+strike > 0 {
		lines = append(lines, fill(lineSAMs[0].Text, shot, back))
	}
	lost += shot

	// The bomb run.
	if strike == 0 {
		lines = append(lines, fill(lineBombs[1].Text, name))
	} else {
		hits, flak := 0, 0
		for range strike {
			res := g.roll(strikeAttack, t.defence, t.hardness)
			if res == sim.DefenderLoses || res == sim.Exchange {
				hits++
			}
			if res == sim.Exchange || res == sim.AttackerLoses {
				flak++
			}
		}
		lines = append(lines, fill(lineBombs[0].Text, name, strike, hits, flak))
		lost += flak
		t.damage = min(t.damage+hits, t.toughness)
		if t.destroyed() {
			g.losses[sim.WOPR][categories[3].Text]++
			hit = p.target
			lines = append(lines, fill(lineDown[0].Text, name, t.value))
			switch p.target {
			case fuelDepot:
				lines = append(lines, lineEffects[0].Text)
			case airfield:
				lines = append(lines, lineEffects[1].Text)
			case radar:
				lines = append(lines, lineEffects[2].Text)
			}
			g.pkg.target = -1
		}
	}
	g.aircraft -= lost
	g.losses[sim.Player][categories[0].Text] += lost
	g.lastTarget, g.lastEscort = p.target, p.escort
	return g.next(lines, hit)
}

// next ends the sortie: the campaign ends, or WOPR repairs, moves its mobile battery and
// the next sortie is planned, the map showing hit, the target the sortie destroyed, or -1.
func (g *Game) next(lines []string, hit int) []proto.Output {
	outs := []proto.Output{say(lines...)}
	end := ""
	switch {
	case g.points() == g.maxPoints():
		end = lineAll[0].Text
	case g.aircraft <= 0:
		end = lineNoPlanes[0].Text
	case g.sortie >= sorties:
		end = fill(lineScore[0].Text, g.points())
	}
	if end != "" {
		g.over = true
		return append(outs, say(end), g.finish())
	}
	g.sortie++
	if !g.targets[fuelDepot].destroyed() && !g.targets[airfield].destroyed() && g.fighters < maxFighters {
		g.fighters++
	}
	g.placeMobile()
	// Shrink a package the squadron can no longer fill: escorts go first, then SEAD.
	for g.pkg.size() > g.aircraft && g.pkg.escort > 0 {
		g.pkg.escort--
	}
	for g.pkg.size() > g.aircraft && g.pkg.sead > 0 {
		g.pkg.sead--
	}
	g.pkg.strike = min(g.pkg.strike, g.aircraft)
	return append(append(outs, g.status(hit)...), g.ask())
}

func (g *Game) maxPoints() int {
	n := 0
	for i := range g.targets {
		n += g.targets[i].value
	}
	return n
}

// finish scores the campaign: 10 points win, fewer than 6 lose, and losing the squadron
// loses whatever the score.
func (g *Game) finish() proto.Output {
	o := proto.Loss
	switch pts := g.points(); {
	case g.aircraft <= 0:
	case pts >= winPoints:
		o = proto.Win
	case pts >= drawPoints:
		o = proto.Draw
	}
	return proto.Done{Result: proto.Result{Outcome: o, Lines: sim.RatioTable(g.losses)}}
}
