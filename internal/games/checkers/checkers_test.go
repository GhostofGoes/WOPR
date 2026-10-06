package checkers_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/checkers"
	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

var info = games.Info{Name: "CHECKERS", Slug: "checkers", Layout: proto.LayoutPanel, PanelRows: 12}

// A deterministic transcript: the player's moves, WOPR's answers, a refusal. The test keeps
// its own position, replaying WOPR's moves from the transcript, and plays the first legal
// move each turn, so it follows whatever WOPR does.
func TestTranscript(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, checkers.New(), info, "", 42)
	g.Type("c3-c4")
	pos := checkers.NewGame()
	for range 8 {
		if asking, _ := g.Asking(); !asking {
			break
		}
		m := pos.Legal()[0]
		g.Type(m.String())
		pos = pos.Play(m)
		lines := strings.Split(strings.TrimSpace(g.Transcript()), "\n")
		last := lines[len(lines)-1]
		reply, ok := strings.CutPrefix(last, "WOPR: ")
		if !ok {
			break // the game ended on the player's move
		}
		wm, ok := pos.ParseMove(reply)
		if !ok {
			t.Fatalf("WOPR's move %q is not legal here:\n%s", reply, g.Transcript())
		}
		pos = pos.Play(wm)
	}
	if !g.Contains("ILLEGAL MOVE.") || strings.Count(g.Transcript(), "WOPR: ") < 4 {
		t.Fatalf("transcript:\n%s", g.Transcript())
	}
	golden.AssertString(t, "transcript", g.Transcript())
}

func TestResign(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, checkers.New(), info, "", 1)
	g.Type("resign")
	if res, over := g.Result(); !over || res.Outcome != proto.Loss {
		t.Fatalf("resigning loses: %+v", res)
	}
}

func TestEveryLineHasProvenance(t *testing.T) {
	t.Parallel()
	notice, err := os.ReadFile(filepath.Join("..", "..", "..", "NOTICE.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, err := range script.Validate(checkers.Lines, string(notice)) {
		t.Error(err)
	}
}

func TestView(t *testing.T) {
	t.Parallel()
	g := checkers.New()
	g.Start(proto.Env{Width: 80, Height: info.PanelRows, Deterministic: true})
	c := proto.NewCanvas(80, info.PanelRows)
	g.View(c)
	golden.AssertString(t, "view", c.String()+"\n"+c.StyleMap())
}
