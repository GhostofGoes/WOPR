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
	reserve   = 300 // SLBMs WOPR keeps at sea until DEFCON 1
	kPerMil   = 400 // damage = 1000*hits/(hits+kPerMil), in per mille
	shootDown = 4   // one arriving bomber in shootDown is shot down
	groundCut = 3   // enemy missiles destroy one grounded bomber in groundCut, rounded up
)

// The forces each side starts with: the machine always has a quarter more.
var startForces = [2]forces{
	you:  {ICBM: 1000, SLBM: 600, Ground: 300},
	wopr: {ICBM: 1250, SLBM: 750, Ground: 375},
}

// warPlan is the Enter default for each strike, in percent of ICBM, SLBM and bombers.
var warPlan = [strikes][3]int{{25, 25, 0}, {50, 50, 100}, {100, 100, 100}}

// WOPR's ladder floors: the least it fires in each strike while it has the stock.
var (
	ladderICBM = [strikes]int{125, 250, 375}
	ladderSLBM = [strikes]int{50, 100, 150}
)

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
	f [2]forces
	t [2]tally
}

func newWar() *war { return &war{f: startForces} }

func ceilDiv(a, b int) int { return (a + b - 1) / b }

// quarterMore is n and a quarter, rounded up: WOPR's answer to n.
func quarterMore(n int) int { return ceilDiv(5*n, 4) }

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

	// 1. Your launch. ICBMs go at WOPR's silos, as many as its silos then hold could
	// absorb; SLBMs at its cities; bombers take off.
	pI, pS, pB := fired(p.ICBM, order[sysICBM]), fired(p.SLBM, order[sysSLBM]), fired(p.Ground, order[sysBombers])
	r.fire[you].ICBMSilo = min(pI, quarterMore(e.ICBM))
	r.fire[you].ICBMCity = pI - r.fire[you].ICBMSilo
	r.fire[you].SLBM, r.fire[you].Bombers = pS, pB
	p.ICBM, p.SLBM, p.Ground, p.Air = p.ICBM-pI, p.SLBM-pS, p.Ground-pB, p.Air+pB
	r.snap[0] = w.f

	// 2. WOPR launches on warning: a quarter more than you fired, never under its ladder,
	// and never into its SLBM reserve. Its bombers all take off at the first strike.
	wI := min(e.ICBM, max(ladderICBM[k-1], quarterMore(pI)))
	wS := min(max(e.SLBM-reserve, 0), max(ladderSLBM[k-1], quarterMore(pS)))
	r.fire[wopr].ICBMSilo = min(wI, quarterMore(p.ICBM))
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
// attack lose a third.
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
			lost := ceilDiv(w.f[s].Ground, groundCut)
			w.f[s].Ground -= lost
			w.t[s].BombersLost += lost
			r.ground[s] += lost
		}
	}
}

// damage is a side's damage in per mille: rising with hits, never reaching 1000.
func damage(hits int) int { return 1000 * hits / (hits + kPerMil) }

// ratioScale are the full-scale values of kill-ratio rows 2-11; civilian rows are weights
// in percent.
var ratioScale = [10]int{100, 2400, 180, 95, 100, 90, 80, 100, 625, 1750}

// ratios derives the kill-ratio table: rows 0-1 are exact counts; the others scale with
// damage, times one noise factor per row (97..103 percent) that both sides share, so a
// side that took more hits on its cities never shows less civilian or human loss.
func (w *war) ratios(noise [10]int) [2][12]int {
	var out [2][12]int
	for s := range 2 {
		t := w.t[s]
		c, m := damage(t.Cities), damage(t.SiloHits)
		out[s][0], out[s][1] = t.BombersLost, t.ICBMLost
		for i, scale := range ratioScale {
			x := c
			if i < 3 { // military units: the worse of city and silo damage
				x = max(c, m)
			}
			v := scale * x * noise[i] / 100000
			if i >= 3 && i < 8 {
				v = min(v, 99)
			}
			out[s][2+i] = v
		}
	}
	return out
}
