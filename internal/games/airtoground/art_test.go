package airtoground

import (
	"fmt"
	"slices"
	"strings"
	"testing"

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
// in Lines, where the provenance test checks it; every line passes checkText.
func TestArtLines(t *testing.T) {
	t.Parallel()
	for _, art := range []script.Ls{artTitle, areaMap, areaIcons, areaWrecks, areaHit, burstsAbove, burstsBelow} {
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
}

// The title card leaves the start screen room to fit 80x24 with the front panel: seven
// rows at most, the name on the last.
func TestTitleCard(t *testing.T) {
	t.Parallel()
	if len(artTitle) > 7 {
		t.Errorf("the title card is %d rows; the start screen fits 80x24 with at most 7", len(artTitle))
	}
	if last := artTitle[len(artTitle)-1].Text; !strings.Contains(last, "A I R - T O - G R O U N D") {
		t.Errorf("the name is not on the title card's last row, %q", last)
	}
}

// The map's rows are one width; each icon is on its row once, its wreck is the same width,
// and there is room for its SAM marks, a ^ apiece, before the map's edge.
func TestAreaMap(t *testing.T) {
	t.Parallel()
	for _, l := range areaMap {
		if len(l.Text) != len(areaMap[0].Text) {
			t.Errorf("%q is %d wide, not %d", l.Text, len(l.Text), len(areaMap[0].Text))
		}
	}
	if tableW+len(areaMap[0].Text) > 80 {
		t.Errorf("the table and the map are %d columns", tableW+len(areaMap[0].Text))
	}
	full := newTargets()
	for i := range numTargets {
		row, icon := areaMap[areaRows[i]].Text, areaIcons[i].Text
		if strings.Count(row, icon) != 1 {
			t.Errorf("target %d: %q is on %q %d times", i+1, icon, row, strings.Count(row, icon))
			continue
		}
		if len(areaWrecks[i].Text) != len(icon) {
			t.Errorf("target %d: the wreck %q is not as wide as %q", i+1, areaWrecks[i].Text, icon)
		}
		marks := strings.Index(row, icon) + len(icon) + 1
		if got := row[marks : marks+full[i].sams]; got != strings.Repeat("^", full[i].sams) {
			t.Errorf("target %d: the map has %q where its %d SAM marks go", i+1, got, full[i].sams)
		}
	}
}

// The status as the campaign goes: the map beside the table, SAM sites going down and
// targets cratered, never wider than 80 columns.
func TestStatusMap(t *testing.T) {
	t.Parallel()
	g := New().(*Game)
	g.Start(proto.Env{Seed: 1, Instant: true})
	var b strings.Builder
	snap := func(label string) {
		fmt.Fprintf(&b, "==== %s ====\n", label)
		for _, o := range g.status(-1) {
			s := o.(proto.Say)
			for _, l := range s.Lines {
				checkText(t, l)
			}
			fmt.Fprintf(&b, "%s\n", strings.Join(s.Lines, "\n"))
		}
	}
	snap("the start")
	g.targets[bunker].sams, g.targets[airfield].sams = 1, 0
	g.targets[radar].damage = g.targets[radar].toughness
	g.targets[yard].damage = 1
	snap("two sites down at the bunker, the airfield's both, the radar destroyed")
	for i := range g.targets {
		g.targets[i].damage = g.targets[i].toughness
	}
	snap("every target destroyed")
	golden.AssertString(t, "status_map", b.String())
}

// Each target's burst fits its span (the icon and two columns either side) inside the
// frame, on a row that is neither the frame nor any target's row, so it never covers an
// icon or a SAM mark; every target has one. The hit title is as wide as the map.
func TestBursts(t *testing.T) {
	t.Parallel()
	width := len(areaMap[0].Text)
	if hit := fill(areaHit[0].Text, numTargets); len(hit) != width {
		t.Errorf("%q is %d wide, not %d", hit, len(hit), width)
	}
	iconRow := map[int]bool{}
	for _, y := range areaRows {
		iconRow[y] = true
	}
	for i := range numTargets {
		icon, y := areaIcons[i].Text, areaRows[i]
		x, span := strings.Index(areaMap[y].Text, icon), len(icon)+4
		if burstsAbove[i].Text == "" && burstsBelow[i].Text == "" {
			t.Errorf("target %d has no burst", i+1)
		}
		if x-2 < 1 || x-2+span > width-1 {
			t.Errorf("target %d: the burst's span %d..%d leaves the frame", i+1, x-2, x-2+span)
		}
		for dy, burst := range map[int]string{-1: burstsAbove[i].Text, 1: burstsBelow[i].Text} {
			if burst == "" {
				continue
			}
			if len(burst) > span {
				t.Errorf("target %d: %q is wider than its span of %d", i+1, burst, span)
			}
			if r := y + dy; r <= 0 || r >= len(areaMap)-1 || iconRow[r] {
				t.Errorf("target %d: %q is on map row %d", i+1, burst, r)
			}
		}
	}
}

// The map as each target goes up, the others standing, then the bunker going up last of
// all; the burst and the title show only for the sortie's hit.
func TestHitMaps(t *testing.T) {
	t.Parallel()
	var b strings.Builder
	for i := range numTargets {
		g := New().(*Game)
		g.Start(proto.Env{Seed: 1, Instant: true})
		g.targets[i].damage = g.targets[i].toughness
		fmt.Fprintf(&b, "==== %s ====\n", targetNames[i].Text)
		for _, l := range g.area(i) {
			checkText(t, l)
			if len(l) != len(areaMap[0].Text) {
				t.Errorf("%q is %d wide", l, len(l))
			}
			fmt.Fprintf(&b, "%s\n", l)
		}
		if plain := g.area(-1); plain[0] != areaMap[0].Text || plain[2] != areaMap[2].Text || plain[4] != areaMap[4].Text {
			t.Errorf("%s: the map without a hit still shows it:\n%s", targetNames[i].Text, strings.Join(plain, "\n"))
		}
	}
	g := New().(*Game)
	g.Start(proto.Env{Seed: 1, Instant: true})
	for i := range g.targets {
		g.targets[i].damage = g.targets[i].toughness
	}
	fmt.Fprintf(&b, "==== %s, LAST OF ALL ====\n%s\n", targetNames[bunker].Text, strings.Join(g.area(bunker), "\n"))
	golden.AssertString(t, "hit_maps", b.String())
}

// The status straight after a sortie that destroys a target shows the hit; the next
// STATUS does not.
func TestHitShowsOnce(t *testing.T) {
	t.Parallel()
	for seed := range uint64(50) {
		g := New().(*Game)
		g.Start(proto.Env{Seed: seed, Instant: true})
		outs := g.Handle(proto.LineEvent{Text: "target 5 strike 6 sead 3 escort 3 go"})
		if !g.targets[radar].destroyed() || g.over {
			continue
		}
		if !strings.Contains(sayText(outs), fill(areaHit[0].Text, radar+1)) {
			t.Fatalf("seed %d: the radar went up and the map does not show it:\n%s", seed, sayText(outs))
		}
		if again := sayText(g.Handle(proto.LineEvent{Text: "status"})); strings.Contains(again, "DIRECT HIT") {
			t.Fatalf("seed %d: STATUS shows the old hit:\n%s", seed, again)
		}
		return
	}
	t.Fatal("no seed up to 50 destroys the radar on the first sortie")
}

// sayText is the console text of outs.
func sayText(outs []proto.Output) string {
	var b strings.Builder
	for _, o := range outs {
		if s, ok := o.(proto.Say); ok {
			fmt.Fprintf(&b, "%s\n", strings.Join(s.Lines, "\n"))
		}
	}
	return b.String()
}
