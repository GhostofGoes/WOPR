package chess

import (
	"context"
	"math/rand/v2"
	"slices"

	cg "github.com/corentings/chess/v2"

	"github.com/GhostofGoes/WOPR/internal/games/ai"
)

// move identifies a legal move by its squares and promotion; unlike cg.Move it is
// comparable, which the search needs.
type move struct {
	from, to cg.Square
	promo    cg.PieceType
}

func moveOf(m cg.Move) move { return move{from: m.S1(), to: m.S2(), promo: m.Promo()} }

// node adapts a library position to ai.Position. The position is owned by the search: a
// *cg.Position caches its moves lazily and is not safe to share between goroutines, so
// the game hands the search a FEN string, never its own position.
//
// history holds the hashes of the positions before this one since the last capture or
// pawn move, the game's and the search's own, so that the search sees repetitions: below
// the root, a position that occurred before scores as a draw.
type node struct {
	pos     *cg.Position
	history []uint64
	ply     int
}

var value = map[cg.PieceType]int{cg.Pawn: 100, cg.Knight: 320, cg.Bishop: 330, cg.Rook: 500, cg.Queen: 900}

// Moves orders captures first, most valuable victim and least valuable attacker first,
// then promotions, then the rest.
func (n node) Moves() []move { return n.ordered(false) }

// NoisyMoves are captures and promotions, searched past the horizon.
func (n node) NoisyMoves() []move { return n.ordered(true) }

func (n node) ordered(noisyOnly bool) []move {
	legal := n.pos.ValidMovesUnsafe()
	type scored struct {
		m move
		s int
	}
	out := make([]scored, 0, len(legal))
	b := n.pos.Board()
	for _, m := range legal {
		capture, promo := m.HasTag(cg.Capture), m.Promo() != cg.NoPieceType
		if noisyOnly && !capture && !promo {
			continue
		}
		s := 0
		if capture {
			victim := value[b.Piece(m.S2()).Type()]
			if m.HasTag(cg.EnPassant) {
				victim = value[cg.Pawn]
			}
			s = 10_000 + 10*victim - value[b.Piece(m.S1()).Type()]
		}
		if promo {
			s += value[m.Promo()]
		}
		out = append(out, scored{moveOf(m), s})
	}
	slices.SortStableFunc(out, func(a, b scored) int { return b.s - a.s })
	moves := make([]move, len(out))
	for i, s := range out {
		moves[i] = s.m
	}
	return moves
}

func (n node) Play(m move) ai.Position[move] {
	for _, l := range n.pos.ValidMovesUnsafe() {
		if moveOf(l) == m {
			next := n.pos.Update(&l)
			var history []uint64
			if next.HalfMoveClock() > 0 { // a capture or pawn move makes earlier positions unreachable
				// A fresh slice: a child must never write into its siblings' history.
				history = make([]uint64, len(n.history), len(n.history)+1)
				copy(history, n.history)
				history = append(history, n.pos.ZobristHash())
			}
			return node{pos: next, history: history, ply: n.ply + 1}
		}
	}
	panic("chess: search played a move that is not legal")
}

// Over reports checkmate (a loss for the side to move), stalemate, the fifty-move rule,
// material that cannot mate, and (below the root) a repeated position, as the game
// claims repetition at once.
func (n node) Over() (int, bool) {
	switch n.pos.Status() {
	case cg.Checkmate:
		return -ai.Mate, true
	case cg.Stalemate:
		return 0, true
	}
	if n.pos.HalfMoveClock() >= 100 || cannotMate(n.pos.Board()) {
		return 0, true
	}
	if n.ply > 0 && slices.Contains(n.history, n.pos.ZobristHash()) {
		return 0, true
	}
	return 0, false
}

// cannotMate reports bare kings, or a king and one knight or bishop against a king.
func cannotMate(b *cg.Board) bool {
	minors := 0
	for sq := cg.A1; sq <= cg.H8; sq++ {
		switch b.Piece(sq).Type() {
		case cg.NoPieceType, cg.King:
		case cg.Knight, cg.Bishop:
			minors++
		default:
			return false
		}
	}
	return minors <= 1
}

