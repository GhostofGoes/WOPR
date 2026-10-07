package ai

import (
	"context"
	"testing"
	"time"

	"github.com/GhostofGoes/WOPR/internal/proto"
)

// nim: take 1 to 3 stones; whoever takes the last stone wins. Leaving a multiple of four
// wins, so the right move is known for every position.
type nim struct{ stones int }

func (n nim) Moves() []int {
	var out []int
	for take := 1; take <= 3 && take <= n.stones; take++ {
		out = append(out, take)
	}
	return out
}

func (n nim) Play(take int) Position[int] { return nim{n.stones - take} }
func (n nim) Eval() int                   { return 0 }

func (n nim) Over() (int, bool) {
	if n.stones == 0 {
		return -Mate, true // the previous player took the last stone
	}
	return 0, false
}

func TestSearchPlaysNimPerfectly(t *testing.T) {
	t.Parallel()
	for stones := 1; stones <= 21; stones++ {
		if stones%4 == 0 {
			continue // a lost position: every move is as good
		}
		res, ok := Search[int](context.Background(), nim{stones}, Limits{MaxDepth: 24}, proto.NewRand(1, 2))
		if !ok || res.Move != stones%4 || (stones > 1 && res.Score < Mate-24) { // one legal move: no search
			t.Errorf("%d stones: took %d (score %d, depth %d), want %d", stones, res.Move, res.Score, res.Depth, stones%4)
		}
	}
}

func TestSearchIsDeterministicUnderLimits(t *testing.T) {
	t.Parallel()
	run := func(seed uint64) Result[int] {
		res, _ := Search[int](context.Background(), nim{40}, Limits{MaxNodes: 5000}, proto.NewRand(seed, 3))
		return res
	}
	if a, b := run(9), run(9); a != b {
		t.Fatalf("same seed, different results: %+v vs %+v", a, b)
	}
	if res := run(9); res.Nodes > 5000+1 || res.Depth == 0 {
		t.Errorf("node limit not honoured or no iteration finished: %+v", res)
	}
	moves := map[int]bool{}
	for seed := range uint64(32) {
		moves[run(seed).Move] = true // 40 stones is lost: every move scores the same
	}
	if len(moves) < 2 {
		t.Errorf("seeds should vary the choice among equal moves, got %v", moves)
	}
}

func TestSearchStopsAtTheDeadline(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	start := time.Now()
	res, ok := Search[int](ctx, nim{1 << 20}, Limits{}, proto.NewRand(1, 1))
	if !ok || res.Move < 1 || res.Move > 3 {
		t.Fatalf("no move after the deadline: %+v", res)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("search ran %v past a 30 ms deadline", elapsed)
	}
}

func TestSearchWithoutMoves(t *testing.T) {
	t.Parallel()
	if _, ok := Search[int](context.Background(), nim{0}, Limits{MaxDepth: 3}, proto.NewRand(1, 1)); ok {
		t.Error("no legal move must report ok=false")
	}
}
