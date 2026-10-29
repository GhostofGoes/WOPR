// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package gtw

// The exchange model (docs/PLAN.md §6.2): three strikes and WOPR's DEFCON 1 attack, in
// integer arithmetic. It knows nothing of the board or the protocol; gtw.go plays a
// report out frame by frame. The force levels and rules are original game design.

// Sides, as indexes into the model's arrays.
const (
	you  = 0 // the player
	wopr = 1
)

// Systems, as indexes into an order.
const (
	sysICBM = iota
	sysSLBM
	sysBombers
)

const (
	strikes   = 3   // ordered strikes; WOPR's DEFCON 1 attack follows
	kPerMil   = 400 // damage = 1000*hits/(hits+kPerMil), in per mille
	shootDown = 4   // one arriving bomber in shootDown is shot down
	groundCut = 3   // enemy missiles destroy one grounded bomber in groundCut, rounded up (land)
)

// doctrine is a war's rules: the forces each side starts with, the war plan, how hard WOPR
// answers, and WOPR's first strike when it strikes first. The film's war is filmDoctrine;
// each scenario (scenarios.go) has its own.
type doctrine struct {
	forces [2]forces       // each side's launchers at the start
	plan   [strikes][3]int // the Enter default for each strike, in percent of ICBM, SLBM and bombers
	// ladder is WOPR's floor in each strike, the least it fires while it has the stock: this
	// percentage of its starting ICBMs, and two thirds of it of its starting SLBMs.
	ladder [strikes]int
	answer int    // WOPR answers each of your launches with this percentage of it, rounded up
	first  [3]int // WOPR's first strike (ICBMs, SLBMs, bombers), fired before yours; zero when you strike first
}

// filmDoctrine is the film's war. You hold ICBM 1000, SLBM 600 and bombers 300, and the
// machine a quarter more; WOPR answers with a quarter more, and you strike first.
var filmDoctrine = doctrine{
	forces: withQuarterMore(forces{ICBM: 1000, SLBM: 600, Ground: 300}),
	plan:   [strikes][3]int{{25, 25, 0}, {50, 50, 100}, {100, 100, 100}},
	ladder: [strikes]int{10, 20, 30},
	answer: 125,
}

// withQuarterMore is your forces f and WOPR's, a quarter more of each.
func withQuarterMore(f forces) [2]forces {
	return [2]forces{you: f, wopr: {ICBM: quarterMore(f.ICBM), SLBM: quarterMore(f.SLBM), Ground: quarterMore(f.Ground)}}
}

// reserve is the SLBMs WOPR keeps at sea until DEFCON 1: two in five of its own.
func (d doctrine) reserve() int { return 2 * d.forces[wopr].SLBM / 5 }

// floors are WOPR's ladder floors in strike k (1..3): ICBMs, then SLBMs.
func (d doctrine) floors(k int) (icbm, slbm int) {
	pct := d.ladder[k-1]
	return d.forces[wopr].ICBM * pct / 100, d.forces[wopr].SLBM * pct * 2 / 300
}

// woprFirst reports whether WOPR strikes first.
func (d doctrine) woprFirst() bool { return d.first != [3]int{} }

// forces are launchers: ICBMs in silos, SLBMs at sea, bombers on the ground and airborne.
type forces struct{ ICBM, SLBM, Ground, Air int }

// tally is what a side has suffered so far.
type tally struct {
	Cities      int // warheads on its cities
	SiloHits    int // enemy ICBMs that landed on its silo field
	ICBMLost    int // ICBMs destroyed in their silos
	BombersLost int // bombers destroyed on the ground or shot down
}

// salvo is what one side fires in a stage.
type salvo struct {
	ICBMSilo, ICBMCity, SLBM int
	Bombers                  int // taking off (strikes) or arriving (DEFCON 1)
}

func (s salvo) missiles() int { return s.ICBMSilo + s.ICBMCity + s.SLBM }

