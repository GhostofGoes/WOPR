// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package proto

import "math/rand/v2"

// Stream domains keep every consumer's random sequence separate. A StreamID is
// Domain<<56 | id.
const (
	DomainBrain uint64 = 1 << 56 // id: brain turn
	DomainGame  uint64 = 2 << 56 // id: hash32(slug)<<16 | play index
	DomainAI    uint64 = 3 << 56 // id: think sequence number
	DomainMovie uint64 = 4 << 56 // id: scene number
	DomainUI    uint64 = 5 << 56 // id: 0
)

// NewRand is the only sanctioned way to make a random source (forbidigo bans the
// top-level math/rand/v2 functions). The same seed and stream always produce the same
// sequence, across Go releases.
func NewRand(seed, stream uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, stream)) //nolint:gosec // seeded on purpose for replays; never used for security
}

// GameStream returns the stream id for the playIndex-th game of slug in a session.
func GameStream(slug string, playIndex uint16) uint64 {
	return DomainGame | uint64(fnv32(slug))<<16 | uint64(playIndex)
}

// LaunchSeed is the Env.Seed the host gives the playIndex-th launch of slug in a session
// seeded session (docs/PLAN.md §4.6): every game and every replay draws its own sequence.
func LaunchSeed(session uint64, slug string, playIndex uint16) uint64 {
	return NewRand(session, GameStream(slug, playIndex)).Uint64()
}

func fnv32(s string) uint32 {
	h := uint32(2166136261)
	for i := range len(s) {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}
