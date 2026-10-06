// Package board draws 8x8 boards the way WOPR's screen shows them (docs/PLAN.md Appendix C):
// no grid, rank numbers on the left, file letters below, and a dot on each empty square
// that pieces can use. Checkers and chess share it, and parse squares with it.
package board

import (
	"strings"

	"github.com/GhostofGoes/WOPR/internal/proto"
)

// Glyph is what one square shows.
type Glyph struct {
	R rune
	S proto.Style
	A proto.Attr
}

// Rows is the height Draw uses: a title, a blank row, eight ranks and the file letters,
// with a blank row to spare in a 12-row panel.
const Rows = 11

// width is the drawn board: a rank label, three spaces, then eight squares three apart.
const width = 4 + 8*3 - 2

// Draw draws title and the board into c, centred. at returns the glyph for a square, with
// file and rank counted from 0 (a1 is 0, 0). Rank 8 is at the top.
func Draw(c *proto.Canvas, title string, at func(file, rank int) Glyph) {
	c.Put((c.W-len(title))/2, 0, title, proto.StyleBright, 0)
	x0 := max((c.W-width)/2, 0)
	for row := range 8 {
		rank := 7 - row
		y := 2 + row
		c.Put(x0, y, string(rune('1'+rank)), proto.StyleDim, 0)
		for file := range 8 {
			g := at(file, rank)
			if g.R == 0 {
				g.R = ' '
			}
			c.Set(x0+4+3*file, y, proto.Cell{R: g.R, S: g.S, A: g.A})
		}
	}
	var files strings.Builder
	for file := range 8 {
		if file > 0 {
			files.WriteString("  ")
		}
		files.WriteByte(byte('a' + file))
	}
	c.Put(x0+4, 10, files.String(), proto.StyleDim, 0)
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
