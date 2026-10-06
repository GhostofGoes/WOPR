package fightercombat

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// checkText reports a line that is not printable ASCII in capitals or is wider than 80
// columns.
func checkText(t *testing.T, line string) {
	t.Helper()
	if strings.ContainsFunc(line, func(r rune) bool { return r < 0x20 || r > 0x7e }) {
		t.Errorf("%q is not printable ASCII", line)
	}
	if len(line) > 80 || line != strings.ToUpper(line) {
		t.Errorf("%q: wider than 80 columns, or lower case", line)
	}
}

// Every art block is at most 12 rows without trailing spaces, tagged original and listed
// in Lines (where the provenance test checks it); every line, and the splash with each
// call filled in, passes checkText.
func TestArtLines(t *testing.T) {
	t.Parallel()
	for _, art := range []script.Ls{artTitle, artSplash, artEject} {
		if len(art) == 0 || len(art) > 12 {
			t.Errorf("%q...: %d rows; a console block is at most 12", art[0].Text, len(art))
		}
		if !slices.ContainsFunc(Lines, func(b script.Ls) bool { return &b[0] == &art[0] }) {
			t.Errorf("%q... is not in Lines", art[0].Text)
		}
		for _, l := range art {
			if l.Text != strings.TrimRight(l.Text, " ") {
				t.Errorf("%q has trailing spaces", l.Text)
			}
		}
	}
	for _, block := range Lines {
		for _, l := range block {
			checkText(t, l.Text)
			if l.Prov != script.Original {
				t.Errorf("%q is tagged %q, not original", l.Text, l.Prov)
			}
		}
	}
	for _, both := range []bool{false, true} {
		for _, l := range splash(both) {
			checkText(t, l)
		}
	}
}

// The tactical picture for every aspect and range, high and low, whole and damaged: four
// rows each, and the aircraft never overlap.
func TestPictures(t *testing.T) {
	t.Parallel()
	var b strings.Builder
	for pos := -1; pos <= 1; pos++ {
		for dist := rangeClose; dist <= rangeLong; dist++ {
			for _, c := range []struct{ energy, hits [2]int }{{[2]int{3, 3}, [2]int{0, 0}}, {[2]int{0, 6}, [2]int{1, 1}}} {
				g := &Game{pos: pos, dist: dist, energy: c.energy, hits: c.hits}
				pic := g.picture()
				fmt.Fprintf(&b, "==== aspect %d, range %d, energy %v, hits %v ====\n%s\n", pos, dist, c.energy, c.hits, strings.Join(pic, "\n"))
				if len(pic) != 4 {
					t.Errorf("%+v: %d rows", c, len(pic))
				}
				for _, l := range pic {
					checkText(t, l)
				}
				if n := strings.Count(strings.Join(pic[:3], ""), picNames[1].Text); n != 1 {
					t.Errorf("aspect %d range %d %+v: WOPR is drawn %d times (overwritten?)", pos, dist, c, n)
				}
			}
		}
	}
	golden.AssertString(t, "pictures", b.String())
}

// How fights end: the first seeded fight the sensible pilot wins shows the splash, the
// first it loses shows the ejection, and a mutual kill shows SPLASH TWO.
func TestEndings(t *testing.T) {
	t.Parallel()
	want := []struct {
		outcome proto.Outcome
		art     []string
	}{
		{proto.Win, splash(false)},
		{proto.Loss, artEject.Texts()},
		{proto.NoWinner, splash(true)},
	}
	var b strings.Builder
	for _, w := range want {
		found := false
		for seed := uint64(0); seed < 300 && !found; seed++ {
			g := New().(*Game)
			s := testkit.Game(t, g, info, "", seed)
			for range 40 {
				if asking, _ := s.Asking(); !asking {
					break
				}
				s.Type(pilot(g))
			}
			if res, _ := s.Result(); res.Outcome != w.outcome {
				continue
			}
			found = true
			if !s.Contains(strings.Join(w.art, "\n")) {
				t.Errorf("seed %d, outcome %d: the art is missing:\n%s", seed, w.outcome, s.Transcript())
			}
			if wide := s.Wide(80); len(wide) > 0 {
				t.Errorf("seed %d: lines wider than 80 columns: %q", seed, wide)
			}
			lines := strings.Split(strings.TrimSuffix(s.Transcript(), "\n"), "\n")
			fmt.Fprintf(&b, "==== seed %d, the last turn ====\n%s\n", seed, strings.Join(lines[max(len(lines)-len(w.art)-9, 0):], "\n"))
		}
		if !found && w.outcome != proto.NoWinner { // a mutual kill is rare; the others are not
			t.Errorf("no seeded fight ended with outcome %d", w.outcome)
		}
	}
	golden.AssertString(t, "endings", b.String())
}
