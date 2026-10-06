// Package board draws 8x8 boards for checkers and chess, and parses their squares.
//
// The board is framed, with file letters above and below and rank numbers on both sides,
// so a square can be read off from any edge. Dark squares (a1's colour) are shaded with a
// dim stipple, so the board reads as a board at a glance even with pieces on it; light
// squares are blank. Docs/PLAN.md Appendix C drew the board gridless and unshaded; the
// frame and shading were added for the owner's request for more visual flair, and stay
// spare: one thin outline, the shading in the dim style, the pieces bright.
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
	Disc bool        // draw the piece as a disc, (b): a checker
	Mark bool        // bracket the square, [n] or [ ]: a square the last move touched
}

// Geometry. Each square is three columns: the piece in the middle, shading or brackets on
// either side.
const (
	Rows  = 12 // file letters, the frame's top, eight ranks, its bottom, file letters
	Width = 30 // a rank number, a space, the frame, 8 squares of 3, the frame, a space, a rank number

	squareW = 3
	inner   = 8 * squareW
	shade   = ':'
)

// Dark reports whether a square is dark; a1 is.
func Dark(file, rank int) bool { return (file+rank)%2 == 0 }

// Files is the file letters, one over the middle of each square.
const Files = "a  b  c  d  e  f  g  h"

// Draw draws the board with its top-left corner at x, y; it takes Width columns and Rows
// rows. at returns the glyph for a square, with file and rank counted from 0 (a1 is 0, 0).
// Rank 8 is at the top.
func Draw(c *proto.Canvas, x, y int, at func(file, rank int) Glyph) {
	frame := x + 2
	c.Put(frame+2, y, Files, proto.StyleDim, 0)
	c.Put(frame, y+1, "."+strings.Repeat("-", inner)+".", proto.StyleDim, 0)
	for row := range 8 {
		rank := 7 - row
		ry := y + 2 + row
		label := string(rune('1' + rank))
		c.Put(x, ry, label, proto.StyleDim, 0)
		c.Put(frame, ry, "|", proto.StyleDim, 0)
		for file := range 8 {
			drawSquare(c, frame+1+squareW*file, ry, Dark(file, rank), at(file, rank))
		}
		c.Put(frame+1+inner, ry, "|", proto.StyleDim, 0)
		c.Put(frame+inner+3, ry, label, proto.StyleDim, 0)
	}
	c.Put(frame, y+10, "'"+strings.Repeat("-", inner)+"'", proto.StyleDim, 0)
	c.Put(frame+2, y+11, Files, proto.StyleDim, 0)
}

// drawSquare draws one square, three columns from x.
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
