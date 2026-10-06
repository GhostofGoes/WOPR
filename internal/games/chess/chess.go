// Package chess is chess against WOPR. The rules, SAN and coordinate notation come from
// github.com/corentings/chess/v2; the search is games/ai's alpha-beta with quiescence.
// The player is White and moves first.
package chess

import (
	"context"
	"strings"
	"time"

	cg "github.com/corentings/chess/v2"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/ai"
	"github.com/GhostofGoes/WOPR/internal/games/board"
	"github.com/GhostofGoes/WOPR/internal/prompt"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// Search bounds (docs/PLAN.md §6.1).
const (
	deterministicDepth = 3
	interactiveDepth   = 4
	budget             = 1500 * time.Millisecond
)

// Script text, all original.
var (
	lineRules   = script.Orig("YOU ARE WHITE AND MOVE FIRST.", "TYPE A MOVE AS E2E4 OR NF3. RESIGN TO END.")
	promptMove  = script.Orig("YOUR MOVE: ")
	lineIllegal = script.Orig("ILLEGAL MOVE.")
	lineWOPR    = script.Orig("WOPR: ") // followed by WOPR's move
	lineCheck   = script.Orig("CHECK.")
	lineMate    = script.Orig("CHECKMATE.")
	lineStale   = script.Orig("STALEMATE.")
	lineDraw    = script.Orig("DRAWN BY REPETITION OR THE FIFTY-MOVE RULE.")
	lineNoMat   = script.Orig("DRAWN: NEITHER SIDE CAN MATE.")
	lineResign  = script.Orig("RESIGNATION ACCEPTED.")
)

// Lines is every script block, for the provenance test.
var Lines = []script.Ls{lineRules, promptMove, lineIllegal, lineWOPR, lineCheck, lineMate, lineStale, lineDraw, lineNoMat, lineResign}

// Game is chess as a proto.Program.
type Game struct {
	env      proto.Env
	g        *cg.Game
	last     cg.Square // the square WOPR's last move landed on
	thinking bool
	thinkSeq uint64
}

// New returns a game from the starting position.
func New() games.Game { return &Game{g: cg.NewGame(), last: cg.NoSquare} }

func say(l script.Ls) proto.Output { return proto.Say{Lines: l.Texts(), Pace: proto.PaceSpeech} }

func ask() proto.Output { return proto.Prompt{Text: promptMove[0].Text} }

// Start implements proto.Program.
func (g *Game) Start(env proto.Env) []proto.Output {
	g.env = env
	return []proto.Output{say(lineRules), ask()}
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

// ParseMove reads a move in coordinate notation (e2e4, e7e8q) or SAN (Nf3, exd5, O-O),
// forgiving the case of the piece letter, against pos.
func ParseMove(pos *cg.Position, input string) (*cg.Move, bool) {
	s := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(input), "!?+#"))
	if s == "" {
		return nil, false
	}
	candidates := []string{s}
	if l := strings.ToLower(s); l != s {
		candidates = append(candidates, l)
	}
	if strings.ContainsRune("kqrnKQRN", rune(s[0])) {
		candidates = append(candidates, strings.ToUpper(s[:1])+strings.ToLower(s[1:]))
	}
	if strings.EqualFold(s, "o-o") || strings.EqualFold(s, "0-0") {
		candidates = append(candidates, "O-O")
	}
	if strings.EqualFold(s, "o-o-o") || strings.EqualFold(s, "0-0-0") {
		candidates = append(candidates, "O-O-O")
	}
	for _, c := range candidates {
		if m, err := (cg.UCINotation{}).Decode(pos, c); err == nil && legal(pos, m) {
			return m, true
		}
		if m, err := (cg.AlgebraicNotation{}).Decode(pos, c); err == nil && legal(pos, m) {
			return m, true
		}
	}
	return nil, false
}

func legal(pos *cg.Position, m *cg.Move) bool {
	for _, l := range pos.ValidMoves() {
		if moveOf(l) == moveOf(*m) {
			return true
		}
	}
	return false
}

