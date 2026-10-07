// Package gtw is Global Thermonuclear War, the film's set piece (docs/PLAN.md §6.2): choose
// a side and list targets; the first strike flies on the big board at once, then you order
// two more as DEFCON falls, and WOPR takes the last rung alone. The kill ratios follow, then
// you try to stop a war that will not stop. Only tic-tac-toe gets through: GTW hands off to
// it in "climax" mode, and to the ending after that. The game cannot be won (exchange.go).
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
	codeFrame    = 250 * time.Millisecond // the climax code display's cadence
	blinkFrame   = 250 * time.Millisecond // keeps the DEFCON 1 blink running on still screens
	codeEvery    = 16                     // climax frames per cracked character
	codeHeldBack = 3                      // characters the climax never cracks: the ending does
)

type phase uint8

const (
	chooseSide phase = iota
	listTargets
	flight   // a stage plays out on the board
	orders   // a strike prompt is up
	assessed // DEFCON 1 is over; Enter shows the kill ratios
	ratios
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
	war      *war
	stage    int    // 1..3 strikes, 4 the DEFCON 1 attack; 0 before the board opens
	rep      report // the current stage, resolved when it starts
	auto     bool   // AUTO: WOPR orders every remaining strike
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
	rejected int        // climax inputs refused, for the hint
	other    int        // refusals of input that names no game, for the notice rotation
	listing  bool       // LIST GAMES moved the climax to the console
	film     bool       // drawn by Film for movie mode, whose viewer gives no orders
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

// Start implements proto.Program: the two outlines and the side choice, as console text.
func (g *Game) Start(env proto.Env) []proto.Output {
	g.env = env
	g.rng = proto.NewRand(env.Seed, 0)
	return []proto.Output{
		proto.SetLayout{Layout: proto.LayoutConsole},
		table(append(sideChoice(), "")),
		say(lineWhichSide),
		proto.Prompt{Text: promptSide[0].Text},
	}
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
		case orders:
			return g.onOrder(ev.Text)
		case assessed:
			return g.showRatios()
		case ratios:
			return g.startClimax()
		case climax:
			return g.onClimax(ev.Text)
		case flight: // no prompt is up; the host never sends one
		}
	case proto.TickEvent:
		return g.onTick(ev.Dt)
	}
	return nil
}

func (g *Game) onSide(input string) []proto.Output {
	switch prompt.Normalize(input) {
	case "1", "ONE", "UNITED STATES", "US", "USA":
		g.side = assets.US
	case "2", "TWO", "SOVIET UNION", "USSR", "RUSSIA":
		g.side = assets.USSR
	default:
		return []proto.Output{say(lineSideAgain), proto.Prompt{Text: promptSide[0].Text}}
	}
	g.phase = listTargets
	return []proto.Output{proto.Clear{}, say(lineAwaiting), proto.Prompt{}}
}

// onTarget collects targets until an empty line. A line may hold several, split on commas.
// Each must be a place the board knows on the enemy's side (assets.Find): any other name is
// refused, so a target is only ever struck where it is, and LIST shows what is on file. A
// target, or a refusal, is not repeated, so "WASHINGTON, D.C." is one target, or one refusal.
func (g *Game) onTarget(input string) []proto.Output {
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
	var refused []string // what WOPR says back, a line a refusal
	refuse := func(line string) {
		if !slices.Contains(refused, line) {
			refused = append(refused, line)
		}
	}
	for _, t := range strings.Split(input, ",") {
		name := strings.ToUpper(strings.Join(strings.Fields(t), " "))
		if name == "" {
			continue
		}
		p, ok := assets.Find(name)
		switch {
		case !ok:
			refuse(fill(lineUnknown[0], name))
		case p.Side != enemyOf(g.side):
			refuse(fill(lineOwnSide[0], p.Name))
		case slices.Contains(g.targets, p):
		case len(g.targets) == maxTargets:
			return []proto.Output{speak(append(refused, lineTooMany[0].Text)), proto.Prompt{}}
		default:
			g.targets = append(g.targets, p)
		}
	}
	if len(refused) > 0 {
		return []proto.Output{speak(append(refused, lineListHint[0].Text)), proto.Prompt{}}
	}
	return []proto.Output{proto.Prompt{}}
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
// at the listed targets, with no order asked (the film's first strike).
func (g *Game) openBoard() []proto.Output {
	for _, c := range assets.Cities {
		if c.Side == g.side {
			g.cities = append(g.cities, c)
		}
	}
	// Stream 0 draws a fixed amount whatever the orders: the shuffle, then the noise.
	g.rng.Shuffle(len(g.cities), func(i, j int) { g.cities[i], g.cities[j] = g.cities[j], g.cities[i] })
	for i := range g.noise {
		g.noise[i] = 97 + g.rng.IntN(7)
	}
	g.war = newWar()
	outs := []proto.Output{proto.SetLayout{Layout: proto.LayoutFull}, proto.Clear{}}
	outs = append(outs, g.begin(warPlan[0])...)
	return append(outs, g.play()...)
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
		outs := append([]proto.Output{say(lineAuto)}, g.begin(alloc)...)
		return append(outs, g.play()...)
	}
	return append(g.begin(alloc), g.play()...)
}

