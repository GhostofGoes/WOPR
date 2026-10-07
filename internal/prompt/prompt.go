// Package prompt parses what the user types: normalisation, clauses, menu choices,
// numbers and yes/no answers. It is pure and shared by the persona and every game.
package prompt

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// Normalize upper-cases s, turns typographic apostrophes into ASCII ones, and collapses
// every run of characters other than A-Z, 0-9 and the apostrophe into a single space.
// The result has no leading or trailing space.
//
//	Normalize("Later. Let’s play  Global Thermonuclear War!") == "LATER LET'S PLAY GLOBAL THERMONUCLEAR WAR"
func Normalize(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	space := false
	for _, r := range s {
		switch {
		case r == '’' || r == '‘' || r == '`':
			r = '\''
		case r >= 'a' && r <= 'z':
			r -= 'a' - 'A'
		}
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '\'' {
			if space && b.Len() > 0 {
				b.WriteByte(' ')
			}
			space = false
			b.WriteRune(r)
			continue
		}
		space = true
	}
	return b.String()
}

// Clause is one sentence-like part of an input line.
type Clause struct {
	Raw   string   // the text as typed, trimmed
	Norm  string   // Normalize(Raw), with LET'S / LET US folded to LETS
	Words []string // the words of Norm
}

// clauseBreaks are the characters that end a clause.
const clauseBreaks = ".!?;,"

// Clauses splits an input line into clauses at . ! ? ; and , and drops empty ones.
//
//	Clauses("Love to. How about Global Thermonuclear War?") → ["LOVE TO", "HOW ABOUT GLOBAL THERMONUCLEAR WAR"]
func Clauses(s string) []Clause {
	var out []Clause
	for _, part := range strings.FieldsFunc(s, func(r rune) bool { return strings.ContainsRune(clauseBreaks, r) }) {
		norm := foldLets(Normalize(part))
		if norm == "" {
			continue
		}
		out = append(out, Clause{Raw: strings.TrimSpace(part), Norm: norm, Words: strings.Fields(norm)})
	}
	return out
}

func foldLets(norm string) string {
	words := strings.Fields(norm)
	out := words[:0]
	for i := 0; i < len(words); i++ {
		switch {
		case words[i] == "LET'S":
			out = append(out, "LETS")
		case words[i] == "LET" && i+1 < len(words) && words[i+1] == "US":
			out = append(out, "LETS")
			i++
		default:
			out = append(out, words[i])
		}
	}
	return strings.Join(out, " ")
}

// HasPrefix reports whether words starts with the words of phrase (already normalised).
func HasPrefix(words []string, phrase string) bool {
	p := strings.Fields(phrase)
	if len(p) > len(words) {
		return false
	}
	for i := range p {
		if words[i] != p[i] {
			return false
		}
	}
	return true
}

// numberWords maps spelled-out numbers, as players type them, to values.
var numberWords = map[string]int{
	"ZERO": 0, "NONE": 0, "ONE": 1, "TWO": 2, "THREE": 3, "FOUR": 4, "FIVE": 5,
	"SIX": 6, "SEVEN": 7, "EIGHT": 8, "NINE": 9, "TEN": 10, "ELEVEN": 11, "TWELVE": 12,
	"THIRTEEN": 13, "FOURTEEN": 14, "FIFTEEN": 15,
}

// Number parses a whole input as a non-negative number: digits ("7", "7.", "#7") or a
// spelled-out number from zero to fifteen ("ZERO", "seven").
func Number(s string) (int, bool) {
	t := strings.TrimPrefix(strings.TrimSpace(s), "#")
	if strings.HasPrefix(t, "-") || strings.HasPrefix(t, "+") {
		return 0, false // Normalize would drop the sign
	}
	n := Normalize(t)
	if n == "" || strings.ContainsRune(n, ' ') {
		return 0, false
	}
	if v, ok := numberWords[n]; ok {
		return v, true
	}
	if len(n) > 6 { // no menu or player count needs more; also bounds Atoi on pasted junk
		return 0, false
	}
	v, err := strconv.Atoi(n)
	if err != nil || v < 0 {
		return 0, false
	}
	return v, true
}

// MenuChoice parses a selection from a numbered menu of n entries (1..n).
func MenuChoice(s string, n int) (int, bool) {
	v, ok := Number(s)
	if !ok || v < 1 || v > n {
		return 0, false
	}
	return v, true
}

// Answer is a parsed yes/no reply.
type Answer int

// Possible answers.
const (
	Unclear Answer = iota // neither yes nor no
	Yes
	No
)

var (
	yesPhrases = []string{"YES", "Y", "YEAH", "YEP", "YUP", "SURE", "OK", "OKAY", "LOVE TO", "FINE", "AFFIRMATIVE", "CERTAINLY", "OF COURSE", "WHY NOT", "ALRIGHT", "ALL RIGHT"}
	noPhrases  = []string{"NO", "N", "NOPE", "NAH", "NEGATIVE", "NOT NOW", "LATER", "NO THANKS", "NO THANK YOU", "NEVER"}
)

// YesNo parses a reply to a yes/no question from its first clause. A clause that starts
// with a no phrase is No ("No I wouldn't", "Never mind"), except NO PROBLEM and NO DOUBT;
// one that starts with a yes phrase is Yes unless a negator follows ("Of course not",
// "Yes but not now" stay Unclear). "Love to. How about chess?" is Yes; "Later." is No.
func YesNo(s string) Answer {
	cs := Clauses(s)
	if len(cs) == 0 {
		return Unclear
	}
	words := cs[0].Words
	if !HasPrefix(words, "NO PROBLEM") && !HasPrefix(words, "NO DOUBT") {
		for _, p := range noPhrases {
			if HasPrefix(words, p) {
				return No
			}
		}
	}
	for _, p := range yesPhrases {
		if n := len(strings.Fields(p)); HasPrefix(words, p) && !NegatedBefore(words[n:], len(words)) {
			return Yes
		}
	}
	return Unclear
}

// Negators are words that negate a request ("I DON'T WANT TO PLAY CHESS").
var Negators = map[string]bool{"NOT": true, "DON'T": true, "DONT": true, "NO": true, "NEVER": true, "WON'T": true, "WONT": true, "CAN'T": true, "CANT": true} // codespell:ignore wont,cant

// NegatedBefore reports whether any of words[:i] is a negator.
func NegatedBefore(words []string, i int) bool {
	for _, w := range words[:min(i, len(words))] {
		if Negators[w] {
			return true
		}
	}
	return false
}

// Valid reports whether s is valid UTF-8; callers sanitise before parsing, so this is a
// cheap guard for fuzzing.
func Valid(s string) bool { return utf8.ValidString(s) }
