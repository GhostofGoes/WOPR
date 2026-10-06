// Package sim is the engine the war-game scenarios share (docs/PLAN.md §6.3.1): a strip of
// regions, units with two steps, orders, one combat-results table, an event deck, a turn
// limit, a victory hook and kill ratios in GTW's table format. A scenario is data plus a
// few hooks; NewGame turns it into a proto.Program.
package sim

import (
	"math/rand/v2"
	"slices"
)

// Sides.
const (
	Player = 0
	WOPR   = 1
	nobody = -1
)

// Terrain shifts the odds in the defender's favour by its value in columns.
type Terrain uint8

// Terrains.
const (
	Open Terrain = iota
	Rough
	City
)

// Region is one place on the strip. Regions are numbered from the player's rear.
type Region struct {
	Name    string
	Terrain Terrain
}

// UnitType is a row of a scenario's unit table.
type UnitType struct {
	Name     string // short, as the map shows it: ARM, INF, ART
	Long     string // as prompts say it: ARMOUR
	Attack   int
	Defence  int
	Move     int    // regions a turn
	Range    int    // how far it can attack: 1 next door, 2 one region further
	Category string // its kill-ratio row
}

// Unit is one unit on the map.
type Unit struct {
	ID     int // numbered within its side and type, as the player names it: ARM1
	Side   int
	Type   *UnitType
	Region int // index into State.Regions
	Steps  int // 2 full, 1 reduced, 0 destroyed
	Hidden bool
}

// Name is the unit as the map shows it: ARM1, or ARM1* when reduced.
func (u *Unit) Name() string {
	n := u.Type.Name + string(rune('0'+u.ID))
	if u.Steps == 1 {
		n += "*"
	}
	return n
}

// Alive reports whether the unit has steps left.
func (u *Unit) Alive() bool { return u.Steps > 0 }

func half(v int) int { return (v + 1) / 2 }

// State is a scenario in play. Hooks read and change it.
type State struct {
	Scenario *Scenario
	Regions  []Region
	Control  []int // the side holding each region, or -1
	Units    []*Unit
	Turn     int // 1-based
	Vars     map[string]int
	Losses   [2]map[string]int // steps lost, by kill-ratio category
	rng      *rand.Rand
	said     []string
	deck     []int
}

// Say queues lines for this turn's report.
func (s *State) Say(lines ...string) { s.said = append(s.said, lines...) }

// Rand is the scenario's random stream.
func (s *State) Rand() *rand.Rand { return s.rng }

// Living lists a side's units with steps left.
func (s *State) Living(side int) []*Unit {
	var out []*Unit
	for _, u := range s.Units {
		if u.Side == side && u.Alive() {
			out = append(out, u)
		}
	}
	return out
}

// In lists a side's living units in a region.
func (s *State) In(region, side int) []*Unit {
	var out []*Unit
	for _, u := range s.Units {
		if u.Side == side && u.Alive() && u.Region == region {
			out = append(out, u)
		}
	}
	return out
}

// Lose takes steps from a unit and counts them in the kill ratios.
func (s *State) Lose(u *Unit, steps int) {
	steps = min(steps, u.Steps)
	u.Steps -= steps
	s.Losses[u.Side][u.Type.Category] += steps
	if !u.Alive() {
		s.Say(s.Scenario.text(TextDestroyed, s.owner(u)+u.Type.Name+string(rune('0'+u.ID))))
	}
}

// AttackOf is a unit's attack after its losses and the scenario's modifiers.
func (s *State) AttackOf(u *Unit) int {
	a := u.Type.Attack
	if u.Steps == 1 {
		a = half(a)
	}
	if s.Scenario.AttackMod != nil {
		a = s.Scenario.AttackMod(s, u, a)
	}
	return max(a, 1)
}

// DefenceOf is a unit's defence after its losses.
func (s *State) DefenceOf(u *Unit) int {
	d := u.Type.Defence
	if u.Steps == 1 {
		d = half(d)
	}
	return max(d, 1)
}

// rear is the direction of a side's own rear on the strip.
func rear(side int) int {
	if side == Player {
		return -1
	}
	return 1
}

// Enemy is the other side.
func Enemy(side int) int { return 1 - side }

// updateControl gives each region to the side with units in it alone.
func (s *State) updateControl() {
	for r := range s.Regions {
		p, w := len(s.In(r, Player)) > 0, len(s.In(r, WOPR)) > 0
		switch {
		case p && !w:
			s.Control[r] = Player
		case w && !p:
			s.Control[r] = WOPR
		}
	}
}

