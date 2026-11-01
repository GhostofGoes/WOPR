// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package gtw

// The exchange model (docs/PLAN.md §6.2): three strikes and WOPR's DEFCON 1 attack, in
// integer arithmetic, fought to one of WOPR's doctrines. It knows nothing of the board or
// the protocol; gtw.go plays a report out frame by frame. The force levels, rules and
// doctrines are original game design.

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
)

// The forces each side starts with: the machine always has a quarter more.
var startForces = [2]forces{
	you:  {ICBM: 1000, SLBM: 600, Ground: 300},
	wopr: {ICBM: 1250, SLBM: 750, Ground: 375},
}

// warPlan is the Enter default for each strike, in percent of ICBM, SLBM and bombers.
var warPlan = [strikes][3]int{{25, 25, 0}, {50, 50, 100}, {100, 100, 100}}

// doctrineID names one of WOPR's doctrines. Every war is a measured response, the film's,
// unless the session's seed is one of the easter eggs (doctrine.go).
type doctrineID uint8

// The doctrines.
const (
	measured     doctrineID = iota // a quarter more than you fire, on a rising ladder
	hairTrigger                    // twice what you fire, most of it at once
	counterforce                   // your silos and grounded bombers first, its subs kept for DEFCON 1
	secondStrike                   // rides out your strikes, then empties everything on your cities
)

// doctrine is how WOPR fights: how hard it answers your launches, the least it fires each
// strike, what it keeps back for DEFCON 1, and where its ICBMs and missiles fall.
type doctrine struct {
	answer     [2]int       // WOPR answers n launched with n*answer[0]/answer[1], rounded up
	ladderICBM [strikes]int // the least it fires in each strike while it has the stock
	ladderSLBM [strikes]int
	keepICBM   int    // ICBMs it keeps in silos until DEFCON 1
	keepSLBM   int    // SLBMs it keeps at sea until DEFCON 1
	siloAim    [2]int // its ICBMs go at your silos, up to what they hold times siloAim[0]/siloAim[1]
	groundCut  int    // missiles landing destroy one grounded bomber in groundCut, rounded up
	heldFewest bool   // holding fire at both prompts leaves the fewest dead on both sides (exchange_test.go)
}

// doctrines are WOPR's ways of fighting. Each leaves the war unwinnable (exchange_test.go):
// every missile it holds is fired by DEFCON 1, and its bombers fly at the first strike.
var doctrines = [...]doctrine{
	measured: {
		answer: [2]int{5, 4}, ladderICBM: [strikes]int{125, 250, 375}, ladderSLBM: [strikes]int{50, 100, 150},
		keepSLBM: 300, siloAim: [2]int{5, 4}, groundCut: 3, heldFewest: true,
	},
	hairTrigger: {
		answer: [2]int{2, 1}, ladderICBM: [strikes]int{500, 375, 250}, ladderSLBM: [strikes]int{250, 200, 150},
		siloAim: [2]int{1, 1}, groundCut: 3, heldFewest: true,
	},
	counterforce: {
		answer: [2]int{5, 4}, ladderICBM: [strikes]int{500, 375, 250}, ladderSLBM: [strikes]int{25, 25, 25},
		keepSLBM: 600, siloAim: [2]int{2, 1}, groundCut: 2,
	},
	secondStrike: {
		answer: [2]int{1, 2}, ladderICBM: [strikes]int{50, 50, 50}, ladderSLBM: [strikes]int{25, 25, 25},
		keepICBM: 900, keepSLBM: 600, siloAim: [2]int{0, 1}, groundCut: 3,
	},
}

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
	snap   [3][2]forces // after your launch, after WOPR's launch, after impact
}

// war is the whole exchange.
type war struct {
	id doctrineID // WOPR's doctrine
	d  doctrine   // and its figures
	f  [2]forces
	t  [2]tally
}

func newWar(d doctrineID) *war { return &war{id: d, d: doctrines[d], f: startForces} }

// fight wages a whole war against doctrine d: the first strike on the war plan, then o2 and
// o3, then WOPR's DEFCON 1 attack. The exchange has no dice, so the same doctrine and orders
// always fight the same war, and the after-action report can fight it again with other orders.
func fight(d doctrineID, o2, o3 [3]int) (*war, []report) {
	return wage(d, [strikes][3]int{warPlan[0], o2, o3})
}

