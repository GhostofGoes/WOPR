// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package gtw

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/GhostofGoes/WOPR/internal/assets"
	"github.com/GhostofGoes/WOPR/internal/prompt"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// Scenario playbacks (docs/PLAN.md §6.2): every strategy of the film's closing montage, played
// out by itself on the big board, as WOPR plays each one against itself at the end of the
// film. Nobody gives an order. A scenario's war follows from its name, by two tables of rules:
// the words before the last name the theatre, which sets your side and both sides' targets,
// and the last word names the kind of war, which sets your side's orders, who strikes first
// and the enemy's doctrine. The montage's first four, the film's own wars, are set by hand.
// The names are the montage's (assets.Scenarios), never retyped here; the rules are original
// game design.

// ScenarioMode is the launch mode that starts GTW in a scenario's playback, with the
// scenario's number after a colon (ScenarioLaunch).
const ScenarioMode = "scenario"

// ScenarioLaunch is the launch mode for scenario n, numbered as Scenarios lists them:
// "scenario:4" (--scenario).
func ScenarioLaunch(n int) string { return fmt.Sprintf("%s:%d", ScenarioMode, n) }

const (
	briefPause = 3 * time.Second // how long a briefing stays up, once typed, before the board opens
	worldCols  = 72              // 360 degrees of longitude, at the map's 5 a column
	menuRows   = 18              // the most rows a menu page lists
	menuCols   = 3
	menuW      = 27 // a menu column: the number, the montage's longest names (21 letters) and a space
)

// scenario is one of the montage's strategies as a war that plays itself.
type scenario struct {
	number  int            // its place on the menu, from 1, and the seed its playback draws from
	name    script.L       // the montage's
	side    assets.Side    // yours
	c       conduct        // how both sides fight it
	targets []assets.Place // the enemy's places, your side's aim points
	aims    []assets.Place // your side's places, the enemy's aim points
}

// conduct is how a kind of war is fought: your side's order at each strike, in percent of
// ICBM, SLBM and bombers, and the doctrine the enemy answers with. An order of nothing at
// strike 1 leaves the first strike to the enemy, which fires its ladder's first rung whatever
// you do.
type conduct struct {
	orders [strikes][3]int
	d      doctrineID
}

var (
	hold = [3]int{}
	all  = [3]int{100, 100, 100}
)

// The kinds of war. Every one is lost (TestEveryScenarioIsLost).
var (
	planned    = conduct{warPlan, measured}                                              // the film's war
	assault    = conduct{[strikes][3]int{{50, 50, 100}, {50, 50, 100}, all}, measured}   // you strike first, hard
	preempt    = conduct{[strikes][3]int{{100, 0, 0}, {0, 50, 100}, all}, measured}      // every ICBM at the enemy's silos first
	surprise   = conduct{[strikes][3]int{hold, all, all}, hairTrigger}                   // the enemy strikes first, on a hair trigger
	provoked   = conduct{[strikes][3]int{hold, {25, 25, 0}, all}, measured}              // the enemy strikes first, small; you answer
	decapitate = conduct{[strikes][3]int{{0, 50, 0}, {50, 50, 100}, all}, counterforce}  // your submarines first, at command
	defend     = conduct{[strikes][3]int{hold, {100, 50, 100}, all}, counterforce}       // the enemy goes for your forces first
	escalate   = conduct{[strikes][3]int{{5, 5, 0}, {25, 25, 50}, all}, measured}        // a small first strike, then the climb
	optional   = conduct{warPlan, secondStrike}                                          // the war plan, against an enemy that rides it out
	alert      = conduct{[strikes][3]int{{0, 0, 100}, {50, 50, 0}, all}, hairTrigger}    // bombers up first, against an enemy on alert
	minimal    = conduct{[strikes][3]int{{5, 5, 0}, {50, 50, 100}, all}, measured}       // the war plan, from a smaller first strike
	limited    = conduct{[strikes][3]int{{10, 10, 0}, {50, 50, 100}, all}, measured}     // and a little larger
	heavy      = conduct{[strikes][3]int{{50, 50, 50}, {50, 50, 100}, all}, hairTrigger} // a larger first strike, answered in kind
	maximum    = conduct{[strikes][3]int{all, all, all}, hairTrigger}                    // everything at once
	farEast    = conduct{[strikes][3]int{{10, 25, 50}, {50, 50, 100}, all}, measured}    // half your bombers go with the first strike
	ussrFirst  = conduct{[strikes][3]int{hold, {100, 50, 100}, all}, hairTrigger}        // the Soviet Union strikes first, on a hair trigger
)

