package chess

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	cg "github.com/corentings/chess/v2"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/ai"
	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

var info = games.Info{Name: "CHESS", Slug: "chess", Layout: proto.LayoutPanel, PanelRows: 12}

func position(t *testing.T, fen string) *cg.Position {
	t.Helper()
	opt, err := cg.FEN(fen)
	if err != nil {
		t.Fatal(err)
	}
	return cg.NewGame(opt).Position()
}

// The quality test (docs/PLAN.md §6.1): WOPR finds mates in one and two at the
// deterministic depth.
func TestMatePuzzles(t *testing.T) {
	t.Parallel()
	puzzles := []struct{ name, fen, want string }{
		{"back rank", "6k1/5ppp/8/8/8/8/5PPP/R5K1 w - - 0 1", "a1a8"},
		{"queen and king", "7k/8/6K1/8/8/8/8/1Q6 w - - 0 1", "b1b8"},
		{"black to mate", "2r3k1/8/8/8/8/8/5PPP/6K1 b - - 0 1", "c8c1"},
	}
	for _, p := range puzzles {
		pos := position(t, p.fen)
		m, ok := best(context.Background(), p.fen, nil, ai.Limits{MaxDepth: deterministicDepth}, proto.NewRand(1, 1))
		got := ""
		if ok {
			for _, l := range pos.ValidMoves() {
				if moveOf(l) == m {
					got = (cg.UCINotation{}).Encode(pos, &l)
				}
			}
		}
		if got != p.want {
			t.Errorf("%s: played %q, want %q", p.name, got, p.want)
		}
	}
	// Mate in two with a rook ladder: Ra7 (or Rb7) shuts the seventh rank, then the other
	// rook mates. Two first moves are equally fast, so check the score.
	ladder := position(t, "7k/8/8/8/8/8/R7/1R4K1 w - - 0 1")
	res, ok := ai.Search[move](context.Background(), node{pos: ladder}, ai.Limits{MaxDepth: deterministicDepth}, proto.NewRand(1, 1))
	if !ok || res.Score < ai.Mate-3 {
		t.Errorf("mate in two not found: %+v", res)
	}
}

func TestParseMove(t *testing.T) {
	t.Parallel()
	start := cg.NewGame().Position()
	for in, want := range map[string]string{
		"e2e4": "e2e4", "E2E4": "e2e4", "e4": "e2e4", "Nf3": "g1f3", "nf3": "g1f3", "NF3": "g1f3", "Nf3+": "g1f3",
	} {
		m, ok := ParseMove(start, in)
		if !ok || (cg.UCINotation{}).Encode(start, m) != want {
			t.Errorf("ParseMove(%q) = %v, %v; want %s", in, m, ok, want)
		}
	}
	for _, in := range []string{"", "e5", "e2e5", "Ke2", "hello", "z9z9", "-", "- -", "--+", "-!"} {
		if _, ok := ParseMove(start, in); ok {
			t.Errorf("ParseMove(%q) accepted", in)
		}
	}
	// Coordinates come first, so a capital B there is the b-file; SAN capitals are pieces.
	for in, want := range map[string]string{
		"B1C3": "b1c3", "B2B4": "b2b4", "B3": "b2b3", "e2-e4": "e2e4", "E2-E4": "e2e4", "e2 e4": "e2e4",
		"Ng1-f3": "g1f3", "NG1F3": "g1f3",
	} {
		m, ok := ParseMove(start, in)
		if !ok || (cg.UCINotation{}).Encode(start, m) != want {
			t.Errorf("ParseMove(%q) = %v, %v; want %s", in, m, ok, want)
		}
	}
	bishop := position(t, "4k3/8/2n5/1P6/4B3/8/8/4K3 w - - 0 1")
	for in, want := range map[string]string{"BXC6": "e4c6", "BxC6": "e4c6", "Bxc6": "e4c6", "bxc6": "b5c6"} {
		m, ok := ParseMove(bishop, in)
		if !ok || (cg.UCINotation{}).Encode(bishop, m) != want {
			t.Errorf("ParseMove(%q) = %v, %v; want %s", in, m, ok, want)
		}
	}
	afterE4E5 := position(t, "rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq e6 0 2")
	for in, want := range map[string]string{"BC4": "f1c4", "BB5": "f1b5", "Bc4": "f1c4"} {
		m, ok := ParseMove(afterE4E5, in)
		if !ok || (cg.UCINotation{}).Encode(afterE4E5, m) != want {
			t.Errorf("ParseMove(%q) = %v, %v; want %s", in, m, ok, want)
		}
	}
	promote := position(t, "8/4P3/8/8/8/8/k7/4K3 w - - 0 1")
	for in, want := range map[string]string{
		"e8=Q": "e7e8q", "E8=Q": "e7e8q", "e8=q": "e7e8q", "e8q": "e7e8q", "E8Q": "e7e8q", "e8": "e7e8q",
		"e7e8": "e7e8q", "e7e8q": "e7e8q", "E7E8N": "e7e8n", "e8=N": "e7e8n", "e7-e8=R": "e7e8r",
	} {
		m, ok := ParseMove(promote, in)
		if !ok || (cg.UCINotation{}).Encode(promote, m) != want {
			t.Errorf("ParseMove(%q) = %v, %v; want %s", in, m, ok, want)
		}
	}
	castle := position(t, "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1")
	for _, in := range []string{"O-O", "o-o", "0-0", "e1g1"} {
		if m, ok := ParseMove(castle, in); !ok || m.S2() != cg.G1 {
			t.Errorf("castling %q: %v %v", in, m, ok)
		}
	}
}

