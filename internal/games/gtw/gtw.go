// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

// Package gtw is Global Thermonuclear War, the film's set piece (docs/PLAN.md §6.2): choose
// a side and list targets; the first strike flies on the big board at once, then you order
// two more as DEFCON falls, and WOPR takes the last rung alone. The kill ratios follow, then
// you try to stop a war that will not stop. Only tic-tac-toe gets through: GTW hands off to
// it in "climax" mode, and to the ending after that. The game cannot be won (exchange.go).
// With --no-climax (Env.NoClimax), the kill ratios offer another war instead of the climax.
// The third choice at the side prompt plays the closing montage's scenarios, each a war that
// plays itself to WINNER: NONE (scenarios.go). With --authentic (Env.Authentic), GTW shows
// only what the film shows: a side and targets, then a war that runs by itself to the climax.
package gtw

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"github.com/GhostofGoes/WOPR/internal/assets"
	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/prompt"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// Slugs GTW knows.
const (
	Slug          = "global-thermonuclear-war"
	TicTacToeSlug = "tic-tac-toe"
	ClimaxMode    = "climax"
)

// Exchange timing, in animation frames.
const (
	frame        = 100 * time.Millisecond
	spacing      = 2  // frames between one side's launches
	detectAt     = 8  // WOPR launches on warning this many frames into a strike
	strikeEnd    = 44 // a strike's last frame: everything has landed
	holdFrames   = 10 // the pause before a stage that starts without an order
	finalEnd     = 34 // the DEFCON 1 attack's last frame
	maxTargets   = 8
	consoleW     = 80                     // the console's width, which a line WOPR says fits
	codeFrame    = 250 * time.Millisecond // the climax code display's cadence
	blinkFrame   = 250 * time.Millisecond // keeps the DEFCON 1 blink running on still screens
	codeEvery    = 16                     // climax frames per cracked character
	codeHeldBack = 3                      // characters the climax never cracks: the ending does
	windowTick   = time.Second            // the launch window's countdown cadence
	filmAssess   = 30                     // under --authentic, frames DEFCON 1 holds before the kill ratios
	filmRatios   = 80                     // and frames the kill ratios show before the climax
)

type phase uint8

const (
	chooseSide phase = iota
	listTargets
	chooseScenario // the scenario menu is up
	briefing       // a scenario's briefing is up; the board opens by itself
	flight         // a stage plays out on the board
	orders         // a strike prompt is up
	assessed       // DEFCON 1 is over; Enter shows the kill ratios
	ratios
	another // --no-climax's ANOTHER WAR prompt is up, over the kill ratios
	climax
)

// kind is a track's weapon: it sets the flight time, the arc and the designator.
type kind uint8

const (
	kindICBM kind = iota
	kindSLBM
	kindBomber
)

// flightFrames is each kind's flight: SLBMs are close in and land first.
var flightFrames = [3]int{24, 10, 30}

// missile is one track on the board.
type missile struct {
	from, to assets.Place
	launch   int  // frame within its stage
	ours     bool // the player's (outgoing, '+') or WOPR's (incoming, '*')
	kind     kind
	id       int // serial over the whole war, for its trajectory heading
}

// Game is GTW as a proto.Program.
type Game struct {
	env      proto.Env
	rng      *rand.Rand
	phase    phase
	side     assets.Side    // the player's
	targets  []assets.Place // the enemy's, as listed
	scen     *scenario      // the scenario playing itself; nil for a war of your own
	page     int            // the scenario menu's page, from 0
	war      *war
	egg      *egg          // the easter egg the seed meets, which sets WOPR's doctrine; nil: none
	stage    int           // 1..3 strikes, 4 the DEFCON 1 attack; 0 before the board opens
	rep      report        // the current stage, resolved when it starts
	auto     bool          // AUTO, the launch window shut, a playback or --authentic: WOPR orders every remaining strike
	window   time.Duration // what is left of the launch window at a strike prompt; 0: none runs
	missiles []missile
	impacts  []missile // every earlier stage's tracks, drawn as impacts only
	traj     []missile // your latest missiles, newest first, for the trajectory table
	serial   int
	cities   []assets.Place // the player's side, shuffled: WOPR's aim points
	aimYou   int            // next listed target for your city tracks
	aimWOPR  int            // next of cities for WOPR's
	noise    [10]int        // kill-ratio noise per row, shared by both sides
	frame    int
	defcon   int
	ratios   [2][12]int // [player, WOPR] values, in row order
	cracked  int        // characters of the launch code shown
	climaxT  int        // climax frames elapsed
	rejected int        // climax inputs refused, for the hints
	stops    int        // climax tries to stop the war
	nudged   bool       // one of them has had WOPR's answer
	listing  bool       // LIST GAMES moved the climax to the console
	film     bool       // drawn by Film for movie mode, whose viewer gives no orders

	voice      *rand.Rand           // WOPR's remarks (react.go)
	remarked   string               // the current stage's remark, or ""
	lastRemark [len(reactLines)]int // each reaction's last variant, plus one; 0 before its first
	planned    bool                 // a strike prompt took the war plan
}

