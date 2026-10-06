package tictactoe

import (
	"strconv"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// The marks at each size, drawn for this project after the big board's tic-tac-toe in the
// film: an X of two crossed strokes and a ring for O, both with doubled strokes.
var (
	artBigX = script.Orig(
		`\\    //`,
		`   ><`,
		`//    \\`,
	)
	artBigO = script.Orig(
		` //""\\`,
		`((    ))`,
		` \\__//`,
	)
	artMidX = script.Orig(
		`\\//`,
		`//\\`,
	)
	artMidO = script.Orig(
		`/""\`,
		`\__/`,
	)
	artSmallX = script.Orig("X")
	artSmallO = script.Orig("O")
)

// The title card, drawn for this project: the name spelled in its own marks, X and O.
var artTitle = script.Orig(
	`XXXXX XXX  XXX       OOOOO  OOO   OOO       XXXXX  XXX  XXXX`,
	`  X    X  X            O   O   O O            X   X   X X`,
	`  X    X  X     ---    O   OOOOO O     ---    X   X   X XXX`,
	`  X    X  X            O   O   O O            X   X   X X`,
	`  X   XXX  XXX         O   O   O  OOO         X    XXX  XXXX`,
)

// Panel labels.
var (
	panelTitle = script.Orig("TIC-TAC-TOE")
	panelYou   = script.Orig("YOU")
	panelWOPR  = script.Orig("WOPR")
)

// BigRows is the height of the largest board: three rows of cells three rows high, and the
// two rules between them.
const BigRows = 11

// size is one way to draw the board: how many rows high the marks are, and how wide a cell.
type size struct {
	rows, cellW int
	x, o        script.Ls
}

var sizes = []size{
	{rows: 3, cellW: 12, x: artBigX, o: artBigO},
	{rows: 2, cellW: 8, x: artMidX, o: artMidO},
	{rows: 1, cellW: 3, x: artSmallX, o: artSmallO},
}

func (s size) height() int { return 3*s.rows + 2 }
func (s size) width() int  { return 3*s.cellW + 2 }

// fit is the largest size that fits h rows.
func fit(h int) size {
	for _, s := range sizes {
		if s.height() <= h {
			return s
		}
	}
	return sizes[len(sizes)-1]
}

// Size is the width and height of the board Draw draws within h rows.
func Size(h int) (w, height int) {
	s := fit(h)
	return s.width(), s.height()
}

// Draw draws b with its top-left corner at x, y, as large as fits in h rows: an open grid,
// as on the big board, X bright and O plain, the square last (-1 for none) in bold. With
// numbers set, each empty square shows its number.
func (b Board) Draw(c *proto.Canvas, x, y, h, last int, numbers bool) {
	s := fit(h)
	w := s.width()
	for r := range 3 {
		top := y + r*(s.rows+1)
		if r > 0 {
			for i := range w {
				ch := '-'
				if i == s.cellW || i == 2*s.cellW+1 {
					ch = '+'
				}
				c.Set(x+i, top-1, proto.Cell{R: ch, S: proto.StyleDim})
			}
		}
		for col := range 3 {
			cx := x + col*(s.cellW+1)
			if col > 0 {
				for i := range s.rows {
					c.Set(cx-1, top+i, proto.Cell{R: '|', S: proto.StyleDim})
				}
			}
			sq := 3*r + col
			var art script.Ls
			style, attr := proto.StyleText, proto.Attr(0)
			switch b[sq] {
			case 'X':
				art, style = s.x, proto.StyleBright
			case 'O':
				art = s.o
			default:
				if numbers {
					c.Put(cx+(s.cellW-1)/2, top+(s.rows-1)/2, strconv.Itoa(sq+1), proto.StyleDim, 0)
				}
				continue
			}
			if sq == last {
				attr = proto.AttrBold
			}
			s.put(c, cx, top, art, style, attr)
		}
	}
}

// put draws a mark centred in the cell at x, y; its spaces leave the cell as it was.
func (s size) put(c *proto.Canvas, x, y int, art script.Ls, style proto.Style, attr proto.Attr) {
	aw := 0
	for _, l := range art {
		aw = max(aw, len(l.Text))
	}
	ax := x + (s.cellW-aw+1)/2
	for i, l := range art {
		for j, ch := range l.Text {
			if ch != ' ' {
				c.Set(ax+j, y+i, proto.Cell{R: ch, S: style, A: attr})
			}
		}
	}
}

// Text renders b as lines of text, as large as fits in h rows, marks only: the final board
// a game hands to the persona.
func (b Board) Text(h int) []string {
	w, bh := Size(h)
	c := proto.NewCanvas(w, bh)
	b.Draw(c, 0, 0, h, -1, false)
	return strings.Split(strings.TrimSuffix(c.String(), "\n"), "\n")
}

// View implements proto.Program. While it asks for the number of players the panel shows
// the title card; in play, the board as large as the panel allows, with the empty squares
// numbered, the last move in bold and, against WOPR, whose mark is whose.
func (g *Game) View(c *proto.Canvas) {
	if g.state == askPlayers {
		w := len(artTitle[0].Text)
		y := max((c.H-len(artTitle))/2, 0)
		for i, l := range artTitle {
			c.Put((c.W-w)/2, y+i, l.Text, proto.StyleBright, 0)
		}
		return
	}
	title := panelTitle[0].Text
	c.Put((c.W-len(title))/2, 0, title, proto.StyleLabel, 0)
	w, h := Size(c.H - 1)
	x, y := (c.W-w)/2, 1+max((c.H-1-h)/2, 0)
	g.b.Draw(c, x, y, c.H-1, g.last, true)
	if g.players != 1 {
		return
	}
	mid := y + h/2 // YOU  X  [board]  O  WOPR
	you := panelYou[0].Text
	c.Put(x-7-len(you), mid, you, proto.StyleLabel, 0)
	c.Put(x-5, mid, artSmallX[0].Text, proto.StyleBright, 0)
	c.Put(x+w+4, mid, artSmallO[0].Text, proto.StyleText, 0)
	c.Put(x+w+7, mid, panelWOPR[0].Text, proto.StyleLabel, 0)
}
