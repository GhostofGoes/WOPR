package checkers

import (
	"context"
	"sort"
	"strings"
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
// ends (the no-progress rule guarantees it: every reset needs a capture or a man's move,
// and there are only so many of those).
func TestSelfPlayIsLegal(t *testing.T) {
	t.Parallel()
	for seed := range uint64(6) {
		p := NewGame()
		for ply := 0; ; ply++ {
			if _, over := p.Result(); over {
				break
			}
			if ply > 2000 {
				t.Fatalf("seed %d: no result after 2000 plies", seed)
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

// A man that crowns has moved as a man: the no-progress count starts again.
func TestCrowningIsProgress(t *testing.T) {
	t.Parallel()
	p := setup(Black, map[string]int8{"c7": blackMan, "a1": blackKing, "h2": whiteKing})
	p.noProgress = drawPlies - 1
	m, ok := p.ParseMove("c7-d8")
	if !ok {
		t.Fatal("c7-d8 should be legal")
	}
	p = p.Play(m)
	if p.noProgress != 0 {
		t.Fatalf("crowning left the count at %d", p.noProgress)
	}
	if _, over := p.Result(); over {
		t.Fatal("the game must not be drawn on the move that crowned")
	}
}

// The first hop of a multi-jump is enough when only one jump continues it; when several
// do, Continuations lists them.
func TestFirstHopOfAMultiJump(t *testing.T) {
	t.Parallel()
	p := setup(Black, map[string]int8{"c3": blackMan, "a1": blackMan, "d4": whiteMan, "f6": whiteMan, "h8": whiteMan})
	for _, in := range []string{"c3xe5", "C3XE5", "c3-e5", "c3xg7", "c3xe5xg7"} {
		m, ok := p.ParseMove(in)
		if !ok || m.String() != "c3xe5xg7" {
			t.Errorf("%q: %v %v", in, m, ok)
		}
	}
	fork := setup(Black, map[string]int8{"c3": blackMan, "d4": whiteMan, "d6": whiteMan, "f6": whiteMan})
	if _, ok := fork.ParseMove("c3xe5"); ok {
		t.Error("c3xe5 goes on two ways here: not a move by itself")
	}
	if c := fork.Continuations("c3xe5"); len(c) != 2 {
		t.Errorf("continuations: %v", c)
	}
	if c := fork.Continuations("c3xe5xc7"); c != nil {
		t.Errorf("a whole move has no continuations: %v", c)
	}
}

func sayText(outs []proto.Output) string {
	var b strings.Builder
	for _, o := range outs {
		if s, ok := o.(proto.Say); ok {
			b.WriteString(strings.Join(s.Lines, "\n") + "\n")
		}
	}
	return b.String()
}

// The game names the ways a jump goes on, and says why it ended.
func TestGameExplains(t *testing.T) {
	t.Parallel()
	g := New().(*Game)
	g.Start(proto.Env{Seed: 1, Deterministic: true, Instant: true})
	g.pos = setup(Black, map[string]int8{"c3": blackMan, "d4": whiteMan, "d6": whiteMan, "f6": whiteMan})
	if out := sayText(g.Handle(proto.LineEvent{Text: "c3xe5"})); !strings.Contains(out, "THE JUMP GOES ON: C3XE5XC7 OR C3XE5XG7.") &&
		!strings.Contains(out, "THE JUMP GOES ON: C3XE5XG7 OR C3XE5XC7.") {
		t.Errorf("fork: %q", out)
	}
	// Black takes White's last man: White has no move left.
	g.pos = setup(Black, map[string]int8{"c3": blackMan, "d4": whiteMan})
	if out := sayText(g.Handle(proto.LineEvent{Text: "c3xe5"})); !strings.Contains(out, "NO MOVES LEFT FOR WHITE.") {
		t.Errorf("win: %q", out)
	}
}

// Every way of typing a move: separated by '-', 'x', ':' or spaces, or run together, in
// either case. A pair of squares is a step or a jump by its distance, not its separator.
func TestParseMoveForms(t *testing.T) {
	t.Parallel()
	opening := NewGame()
	single := setup(Black, map[string]int8{"c3": blackMan, "d4": whiteMan, "h8": whiteMan})
	double := setup(Black, map[string]int8{"c3": blackMan, "a1": blackMan, "d4": whiteMan, "f6": whiteMan, "h8": whiteMan})
	for _, tc := range []struct {
		pos  Position
		in   string
		want string
	}{
		{opening, "a3b4", "a3-b4"},
		{opening, "A3B4", "a3-b4"},
		{opening, "a3-b4", "a3-b4"},
		{opening, "A3-B4", "a3-b4"},
		{opening, "a3 b4", "a3-b4"},
		{opening, "  a3   b4  ", "a3-b4"},
		{opening, "a3\tb4", "a3-b4"},
		{opening, "a3:b4", "a3-b4"},
		{opening, "a3xb4", "a3-b4"}, // the squares say it is a step
		{single, "c3e5", "c3xe5"},   // the squares say it is a jump
		{single, "C3E5", "c3xe5"},
		{single, "c3xe5", "c3xe5"},
		{single, "C3XE5", "c3xe5"},
		{single, "c3-e5", "c3xe5"},
		{single, "c3 e5", "c3xe5"},
		{double, "c3e5g7", "c3xe5xg7"},
		{double, "C3E5G7", "c3xe5xg7"},
		{double, "c3xe5xg7", "c3xe5xg7"},
		{double, "C3XE5XG7", "c3xe5xg7"},
		{double, "c3 e5 g7", "c3xe5xg7"},
		{double, "c3e5 g7", "c3xe5xg7"},
		{double, "c3e5", "c3xe5xg7"}, // the first hop, when only one jump continues it
		{double, "c3g7", "c3xe5xg7"}, // the start and the end
	} {
		m, ok := tc.pos.ParseMove(tc.in)
		if !ok || m.String() != tc.want {
			t.Errorf("ParseMove(%q) = %v, %v; want %s", tc.in, m, ok, tc.want)
		}
	}
}

func TestParseMoveRejects(t *testing.T) {
	t.Parallel()
	opening := NewGame()
	single := setup(Black, map[string]int8{"c3": blackMan, "d4": whiteMan, "h8": whiteMan})
	for _, tc := range []struct {
		pos      Position
		in       string
		readable bool // it names squares, so the game calls it illegal rather than unreadable
	}{
		{opening, "", false},
		{opening, "a3", false},                         // one square
		{opening, "a3b", false},                        // half a square
		{opening, "a3b4c", false},                      // half a square at the end
		{opening, "a3bb4", false},                      // a stray letter
		{opening, "a3b45", false},                      // a stray digit
		{opening, "i3j4", false},                       // off the board
		{opening, "a0b1", false},                       // off the board
		{opening, "a3,b4", false},                      // not a separator
		{opening, "a3–b4", false},                      // an en dash is not a hyphen
		{opening, "a3 to b4", false},                   // words
		{opening, "a3a3a3a3a3a3a3a3a3a3a3a3a3", false}, // more squares than any move visits
		{opening, "a3a3a3a3a3a3a3", true},              // squares, but no move
		{opening, "c3c4", true},                        // not diagonal
		{opening, "a3b4c5", true},                      // a step goes one square
		{opening, "b4a5", true},                        // no man on b4
		{opening, "c3e5", true},                        // nothing to jump
		{opening, "c3 e5", true},
		{single, "c3d4", true},   // d4 is taken
		{single, "c3b4", true},   // a jump is compulsory
		{single, "c3e5g7", true}, // the jump ends at e5
	} {
		if m, ok := tc.pos.ParseMove(tc.in); ok {
			t.Errorf("ParseMove(%q) accepted %v", tc.in, m)
		}
		if got := readable(tc.in); got != tc.readable {
			t.Errorf("readable(%q) = %v, want %v", tc.in, got, tc.readable)
		}
	}
}

// Every legal move, as WOPR writes it, run together, spaced, or in capitals, reads back as
// that same move, over the positions of a seeded self-play game.
func TestEveryMoveParsesInEveryForm(t *testing.T) {
	t.Parallel()
	compact := strings.NewReplacer("-", "", "x", "")
	spaced := strings.NewReplacer("-", " ", "x", " ")
	p := NewGame()
	for ply := range 200 {
		if _, over := p.Result(); over {
			break
		}
		for _, m := range p.Legal() {
			s := m.String()
			for _, in := range []string{s, strings.ToUpper(s), compact.Replace(s), strings.ToUpper(compact.Replace(s)), spaced.Replace(s)} {
				if got, ok := p.ParseMove(in); !ok || got != m {
					t.Fatalf("ply %d: ParseMove(%q) = %v, %v; want %v", ply, in, got, ok, m)
				}
			}
		}
		res, ok := ai.Search[Move](context.Background(), node{p}, ai.Limits{MaxDepth: 2}, proto.NewRand(7, uint64(ply)))
		if !ok {
			t.Fatalf("ply %d: no move", ply)
		}
		p = p.Play(res.Move)
	}
}

// When a jump forks, its first hop run together is not a move by itself either: the game
// names the ways it goes on.
func TestCompactFirstHopOfAFork(t *testing.T) {
	t.Parallel()
	fork := setup(Black, map[string]int8{"c3": blackMan, "d4": whiteMan, "d6": whiteMan, "f6": whiteMan})
	if _, ok := fork.ParseMove("c3e5"); ok {
		t.Error("c3e5 goes on two ways here: not a move by itself")
	}
	if c := moveStrings(fork.Continuations("C3E5")); len(c) != 2 || c[0] != "c3xe5xc7" || c[1] != "c3xe5xg7" {
		t.Errorf("continuations: %v", c)
	}
	for _, in := range []string{"c3e5c7", "C3E5G7"} {
		if _, ok := fork.ParseMove(in); !ok {
			t.Errorf("%q is a whole move", in)
		}
		if c := fork.Continuations(in); c != nil {
			t.Errorf("%q: a whole move has no continuations: %v", in, c)
		}
	}
}

// The game answers each kind of mistake in its own words: input that is not squares gets
// the format, squares that are not a move are illegal, a step when a jump is due says so,
// and a jump that forks names the ways on. None of them moves a piece.
func TestGameExplainsMistakes(t *testing.T) {
	t.Parallel()
	single := setup(Black, map[string]int8{"c3": blackMan, "d4": whiteMan, "h8": whiteMan})
	fork := setup(Black, map[string]int8{"c3": blackMan, "d4": whiteMan, "d6": whiteMan, "f6": whiteMan})
	for _, tc := range []struct {
		pos  Position
		in   string
		want string
	}{
		{NewGame(), "hello", lineFormat[0].Text},
		{NewGame(), "a3b", lineFormat[0].Text},
		{NewGame(), "c3", lineFormat[0].Text},
		{NewGame(), "c3c4", lineIllegal[0].Text},
		{single, "c3b4", lineMustJmp[0].Text},
		{single, "c3 b4", lineMustJmp[0].Text},
		{fork, "c3e5", "THE JUMP GOES ON: C3XE5X"},
	} {
		g := New().(*Game)
		g.Start(proto.Env{Seed: 1, Deterministic: true, Instant: true})
		g.pos = tc.pos
		if got := sayText(g.Handle(proto.LineEvent{Text: tc.in})); !strings.Contains(got, tc.want) {
			t.Errorf("%q: said %q, want %q", tc.in, got, tc.want)
		}
		if g.thinking || g.pos != tc.pos {
			t.Errorf("%q: a mistake must not move a piece", tc.in)
		}
	}
	// A move run together is played like any other.
	g := New().(*Game)
	g.Start(proto.Env{Seed: 1, Deterministic: true, Instant: true})
	g.Handle(proto.LineEvent{Text: "A3B4"})
	if g.last.String() != "a3-b4" || !g.thinking {
		t.Errorf("A3B4 was not played: last %v, thinking %v", g.last, g.thinking)
	}
}

// No input panics; whatever ParseMove accepts is legal and reads as squares; every
// continuation offered is a legal jump; and a single continuation is accepted as the move.
func FuzzParseMove(f *testing.F) {
	for _, s := range []string{
		"", "a3b4", "A3B4", "c3-d4", "c3 d4", "c3:d4", "c3xe5", "C3XE5XG7", "c3e5g7", "c3e5", "c3g7",
		"a3b", "i3j4", "a3–b4", "xx", "resign", "c3c3c3c3c3c3c3c3c3c3c3c3c3", "\tc3\nd4 ",
	} {
		f.Add(s)
	}
	positions := []Position{
		NewGame(),
		setup(Black, map[string]int8{"c3": blackMan, "a1": blackMan, "d4": whiteMan, "f6": whiteMan, "h8": whiteMan}),
		setup(Black, map[string]int8{"c3": blackMan, "d4": whiteMan, "d6": whiteMan, "f6": whiteMan}),
		setup(Black, map[string]int8{"d4": blackKing, "c5": whiteMan, "e5": whiteMan, "c3": whiteMan, "e3": whiteMan}),
	}
	f.Fuzz(func(t *testing.T, in string) {
		for _, p := range positions {
			legal := p.Legal()
			isLegal := func(m Move) bool {
				for _, l := range legal {
					if l == m {
						return true
					}
				}
				return false
			}
			m, ok := p.ParseMove(in)
			if ok && (!isLegal(m) || !readable(in)) {
				t.Fatalf("ParseMove(%q) accepted %v", in, m)
			}
			more := p.Continuations(in)
			for _, c := range more {
				if !isLegal(c) || !c.jump() {
					t.Fatalf("Continuations(%q) offered %v", in, c)
				}
			}
			if !ok && len(more) == 1 {
				t.Fatalf("%q: the only continuation %v was not accepted", in, more[0])
			}
		}
	})
}