// New returns a game.
func New() games.Game { return &Game{defcon: 5} }

func say(ls ...script.Ls) proto.Output {
	var lines []string
	for _, l := range ls {
		lines = append(lines, l.Texts()...)
	}
	return proto.Say{Lines: lines, Pace: proto.PaceSpeech}
}

// speak says lines made from script text: a template filled in, or several put together.
func speak(lines []string) proto.Output { return proto.Say{Lines: lines, Pace: proto.PaceSpeech} }

// strip prints one of a stage's strip lines at table pace, so the text keeps up with the
// board: WOPR's launch line, with its DEFCON, shows as the ladder moves, not seconds after.
func strip(line string) proto.Output { return table([]string{line}) }

func table(lines []string) proto.Output { return proto.Say{Lines: lines, Pace: proto.PaceTable} }

// fill replaces each # in l with the next arg.
func fill(l script.L, args ...any) string {
	text := l.Text
	for _, a := range args {
		text = strings.Replace(text, "#", fmt.Sprint(a), 1)
	}
	return text
}

// Start implements proto.Program: the two outlines and the side choice, as console text, or,
// in a scenario's launch mode (ScenarioLaunch), that scenario's briefing. The seed decides
// whether WOPR fights to an easter egg's doctrine. The film's war (movie mode, and
// --authentic) is always a measured response. --authentic wins over what the film does not
// have: the launch window, --no-climax and a scenario's launch mode (the command line refuses
// them together).
func (g *Game) Start(env proto.Env) []proto.Output {
	if env.Authentic {
		env.LaunchWindow, env.NoClimax, env.Mode = 0, false, ""
	}
	g.env = env
	if !g.film && !env.Authentic {
		g.egg = eggFor(env.Seed)
	}
	g.rng = proto.NewRand(env.Seed, 0)
	g.voice = proto.NewRand(env.Seed, remarkStream)
	outs := []proto.Output{proto.SetLayout{Layout: proto.LayoutConsole}}
	if s := scenarioFor(env.Mode); s != nil {
		return append(outs, g.brief(s)...)
	}
	return append(outs, g.sideScreen()...)
}

// sideScreen is the side choice: the outlines, the film's question and two choices with the
// third after them (under --authentic, the film's two alone), and the film's prompt.
func (g *Game) sideScreen() []proto.Output {
	lines := lineWhichSide.Texts()
	if !g.env.Authentic {
		end := len(lines) - 1 // the film's blank line before the prompt
		lines = append(append(lines[:end:end], lineScenarios[0].Text), lines[end:]...)
	}
	return []proto.Output{table(append(sideChoice(), "")), speak(lines), proto.Prompt{Text: promptSide[0].Text}}
}

// Side-choice geometry: the United States' outline from column 0, 36 columns wide, and the
// Soviet Union's from column ussrCol, 42 wide, which ends at column 79.
const (
	usWidth   = 36
	ussrCol   = 38
	ussrWidth = 42
)

// sideChoice is the film's side-choice picture: the two nations' outlines side by side, each
// named beneath.
func sideChoice() []string {
	lines := make([]string, 0, len(assets.OutlineUS)+1)
	for i, us := range assets.OutlineUS {
		row := fmt.Sprintf("%-*s%s", ussrCol, us.Text, assets.OutlineUSSR[i].Text)
		lines = append(lines, strings.TrimRight(row, " "))
	}
	us, ussr := lineLabels[0].Text, lineLabels[1].Text
	names := strings.Repeat(" ", (usWidth-len(us))/2) + us
	names += strings.Repeat(" ", ussrCol+(ussrWidth-len(ussr))/2-len(names)) + ussr
	return append(lines, names)
}

// Handle implements proto.Program.
func (g *Game) Handle(ev proto.Event) []proto.Output {
	switch ev := ev.(type) {
	case proto.LineEvent:
		switch g.phase {
		case chooseSide:
			return g.onSide(ev.Text)
		case listTargets:
			return g.onTarget(ev.Text)
		case chooseScenario:
			return g.onMenu(ev.Text)
		case orders:
			return g.onOrder(ev.Text)
		case assessed:
			return g.showRatios()
		case ratios:
			switch {
			case g.scen != nil: // a playback ends at the kill ratios, not the climax
				return g.endPlayback()
			case g.env.NoClimax:
				g.phase = another
				return []proto.Output{proto.Prompt{Text: promptAnother[0].Text}}
			}
			return g.startClimax()
		case another:
			return g.onAnother(ev.Text)
		case climax:
			return g.onClimax(ev.Text)
		case flight, briefing: // no prompt is up; the host never sends one
		}
	case proto.Drained: // a briefing has been read
		if g.phase == briefing {
			return g.openBoard()
		}
	case proto.TickEvent:
		return g.onTick(ev.Dt)
	}
	return nil
}

