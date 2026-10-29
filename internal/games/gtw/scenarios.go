// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package gtw

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/assets"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// scenario is one of the closing montage's strategies as a war to play (docs/PLAN.md §6.2):
// the montage's name, the player's side, a briefing, the targets, WOPR's aim points and the
// war's doctrine. The names are the montage's own (assets.Scenarios), never retyped here; the
// rest is original game design.
type scenario struct {
	name    script.L
	side    assets.Side
	brief   script.Ls      // what makes this war different, in WOPR's words
	targets []assets.Place // the enemy's, as if listed
	aims    []assets.Place // WOPR's aim points on your side; nil for every city there
	d       doctrine
}

// places is the places the board knows by these names (assets.Find). A name it does not
// know is a mistake in this file, caught as the package loads.
func places(names ...string) []assets.Place {
	out := make([]assets.Place, 0, len(names))
	for _, n := range names {
		p, ok := assets.Find(n)
		if !ok {
			panic("gtw: no place " + n) // static data; TestScenarioPlaces keeps it right
		}
		out = append(out, p)
	}
	return out
}

// Theatres: the places a regional war is fought over, on each side.
var (
	usPacific   = places("SEATTLE", "PORTLAND", "SAN FRANCISCO", "LOS ANGELES", "SAN DIEGO", "ANCHORAGE", "HONOLULU")
	usAtlantic  = places("NEW YORK", "BOSTON", "WASHINGTON", "PHILADELPHIA", "NORFOLK", "CHARLESTON", "JACKSONVILLE", "MIAMI")
	usArctic    = places("ANCHORAGE", "SEATTLE", "SPOKANE", "GREAT FALLS", "MINOT", "GRAND FORKS")
	ussrFarEast = places("VLADIVOSTOK", "KHABAROVSK", "PETROPAVLOVSK", "MAGADAN", "YAKUTSK", "CHITA", "IRKUTSK")
	ussrNorth   = places("MURMANSK", "SEVEROMORSK", "ARKHANGELSK", "NORILSK", "YAKUTSK", "MAGADAN", "PETROPAVLOVSK")
)

