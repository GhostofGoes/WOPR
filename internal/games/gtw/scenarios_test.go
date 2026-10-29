// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package gtw

import (
	"slices"
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/assets"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// The scenarios are the montage's names, each once, in the montage's order: all 156 of its
// 157 rows, since it runs CHAD ALERT twice. A name is found by its number and by its own
// text, and by no other scenario's.
func TestScenariosAreTheMontage(t *testing.T) {
	t.Parallel()
	var want []string
	for _, n := range assets.Scenarios.Texts() {
		if !slices.Contains(want, n) {
			want = append(want, n)
		}
	}
	if got := Scenarios(); !slices.Equal(got, want) || len(got) != 156 {
		t.Fatalf("%d scenarios, want the montage's %d names in order", len(got), len(want))
	}
	for i, s := range scenarios {
		if s.number != i+1 || !slices.Contains(assets.Scenarios, s.name) {
			t.Errorf("scenario %d is %q, numbered %d", i+1, s.name.Text, s.number)
		}
		if n, ok := FindScenario(s.name.Text); !ok || n != s.number {
			t.Errorf("%q finds %d, %v", s.name.Text, n, ok)
		}
	}
}

// Every rule is used, and every name is read by one: no theatre or kind of war is left over,
// and a word the montage cuts short is the start of one kind only.
func TestEveryRuleIsUsed(t *testing.T) {
	t.Parallel()
	places, words := map[string]bool{}, map[string]bool{}
	for i, name := range assets.Scenarios.Texts()[len(famous):] {
		place, word := ruleWords(name)
		if _, ok := theatres[place]; !ok {
			t.Errorf("%d %q: no theatre %q", i+len(famous), name, place)
		}
		places[place] = true
		if _, ok := kinds[word]; !ok {
			var starts []string
			for k := range kinds {
				if strings.HasPrefix(k, word) {
					starts = append(starts, k)
				}
			}
			if len(starts) != 1 {
				t.Errorf("%q: %q starts %q", name, word, starts)
			}
			word = starts[0]
		}
		words[word] = true
	}
	for p := range theatres {
		if !places[p] {
			t.Errorf("no scenario is fought in the theatre %q", p)
		}
	}
	for k := range kinds {
		if !words[k] {
			t.Errorf("no scenario is of the kind %q", k)
		}
	}
}

// Every scenario's targets are eight of the enemy's places and its aim points the player's,
// each named once.
func TestScenarioPlaces(t *testing.T) {
	t.Parallel()
	for _, s := range scenarios {
		if len(s.targets) != maxTargets || len(s.aims) == 0 {
			t.Errorf("%s: %d targets, %d aim points", s.name.Text, len(s.targets), len(s.aims))
		}
		for _, group := range []struct {
			ps   []assets.Place
			side assets.Side
		}{{s.targets, enemyOf(s.side)}, {s.aims, s.side}} {
			seen := map[string]bool{}
			for _, p := range group.ps {
				if p.Side != group.side || seen[p.Name] {
					t.Errorf("%s: %s is on side %d, or named twice; want side %d", s.name.Text, p.Name, p.Side, group.side)
				}
				seen[p.Name] = true
			}
		}
	}
	// A theatre nearer one side is that side's war: Cuba's is the United States'.
	cuban := scenarios[11]
	if cuban.side != assets.US || cuban.aims[0].Name != "MIAMI" {
		t.Errorf("%s: side %d, aimed first at %s", cuban.name.Text, cuban.side, cuban.aims[0].Name)
	}
}

// Every scenario is lost: whoever strikes first, your side takes more warheads on its cities
// than it puts on the enemy's, more of its people die, and no civilian or human row of the
// kill ratios is better for you, with the playback's own noise and at either end of the noise.
// The war the board plays is the one the exchange wages for the scenario's orders.
func TestEveryScenarioIsLost(t *testing.T) {
	t.Parallel()
	for _, s := range scenarios {
		w, _ := wage(s.c.d, s.c.orders)
		if w.t[you].Cities <= w.t[wopr].Cities {
			t.Errorf("%s: warheads on your cities %d, on the enemy's %d", s.name.Text, w.t[you].Cities, w.t[wopr].Cities)
		}
		g := New().(*Game)
		g.Start(proto.Env{Seed: 7, Instant: true, Width: 80, Height: 19, Mode: ScenarioLaunch(s.number)})
		g.Handle(proto.Drained{})
		if g.phase != assessed || g.war.t != w.t {
			t.Fatalf("%s: the playback reached phase %d with tallies %+v; the exchange's %+v", s.name.Text, g.phase, g.war.t, w.t)
		}
		var low, high [10]int
		for i := range low {
			low[i], high[i] = 97, 103
		}
		for _, noise := range [][10]int{g.noise, low, high} {
			r := w.ratios(noise)
			for row := 5; row < 12; row++ {
				if r[you][row] < r[wopr][row] {
					t.Errorf("%s: row %d favours you: %v", s.name.Text, row, r)
				}
			}
			if r[you][11] <= r[wopr][11] {
				t.Errorf("%s: your dead %d million, the enemy's %d", s.name.Text, r[you][11], r[wopr][11])
			}
		}
	}
}
