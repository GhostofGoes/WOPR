package falkensmaze_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/falkensmaze"
	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

var info = games.Info{Name: "FALKEN'S MAZE", Slug: "falkens-maze", Layout: proto.LayoutPanel, PanelRows: 12}

// The opening view, and the view after a few moves: fog, the player, the exit.
func TestView(t *testing.T) {
	t.Parallel()
	g := falkensmaze.New()
	g.Start(proto.Env{Seed: 7, Width: 80, Height: info.PanelRows, Instant: true})
	var shots string
	snap := func(label string) {
		c := proto.NewCanvas(80, info.PanelRows)
		g.View(c)
		shots += "==== " + label + " ====\n" + c.String() + "\n" + c.StyleMap() + "\n"
	}
	snap("start")
	for _, k := range []proto.Key{proto.KeyRight, proto.KeyRight, proto.KeyDown, proto.KeyDown, proto.KeyRight, proto.KeyUp, proto.KeyLeft} {
		g.Handle(proto.KeyEvent{Key: k})
	}
	snap("a few keys later")
	golden.AssertString(t, "view", shots)
}

// Under the host, in key mode: Q gives up and is a loss; Esc twice abandons the game.
func TestUnderTheHost(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, falkensmaze.New(), info, "", 1)
	g.Key(proto.KeyRight, 0).Key(proto.KeyRune, 'q')
	if res, over := g.Result(); !over || res.Outcome != proto.Loss || !g.Contains("RETREAT ACCEPTED.") {
		t.Fatalf("give up: %+v\n%s", res, g.Transcript())
	}
	e := testkit.Game(t, falkensmaze.New(), info, "", 1)
	e.Esc().Esc()
	if res, over := e.Result(); !over || res.Outcome != proto.Aborted {
		t.Fatalf("Esc twice aborts: %+v", res)
	}
}

func TestEveryLineHasProvenance(t *testing.T) {
	t.Parallel()
	notice, err := os.ReadFile(filepath.Join("..", "..", "..", "NOTICE.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, err := range script.Validate(falkensmaze.Lines, string(notice)) {
		t.Error(err)
	}
}
