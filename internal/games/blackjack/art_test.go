package blackjack

import (
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/script"
)

// checkArt reports art that is not printable ASCII capitals, is taller than 12 rows or
// wider than 80 columns, or is missing from lines (where the provenance test checks it).
func checkArt(t *testing.T, art script.Ls, lines []script.Ls) {
	t.Helper()
	if len(art) == 0 || len(art) > 12 {
		t.Errorf("the art is %d rows; a block on the console is at most 12", len(art))
	}
	listed := false
	for _, block := range lines {
		listed = listed || &block[0] == &art[0]
	}
	if !listed {
		t.Error("the art is not in Lines")
	}
	for _, l := range art {
		for _, r := range l.Text {
			if r < 0x20 || r > 0x7e {
				t.Errorf("%q is not printable ASCII", l.Text)
				break
			}
		}
		if len(l.Text) > 80 || l.Text != strings.ToUpper(l.Text) || l.Text != strings.TrimRight(l.Text, " ") {
			t.Errorf("%q: wider than 80 columns, lower case, or trailing spaces", l.Text)
		}
		if l.Prov != script.Original {
			t.Errorf("%q is tagged %q, not original", l.Text, l.Prov)
		}
	}
}

func TestTitleArt(t *testing.T) {
	t.Parallel()
	checkArt(t, artTitle, Lines)
	for _, block := range Lines {
		for _, l := range block {
			if strings.ContainsFunc(l.Text, func(r rune) bool { return r < 0x20 || r > 0x7e }) {
				t.Errorf("%q is not printable ASCII", l.Text)
			}
		}
	}
}
