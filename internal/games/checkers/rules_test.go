package checkers

import (
	"context"
	"sort"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games/ai"
	"github.com/GhostofGoes/WOPR/internal/games/board"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// setup builds a position from square names.
func setup(toMove int8, pieces map[string]int8) Position {
	p := Position{toMove: toMove}
	for name, piece := range pieces {
		f, r, ok := board.ParseSquare(name)
		if !ok || (f+r)%2 != 0 {
			panic("bad square " + name)
		}
		p.sq[at(f, r)] = piece
	}
	return p
}

func moveStrings(ms []Move) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.String()
	}
	sort.Strings(out)
	return out
}

func TestOpeningMoves(t *testing.T) {
	t.Parallel()
	got := moveStrings(NewGame().Legal())
	want := []string{"a3-b4", "c3-b4", "c3-d4", "e3-d4", "e3-f4", "g3-f4", "g3-h4"}
	if len(got) != len(want) {
		t.Fatalf("opening moves %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("opening moves %v, want %v", got, want)
		}
	}
}

func TestCapturesAreCompulsoryAndChain(t *testing.T) {
	t.Parallel()
	p := setup(Black, map[string]int8{"c3": blackMan, "a1": blackMan, "d4": whiteMan, "f6": whiteMan, "h8": whiteMan})
	got := moveStrings(p.Legal())
	if len(got) != 1 || got[0] != "c3xe5xg7" {
		t.Fatalf("the double jump is the only legal move, got %v", got)
	}
	next := p.Play(p.Legal()[0])
	if next.sq[at(3, 3)] != empty || next.sq[at(5, 5)] != empty || next.sq[at(6, 6)] != blackMan {
		t.Error("both jumped pieces must go and the man lands on g7")
	}
}

func TestCrowningEndsTheMove(t *testing.T) {
	t.Parallel()
	// The man jumps into the king row at f8; even though a king could jump on, the move ends.
	p := setup(Black, map[string]int8{"d6": blackMan, "e7": whiteMan, "g7": whiteMan})
	got := moveStrings(p.Legal())
	if len(got) != 1 || got[0] != "d6xf8" {
		t.Fatalf("got %v", got)
	}
	if next := p.Play(p.Legal()[0]); next.sq[at(5, 7)] != blackKing || next.noProgress != 0 {
		t.Error("the man must be crowned")
	}
}

func TestKingsMoveBothWaysAndNeverJumpTwice(t *testing.T) {
	t.Parallel()
	p := setup(White, map[string]int8{"d4": whiteKing, "h8": blackMan})
	if got := len(p.Legal()); got != 4 {
		t.Errorf("a lone king in the centre has 4 steps, got %d", got)
	}
	// A king surrounded by four men can jump in a loop, but never the same man twice.
	p = setup(Black, map[string]int8{"d4": blackKing, "c5": whiteMan, "e5": whiteMan, "c3": whiteMan, "e3": whiteMan})
	for _, m := range p.Legal() {
		if m.n > 5 {
			t.Errorf("a jump path longer than four captures: %v", m)
		}
		next := p.Play(m)
		men := 0
		for _, piece := range next.sq {
			if piece == whiteMan {
				men++
			}
		}
		if men != 4-int(m.n-1) {
			t.Errorf("%v captured the wrong number of men", m)
		}
	}
}

func TestParseMove(t *testing.T) {
	t.Parallel()
	p := NewGame()
	for _, in := range []string{"c3-d4", "C3 D4", "c3:d4"} {
		if m, ok := p.ParseMove(in); !ok || m.String() != "c3-d4" {
			t.Errorf("ParseMove(%q) = %v, %v", in, m, ok)
		}
	}
	for _, in := range []string{"c3-c4", "d4-e5", "c3", "z9-a1", ""} {
		if _, ok := p.ParseMove(in); ok {
			t.Errorf("ParseMove(%q) accepted", in)
		}
	}
	j := setup(Black, map[string]int8{"c3": blackMan, "d4": whiteMan, "f6": whiteMan, "h8": whiteMan})
	if m, ok := j.ParseMove("c3xg7"); !ok || m.String() != "c3xe5xg7" {
		t.Errorf("start and end of a unique multi-jump: %v %v", m, ok)
	}
}

func TestGameEnds(t *testing.T) {
	t.Parallel()
	blocked := setup(White, map[string]int8{"a1": whiteMan, "h8": blackMan})
	if w, over := blocked.Result(); !over || w != Black {
		t.Error("a side with no move loses")
	}
	stale := setup(Black, map[string]int8{"a1": blackKing, "h8": whiteKing})
	stale.noProgress = drawPlies
	if w, over := stale.Result(); !over || w != 0 {
		t.Error("40 moves each without progress is a draw")
	}
}

// The quality test: WOPR finds the capture that wins two men rather than one.
func TestSearchPrefersTheDoubleJump(t *testing.T) {
	t.Parallel()
	p := setup(White, map[string]int8{
		"f6": whiteMan, "e5": blackMan, "c3": blackMan, // f6xd4xb2 takes two
		"h6": whiteMan, "g5": blackMan, // h6xf4 takes one
		"h8": whiteMan,
	})
	res, ok := ai.Search[Move](context.Background(), node{p}, ai.Limits{MaxDepth: deterministicDepth}, proto.NewRand(1, 1))
	if !ok || res.Move.String() != "f6xd4xb2" {
		t.Fatalf("chose %v (score %d)", res.Move, res.Score)
	}
}

// Legality over seeded self-play: every move the search returns is legal, and every game
// ends (the no-progress rule guarantees it).
func TestSelfPlayIsLegal(t *testing.T) {
	t.Parallel()
	for seed := range uint64(6) {
		p := NewGame()
		for ply := 0; ; ply++ {
			if _, over := p.Result(); over {
				break
			}
			if ply > 600 {
				t.Fatalf("seed %d: no result after 600 plies", seed)
			}
			res, ok := ai.Search[Move](context.Background(), node{p}, ai.Limits{MaxDepth: 2}, proto.NewRand(seed, uint64(ply)))
			if !ok {
				t.Fatalf("seed %d ply %d: no move but the game is not over", seed, ply)
			}
			legal := false
			for _, m := range p.Legal() {
				legal = legal || m == res.Move
			}
			if !legal {
				t.Fatalf("seed %d ply %d: illegal move %v", seed, ply, res.Move)
			}
			p = p.Play(res.Move)
		}
	}
}