// wage wages a whole war against doctrine d with an order for each strike, the first strike's
// included, then WOPR's DEFCON 1 attack: a scenario's war (scenarios.go), or fight's.
func wage(d doctrineID, orders [strikes][3]int) (*war, []report) {
	w := newWar(d)
	var rs []report
	for k, o := range orders {
		rs = append(rs, w.strike(k+1, o))
	}
	return w, append(rs, w.final())
}

func ceilDiv(a, b int) int { return (a + b - 1) / b }

// times is n times r[0]/r[1], rounded up.
func times(n int, r [2]int) int { return ceilDiv(r[0]*n, r[1]) }

// siloShots is how many of n ICBMs WOPR aims at your silo field, which holds held; the rest
// go at your cities.
func (w *war) siloShots(n, held int) int { return min(n, times(held, w.d.siloAim)) }

// launchable reports whether the player still holds anything to order.
func (w *war) launchable() bool {
	f := w.f[you]
	return f.ICBM+f.SLBM+f.Ground > 0
}

// fired is what pct percent of held comes to: floored, and all of it at 100.
func fired(held, pct int) int { return held * pct / 100 }

// strike resolves strike k (1..3) for the player's order, in percent.
func (w *war) strike(k int, order [3]int) report {
	r := report{stage: k}
	p, e := &w.f[you], &w.f[wopr]

	// 1. Your launch. ICBMs go at WOPR's silos, whatever they still hold; SLBMs at its
	// cities; bombers take off.
	pI, pS, pB := fired(p.ICBM, order[sysICBM]), fired(p.SLBM, order[sysSLBM]), fired(p.Ground, order[sysBombers])
	r.fire[you] = salvo{ICBMSilo: pI, SLBM: pS, Bombers: pB}
	p.ICBM, p.SLBM, p.Ground, p.Air = p.ICBM-pI, p.SLBM-pS, p.Ground-pB, p.Air+pB
	r.snap[0] = w.f

	// 2. WOPR launches on warning, to its doctrine: its answer to what you fired, never under
	// its ladder, and never into what it keeps for DEFCON 1. Its bombers all take off at the
	// first strike.
	d := w.d
	wI := min(max(e.ICBM-d.keepICBM, 0), max(d.ladderICBM[k-1], times(pI, d.answer)))
	wS := min(max(e.SLBM-d.keepSLBM, 0), max(d.ladderSLBM[k-1], times(pS, d.answer)))
	r.fire[wopr].ICBMSilo = w.siloShots(wI, p.ICBM)
	r.fire[wopr].ICBMCity = wI - r.fire[wopr].ICBMSilo
	r.fire[wopr].SLBM = wS
	r.fire[wopr].Bombers = e.Ground
	e.ICBM, e.SLBM, e.Air, e.Ground = e.ICBM-wI, e.SLBM-wS, e.Air+e.Ground, 0
	r.snap[1] = w.f

	// 3. Impact.
	w.land(&r)
	r.snap[2] = w.f
	return r
}

// final resolves DEFCON 1: WOPR fires everything it has left, every airborne bomber
// arrives, and every bomber still on the ground is destroyed. Your unfired missiles stay
// where they are; nobody orders them.
func (w *war) final() report {
	r := report{stage: strikes + 1}
	p, e := &w.f[you], &w.f[wopr]
	r.snap[0] = w.f
	r.fire[wopr].ICBMSilo = w.siloShots(e.ICBM, p.ICBM)
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
// attack lose a third (half, to a counterforce doctrine).
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
			lost := ceilDiv(w.f[s].Ground, w.d.groundCut) // WOPR's are airborne: only yours are on the ground
			w.f[s].Ground -= lost
			w.t[s].BombersLost += lost
			r.ground[s] += lost
		}
	}
}

// dead is each side's POPULATION DEATHS row, in millions, as the kill ratios show it.
func (w *war) dead(noise [10]int) [2]int {
	r := w.ratios(noise)
	return [2]int{r[you][11], r[wopr][11]}
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
		t, f := w.t[s], startForces[s]
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
