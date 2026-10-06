// Package ending is the film's climax after zero players (docs/PLAN.md §6.2): WOPR plays
// tic-tac-toe against itself faster and faster while it cracks the last of the launch
// code, runs every war scenario to WINNER: NONE, and concludes that the only winning move
// is not to play. It is an internal program: only a hand-off reaches it, and Esc cannot
// end it.
package ending

import (
	"time"

	"github.com/GhostofGoes/WOPR/internal/assets"
	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/tictactoe"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// Slug is the ending's registry slug (an internal entry).
const Slug = tictactoe.EndingSlug

// MovieMode is the launch mode movie mode uses: the ending types its own "Hello." (M6).
const MovieMode = "movie"

// Timing.
const (
	tick         = 50 * time.Millisecond
	boards       = 10 // tic-tac-toe boards played at once, 5 by 2
	rounds       = 6  // games each board plays
	firstStep    = 350 * time.Millisecond
	fastestStep  = 60 * time.Millisecond
	speedUp      = 0.75 // each round's step is this fraction of the last
	firstLine    = 600 * time.Millisecond
	fastestLine  = 100 * time.Millisecond // the montage scrolls at most 10 lines a second
	lineSpeedUp  = 0.9
	pauseBetween = 1200 * time.Millisecond
)

// Script text (docs/PLAN.md Appendix B).
var (
	lineGreetings = script.Recon("GREETINGS PROFESSOR FALKEN.")
	lineStrange   = script.Recon("A STRANGE GAME.", "THE ONLY WINNING MOVE IS", "NOT TO PLAY.")
	lineChess     = script.Recon("", "HOW ABOUT A NICE GAME OF CHESS?")
	lineStrategy  = script.Recon("STRATEGY:", "WINNER:", "NONE")
	lineSelfPlay  = script.Orig("LEARNING...")
	lineCodeLabel = script.Orig("LAUNCH CODE: ")
	lineCode      = script.Recon("CPE1704TKS")
)

// Lines is every script block, for the provenance test.
var Lines = []script.Ls{lineGreetings, lineStrange, lineChess, lineStrategy, lineSelfPlay, lineCodeLabel, lineCode}

type phase uint8

const (
	selfPlay phase = iota
	montage
	greeting
	done
)

// heldBack is how much of the code GTW's climax left for the ending to crack.
const heldBack = 3

// Game is the ending as a proto.Program.
type Game struct {
	env   proto.Env
	phase phase

	boards [boards]tictactoe.Board
	toMove [boards]byte
	round  int
	step   time.Duration
	acc    time.Duration
	games  int // games finished, for the random streams

	shown    int // montage lines on screen
	lineStep time.Duration
}

// New returns the ending.
func New() games.Game { return &Game{} }

// Start implements proto.Program.
func (g *Game) Start(env proto.Env) []proto.Output {
	g.env = env
	for i := range g.boards {
		g.toMove[i] = 'X'
	}
	g.step, g.lineStep = firstStep, firstLine
	if env.Instant {
		g.phase, g.round, g.shown = montage, rounds, len(assets.Scenarios)
		return g.greet()
	}
	return []proto.Output{
		proto.SetLayout{Layout: proto.LayoutFull},
		proto.Say{Lines: lineSelfPlay.Texts(), Pace: proto.PaceSpeech},
		proto.Animate{Every: tick},
	}
}

// Handle implements proto.Program.
func (g *Game) Handle(ev proto.Event) []proto.Output {
	switch ev := ev.(type) {
	case proto.TickEvent:
		return g.onTick(ev.Dt)
	case proto.LineEvent:
		if g.phase == greeting {
			return g.conclude()
		}
	}
	return nil
}

func (g *Game) onTick(dt time.Duration) []proto.Output {
	g.acc += dt
	switch g.phase {
	case selfPlay:
		for g.acc >= g.step && g.phase == selfPlay {
			g.acc -= g.step
			g.playStep()
		}
	case montage:
		for g.acc >= g.lineStep && g.phase == montage {
			g.acc -= g.lineStep
			g.shown++
			g.lineStep = max(time.Duration(float64(g.lineStep)*lineSpeedUp), fastestLine)
			if g.shown >= len(assets.Scenarios) {
				return g.greet()
			}
		}
	case greeting, done:
	}
	return []proto.Output{proto.Redraw{}}
}

// playStep makes one move on every board; a finished board starts its next game, and when
// every board has played its rounds, the montage begins.
func (g *Game) playStep() {
	finished := 0
	for i := range g.boards {
		b := g.boards[i]
		if b.Over() {
			finished++
			continue
		}
		r := proto.NewRand(g.env.Seed, proto.DomainAI|uint64(g.games*boards+i))
		b[tictactoe.Best(b, g.toMove[i], r)] = g.toMove[i]
		g.boards[i], g.toMove[i] = b, tictactoe.Other(g.toMove[i])
	}
	if finished < boards {
		return
	}
	g.round++
	g.games++
	if g.round >= rounds {
		g.phase, g.acc = montage, 0
		return
	}
	g.step = max(time.Duration(float64(g.step)*speedUp), fastestStep)
	for i := range g.boards {
		g.boards[i], g.toMove[i] = tictactoe.Board{}, 'X'
	}
}

// greet ends the show: the screens go back to the console for the film's last exchange.
func (g *Game) greet() []proto.Output {
	g.phase = greeting
	outs := []proto.Output{
		proto.Animate{},
		proto.SetLayout{Layout: proto.LayoutConsole},
		proto.Clear{},
		proto.Wait{D: pauseBetween},
		proto.Say{Lines: lineGreetings.Texts(), Pace: proto.PaceSpeech},
	}
	if g.env.Mode == MovieMode {
		return append(outs, g.conclude()...)
	}
	return append(outs, proto.Prompt{})
}

func (g *Game) conclude() []proto.Output {
	g.phase = done
	strange := lineStrange.Texts()
	return []proto.Output{
		proto.Say{Lines: strange[:1], Pace: proto.PaceSpeech},
		proto.Wait{D: pauseBetween},
		proto.Say{Lines: strange[1:], Pace: proto.PaceSpeech},
		proto.Wait{D: pauseBetween},
		proto.Say{Lines: lineChess.Texts(), Pace: proto.PaceSpeech},
		proto.Done{Result: proto.Result{Outcome: proto.NoWinner, NoVerdict: true}},
	}
}

// cracked is how much of the launch code shows: GTW's part, then one more character every
// two rounds of self-play.
func (g *Game) cracked() int {
	n := len(lineCode[0].Text) - heldBack + g.round/2
	if g.phase != selfPlay {
		n = len(lineCode[0].Text)
	}
	return min(n, len(lineCode[0].Text))
}
