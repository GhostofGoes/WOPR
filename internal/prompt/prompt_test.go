package prompt

import (
	"strings"
	"testing"
)

func TestNormalize(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"Later. Let’s play  Global Thermonuclear War!": "LATER LET'S PLAY GLOBAL THERMONUCLEAR WAR",
		"  tic-tac-toe ":        "TIC TAC TOE",
		"6/23/73":               "6 23 73",
		"air-to-ground actions": "AIR TO GROUND ACTIONS",
		"":                      "",
		"!!!":                   "",
		"Falken's Maze":         "FALKEN'S MAZE",
	} {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestClauses(t *testing.T) {
	t.Parallel()
	got := Clauses("Love to. How about Global Thermonuclear War?")
	if len(got) != 2 || got[0].Norm != "LOVE TO" || got[1].Norm != "HOW ABOUT GLOBAL THERMONUCLEAR WAR" {
		t.Fatalf("film line split wrong: %+v", got)
	}
	got = Clauses("Later. Let's play Global Thermonuclear War.")
	if len(got) != 2 || got[1].Norm != "LETS PLAY GLOBAL THERMONUCLEAR WAR" {
		t.Fatalf("LET'S not folded: %+v", got)
	}
	for _, in := range []string{"lets play chess", "let us play chess", "LET'S PLAY CHESS"} {
		if c := Clauses(in); len(c) != 1 || c[0].Norm != "LETS PLAY CHESS" {
			t.Errorf("Clauses(%q) = %+v", in, c)
		}
	}
	if c := Clauses(" . , ! "); len(c) != 0 {
		t.Errorf("empty clauses kept: %+v", c)
	}
}

func TestNumber(t *testing.T) {
	t.Parallel()
	ok := map[string]int{"7": 7, "7.": 7, "#15": 15, "\tzero\t": 0, "Seven": 7, "0": 0}
	for in, want := range ok {
		if got, good := Number(in); !good || got != want {
			t.Errorf("Number(%q) = %d,%v want %d", in, got, good, want)
		}
	}
	for _, in := range []string{"", "seven games", "-1", "99999999999999999999", "7 8", "x", "no"} { // "no" answers yes/no questions
		if _, good := Number(in); good {
			t.Errorf("Number(%q) accepted", in)
		}
	}
	if _, good := MenuChoice("16", 15); good {
		t.Error("MenuChoice must reject out-of-range")
	}
	if v, good := MenuChoice("15", 15); !good || v != 15 {
		t.Error("MenuChoice(15) failed")
	}
}

func TestYesNo(t *testing.T) {
	t.Parallel()
	cases := map[string]Answer{
		"Yes":                       Yes,
		"Love to. How about chess?": Yes,
		"ok please":                 Yes,
		"Later.":                    No,
		"Later. Let's play Global Thermonuclear War.": No,
		"no":                 No,
		"maybe":              Unclear,
		"":                   Unclear,
		"yes, but not chess": Yes,
		"Yes I would":        Yes,
		"Yeah sure":          Yes,
		"Sure thing":         Yes,
		"Why not chess":      Yes,
		"Okay then":          Yes,
		"No I wouldn't":      No,
		"No way":             No,
		"Never mind":         No,
		"Of course not":      Unclear,
		"Yes but not now":    Unclear,
		"OK no":              Unclear,
		"No problem":         Unclear,
		"no doubt":           Unclear,
	}
	for in, want := range cases {
		if got := YesNo(in); got != want {
			t.Errorf("YesNo(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestNegatedBefore(t *testing.T) {
	t.Parallel()
	w := strings.Fields("I DON'T WANT TO PLAY CHESS")
	if !NegatedBefore(w, 4) {
		t.Error("DON'T before PLAY must negate")
	}
	if NegatedBefore(strings.Fields("PLAY CHESS NOT"), 0) {
		t.Error("nothing before index 0")
	}
}

func FuzzClauses(f *testing.F) {
	for _, s := range []string{"Love to. How about Global Thermonuclear War?", "let us play", "’’’", "a,b;c!d?e.", "\x00\xff"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		for _, c := range Clauses(s) {
			if c.Norm == "" || strings.Join(c.Words, " ") != c.Norm {
				t.Fatalf("bad clause %+v from %q", c, s)
			}
			if c.Norm != strings.ToUpper(c.Norm) || strings.ContainsAny(c.Norm, clauseBreaks) {
				t.Fatalf("clause not normalised: %q", c.Norm)
			}
		}
		_, _ = Number(s)
		_ = YesNo(s)
	})
}
