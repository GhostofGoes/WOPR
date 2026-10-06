package sim

import (
	"fmt"
	"slices"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/prompt"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// Game runs a scenario as a proto.Program: the player orders each unit in turn, WOPR
// orders its own, and the turn resolves.
type Game struct {
	sc     *Scenario
	s      *State
	queue  []*Unit // the player's units still to order this turn
	orders []Order
	over   bool
}

// NewGame returns a scenario ready to Start.
func NewGame(sc *Scenario) *Game { return &Game{sc: sc} }

// State exposes the game state, for scenario tests.
func (g *Game) State() *State { return g.s }

// Next is the unit the player is ordering, or nil.
func (g *Game) Next() *Unit {
	if len(g.queue) == 0 {
		return nil
	}
	return g.queue[0]
}

// AIOrders are the orders WOPR would give now, for tests that check them.
func (g *Game) AIOrders() []Order {
	if g.sc.AI != nil {
		return g.sc.AI(g.s)
	}
	return DefaultAI(g.s)
}

func say(lines ...string) proto.Output { return proto.Say{Lines: lines, Pace: proto.PaceSpeech} }

func table(lines ...string) proto.Output { return proto.Say{Lines: lines, Pace: proto.PaceTable} }

// Start implements proto.Program.
func (g *Game) Start(env proto.Env) []proto.Output {
	g.s = &State{
		Scenario: g.sc, Regions: g.sc.Regions, Control: make([]int, len(g.sc.Regions)), Turn: 1,
		Vars: map[string]int{}, Losses: [2]map[string]int{{}, {}}, rng: proto.NewRand(env.Seed, 0),
	}
	for i := range g.s.Control {
		g.s.Control[i] = nobody
	}
	g.sc.Setup(g.s)
	g.s.updateControl()
	outs := []proto.Output{say(g.sc.Intro.Texts()...), table(g.mapLines()...)}
	return append(outs, g.beginTurn()...)
}

// View implements proto.Program; the sims are console text.
func (g *Game) View(*proto.Canvas) {}

// Handle implements proto.Program.
func (g *Game) Handle(ev proto.Event) []proto.Output {
	line, ok := ev.(proto.LineEvent)
	if !ok || g.over || len(g.queue) == 0 {
		return nil
	}
	return g.onOrder(line.Text)
}

func (g *Game) beginTurn() []proto.Output {
	g.queue = g.s.Living(Player)
	sortUnits(g.queue)
	g.orders = nil
	return []proto.Output{g.ask()}
}

func (g *Game) ask() proto.Output {
	u := g.queue[0]
	return proto.Prompt{Text: g.sc.text(TextOrders, u.Name(), g.s.Regions[u.Region].Name)}
}

func (g *Game) again(text string) []proto.Output { return []proto.Output{say(text), g.ask()} }

// onOrder reads the current unit's order: a verb, and a region by number or name.
func (g *Game) onOrder(input string) []proto.Output {
	u := g.queue[0]
	words := strings.Fields(prompt.Normalize(input))
	if len(words) == 0 {
		words = []string{Hold.Name}
	}
	switch words[0] {
	case "STATUS", "MAP":
		return append([]proto.Output{table(g.mapLines()...)}, g.ask())
	case "HELP":
		var help []string
		var names []string
		for _, v := range g.sc.verbs() {
			names = append(names, v.Name)
			help = append(help, "  "+v.Help)
		}
		return append([]proto.Output{say(g.sc.text(TextHelp, strings.Join(names, ", "))), table(help...)}, g.ask())
	case "END", "DONE":
		for _, rest := range g.queue {
			g.orders = append(g.orders, Order{Unit: rest, Verb: Hold, Target: -1})
		}
		g.queue = nil
		return g.resolve()
	}
	v := g.verb(words[0])
	if v == nil {
		return g.again(g.sc.text(TextUnknown))
	}
	target := -1
	if v.Target {
		if len(words) < 2 {
			return g.again(g.sc.text(TextWhere))
		}
		target = g.region(strings.Join(words[1:], " "))
		if target < 0 {
			return g.again(g.sc.text(TextWhere))
		}
	}
	if v.Check != nil {
		if why := v.Check(g.s, u, target); why != "" {
			return g.again(why)
		}
	}
	g.orders = append(g.orders, Order{Unit: u, Verb: v, Target: target})
	g.queue = g.queue[1:]
	if len(g.queue) > 0 {
		return []proto.Output{g.ask()}
	}
	return g.resolve()
}

// verb finds a verb by name or unique prefix (M for MOVE).
func (g *Game) verb(word string) *Verb {
	var match []*Verb
	for _, v := range g.sc.verbs() {
		if v.Name == word {
			return v
		}
		if strings.HasPrefix(v.Name, word) {
			match = append(match, v)
		}
	}
	if len(match) == 1 {
		return match[0]
	}
	return nil
}

// region reads a region by its number on the map or its name (or a unique prefix).
func (g *Game) region(text string) int {
	if n, ok := prompt.Number(text); ok && n >= 1 && n <= len(g.s.Regions) {
		return n - 1
	}
	found := -1
	for i, r := range g.s.Regions {
		name := prompt.Normalize(r.Name)
		switch {
		case name == text:
			return i
		case strings.HasPrefix(name, text):
			if found >= 0 {
				return -1 // ambiguous
			}
			found = i
		}
	}
	return found
}

// resolve plays the turn: WOPR's orders, movement, combat, the rest, an event card,
// upkeep, and the victory check.
func (g *Game) resolve() []proto.Output {
	s := g.s
	ai := g.sc.AI
	if ai == nil {
		ai = DefaultAI
	}
	all := append(slices.Clone(g.orders), ai(s)...)
	for _, u := range s.Units {
		u.Attacked = false
	}
	for phase := range 3 {
		for _, o := range all {
			if o.Verb.Phase != phase || !o.Unit.Alive() || o.Verb.Apply == nil {
				continue
			}
			if o.Verb.Check != nil && o.Verb.Check(s, o.Unit, o.Target) != "" {
				continue // the other side's moves made it impossible
			}
			o.Verb.Apply(s, o.Unit, o.Target)
		}
		s.updateControl()
	}
	s.drawEvent()
	if g.sc.Upkeep != nil {
		g.sc.Upkeep(s)
	}
	s.updateControl()
	report := s.said
	s.said = nil
	outs := []proto.Output{say(report...)}

	final := s.Turn >= g.sc.TurnLimit
	outcome, why, done := g.decide(final)
	if done {
		g.over = true
		return append(outs, table(g.mapLines()...), say(why), proto.Done{Result: proto.Result{Outcome: outcome, Lines: g.ratioLines()}})
	}
	s.Turn++ // the map shows the turn about to be ordered
	return append(append(outs, table(g.mapLines()...)), g.beginTurn()...)
}

// decide asks the scenario, after the engine's own rule: a side with no units has lost.
func (g *Game) decide(final bool) (proto.Outcome, string, bool) {
	if g.sc.Victory != nil {
		if o, why, done := g.sc.Victory(g.s, final); done {
			return o, why, true
		}
	}
	switch {
	case len(g.s.Living(Player)) == 0 && len(g.s.Living(WOPR)) == 0:
		return proto.NoWinner, g.sc.text(TextEnd), true
	case len(g.s.Living(Player)) == 0:
		return proto.Loss, g.sc.text(TextEnd), true
	case len(g.s.Living(WOPR)) == 0:
		return proto.Win, g.sc.text(TextEnd), true
	case final:
		return proto.Draw, g.sc.text(TextEnd), true
	}
	return 0, "", false
}

// mapRow lays out a row of the map: number, region, ground, holder, both sides' units.
const mapRow = "%2s %-20s %-7s %-8s %-17s %s"

// mapLines is the strip as a table, with the turn and the scenario's status line.
func (g *Game) mapLines() []string {
	s := g.s
	head := g.sc.Title + "   " + g.sc.text(TextTurn, fmt.Sprint(s.Turn), fmt.Sprint(g.sc.TurnLimit))
	if g.sc.Status != nil {
		head += "   " + g.sc.Status(s)
	}
	h := engineText[TextMapHead].Texts()
	lines := []string{head, fmt.Sprintf(mapRow, "", h[0], h[1], h[2], h[3], h[4])}
	for i, r := range s.Regions {
		held := g.sc.text(TextHeld)
		switch s.Control[i] {
		case Player:
			held = g.sc.text(TextYou)
		case WOPR:
			held = g.sc.text(TextWOPR)
		}
		name := r.Name
		if g.sc.RegionNote != nil {
			if note := g.sc.RegionNote(s, i); note != "" {
				name += " " + note
			}
		}
		lines = append(lines, fmt.Sprintf(mapRow, fmt.Sprint(i+1), name, terrainName(r.Terrain), held,
			unitList(s.In(i, Player), false), unitList(s.In(i, WOPR), true)))
	}
	return lines
}

// unitList names units for the map; WOPR's hidden units are not shown.
func unitList(us []*Unit, hideHidden bool) string {
	sortUnits(us)
	var names []string
	for _, u := range us {
		if hideHidden && u.Hidden {
			continue
		}
		names = append(names, u.Name())
	}
	return strings.Join(names, " ")
}

// ratioLines is the kill-ratio table: steps each side lost, by category.
func (g *Game) ratioLines() []string { return RatioTable(g.s.Losses) }

// RatioTable is the kill-ratio table for losses by category (player's, then WOPR's), in
// the format every war game ends with; the bespoke games use it too.
func RatioTable(losses [2]map[string]int) []string {
	var cats []string
	for side := range 2 {
		for c := range losses[side] {
			if !slices.Contains(cats, c) {
				cats = append(cats, c)
			}
		}
	}
	slices.Sort(cats)
	lines := []string{engineText[TextRatios][0].Text, engineText[TextRatioHead][0].Text}
	for _, c := range cats {
		lines = append(lines, fmt.Sprintf("%-24s %6d %9d", c, losses[Player][c], losses[WOPR][c]))
	}
	return lines
}

// DefaultAI orders WOPR's units: attack at 2:1 or better (artillery at any odds), fall
// back when reduced and threatened, otherwise press toward the player's rear.
func DefaultAI(s *State) []Order {
	var orders []Order
	units := s.Living(WOPR)
	sortUnits(units)
	for _, u := range units {
		bestCol, bestAt := -1, -1
		for at := range s.Regions {
			if !s.CanAttack(u, at) {
				continue
			}
			def := strongest(s, at, Player)
			if col := Column(s.AttackOf(u), s.DefenceOf(def), s.Regions[at].Terrain); col > bestCol {
				bestCol, bestAt = col, at
			}
		}
		switch {
		case bestAt >= 0 && (bestCol >= col2to1 || u.Type.Range > 1):
			orders = append(orders, Order{Unit: u, Verb: Attack, Target: bestAt})
		case u.Steps == 1 && threatened(s, u) && s.CanMove(u, u.Region+rear(WOPR)):
			orders = append(orders, Order{Unit: u, Verb: Move, Target: u.Region + rear(WOPR)})
		default:
			if to := s.Toward(u, 0); to >= 0 {
				orders = append(orders, Order{Unit: u, Verb: Move, Target: to})
			} else {
				orders = append(orders, Order{Unit: u, Verb: Hold, Target: -1})
			}
		}
	}
	return orders
}

func strongest(s *State, at, side int) *Unit {
	var best *Unit
	for _, e := range s.In(at, side) {
		if !e.Hidden && (best == nil || s.DefenceOf(e) > s.DefenceOf(best)) {
			best = e
		}
	}
	return best
}

// threatened reports a visible enemy next door.
func threatened(s *State, u *Unit) bool {
	for _, r := range []int{u.Region - 1, u.Region + 1} {
		if r >= 0 && r < len(s.Regions) && strongest(s, r, Enemy(u.Side)) != nil {
			return true
		}
	}
	return false
}