// Legality over 20 seeded self-play games at depth 2 (the plan's quality test): every move
// the search returns is legal, and the library's own move validation accepts it.
func TestSelfPlayIsLegal(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("self-play is slow under -short")
	}
	for seed := range uint64(20) {
		t.Run(fmt.Sprint("seed", seed), func(t *testing.T) {
			t.Parallel()
			selfPlay(t, seed)
		})
	}
}

func selfPlay(t *testing.T, seed uint64) {
	g := cg.NewGame()
	for ply := 0; ply < 40 && g.Outcome() == cg.NoOutcome; ply++ {
		m, ok := best(context.Background(), g.FEN(), nil, ai.Limits{MaxDepth: 2}, proto.NewRand(seed, uint64(ply)))
		if !ok {
			t.Fatalf("seed %d ply %d: no move in a live game", seed, ply)
		}
		var chosen *cg.Move
		for _, l := range g.Position().ValidMoves() {
			if moveOf(l) == m {
				chosen = &l
			}
		}
		if chosen == nil {
			t.Fatalf("seed %d ply %d: illegal move %+v", seed, ply, m)
		}
		if err := g.Move(chosen, nil); err != nil {
			t.Fatalf("seed %d ply %d: %v", seed, ply, err)
		}
	}
}

// How long the deterministic search takes over a game's opening and middlegame; the plan
// quotes a median of 27 ms and a p99 of 150 ms. Logged, not asserted: machines differ.
func TestDeterministicSearchTime(t *testing.T) {
	t.Parallel()
	if testing.Short() || raceDetector {
		t.Skip("timing means nothing under -short or -race")
	}
	g := cg.NewGame()
	var times []time.Duration
	for ply := 0; ply < 30 && g.Outcome() == cg.NoOutcome; ply++ {
		start := time.Now()
		m, _ := best(context.Background(), g.FEN(), nil, ai.Limits{MaxDepth: deterministicDepth}, proto.NewRand(3, uint64(ply)))
		times = append(times, time.Since(start))
		for _, l := range g.Position().ValidMoves() {
			if moveOf(l) == m {
				_ = g.Move(&l, nil)
				break
			}
		}
	}
	sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
	t.Logf("depth %d: median %v, max %v over %d searches", deterministicDepth, times[len(times)/2], times[len(times)-1], len(times))
}