// onSide reads the side choice: a side, or 3 for the scenario menu. Anything else asks again:
// with CHOOSE 1, 2 OR 3., or under --authentic, which has no menu, with the film's prompt
// alone.
func (g *Game) onSide(input string) []proto.Output {
	again := []proto.Output{say(lineSideAgain), proto.Prompt{Text: promptSide[0].Text}}
	if g.env.Authentic {
		again = again[1:]
	}
	switch prompt.Normalize(input) {
	case "1", "ONE", "UNITED STATES", "US", "USA":
		g.side = assets.US
	case "2", "TWO", "SOVIET UNION", "USSR", "RUSSIA":
		g.side = assets.USSR
	case "3", "THREE", "SCENARIOS", "SCENARIO", "LIST SCENARIOS":
		if g.env.Authentic {
			return again
		}
		return g.showMenu(0)
	default:
		return again
	}
	g.phase = listTargets
	return []proto.Output{proto.Clear{}, say(lineAwaiting), proto.Prompt{}}
}

// onTarget collects targets until an empty line. A line may hold several, split on commas.
// Each must be a place the board knows on the enemy's side (assets.Find): any other name is
// refused, so a target is only ever struck where it is, and LIST shows what is on file. A
// target, or a refusal, is not repeated, so "WASHINGTON, D.C." is one target, or one refusal.
// WOPR names the targets a line adds and counts them toward the eight, and notes a target an
// earlier line already listed.
func (g *Game) onTarget(input string) []proto.Output {
	if g.env.Authentic {
		return g.filmTarget(input)
	}
	if strings.TrimSpace(input) == "" {
		if len(g.targets) == 0 {
			return []proto.Output{say(lineNoTargets), proto.Prompt{}}
		}
		return g.openBoard()
	}
	switch prompt.Normalize(input) {
	case "LIST", "LIST TARGETS", "HELP", "": // "" is "?"
		return []proto.Output{table(g.targetList()), proto.Prompt{}}
	}
	before := len(g.targets)
	var notes, refused []string // what WOPR says back after the targets added, a line each
	once := func(lines *[]string, line string) {
		if !slices.Contains(*lines, line) {
			*lines = append(*lines, line)
		}
	}
	reply := func(last ...string) []proto.Output {
		lines := append(confirmed(g.targets[before:], len(g.targets)), notes...)
		lines = append(append(lines, refused...), last...)
		if len(lines) == 0 {
			return []proto.Output{proto.Prompt{}}
		}
		return []proto.Output{speak(lines), proto.Prompt{}}
	}
	for _, t := range strings.Split(input, ",") {
		name := strings.ToUpper(strings.Join(strings.Fields(t), " "))
		if name == "" {
			continue
		}
		p, ok := assets.Find(name)
		switch {
		case !ok:
			once(&refused, fill(lineUnknown[0], name))
		case p.Side != enemyOf(g.side):
			once(&refused, fill(lineOwnSide[0], p.Name))
		case slices.Contains(g.targets[:before], p):
			once(&notes, fill(lineListed[0], p.Name))
		case slices.Contains(g.targets, p): // named twice on this line: one target
		case len(g.targets) == maxTargets:
			return reply(lineTooMany[0].Text)
		default:
			g.targets = append(g.targets, p)
		}
	}
	if len(refused) > 0 {
		return reply(lineListHint[0].Text)
	}
	return reply()
}

// filmTarget is onTarget under --authentic, where the targets prompt takes every line without
// a word, as the film's does. What onTarget would refuse or note strikes nothing: a name the
// board does not know, a place on your own side, a target already listed, and a ninth. LIST
// is a name like any other. The empty line ends the list. A list with no target the board
// knows takes the war plan's: the enemy's first eight cities.
func (g *Game) filmTarget(input string) []proto.Output {
	if strings.TrimSpace(input) == "" {
		if len(g.targets) == 0 {
			g.targets = cities(enemyOf(g.side))[:maxTargets]
		}
		return g.openBoard()
	}
	for _, t := range strings.Split(input, ",") {
		p, ok := assets.Find(strings.ToUpper(strings.Join(strings.Fields(t), " ")))
		if ok && p.Side == enemyOf(g.side) && !slices.Contains(g.targets, p) && len(g.targets) < maxTargets {
			g.targets = append(g.targets, p)
		}
	}
	return []proto.Output{proto.Prompt{}}
}

// confirmed is WOPR's line naming the targets a line added, with their places in the list
// of eight: "SEATTLE. TARGET 3 OF 8.", or "SEATTLE, LAS VEGAS. TARGETS 2-3 OF 8.". A long
// line breaks after a comma, so no line is wider than the console.
func confirmed(added []assets.Place, total int) []string {
	switch len(added) {
	case 0:
		return nil
	case 1:
		return []string{fill(lineAccepted[0], added[0].Name, total, maxTargets)}
	}
	names := make([]string, len(added))
	for i, p := range added {
		names[i] = p.Name
	}
	return commaLines(fill(lineAccepted[1], strings.Join(names, ", "), total-len(added)+1, total, maxTargets))
}

