package hearts

import (
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games/cards"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// checkArt reports art that is not printable ASCII capitals, is taller than 12 rows or
// wider than 80 columns, or is missing from lines (where the provenance test checks it).
func checkArt(t *testing.T, art script.Ls, lines []script.Ls) {
	t.Helper()
	if len(art) == 0 || len(art) > 12 {
		t.Errorf("the art is %d rows; a block on the console is at most 12", len(art))
	}
	listed := false
	for _, block := range lines {
		listed = listed || &block[0] == &art[0]
	}
	if !listed {
		t.Error("the art is not in Lines")
	}
	for _, l := range art {
		for _, r := range l.Text {
			if r < 0x20 || r > 0x7e {
				t.Errorf("%q is not printable ASCII", l.Text)
				break
			}
		}
		if len(l.Text) > 80 || l.Text != strings.ToUpper(l.Text) || l.Text != strings.TrimRight(l.Text, " ") {
			t.Errorf("%q: wider than 80 columns, lower case, or trailing spaces", l.Text)
		}
		if l.Prov != script.Original {
			t.Errorf("%q is tagged %q, not original", l.Text, l.Prov)
		}
	}
}

func TestTitleArt(t *testing.T) {
	t.Parallel()
	checkArt(t, artTitle, Lines)
	for _, bad := range cards.CornerProblems(artTitle.Texts()) { // every whole card shows its index twice
		t.Error(bad)
	}
	for _, block := range Lines {
		for _, l := range block {
			if strings.ContainsFunc(l.Text, func(r rune) bool { return r < 0x20 || r > 0x7e }) {
				t.Errorf("%q is not printable ASCII", l.Text)
			}
		}
	}
}

// The panel stays inside its rows through whole games.
func TestViewFitsThePanel(t *testing.T) {
	t.Parallel()
	for seed := range uint64(5) {
		g := New().(*Game)
		outs := g.Start(proto.Env{Seed: seed, Instant: true, Deterministic: true})
		r := proto.NewRand(seed, 9)
		for step := 0; step < 2000 && !done(outs); step++ {
			c := proto.NewCanvas(80, info.PanelRows+2)
			g.View(c)
			if extra := strings.Split(c.String(), "\n")[info.PanelRows:]; strings.TrimSpace(strings.Join(extra, "")) != "" {
				t.Fatalf("seed %d: the view draws below its %d rows:\n%s", seed, info.PanelRows, c.String())
			}
			input := ""
			switch g.phase {
			case passing:
				input = "1 2 3"
			case playing:
				legal := Legal(g.hands[you], g.trick, g.tricks == 0, g.broken)
				input = legal[r.IntN(len(legal))].String()
			case between:
			}
			outs = g.Handle(proto.LineEvent{Text: input})
		}
	}
}