// kinds are the kinds of war by the last word of a name: what it says the war is, or how
// large. A name the montage cuts short (THEATERWID) is read as the word it starts.
var kinds = map[string]conduct{
	"WAR": planned, "MANEUVER": planned, "MEDIUM": planned,
	"STRIKE": assault, "ATTACK": assault, "OFFENSIVE": assault, "THRUST": assault, "TAKEOVER": assault,
	"INCURSION": assault, "CAMPAIGN": assault, "ACTION": assault,
	"PREEMPTIVE": preempt,
	"SURPRISE":   surprise, "SUDDEN": surprise,
	"PROVOCATION": provoked, "INCIDENT": provoked, "DECOY": provoked, "MISDIRECTION": provoked, "DIVERSION": provoked,
	"SUBVERSION": provoked, "CLANDESTINE": provoked, "PARAMILITARY": provoked, "REBELLION": provoked, "REVOLUTION": provoked,
	"DECAPITATING": decapitate, "SURGICAL": decapitate, "INTERDICTION": decapitate, "TACTICAL": decapitate,
	"DISCRETIONARY": decapitate,
	"DEFENSE":       defend, "COUNTER": defend, "CONTAINMENT": defend, "RETALIATION": defend, "SOVEREIGNTY": defend,
	"DOMESTIC": defend, "TERRITORIAL": defend,
	"ESCALATION": escalate, "CONFRONTATION": escalate, "ALLIANCE": escalate,
	"OPTION": optional, "ALTERNATE": optional, "VARIANT": optional, "VARIATION": optional,
	"ALERT":   alert,
	"MINIMAL": minimal,
	"LIMITED": limited, "LIGHT": limited, "LOCAL": limited,
	"HEAVY":   heavy,
	"MAXIMUM": maximum, "MASSIVE": maximum, "THEATERWIDE": maximum,
}

// theatre is where a war is fought: a point on the map, and the side it belongs to.
type theatre struct {
	lat, lon float64
	side     assets.Side // the side it names or is allied to; 0: whichever side's places are nearer
	global   bool        // the whole of both sides, as in the film's war
}