func (g *Game) onLine(input string) []proto.Output {
	if g.thinking {
		return nil
	}
	if norm := prompt.Normalize(input); norm == "RESIGN" || norm == "I RESIGN" {
		return []proto.Output{say(lineResign), proto.Done{Result: g.result(proto.Loss)}}
	}
	m, ok := ParseMove(g.g.Position(), input)
	if !ok || g.g.Move(m, nil) != nil {
		return []proto.Output{say(lineIllegal), ask()}
	}
	g.last = cg.NoSquare
	if out := g.over(); out != nil {
		return out
	}
	return g.think()
}

// think asks for WOPR's move. Fn gets the position as a FEN string.
func (g *Game) think() []proto.Output {
	g.thinking = true
	g.thinkSeq++
	fen, seed, stream := g.g.FEN(), g.env.Seed, proto.DomainAI|g.thinkSeq
	limits := ai.Limits{MaxDepth: interactiveDepth}
	if g.env.Deterministic {
		limits.MaxDepth = deterministicDepth
	}
	return []proto.Output{proto.Redraw{}, proto.Think{
		Fn: func(ctx context.Context) (any, error) {
			m, ok := best(ctx, fen, limits, proto.NewRand(seed, stream))
			if !ok {
				return nil, nil
			}
			return m, nil
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
	pos := g.g.Position()
	legalMoves := pos.ValidMoves()
	if len(legalMoves) == 0 {
		return g.over()
	}
	chosen := legalMoves[0] // never play a move the rules do not allow
	if m, ok := done.Value.(move); ok {
		for _, l := range legalMoves {
			if moveOf(l) == m {
				chosen = l
			}
		}
	}
	text := strings.ToUpper((cg.UCINotation{}).Encode(pos, &chosen))
	if err := g.g.Move(&chosen, nil); err != nil {
		return []proto.Output{proto.Done{Result: proto.Result{Outcome: proto.Draw}}}
	}
	g.last = chosen.S2()
	outs := []proto.Output{proto.Redraw{}, proto.Say{Lines: []string{lineWOPR[0].Text + text}, Pace: proto.PaceSpeech}}
	if over := g.over(); over != nil {
		return append(outs, over...)
	}
	if chosen.HasTag(cg.Check) {
		outs = append(outs, say(lineCheck))
	}
	return append(outs, ask())
}

// over ends the game on mate, stalemate, a claimable draw (claimed at once) or material
// that cannot mate.
func (g *Game) over() []proto.Output {
	for _, method := range g.g.EligibleDraws() {
		if method == cg.ThreefoldRepetition || method == cg.FiftyMoveRule {
			_ = g.g.Draw(method)
		}
	}
	if g.g.Outcome() == cg.NoOutcome {
		return nil
	}
	line, outcome := lineDraw, proto.Draw
	switch g.g.Method() {
	case cg.Checkmate:
		line, outcome = lineMate, proto.Loss
		if g.g.Outcome() == cg.WhiteWon {
			outcome = proto.Win
		}
	case cg.Stalemate:
		line = lineStale
	case cg.InsufficientMaterial:
		line = lineNoMat
	}
	return []proto.Output{proto.Redraw{}, say(line), proto.Done{Result: g.result(outcome)}}
}

// result carries the final position: the panel goes when the game does.
func (g *Game) result(o proto.Outcome) proto.Result {
	return proto.Result{Outcome: o, Lines: board.Text(g.View, max(g.env.Width, 80), max(g.env.Height, 12))}
}

// View implements proto.Program: White in capitals, Black in lower case, a dot on each
// empty square; WOPR's last move is underlined.
func (g *Game) View(c *proto.Canvas) {
	b := g.g.Position().Board()
	board.Draw(c, "CHESS", func(file, rank int) board.Glyph {
		sq := cg.NewSquare(cg.File(file), cg.Rank(rank))
		p := b.Piece(sq)
		if p == cg.NoPiece {
			return board.Glyph{R: '.', S: proto.StyleDim}
		}
		r := rune(strings.ToUpper(p.Type().String())[0])
		gl := board.Glyph{R: r, S: proto.StyleBright}
		if p.Color() == cg.Black {
			gl = board.Glyph{R: r + ('a' - 'A'), S: proto.StyleText}
		}
		if sq == g.last {
			gl.A |= proto.AttrUnderline
		}
		return gl
	})
}
