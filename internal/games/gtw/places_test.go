// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package gtw

import (
	"slices"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/assets"
)

// Every scenario's targets are the enemy's and WOPR's aim points yours, each named once, and
// a target list no longer than the targets prompt takes. Its name is the montage's own, and
// no two scenarios share one.
func TestScenarioPlaces(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	for _, s := range scenarios {
		if !slices.Contains(assets.Scenarios, s.name) || seen[s.name.Text] {
			t.Errorf("%q is not a montage name, or is used twice", s.name.Text)
		}
		seen[s.name.Text] = true
		if len(s.targets) == 0 || len(s.targets) > maxTargets || len(s.brief) == 0 {
			t.Errorf("%s: %d targets, %d briefing lines", s.name.Text, len(s.targets), len(s.brief))
		}
		for _, group := range []struct {
			ps   []assets.Place
			side assets.Side
		}{{s.targets, enemyOf(s.side)}, {s.aims, s.side}} {
			names := map[string]bool{}
			for _, p := range group.ps {
				if p.Side != group.side || names[p.Name] {
					t.Errorf("%s: %s is on side %d, or named twice; want side %d", s.name.Text, p.Name, p.Side, group.side)
				}
				names[p.Name] = true
			}
		}
	}
}
