package board

import (
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

func TestDraw(t *testing.T) {
	t.Parallel()
	c := proto.NewCanvas(Width+4, Rows+2)
	Draw(c, 2, 1, func(file, rank int) Glyph {
		switch SquareName(file, rank) {
		case "d4":
			return Glyph{R: 'P', S: proto.StyleBright}
		case "e5":
			return Glyph{R: 'p', S: proto.StyleText}
		case "c3":
			return Glyph{R: 'b', S: proto.StyleText, Disc: true}
		case "e2":
			return Glyph{Mark: true}
		case "f3":
			return Glyph{R: 'N', S: proto.StyleBright, Mark: true}
		}
		return Glyph{}
	})
	rows := strings.Split(c.String(), "\n")
	for i, want := range map[int]string{
		1:  "      " + Files,
		2:  "    ." + strings.Repeat("-", 24) + ".",
		3:  "  8 |   :::   :::   :::   :::| 8", // a8 is light, b8 dark
		6:  "  5 |:::   :::   :p:   :::   | 5", // a piece keeps its square's shading
		7:  "  4 |   :::   :P:   :::   :::| 4",
		8:  "  3 |:::   (b)   :::[N]:::   | 3", // a disc, and a marked piece
		9:  "  2 |   :::   :::[ ]:::   :::| 2", // a marked empty square
		10: "  1 |:::   :::   :::   :::   | 1", // a1 is dark
		11: "    '" + strings.Repeat("-", 24) + "'",
		12: "      " + Files,
	} {
		if rows[i] != want {
			t.Errorf("row %d:\n got %q\nwant %q", i, rows[i], want)
		}
	}
	for _, r := range rows {
		if len(r) > Width+2 {
			t.Errorf("row wider than the board: %q", r)
		}
	}
	golden.AssertString(t, "draw", c.String()+"\n"+c.StyleMap())
}

// The compact board has squares two columns by one row: the piece, then the square's
// shading, or the last move's marks, [] where it left and n< where it landed.
func TestDrawCompact(t *testing.T) {
	t.Parallel()
	c := proto.NewCanvas(CompactWidth+4, Rows+2)
	DrawCompact(c, 2, 1, func(file, rank int) Glyph {
		switch SquareName(file, rank) {
		case "d4":
			return Glyph{R: 'P', S: proto.StyleBright}
		case "e5":
			return Glyph{R: 'p', S: proto.StyleText}
		case "c3":
			return Glyph{R: 'b', S: proto.StyleText, Disc: true} // no discs: a plain piece
		case "e2":
			return Glyph{Mark: true}
		case "f3":
			return Glyph{R: 'N', S: proto.StyleBright, Mark: true}
		}
		return Glyph{}
	})
	rows := strings.Split(c.String(), "\n")
	for i, want := range map[int]string{
		1:  "     " + CompactFiles,
		2:  "    ." + strings.Repeat("-", 16) + ".",
		3:  "  8 |  ::  ::  ::  ::| 8", // a8 is light, b8 dark
		6:  "  5 |::  ::  p:  ::  | 5", // a piece keeps its square's shading
		7:  "  4 |  ::  P:  ::  ::| 4",
		8:  "  3 |::  b:  ::N<::  | 3", // a disc drawn as its piece, and a marked piece
		9:  "  2 |  ::  ::[]::  ::| 2", // a marked empty square
		10: "  1 |::  ::  ::  ::  | 1", // a1 is dark
		11: "    '" + strings.Repeat("-", 16) + "'",
		12: "     " + CompactFiles,
	} {
		if rows[i] != want {
			t.Errorf("row %d:\n got %q\nwant %q", i, rows[i], want)
		}
	}
	for _, r := range rows {
		if len(r) > CompactWidth+2 {
			t.Errorf("row wider than the board: %q", r)
		}
	}
	golden.AssertString(t, "draw_compact", c.String()+"\n"+c.StyleMap())
}

// The compact board looks square: inside its frame, eight ranks of 16 columns, two to a
// square, since a terminal cell is about twice as tall as it is wide. Each square's second
// column is a stipple if the square is dark and blank if light, with a piece on it or not,
// so the squares differ in glyph, not only in style, in every theme and with no colour.
func TestDrawCompactIsSquare(t *testing.T) {
	t.Parallel()
	for _, piece := range []rune{0, 'q'} {
		c := proto.NewCanvas(CompactWidth, Rows)
		DrawCompact(c, 0, 0, func(int, int) Glyph { return Glyph{R: piece, S: proto.StyleText} })
		rows := strings.Split(c.String(), "\n")
		top, bottom := strings.Index(rows[1], "."), strings.LastIndex(rows[1], ".")
		if bottom-top-1 != 16 || strings.Count(c.String(), "|") != 2*8 {
			t.Fatalf("the frame should hold 8 rows of 16 columns:\n%s", c.String())
		}
		for rank := range 8 {
			y := 2 + 7 - rank
			for file := range 8 {
				x := top + 1 + 2*file
				shading := ' '
				if Dark(file, rank) {
					shading = ':'
				}
				want := [2]rune{piece, shading}
				if piece == 0 {
					want[0] = shading
				}
				if got := [2]rune{c.At(x, y).R, c.At(x+1, y).R}; got != want {
					t.Errorf("%s (piece %q): %q, want %q", SquareName(file, rank), piece, string(got[:]), string(want[:]))
				}
				if rank == 0 && rows[0][x] != byte('a'+file) {
					t.Errorf("file letter %c is not over its squares' pieces: %q", 'a'+file, rows[0])
				}
			}
		}
	}
}

func TestDark(t *testing.T) {
	t.Parallel()
	for sq, dark := range map[string]bool{"a1": true, "h1": false, "a8": false, "h8": true, "e4": false, "d4": true} {
		f, r, _ := ParseSquare(sq)
		if Dark(f, r) != dark {
			t.Errorf("%s dark = %v", sq, !dark)
		}
	}
}

func TestText(t *testing.T) {
	t.Parallel()
	lines := Text(func(c *proto.Canvas) {
		c.Put(10, 2, "AB", proto.StyleText, 0)
		c.Put(12, 3, "C", proto.StyleText, 0)
	}, 20, 6)
	if strings.Join(lines, "|") != "AB|  C" {
		t.Errorf("Text drops the blank rows around and the common indent: %q", lines)
	}
}

func TestParseSquare(t *testing.T) {
	t.Parallel()
	for _, s := range []string{"a1", "H8", " e4 "} {
		f, r, ok := ParseSquare(s)
		if !ok || SquareName(f, r) != strings.ToLower(strings.TrimSpace(s)) {
			t.Errorf("ParseSquare(%q) = %d,%d,%v", s, f, r, ok)
		}
	}
	for _, s := range []string{"", "i1", "a9", "a", "e44"} {
		if _, _, ok := ParseSquare(s); ok {
			t.Errorf("ParseSquare(%q) accepted", s)
		}
	}
}