// Eval is material plus small piece-square bonuses, for the side to move. Against a bare
// king it adds a mop-up term: drive that king to the edge and bring the other one close,
// or the search shuffles a won ending into a draw.
func (n node) Eval() int {
	b := n.pos.Board()
	score := 0
	var material [2]int // non-king material, White and Black
	var kings [2]cg.Square
	queens := false
	for sq := cg.A1; sq <= cg.H8; sq++ {
		p := b.Piece(sq)
		if p == cg.NoPiece {
			continue
		}
		side, sign := 0, 1
		if p.Color() == cg.Black {
			side, sign = 1, -1
		}
		switch p.Type() {
		case cg.King:
			kings[side] = sq // placed below, once the phase is known
			continue
		case cg.Queen:
			queens = true
		}
		material[side] += value[p.Type()]
		score += sign * (value[p.Type()] + placement(p, sq, false))
	}
	endgame := !queens || material[0] == 0 || material[1] == 0 // the kings come out
	score += placement(cg.WhiteKing, kings[0], endgame) - placement(cg.BlackKing, kings[1], endgame)
	switch {
	case material[1] == 0 && material[0] >= value[cg.Rook]:
		score += mopUp(kings[1], kings[0])
	case material[0] == 0 && material[1] >= value[cg.Rook]:
		score -= mopUp(kings[0], kings[1])
	}
	if n.pos.Turn() == cg.Black {
		score = -score
	}
	return score
}

// mopUp scores a bare king's plight: far from the centre, and the other king close by.
func mopUp(lone, strong cg.Square) int {
	lf, lr := int(lone.File()), int(lone.Rank())
	sf, sr := int(strong.File()), int(strong.Rank())
	fromCentre := max(abs(2*lf-7), abs(2*lr-7)) // 1 in the middle, 7 in a corner
	between := abs(lf-sf) + abs(lr-sr)
	return 10*fromCentre + 4*(14-between)
}

// placement rewards central knights and bishops, advanced pawns, and a king that stays
// home in the middlegame and comes to the centre in the endgame (queens off, or a bare king).
func placement(p cg.Piece, sq cg.Square, endgame bool) int {
	f, r := int(sq.File()), int(sq.Rank())
	if p.Color() == cg.Black {
		r = 7 - r
	}
	centre := 3 - max(abs(2*f-7), abs(2*r-7))/2 // 0 on the rim, 3 in the middle
	switch p.Type() {
	case cg.Knight:
		return 10*centre - 15
	case cg.Bishop:
		return 5 * centre
	case cg.Pawn:
		bonus := 5 * r
		if f >= 2 && f <= 5 {
			bonus += 5 * min(r, 3)
		}
		return bonus
	case cg.King:
		if endgame {
			return 5 * centre
		}
		if r == 0 && (f <= 2 || f >= 6) {
			return 20
		}
		return -5 * r
	}
	return 0
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// best searches the position given as FEN; history holds the hashes of the game's earlier
// positions since its last capture or pawn move.
func best(ctx context.Context, fen string, history []uint64, limits ai.Limits, r *rand.Rand) (move, bool) {
	opt, err := cg.FEN(fen)
	if err != nil {
		return move{}, false
	}
	res, ok := ai.Search[move](ctx, node{pos: cg.NewGame(opt).Position(), history: history}, limits, r)
	return res.Move, ok
}

// reversible returns the hashes of the game's positions before the current one, back to
// its last capture or pawn move.
func reversible(g *cg.Game) []uint64 {
	positions := g.Positions()
	if len(positions) == 0 {
		return nil
	}
	cur := positions[len(positions)-1]
	start := max(len(positions)-1-cur.HalfMoveClock(), 0)
	out := make([]uint64, 0, len(positions)-1-start)
	for _, p := range positions[start : len(positions)-1] {
		out = append(out, p.ZobristHash())
	}
	return out
}