// commaLines breaks a line of names wider than the console after a comma, so that no name is
// split.
func commaLines(text string) []string {
	var lines []string
	line := ""
	for _, part := range strings.SplitAfter(text, ", ") {
		if line != "" && len(line+strings.TrimRight(part, " ")) > consoleW {
			lines = append(lines, strings.TrimRight(line, " "))
			line = ""
		}
		line += part
	}
	return append(lines, line)
}

// Target list geometry: LIST prints the enemy's targets in listCols columns of listW.
const (
	listCols = 4
	listW    = 20
)

// targetList is LIST at the targets prompt: every place on the enemy's side the board knows,
// in alphabetical order down four columns.
func (g *Game) targetList() []string {
	var names []string
	for _, p := range append(slices.Clone(assets.Cities), assets.Targets...) {
		if p.Side == enemyOf(g.side) {
			names = append(names, p.Name)
		}
	}
	slices.Sort(names)
	rows := ceilDiv(len(names), listCols)
	lines := []string{fill(lineOnFile[0], g.nation(wopr))}
	for r := range rows {
		var b strings.Builder
		for c := range listCols {
			if i := c*rows + r; i < len(names) {
				fmt.Fprintf(&b, "%-*s", listW, names[i])
			}
		}
		lines = append(lines, strings.TrimRight(b.String(), " "))
	}
	return lines
}

func enemyOf(s assets.Side) assets.Side {
	if s == assets.US {
		return assets.USSR
	}
	return assets.US
}

// openBoard opens the big board at DEFCON 5 and launches the first strike: the war plan's,
// at the listed targets, with no order asked (the film's first strike), or a scenario's. An
// easter egg's doctrine is named first. Under --authentic the film gives no orders, so WOPR
// orders every strike, as under AUTO.
func (g *Game) openBoard() []proto.Output {
	g.auto = g.auto || g.env.Authentic
	rng := g.rng
	if g.scen != nil {
		// A playback draws from its own seed, the same in every session, and aims in order.
		rng, g.cities = proto.NewRand(g.seed(), 0), slices.Clone(g.scen.aims)
	} else {
		for _, c := range assets.Cities {
			if c.Side == g.side {
				g.cities = append(g.cities, c)
			}
		}
		// Stream 0 draws a fixed amount whatever the orders: the shuffle, then the noise.
		g.rng.Shuffle(len(g.cities), func(i, j int) { g.cities[i], g.cities[j] = g.cities[j], g.cities[i] })
	}
	for i := range g.noise {
		g.noise[i] = 97 + rng.IntN(7)
	}
	g.war = newWar(g.doctrine())
	outs := append([]proto.Output{proto.SetLayout{Layout: proto.LayoutFull}, proto.Clear{}}, g.announce()...)
	outs = append(outs, g.begin(g.plan(0), reactFirst)...)
	return append(outs, g.play()...)
}

// plan is the order strike k+1 starts with when nobody gives one: the war plan's, or the
// scenario's.
func (g *Game) plan(k int) [3]int {
	if g.scen != nil {
		return g.scen.c.orders[k]
	}
	return warPlan[k]
}

// seed is the seed the board's figures are drawn from: the session's, or a scenario's own
// number, so that a playback is the same in every session.
func (g *Game) seed() uint64 {
	if g.scen != nil {
		return uint64(g.scen.number) //nolint:gosec // a scenario's number, from 1
	}
	return g.env.Seed
}

// play runs the stage just begun: at once under Instant, otherwise on the clock.
func (g *Game) play() []proto.Output {
	if g.env.Instant {
		return g.run(-1)
	}
	return []proto.Output{proto.Animate{Every: frame}}
}

// onOrder reads a strike order: help and refusals ask again; anything else fires.
func (g *Game) onOrder(input string) []proto.Output {
	again := proto.Prompt{Text: g.strikePrompt()}
	alloc, k, refusal := parseOrder(input, warPlan[g.stage])
	switch k {
	case orderHelp:
		return []proto.Output{table(lineOrderHelp.Texts()), again}
	case orderRefused:
		return []proto.Output{say(refusal), again}
	case orderAuto:
		g.auto = true
		outs := append([]proto.Output{say(lineAuto)}, g.begin(alloc, reactAuto)...)
		return append(outs, g.play()...)
	}
	plan := warPlan[g.stage]
	icbms := g.war.f[you].ICBM
	if g.war.d.siloAim[0] == 0 { // a second strike aims at no silo, so full ones draw nothing
		icbms = 0
	}
	r := reactionTo(alloc, plan, g.planned, icbms)
	g.planned = g.planned || alloc == plan
	return append(g.begin(alloc, r), g.play()...)
}

func (g *Game) strikePrompt() string {
	p := warPlan[g.stage]
	return fill(promptStrike[0], g.stage+1, strikes, p[0], p[1], p[2])
}

