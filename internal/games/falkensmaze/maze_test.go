package falkensmaze

import (
	"testing"

	"github.com/GhostofGoes/WOPR/internal/proto"
)

func perfect(t *testing.T, m *maze, why string) {
	t.Helper()
	if m.passages() != m.cells()-1 {
		t.Fatalf("%s: %d passages for %d cells", why, m.passages(), m.cells())
	}
	for c, in := range m.component(0) {
		if !in {
			t.Fatalf("%s: cell %d is cut off", why, c)
		}
	}
}

func TestGenerateIsPerfect(t *testing.T) {
	t.Parallel()
	for seed := range uint64(50) {
		m := newMaze(mazeW, mazeH)
		m.generate(proto.NewRand(seed, 0))
		perfect(t, m, "generated")
	}
}

// walls records every wall state around the given cells.
func walls(m *maze, cells []bool) map[[2]int]bool {
	out := map[[2]int]bool{}
	for c, in := range cells {
		if in {
			for d := range 4 {
				out[[2]int{c, d}] = m.open(c, d)
			}
		}
	}
	return out
}

// G-6, after every move over many seeded walks (some following a habit, some random):
// the exit is reachable from the player's cell, the maze stays perfect, and no wall the
// player has seen changes.
func TestRerouteKeepsTheMazeFair(t *testing.T) {
	t.Parallel()
	for seed := range uint64(60) {
		g := New().(*Game)
		g.Start(proto.Env{Seed: seed, Instant: true})
		r := proto.NewRand(seed, 1)
		habit := int(seed % 3) // 0: hug left, 1: straight, 2: random
		for step := 0; step < 400; step++ {
			before := walls(g.m, g.seen)
			seenBefore := append([]bool(nil), g.seen...)
			d := choose(g, habit, r)
			outs := g.Handle(proto.KeyEvent{Key: []proto.Key{proto.KeyUp, proto.KeyRight, proto.KeyDown, proto.KeyLeft}[d]})
			if done(outs) {
				break
			}
			if g.m.path(g.player, g.exit) == nil {
				t.Fatalf("seed %d step %d: the exit is cut off", seed, step)
			}
			perfect(t, g.m, "after a move")
			after := walls(g.m, seenBefore)
			for k, v := range before {
				if after[k] != v {
					t.Fatalf("seed %d step %d: a seen wall changed at cell %d dir %d", seed, step, k[0], k[1])
				}
			}
		}
	}
}

// choose picks the walker's next direction: the left-hand rule, straight when it can, or
// random among open passages.
func choose(g *Game, habit int, r interface{ IntN(int) int }) int {
	var open []int
	for d := range 4 {
		if g.m.open(g.player, d) {
			open = append(open, d)
		}
	}
	switch habit {
	case 0:
		for _, d := range []int{(g.heading + 3) % 4, g.heading, (g.heading + 1) % 4, (g.heading + 2) % 4} {
			if g.m.open(g.player, d) {
				return d
			}
		}
	case 1:
		if g.m.open(g.player, g.heading) {
			return g.heading
		}
	}
	return open[r.IntN(len(open))]
}

func done(outs []proto.Output) bool {
	for _, o := range outs {
		if _, ok := o.(proto.Done); ok {
			return true
		}
	}
	return false
}

// WOPR learns the habit: a left-hand walker is called out as favouring left turns, and
// WOPR moves walls.
func TestWOPRLearnsTheHabit(t *testing.T) {
	t.Parallel()
	g := New().(*Game)
	g.Start(proto.Env{Seed: 4, Instant: true})
	r := proto.NewRand(4, 1)
	var said []string
	for range 300 {
		outs := g.Handle(proto.KeyEvent{Key: []proto.Key{proto.KeyUp, proto.KeyRight, proto.KeyDown, proto.KeyLeft}[choose(g, 0, r)]})
		for _, o := range outs {
			if s, ok := o.(proto.Say); ok {
				said = append(said, s.Lines...)
			}
		}
		if done(outs) {
			break
		}
	}
	if g.favourite() != turnLeft {
		t.Fatalf("bias %v", g.bias)
	}
	found := false
	for _, s := range said {
		found = found || s == "YOU FAVOUR LEFT TURNS, PROFESSOR. SO NOTED."
	}
	if !found || g.reroutes == 0 {
		t.Fatalf("said %q, reroutes %d", said, g.reroutes)
	}
}

func TestWallsAndOtherKeysDoNothing(t *testing.T) {
	t.Parallel()
	g := New().(*Game)
	g.Start(proto.Env{Seed: 1, Instant: true})
	// Cell 0 is the top-left corner: north and west are the border.
	for _, k := range []proto.KeyEvent{{Key: proto.KeyUp}, {Key: proto.KeyLeft}, {Key: proto.KeyRune, Rune: 'z'}, {Key: proto.KeyEnter}} {
		if outs := g.Handle(k); outs != nil || g.player != 0 || g.moves != 0 {
			t.Fatalf("%v moved the player or said something: %v", k, outs)
		}
	}
	outs := g.Handle(proto.KeyEvent{Key: proto.KeyRune, Rune: 'q'})
	if !done(outs) {
		t.Fatal("q gives up")
	}
}

// A deterministic transcript: a player who follows the shortest way to the exit as far as
// they can see it reaches the exit, however WOPR moves the walls.
func TestTranscriptSolver(t *testing.T) {
	t.Parallel()
	for seed := range uint64(20) {
		g := New().(*Game)
		g.Start(proto.Env{Seed: seed, Instant: true})
		won := false
		for step := 0; step < 2000 && !won; step++ {
			path := g.m.path(g.player, g.exit)
			d := g.m.dirTo(path[0], path[1])
			won = done(g.Handle(proto.KeyEvent{Key: []proto.Key{proto.KeyUp, proto.KeyRight, proto.KeyDown, proto.KeyLeft}[d]}))
		}
		if !won {
			t.Fatalf("seed %d: the exit was never reached", seed)
		}
	}
}
