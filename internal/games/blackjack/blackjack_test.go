package blackjack_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/blackjack"
	"github.com/GhostofGoes/WOPR/internal/games/cards"
	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

var info = games.Info{Name: "BLACK JACK", Slug: "black-jack", Layout: proto.LayoutConsole}

// A deterministic transcript: five rounds of $10, hitting once on each hand and then
// standing, a bad bet, and leaving the table.
func TestTranscript(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, blackjack.New(), info, "", 42)
	g.Type("a lot")
	rounds, hit := 0, false
	for range 40 {
		asking, p := g.Asking()
		if !asking {
			break
		}
		switch {
		case strings.HasPrefix(p, "YOUR BET"):
			if rounds == 5 {
				g.Type("leave")
				continue
			}
			rounds++
			hit = false
			g.Type("10")
		case !hit:
			hit = true
			g.Type("h")
		default:
			g.Type("s")
		}
	}
	res, over := g.Result()
	if !over || !g.Contains("YOU LEAVE THE TABLE WITH") || !g.Contains("A BET IS A WHOLE NUMBER") {
		t.Fatalf("transcript:\n%s", g.Transcript())
	}
	if res.Outcome == proto.Aborted {
		t.Fatal("leaving is not an abort")
	}
	if wide := g.Wide(80); len(wide) > 0 {
		t.Errorf("wider than 80 columns: %q", wide)
	}
	for _, bad := range cards.CornerProblems(strings.Split(g.Transcript(), "\n")) {
		t.Error(bad) // every card dealt face up shows its index in both corners
	}
	golden.AssertString(t, "transcript", g.Transcript())
}

func TestEscAbandonsTheTable(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, blackjack.New(), info, "", 1)
	g.Type("10").Esc().Esc()
	if res, over := g.Result(); !over || res.Outcome != proto.Aborted {
		t.Fatalf("Esc twice aborts: %+v", res)
	}
}

func TestLeavingEvenIsADraw(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, blackjack.New(), info, "", 1)
	g.Type("leave")
	if res, over := g.Result(); !over || res.Outcome != proto.Draw {
		t.Fatalf("leaving with the stake: %+v %v\n%s", res, over, g.Transcript())
	}
}

func TestEveryLineHasProvenance(t *testing.T) {
	t.Parallel()
	notice, err := os.ReadFile(filepath.Join("..", "..", "..", "NOTICE.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, err := range script.Validate(blackjack.Lines, string(notice)) {
		t.Error(err)
	}
}

// Every line fits the 80-column console.
func TestLinesFit(t *testing.T) {
	t.Parallel()
	for _, block := range blackjack.Lines {
		for _, l := range block {
			if len(l.Text) > 80 {
				t.Errorf("%q is wider than 80 columns", l.Text)
			}
		}
	}
}