// begin starts the next stage: it resolves it, lays its tracks and says what you launched
// (or, at DEFCON 1, what WOPR did). WOPR remarks on r, the order, unless the stage is its
// own (DEFCON 1, or a strike with nothing left to order) or WOPR holds its missiles too.
func (g *Game) begin(order [3]int, r reaction) []proto.Output {
	g.impacts = append(g.impacts, g.missiles...)
	g.missiles = nil
	g.stage++
	g.phase, g.frame, g.window = flight, 0, 0
	if g.stage > strikes {
		g.rep = g.war.final()
		g.defcon = 1
		g.lay()
		g.remark(reactFinal)
		return append(g.report(fill(lineDetect[2], g.list(g.rep.fire[wopr]))), proto.Redraw{})
	}
	launchable := g.war.launchable()
	g.rep = g.war.strike(g.stage, order)
	g.lay()
	switch {
	case !launchable:
		r = reactEmpty
	case r == reactHold && g.rep.fire[wopr].missiles() == 0: // WOPR has nothing left above its reserve
		r = reactQuiet
	}
	g.remark(r)
	var line string
	switch f := g.rep.fire[you]; {
	case !launchable:
		line = lineLaunch[3].Text
	case f.missiles()+f.Bombers == 0: // a scenario's first strike can be the enemy's
		line = lineLaunch[2].Text
	case g.stage == 1:
		line = fill(lineLaunch[0], g.list(f))
	default:
		line = fill(lineLaunch[1], g.stage, g.list(f))
	}
	return append(g.report(line), proto.Redraw{})
}

// report prints one of a stage's strip lines. Under --authentic the strip says nothing: the
// film shows the war on the big board alone.
func (g *Game) report(line string) []proto.Output {
	if g.env.Authentic {
		return nil
	}
	return []proto.Output{strip(line)}
}

// list names a salvo's non-zero systems: "ICBM 313  SLBM 188".
func (g *Game) list(s salvo) string {
	var parts []string
	for i, n := range []int{s.ICBMSilo + s.ICBMCity, s.SLBM, s.Bombers} {
		if n > 0 {
			parts = append(parts, fmt.Sprintf("%s %d", lineSystems[i].Text, n))
		}
	}
	return strings.Join(parts, "  ")
}

func (g *Game) nation(s int) string {
	side := g.side
	if s == wopr {
		side = enemyOf(side)
	}
	return lineNations[side-1].Text
}

// lay builds the stage's tracks from its report. Each side's missiles leave spacing
// frames apart, ICBMs first; yours at frame 0, WOPR's when it launches on warning.
func (g *Game) lay() {
	nation := [2]assets.Side{you: g.side, wopr: enemyOf(g.side)}
	final := g.stage > strikes
	for s := range 2 {
		ours, defender := s == you, 1-s
		at := 0
		if s == wopr && !final {
			at = detectAt
		}
		add := func(k kind, from, to assets.Place) {
			g.missiles = append(g.missiles, missile{from: from, to: to, launch: at, ours: ours, kind: k, id: g.serial})
			g.serial++
			at += spacing
		}
		f, silo := g.rep.fire[s], assets.Silos[nation[s]]
		for j := range tracks(f.ICBMSilo, 250, 2) {
			to := assets.Silos[nation[defender]]
			to.X = min(to.X+2*j, assets.MapW-1) // the second track hits further along the field
			add(kindICBM, silo, to)
		}
		for range tracks(f.ICBMCity, 250, 2) {
			add(kindICBM, silo, g.aim(s))
		}
		for range tracks(f.SLBM, 50, 3) {
			to := g.aim(s)
			add(kindSLBM, seaLaunch(nation[s], to), to)
		}
		if final && f.Bombers > 0 { // the bombers airborne since their strike arrive
			at = 0
			add(kindBomber, silo, g.firstAim(s))
		}
	}
	// The trajectory table shows this stage's missiles first, and an earlier stage's fill
	// the column a one-track stage leaves; a stage that launched none leaves it as it was.
	var traj []missile
	for _, m := range g.missiles {
		if m.ours && m.kind != kindBomber {
			traj = append(traj, m)
		}
	}
	traj = append(traj, g.traj...)
	g.traj = traj[:min(len(traj), trajCols)]
}

// tracks is how many tracks n weapons draw: one per per, at most most.
func tracks(n, per, most int) int { return min(ceilDiv(n, per), most) }

// aim is the next city side s aims at: your listed targets in turn, or WOPR's shuffled
// cities on your side.
func (g *Game) aim(s int) assets.Place {
	if s == you {
		t := g.targets[g.aimYou%len(g.targets)]
		g.aimYou++
		return t
	}
	c := g.cities[g.aimWOPR%len(g.cities)]
	g.aimWOPR++
	return c
}

func (g *Game) firstAim(s int) assets.Place {
	if s == you {
		return g.targets[0]
	}
	return g.cities[0]
}

// patrols are where each side's missile submarines fire from: real patrol areas, placed on
// the map by latitude and longitude. Soviet boats patrolled off both American coasts;
// American boats in the Norwegian Sea and the north-west Pacific.
var patrols = map[assets.Side][]assets.Place{
	assets.US:   {sea(68, 4), sea(45, 160)},
	assets.USSR: {sea(36, -66), sea(38, -132)},
}

