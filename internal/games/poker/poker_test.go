package poker_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/poker"
	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

var info = games.Info{Name: "POKER", Slug: "poker", Layout: proto.LayoutConsole}

// A deterministic transcript: four hands where the player calls or checks, bets once,
// draws the first two cards, and then leaves.
func TestTranscript(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, poker.New(), info, "", 42)
	hands, bet := 1, false
	for range 60 {
		asking, p := g.Asking()
		if !asking {
			break
		}
		switch {
		case strings.HasPrefix(p, "ANOTHER"):
			if hands == 4 {
				g.Type("leave")
				continue
			}
			hands++
			g.Type("y")
		case strings.HasPrefix(p, "DISCARD"):
			g.Type("1 2")
		case strings.Contains(p, "CHECK OR BET") && !bet:
			bet = true
			g.Type("bet")
		default:
			g.Type("c")
		}
	}
	if _, over := g.Result(); !over || !g.Contains("YOU LEAVE WITH") || !g.Contains("WOPR") {
		t.Fatalf("transcript:\n%s", g.Transcript())
	}
	if wide := g.Wide(80); len(wide) > 0 {
		t.Errorf("wider than 80 columns: %q", wide)
	}
	golden.AssertString(t, "transcript", g.Transcript())
}

func TestEscAbandonsTheGame(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, poker.New(), info, "", 1)
	g.Esc().Esc()
	if res, over := g.Result(); !over || res.Outcome != proto.Aborted {
		t.Fatalf("Esc twice aborts: %+v", res)
	}
}

func TestEveryLineHasProvenance(t *testing.T) {
	t.Parallel()
	notice, err := os.ReadFile(filepath.Join("..", "..", "..", "NOTICE.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, err := range script.Validate(poker.Lines, string(notice)) {
		t.Error(err)
	}
	for _, block := range poker.Lines {
		for _, l := range block {
			if len(l.Text) > 80 {
				t.Errorf("%q is wider than 80 columns", l.Text)
			}
		}
	}
}
