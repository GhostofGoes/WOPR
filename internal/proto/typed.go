package proto

import (
	"math/rand/v2"
	"time"
)

// A simulated user's typing (movie mode, docs/PLAN.md §7): a hesitation before the line, then
// bursts of keystrokes at PaceTyping with short pauses between them.
const (
	typingHesitation = 450 * time.Millisecond // the shortest pause before the first keystroke
	typingHesitates  = 700 * time.Millisecond // how much longer it may be
	typingBurst      = 4                      // the most keystrokes without a pause
	typingPauses     = 220 * time.Millisecond // the longest pause between bursts
)

// Typed returns the outputs that show a user typing text after prompt, laid out as the console
// echoes an answer: the prompt at once, a pause, the text a few keystrokes at a time at
// PaceTyping on the same row, then a blank line. r jitters the pauses and the bursts (movie mode
// seeds it); with nil every keystroke is its own burst and the pauses are fixed. Under
// Env.Instant the host drops the pauses, so the line appears whole.
func Typed(prompt, text string, r *rand.Rand) []Output {
	outs := []Output{
		Say{Lines: []string{prompt}, Pace: PaceInstant, Open: true},
		Wait{D: typingHesitation + jitter(r, typingHesitates)},
	}
	keys := []rune(text)
	if len(keys) == 0 {
		outs = append(outs, Say{Lines: []string{""}, Pace: PaceInstant}) // Enter on its own
	}
	for len(keys) > 0 {
		n := 1
		if r != nil {
			n = 1 + r.IntN(typingBurst)
		}
		n = min(n, len(keys))
		chunk := string(keys[:n])
		keys = keys[n:]
		outs = append(outs, Say{Lines: []string{chunk}, Pace: PaceTyping, Open: len(keys) > 0})
		if d := jitter(r, typingPauses); len(keys) > 0 && d > 0 {
			outs = append(outs, Wait{D: d})
		}
	}
	return append(outs, Say{Lines: []string{""}, Pace: PaceInstant})
}

// jitter is a pause up to most, drawn from r; without r, half of most.
func jitter(r *rand.Rand, most time.Duration) time.Duration {
	if r == nil {
		return most / 2
	}
	return time.Duration(r.Int64N(int64(most)))
}