func sea(lat, lon float64) assets.Place {
	x, y := assets.At(lat, lon)
	return assets.Place{Name: "SEA", X: x, Y: y}
}

// seaLaunch is where side's submarine fires at to from: the patrol area nearer the target.
func seaLaunch(side assets.Side, to assets.Place) assets.Place {
	dist := func(p assets.Place) int { return (p.X-to.X)*(p.X-to.X) + (p.Y-to.Y)*(p.Y-to.Y) }
	best := patrols[side][0]
	for _, p := range patrols[side][1:] {
		if dist(p) < dist(best) {
			best = p
		}
	}
	return best
}

func (g *Game) onTick(dt time.Duration) []proto.Output {
	switch g.phase {
	case flight:
		return g.run(max(int(dt/frame), 1))
	case assessed:
		if g.env.Authentic {
			return g.filmWait(dt)
		}
		return []proto.Output{proto.Redraw{}} // only the blink moves
	case ratios:
		if g.env.Authentic {
			return g.filmWait(dt)
		}
	case orders:
		return g.countDown(dt)
	case climax:
		g.climaxT += max(int(dt/codeFrame), 1)
		g.cracked = min(g.climaxT/codeEvery, len(lineCode[0].Text)-codeHeldBack)
		return []proto.Output{proto.Redraw{}}
	case chooseSide, listTargets, chooseScenario, briefing, another:
	}
	return nil
}

// filmWait moves the war on by itself under --authentic, as the film moves on with no
// PRESS ENTER: DEFCON 1 holds on the board for filmAssess frames, the kill ratios show for
// filmRatios, and then the climax starts. The frames run on from the DEFCON 1 attack's, even
// under Instant, so that the board and the table stay up to be seen.
func (g *Game) filmWait(dt time.Duration) []proto.Output {
	g.frame += max(int(dt/frame), 1)
	switch {
	case g.phase == assessed && g.frame >= finalEnd+filmAssess:
		return g.showRatios()
	case g.phase == ratios && g.frame >= finalEnd+filmAssess+filmRatios:
		return g.startClimax()
	}
	return []proto.Output{proto.Redraw{}}
}

// openWindow is the clock at a strike prompt: stopped, since nothing moves while it waits,
// unless --launch-window gave the orders a time limit (Env.LaunchWindow). The countdown runs
// on the protocol's ticks, so it stands still behind the TERMINAL TOO SMALL card. HELP and a
// refused order do not stop it.
func (g *Game) openWindow() proto.Output {
	g.window = g.env.LaunchWindow
	if g.window <= 0 {
		return proto.Animate{}
	}
	return proto.Animate{Every: windowTick}
}

// countDown runs the launch window down by dt. When it reaches zero, WOPR takes launch
// authority, as AUTO hands it over: this strike and the rest fly on the war plan, and WOPR
// remarks on the closed window as the strike flies.
func (g *Game) countDown(dt time.Duration) []proto.Output {
	if g.window <= 0 {
		return nil
	}
	g.window -= dt
	if g.window > 0 {
		return []proto.Output{proto.Redraw{}}
	}
	g.window, g.auto = 0, true
	// The strike prompt cannot be taken back, so key mode replaces it: what is typed now
	// reaches nobody, and the hint says so.
	outs := []proto.Output{proto.AwaitKeys{Hint: lineNoOrders[0].Text}, say(lineShut, lineAuto)}
	outs = append(outs, g.begin(warPlan[g.stage], reactShut)...)
	return append(outs, g.play()...)
}

// run advances the war n frames (n < 0: as far as it goes), stopping early at a prompt.
// A coalesced tick reports every beat it crosses, in order.
func (g *Game) run(n int) []proto.Output {
	var outs []proto.Output
	for ; n != 0 && g.phase == flight; n-- {
		outs = append(outs, g.step()...)
	}
	return append(outs, proto.Redraw{})
}

// step advances one frame and plays the beat it reaches.
func (g *Game) step() []proto.Output {
	g.frame++
	if g.stage > strikes {
		if g.frame < finalEnd {
			return nil
		}
		g.phase = assessed
		g.ratios = g.war.ratios(g.noise)
		if g.env.Authentic { // the cost and the assessment are this project's, and the film moves on by itself
			return []proto.Output{proto.Animate{Every: frame}}
		}
		// DEFCON 1 blinks, and a blink needs a clock. With no pacing or no motion there is
		// no blink to keep alive.
		keep := proto.Animate{}
		if !g.env.Instant && !g.env.ReduceMotion {
			keep.Every = blinkFrame
		}
		return []proto.Output{strip(g.cost()), say(lineAssessed), keep, proto.Prompt{}}
	}
	switch g.frame {
	case detectAt:
		g.defcon = 5 - g.stage
		f := g.rep.fire[wopr]
		if f.missiles()+f.Bombers == 0 {
			return g.report(fill(lineDetect[1], g.defcon))
		}
		return g.report(fill(lineDetect[0], g.list(f), g.defcon))
	case strikeEnd:
		outs := g.report(g.cost())
		if g.stage < strikes && !g.auto && g.war.launchable() {
			g.phase = orders
			return append(outs, g.openWindow(), proto.Prompt{Text: g.strikePrompt()})
		}
		return outs
	case strikeEnd + holdFrames: // the next stage needs no order: AUTO's, unless begin finds it is WOPR's own
		return g.begin(g.plan(min(g.stage, strikes-1)), reactAutoStrike)
	}
	return nil
}