// report is one stage, resolved up front; the board only reveals it.
type report struct {
	stage  int          // 1..3 for strikes, 4 for DEFCON 1
	fire   [2]salvo     // what each side fired
	ground [2]int       // ICBMs destroyed in silos plus bombers destroyed on the ground
	cities [2]int       // warheads on each side's cities
	snap   [3][2]forces // after the first launch (yours, or WOPR's first strike), after the second, after impact
}

// war is the whole exchange.
type war struct {
	d doctrine
	f [2]forces
	t [2]tally
}

func newWar(d doctrine) *war { return &war{d: d, f: d.forces} }

func ceilDiv(a, b int) int { return (a + b - 1) / b }

// quarterMore is n and a quarter, rounded up: as many ICBMs as it takes to empty n silos.
func quarterMore(n int) int { return ceilDiv(5*n, 4) }

// launchable reports whether the player still holds anything to order.
func (w *war) launchable() bool {
	f := w.f[you]
	return f.ICBM+f.SLBM+f.Ground > 0
}

// fired is what pct percent of held comes to: floored, and all of it at 100.
func fired(held, pct int) int { return held * pct / 100 }

// strike resolves strike k (1..3) for the player's order, in percent. Every launch leaves
// before anything lands: both sides launch on warning.
func (w *war) strike(k int, order [3]int) report {
	r := report{stage: k}
	if k == 1 && w.d.woprFirst() {
		// WOPR's first strike leaves first, at your silos as they stand, and yours answers.
		w.woprLaunch(&r, w.d.first[sysICBM], w.d.first[sysSLBM], w.d.first[sysBombers])
		r.snap[0] = w.f
		w.yourLaunch(&r, order)
		r.snap[1] = w.f
	} else {
		// WOPR launches on warning: its answer to what you fired, never under its ladder.
		// Its bombers take off at once.
		w.yourLaunch(&r, order)
		r.snap[0] = w.f
		fI, fS := w.d.floors(k)
		y := r.fire[you]
		w.woprLaunch(&r, max(fI, w.answerTo(y.ICBMSilo+y.ICBMCity)), max(fS, w.answerTo(y.SLBM)), w.f[wopr].Ground)
		r.snap[1] = w.f
	}
	w.land(&r)
	r.snap[2] = w.f
	return r
}

// answerTo is WOPR's answer to n of your missiles, rounded up.
func (w *war) answerTo(n int) int { return ceilDiv(w.d.answer*n, 100) }

// yourLaunch fires the order, in percent of what you hold. ICBMs go at WOPR's silos, as many
// as its silos then hold could absorb; SLBMs at its cities; bombers take off.
func (w *war) yourLaunch(r *report, order [3]int) {
	p, e := &w.f[you], &w.f[wopr]
	pI, pS, pB := fired(p.ICBM, order[sysICBM]), fired(p.SLBM, order[sysSLBM]), fired(p.Ground, order[sysBombers])
	r.fire[you].ICBMSilo = min(pI, quarterMore(e.ICBM))
	r.fire[you].ICBMCity = pI - r.fire[you].ICBMSilo
	r.fire[you].SLBM, r.fire[you].Bombers = pS, pB
	p.ICBM, p.SLBM, p.Ground, p.Air = p.ICBM-pI, p.SLBM-pS, p.Ground-pB, p.Air+pB
}

// woprLaunch fires up to icbm ICBMs, slbm SLBMs (never into its SLBM reserve) and bombers
// bombers: its ICBMs at your silos, as many as they then hold could absorb, and the rest at
// cities.
func (w *war) woprLaunch(r *report, icbm, slbm, bombers int) {
	p, e := &w.f[you], &w.f[wopr]
	wI := min(e.ICBM, icbm)
	wS := min(max(e.SLBM-w.d.reserve(), 0), slbm)
	wB := min(e.Ground, bombers)
	r.fire[wopr].ICBMSilo = min(wI, quarterMore(p.ICBM))
	r.fire[wopr].ICBMCity = wI - r.fire[wopr].ICBMSilo
	r.fire[wopr].SLBM, r.fire[wopr].Bombers = wS, wB
	e.ICBM, e.SLBM, e.Ground, e.Air = e.ICBM-wI, e.SLBM-wS, e.Ground-wB, e.Air+wB
}

