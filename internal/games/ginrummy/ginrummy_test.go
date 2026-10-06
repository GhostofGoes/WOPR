package ginrummy_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/ginrummy"
	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

var info = games.Info{Name: "GIN RUMMY", Slug: "gin-rummy", Layout: proto.LayoutPanel, PanelRows: 8}

// A deterministic transcript: the player draws from the stock and throws the last card
// the panel shows (its worst deadwood), knocking as soon as the rules allow, for two hands.
func TestTranscript(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, ginrummy.New(), info, "", 42)
	hands := 1
	view := func() string {
		prog, _ := g.Runner().Top()
		c := proto.NewCanvas(80, info.PanelRows)
		prog.View(c)
		return c.String()
	}
	for range 200 {
		asking, p := g.Asking()
		if !asking {
			break
		}
		switch {
		case strings.HasPrefix(p, "NEXT"):
			if hands == 2 {
				g.Type("leave")
				continue
			}
			hands++
			g.Type("")
		case strings.HasPrefix(p, "STOCK"):
			g.Type("s")
		default:
			row := strings.Split(view(), "\n")[6] // the cards' indices, then the deadwood
			at := strings.Index(row, "DEADWOOD: ")
			cardsRow := strings.Fields(strings.ReplaceAll(row[:at], "|", " "))
			last := cardsRow[len(cardsRow)-1]
			deadwood := strings.TrimPrefix(row[at:], "DEADWOOD: ")
			if dw := deadwood; len(cardsRow) == 11 && (dw == "0" || len(dw) == 1 || dw == "10") {
				g.Type("knock " + last)
				if ok, p2 := g.Asking(); ok && strings.HasPrefix(p2, "DISCARD") {
					g.Type(last) // the knock was refused: the discard decides
				}
				continue
			}
			g.Type(last)
		}
	}
	if _, over := g.Result(); !over || !g.Contains("SCORE: YOU") {
		t.Fatalf("transcript:\n%s", g.Transcript())
	}
	if wide := g.Wide(80); len(wide) > 0 {
		t.Errorf("wider than 80 columns: %q", wide)
	}
	golden.AssertString(t, "transcript", g.Transcript())
}

func TestView(t *testing.T) {
	t.Parallel()
	g := ginrummy.New()
	g.Start(proto.Env{Seed: 3, Width: 80, Height: info.PanelRows, Instant: true, Deterministic: true})
	c := proto.NewCanvas(80, info.PanelRows)
	g.View(c)
	golden.AssertString(t, "view", c.String()+"\n"+c.StyleMap())
}

func TestEscAbandonsTheGame(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, ginrummy.New(), info, "", 1)
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
	for _, err := range script.Validate(ginrummy.Lines, string(notice)) {
		t.Error(err)
	}
	for _, block := range ginrummy.Lines {
		for _, l := range block {
			if len(l.Text) > 80 {
				t.Errorf("%q is wider than 80 columns", l.Text)
			}
		}
	}
}
