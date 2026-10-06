package tictactoe_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/games/tictactoe"
	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

var info = games.Info{Name: "TIC-TAC-TOE", Slug: "tic-tac-toe", Layout: proto.LayoutPanel, PanelRows: 9}

// The quality test (docs/PLAN.md §6.1): against every possible opponent, and whichever
// optimal move WOPR's seed picks, WOPR never loses, moving first or second.
func TestNeverLoses(t *testing.T) {
	t.Parallel()
	var games, losses int
	var walk func(b tictactoe.Board, toMove, wopr byte)
	walk = func(b tictactoe.Board, toMove, wopr byte) {
		if w := b.Winner(); w != 0 || b.Full() {
			games++
			if w != 0 && w != wopr {
				losses++
				t.Errorf("WOPR (%c) lost:\n%s", wopr, strings.Join(b.Rows(), "\n"))
			}
			return
		}
		moves := allMoves(b)
		if toMove == wopr {
			moves = optimal(b, wopr)
		}
		for _, sq := range moves {
			b[sq] = toMove
			walk(b, tictactoe.Other(toMove), wopr)
			b[sq] = 0
		}
	}
	walk(tictactoe.Board{}, 'X', 'X')
	walk(tictactoe.Board{}, 'X', 'O')
	if games == 0 || losses != 0 {
		t.Fatalf("%d games, %d losses", games, losses)
	}
}

func allMoves(b tictactoe.Board) []int {
	var out []int
	for sq := range b {
		if b[sq] == 0 {
			out = append(out, sq)
		}
	}
	return out
}

// optimal is every move Best may return.
func optimal(b tictactoe.Board, mark byte) []int { return tictactoe.Optimal(b, mark) }

func TestBestTakesTheWinAndBlocks(t *testing.T) {
	t.Parallel()
	b := tictactoe.Board{'O', 'O', 0, 'X', 'X', 0, 0, 0, 0}
	if got := tictactoe.Best(b, 'O', proto.NewRand(1, 1)); got != 2 {
		t.Errorf("O should win at square 3, chose %d", got+1)
	}
	if got := tictactoe.Best(b, 'X', proto.NewRand(1, 1)); got != 5 {
		t.Errorf("X should win at square 6, chose %d", got+1)
	}
	if got := tictactoe.Best(tictactoe.Board{'X', 'X', 'O', 'O', 'O', 'X', 'X', 'O', 'X'}, 'X', nil); got != -1 {
		t.Errorf("a full board has no move, got %d", got)
	}
}

func TestOnePlayerGame(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, tictactoe.New(), info, "", 7)
	g.Type("1")
	for _, sq := range []string{"5", "1", "3", "4", "6", "7", "8", "9", "2"} {
		if _, over := g.Result(); over {
			break
		}
		if asking, _ := g.Asking(); !asking {
			t.Fatalf("no prompt:\n%s", g.Transcript())
		}
		before := g.Transcript()
		g.Type(sq)
		if strings.Contains(g.Transcript()[len(before):], "CHOOSE AN EMPTY SQUARE") {
			continue // taken: the transcript records the refusal
		}
	}
	res, over := g.Result()
	if !over || res.Outcome == proto.Win || len(res.Lines) != 5 {
		t.Fatalf("result %+v over=%v:\n%s", res, over, g.Transcript())
	}
	golden.AssertString(t, "one_player", g.Transcript()+"\n"+strings.Join(res.Lines, "\n")+"\n")
}

func TestTwoPlayersHotseat(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, tictactoe.New(), info, "", 1)
	g.Type("2").Type("A1").Type("4").Type("B1").Type("5") // coordinates: A1 is square 1, B1 square 2
	res, over := g.Result()
	if over {
		t.Fatalf("game ended early: %+v\n%s", res, g.Transcript())
	}
	g.Type("C1")
	res, over = g.Result()
	if !over || res.Outcome != proto.Win {
		t.Fatalf("X wins the top row: %+v\n%s", res, g.Transcript())
	}
}

func TestZeroPlayersHandsOffToTheEnding(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"", tictactoe.Climax} {
		g := testkit.Game(t, tictactoe.New(), info, mode, 1)
		g.Type("zero")
		res, over := g.Result()
		launches := g.Launches()
		if !over || res.Outcome != proto.NoWinner || len(launches) != 2 || launches[1] != tictactoe.EndingSlug {
			t.Errorf("mode %q: result %+v launches %v", mode, res, launches)
		}
	}
}

// The climax is a closed state machine (docs/PLAN.md §6.2).
func TestClimax(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, tictactoe.New(), info, tictactoe.Climax, 3)
	g.Type("2").Type("three").Type("1") // 2 is refused at the climax
	if strings.Count(g.Transcript(), "PLEASE LIST NUMBER OF PLAYERS:") < 3 {
		t.Fatalf("2 and nonsense must re-ask:\n%s", g.Transcript())
	}
	for _, sq := range []string{"5", "1", "2", "3", "4", "6", "7", "8", "9"} {
		if asking, p := g.Asking(); !asking || p != "YOUR MOVE: " {
			break
		}
		g.Type(sq)
	}
	if !g.Contains("STALEMATE. WANT TO PLAY AGAIN?") && !g.Contains("WINNER: WOPR.") {
		t.Fatalf("the game must end with the film's question:\n%s", g.Transcript())
	}
	g.Type("maybe").Type("no")
	if asking, p := g.Asking(); !asking || p != "PLEASE LIST NUMBER OF PLAYERS: " {
		t.Fatalf("NO goes back to the number of players: %q\n%s", p, g.Transcript())
	}
	g.Type("1").Type("1").Type("9").Type("3").Type("7").Type("4").Type("6").Type("2").Type("8")
	if _, over := g.Result(); over {
		t.Fatalf("the climax never reports a verdict:\n%s", g.Transcript())
	}
	g.Type("0")
	res, over := g.Result()
	if !over || g.Launches()[len(g.Launches())-1] != tictactoe.EndingSlug || res.Outcome != proto.NoWinner {
		t.Fatalf("0 at WANT TO PLAY AGAIN? goes to the ending: %+v %v", res, g.Launches())
	}
	golden.AssertString(t, "climax", g.Transcript())
}

func TestEveryLineHasProvenance(t *testing.T) {
	t.Parallel()
	notice, err := os.ReadFile(filepath.Join("..", "..", "..", "NOTICE.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, err := range script.Validate(tictactoe.Lines, string(notice)) {
		t.Error(err)
	}
}

func TestViewFitsItsPanel(t *testing.T) {
	t.Parallel()
	g := tictactoe.New()
	g.Start(proto.Env{Width: 80, Height: info.PanelRows})
	g.Handle(proto.LineEvent{Text: "2"})
	g.Handle(proto.LineEvent{Text: "5"})
	c := proto.NewCanvas(80, info.PanelRows)
	g.View(c)
	golden.AssertString(t, "view", c.String()+"\n"+c.StyleMap())
}
