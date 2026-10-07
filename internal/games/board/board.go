// Package board draws 8x8 boards for checkers and chess, and parses their squares.
//
// The board is framed, with file letters above and below and rank numbers on both sides,
// so a square can be read off from any edge. Dark squares (a1's colour) are shaded with a
// dim stipple, so the board reads as a board at a glance even with pieces on it; light
// squares are blank. Docs/PLAN.md Appendix C drew the board gridless and unshaded; the
// frame and shading were added for the owner's request for more visual flair, and stay
// spare: one thin outline, the shading in the dim style, the pieces bright.
//
// Draw's squares are three columns wide and one row tall (checkers). DrawCompact's are two
// by one, which a terminal, its cells about twice as tall as wide, shows square (chess, at
// the owner's request: three columns made the board half as wide again as it is tall).
package board

import (
	"strings"

	"github.com/GhostofGoes/WOPR/internal/proto"
)

// Glyph is what one square shows.
type Glyph struct {
	R    rune        // the piece, or 0 for an empty square
	S    proto.Style // the piece's style
	A    proto.Attr  // the piece's attributes
	Disc bool        // draw the piece as a disc, (b): a checker; Draw only
	Mark bool        // mark a square the last move touched: [n] or [ ] in Draw, n< or [] in DrawCompact
}

// Geometry. Draw's squares are three columns: the piece in the middle, shading or brackets
// on either side. DrawCompact's are two: the piece or shading, then shading or a mark.
const (
	Rows         = 12 // file letters, the frame's top, eight ranks, its bottom, file letters
	Width        = 30 // a rank number, a space, the frame, 8 squares of 3, the frame, a space, a rank number
	CompactWidth = 22 // the same with 8 squares of 2

	shade = ':'
)

// Dark reports whether a square is dark; a1 is.
func Dark(file, rank int) bool { return (file+rank)%2 == 0 }

// The file letters, one over each square's piece: Files for Draw, over the middle of each
// square, and CompactFiles for DrawCompact, over the first column of each.
const (
	Files        = "a  b  c  d  e  f  g  h"
	CompactFiles = "a b c d e f g h"
)

// geometry is one way of drawing the squares.
type geometry struct {
	w      int    // columns per square
	piece  int    // the piece's column within its square
	files  string // the file letters, one over each square's piece
	square func(c *proto.Canvas, x, y int, dark bool, g Glyph)
}

var (
	wide    = geometry{w: 3, piece: 1, files: Files, square: drawSquare}
	compact = geometry{w: 2, piece: 0, files: CompactFiles, square: drawCompactSquare}
)

// Draw draws the board with its top-left corner at x, y; it takes Width columns and Rows
// rows. at returns the glyph for a square, with file and rank counted from 0 (a1 is 0, 0).
// Rank 8 is at the top.
func Draw(c *proto.Canvas, x, y int, at func(file, rank int) Glyph) { wide.draw(c, x, y, at) }

// DrawCompact draws the board as Draw does, with squares two columns wide so that it looks
// square: it takes CompactWidth columns and Rows rows. It draws no discs.
func DrawCompact(c *proto.Canvas, x, y int, at func(file, rank int) Glyph) {
	compact.draw(c, x, y, at)
}

func (gm geometry) draw(c *proto.Canvas, x, y int, at func(file, rank int) Glyph) {
	inner := 8 * gm.w
	frame := x + 2
	c.Put(frame+1+gm.piece, y, gm.files, proto.StyleDim, 0)
	c.Put(frame, y+1, "."+strings.Repeat("-", inner)+".", proto.StyleDim, 0)
	for row := range 8 {
		rank := 7 - row
		ry := y + 2 + row
		label := string(rune('1' + rank))
		c.Put(x, ry, label, proto.StyleDim, 0)
		c.Put(frame, ry, "|", proto.StyleDim, 0)
		for file := range 8 {
			gm.square(c, frame+1+gm.w*file, ry, Dark(file, rank), at(file, rank))
		}
		c.Put(frame+1+inner, ry, "|", proto.StyleDim, 0)
		c.Put(frame+inner+3, ry, label, proto.StyleDim, 0)
	}
	c.Put(frame, y+10, "'"+strings.Repeat("-", inner)+"'", proto.StyleDim, 0)
	c.Put(frame+1+gm.piece, y+11, gm.files, proto.StyleDim, 0)
}

// drawSquare draws one of Draw's squares, three columns from x.
func drawSquare(c *proto.Canvas, x, y int, dark bool, g Glyph) {
	side := proto.Cell{R: ' ', S: proto.StyleDim}
	if dark {
		side.R = shade
	}
	mid := side
	if g.R != 0 {
		mid = proto.Cell{R: g.R, S: g.S, A: g.A}
	}
	left, right := side, side
	switch {
	case g.Mark:
		left, right = proto.Cell{R: '[', S: proto.StyleAccent}, proto.Cell{R: ']', S: proto.StyleAccent}
		if g.R == 0 {
			mid = proto.Cell{R: ' ', S: proto.StyleDim}
		}
	case g.Disc && g.R != 0:
		left, right = proto.Cell{R: '(', S: g.S}, proto.Cell{R: ')', S: g.S}
	}
	c.Set(x, y, left)
	c.Set(x+1, y, mid)
	c.Set(x+2, y, right)
}

// drawCompactSquare draws one of DrawCompact's squares, two columns from x: the piece (or
// shading), then shading, so the second column shows each square's colour in its glyph,
// piece or not. A marked square is [] when empty; otherwise a pointer follows its piece, n<.
func drawCompactSquare(c *proto.Canvas, x, y int, dark bool, g Glyph) {
	fill := proto.Cell{R: ' ', S: proto.StyleDim}
	if dark {
		fill.R = shade
	}
	piece := fill
	if g.R != 0 {
		piece = proto.Cell{R: g.R, S: g.S, A: g.A}
	}
	if g.Mark {
		fill = proto.Cell{R: '<', S: proto.StyleAccent}
		if g.R == 0 {
			piece, fill = proto.Cell{R: '[', S: proto.StyleAccent}, proto.Cell{R: ']', S: proto.StyleAccent}
		}
	}
	c.Set(x, y, piece)
	c.Set(x+1, y, fill)
}

// ParseSquare reads a square such as "e4" (any case) as file and rank from 0.
func ParseSquare(s string) (file, rank int, ok bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if len(s) != 2 || s[0] < 'a' || s[0] > 'h' || s[1] < '1' || s[1] > '8' {
		return 0, 0, false
	}
	return int(s[0] - 'a'), int(s[1] - '1'), true
}

// SquareName is the inverse of ParseSquare.
func SquareName(file, rank int) string {
	return string([]byte{byte('a' + file), byte('1' + rank)})
}

// Text renders a View as plain lines for the console: the final position a game hands to
// the persona in Result.Lines, since the panel goes when the game does. Common leading
// spaces and trailing blank lines are dropped.
func Text(view func(*proto.Canvas), w, h int) []string {
	c := proto.NewCanvas(w, h)
	view(c)
	lines := strings.Split(c.String(), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " ")
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	indent := w
	for _, l := range lines {
		if l != "" {
			indent = min(indent, len(l)-len(strings.TrimLeft(l, " ")))
		}
	}
	for i, l := range lines {
		if len(l) >= indent {
			lines[i] = l[indent:]
		}
	}
	return lines
}
