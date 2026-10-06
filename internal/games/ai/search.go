// Package ai is the game-tree search WOPR plays board games with: negamax with alpha-beta
// pruning, iterative deepening, and an optional quiescence search. It runs inside a
// Think (docs/PLAN.md §4.4): with Env.Deterministic the search stops at its Limits, so a
// seed reproduces it exactly; otherwise the context's deadline bounds it and the best
// move of the last finished iteration is played.
package ai

import (
	"context"
	"math/rand/v2"
)

// Mate is the score of a won position; a win found nearer the root scores higher.
const Mate = 1_000_000

// Position is a game state seen from the side to move. Play must return a new position
// and leave the receiver unchanged, because the search keeps both.
type Position[M comparable] interface {
	// Moves lists the legal moves; good moves first makes pruning work better.
	Moves() []M
	// Play returns the position after m.
	Play(m M) Position[M]
	// Eval scores the position for the side to move, without searching.
	Eval() int
	// Over reports whether the game has ended, with the score for the side to move: -Mate
	// for a loss, 0 for a draw.
	Over() (score int, over bool)
}

// Noisy is implemented by positions that want a quiescence search: at the horizon, the
// search keeps playing the noisy moves (captures) until the position is quiet.
type Noisy[M comparable] interface {
	NoisyMoves() []M
}

// Limits bound a search. Zero means no bound; a deterministic search sets at least one.
type Limits struct {
	MaxDepth int    // plies of full-width search
	MaxNodes uint64 // positions visited, across all iterations
}

// Result is a search's answer.
type Result[M comparable] struct {
	Move  M
	Score int    // for the side to move
	Depth int    // the deepest iteration that finished
	Nodes uint64 // positions visited
}

// maxQuiescence bounds the quiescence search, so a long capture chain cannot run away.
const maxQuiescence = 8

type searcher[M comparable] struct {
	ctx     context.Context
	limits  Limits
	nodes   uint64
	stopped bool
}

// Search finds a move for the side to move. Root moves are shuffled with r first, so
// equally good moves vary from game to game while a seed reproduces the choice. ok is
// false only when there is no legal move.
func Search[M comparable](ctx context.Context, p Position[M], limits Limits, r *rand.Rand) (res Result[M], ok bool) {
	moves := p.Moves()
	if len(moves) == 0 {
		return res, false
	}
	r.Shuffle(len(moves), func(i, j int) { moves[i], moves[j] = moves[j], moves[i] })
	res.Move = moves[0]
	if len(moves) == 1 {
		return res, true
	}
	s := &searcher[M]{ctx: ctx, limits: limits}
	maxDepth := limits.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 64 // bounded by the deadline or the node limit instead
	}
	for depth := 1; depth <= maxDepth; depth++ {
		best, score, done := s.root(p, moves, depth)
		if !done {
			break // an unfinished iteration is not trusted; keep the last finished one
		}
		res.Move, res.Score, res.Depth = best, score, depth
		// Search the best move first next time: alpha-beta prunes more.
		for i, m := range moves {
			if m == best {
				copy(moves[1:i+1], moves[:i])
				moves[0] = best
				break
			}
		}
		if score >= Mate-depth || score <= -Mate+depth {
			break // a forced result is found; deeper search cannot change it
		}
	}
	res.Nodes = s.nodes
	return res, true
}

func (s *searcher[M]) root(p Position[M], moves []M, depth int) (best M, score int, done bool) {
	alpha, beta := -2*Mate, 2*Mate
	best = moves[0]
	for _, m := range moves {
		v := -s.negamax(p.Play(m), depth-1, 1, -beta, -alpha)
		if s.stopped {
			return best, alpha, false
		}
		if v > alpha {
			alpha, best = v, m
		}
	}
	return best, alpha, true
}

// stop reports whether the search must end: the node limit or the context.
func (s *searcher[M]) stop() bool {
	if s.stopped {
		return true
	}
	s.nodes++
	if s.limits.MaxNodes > 0 && s.nodes > s.limits.MaxNodes {
		s.stopped = true
	} else if s.nodes%1024 == 0 && s.ctx.Err() != nil {
		s.stopped = true
	}
	return s.stopped
}

func (s *searcher[M]) negamax(p Position[M], depth, ply, alpha, beta int) int {
	if s.stop() {
		return 0
	}
	if score, over := p.Over(); over {
		if score <= -Mate {
			return -Mate + ply // prefer the slowest loss and the fastest win
		}
		return score
	}
	if depth <= 0 {
		return s.quiesce(p, ply, alpha, beta, maxQuiescence)
	}
	moves := p.Moves()
	if len(moves) == 0 {
		return -Mate + ply
	}
	for _, m := range moves {
		v := -s.negamax(p.Play(m), depth-1, ply+1, -beta, -alpha)
		if s.stopped {
			return 0
		}
		if v > alpha {
			alpha = v
		}
		if alpha >= beta {
			break
		}
	}
	return alpha
}

// quiesce extends the search along noisy moves, so the horizon never falls in the middle
// of an exchange. The side to move may always "stand pat" on the static score.
func (s *searcher[M]) quiesce(p Position[M], ply, alpha, beta, left int) int {
	stand := p.Eval()
	n, ok := p.(Noisy[M])
	if !ok || left == 0 {
		return stand
	}
	if stand >= beta {
		return stand
	}
	alpha = max(alpha, stand)
	for _, m := range n.NoisyMoves() {
		if s.stop() {
			return 0
		}
		next := p.Play(m)
		var v int
		if score, over := next.Over(); over {
			v = -score
			if score <= -Mate {
				v = Mate - ply - 1
			}
		} else {
			v = -s.quiesce(next, ply+1, -beta, -alpha, left-1)
		}
		if s.stopped {
			return 0
		}
		if v > alpha {
			alpha = v
		}
		if alpha >= beta {
			break
		}
	}
	return alpha
}