// theatres are the theatres by the words of a name before its last: the country, region or
// alliance it names, at about its middle, or the whole war. The words are spelt as the
// montage spells them.
//
//nolint:misspell // the film's own spelling
var theatres = map[string]theatre{
	"U.S.": {lat: 39, lon: -98, side: assets.US}, "ALASKAN": {lat: 64, lon: -150, side: assets.US},
	"HAWAIIAN": {lat: 21, lon: -157, side: assets.US}, "USSR": {lat: 60, lon: 60, side: assets.USSR},
	"RUSSIAN": {lat: 56, lon: 38, side: assets.USSR}, "USSR CHINA": {lat: 35, lon: 105, side: assets.USSR},
	"US USSR": {side: assets.US, global: true}, "INADVERTENT": {side: assets.USSR, global: true},
	"NATO": {lat: 50, lon: 8, side: assets.US}, "SEATO": {lat: 15, lon: 105, side: assets.US},
	"PACT":     {lat: 52, lon: 20, side: assets.USSR},
	"ATLANTIC": {lat: 40, lon: -40}, "PACIFIC": {lat: 30, lon: -160}, "ARCTIC": {lat: 72, lon: -150},
	"NORTHERN": {lat: 70, lon: 35}, "MEDITERRANEAN": {lat: 38, lon: 15}, "CASPIAN": {lat: 42, lon: 51},
	"MIDDLE EAST": {lat: 30, lon: 42}, "ASIAN": {lat: 35, lon: 100}, "AFRICAN": {lat: 5, lon: 20},
	"EASTERN": {lat: 50, lon: 22}, "SCANDINAVIAN": {lat: 62, lon: 15},
	"AFGHAN": {lat: 34, lon: 66}, "ALBANIAN": {lat: 41, lon: 20}, "ALGERIAN": {lat: 28, lon: 3},
	"ANGENTINA": {lat: -34, lon: -64}, "ARABIAN": {lat: 24, lon: 45}, "AUSTRALIAN": {lat: -25, lon: 135},
	"BANGLADESH": {lat: 24, lon: 90}, "BRAZILIAN": {lat: -10, lon: -52}, "BULGARIAN": {lat: 43, lon: 25},
	"BURMESE": {lat: 21, lon: 96}, "CAMBODIAN": {lat: 12, lon: 105}, "CANADIAN": {lat: 56, lon: -100},
	"CHAD": {lat: 15, lon: 19}, "CHILE": {lat: -30, lon: -71}, "CUBAN": {lat: 22, lon: -80},
	"CYPRESS": {lat: 35, lon: 33}, "CZECH": {lat: 50, lon: 15}, "DENMARK": {lat: 56, lon: 10},
	"EGYPT": {lat: 27, lon: 30}, "ENGLISH": {lat: 52, lon: -1}, "ETHIOPIAN": {lat: 9, lon: 40},
	"FRENCH": {lat: 46, lon: 2}, "GABON": {lat: -1, lon: 12}, "GREENLAND": {lat: 72, lon: -40},
	"HONGKONG": {lat: 22, lon: 114}, "ICELAND": {lat: 65, lon: -19}, "INDIA PAKISTAN": {lat: 28, lon: 72},
	"IRANIAN": {lat: 32, lon: 53}, "IRAQ": {lat: 33, lon: 44}, "ISREAL": {lat: 31, lon: 35},
	"ITALIAN": {lat: 42, lon: 12}, "JAMAICA": {lat: 18, lon: -77}, "JORDAN": {lat: 31, lon: 36},
	"KENYA": {lat: 0, lon: 38}, "LEBANON": {lat: 34, lon: 36}, "LIBYAN": {lat: 27, lon: 17},
	"MAYLASIAN": {lat: 4, lon: 102}, "MEXICAN": {lat: 23, lon: -102}, "MONGOLIAN": {lat: 46, lon: 105},
	"MOROCCAN": {lat: 32, lon: -6}, "NICARAGUAN": {lat: 13, lon: -85}, "PAKISTAN": {lat: 30, lon: 70},
	"PALISTINIAN": {lat: 32, lon: 35}, "PANAMA": {lat: 9, lon: -80}, "POLISH": {lat: 52, lon: 19},
	"PORTUGAL": {lat: 39, lon: -8}, "ROMANIAN": {lat: 46, lon: 25}, "S.AFRICAN": {lat: -29, lon: 24},
	"SAUDI": {lat: 24, lon: 45}, "SPAIN": {lat: 40, lon: -4}, "SUDAN": {lat: 15, lon: 30},
	"SWISS": {lat: 47, lon: 8}, "SYRIAN": {lat: 35, lon: 38}, "TAIWAN": {lat: 24, lon: 121},
	"THAI": {lat: 15, lon: 101}, "TUNISIAN": {lat: 34, lon: 9}, "TURKISH": {lat: 39, lon: 35},
	"UGANDA": {lat: 1, lon: 32}, "VENEZUELA": {lat: 7, lon: -66}, "VIETNAMESE": {lat: 16, lon: 107},
	"WELSH": {lat: 52, lon: -4}, "ZAIRE": {lat: -3, lon: 23},
}

// famous are the montage's first four, the film's own wars, set by hand: each with your side
// and its theatre, and how it is fought.
var famous = [...]struct {
	t theatre
	c conduct
}{
	{theatre{side: assets.US, global: true}, assault},      // U.S. FIRST STRIKE
	{theatre{side: assets.US, global: true}, ussrFirst},    // USSR FIRST STRIKE
	{theatre{lat: 52, lon: 15, side: assets.US}, escalate}, // NATO / WARSAW PACT: a war in Europe
	{theatre{lat: 45, lon: 135, side: assets.US}, farEast}, // FAR EAST STRATEGY
}

// scenarios are the montage's strategies, each name once, in the montage's order.
var scenarios = func() []scenario {
	var out []scenario
	seen := map[string]bool{}
	for i, name := range assets.Scenarios {
		if seen[name.Text] { // the montage runs CHAD ALERT twice
			continue
		}
		seen[name.Text] = true
		t, c := rulesFor(i, name.Text)
		out = append(out, t.scenario(len(out)+1, name, c))
	}
	return out
}()

// ruleWords splits a name into the words the rules read: its theatre, and its kind.
func ruleWords(name string) (place, kind string) {
	words := strings.Fields(name)
	return strings.Join(words[:len(words)-1], " "), words[len(words)-1]
}

