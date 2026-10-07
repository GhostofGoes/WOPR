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