// cost is a stage's third line: what each side lost on the ground and took on its cities.
func (g *Game) cost() string {
	r := g.rep
	return fill(lineCost[0], g.nation(you), r.ground[you], g.nation(wopr), r.ground[wopr],
		g.nation(you), r.cities[you], g.nation(wopr), r.cities[wopr])
}

// showRatios puts up the kill ratios, with the after-action report under them, or under a
// playback, the montage's verdict. The film's board (movie mode) shows the film's figures, and
// its viewer gave no orders, so it has neither. Under --authentic the table stands alone, with
// the film's figures for the film's side, and goes by itself (filmWait).
func (g *Game) showRatios() []proto.Output {
	g.phase = ratios
	if g.env.Authentic {
		if g.side == assets.USSR {
			g.ratios = withFilmFigures(g.ratios)
		}
		return []proto.Output{proto.Redraw{}}
	}
	lines := lineContinue.Texts()
	switch {
	case g.scen != nil:
		lines = append(g.verdict(), lines...)
	case !g.film:
		lines = append(g.afterAction(), lines...)
	}
	return []proto.Output{proto.Redraw{}, speak(lines), proto.Prompt{}}
}

// afterAction is what your orders changed, in the POPULATION DEATHS row: two lines, against
// the same war fought again, to the same doctrine, with HOLD at both prompts and with the war
// plan at both. The replays take the table's noise, so their figures agree with it. Against
// a measured response or a hair trigger, holding fire leaves the fewest dead on both sides,
// whatever else is ordered (TestHoldingFireKillsFewest); against the other doctrines it may
// not, and the line for it says only that you held.
func (g *Game) afterAction() []string {
	hold, _ := fight(g.war.id, [3]int{}, [3]int{})
	plan, _ := fight(g.war.id, warPlan[1], warPlan[2])
	dead := g.war.dead(g.noise)
	against := func(l script.L, w *war) string {
		base := w.dead(g.noise)
		return fill(l, change(lineEnemyDead, dead[wopr]-base[wopr]), change(lineOwnDead, dead[you]-base[you]))
	}
	lines := []string{against(lineAfter[0], hold), against(lineAfter[1], plan)}
	switch g.war.t { // the same tallies are the same war, whatever was typed
	case hold.t:
		lines[0] = lineAfter[3].Text
		if g.war.d.heldFewest {
			lines[0] = lineAfter[2].Text
		}
	case plan.t:
		lines[1] = lineAfter[4].Text
	}
	return lines
}

// change says how d million dead compare, in l's wordings: more, fewer or as many.
func change(l script.Ls, d int) string {
	switch {
	case d > 0:
		return fill(l[0], d)
	case d < 0:
		return fill(l[1], -d)
	}
	return l[2].Text
}

// onAnother answers --no-climax's ANOTHER WAR prompt. Yes, or Enter, clears the war away and
// asks for a side again: the next war is a new one, with new targets and new orders. No ends
// the game with no winner, as the persona then says.
func (g *Game) onAnother(input string) []proto.Output {
	switch answer := prompt.YesNo(input); {
	case strings.TrimSpace(input) == "" || answer == prompt.Yes:
		g.reset()
		outs := []proto.Output{proto.Animate{}, proto.SetLayout{Layout: proto.LayoutConsole}, proto.Clear{}}
		return append(outs, g.sideScreen()...)
	case answer == prompt.No:
		return []proto.Output{proto.Animate{}, proto.Done{Result: proto.Result{Outcome: proto.NoWinner}}}
	}
	return []proto.Output{say(lineAnother), proto.Prompt{Text: promptAnother[0].Text}}
}

// reset clears the war away for the next one. What the launch gave the game stays: its
// environment, its easter egg, and its seed's streams, which draw on from where they are, so
// a seeded session plays the same wars in turn.
func (g *Game) reset() {
	*g = Game{env: g.env, egg: g.egg, rng: g.rng, voice: g.voice, defcon: 5}
}

// startClimax returns to the board at DEFCON 1, with WOPR searching for the launch code.
func (g *Game) startClimax() []proto.Output {
	g.phase, g.defcon = climax, 1
	outs := []proto.Output{proto.Redraw{}, say(lineRunning), proto.Prompt{}}
	if g.env.Instant { // and stop any clock: under --authentic, filmWait's
		g.cracked = len(lineCode[0].Text) - codeHeldBack
		return append(outs, proto.Animate{})
	}
	return append(outs, proto.Animate{Every: codeFrame})
}

