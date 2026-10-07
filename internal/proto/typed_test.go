package proto

import (
	"strings"
	"testing"
)

// A typed line is the prompt, then the text in bursts on the same row, then a blank line, as
// the console echoes an answer; seeded jitter varies the bursts but never the text.
func TestTyped(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ prompt, text string }{
		{"LOGON: ", "Joshua"}, {"", "Is this a game or is it real?"}, {"STRIKE 2 OF 3 [50 50 100]: ", ""}, {"", ""},
	} {
		for _, seeded := range []bool{false, true} {
			r := NewRand(1, DomainMovie)
			if !seeded {
				r = nil
			}
			var rows []string
			open := false
			for i, o := range Typed(tc.prompt, tc.text, r) {
				switch o := o.(type) {
				case Say:
					if open {
						rows[len(rows)-1] += o.Lines[0]
					} else {
						rows = append(rows, o.Lines...)
					}
					open = o.Open
					if o.Pace != PaceTyping && o.Pace != PaceInstant {
						t.Errorf("%q: output %d at pace %d", tc.text, i, o.Pace)
					}
				case Wait:
					if !open || o.D <= 0 {
						t.Errorf("%q: a pause belongs inside the line: %+v", tc.text, o)
					}
				default:
					t.Errorf("%q: unexpected output %T", tc.text, o)
				}
			}
			if want := []string{tc.prompt + tc.text, ""}; open || strings.Join(rows, "|") != strings.Join(want, "|") {
				t.Errorf("seeded %v: rows %q (open %v), want %q", seeded, rows, open, want)
			}
		}
	}
}
