package ending

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GhostofGoes/WOPR/internal/assets"
	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/games/tictactoe"
	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

var info = games.Info{Name: "ENDING", Slug: Slug, Layout: proto.LayoutFull, Internal: true}

// The film's last exchange, as the player sees it after zero players.
func TestFinalDialogue(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, New(), info, "", 1)
	if asking, _ := g.Asking(); !asking || !g.Contains("GREETINGS PROFESSOR FALKEN.") {
		t.Fatalf("the ending greets and waits:\n%s", g.Transcript())
	}
	g.Type("Hello.")
	res, over := g.Result()
	if !over || !res.NoVerdict || res.Outcome != proto.NoWinner {
		t.Fatalf("result %+v", res)
	}
	golden.AssertString(t, "final_dialogue", g.Transcript())
}

func TestMovieModeTypesItsOwnHello(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, New(), info, MovieMode, 1)
	if res, over := g.Result(); !over || !res.NoVerdict || !g.Contains("NOT TO PLAY.") {
		t.Fatalf("movie mode runs to the end unaided: %+v\n%s", res, g.Transcript())
	}
}

// With pacing on, the show runs self-play, then the montage, then the greeting; every
// self-play game is a draw, and the whole show is bounded.
func TestTheShow(t *testing.T) {
	t.Parallel()
	g := New().(*Game)
	g.Start(proto.Env{Seed: 3, Width: 80, Height: 19})
	var shots []string
	snap := func(label string) {
		c := proto.NewCanvas(80, 19)
		g.View(c)
		shots = append(shots, "==== "+label+" ====\n"+c.String())
	}
	elapsed := time.Duration(0)
	sawMontage, sawBoards := false, false
	for g.phase != greeting {
		g.Handle(proto.TickEvent{Dt: tick})
		elapsed += tick
		for _, b := range g.boards {
			if w := b.Winner(); w != 0 {
				t.Fatalf("self-play must never be won, but %c won:\n%s", w, strings.Join(b.Rows(), "\n"))
			}
		}
		if g.phase == selfPlay && g.round == 3 && marks(g.boards[0]) >= 5 && !sawBoards {
			sawBoards = true
			snap("self-play, round 4")
		}
		if g.phase == montage && g.shown == 25 && !sawMontage {
			sawMontage = true
			snap("montage")
		}
		if elapsed > 2*time.Minute {
			t.Fatalf("the show never ends (phase %d)", g.phase)
		}
	}
	if !sawBoards || !sawMontage || g.shown != len(assets.Scenarios) || g.cracked() != len("CPE1704TKS") {
		t.Errorf("montage %v, shown %d, cracked %d", sawMontage, g.shown, g.cracked())
	}
	t.Logf("the show lasts %v", elapsed)
	golden.AssertString(t, "show", strings.Join(shots, "\n"))
}

func marks(b tictactoe.Board) int {
	n := 0
	for _, m := range b {
		if m != 0 {
			n++
		}
	}
	return n
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
	if Slug != tictactoe.EndingSlug {
		t.Error("tic-tac-toe hands off to this slug")
	}
}
