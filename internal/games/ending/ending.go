// Package ending is the film's climax after zero players (docs/PLAN.md §6.2): WOPR plays
// tic-tac-toe against itself faster and faster while it cracks the last of the launch
// code, runs every war scenario to WINNER: NONE, and concludes that the only winning move
// is not to play. It is an internal program: only a hand-off reaches it, and Esc cannot
// end it.
package ending

import (
	"strconv"
	"strings"
	"time"

	"github.com/GhostofGoes/WOPR/internal/assets"
	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/tictactoe"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// Slug is the ending's registry slug (an internal entry).
const Slug = tictactoe.EndingSlug

// MovieMode is the launch mode movie mode uses, alone or with the launch code characters
// already cracked ("movie:5"): the ending types the film's "Hello." itself (docs/PLAN.md §7).
// A climax hand-off's mode instead says how much of the launch code GTW cracked
// (tictactoe.CodeMode).
const MovieMode = tictactoe.MovieMode

// Timing.
const (
	tick         = 50 * time.Millisecond
	boards       = 7 // tic-tac-toe boards played at once: the big board and three each side
	rounds       = 6 // games each board plays
	firstStep    = 350 * time.Millisecond
	fastestStep  = 60 * time.Millisecond
	speedUp      = 0.75                   // each round's step is this fraction of the last
	steadyStep   = 200 * time.Millisecond // with --reduce-motion: no speeding up, about as long in all
	pauseBetween = 1200 * time.Millisecond

	// The montage moves the whole table, so it redraws under the 3 Hz flash cap
	// (docs/PLAN.md §4.5) and speeds up by adding lines to each frame instead.
	montageFrame   = 400 * time.Millisecond
	framesPerStep  = 3 // frames before each extra line per frame
	mostPerFrame   = 8
	steadyPerFrame = 3 // with --reduce-motion: no acceleration
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
	lineHello     = script.User(script.Reconstructed, "Hello.") // typed by the ending itself in movie mode
)

// Lines is every script block, for the provenance test.
var Lines = []script.Ls{lineGreetings, lineStrange, lineChess, lineStrategy, lineSelfPlay, lineCodeLabel, lineCode, lineHello}

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
	movie bool // movie mode: the ending answers its own greeting

	boards [boards]tictactoe.Board
	toMove [boards]byte
	last   [boards]int // the square each board last played, -1 for none
	round  int
	step   time.Duration
	acc    time.Duration
	games  int // games finished, for the random streams

	shown  int // montage lines on screen
	frames int // montage frames so far
	start  int // launch code characters already cracked when the ending began
}

// New returns the ending.
func New() games.Game { return &Game{} }

// Start implements proto.Program.
func (g *Game) Start(env proto.Env) []proto.Output {
	g.env = env
	g.movie = env.Mode == MovieMode || strings.HasPrefix(env.Mode, MovieMode+":")
	for i := range g.boards {
		g.toMove[i], g.last[i] = 'X', -1
	}
	g.step = firstStep
	if env.ReduceMotion {
		g.step = steadyStep
	}
	g.start = min(len(lineCode[0].Text)-heldBack, len(lineCode[0].Text))
	if _, code, ok := strings.Cut(env.Mode, ":"); ok && (g.movie || strings.HasPrefix(env.Mode, tictactoe.CodeMode)) {
		if n, err := strconv.Atoi(code); err == nil {
			g.start = min(max(n, 0), len(lineCode[0].Text))
		}
	}
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
		switch {
		case g.phase != greeting:
		case strings.TrimSpace(ev.Text) == "": // an Enter pressed to hurry the show is not an answer
			return []proto.Output{proto.Prompt{}}
		default:
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
		for g.acc >= montageFrame && g.phase == montage {
			g.acc -= montageFrame
			g.shown = min(g.shown+g.perFrame(), len(assets.Scenarios))
			g.frames++
			if g.shown >= len(assets.Scenarios) {
				return g.greet()
			}
		}
	case greeting, done:
	}
	return []proto.Output{proto.Redraw{}}
}

// playStep makes one move on every board; a finished board starts its next game, and when
// every board has played its rounds, the montage begins. Each round plays faster than the
// last, except with --reduce-motion, which keeps a steady step.
func (g *Game) playStep() {
	finished := 0
	for i := range g.boards {
		b := g.boards[i]
		if b.Over() {
			finished++
			continue
		}
		r := proto.NewRand(g.env.Seed, proto.DomainAI|uint64(g.games*boards+i))
		sq := tictactoe.Best(b, g.toMove[i], r)
		b[sq] = g.toMove[i]
		g.boards[i], g.toMove[i], g.last[i] = b, tictactoe.Other(g.toMove[i]), sq
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
	if !g.env.ReduceMotion {
		g.step = max(time.Duration(float64(g.step)*speedUp), fastestStep)
	}
	for i := range g.boards {
		g.boards[i], g.toMove[i], g.last[i] = tictactoe.Board{}, 'X', -1
	}
}

// perFrame is how many scenario lines the next montage frame adds: one at first, one more
// every few frames, or a steady few with --reduce-motion.
func (g *Game) perFrame() int {
	if g.env.ReduceMotion {
		return steadyPerFrame
	}
	return min(1+g.frames/framesPerStep, mostPerFrame)
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
	if g.movie {
		outs = append(outs, proto.Say{Lines: []string{""}, Pace: proto.PaceInstant})
		outs = append(outs, proto.Typed("", lineHello[0].Text, proto.NewRand(g.env.Seed, proto.DomainMovie))...)
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

// cracked is how much of the launch code shows: what GTW cracked, then the rest spread over
// the rounds of self-play, complete on the last.
func (g *Game) cracked() int {
	total := len(lineCode[0].Text)
	if g.phase != selfPlay {
		return total
	}
	return min(g.start+(total-g.start)*g.round/(rounds-1), total)
}
