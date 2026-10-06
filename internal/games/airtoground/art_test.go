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
	for _, art := range []script.Ls{artTitle, artImpact, areaMap, areaIcons, areaWrecks} {
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
		for _, o := range g.status() {
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
