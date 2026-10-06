// Package gtw is Global Thermonuclear War, the film's set piece (docs/PLAN.md §6.2): choose
// a side, list targets, watch the exchange on the big board and the kill ratios that
// follow, then try to stop a war that will not stop. Only tic-tac-toe gets through: GTW
// hands off to it in "climax" mode, and to the ending after that. The game cannot be won.
package gtw

import (
	"math/rand/v2"
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
	frame         = 100 * time.Millisecond
	launchSpacing = 3  // frames between one side's launches
	flightFrames  = 30 // a missile's flight
	enemyDelay    = 12 // the enemy answers this many frames after the first launch
	lastFrame     = 80
	maxTargets    = 8
	codeFrame     = 250 * time.Millisecond // the climax code display's cadence
	codeEvery     = 16                     // climax frames per cracked character
	codeHeldBack  = 3                      // characters the climax never cracks: the ending does
)

type phase uint8

const (
	chooseSide phase = iota
	listTargets
	exchange
	ratios
	climax
)

// missile is one track on the board.
type missile struct {
	from, to assets.Place
	launch   int  // frame
	ours     bool // the player's (outgoing, '+') or WOPR's (incoming, '*')
}

// Game is GTW as a proto.Program.
type Game struct {
	env      proto.Env
	rng      *rand.Rand
	phase    phase
	side     assets.Side // the player's
	targets  []string
	missiles []missile
	frame    int
	defcon   int
	ratios   [2][12]int // [player, WOPR] values, in row order
	cracked  int        // characters of the launch code shown
	climaxT  int        // climax frames elapsed
	rejected int        // climax inputs refused, for the hint
	other    int        // refusals of input that names no game, for the notice rotation
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

func table(lines []string) proto.Output { return proto.Say{Lines: lines, Pace: proto.PaceTable} }

// Start implements proto.Program: the map and the side choice, as console text.
func (g *Game) Start(env proto.Env) []proto.Output {
	g.env = env
	g.rng = proto.NewRand(env.Seed, 0)
	mapLines := make([]string, 0, len(assets.Map)+2)
	for _, l := range assets.Map {
		mapLines = append(mapLines, "   "+l.Text)
	}
	mapLines = append(mapLines, lineLabels[0].Text, "")
	return []proto.Output{
		proto.SetLayout{Layout: proto.LayoutConsole},
		table(mapLines),
		say(lineWhichSide),
		proto.Prompt{Text: promptSide[0].Text},
	}
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
		case ratios:
			return g.startClimax()
		case climax:
			return g.onClimax(ev.Text)
		case exchange:
			if g.frame >= lastFrame { // the "press Enter" after the exchange
				return g.showRatios()
			}
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
func (g *Game) onTarget(input string) []proto.Output {
	if strings.TrimSpace(input) == "" {
		if len(g.targets) == 0 {
			return []proto.Output{say(lineNoTargets), proto.Prompt{}}
		}
		return g.launch()
	}
	for _, t := range strings.Split(input, ",") {
		if name := strings.ToUpper(strings.Join(strings.Fields(t), " ")); name != "" {
			if len(g.targets) == maxTargets {
				return []proto.Output{say(lineTooMany), proto.Prompt{}}
			}
			g.targets = append(g.targets, name)
		}
	}
	return []proto.Output{proto.Prompt{}}
}

func enemyOf(s assets.Side) assets.Side {
	if s == assets.US {
		return assets.USSR
	}
	return assets.US
}

// launch plans both sides' missiles and starts the exchange on the big board.
func (g *Game) launch() []proto.Output {
	enemy := enemyOf(g.side)
	for i, t := range g.targets {
		g.missiles = append(g.missiles, missile{from: assets.Silos[g.side], to: assets.Locate(t, enemy), launch: i * launchSpacing, ours: true})
	}
	// WOPR answers with more than it received, at cities on the player's side.
	var own []assets.Place
	for _, c := range assets.Cities {
		if c.Side == g.side {
			own = append(own, c)
		}
	}
	g.rng.Shuffle(len(own), func(i, j int) { own[i], own[j] = own[j], own[i] })
	for i := range min(len(g.targets)+2, len(own)) {
		g.missiles = append(g.missiles, missile{from: assets.Silos[enemy], to: own[i], launch: enemyDelay + i*launchSpacing})
	}
	for i := range g.ratios[0] {
		g.ratios[0][i], g.ratios[1][i] = g.ratioValue(i), g.ratioValue(i)
	}
	g.phase, g.frame, g.defcon = exchange, 0, 4
	outs := []proto.Output{proto.SetLayout{Layout: proto.LayoutFull}, proto.Clear{}, say(lineLaunched)}
	if g.env.Instant {
		return append(outs, g.advanceTo(lastFrame)...)
	}
	return append(outs, proto.Animate{Every: frame})
}

// ratioValue draws one kill-ratio figure: unit counts, percentages, then millions (in
// tenths) for the two human-resources rows.
func (g *Game) ratioValue(row int) int {
	switch {
	case row < 5:
		return 20 + g.rng.IntN([5]int{480, 980, 90, 2400, 180}[row])
	case row < 10:
		return 35 + g.rng.IntN(60)
	default:
		return 100 + g.rng.IntN(900)
	}
}

func (g *Game) onTick(dt time.Duration) []proto.Output {
	switch g.phase {
	case exchange:
		n := max(int(dt/frame), 1)
		return g.advanceTo(min(g.frame+n, lastFrame))
	case climax:
		g.climaxT += max(int(dt/codeFrame), 1)
		g.cracked = min(g.climaxT/codeEvery, len(lineCode[0].Text)-codeHeldBack)
		return []proto.Output{proto.Redraw{}}
	}
	return nil
}

// advanceTo moves the exchange to frame f, lowering DEFCON and reporting as it goes.
func (g *Game) advanceTo(f int) []proto.Output {
	var outs []proto.Output
	for g.frame < f {
		g.frame++
		switch g.frame {
		case enemyDelay:
			g.defcon = 3
			outs = append(outs, say(lineDetected))
		case 35:
			g.defcon = 2
		case 55:
			g.defcon = 1
		}
	}
	outs = append(outs, proto.Redraw{})
	if g.frame >= lastFrame {
		outs = append(outs, proto.Animate{}, say(lineAssessed), proto.Prompt{})
	}
	return outs
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
	norm := prompt.Normalize(input)
	if norm == "" {
		return []proto.Output{proto.Prompt{}}
	}
	if norm == "LIST GAMES" {
		return []proto.Output{table(filmList.Texts()), proto.Prompt{}}
	}
	if wantsTicTacToe(norm) {
		return []proto.Output{proto.Animate{}, proto.Done{Result: proto.Result{
			Outcome: proto.NoWinner, Next: &proto.Launch{Slug: TicTacToeSlug, Mode: ClimaxMode},
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

func wantsTicTacToe(norm string) bool {
	return strings.Contains(norm, "TIC TAC TOE") || strings.Contains(norm, "TICTACTOE") ||
		norm == "TTT" || strings.Contains(norm, "NOUGHTS AND CROSSES")
}

// gameNamed returns the film-list name the input asks for, if any: the bare name, or a
// request such as "PLAY CHESS".
func gameNamed(norm string) string {
	norm = strings.TrimPrefix(norm, "PLAY ")
	norm = strings.TrimPrefix(norm, "LETS PLAY ")
	if norm == "GTW" {
		return "GLOBAL THERMONUCLEAR WAR"
	}
	for _, name := range FilmList() {
		if prompt.Normalize(name) == norm {
			return name
		}
	}
	return ""
}
