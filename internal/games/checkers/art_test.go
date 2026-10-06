package checkers

import (
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

const panelRows = 12 // the catalog's PanelRows for checkers

// Every line of art and panel text is printable ASCII in capitals, at most 80 columns,
// tagged original and listed in Lines (where the provenance test checks it); the title card
// is at most 12 rows, the panel's height.
func TestArt(t *testing.T) {
	t.Parallel()
	if len(artTitle) > panelRows {
		t.Errorf("the title card is %d rows; the panel is %d", len(artTitle), panelRows)
	}
	for _, block := range []script.Ls{artTitle, panelWOPR, panelYou, panelSides, panelMen, panelKings, panelLast, panelMove, panelMore} {
		listed := false
		for _, l := range Lines {
			listed = listed || &l[0] == &block[0]
		}
		if !listed {
			t.Errorf("%q is not in Lines", block[0].Text)
		}
		for _, l := range block {
			if strings.ContainsFunc(l.Text, func(r rune) bool { return r < 0x20 || r > 0x7e }) {
				t.Errorf("%q is not printable ASCII", l.Text)
			}
			if len(l.Text) > 80 || l.Text != strings.ToUpper(l.Text) || l.Text != strings.TrimRight(l.Text, " ") {
				t.Errorf("%q: wider than 80 columns, lower case, or trailing spaces", l.Text)
			}
			if l.Prov != script.Original {
				t.Errorf("%q is tagged %q, not original", l.Text, l.Prov)
			}
		}
	}
}

// The last move's squares are bracketed, and the men a jump took are crossed out where they
// stood: here White's E5XC3XA1 took d4 and b2.
func TestJumpMarks(t *testing.T) {
	t.Parallel()
	g := New().(*Game)
	for _, s := range []string{"a3-b4", "b6-c5", "b2-a3", "f6-e5", "a1-b2", "e7-f6", "c3-d4", "e5xc3xa1"} {
		m, ok := g.pos.ParseMove(s)
		if !ok {
			t.Fatalf("%s is not legal here", s)
		}
		g.pos, g.last = g.pos.Play(m), m
	}
	lines := g.result(proto.Win).Lines
	for row, want := range map[int]string{
		5: "5 |:::   (w)   [ ]   :::   | 5", // e5, where the jump began
		6: "4 |   (b)   :x:   :::   :::| 4", // d4, taken
		8: "2 |   :x:   (b)   (b)   (b)| 2", // b2, taken
		9: "1 |[W]   (b)   (b)   (b)   | 1", // a1, where it ended, crowned
	} {
		if !strings.HasPrefix(lines[row], want) {
			t.Errorf("row %d:\n got %q\nwant %q...", row, lines[row], want)
		}
	}
}

// The panel stays inside its 80 columns and 12 rows, centred on a wider canvas, with
// everything showing: kings on both sides and the longest jump the board allows.
func TestPanelFits(t *testing.T) {
	t.Parallel()
	g := New().(*Game)
	g.pos = Position{toMove: White}
	for _, sq := range []string{"a1", "c1", "e1", "g1"} {
		f, r := int(sq[0]-'a'), int(sq[1]-'1')
		g.pos.sq[at(f, r)] = blackKing
	}
	for _, sq := range []string{"b8", "d8", "f8", "h8", "b2", "d2", "f2", "b4", "d4", "f4", "b6", "d6", "f6"} {
		f, r := int(sq[0]-'a'), int(sq[1]-'1')
		g.pos.sq[at(f, r)] = whiteKing
		if r < 7 {
			g.pos.sq[at(f, r)] = whiteMan
		}
	}
	for _, s := range []string{"a1", "c3", "e5", "g7", "e5", "c7", "a5", "c3", "e1"} { // a tour of jumps
		f, r := int(s[0]-'a'), int(s[1]-'1')
		g.last = g.last.add(at(f, r))
	}
	g.moves = 99
	const w, h = 100, 20
	c := proto.NewCanvas(w, h)
	g.View(c)
	for y := range h {
		for x := range w {
			if c.At(x, y).R != ' ' && (y >= panelRows || x < (w-80)/2 || x >= (w+80)/2) {
				t.Fatalf("the panel draws at %d,%d, outside 80x%d:\n%s", x, y, panelRows, c.String())
			}
		}
	}
	text := c.String()
	if strings.Count(text, "KINGS 4") != 2 {
		t.Errorf("both sides have four kings:\n%s", text)
	}
	for _, want := range []string{"A1X..XE1", "MOVE   100", "[B]"} {
		if !strings.Contains(text, want) {
			t.Errorf("the panel should show %q:\n%s", want, text)
		}
	}
	if lines := g.result(proto.Draw).Lines; len(lines) > 12 {
		t.Errorf("the final position is %d lines", len(lines))
	}
}
