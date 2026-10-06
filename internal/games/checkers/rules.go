package checkers

import (
	"strings"

	"github.com/GhostofGoes/WOPR/internal/games/ai"
	"github.com/GhostofGoes/WOPR/internal/games/board"
)

// Pieces. Black moves up the board (towards rank 8) and moves first; White moves down.
const (
	empty     int8 = 0
	blackMan  int8 = 1
	blackKing int8 = 2
	whiteMan  int8 = -1
	whiteKing int8 = -2
)

// Sides.
const (
	Black int8 = 1
	White int8 = -1
)

// drawPlies is the no-progress rule: 40 moves each without a capture or a man moving is
// a draw.
const drawPlies = 80

// Position is a checkers position: squares indexed rank*8+file, a1 = 0. Only dark squares
// ((file+rank) even) are used. It is a value, so the search can copy it freely.
type Position struct {
	sq         [64]int8
	toMove     int8
	noProgress int // plies since a capture or a man's move
}

// Move is a path of squares: the start, then each landing square. A move with more than
// two squares, or whose step is two ranks, is a capture.
type Move struct {
	path [12]int8
	n    int8
}

func (m Move) from() int  { return int(m.path[0]) }
func (m Move) to() int    { return int(m.path[m.n-1]) }
func (m Move) jump() bool { return abs(rankOf(int(m.path[1]))-rankOf(int(m.path[0]))) == 2 }

func (m Move) add(sq int) Move {
	m.path[m.n] = int8(sq)
	m.n++
	return m
}

// String writes the move as squares joined by '-' (a step) or 'x' (jumps): "c3-d4",
// "c3xe5xg7".
func (m Move) String() string {
	sep := "-"
	if m.jump() {
		sep = "x"
	}
	parts := make([]string, m.n)
	for i := range m.n {
		parts[i] = name(int(m.path[i]))
	}
	return strings.Join(parts, sep)
}

func fileOf(sq int) int { return sq % 8 }
func rankOf(sq int) int { return sq / 8 }
func at(file, rank int) int {
	if file < 0 || file > 7 || rank < 0 || rank > 7 {
		return -1
	}
	return rank*8 + file
}
func name(sq int) string { return board.SquareName(fileOf(sq), rankOf(sq)) }

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func sideOf(p int8) int8 {
	switch {
	case p > 0:
		return Black
	case p < 0:
		return White
	}
	return 0
}

func isKing(p int8) bool { return p == blackKing || p == whiteKing }

// NewGame returns the starting position: Black on ranks 1-3, White on ranks 6-8.
func NewGame() Position {
	var p Position
	for sq := range 64 {
		if (fileOf(sq)+rankOf(sq))%2 != 0 {
			continue
		}
		switch r := rankOf(sq); {
		case r <= 2:
			p.sq[sq] = blackMan
		case r >= 5:
			p.sq[sq] = whiteMan
		}
	}
	p.toMove = Black
	return p
}

// directions a piece may move or jump in, as (file, rank) steps.
func directions(piece int8) [][2]int {
	switch piece {
	case blackMan:
		return [][2]int{{-1, 1}, {1, 1}}
	case whiteMan:
		return [][2]int{{-1, -1}, {1, -1}}
	default:
		return [][2]int{{-1, 1}, {1, 1}, {-1, -1}, {1, -1}}
	}
}

func crowns(piece int8, sq int) bool {
	return (piece == blackMan && rankOf(sq) == 7) || (piece == whiteMan && rankOf(sq) == 0)
}

// Legal returns the legal moves for the side to move. Captures are compulsory: when any
// jump exists, only jumps are legal, and a jumping piece must keep jumping while it can.
// A man that reaches the far rank is crowned and its move ends there.
func (p Position) Legal() []Move {
	var jumps, steps []Move
	for sq := range 64 {
		piece := p.sq[sq]
		if sideOf(piece) != p.toMove {
			continue
		}
		jumps = p.jumpsFrom(sq, piece, Move{}.add(sq), [64]bool{}, jumps)
		if len(jumps) > 0 {
			continue
		}
		for _, d := range directions(piece) {
			if to := at(fileOf(sq)+d[0], rankOf(sq)+d[1]); to >= 0 && p.sq[to] == empty {
				steps = append(steps, Move{}.add(sq).add(to))
			}
		}
	}
	if len(jumps) > 0 {
		return jumps
	}
	return steps
}

