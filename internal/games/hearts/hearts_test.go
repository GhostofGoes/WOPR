package hearts_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/hearts"
	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

var info = games.Info{Name: "HEARTS", Slug: "hearts", Layout: proto.LayoutPanel, PanelRows: 10}

func TestView(t *testing.T) {
	t.Parallel()
	g := hearts.New()
	g.Start(proto.Env{Seed: 3, Width: 80, Height: info.PanelRows, Instant: true, Deterministic: true})
	g.Handle(proto.LineEvent{Text: "1 2 3"})
	c := proto.NewCanvas(80, info.PanelRows)
	g.View(c)
	golden.AssertString(t, "view", c.String()+"\n"+c.StyleMap())
}

func TestEscAbandonsTheGame(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, hearts.New(), info, "", 1)
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
	for _, err := range script.Validate(hearts.Lines, string(notice)) {
		t.Error(err)
	}
	for _, block := range hearts.Lines {
		for _, l := range block {
			if len(l.Text) > 80 {
				t.Errorf("%q is wider than 80 columns", l.Text)
			}
		}
	}
}