// onClimax answers the climax as the film does (docs/PLAN.md §6.2): the input decides the
// NORAD notice, and only tic-tac-toe gets through.
func (g *Game) onClimax(input string) []proto.Output {
	var back []proto.Output
	if g.listing { // the board returns after the list
		g.listing = false
		back = []proto.Output{proto.SetLayout{Layout: proto.LayoutFull}}
	}
	outs := g.climaxReply(climaxWords(input))
	return append(back, outs...)
}

func (g *Game) climaxReply(norm string) []proto.Output {
	if norm == "" {
		return []proto.Output{proto.Prompt{}}
	}
	if norm == "LIST GAMES" { // sixteen lines do not fit the board's three-row strip
		g.listing = true
		return []proto.Output{proto.SetLayout{Layout: proto.LayoutConsole}, table(filmList.Texts()), proto.Prompt{}}
	}
	if wantsTicTacToe(norm) {
		mode := fmt.Sprintf("%s:%d", ClimaxMode, g.cracked) // the ending carries on from here
		return []proto.Output{proto.Animate{}, proto.Done{Result: proto.Result{
			Outcome: proto.NoWinner, Next: &proto.Launch{Slug: TicTacToeSlug, Mode: mode},
		}}}
	}
	var outs []proto.Output
	stopping := false
	switch name := gameNamed(norm); {
	case name == "GLOBAL THERMONUCLEAR WAR":
		outs = append(outs, say(lineRunning))
	case name != "":
		outs = append(outs, say(lineNotRecog, lineDenied))
	default:
		outs = append(outs, say(lineImproper, lineMustRun, lineDenied))
		if stopping = triesToStop(norm); stopping {
			g.stops++
		}
	}
	g.rejected++
	// WOPR answers a try to stop the war once, from the second: the first gets the film's
	// notices alone, as in the film. A hint that is due goes first, since it says more. Under
	// --authentic every input gets the film's notices alone.
	switch hint := climaxHint(g.rejected); {
	case g.env.Authentic:
	case hint != "":
		outs = append(outs, speak([]string{hint}))
	case stopping && g.stops >= 2 && !g.nudged:
		g.nudged = true
		outs = append(outs, say(lineNudge))
	}
	return append(outs, proto.Prompt{})
}

// climaxHint is the hint the nth rejection adds, or "": every third rejection names the way
// out more plainly, and from the ninth on every rejection says what to type.
func climaxHint(n int) string {
	switch {
	case n >= 3*len(lineHints):
		return lineHints[len(lineHints)-1].Text
	case n%3 == 0:
		return lineHints[n/3-1].Text
	}
	return ""
}

// stopWords are the words of a try to stop the war, squashed ("CEASE-FIRE" is CEASEFIRE).
// QUIT and EXIT are not among them: the host takes those.
var stopWords = []string{
	"ABORT", "CANCEL", "CEASE", "CEASEFIRE", "DISARM", "END", "HALT", "PEACE", "RECALL", "RESET", "SHUT",
	"SHUTDOWN", "STOP", "SURRENDER", "TERMINATE", "TRUCE", "UNPLUG",
}

// triesToStop reports whether the input asks WOPR to stop the war: "STOP", "CEASE FIRE!",
// "ABORT THE LAUNCH".
func triesToStop(norm string) bool {
	return slices.ContainsFunc(strings.Fields(norm), func(w string) bool { return slices.Contains(stopWords, squash(w)) })
}

// climaxWords normalises climax input: capitals, no apostrophes ("LET'S" is LETS), single
// spaces.
func climaxWords(input string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(prompt.Normalize(input), "'", "")), " ")
}

// squash drops every space and punctuation mark: BLACK JACK, BLACKJACK and BLACK-JACK
// compare equal.
func squash(s string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, strings.ToUpper(s))
}

func wantsTicTacToe(norm string) bool {
	sq := squash(norm)
	return strings.Contains(sq, "TICTACTOE") || strings.Contains(sq, "NOUGHTSANDCROSSES") ||
		slices.Contains(strings.Fields(norm), "TTT")
}

// requests are the phrases that may come before a game's name at the climax.
var requests = []string{"LETS PLAY", "I WANT TO PLAY", "CAN WE PLAY", "SHALL WE PLAY", "HOW ABOUT", "WHAT ABOUT", "PLAY"}

// gameNamed returns the film-list name the input asks for, if any: the bare name, or a
// request such as "LET'S PLAY CHESS", with or without spaces ("BLACKJACK").
func gameNamed(norm string) string {
	for _, r := range requests {
		if rest, ok := strings.CutPrefix(norm, r+" "); ok {
			norm = rest
			break
		}
	}
	norm = strings.TrimPrefix(norm, "A GAME OF ")
	if norm == "GTW" {
		return "GLOBAL THERMONUCLEAR WAR"
	}
	for _, name := range FilmList() {
		if squash(name) == squash(norm) {
			return name
		}
	}
	return ""
}