// CanMove reports whether u can move to region to: within its move, and no enemy unit in
// the way or at the end.
func (s *State) CanMove(u *Unit, to int) bool {
	if to < 0 || to >= len(s.Regions) || to == u.Region || abs(to-u.Region) > u.Type.Move {
		return false
	}
	step := 1
	if to < u.Region {
		step = -1
	}
	for r := u.Region + step; ; r += step {
		if len(s.In(r, Enemy(u.Side))) > 0 {
			return false
		}
		if r == to {
			return true
		}
	}
}

// CanAttack reports whether u can attack region at: an enemy there, within range, and (for
// a range beyond one) nothing friendly needed in between.
func (s *State) CanAttack(u *Unit, at int) bool {
	if at < 0 || at >= len(s.Regions) || at == u.Region || abs(at-u.Region) > u.Type.Range {
		return false
	}
	for _, e := range s.In(at, Enemy(u.Side)) {
		if !e.Hidden {
			return true
		}
	}
	return false
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Attack resolves u's attack on region at against its strongest visible defender. A clear
// result lets a unit next door advance into the region.
func (s *State) Attack(u *Unit, at int) {
	var def *Unit
	for _, e := range s.In(at, Enemy(u.Side)) {
		if !e.Hidden && (def == nil || s.DefenceOf(e) > s.DefenceOf(def)) {
			def = e
		}
	}
	if def == nil || !u.Alive() {
		return
	}
	res := CRT(s.AttackOf(u), s.DefenceOf(def), s.Regions[at].Terrain, s.rng.IntN(6)+1)
	s.Say(s.Scenario.text(TextAttack, s.owner(u)+u.Name(), s.owner(def)+def.Name(), s.Regions[at].Name, res.String()))
	switch res {
	case AttackerLoses:
		s.Lose(u, 1)
	case Exchange:
		s.Lose(u, 1)
		s.Lose(def, 1)
	case DefenderRetreats:
		back := at + rear(def.Side)
		if back < 0 || back >= len(s.Regions) || len(s.In(back, u.Side)) > 0 {
			s.Lose(def, 1) // nowhere to go
		} else {
			def.Region = back
		}
	case DefenderLoses:
		s.Lose(def, 1)
	}
	if len(s.In(at, Enemy(u.Side))) == 0 && u.Alive() && abs(at-u.Region) == 1 {
		u.Region = at // the way is clear: advance
	}
}

// owner prefixes WOPR's units in reports, so YOUR ARM1 and WOPR'S ARM1 are told apart.
func (s *State) owner(u *Unit) string {
	if u.Side == WOPR {
		return s.Scenario.text(TextWOPRs)
	}
	return ""
}

// drawEvent plays the next event card, reshuffling the deck from the stream when empty.
func (s *State) drawEvent() {
	ev := s.Scenario.Events
	if len(ev) == 0 {
		return
	}
	if len(s.deck) == 0 {
		s.deck = make([]int, len(ev))
		for i := range s.deck {
			s.deck[i] = i
		}
		s.rng.Shuffle(len(s.deck), func(i, j int) { s.deck[i], s.deck[j] = s.deck[j], s.deck[i] })
	}
	e := ev[s.deck[0]]
	s.deck = s.deck[1:]
	s.Say(e.Text)
	if e.Apply != nil {
		e.Apply(s)
	}
}

// Nearest is the closest region (to from) that test accepts, or -1.
func Nearest(regions int, from int, test func(int) bool) int {
	for d := 0; d < regions; d++ {
		for _, r := range []int{from - d, from + d} {
			if r >= 0 && r < regions && test(r) {
				return r
			}
		}
	}
	return -1
}

// Toward is the farthest region u can move to on the way to target, or -1.
func (s *State) Toward(u *Unit, target int) int {
	step := 1
	if target < u.Region {
		step = -1
	}
	best := -1
	for r := u.Region + step; abs(r-u.Region) <= u.Type.Move && r >= 0 && r < len(s.Regions); r += step {
		if !s.CanMove(u, r) {
			break
		}
		best = r
		if r == target {
			break
		}
	}
	return best
}

// sortUnits orders units by type then number, for stable prompts and maps.
func sortUnits(us []*Unit) {
	slices.SortStableFunc(us, func(a, b *Unit) int {
		if a.Type.Name != b.Type.Name {
			if a.Type.Name < b.Type.Name {
				return -1
			}
			return 1
		}
		return a.ID - b.ID
	})
}