// scenarios are the wars SCENARIOS offers, in the montage's order. Each changes one or two
// things from the film's war, and none can be won (TestNoAllocationWins plays each).
var scenarios = []scenario{
	{
		name: assets.Scenarios[0], // U.S. FIRST STRIKE
		side: assets.US,
		brief: script.Orig(
			"THE UNITED STATES STRIKES FIRST, PROFESSOR. HALF YOUR MISSILES AND",
			"EVERY BOMBER LEAVE AT ONCE. THE SOVIET UNION LAUNCHES ON WARNING.",
		),
		targets: places("MOSCOW", "LENINGRAD", "KIEV", "GORKY", "SVERDLOVSK", "NOVOSIBIRSK", "MURMANSK", "VLADIVOSTOK"),
		d:       with(filmDoctrine, plan(50, 50, 100)),
	},
	{
		name: assets.Scenarios[1], // USSR FIRST STRIKE
		side: assets.US,
		brief: script.Orig(
			"THE SOVIET UNION STRIKES FIRST. ITS MISSILES ARE IN THE AIR, PROFESSOR.",
			"YOUR FORCES WAIT FOR ORDERS. YOU ANSWER AT STRIKE 2.",
		),
		targets: places("MOSCOW", "LENINGRAD", "KIEV", "MINSK", "GORKY", "KHARKOV", "SVERDLOVSK", "TASHKENT"),
		d:       with(filmDoctrine, plan(0, 0, 0), first(500, 150, 375)),
	},
	{
		name: assets.Scenarios[2], // NATO / WARSAW PACT
		side: assets.US,
		brief: script.Orig(
			"A WAR IN EUROPE, PROFESSOR. THE FIRST EXCHANGE IS LIMITED TO THE",
			"WARSAW PACT'S WESTERN FRONT. THE ANSWER FALLS ON THE ATLANTIC SEABOARD.",
		),
		targets: places("KALININGRAD", "RIGA", "VILNIUS", "MINSK", "KIEV", "KISHINEV", "ODESSA", "LENINGRAD"),
		aims:    places("WASHINGTON", "NEW YORK", "BOSTON", "PHILADELPHIA", "BALTIMORE", "NORFOLK", "PITTSBURGH", "BUFFALO"),
		d:       with(filmDoctrine, plan(5, 10, 0), plan2(25, 25, 50), ladder(2, 20, 30)),
	},
	{
		name: assets.Scenarios[3], // FAR EAST STRATEGY
		side: assets.US,
		brief: script.Orig(
			"THE WAR IS FOUGHT ACROSS THE PACIFIC, PROFESSOR. HALF YOUR BOMBERS",
			"LEAVE WITH THE FIRST STRIKE. THE SOVIET FAR EAST ANSWERS.",
		),
		targets: ussrFarEast,
		aims:    usPacific,
		d:       with(filmDoctrine, plan(10, 25, 50)),
	},
	{
		name: assets.Scenarios[4], // US USSR ESCALATION
		side: assets.USSR,
		brief: script.Orig(
			"THE FIRST STRIKE IS SMALL, PROFESSOR. THE UNITED STATES ANSWERS",
			"EVERY LAUNCH HALF AGAIN, AND ITS LADDER CLIMBS FAST.",
		),
		targets: places("WASHINGTON", "NEW YORK", "CHICAGO", "LOS ANGELES", "HOUSTON", "OMAHA"),
		d:       with(filmDoctrine, plan(5, 5, 0), plan2(25, 25, 50), ladder(2, 15, 40), answer(150)),
	},
	{
		name: assets.Scenarios[6], // USSR CHINA ATTACK
		side: assets.USSR,
		brief: script.Orig(
			"THE SOVIET UNION IS AT WAR WITH CHINA, PROFESSOR. A THIRD OF YOUR",
			"MISSILES ARE SPENT IN THE EAST. THE UNITED STATES HAS SPENT NONE.",
		),
		targets: usPacific,
		d:       with(filmDoctrine, yours(forces{ICBM: 700, SLBM: 400, Ground: 200})),
	},
	{
		name: assets.Scenarios[10], // SEATO DECAPITATING
		side: assets.US,
		brief: script.Orig(
			"YOUR SUBMARINES STRIKE MOSCOW FIRST, PROFESSOR, TO CUT OFF ITS HEAD.",
			"BOTH SIDES AIM AT COMMAND. LAUNCH ON WARNING NEEDS NO HEAD.",
		),
		targets: places("MOSCOW", "KUYBYSHEV"),
		aims:    places("WASHINGTON", "COLORADO SPRINGS", "OMAHA"),
		d:       with(filmDoctrine, plan(0, 50, 0)),
	},
	{
		name: assets.Scenarios[11], // CUBAN PROVOCATION
		side: assets.US,
		brief: script.Orig(
			"SOVIET SUBMARINES OFF THE COAST FIRE FIRST, PROFESSOR. THEIR FLIGHT",
			"IS SHORT. YOUR FIRST STRIKE LAUNCHES ON WARNING.",
		),
		targets: places("MOSCOW", "LENINGRAD", "SEVASTOPOL", "MURMANSK", "SEVEROMORSK"),
		aims:    places("MIAMI", "TAMPA", "ORLANDO", "JACKSONVILLE", "ATLANTA", "NEW ORLEANS", "HOUSTON", "CHARLESTON"),
		d:       with(filmDoctrine, first(0, 120, 0)),
	},
	{
		name: assets.Scenarios[12], // INADVERTENT INCIDENT
		side: assets.US,
		brief: script.Orig(
			"ONE SOVIET MISSILE HAS LEFT ITS SUBMARINE, PROFESSOR. NOBODY",
			"ORDERED IT. YOU ANSWER AT STRIKE 2, OR YOU HOLD.",
		),
		targets: places("MOSCOW", "LENINGRAD", "MURMANSK", "KIEV"),
		d:       with(filmDoctrine, plan(0, 0, 0), first(0, 1, 0)),
	},
	{
		name: assets.Scenarios[14], // ATLANTIC HEAVY
		side: assets.USSR,
		brief: script.Orig(
			"THE WAR IS FOUGHT FROM THE SEA, PROFESSOR. YOUR SUBMARINES CARRY",
			"MOST OF YOUR WARHEADS, AND THE ATLANTIC SEABOARD IS IN RANGE.",
		),
		targets: usAtlantic,
		aims:    places("LENINGRAD", "MURMANSK", "SEVEROMORSK", "ARKHANGELSK", "TALLINN", "RIGA", "KALININGRAD", "MOSCOW"),
		d: with(filmDoctrine, sized(forces{ICBM: 500, SLBM: 1000, Ground: 200}), plan(10, 40, 0),
			plan2(25, 50, 100)),
	},
	{
		name: assets.Scenarios[17], // PACIFIC TERRITORIAL
		side: assets.USSR,
		brief: script.Orig(
			"THE WAR IS FOR THE PACIFIC, PROFESSOR. YOUR SUBMARINES OPEN IT.",
			"THE UNITED STATES ANSWERS ON THE SOVIET FAR EAST.",
		),
		targets: usPacific,
		aims:    ussrFarEast,
		d:       with(filmDoctrine, plan(0, 25, 0), plan2(25, 50, 50), ladder(5, 20, 30)),
	},
	{
		name: assets.Scenarios[47], // ARCTIC MINIMAL
		side: assets.USSR,
		brief: script.Orig(
			"A SMALL WAR ACROSS THE POLE, PROFESSOR. EACH SIDE HAS A FIFTH OF ITS",
			"FORCES. A SMALL WAR HAS NO WINNER EITHER.",
		),
		targets: usArctic,
		aims:    ussrNorth,
		d:       with(filmDoctrine, sized(forces{ICBM: 200, SLBM: 120, Ground: 60})),
	},
}

