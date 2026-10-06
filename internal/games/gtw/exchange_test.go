package gtw

import (
	"testing"

	"github.com/GhostofGoes/WOPR/internal/proto"
)

// wage fights a whole war: the first strike on the war plan, then your two orders.
func wage(o2, o3 [3]int) (*war, []report) {
	w := newWar()
	rs := []report{w.strike(1, warPlan[0]), w.strike(2, o2), w.strike(3, o3)}
	return w, append(rs, w.final())
}

// check holds the guarantees for one war: nothing you order wins, and nothing is created
// or lost from the books.
func check(t *testing.T, w *war, rs []report, label string) {
	t.Helper()
	if you, them := w.t[you].Cities, w.t[wopr].Cities; you < 1144 || them > 750 {
		t.Fatalf("%s: warheads on cities: yours %d (want >= 1144), WOPR's side %d (want <= 750)", label, you, them)
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
		f, st, tl := w.f[s], startForces[s], w.t[s]
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

// The quality test: no allocation wins. ICBM orders run at every percent (they alone move
// what lands on your cities); SLBMs and bombers at the extremes and the middle; then
// random orders at every percent.
func TestNoAllocationWins(t *testing.T) {
	t.Parallel()
	ends := []int{0, 50, 100}
	for i2 := 0; i2 <= 100; i2++ {
		for i3 := 0; i3 <= 100; i3++ {
			for _, s2 := range ends {
				for _, s3 := range ends {
					for _, b := range []int{0, 100} {
						w, rs := wage([3]int{i2, s2, b}, [3]int{i3, s3, 100 - b})
						check(t, w, rs, "grid")
					}
				}
			}
		}
	}
	r := proto.NewRand(1, 2)
	for range 2000 {
		o := func() [3]int { return [3]int{r.IntN(101), r.IntN(101), r.IntN(101)} }
		w, rs := wage(o(), o())
		check(t, w, rs, "random")
	}
}

// The doctrine's numbers on the film path (Enter at every prompt) and on HOLD, HOLD.
func TestDoctrine(t *testing.T) {
	t.Parallel()
	w, rs := wage(warPlan[1], warPlan[2])
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

// Whatever the orders and the noise, your civilian and human rows are never better than
// the other side's, and your dead are never under 120 million.
func TestRatiosFavourNoOne(t *testing.T) {
	t.Parallel()
	r := proto.NewRand(3, 4)
	for range 2000 {
		o := func() [3]int { return [3]int{r.IntN(101), r.IntN(101), r.IntN(101)} }
		w, _ := wage(o(), o())
		var noise [10]int
		for i := range noise {
			noise[i] = 97 + r.IntN(7)
		}
		got := w.ratios(noise)
		for row := 5; row < 12; row++ {
			if got[0][row] < got[1][row] {
				t.Fatalf("row %d favours you: %v", row, got)
			}
		}
		if got[0][11] < 1200 {
			t.Fatalf("your dead %d tenths of a million", got[0][11])
		}
		if got[0][0] != w.t[you].BombersLost || got[0][1] != w.t[you].ICBMLost {
			t.Fatalf("the BOMBERS and ICBM rows are the exchange's counts: %v, %+v", got, w.t)
		}
	}
	// The military rows take the worse of city and silo damage. Firing every ICBM at strike 2
	// puts 750 warheads on WOPR's silos and 150 on its cities, so its silos set them.
	w, _ := wage([3]int{100, 0, 0}, [3]int{})
	if tl := w.t[wopr]; tl.SiloHits != 750 || tl.Cities != 150 {
		t.Fatalf("WOPR's tally %+v", tl)
	}
	noise := [10]int{100, 100, 100, 100, 100, 100, 100, 100, 100, 100}
	if r := w.ratios(noise); [3]int(r[1][2:5]) != [3]int{65, 1564, 117} {
		t.Errorf("WOPR's ATTACK SUBS, TACTICAL AIRCRAFT and GROUND FORCES: %v, want silo damage's [65 1564 117]", r[1][2:5])
	}
}

// Each system has one effect a player can learn: ICBM orders alone decide what lands on
// your cities (holding them soaks up WOPR's ICBMs; firing them turns WOPR's on your
// cities), and your SLBMs and bombers alone decide what lands on theirs.
func TestChoicesMatter(t *testing.T) {
	t.Parallel()
	r := proto.NewRand(5, 6)
	for range 2000 {
		i2, i3 := r.IntN(101), r.IntN(101)
		a, _ := wage([3]int{i2, r.IntN(101), r.IntN(101)}, [3]int{i3, r.IntN(101), r.IntN(101)})
		b, rs := wage([3]int{i2, r.IntN(101), r.IntN(101)}, [3]int{i3, r.IntN(101), r.IntN(101)})
		if a.t[you].Cities != b.t[you].Cities {
			t.Fatalf("your cities moved with SLBM or bomber orders: %d, %d", a.t[you].Cities, b.t[you].Cities)
		}
		theirs := 0
		for _, rp := range rs {
			theirs += rp.fire[you].SLBM
		}
		theirs += rs[len(rs)-1].fire[you].Bombers - rs[len(rs)-1].fire[you].Bombers/shootDown
		if b.t[wopr].Cities != theirs {
			t.Fatalf("their cities %d, want your SLBMs and arriving bombers %d", b.t[wopr].Cities, theirs)
		}
	}
	hold, _ := wage([3]int{}, [3]int{})
	fire, _ := wage([3]int{100, 0, 0}, [3]int{})
	if hold.t[you].Cities != 1144 || fire.t[you].Cities != 1657 || hold.t[you].ICBMLost != 750 {
		t.Errorf("hold: %+v; fire your ICBMs at strike 2: %+v", hold.t[you], fire.t[you])
	}
}
