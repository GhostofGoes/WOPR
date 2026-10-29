// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package gtw

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/assets"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// What WOPR reacts to, for each kind of order at a strike prompt.
func TestReactionTo(t *testing.T) {
	t.Parallel()
	plan2, plan3 := warPlan[1], warPlan[2]
	for _, tc := range []struct {
		alloc, plan [3]int
		planned     bool
		icbms       int
		want        reaction
	}{
		{plan2, plan2, false, 500, reactPlan},
		{plan3, plan3, true, 250, reactPlanAgain},
		{plan3, plan3, false, 250, reactPlan}, // ALL at strike 3 is the plan
		{[3]int{100, 100, 100}, plan2, false, 500, reactAll},
		{[3]int{}, plan2, true, 500, reactHold},
		{[3]int{0, 100, 0}, plan2, false, 500, reactSilosFull},
		{[3]int{0, 0, 100}, plan3, false, 500, reactSilosFull},
		{[3]int{0, 100, 0}, plan3, false, 0, reactPartial}, // no ICBMs left to keep
		{[3]int{100, 0, 100}, plan2, false, 500, reactPartial},
		{[3]int{1, 0, 0}, plan2, false, 500, reactPartial},
	} {
		if got := reactionTo(tc.alloc, tc.plan, tc.planned, tc.icbms); got != tc.want {
			t.Errorf("%v (plan %v, planned %v, %d ICBMs): %d, want %d", tc.alloc, tc.plan, tc.planned, tc.icbms, got, tc.want)
		}
	}
}

// Every reaction has two or more variants, so a seed varies them and a repeat reads
// differently, and every variant fits the board's last row with any target named in it.
func TestRemarksFit(t *testing.T) {
	t.Parallel()
	longest := ""
	for _, p := range append(slices.Clone(assets.Cities), assets.Targets...) {
		if len(p.Name) > len(longest) {
			longest = p.Name
		}
	}
	for r, lines := range reactLines {
		if len(lines) < 2 {
			t.Errorf("reaction %d has %d variants, want 2 or more", r, len(lines))
		}
		for _, l := range lines {
			if text := fill(l, longest); len(text) > 80 || strings.Contains(text, "#") {
				t.Errorf("reaction %d: %q is %d wide", r, text, len(text))
			}
		}
	}
}

// remarks plays a paced war with seed against the film's targets, answering each strike
// prompt with the next of given (Enter once they run out), and returns each stage's remark
// and the board's last row at frame 10 and at remarkFrames of each stage.
func remarks(t *testing.T, seed uint64, given ...string) (said, early, late []string) {
	t.Helper()
	g := New().(*Game)
	g.Start(proto.Env{Seed: seed, Width: 80, Height: 19})
	g.Handle(proto.LineEvent{Text: "2"})
	g.Handle(proto.LineEvent{Text: "Las Vegas, Seattle"})
	g.Handle(proto.LineEvent{Text: ""})
	foot := func() string {
		c := proto.NewCanvas(80, 19)
		g.View(c)
		return strings.TrimSpace(strings.Split(c.String(), "\n")[footRow])
	}
	stage := 0
	for g.phase != assessed {
		if g.phase == orders {
			next := ""
			if len(given) > 0 {
				next, given = given[0], given[1:]
			}
			g.Handle(proto.LineEvent{Text: next})
			continue
		}
		if g.stage != stage {
			stage = g.stage
			said = append(said, g.remarked)
		}
		switch g.frame {
		case 10:
			early = append(early, foot())
		case remarkFrames:
			late = append(late, foot())
		}
		g.Handle(proto.TickEvent{Dt: frame})
	}
	return said, early, late
}

// WOPR remarks on each stage's order as it flies, on the board's last row: typed in, then
// gone, the legend back for the rest of the flight. Each order gets its own kind of remark,
// and a kind that comes twice reads differently.
func TestRemarksFollowOrders(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orders []string
		want   []reaction
	}{
		{nil, []reaction{reactFirst, reactPlan, reactPlanAgain, reactFinal}},
		{[]string{"hold", "stop"}, []reaction{reactFirst, reactHold, reactHold, reactFinal}},
		{[]string{"all"}, []reaction{reactFirst, reactAll, reactEmpty, reactFinal}},
		{[]string{"auto"}, []reaction{reactFirst, reactAuto, reactAutoStrike, reactFinal}},
		{[]string{"subs", "icbm 50"}, []reaction{reactFirst, reactSilosFull, reactPartial, reactFinal}},
		{[]string{"40", ""}, []reaction{reactFirst, reactPartial, reactPlan, reactFinal}},
		{[]string{"100 100 0", "hold"}, []reaction{reactFirst, reactPartial, reactQuiet, reactFinal}}, // WOPR has nothing left to answer with
	} {
		label := strings.Join(tc.orders, "|")
		for seed := range uint64(8) {
			said, early, late := remarks(t, seed, tc.orders...)
			if len(said) != len(tc.want) {
				t.Fatalf("%s, seed %d: %d remarks %q, want %d", label, seed, len(said), said, len(tc.want))
			}
			for i, r := range tc.want {
				var texts []string
				for _, l := range reactLines[r] {
					texts = append(texts, fill(l, "LAS VEGAS"))
				}
				if !slices.Contains(texts, said[i]) {
					t.Errorf("%s, seed %d: stage %d says %q, want one of %q", label, seed, i+1, said[i], texts)
				}
				if early[i] != strings.TrimSpace(said[i][:min(len(said[i]), remarkChars*11)]) {
					t.Errorf("%s, seed %d: stage %d's last row at frame 10 is %q, want %q typed in", label, seed, i+1, early[i], said[i])
				}
				if late[i] != lineLegend[0].Text {
					t.Errorf("%s, seed %d: stage %d's last row at frame %d is %q, want the legend", label, seed, i+1, remarkFrames, late[i])
				}
			}
			if tc.want[1] == tc.want[2] && said[1] == said[2] {
				t.Errorf("%s, seed %d: the same remark twice: %q", label, seed, said[1])
			}
		}
	}
}

// The remarks come from the seed: the same seed and orders say the same, and seeds vary them.
func TestRemarksAreSeeded(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	for seed := range uint64(16) {
		a, _, _ := remarks(t, seed)
		b, _, _ := remarks(t, seed)
		if fmt.Sprint(a) != fmt.Sprint(b) {
			t.Errorf("seed %d: %q, then %q", seed, a, b)
		}
		seen[a[1]] = true
	}
	if len(seen) < 2 {
		t.Errorf("sixteen seeds gave one remark on the plan: %v", seen)
	}
}

// The film's board makes no remarks: movie mode's war is the film's, and nobody gives orders.
func TestFilmMakesNoRemarks(t *testing.T) {
	t.Parallel()
	f := NewFilm(1, false)
	for f.Stage() < FilmRatios {
		f.Advance(frame, FilmRatios)
		if f.g.remarked != "" {
			t.Fatalf("stage %d: the film's board says %q", f.g.stage, f.g.remarked)
		}
	}
}
