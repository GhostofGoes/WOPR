package proto

import "testing"

func TestKeyString(t *testing.T) {
	t.Parallel()
	want := map[Key]string{
		KeyRune: "rune", KeyUp: "up", KeyDown: "down", KeyLeft: "left", KeyRight: "right",
		KeyEnter: "enter", KeyBackspace: "backspace", KeyEsc: "esc", KeyEsc + 1: "unknown",
	}
	for k, s := range want {
		if got := k.String(); got != s {
			t.Errorf("Key(%d).String() = %q, want %q", k, got, s)
		}
	}
}

func TestStyleLetterOutOfRange(t *testing.T) {
	t.Parallel()
	if got := Style(len(styleLetters)).Letter(); got != '?' {
		t.Errorf("a style past the last has letter %q, want '?'", got)
	}
}

func TestPaceCharsPerSecond(t *testing.T) {
	t.Parallel()
	for p, want := range map[Pace]float64{PaceSpeech: 30, PaceTable: 240, PaceTyping: 8, PaceInstant: 0} {
		if got := p.CharsPerSecond(); got != want {
			t.Errorf("Pace(%d): %v characters a second, want %v", p, got, want)
		}
	}
}

// Writes outside the canvas are dropped, reads outside it are blank, and a smaller Reset
// reuses the cells, blanked.
func TestCanvasEdges(t *testing.T) {
	t.Parallel()
	c := NewCanvas(3, 2)
	x := Cell{R: 'X', S: StyleBright}
	for _, at := range [][2]int{{-1, 0}, {0, -1}, {3, 0}, {0, 2}} {
		c.Set(at[0], at[1], x)
		if got := c.At(at[0], at[1]); got != (Cell{R: ' '}) {
			t.Errorf("At%v outside the canvas = %+v, want a blank", at, got)
		}
	}
	for y := range 2 {
		for x := range 3 {
			if c.At(x, y).R != ' ' {
				t.Fatalf("a write outside the canvas landed at %d,%d", x, y)
			}
		}
	}
	c.Set(1, 1, x)
	c.Reset(2, 1)
	if c.W != 2 || c.H != 1 || c.At(1, 0) != (Cell{R: ' '}) || c.At(1, 1) != (Cell{R: ' '}) {
		t.Errorf("Reset(2, 1) left %dx%d with %+v", c.W, c.H, c.At(1, 0))
	}
}
