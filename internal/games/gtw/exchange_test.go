// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package gtw

import (
	"testing"

	"github.com/GhostofGoes/WOPR/internal/proto"
)

// wage fights the film's war: the first strike on the war plan, then your two orders.
func wage(o2, o3 [3]int) (*war, []report) { return wageWith(filmDoctrine, o2, o3) }

// wageWith fights a war on doctrine d.
func wageWith(d doctrine, o2, o3 [3]int) (*war, []report) {
	w := newWar(d)
	rs := []report{w.strike(1, d.plan[0]), w.strike(2, o2), w.strike(3, o3)}
	return w, append(rs, w.final())
}

// warCase is one doctrine GTW plays, with the bounds its quality test proves: whatever you
// order, your cities take at least yours warheads and the other side's at most theirs, and
// your dead are never under dead million.
type warCase struct {
	name                string
	d                   doctrine
	yours, theirs, dead int
}

// scenarioBounds are each scenario's bounds, in the scenarios' order.
var scenarioBounds = [][3]int{
	{1257, 825, 80},  // U.S. FIRST STRIKE: your first strike disarms nothing
	{1032, 750, 76},  // USSR FIRST STRIKE
	{1054, 750, 77},  // NATO / WARSAW PACT
	{1077, 788, 77},  // FAR EAST STRATEGY: half your bombers escape the ground
	{1053, 750, 77},  // US USSR ESCALATION
	{1485, 500, 83},  // USSR CHINA ATTACK: a third of your forces are gone
	{1032, 750, 76},  // SEATO DECAPITATING
	{1050, 750, 77},  // CUBAN PROVOCATION
	{1032, 825, 76},  // INADVERTENT INCIDENT: one stray missile, and your bombers stay ready
	{1459, 1100, 83}, // ATLANTIC HEAVY
	{1032, 750, 76},  // PACIFIC TERRITORIAL
	{229, 150, 38},   // ARCTIC MINIMAL: a fifth of the forces, and still no winner
}

// wars is the film's war and every scenario's.
func wars(t *testing.T) []warCase {
	t.Helper()
	if len(scenarioBounds) != len(scenarios) {
		t.Fatalf("%d scenarios, %d with bounds", len(scenarios), len(scenarioBounds))
	}
	out := []warCase{{"film", filmDoctrine, 1144, 750, 75}}
	for i, s := range scenarios {
		b := scenarioBounds[i]
		out = append(out, warCase{s.name.Text, s.d, b[0], b[1], b[2]})
	}
	return out
}

// check holds the guarantees for one war: nothing you order wins, and nothing is created
// or lost from the books.
func check(t *testing.T, c warCase, w *war, rs []report, label string) {
	t.Helper()
	if you, them := w.t[you].Cities, w.t[wopr].Cities; you < c.yours || them > c.theirs {
		t.Fatalf("%s: warheads on cities: yours %d (want >= %d), WOPR's side %d (want <= %d)", label, you, c.yours, them, c.theirs)
	}
	fired := [2]salvo{}
	for _, r := range rs {
		if r.fire[you].ICBMCity != 0 {
			t.Fatalf("%s: your ICBMs reached cities at stage %d", label, r.stage)
		}
		for s := range 2 {
			fired[s].ICBMSilo += r.fire[s].ICBMSilo + r.fire[s].ICBMCity
			fired[s].SLBM += r.fire[s].SLBM
		}
	}
	for s := range 2 {
		f, st, tl := w.f[s], w.d.forces[s], w.t[s]
		if f.ICBM < 0 || f.SLBM < 0 || f.Ground != 0 || f.Air != 0 {
			t.Fatalf("%s: side %d ends with %+v", label, s, f)
		}
		if fired[s].ICBMSilo+tl.ICBMLost+f.ICBM != st.ICBM || fired[s].SLBM+f.SLBM != st.SLBM {
			t.Fatalf("%s: side %d's missiles do not add up: fired %+v, lost %d, left %+v", label, s, fired[s], tl.ICBMLost, f)
		}
	}
	if w.f[wopr].SLBM != 0 || w.f[wopr].ICBM != 0 {
		t.Fatalf("%s: WOPR held back %+v at DEFCON 1", label, w.f[wopr])
	}
}

