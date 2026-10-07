package proto

import "strings"

// Style is the meaning of a cell, not its colour. The ui maps each Style to the active
// theme, with explicit 16-colour and no-colour fallbacks.
type Style uint8

// Styles.
const (
	StyleText Style = iota
	StyleBright
	StyleDim
	StyleAccent
	StyleLand
	StyleLabel
	StyleIncoming
	StyleOutgoing
	StyleTarget
	StyleDefcon1
	StyleDefcon2
	StyleDefcon3
	StyleDefcon4
	StyleDefcon5
	StyleAlert
	StyleSuitRed
	StyleSuitBlack
	StyleSelected
	numStyles
)

// styleLetters render a StyleMap: one letter per style, in Style order.
const styleLetters = "tbdalLiox12345Arks"

// Styles lists every Style in order. Tests that must hold for each style (every theme
// defines it, its contrast) range over it, so a new Style is covered once it is declared.
func Styles() []Style {
	out := make([]Style, numStyles)
	for i := range out {
		out[i] = Style(i)
	}
	return out
}

// Letter returns the one-character code StyleMap uses for s.
func (s Style) Letter() byte {
	if int(s) < len(styleLetters) {
		return styleLetters[s]
	}
	return '?'
}

// Attr is a set of text attributes.
type Attr uint8

// Attributes. Blink is rendered by the ui's clock (at most 3 Hz), not by SGR 5.
const (
	AttrBlink Attr = 1 << iota
	AttrReverse
	AttrUnderline
	AttrBold
)

// Cell is one character position.
type Cell struct {
	R rune
	S Style
	A Attr
}

// Canvas is a fixed-size grid of cells that programs draw into. Every rune occupies one
// cell; art is ASCII by design.
type Canvas struct {
	W, H  int
	cells []Cell
}

// NewCanvas returns a blank w×h canvas.
func NewCanvas(w, h int) *Canvas {
	c := &Canvas{}
	c.Reset(w, h)
	return c
}

// Reset resizes the canvas and blanks it.
func (c *Canvas) Reset(w, h int) {
	w, h = max(w, 0), max(h, 0)
	c.W, c.H = w, h
	if cap(c.cells) >= w*h {
		c.cells = c.cells[:w*h]
	} else {
		c.cells = make([]Cell, w*h)
	}
	for i := range c.cells {
		c.cells[i] = Cell{R: ' '}
	}
}

// Set writes one cell; positions outside the canvas are ignored.
func (c *Canvas) Set(x, y int, cell Cell) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	c.cells[y*c.W+x] = cell
}

// At returns the cell at x, y, or a blank cell outside the canvas.
func (c *Canvas) At(x, y int) Cell {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return Cell{R: ' '}
	}
	return c.cells[y*c.W+x]
}

// Put writes s starting at x, y in one style, clipping at the edges. It returns the x
// after the last rune written.
func (c *Canvas) Put(x, y int, s string, st Style, a Attr) int {
	for _, r := range s {
		c.Set(x, y, Cell{R: r, S: st, A: a})
		x++
	}
	return x
}

// String renders the text of the canvas, one line per row, with trailing spaces trimmed.
// Golden tests use it so they stay independent of the theme.
func (c *Canvas) String() string {
	var b strings.Builder
	for y := range c.H {
		row := make([]rune, c.W)
		for x := range c.W {
			row[x] = c.cells[y*c.W+x].R
		}
		b.WriteString(strings.TrimRight(string(row), " "))
		b.WriteByte('\n')
	}
	return b.String()
}

// StyleMap renders one style letter per non-blank cell ('.' for blanks), so goldens can
// assert colour meaning without escape codes.
func (c *Canvas) StyleMap() string {
	var b strings.Builder
	for y := range c.H {
		row := make([]byte, c.W)
		for x := range c.W {
			cell := c.cells[y*c.W+x]
			if cell.R == ' ' && cell.A&AttrReverse == 0 {
				row[x] = '.'
				continue
			}
			row[x] = cell.S.Letter()
		}
		b.WriteString(strings.TrimRight(string(row), "."))
		b.WriteByte('\n')
	}
	return b.String()
}