// final resolves DEFCON 1: WOPR fires everything it has left, every airborne bomber
// arrives, and every bomber still on the ground is destroyed. Your unfired missiles stay
// where they are; nobody orders them.
func (w *war) final() report {
	r := report{stage: strikes + 1}
	p, e := &w.f[you], &w.f[wopr]
	r.snap[0] = w.f
	r.fire[wopr].ICBMSilo = min(e.ICBM, quarterMore(p.ICBM))
	r.fire[wopr].ICBMCity = e.ICBM - r.fire[wopr].ICBMSilo
	r.fire[wopr].SLBM = e.SLBM
	e.ICBM, e.SLBM = 0, 0
	r.fire[you].Bombers, r.fire[wopr].Bombers = p.Air, e.Air
	r.snap[1] = w.f
	w.land(&r)
	for s := range 2 {
		shot := w.f[s].Air / shootDown
		w.t[s].BombersLost += shot
		r.cities[1-s] += w.f[s].Air - shot
		w.t[1-s].Cities += w.f[s].Air - shot
		w.f[s].Air = 0
		r.ground[s] += w.f[s].Ground
		w.t[s].BombersLost += w.f[s].Ground
		w.f[s].Ground = 0
	}
	r.snap[2] = w.f
	return r
}

// land applies the missiles of r: ICBMs on silo fields destroy four in five of the
// ICBMs held there, every warhead aimed at cities lands, and grounded bombers under
// attack lose a third, but never more than the missiles that land: one stray SLBM destroys
// one bomber, not a hundred.
func (w *war) land(r *report) {
	for s := range 2 {
		in := r.fire[1-s] // what landed on side s
		kill := min(w.f[s].ICBM, 4*in.ICBMSilo/5)
		w.f[s].ICBM -= kill
		w.t[s].ICBMLost += kill
		w.t[s].SiloHits += in.ICBMSilo
		r.ground[s] += kill
		hits := in.ICBMCity + in.SLBM
		w.t[s].Cities += hits
		r.cities[s] += hits
		if in.missiles() > 0 && r.stage <= strikes {
			lost := min(ceilDiv(w.f[s].Ground, groundCut), in.missiles())
			w.f[s].Ground -= lost
			w.t[s].BombersLost += lost
			r.ground[s] += lost
		}
	}
}

// damage is a side's damage in per mille: rising with hits, never reaching 1000.
func damage(hits int) int { return 1000 * hits / (hits + kPerMil) }

// ratioScale weights kill-ratio rows 2-9 as a percentage of full damage, and rows 10-11 as
// the millions at full damage.
var ratioScale = [10]int{40, 95, 70, 95, 100, 90, 80, 100, 65, 110}

// ratios derives the kill-ratio table, in the film's rows: five military systems and five
// civilian assets destroyed, in percent, then the injured and the dead in millions. The
// BOMBERS and ICBM'S rows are the exchange's own losses as a percentage of the force; the
// others scale with damage, times one noise factor per row (97..103 percent) that both sides
// share, so a side that took more hits on its cities never shows less civilian or human loss.
func (w *war) ratios(noise [10]int) [2][12]int {
	var out [2][12]int
	for s := range 2 {
		t, f := w.t[s], w.d.forces[s]
		c, m := damage(t.Cities), damage(t.SiloHits)
		out[s][0] = min(100*t.BombersLost/f.Ground, 100)
		out[s][1] = min(100*t.ICBMLost/f.ICBM, 100)
		for i, scale := range ratioScale {
			x := c
			if i < 3 { // military units: the worse of city and silo damage
				x = max(c, m)
			}
			v := scale * x * noise[i] / 100000
			if i < 8 {
				v = min(v, 99)
			}
			out[s][2+i] = v
		}
	}
	return out
}