// scenarioBriefs is every scenario's briefing, for Lines.
func scenarioBriefs() []script.Ls {
	out := make([]script.Ls, len(scenarios))
	for i, s := range scenarios {
		out[i] = s.brief
	}
	return out
}

// with is d changed by each of changes, in turn.
func with(d doctrine, changes ...func(*doctrine)) doctrine {
	for _, c := range changes {
		c(&d)
	}
	return d
}

// plan sets the first strike's war plan, and plan2 the second's. The third is always
// everything.
func plan(icbm, slbm, bombers int) func(*doctrine) {
	return func(d *doctrine) { d.plan[0] = [3]int{icbm, slbm, bombers} }
}

func plan2(icbm, slbm, bombers int) func(*doctrine) {
	return func(d *doctrine) { d.plan[1] = [3]int{icbm, slbm, bombers} }
}

// ladder sets WOPR's floors.
func ladder(s1, s2, s3 int) func(*doctrine) {
	return func(d *doctrine) { d.ladder = [strikes]int{s1, s2, s3} }
}

// answer sets how hard WOPR answers each launch, in percent.
func answer(pct int) func(*doctrine) { return func(d *doctrine) { d.answer = pct } }

// first gives WOPR a first strike.
func first(icbm, slbm, bombers int) func(*doctrine) {
	return func(d *doctrine) { d.first = [3]int{icbm, slbm, bombers} }
}

// sized sets your forces, and WOPR's a quarter more; yours sets yours alone.
func sized(f forces) func(*doctrine) { return func(d *doctrine) { d.forces = withQuarterMore(f) } }

func yours(f forces) func(*doctrine) { return func(d *doctrine) { d.forces[you] = f } }

// findScenario is the scenario a line names: its number in the list, or its name.
func findScenario(norm string) (*scenario, bool) {
	if n, err := strconv.Atoi(norm); err == nil {
		if n >= 1 && n <= len(scenarios) {
			return &scenarios[n-1], true
		}
		return nil, false
	}
	return scenarioNamed(norm)
}

// scenarioNamed is the scenario a line names by its name, compared without spaces or
// punctuation (US FIRST STRIKE, NATO WARSAW PACT).
func scenarioNamed(norm string) (*scenario, bool) {
	for i := range scenarios {
		if squash(scenarios[i].name.Text) == squash(norm) {
			return &scenarios[i], true
		}
	}
	return nil, false
}

// Scenario list geometry: SCENARIOS prints the list in two columns of scenarioW.
const scenarioW = 40

// scenarioList is SCENARIOS: the list, numbered as WOPR numbers its lists, down two columns.
func scenarioList() []string {
	rows := ceilDiv(len(scenarios), 2)
	lines := []string{lineScenarios[0].Text, ""}
	for r := range rows {
		var b strings.Builder
		for c := range 2 {
			if i := c*rows + r; i < len(scenarios) {
				fmt.Fprintf(&b, "%-*s", scenarioW, fmt.Sprintf("%3d.  %s", i+1, scenarios[i].name.Text))
			}
		}
		lines = append(lines, strings.TrimRight(b.String(), " "))
	}
	return append(lines, "")
}
