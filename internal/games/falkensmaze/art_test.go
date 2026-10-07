package falkensmaze

import (
	"slices"
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// Every line is printable ASCII in capitals and at most 80 columns; the win art is at most
// 12 rows, tagged original and listed in Lines, where the provenance test checks it.
func TestArtLines(t *testing.T) {
	t.Parallel()
	if n := len(artEscaped); n == 0 || n > 12 {
		t.Errorf("the win art is %d rows; a console block is at most 12", n)
	}
	if !slices.ContainsFunc(Lines, func(b script.Ls) bool { return &b[0] == &artEscaped[0] }) {
		t.Error("the win art is not in Lines")
	}
	for _, block := range Lines {
		for _, l := range block {
			if strings.ContainsFunc(l.Text, func(r rune) bool { return r < 0x20 || r > 0x7e }) {
				t.Errorf("%q is not printable ASCII", l.Text)
			}
			if len(l.Text) > 80 || l.Text != strings.ToUpper(l.Text) || l.Text != strings.TrimRight(l.Text, " ") {
				t.Errorf("%q: wider than 80 columns, lower case, or trailing spaces", l.Text)
			}
		}
	}
	for _, l := range append(append(script.Ls{}, artEscaped...), frameMarks...) {
		if l.Prov != script.Original {
			t.Errorf("%q is tagged %q, not original", l.Text, l.Prov)
		}
	}
}

// The way out: a player who walks the shortest way reaches the exit; the win prints the
// ESCAPED art before WOPR's line, and the panel lifts the fog, opens the east wall and
// dots the trail.
func TestWinView(t *testing.T) {
	t.Parallel()
	g := New().(*Game)
	g.Start(proto.Env{Seed: 7, Instant: true})
	var last []proto.Output
	for step := 0; step < 2000 && !g.won; step++ {
		path := g.m.path(g.player, g.exit)
		d := g.m.dirTo(path[0], path[1])
		last = g.Handle(proto.KeyEvent{Key: []proto.Key{proto.KeyUp, proto.KeyRight, proto.KeyDown, proto.KeyLeft}[d]})
	}
	if !g.won || len(last) != 4 {
		t.Fatalf("won %v, last outputs %v", g.won, last)
	}
	if s, ok := last[1].(proto.Say); !ok || s.Pace != proto.PaceTable || !slices.Equal(s.Lines, artEscaped.Texts()) {
		t.Errorf("the win art is not printed as a table before WOPR's line: %+v", last[1])
	}
	c := proto.NewCanvas(80, PanelRows)
	g.View(c)
	golden.AssertString(t, "win_view", c.String()+"\n"+c.StyleMap())
}