// rulesFor is the theatre and the conduct of the montage's ith name. A name the rules do not
// cover is a mistake in this file, caught as the package loads.
func rulesFor(i int, name string) (theatre, conduct) {
	if i < len(famous) {
		return famous[i].t, famous[i].c
	}
	place, word := ruleWords(name)
	t, ok := theatres[place]
	c, known := kindOf(word)
	if !ok || !known {
		panic("gtw: no rule for the scenario " + name) // static data; TestEveryRuleIsUsed keeps it right
	}
	return t, c
}

// kindOf is the conduct a kind's word names: whole, or cut short, the start of one word only.
func kindOf(word string) (conduct, bool) {
	if c, ok := kinds[word]; ok {
		return c, true
	}
	var found []conduct
	for k, c := range kinds {
		if strings.HasPrefix(k, word) {
			found = append(found, c)
		}
	}
	if len(found) != 1 {
		return conduct{}, false
	}
	return found[0], true
}

// scenario is the war numbered n, named name and fought by c in this theatre. A regional war
// is fought over the places nearest the theatre on both sides; the whole war over the
// enemy's first cities and every one of yours, as in the film's.
func (t theatre) scenario(n int, name script.L, c conduct) scenario {
	side := t.side
	if side == 0 {
		side = t.nearer()
	}
	s := scenario{number: n, name: name, side: side, c: c}
	if t.global {
		s.targets, s.aims = cities(enemyOf(side))[:maxTargets], cities(side)
		return s
	}
	s.targets, s.aims = t.nearest(enemyOf(side))[:maxTargets], t.nearest(side)[:maxTargets]
	return s
}

func cities(side assets.Side) []assets.Place {
	var out []assets.Place
	for _, c := range assets.Cities {
		if c.Side == side {
			out = append(out, c)
		}
	}
	return out
}

// nearest is side's places on the board, nearest the theatre first, and a city before
// another place as near.
func (t theatre) nearest(side assets.Side) []assets.Place {
	var out []assets.Place
	for _, p := range append(slices.Clone(assets.Cities), assets.Targets...) {
		if p.Side == side {
			out = append(out, p)
		}
	}
	slices.SortStableFunc(out, func(a, b assets.Place) int { return cmp.Compare(t.cells(a), t.cells(b)) })
	return out
}

// nearer is the side whose nearest place is nearer the theatre: the Soviet Union's on a tie,
// the film's side.
func (t theatre) nearer() assets.Side {
	if t.cells(t.nearest(assets.US)[0]) < t.cells(t.nearest(assets.USSR)[0]) {
		return assets.US
	}
	return assets.USSR
}

// cells is how far p is from the theatre, squared, in map cells: across the Pacific, where
// that way round the world is shorter.
func (t theatre) cells(p assets.Place) int {
	x, y := assets.At(t.lat, t.lon)
	dx, dy := p.X-x, p.Y-y
	if dx < 0 {
		dx = -dx
	}
	dx = min(dx, worldCols-dx)
	return dx*dx + dy*dy
}

// Scenarios are the names of GTW's scenarios, numbered from 1 in this order: the montage's,
// each name once.
func Scenarios() []string {
	out := make([]string, len(scenarios))
	for i, s := range scenarios {
		out[i] = s.name.Text
	}
	return out
}

// FindScenario is the number of the scenario text names: its number, or its name, in any
// capitals, spacing and punctuation ("US FIRST STRIKE", "nato warsaw pact"), with or without
// SCENARIO before it. A name the montage cuts short is also found by its whole last word
// (VIETNAMESE RETALIATION).
func FindScenario(text string) (int, bool) {
	norm := strings.TrimPrefix(prompt.Normalize(text), "SCENARIO ")
	if n, err := strconv.Atoi(norm); err == nil {
		return n, n >= 1 && n <= len(scenarios)
	}
	in, found, longest := squash(norm), 0, 0
	for _, s := range scenarios {
		name := squash(s.name.Text)
		switch {
		case in == name:
			return s.number, true
		case strings.HasPrefix(in, name) && len(name) > longest:
			found, longest = s.number, len(name)
		}
	}
	return found, found > 0
}

// scenarioFor is the scenario a launch mode names (ScenarioLaunch), or nil.
func scenarioFor(mode string) *scenario {
	rest, ok := strings.CutPrefix(mode, ScenarioMode+":")
	n, err := strconv.Atoi(rest)
	if !ok || err != nil || n < 1 || n > len(scenarios) {
		return nil
	}
	return &scenarios[n-1]
}

// menuPages is how the menu splits the scenarios: as few pages as hold them at menuRows rows
// of menuCols, and as many on each.
func menuPages() (pages, per int) {
	pages = ceilDiv(len(scenarios), menuRows*menuCols)
	return pages, ceilDiv(len(scenarios), pages)
}

