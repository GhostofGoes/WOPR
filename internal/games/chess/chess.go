// Package chess is chess against WOPR. The rules, SAN and coordinate notation come from
// github.com/corentings/chess/v2; the search is games/ai's alpha-beta with quiescence.
// The player is White and moves first.
package chess

import (
	"context"
	"slices"
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
var Lines = []script.Ls{
	lineRules, promptMove, lineIllegal, lineWOPR, lineCheck, lineMate, lineStale, lineDraw, lineNoMat, lineResign,
	artTitle, panelWOPR, panelYou, panelSides, panelTaken, panelLast, panelMove, panelCheck,
}

// Game is chess as a proto.Program.
type Game struct {
	env      proto.Env
	g        *cg.Game
	from     cg.Square // the square the last move left, either side's
	last     cg.Square // the square the last move landed on
	lastText string    // the last move in coordinates, as WOPR says its own
	thinking bool
	thinkSeq uint64
}

// New returns a game from the starting position.
func New() games.Game { return &Game{g: cg.NewGame(), from: cg.NoSquare, last: cg.NoSquare} }

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

// ParseMove reads a move against pos: coordinates (e2e4, E2-E4, e2 e4, e7e8q, e7e8=Q, and
// e7e8 alone promotes to a queen) or SAN (Nf3, exd5, O-O, e8=Q). Case is forgiven where it
// cannot mislead: coordinates are read first, so B1C3 is a knight move, and in SAN a
// capital piece letter wins (BXC6 is a bishop capture when one is legal, else the b-pawn's).
func ParseMove(pos *cg.Position, input string) (*cg.Move, bool) {
	s := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(input), "!?+#"))
	if s == "" {
		return nil, false
	}
	if m, ok := coordinates(pos, s); ok {
		return m, true
	}
	var candidates []string
	add := func(c string) {
		if c != "" && !slices.Contains(candidates, c) {
			candidates = append(candidates, c)
		}
	}
	switch {
	case strings.EqualFold(s, "o-o") || s == "0-0":
		add("O-O")
	case strings.EqualFold(s, "o-o-o") || s == "0-0-0":
		add("O-O-O")
	}
	san := strings.NewReplacer("-", "", " ", "").Replace(s) // Ng1-f3
	if san == "" {                                          // only dashes and spaces
		return nil, false
	}
	add(san)
	if strings.ContainsRune("KQRBN", rune(san[0])) {
		add(san[:1] + strings.ToLower(san[1:]))
	}
	add(strings.ToLower(san))
	if strings.ContainsRune("kqrn", rune(san[0])) {
		add(strings.ToUpper(san[:1]) + strings.ToLower(san[1:]))
	}
	for _, c := range slices.Clone(candidates) {
		add(promotion(c))
	}
	for _, c := range candidates {
		for _, n := range []cg.Notation{cg.AlgebraicNotation{}, cg.LongAlgebraicNotation{}} {
			if m, err := n.Decode(pos, c); err == nil && legal(pos, m) {
				return m, true
			}
		}
	}
	return nil, false
}

// coordinates reads from-and-to squares, with optional separators and promotion piece.
func coordinates(pos *cg.Position, s string) (*cg.Move, bool) {
	c := strings.ToLower(strings.NewReplacer("-", "", " ", "", "=", "").Replace(s))
	if len(c) < 4 || len(c) > 5 || c[0] < 'a' || c[0] > 'h' || c[1] < '1' || c[1] > '8' || c[2] < 'a' || c[2] > 'h' || c[3] < '1' || c[3] > '8' {
		return nil, false
	}
	tries := []string{c}
	if len(c) == 4 {
		tries = append(tries, c+"q") // a pawn reaching the last rank with no piece named
	}
	for _, t := range tries {
		if m, err := (cg.UCINotation{}).Decode(pos, t); err == nil && legal(pos, m) {
			return m, true
		}
	}
	return nil, false
}

// promotion spells a pawn's promotion the way SAN wants it: e8q, e8=q and E8Q become
// e8=Q, and a bare e8 becomes e8=Q (the queen).
func promotion(c string) string {
	t := strings.ReplaceAll(c, "=", "")
	n := len(t)
	switch {
	case n >= 3 && strings.ContainsRune("qrbnQRBN", rune(t[n-1])) && (t[n-2] == '8' || t[n-2] == '1'):
		return strings.ToLower(t[:n-1]) + "=" + strings.ToUpper(t[n-1:])
	case n >= 2 && (t[n-1] == '8' || t[n-1] == '1') && t[0] >= 'a' && t[0] <= 'h':
		return strings.ToLower(t) + "=Q"
	}
	return ""
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
	if !ok {
		return []proto.Output{say(lineIllegal), ask()}
	}
	if _, err := g.play(m); err != nil {
		return []proto.Output{say(lineIllegal), ask()}
	}
	if out := g.over(); out != nil {
		return out
	}
	return g.think()
}

// think asks for WOPR's move. Fn gets the position as a FEN string.
func (g *Game) think() []proto.Output {
	g.thinking = true
	g.thinkSeq++
	fen, history, seed, stream := g.g.FEN(), reversible(g.g), g.env.Seed, proto.DomainAI|g.thinkSeq
	limits := ai.Limits{MaxDepth: interactiveDepth}
	if g.env.Deterministic {
		limits.MaxDepth = deterministicDepth
	}
	return []proto.Output{proto.Redraw{}, proto.Think{
		Fn: func(ctx context.Context) (any, error) {
			m, ok := best(ctx, fen, history, limits, proto.NewRand(seed, stream))
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
	text, err := g.play(&chosen)
	if err != nil {
		return []proto.Output{proto.Done{Result: proto.Result{Outcome: proto.Draw}}}
	}
	outs := []proto.Output{proto.Redraw{}, proto.Say{Lines: []string{lineWOPR[0].Text + text}, Pace: proto.PaceSpeech}}
	if over := g.over(); over != nil {
		return append(outs, over...)
	}
	if chosen.HasTag(cg.Check) {
		outs = append(outs, say(lineCheck))
	}
	return append(outs, ask())
}

// play makes m and keeps it as the last move, which the panel shows whichever side made it.
// It returns the move in coordinates, as WOPR says its own.
func (g *Game) play(m *cg.Move) (string, error) {
	text := strings.ToUpper((cg.UCINotation{}).Encode(g.g.Position(), m))
	if err := g.g.Move(m, nil); err != nil {
		return "", err
	}
	g.from, g.last, g.lastText = m.S1(), m.S2(), text
	return text, nil
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

// result carries the final position: the panel goes when the game does. The title card
// stays behind.
func (g *Game) result(o proto.Outcome) proto.Result {
	return proto.Result{Outcome: o, Lines: board.Text(func(c *proto.Canvas) { g.drawBoard(c, 0) }, 80, board.Rows)}
}
