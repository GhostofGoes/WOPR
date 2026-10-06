package board

import (
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/proto"
)

func TestDraw(t *testing.T) {
	t.Parallel()
	c := proto.NewCanvas(80, 12)
	Draw(c, "BOARD", func(file, rank int) Glyph {
		if file == 4 && rank == 3 {
			return Glyph{R: 'P', S: proto.StyleBright}
		}
		if (file+rank)%2 == 0 {
			return Glyph{R: '.', S: proto.StyleDim}
		}
		return Glyph{}
	})
	rows := strings.Split(c.String(), "\n")
	if !strings.Contains(rows[0], "BOARD") || !strings.HasSuffix(strings.TrimRight(rows[2], " "), "8      .     .     .     .") {
		t.Errorf("rank 8 row %q", rows[2])
	}
	if got := strings.TrimSpace(rows[10]); got != "a  b  c  d  e  f  g  h" {
		t.Errorf("files row %q", got)
	}
	if !strings.Contains(rows[6], "P") {
		t.Errorf("e4 should hold P: %q", rows[6])
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