// A deterministic transcript that follows WOPR's replies: the test replays them on its own
// game and plays its first legal move each turn.
func TestTranscript(t *testing.T) {
	t.Parallel()
	s := testkit.Game(t, New(), info, "", 5)
	s.Type("e2e5").Type("e4")
	mirror := cg.NewGame()
	_ = mirror.PushNotationMove("e4", cg.AlgebraicNotation{}, nil)
	for turn := 0; turn < 6; turn++ {
		lines := strings.Split(strings.TrimSpace(s.Transcript()), "\n")
		reply := ""
		for i := len(lines) - 1; i >= 0; i-- {
			if r, ok := strings.CutPrefix(lines[i], "WOPR: "); ok {
				reply = r
				break
			}
		}
		if err := mirror.PushNotationMove(strings.ToLower(reply), cg.UCINotation{}, nil); err != nil {
			t.Fatalf("WOPR's %q: %v\n%s", reply, err, s.Transcript())
		}
		if asking, _ := s.Asking(); !asking || mirror.Outcome() != cg.NoOutcome {
			break
		}
		m := mirror.Position().ValidMoves()[0]
		uci := (cg.UCINotation{}).Encode(mirror.Position(), &m)
		_ = mirror.Move(&m, nil)
		s.Type(uci)
	}
	if !s.Contains("ILLEGAL MOVE.") || strings.Count(s.Transcript(), "WOPR: ") < 3 {
		t.Fatalf("transcript:\n%s", s.Transcript())
	}
	if wide := s.Wide(80); len(wide) > 0 {
		t.Errorf("lines wider than 80 columns: %q", wide)
	}
	golden.AssertString(t, "transcript", s.Transcript())
}

func TestResign(t *testing.T) {
	t.Parallel()
	s := testkit.Game(t, New(), info, "", 1)
	s.Type("resign")
	res, over := s.Result()
	if !over || res.Outcome != proto.Loss {
		t.Fatalf("resigning loses: %+v", res)
	}
	if len(res.Lines) < 9 || !strings.Contains(strings.Join(res.Lines, "\n"), "|R:N B:Q K:B N:R |") {
		t.Fatalf("the result carries the final board: %q", res.Lines)
	}
}

func TestEveryLineHasProvenance(t *testing.T) {
	t.Parallel()
	notice, err := os.ReadFile(filepath.Join("..", "..", "..", "NOTICE.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, err := range script.Validate(Lines, string(notice)) {
		t.Error(err)
	}
}

func TestView(t *testing.T) {
	t.Parallel()
	g := New()
	g.Start(proto.Env{Width: 80, Height: info.PanelRows, Deterministic: true})
	c := proto.NewCanvas(80, info.PanelRows)
	g.View(c)
	golden.AssertString(t, "view", c.String()+"\n"+c.StyleMap())
}

// WOPR wins basic endings instead of repeating them away: from king and queen, or king and
// rook, against a bare king, it mates a shuffling player within 50 moves.
func TestWinsBasicEndings(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("endings are slow under -short")
	}
	for _, fen := range []string{
		"8/8/8/4k3/8/8/8/q3K3 b - - 0 1", // WOPR (Black) has K+Q
		"7k/8/8/8/3K4/8/8/r7 b - - 0 1",  // K+R
	} {
		for seed := range uint64(6) {
			opt, err := cg.FEN(fen)
			if err != nil {
				t.Fatal(err)
			}
			g := &Game{g: cg.NewGame(opt), last: cg.NoSquare}
			g.Start(proto.Env{Seed: seed, Deterministic: true})
			r := proto.NewRand(seed, 9)
			mated := false
			for move := 0; move < 50 && !mated; move++ {
				// WOPR moves (it is Black's turn), then the player shuffles a king move.
				outs := g.think()
				th := outs[len(outs)-1].(proto.Think)
				v, _ := th.Fn(context.Background())
				g.onThink(proto.ThinkDone{Value: v})
				if g.g.Outcome() != cg.NoOutcome {
					mated = g.g.Method() == cg.Checkmate
					if !mated {
						t.Fatalf("%s seed %d: the game ended %v after %d moves", fen, seed, g.g.Method(), move)
					}
					break
				}
				legal := g.g.Position().ValidMoves()
				m := legal[r.IntN(len(legal))]
				if err := g.g.Move(&m, nil); err != nil {
					t.Fatal(err)
				}
				if g.g.Outcome() != cg.NoOutcome {
					t.Fatalf("%s seed %d: ended %v after the player's move", fen, seed, g.g.Method())
				}
			}
			if !mated {
				t.Errorf("%s seed %d: no mate in 50 moves (final %s)", fen, seed, g.g.FEN())
			}
		}
	}
}
