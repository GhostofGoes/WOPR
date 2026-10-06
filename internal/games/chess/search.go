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
type node struct{ pos *cg.Position }

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
			return node{n.pos.Update(&l)}
		}
	}
	panic("chess: search played a move that is not legal")
}

// Over reports checkmate (a loss for the side to move), stalemate and the fifty-move
// rule. Repetition needs the game's history, so only the game itself checks it.
func (n node) Over() (int, bool) {
	switch n.pos.Status() {
	case cg.Checkmate:
		return -ai.Mate, true
	case cg.Stalemate:
		return 0, true
	}
	if n.pos.HalfMoveClock() >= 100 {
		return 0, true
	}
	return 0, false
}

// Eval is material plus small piece-square bonuses, for the side to move.
func (n node) Eval() int {
	b := n.pos.Board()
	score := 0
	for sq := cg.A1; sq <= cg.H8; sq++ {
		p := b.Piece(sq)
		if p == cg.NoPiece {
			continue
		}
		v := value[p.Type()] + placement(p, sq)
		if p.Color() == cg.Black {
			v = -v
		}
		score += v
	}
	if n.pos.Turn() == cg.Black {
		score = -score
	}
	return score
}

// placement rewards central knights and bishops, advanced pawns and a king that stays
// home while queens are on the board.
func placement(p cg.Piece, sq cg.Square) int {
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

// best searches the position given as FEN.
func best(ctx context.Context, fen string, limits ai.Limits, r *rand.Rand) (move, bool) {
	opt, err := cg.FEN(fen)
	if err != nil {
		return move{}, false
	}
	res, ok := ai.Search[move](ctx, node{cg.NewGame(opt).Position()}, limits, r)
	return res.Move, ok
}
