package tictactoe

import (
	"context"
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

const panelRows = 12 // the catalog's PanelRows for tic-tac-toe

// Every line of art and panel text is printable ASCII in capitals, at most 80 columns,
// tagged original and listed in Lines (where the provenance test checks it); the title card
// fits the panel.
func TestArt(t *testing.T) {
	t.Parallel()
	if len(artTitle) > panelRows {
		t.Errorf("the title card is %d rows; the panel is %d", len(artTitle), panelRows)
	}
	blocks := []script.Ls{artTitle, artBigX, artBigO, artMidX, artMidO, artSmallX, artSmallO, panelTitle, panelYou, panelWOPR}
	for _, block := range blocks {
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

// Each size's marks are as tall as its cells and leave a column clear on either side; the
// boards fit 80 columns, and every height from the smallest board up gets one that fits.
func TestSizes(t *testing.T) {
	t.Parallel()
	for _, s := range sizes {
		for _, art := range []script.Ls{s.x, s.o} {
			if len(art) != s.rows {
				t.Errorf("%q is %d rows, its cells %d", art[0].Text, len(art), s.rows)
			}
			for _, l := range art {
				if len(l.Text) > s.cellW-2 {
					t.Errorf("%q is too wide for a %d-column cell", l.Text, s.cellW)
				}
			}
		}
		if s.width() > 80 {
			t.Errorf("a %d-row board is %d columns", s.rows, s.width())
		}
	}
	for h := sizes[len(sizes)-1].height(); h <= 24; h++ {
		if _, bh := Size(h); bh > h {
			t.Errorf("Size(%d) is %d rows", h, bh)
		}
	}
	if _, bh := Size(BigRows); bh != BigRows {
		t.Errorf("the big board is %d rows, not BigRows", bh)
	}
	b := Board{'X', 'O', 0, 0, 'X', 0, 'O', 0, 'X'}
	if got := strings.Join(b.Text(5), "\n"); got != " X | O |\n---+---+---\n   | X |\n---+---+---\n O |   | X" {
		t.Errorf("the small board:\n%s", got)
	}
}

// The panel as the player sees it: the title card while it asks for players, then the board
// in today's 9 rows and in the 12 a taller panel would give it.
func TestViews(t *testing.T) {
	t.Parallel()
	var shots []string
	snap := func(label string, g *Game, h int) {
		c := proto.NewCanvas(80, h)
		g.View(c)
		shots = append(shots, "==== "+label+" ====\n"+c.String()+c.StyleMap())
	}
	for _, mode := range []string{"", Climax} {
		g := New().(*Game)
		g.Start(proto.Env{Width: 80, Height: panelRows, Mode: mode})
		c := proto.NewCanvas(80, panelRows)
		g.View(c)
		if !strings.Contains(c.String(), artTitle[2].Text) {
			t.Errorf("mode %q: the title card shows while it asks for players:\n%s", mode, c.String())
		}
	}
	g := New().(*Game)
	g.Start(proto.Env{Width: 80, Height: panelRows, Seed: 3})
	snap("players, 9 rows", g, panelRows)
	g.Handle(proto.LineEvent{Text: "1"})
	for _, sq := range []string{"5", "1"} {
		for _, o := range g.Handle(proto.LineEvent{Text: sq}) {
			if th, ok := o.(proto.Think); ok {
				v, _ := th.Fn(context.Background())
				g.Handle(proto.ThinkDone{Value: v})
			}
		}
	}
	snap("one player, 9 rows", g, panelRows)
	snap("one player, 12 rows", g, 12)
	golden.AssertString(t, "views", strings.Join(shots, "\n"))
}

// The last move is told by more than colour: at every size, with an X or an O last, every
// cell of that mark is underlined and no other cell is. X is bright, which is bold in every
// theme, so bold could not mark it.
func TestLastMoveIsUnderlined(t *testing.T) {
	t.Parallel()
	b := Board{'X', 'O', 0, 0, 'X', 'O', 0, 0, 0}
	for _, s := range sizes {
		for _, last := range []int{4, 5} { // an X, then an O
			c := proto.NewCanvas(s.width(), s.height())
			b.Draw(c, 0, 0, s.height(), last, true)
			marked := 0
			for y := range c.H {
				for x := range c.W {
					cell := c.At(x, y)
					sq := 3*(y/(s.rows+1)) + x/(s.cellW+1)
					inLast := sq == last && x%(s.cellW+1) != s.cellW && y%(s.rows+1) != s.rows
					underlined := cell.A&proto.AttrUnderline != 0
					switch {
					case inLast && cell.R != ' ' && !underlined:
						t.Errorf("%d-row board, square %d last: %q at %d,%d is not underlined:\n%s", s.rows, last+1, cell.R, x, y, c.String())
					case !inLast && underlined:
						t.Errorf("%d-row board, square %d last: %q at %d,%d is underlined:\n%s", s.rows, last+1, cell.R, x, y, c.String())
					case underlined:
						marked++
					}
				}
			}
			if marked == 0 {
				t.Errorf("%d-row board, square %d last: nothing is underlined", s.rows, last+1)
			}
		}
	}
	c := proto.NewCanvas(sizes[0].width(), sizes[0].height())
	b.Draw(c, 0, 0, c.H, -1, true)
	for y := range c.H {
		for x := range c.W {
			if c.At(x, y).A&proto.AttrUnderline != 0 {
				t.Fatalf("no last move, yet %d,%d is underlined:\n%s", x, y, c.String())
			}
		}
	}
}