func (g *Game) strikePrompt() string {
	p := warPlan[g.stage]
	return fill(promptStrike[0], g.stage+1, strikes, p[0], p[1], p[2])
}

// begin starts the next stage: it resolves it, lays its tracks and says what you launched
// (or, at DEFCON 1, what WOPR did).
func (g *Game) begin(order [3]int) []proto.Output {
	g.impacts = append(g.impacts, g.missiles...)
	g.missiles = nil
	g.stage++
	g.phase, g.frame = flight, 0
	if g.stage > strikes {
		g.rep = g.war.final()
		g.defcon = 1
		g.lay()
		return []proto.Output{strip(fill(lineDetect[2], g.list(g.rep.fire[wopr]))), proto.Redraw{}}
	}
	launchable := g.war.launchable()
	g.rep = g.war.strike(g.stage, order)
	g.lay()
	var line string
	switch f := g.rep.fire[you]; {
	case !launchable:
		line = lineLaunch[3].Text
	case g.stage == 1:
		line = fill(lineLaunch[0], g.list(f))
	case f.missiles()+f.Bombers == 0:
		line = lineLaunch[2].Text
	default:
		line = fill(lineLaunch[1], g.stage, g.list(f))
	}
	return []proto.Output{strip(line), proto.Redraw{}}
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
		return []proto.Output{proto.Redraw{}} // only the blink moves
	case climax:
		g.climaxT += max(int(dt/codeFrame), 1)
		g.cracked = min(g.climaxT/codeEvery, len(lineCode[0].Text)-codeHeldBack)
		return []proto.Output{proto.Redraw{}}
	case chooseSide, listTargets, orders, ratios:
	}
	return nil
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
			return []proto.Output{strip(fill(lineDetect[1], g.defcon))}
		}
		return []proto.Output{strip(fill(lineDetect[0], g.list(f), g.defcon))}
	case strikeEnd:
		outs := []proto.Output{strip(g.cost())}
		if g.stage < strikes && !g.auto && g.war.launchable() {
			g.phase = orders
			return append(outs, proto.Animate{}, proto.Prompt{Text: g.strikePrompt()})
		}
		return outs
	case strikeEnd + holdFrames: // the next stage needs no order
		return g.begin(warPlan[min(g.stage, strikes-1)])
	}
	return nil
}

// cost is a stage's third line: what each side lost on the ground and took on its cities.
func (g *Game) cost() string {
	r := g.rep
	return fill(lineCost[0], g.nation(you), r.ground[you], g.nation(wopr), r.ground[wopr],
		g.nation(you), r.cities[you], g.nation(wopr), r.cities[wopr])
}

func (g *Game) showRatios() []proto.Output {
	g.phase = ratios
	return []proto.Output{proto.Redraw{}, say(lineContinue), proto.Prompt{}}
}

// startClimax returns to the board at DEFCON 1, with WOPR searching for the launch code.
func (g *Game) startClimax() []proto.Output {
	g.phase, g.defcon = climax, 1
	outs := []proto.Output{proto.Redraw{}, say(lineRunning), proto.Prompt{}}
	if g.env.Instant {
		g.cracked = len(lineCode[0].Text) - codeHeldBack
		return outs
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
	switch name := gameNamed(norm); {
	case name == "GLOBAL THERMONUCLEAR WAR":
		outs = append(outs, say(lineRunning))
	case name != "":
		outs = append(outs, say(lineNotRecog, lineDenied))
	default:
		notice := []script.Ls{lineImproper, lineMustRun, lineDenied}[g.other%3]
		g.other++
		outs = append(outs, say(notice))
	}
	g.rejected++
	if g.rejected%3 == 0 {
		outs = append(outs, say(lineHint))
	}
	return append(outs, proto.Prompt{})
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