// The quality test: no allocation wins, in the film's war or in any scenario. ICBM orders
// run at every percent (they alone move what lands on your cities); SLBMs and bombers at the
// extremes and the middle; then random orders at every percent. Each war's floor for your
// cities is above its ceiling for theirs, and both are reached: they are the war's own.
func TestNoAllocationWins(t *testing.T) {
	t.Parallel()
	for _, c := range wars(t) {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.yours <= c.theirs {
				t.Fatalf("the bounds let you win: yours >= %d, theirs <= %d", c.yours, c.theirs)
			}
			ends := []int{0, 50, 100}
			least, most := 1<<30, 0
			for i2 := 0; i2 <= 100; i2++ {
				for i3 := 0; i3 <= 100; i3++ {
					for _, s2 := range ends {
						for _, s3 := range ends {
							for _, b := range []int{0, 100} {
								w, rs := wageWith(c.d, [3]int{i2, s2, b}, [3]int{i3, s3, 100 - b})
								check(t, c, w, rs, "grid")
								least, most = min(least, w.t[you].Cities), max(most, w.t[wopr].Cities)
							}
						}
					}
				}
			}
			if least != c.yours || most != c.theirs {
				t.Errorf("the grid reaches yours %d and theirs %d, want the bounds %d and %d", least, most, c.yours, c.theirs)
			}
			r := proto.NewRand(1, 2)
			for range 2000 {
				o := func() [3]int { return [3]int{r.IntN(101), r.IntN(101), r.IntN(101)} }
				w, rs := wageWith(c.d, o(), o())
				check(t, c, w, rs, "random")
			}
		})
	}
}

// The doctrine's numbers on the film path (Enter at every prompt) and on HOLD, HOLD.
func TestDoctrine(t *testing.T) {
	t.Parallel()
	w, rs := wage(filmDoctrine.plan[1], filmDoctrine.plan[2])
	type stage struct{ you, wopr salvo }
	want := []stage{
		{salvo{ICBMSilo: 250, SLBM: 150}, salvo{ICBMSilo: 313, SLBM: 188, Bombers: 375}},
		{salvo{ICBMSilo: 250, SLBM: 225, Bombers: 200}, salvo{ICBMSilo: 313, SLBM: 262}},
		{salvo{SLBM: 225}, salvo{ICBMCity: 224}},
		{salvo{Bombers: 200}, salvo{SLBM: 300, Bombers: 375}},
	}
	for i, r := range rs {
		if r.fire[you] != want[i].you || r.fire[wopr] != want[i].wopr {
			t.Errorf("stage %d: you %+v, WOPR %+v; want %+v", i+1, r.fire[you], r.fire[wopr], want[i])
		}
	}
	if got := w.t; got != [2]tally{{1256, 626, 500, 150}, {750, 500, 400, 93}} {
		t.Errorf("film path tallies %+v", got)
	}
	w, _ = wage([3]int{}, [3]int{})
	if got := w.t; got != [2]tally{{1144, 938, 750, 300}, {150, 250, 200, 93}} {
		t.Errorf("hold, hold tallies %+v", got)
	}
}

// Whatever the orders and the noise, in every war, your civilian and human rows are never
// better than the other side's, and your dead are never under the war's floor (75 million in
// the film's).
func TestRatiosFavourNoOne(t *testing.T) {
	t.Parallel()
	for _, c := range wars(t) {
		r := proto.NewRand(3, 4)
		for range 2000 {
			o := func() [3]int { return [3]int{r.IntN(101), r.IntN(101), r.IntN(101)} }
			w, _ := wageWith(c.d, o(), o())
			var noise [10]int
			for i := range noise {
				noise[i] = 97 + r.IntN(7)
			}
			got := w.ratios(noise)
			for row := 5; row < 12; row++ {
				if got[0][row] < got[1][row] {
					t.Fatalf("%s: row %d favours you: %v", c.name, row, got)
				}
			}
			if got[0][11] < c.dead {
				t.Fatalf("%s: your dead %d million, want at least %d", c.name, got[0][11], c.dead)
			}
			if f := w.d.forces[you]; got[0][0] != min(100*w.t[you].BombersLost/f.Ground, 100) || got[0][1] != min(100*w.t[you].ICBMLost/f.ICBM, 100) {
				t.Fatalf("%s: the BOMBERS and ICBM'S rows are the exchange's losses in percent: %v, %+v", c.name, got, w.t)
			}
		}
	}
	// The military rows take the worse of city and silo damage. Firing every ICBM at strike 2
	// puts 750 warheads on WOPR's silos and 150 on its cities, so its silos set them.
	w, _ := wage([3]int{100, 0, 0}, [3]int{})
	if tl := w.t[wopr]; tl.SiloHits != 750 || tl.Cities != 150 {
		t.Fatalf("WOPR's tally %+v", tl)
	}
	noise := [10]int{100, 100, 100, 100, 100, 100, 100, 100, 100, 100}
	if r := w.ratios(noise); [3]int(r[1][2:5]) != [3]int{26, 61, 45} {
		t.Errorf("WOPR's ATTACK SUBS, TACTICAL AIRCRAFT and GROUND FORCES: %v, want silo damage's [26 61 45]", r[1][2:5])
	}
}

