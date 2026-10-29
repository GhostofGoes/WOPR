// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package gtw

import (
	"testing"

	"github.com/GhostofGoes/WOPR/internal/assets"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

var sides = []assets.Side{assets.US, assets.USSR}

// wage fights a whole war as side: the first strike on the war plan, then your two orders.
func wage(side assets.Side, o2, o3 [3]int) (*war, []report) {
	w := newWar(side)
	rs := []report{w.strike(1, warPlan[0]), w.strike(2, o2), w.strike(3, o3)}
	return w, append(rs, w.final())
}

// bounds are each side's guarantees, whatever it orders: the least that lands on your cities
// and the most that lands on theirs.
var bounds = map[assets.Side][2]int{assets.US: {1366, 850}, assets.USSR: {1157, 776}}

// check holds the guarantees for one war: nothing you order wins, your ICBMs reach cities
// only once WOPR's silos could absorb no more, and nothing is created or lost from the books.
func check(t *testing.T, side assets.Side, w *war, rs []report, label string) {
	t.Helper()
	b := bounds[side]
	if you, them := w.t[you].Cities, w.t[wopr].Cities; you < b[0] || them > b[1] || you <= them {
		t.Fatalf("%s as %d: warheads on cities: yours %d (want >= %d, and more than WOPR's side), WOPR's side %d (want <= %d)",
			label, side, you, b[0], them, b[1])
	}
	fired := [2]salvo{}
	for _, r := range rs {
		if f := r.fire[you]; f.ICBMCity != 0 && f.ICBMSilo != quarterMore(r.snap[0][wopr].ICBM) {
			t.Fatalf("%s as %d: your ICBMs reached cities at stage %d while WOPR's silos took only %d", label, side, r.stage, f.ICBMSilo)
		}
		for s := range 2 {
			fired[s].ICBMSilo += r.fire[s].ICBMSilo + r.fire[s].ICBMCity
			fired[s].SLBM += r.fire[s].SLBM
		}
	}
	for s := range 2 {
		f, st, tl := w.f[s], w.start[s], w.t[s]
		if f.ICBM < 0 || f.SLBM < 0 || f.Ground != 0 || f.Air != 0 {
			t.Fatalf("%s as %d: side %d ends with %+v", label, side, s, f)
		}
		if fired[s].ICBMSilo+tl.ICBMLost+f.ICBM != st.ICBM || fired[s].SLBM+f.SLBM != st.SLBM {
			t.Fatalf("%s as %d: side %d's missiles do not add up: fired %+v, lost %d, left %+v", label, side, s, fired[s], tl.ICBMLost, f)
		}
	}
	if w.f[wopr].SLBM != 0 || w.f[wopr].ICBM != 0 {
		t.Fatalf("%s as %d: WOPR held back %+v at DEFCON 1", label, side, w.f[wopr])
	}
}

// Each nation has its own forces, and WOPR always holds the other nation's with a quarter
// more of each: more in all than you, whichever side you take.
func TestForcesByNation(t *testing.T) {
	t.Parallel()
	total := func(f forces) int { return f.ICBM + f.SLBM + f.Ground }
	for _, side := range sides {
		w, e := newWar(side), nationForces[enemyOf(side)]
		if w.start[you] != nationForces[side] {
			t.Errorf("as %d you start with %+v, want %+v", side, w.start[you], nationForces[side])
		}
		if want := (forces{ICBM: quarterMore(e.ICBM), SLBM: quarterMore(e.SLBM), Ground: quarterMore(e.Ground)}); w.start[wopr] != want {
			t.Errorf("as %d WOPR starts with %+v, want %+v", side, w.start[wopr], want)
		}
		if total(w.start[wopr]) <= total(w.start[you]) {
			t.Errorf("as %d WOPR holds %+v, no more than your %+v", side, w.start[wopr], w.start[you])
		}
	}
	us, ussr := nationForces[assets.US], nationForces[assets.USSR]
	if us.ICBM >= ussr.ICBM || us.SLBM <= ussr.SLBM || us.Ground <= ussr.Ground {
		t.Errorf("the United States %+v is not lighter in ICBMs and heavier at sea and in the air than the Soviet Union %+v", us, ussr)
	}
}

// The quality test: no allocation wins, for either side. ICBM orders run at every percent
// (they alone move what lands on your cities); SLBMs and bombers at the extremes and the
// middle; then random orders at every percent.
func TestNoAllocationWins(t *testing.T) {
	t.Parallel()
	ends := []int{0, 50, 100}
	for _, side := range sides {
		for i2 := 0; i2 <= 100; i2++ {
			for i3 := 0; i3 <= 100; i3++ {
				for _, s2 := range ends {
					for _, s3 := range ends {
						for _, b := range []int{0, 100} {
							w, rs := wage(side, [3]int{i2, s2, b}, [3]int{i3, s3, 100 - b})
							check(t, side, w, rs, "grid")
						}
					}
				}
			}
		}
		r := proto.NewRand(1, 2)
		for range 2000 {
			o := func() [3]int { return [3]int{r.IntN(101), r.IntN(101), r.IntN(101)} }
			w, rs := wage(side, o(), o())
			check(t, side, w, rs, "random")
		}
	}
}

// The doctrine's numbers on the film path (the Soviet Union, Enter at every prompt), and the
// tallies on it and on HOLD, HOLD for both sides.
func TestDoctrine(t *testing.T) {
	t.Parallel()
	_, rs := wage(assets.USSR, warPlan[1], warPlan[2])
	type stage struct{ you, wopr salvo }
	want := []stage{
		{salvo{ICBMSilo: 325, SLBM: 125}, salvo{ICBMSilo: 407, SLBM: 157, Bombers: 375}},
		{salvo{ICBMSilo: 325, SLBM: 187, Bombers: 100}, salvo{ICBMSilo: 407, SLBM: 234}},
		{salvo{SLBM: 188}, salvo{SLBM: 134}},
		{salvo{Bombers: 100}, salvo{SLBM: 350, Bombers: 375}},
	}
	for i, r := range rs {
		if r.fire[you] != want[i].you || r.fire[wopr] != want[i].wopr {
			t.Errorf("stage %d: you %+v, WOPR %+v; want %+v", i+1, r.fire[you], r.fire[wopr], want[i])
		}
	}
	tallies := map[assets.Side][2][2]tally{ // the war plan's, then HOLD, HOLD's
		assets.US:   {{{1467, 564, 450, 150}, {850, 450, 360, 47}}, {{1366, 845, 675, 300}, {175, 225, 180, 47}}},
		assets.USSR: {{{1157, 814, 650, 75}, {575, 650, 311, 93}}, {{1157, 865, 691, 150}, {125, 325, 260, 93}}},
	}
	for _, side := range sides {
		for i, o := range [2][2][3]int{{warPlan[1], warPlan[2]}, {}} {
			if w, _ := wage(side, o[0], o[1]); w.t != tallies[side][i] {
				t.Errorf("as %d, %v: tallies %+v, want %+v", side, o, w.t, tallies[side][i])
			}
		}
	}
}

// The two nations play differently. As the United States, the ICBMs you fire turn WOPR's on
// your cities, and none of yours ever reaches theirs: holding them all is best. As the Soviet
// Union, your ICBMs outnumber the American silos: half of them at strike 2, and then the rest
// at strike 3, cost your cities nothing, and the ones the silos cannot absorb hit American
// cities, so the best orders fire some at strike 2 and the rest at strike 3.
func TestSidesPlayDifferently(t *testing.T) {
	t.Parallel()
	// best is the ICBM orders, at strikes 2 and 3, that leave the narrowest gap between the
	// warheads on your cities and on theirs, with SLBMs held for strike 3 and bombers sent
	// at strike 2. leak reports whether any of those orders put your ICBMs on cities.
	best := func(side assets.Side) (i2, i3 int, leak bool) {
		gap := 1 << 30
		for a := 0; a <= 100; a++ {
			for b := 0; b <= 100; b++ {
				w, rs := wage(side, [3]int{a, 0, 100}, [3]int{b, 100, 0})
				for _, r := range rs {
					leak = leak || r.fire[you].ICBMCity > 0
				}
				if d := w.t[you].Cities - w.t[wopr].Cities; d < gap {
					gap, i2, i3 = d, a, b
				}
			}
		}
		return i2, i3, leak
	}
	if i2, i3, leak := best(assets.US); i2 != 0 || i3 != 0 || leak {
		t.Errorf("as the United States the best ICBM orders are %d and %d, and ICBMs reach cities: %v; want 0, 0, false", i2, i3, leak)
	}
	if i2, i3, leak := best(assets.USSR); i2 == 0 || i3 == 0 || !leak {
		t.Errorf("as the Soviet Union the best ICBM orders are %d and %d, and ICBMs reach cities: %v; want both fired, true", i2, i3, leak)
	}
	hold, _ := wage(assets.USSR, [3]int{0, 0, 100}, [3]int{})
	for _, i3 := range []int{0, 100} {
		half, _ := wage(assets.USSR, [3]int{50, 0, 100}, [3]int{i3, 0, 0})
		if half.t[you].Cities != hold.t[you].Cities {
			t.Errorf("as the Soviet Union half your ICBMs at strike 2 and %d%% at strike 3 put %d warheads on your cities, holding them %d",
				i3, half.t[you].Cities, hold.t[you].Cities)
		}
	}
}

// Whatever the side, the orders and the noise, your civilian and human rows are never better
// than the other side's, and your dead are never under 75 million.
func TestRatiosFavourNoOne(t *testing.T) {
	t.Parallel()
	for _, side := range sides {
		r := proto.NewRand(3, 4)
		for range 2000 {
			o := func() [3]int { return [3]int{r.IntN(101), r.IntN(101), r.IntN(101)} }
			w, _ := wage(side, o(), o())
			var noise [10]int
			for i := range noise {
				noise[i] = 97 + r.IntN(7)
			}
			got := w.ratios(noise)
			for row := 5; row < 12; row++ {
				if got[0][row] < got[1][row] {
					t.Fatalf("as %d row %d favours you: %v", side, row, got)
				}
			}
			if got[0][11] < 75 {
				t.Fatalf("as %d your dead %d million", side, got[0][11])
			}
			if f := w.start[you]; got[0][0] != min(100*w.t[you].BombersLost/f.Ground, 100) || got[0][1] != min(100*w.t[you].ICBMLost/f.ICBM, 100) {
				t.Fatalf("as %d the BOMBERS and ICBM'S rows are the exchange's losses in percent: %v, %+v", side, got, w.t)
			}
		}
	}
	// The military rows take the worse of city and silo damage. As the United States, firing
	// every ICBM at strike 2 puts 675 warheads on WOPR's silos and 175 on its cities, so its
	// silos set them.
	w, _ := wage(assets.US, [3]int{100, 0, 0}, [3]int{})
	if tl := w.t[wopr]; tl.SiloHits != 675 || tl.Cities != 175 {
		t.Fatalf("WOPR's tally %+v", tl)
	}
	noise := [10]int{100, 100, 100, 100, 100, 100, 100, 100, 100, 100}
	if r := w.ratios(noise); [3]int(r[1][2:5]) != [3]int{25, 59, 43} {
		t.Errorf("WOPR's ATTACK SUBS, TACTICAL AIRCRAFT and GROUND FORCES: %v, want silo damage's [25 59 43]", r[1][2:5])
	}
}

// Each system has one effect a player can learn: ICBM orders alone decide what lands on
// your cities (holding them soaks up WOPR's ICBMs; firing them turns WOPR's on your
// cities), and your SLBMs, your bombers and the ICBMs WOPR's silos could not absorb decide
// what lands on theirs.
func TestChoicesMatter(t *testing.T) {
	t.Parallel()
	for _, side := range sides {
		r := proto.NewRand(5, 6)
		for range 2000 {
			i2, i3 := r.IntN(101), r.IntN(101)
			a, _ := wage(side, [3]int{i2, r.IntN(101), r.IntN(101)}, [3]int{i3, r.IntN(101), r.IntN(101)})
			b, rs := wage(side, [3]int{i2, r.IntN(101), r.IntN(101)}, [3]int{i3, r.IntN(101), r.IntN(101)})
			if a.t[you].Cities != b.t[you].Cities {
				t.Fatalf("as %d your cities moved with SLBM or bomber orders: %d, %d", side, a.t[you].Cities, b.t[you].Cities)
			}
			theirs := 0
			for _, rp := range rs {
				theirs += rp.fire[you].SLBM + rp.fire[you].ICBMCity
			}
			theirs += rs[len(rs)-1].fire[you].Bombers - rs[len(rs)-1].fire[you].Bombers/shootDown
			if b.t[wopr].Cities != theirs {
				t.Fatalf("as %d their cities %d, want your SLBMs, surplus ICBMs and arriving bombers %d", side, b.t[wopr].Cities, theirs)
			}
		}
	}
	hold, _ := wage(assets.US, [3]int{}, [3]int{})
	fire, _ := wage(assets.US, [3]int{100, 0, 0}, [3]int{})
	if hold.t[you].Cities != 1366 || fire.t[you].Cities != 1569 || hold.t[you].ICBMLost != 675 {
		t.Errorf("hold: %+v; fire your ICBMs at strike 2: %+v", hold.t[you], fire.t[you])
	}
}
