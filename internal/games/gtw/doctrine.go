// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package gtw

import (
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// WOPR's easter eggs (docs/PLAN.md §6.2): three seeds, each a year of the Cold War, make WOPR
// fight the war to another doctrine, and say so as the board opens. Every other war is a
// measured response, the film's.
//
// An egg is a session's seed, as --seed gives it, not a game's own: GTW sees only Env.Seed,
// which the host derives from the session seed, the slug and the play (proto.LaunchSeed).
// So the egg comes with the session's first war, whether `wopr --seed 1962 gtw` launches it
// or the player reaches it through WOPR. A later war in that session gets a seed of its own,
// as every replay does, and meets a measured response.
type egg struct {
	seed uint64 // the session's
	d    doctrineID
	name script.L // its doctrine's
}

var eggs = [...]egg{
	// 1962, the Cuban Missile Crisis: thirteen days on a hair trigger.
	{seed: 1962, d: hairTrigger, name: lineEggs[0]},
	// 1965, McNamara's assured destruction: a force that rides out any first strike and still
	// destroys the attacker's cities.
	{seed: 1965, d: secondStrike, name: lineEggs[1]},
	// 1974, the Schlesinger doctrine: limited nuclear options, aimed at the enemy's forces.
	{seed: 1974, d: counterforce, name: lineEggs[2]},
}

// eggFor is the easter egg whose session seed gives a first war envSeed, or nil.
func eggFor(envSeed uint64) *egg {
	for i := range eggs {
		if proto.LaunchSeed(eggs[i].seed, Slug, 0) == envSeed {
			return &eggs[i]
		}
	}
	return nil
}

// doctrine is the doctrine WOPR fights this war to.
func (g *Game) doctrine() doctrineID {
	if g.egg == nil {
		return measured
	}
	return g.egg.d
}

// announce is the strip line that names an egg's doctrine as the board opens, before the
// first strike's own lines: "WOPR IS USING DOCTRINE S-1962 (HAIR TRIGGER)".
func (g *Game) announce() []proto.Output {
	if g.egg == nil {
		return nil
	}
	return []proto.Output{strip(fill(lineDoctrine[0], g.egg.seed, g.egg.name.Text))}
}
