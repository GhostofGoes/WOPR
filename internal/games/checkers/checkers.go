// Package checkers is 8x8 checkers (English draughts) against WOPR: compulsory captures,
// multiple jumps, kings that move both ways, and a draw after 40 moves each without
// progress. The player is Black and moves first.
package checkers

import (
	"context"
	"strings"
	"time"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/ai"
	"github.com/GhostofGoes/WOPR/internal/games/board"
	"github.com/GhostofGoes/WOPR/internal/prompt"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// Search bounds (docs/PLAN.md §6.1).
const (
	deterministicDepth = 6
	interactiveDepth   = 10
	budget             = 1500 * time.Millisecond
)

// Script text, all original.
var (
	lineRules   = script.Orig("YOU ARE BLACK (B) AND MOVE FIRST. WOPR IS WHITE (W).", "TYPE A MOVE AS C3-D4, OR C3XE5 TO JUMP (C3XE5XG7 TO JUMP TWICE).", "KINGS ARE CAPITALS.")
	promptMove  = script.Orig("YOUR MOVE: ")
	lineIllegal = script.Orig("ILLEGAL MOVE.")
	lineMustJmp = script.Orig("A JUMP IS AVAILABLE. YOU MUST TAKE IT.")
	lineWOPR    = script.Orig("WOPR: ") // followed by WOPR's move
	lineResign  = script.Orig("RESIGNATION ACCEPTED.")
	lineGoesOn  = script.Orig("THE JUMP GOES ON: ") // followed by the ways it can
	lineNoMoves = script.Orig("NO MOVES LEFT FOR BLACK.", "NO MOVES LEFT FOR WHITE.")
	lineNoProg  = script.Orig("DRAWN: 40 MOVES EACH WITHOUT A CAPTURE OR A MAN MOVING.")
)

// Lines is every script block, for the provenance test.
var Lines = []script.Ls{lineRules, promptMove, lineIllegal, lineMustJmp, lineWOPR, lineResign, lineGoesOn, lineNoMoves, lineNoProg}

// Game is checkers as a proto.Program.
type Game struct {
	env      proto.Env
	pos      Position
	last     Move
	moves    int // full moves played
	thinking bool
	thinkSeq uint64
}

// New returns a game.
func New() games.Game { return &Game{pos: NewGame()} }

func say(l script.Ls) proto.Output { return proto.Say{Lines: l.Texts(), Pace: proto.PaceSpeech} }

// Start implements proto.Program.
func (g *Game) Start(env proto.Env) []proto.Output {
	g.env = env
	return []proto.Output{say(lineRules), proto.Prompt{Text: promptMove[0].Text}}
}

// Handle implements proto.Program.
func (g *Game) Handle(ev proto.Event) []proto.Output {
	switch ev := ev.(type) {
	case proto.LineEvent:
		return g.onLine(ev.Text)
	case proto.ThinkDone:
		return g.onThink(ev)
	}
	return nil
}

func (g *Game) onLine(input string) []proto.Output {
	if g.thinking {
		return nil
	}
	if norm := prompt.Normalize(input); norm == "RESIGN" || norm == "I RESIGN" {
		return []proto.Output{say(lineResign), proto.Done{Result: g.result(proto.Loss)}}
	}
	m, ok := g.pos.ParseMove(input)
	if more := g.pos.Continuations(input); !ok && len(more) > 0 {
		ways := make([]string, len(more))
		for i, c := range more {
			ways[i] = strings.ToUpper(c.String())
		}
		return []proto.Output{say(script.Orig(lineGoesOn[0].Text + strings.Join(ways, " OR ") + ".")), proto.Prompt{Text: promptMove[0].Text}}
	}
	if !ok {
		line := lineIllegal
		if g.pos.MustJump() {
			line = lineMustJmp
		}
		return []proto.Output{say(line), proto.Prompt{Text: promptMove[0].Text}}
	}
	g.pos, g.last = g.pos.Play(m), m
	if out := g.over(); out != nil {
		return out
	}
	return g.think()
}

func (g *Game) think() []proto.Output {
	g.thinking = true
	g.thinkSeq++
	pos, seed, stream := g.pos, g.env.Seed, proto.DomainAI|g.thinkSeq
	limits := ai.Limits{MaxDepth: interactiveDepth}
	if g.env.Deterministic {
		limits.MaxDepth = deterministicDepth
	}
	return []proto.Output{proto.Redraw{}, proto.Think{
		Fn: func(ctx context.Context) (any, error) {
			res, ok := ai.Search[Move](ctx, node{pos}, limits, proto.NewRand(seed, stream))
			if !ok {
				return nil, nil
			}
			return res.Move, nil
		},
		Limit:  proto.Limit{MaxDepth: limits.MaxDepth},
		Budget: budget,
	}}
}

func (g *Game) onThink(done proto.ThinkDone) []proto.Output {
	if !g.thinking {
		return nil
	}
	g.thinking = false
	m, ok := done.Value.(Move)
	if !ok || !g.legal(m) {
		legal := g.pos.Legal()
		if len(legal) == 0 {
			return g.over()
		}
		m = legal[0] // never play a move the rules do not allow
	}
	g.pos, g.last = g.pos.Play(m), m
	g.moves++
	outs := []proto.Output{proto.Redraw{}, proto.Say{Lines: []string{lineWOPR[0].Text + strings.ToUpper(m.String())}, Pace: proto.PaceSpeech}}
	if over := g.over(); over != nil {
		return append(outs, over...)
	}
	return append(outs, proto.Prompt{Text: promptMove[0].Text})
}

func (g *Game) legal(m Move) bool {
	for _, l := range g.pos.Legal() {
		if l == m {
			return true
		}
	}
	return false
}

// over ends the game when the side to move cannot move, or on the no-progress draw.
func (g *Game) over() []proto.Output {
	winner, over := g.pos.Result()
	if !over {
		return nil
	}
	outcome, why := proto.Draw, lineNoProg[0].Text
	switch winner {
	case Black:
		outcome, why = proto.Win, lineNoMoves[1].Text // White, to move, cannot
	case White:
		outcome, why = proto.Loss, lineNoMoves[0].Text
	}
	return []proto.Output{proto.Redraw{}, proto.Say{Lines: []string{why}, Pace: proto.PaceSpeech}, proto.Done{Result: g.result(outcome)}}
}

// result carries the final position: the panel goes when the game does.
func (g *Game) result(o proto.Outcome) proto.Result {
	return proto.Result{Outcome: o, Lines: board.Text(g.View, max(g.env.Width, 80), max(g.env.Height, 12))}
}

// View implements proto.Program: dark squares show a dot when empty; men are lower case,
// kings capitals; WOPR's last move ends in bold.
func (g *Game) View(c *proto.Canvas) {
	board.Draw(c, "CHECKERS", func(file, rank int) board.Glyph {
		if (file+rank)%2 != 0 {
			return board.Glyph{}
		}
		sq := at(file, rank)
		var gl board.Glyph
		switch g.pos.sq[sq] {
		case empty:
			return board.Glyph{R: '.', S: proto.StyleDim}
		case blackMan:
			gl = board.Glyph{R: 'b', S: proto.StyleText}
		case blackKing:
			gl = board.Glyph{R: 'B', S: proto.StyleText, A: proto.AttrBold}
		case whiteMan:
			gl = board.Glyph{R: 'w', S: proto.StyleBright}
		case whiteKing:
			gl = board.Glyph{R: 'W', S: proto.StyleBright, A: proto.AttrBold}
		}
		if g.last.n > 0 && sq == g.last.to() {
			gl.A |= proto.AttrUnderline
		}
		return gl
	})
}