// jumpsFrom extends a jump sequence from sq. Captured pieces stay on the board until the
// move ends, so they can be neither jumped twice nor landed on; the moving piece's start
// square counts as empty.
func (p Position) jumpsFrom(sq int, piece int8, path Move, taken [64]bool, out []Move) []Move {
	extended := false
	for _, d := range directions(piece) {
		over := at(fileOf(sq)+d[0], rankOf(sq)+d[1])
		land := at(fileOf(sq)+2*d[0], rankOf(sq)+2*d[1])
		if over < 0 || land < 0 || taken[over] || sideOf(p.sq[over]) != -sideOf(piece) {
			continue
		}
		if p.sq[land] != empty && land != path.from() {
			continue
		}
		if int(path.n) >= len(path.path) {
			continue
		}
		extended = true
		next := path.add(land)
		t := taken
		t[over] = true
		if crowns(piece, land) {
			out = append(out, next)
			continue
		}
		out = p.jumpsFrom(land, piece, next, t, out)
	}
	if !extended && path.n > 1 {
		out = append(out, path)
	}
	return out
}

// Play returns the position after m, which must be legal.
func (p Position) Play(m Move) Position {
	piece := p.sq[m.from()]
	p.sq[m.from()] = empty
	captured := false
	for i := 1; i < int(m.n); i++ {
		a, b := int(m.path[i-1]), int(m.path[i])
		if abs(rankOf(b)-rankOf(a)) == 2 {
			p.sq[at((fileOf(a)+fileOf(b))/2, (rankOf(a)+rankOf(b))/2)] = empty
			captured = true
		}
	}
	if crowns(piece, m.to()) {
		piece *= 2
	}
	p.sq[m.to()] = piece
	if captured || !isKing(piece) {
		p.noProgress = 0
	} else {
		p.noProgress++
	}
	p.toMove = -p.toMove
	return p
}

// Result reports whether the game is over and, if so, the winner (0 for a draw). A side
// with no legal move has lost.
func (p Position) Result() (winner int8, over bool) {
	if len(p.Legal()) == 0 {
		return -p.toMove, true
	}
	if p.noProgress >= drawPlies {
		return 0, true
	}
	return 0, false
}

// ParseMove matches what the player typed against the legal moves: "c3-d4", "c3 d4",
// "c3xe5xg7", or just the start and end of a multi-jump ("c3xg7") when that is unique.
func (p Position) ParseMove(input string) (Move, bool) {
	f := strings.FieldsFunc(strings.ToLower(input), func(r rune) bool { return r == '-' || r == 'x' || r == ' ' || r == ':' })
	if len(f) < 2 {
		return Move{}, false
	}
	squares := make([]int, len(f))
	for i, s := range f {
		file, rank, ok := board.ParseSquare(s)
		if !ok {
			return Move{}, false
		}
		squares[i] = at(file, rank)
	}
	var match []Move
	for _, m := range p.Legal() {
		if m.from() != squares[0] || m.to() != squares[len(squares)-1] {
			continue
		}
		if len(squares) == int(m.n) {
			same := true
			for i, sq := range squares {
				same = same && int(m.path[i]) == sq
			}
			if same {
				return m, true
			}
		}
		if len(squares) == 2 {
			match = append(match, m)
		}
	}
	if len(match) == 1 {
		return match[0], true
	}
	return Move{}, false
}

// MustJump reports whether the side to move has a capture available.
func (p Position) MustJump() bool {
	l := p.Legal()
	return len(l) > 0 && l[0].jump()
}

// --- search ---

// node adapts Position to ai.Position.
type node struct{ Position }

func (n node) Moves() []Move {
	moves := n.Legal()
	// Crowning moves first, then the rest: a cheap ordering that helps pruning.
	ordered := moves[:0:0]
	for _, m := range moves {
		if crowns(n.sq[m.from()], m.to()) {
			ordered = append(ordered, m)
		}
	}
	for _, m := range moves {
		if !crowns(n.sq[m.from()], m.to()) {
			ordered = append(ordered, m)
		}
	}
	return ordered
}

func (n node) Play(m Move) ai.Position[Move] { return node{n.Position.Play(m)} }

func (n node) Over() (int, bool) {
	winner, over := n.Result()
	switch {
	case !over:
		return 0, false
	case winner == 0:
		return 0, true
	case winner == n.toMove:
		return ai.Mate, true
	default:
		return -ai.Mate, true
	}
}

// Eval scores the position for the side to move: material, men's advance, guarding the
// back rank, and the centre.
func (n node) Eval() int {
	score := 0
	for sq, piece := range n.sq {
		if piece == empty {
			continue
		}
		v := 0
		r, f := rankOf(sq), fileOf(sq)
		switch piece {
		case blackMan, whiteMan:
			v = 100
			advance := r
			if piece == whiteMan {
				advance = 7 - r
			}
			v += 3 * advance
			if advance == 0 {
				v += 6 // a man on its own back rank keeps the other side from crowning
			}
		default:
			v = 160
		}
		if f >= 2 && f <= 5 && r >= 2 && r <= 5 {
			v += 4
		}
		score += int(sideOf(piece)) * v
	}
	return score * int(n.toMove)
}
