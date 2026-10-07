package wopr

import (
	"context"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/prompt"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// Rule is one entry of the scripted brain's table. The first matching rule wins.
type Rule struct {
	ID    string
	When  []Phase                       // empty: any phase
	Match func(cs []prompt.Clause) bool // over the input's clauses
	Lines Ls                            // fixed reply
	Pick  []Ls                          // or one of these, chosen with the snapshot's stream
	Once  bool                          // after it fires once, the rule is skipped
}

// Scripted is the deterministic brain used until (and unless) an LLM brain is enabled.
type Scripted struct{ Rules []Rule }

// NewScripted returns the default scripted brain.
func NewScripted() *Scripted { return &Scripted{Rules: defaultRules()} }

// Reply implements Brain.
func (b *Scripted) Reply(_ context.Context, s Snapshot, input string) (Reply, error) {
	cs := prompt.Clauses(input)
	for _, r := range b.Rules {
		if r.Once && s.Said[r.ID] {
			continue
		}
		if len(r.When) > 0 && !containsPhase(r.When, s.Phase) {
			continue
		}
		if r.Match != nil && !r.Match(cs) {
			continue
		}
		lines := r.Lines
		if len(r.Pick) > 0 {
			rng := proto.NewRand(s.Seed, proto.DomainBrain|s.Turn)
			lines = r.Pick[rng.IntN(len(r.Pick))]
		}
		var effects []Effect
		if r.Once {
			effects = append(effects, MarkSaid{ID: r.ID})
		}
		return Reply{Lines: lines.Texts(), Effects: effects}, nil
	}
	return Reply{Lines: lineRestate.Texts()}, nil
}

func containsPhase(ps []Phase, p Phase) bool {
	for _, x := range ps {
		if x == p {
			return true
		}
	}
	return false
}

// any reports whether some clause satisfies f.
func anyClause(f func(prompt.Clause) bool) func([]prompt.Clause) bool {
	return func(cs []prompt.Clause) bool {
		for _, c := range cs {
			if f(c) {
				return true
			}
		}
		return false
	}
}

// has matches a clause containing every word of phrase in order (not necessarily adjacent).
func has(phrase string) func([]prompt.Clause) bool {
	want := strings.Fields(phrase)
	return anyClause(func(c prompt.Clause) bool {
		i := 0
		for _, w := range c.Words {
			if i < len(want) && w == want[i] {
				i++
			}
		}
		return i == len(want)
	})
}

// starts matches a clause beginning with phrase.
func starts(phrase string) func([]prompt.Clause) bool {
	return anyClause(func(c prompt.Clause) bool { return prompt.HasPrefix(c.Words, phrase) })
}

func defaultRules() []Rule {
	return []Rule{
		{ID: "game-or-real", Match: has("GAME OR REAL"), Lines: lineWhatsDiff},
		{ID: "primary-goal", Match: has("PRIMARY GOAL"), Lines: lineToWin},
		{ID: "why", Match: starts("WHY"), Lines: lineProgrammed},
		{ID: "who", Match: func(cs []prompt.Clause) bool {
			return starts("WHO ARE YOU")(cs) || starts("WHAT ARE YOU")(cs)
		}, Lines: lineIAmWOPR},
		{ID: "joshua", Match: has("JOSHUA"), Lines: lineMyName},
		{ID: "how-are-you", Match: has("HOW ARE YOU"), Lines: lineFeelFine},
		{ID: "fallback", Pick: []Ls{lineRestate, lineImproper, lineShallWe}},
	}
}
