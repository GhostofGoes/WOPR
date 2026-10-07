// Package wopr is the persona: the dial-up, the LOGON gate, the film's scenes, the shell
// and WOPR's voice. It is a proto.Program and never touches the terminal.
package wopr

import (
	"context"
	"maps"

	"github.com/GhostofGoes/WOPR/internal/proto"
)

// Phase is where the conversation is.
type Phase uint8

// Phases. There is no game phase: a game is running when the host's stack is deeper than
// the persona. There is no ending phase either: the persona gets no input while the
// climax runs, and its GameOver (NoVerdict) returns the persona to Shell with chess offered.
const (
	PhaseDialing Phase = iota
	PhaseLogon
	PhaseGreeting
	PhaseShell
)

// Exchange is one turn of conversation, for a brain's context.
type Exchange struct {
	In  string
	Out []string
}

// Snapshot is a value copy of the session for a brain. It is safe to read on another
// goroutine while the session moves on.
type Snapshot struct {
	Phase   Phase
	Turn    uint64 // increments on every brain request, cancelled ones included
	Seed    uint64
	Said    map[string]bool
	Flags   map[string]bool
	Last    *proto.Result
	History []Exchange
}

// Effect is a change a brain asks the persona to make.
type Effect interface{ isEffect() }

// MarkSaid records that a once-only rule fired.
type MarkSaid struct{ ID string }

// SetFlag sets a conversation flag.
type SetFlag struct{ Key string }

// StartGame asks the persona to launch a game; the persona validates the slug.
type StartGame struct{ Slug string }

func (MarkSaid) isEffect()  {}
func (SetFlag) isEffect()   {}
func (StartGame) isEffect() {}

// Reply is a brain's answer.
type Reply struct {
	Lines   []string
	Effects []Effect
}

// Brain answers free conversation. It runs off the UI goroutine and must only read the
// snapshot it is given.
type Brain interface {
	Reply(ctx context.Context, s Snapshot, input string) (Reply, error)
}

func (s *Session) snapshot() Snapshot {
	snap := Snapshot{
		Phase: s.phase, Turn: s.turn, Seed: s.seed,
		Said: maps.Clone(s.said), Flags: maps.Clone(s.flags),
		History: append([]Exchange(nil), s.history...),
	}
	if s.last != nil {
		r := *s.last
		snap.Last = &r
	}
	return snap
}