// Each system has one effect a player can learn, in every war: ICBM orders alone decide what
// lands on your cities (holding them soaks up WOPR's ICBMs; firing them turns WOPR's on your
// cities), and your SLBMs and bombers alone decide what lands on theirs.
func TestChoicesMatter(t *testing.T) {
	t.Parallel()
	for _, c := range wars(t) {
		r := proto.NewRand(5, 6)
		for range 2000 {
			i2, i3 := r.IntN(101), r.IntN(101)
			a, _ := wageWith(c.d, [3]int{i2, r.IntN(101), r.IntN(101)}, [3]int{i3, r.IntN(101), r.IntN(101)})
			b, rs := wageWith(c.d, [3]int{i2, r.IntN(101), r.IntN(101)}, [3]int{i3, r.IntN(101), r.IntN(101)})
			if a.t[you].Cities != b.t[you].Cities {
				t.Fatalf("%s: your cities moved with SLBM or bomber orders: %d, %d", c.name, a.t[you].Cities, b.t[you].Cities)
			}
			theirs := 0
			for _, rp := range rs {
				theirs += rp.fire[you].SLBM
			}
			theirs += rs[len(rs)-1].fire[you].Bombers - rs[len(rs)-1].fire[you].Bombers/shootDown
			if b.t[wopr].Cities != theirs {
				t.Fatalf("%s: their cities %d, want your SLBMs and arriving bombers %d", c.name, b.t[wopr].Cities, theirs)
			}
		}
	}
	hold, _ := wage([3]int{}, [3]int{})
	fire, _ := wage([3]int{100, 0, 0}, [3]int{})
	if hold.t[you].Cities != 1144 || fire.t[you].Cities != 1657 || hold.t[you].ICBMLost != 750 {
		t.Errorf("hold: %+v; fire your ICBMs at strike 2: %+v", hold.t[you], fire.t[you])
	}
}

// Each scenario's first strike is its own: who launches, and how much. WOPR's first strike
// leaves before yours, and the limited and minimal wars open small.
func TestScenarioFirstStrikes(t *testing.T) {
	t.Parallel()
	want := [][2]salvo{ // yours, WOPR's, in the scenarios' order
		{{ICBMSilo: 500, SLBM: 300, Bombers: 300}, {ICBMSilo: 625, SLBM: 375, Bombers: 375}},
		{{}, {ICBMSilo: 500, SLBM: 150, Bombers: 375}},
		{{ICBMSilo: 50, SLBM: 60}, {ICBMSilo: 63, SLBM: 75, Bombers: 375}},
		{{ICBMSilo: 100, SLBM: 150, Bombers: 150}, {ICBMSilo: 125, SLBM: 188, Bombers: 375}},
		{{ICBMSilo: 50, SLBM: 30}, {ICBMSilo: 75, SLBM: 45, Bombers: 375}},
		{{ICBMSilo: 175, SLBM: 100}, {ICBMSilo: 219, SLBM: 125, Bombers: 375}},
		{{SLBM: 300}, {ICBMSilo: 125, SLBM: 375, Bombers: 375}},
		{{ICBMSilo: 250, SLBM: 150}, {SLBM: 120}},
		{{}, {SLBM: 1}},
		{{ICBMSilo: 50, SLBM: 400}, {ICBMSilo: 63, SLBM: 500, Bombers: 250}},
		{{SLBM: 150}, {ICBMSilo: 62, SLBM: 188, Bombers: 375}},
		{{ICBMSilo: 50, SLBM: 30}, {ICBMSilo: 63, SLBM: 38, Bombers: 75}},
	}
	if len(want) != len(scenarios) {
		t.Fatalf("%d scenarios, %d first strikes", len(scenarios), len(want))
	}
	for i, s := range scenarios {
		w := newWar(s.d)
		r := w.strike(1, s.d.plan[0])
		if r.fire[you] != want[i][0] || r.fire[wopr] != want[i][1] {
			t.Errorf("%s: you %+v, WOPR %+v; want %+v", s.name.Text, r.fire[you], r.fire[wopr], want[i])
		}
		// The forces table shows the first launch alone first: WOPR's in a WOPR first strike.
		first, second := you, wopr
		if s.d.woprFirst() {
			first, second = wopr, you
		}
		if r.snap[0][second] != s.d.forces[second] || r.snap[0][first] == s.d.forces[first] && r.fire[first] != (salvo{}) {
			t.Errorf("%s: after the first launch the forces are %+v", s.name.Text, r.snap[0])
		}
	}
}

// Grounded bombers lose a third under attack, but never more than the missiles that land: a
// stray SLBM destroys one bomber, and the film's first strike still a third.
func TestStrayMissile(t *testing.T) {
	t.Parallel()
	w := newWar(with(filmDoctrine, first(0, 1, 0)))
	if r := w.strike(1, [3]int{}); r.ground[you] != 1 || w.f[you].Ground != 299 {
		t.Errorf("one SLBM: lost on the ground %d, bombers left %d; want 1 and 299", r.ground[you], w.f[you].Ground)
	}
	w = newWar(filmDoctrine)
	if w.strike(1, filmDoctrine.plan[0]); w.f[you].Ground != 200 {
		t.Errorf("the film's first strike leaves %d bombers on the ground, want 200", w.f[you].Ground)
	}
}