// showMenu puts up page p of the scenario menu (wrapping round), numbered down each column.
func (g *Game) showMenu(p int) []proto.Output {
	pages, per := menuPages()
	g.phase, g.page = chooseScenario, (p%pages+pages)%pages
	list := scenarios[g.page*per : min((g.page+1)*per, len(scenarios))]
	rows := ceilDiv(len(list), menuCols)
	lines := []string{fill(lineMenu[0], g.page+1, pages), ""}
	for r := range rows {
		var b strings.Builder
		for c := range menuCols {
			if i := c*rows + r; i < len(list) {
				fmt.Fprintf(&b, "%-*s", menuW, fmt.Sprintf("%3d. %s", list[i].number, list[i].name.Text))
			}
		}
		lines = append(lines, strings.TrimRight(b.String(), " "))
	}
	lines = append(lines, "", lineMenu[1].Text)
	return []proto.Output{proto.Clear{}, table(lines), proto.Prompt{Text: promptMenu[0].Text}}
}

// onMenu reads the scenario menu: a number or a name briefs that scenario, NEXT and PREV turn
// the page, and Enter or BACK returns to the side choice.
func (g *Game) onMenu(input string) []proto.Output {
	if strings.TrimSpace(input) == "" {
		g.phase = chooseSide
		return append([]proto.Output{proto.Clear{}}, sideScreen()...)
	}
	switch prompt.Normalize(input) {
	case "BACK":
		g.phase = chooseSide
		return append([]proto.Output{proto.Clear{}}, sideScreen()...)
	case "NEXT", "N", "MORE":
		return g.showMenu(g.page + 1)
	case "PREV", "PREVIOUS", "P":
		return g.showMenu(g.page - 1)
	case "LIST", "LIST SCENARIOS", "SCENARIOS", "HELP", "": // "" is "?"
		return g.showMenu(g.page)
	}
	if n, ok := FindScenario(input); ok {
		return g.brief(&scenarios[n-1])
	}
	return []proto.Output{speak([]string{fill(lineMenuAgain[0], len(scenarios))}), proto.Prompt{Text: promptMenu[0].Text}}
}

// brief sets scenario s up and briefs it: its name, your side and who strikes first, the
// enemy's doctrine, your side's orders and its targets. The board opens by itself once the
// briefing has been read, and the war plays itself, as under AUTO.
func (g *Game) brief(s *scenario) []proto.Output {
	g.scen, g.side, g.targets, g.phase, g.auto = s, s.side, slices.Clone(s.targets), briefing, true
	first, o := s.side, s.c.orders
	if o[0] == hold {
		first = enemyOf(s.side)
	}
	names := make([]string, len(s.targets))
	for i, p := range s.targets {
		names[i] = p.Name
	}
	lines := []string{
		fill(lineBrief[0], lineLabels[s.side-1].Text, lineLabels[first-1].Text),
		fill(lineBrief[1], doctrineName(s.c.d)),
		fill(lineBrief[2], order(o[0]), order(o[1]), order(o[2])),
	}
	lines = append(append(lines, commaLines(fill(lineBrief[3], strings.Join(names, ", ")))...), lineBrief[4].Text)
	return []proto.Output{
		proto.Clear{},
		table([]string{s.name.Text, strings.Repeat("-", len(s.name.Text)), ""}),
		speak(lines),
		proto.Wait{D: briefPause},
		proto.Drain{},
	}
}

// order is an order as the briefing gives it: "50 50 100", or HOLD.
func order(o [3]int) string {
	if o == hold {
		return lineHold[0].Text
	}
	return fmt.Sprintf("%d %d %d", o[0], o[1], o[2])
}

// verdict is what a playback's kill ratios end with: the montage's own row, the strategy and
// WINNER: NONE.
func (g *Game) verdict() []string {
	head := assets.Strategy.Texts()
	return []string{head[0] + " " + g.scen.name.Text, head[1] + " " + head[2]}
}

// endPlayback clears the war away and returns to the menu, at the page of the scenario just
// played.
func (g *Game) endPlayback() []proto.Output {
	_, per := menuPages()
	page := (g.scen.number - 1) / per
	g.reset()
	outs := []proto.Output{proto.Animate{}, proto.SetLayout{Layout: proto.LayoutConsole}}
	return append(outs, g.showMenu(page)...)
}
