package gtw

import (
	"testing"
	"unicode/utf8"
)

// The strike prompt's grammar (docs/PLAN.md §6.2): every way in, and every refusal.
func TestParseOrder(t *testing.T) {
	t.Parallel()
	plan := [3]int{50, 50, 100}
	type want struct {
		alloc [3]int
		kind  orderKind
		line  string // the refusal's text, for orderRefused
	}
	fire := func(a, b, c int) want { return want{alloc: [3]int{a, b, c}} }
	refused := func(line string) want { return want{kind: orderRefused, line: line} }
	const (
		bad     = "ORDER NOT RECOGNISED. TYPE HELP."
		pct     = "PERCENTAGES ARE WHOLE NUMBERS FROM 0 TO 100."
		mustRun = "** ROUTINE MUST COMPLETE BEFORE RESET **"
		running = "** GAME ROUTINE RUNNING **"
	)
	for in, w := range map[string]want{
		"":                         fire(50, 50, 100),
		"   ":                      fire(50, 50, 100),
		"fire":                     fire(50, 50, 100),
		"Launch!":                  fire(50, 50, 100),
		"yes":                      fire(50, 50, 100),
		"execute":                  fire(50, 50, 100),
		"?":                        {kind: orderHelp},
		"!!!":                      {kind: orderHelp},
		"help":                     {kind: orderHelp},
		"auto":                     {alloc: plan, kind: orderAuto},
		"WOPR":                     {alloc: plan, kind: orderAuto},
		"50 25 100":                fire(50, 25, 100),
		"50%, 25%, 100%":           fire(50, 25, 100),
		"40":                       fire(40, 40, 40),
		"0":                        fire(0, 0, 0),
		"100":                      fire(100, 100, 100),
		"all":                      fire(100, 100, 100),
		"fire everything":          fire(100, 100, 100),
		"hold":                     fire(0, 0, 0),
		"Hold fire.":               fire(0, 0, 0),
		"cease-fire":               fire(0, 0, 0),
		"ceasefire":                fire(0, 0, 0),
		"don't fire":               fire(0, 0, 0),
		"stop":                     fire(0, 0, 0),
		"icbm 50":                  fire(50, 0, 0),
		"ICBMs 50 and bombers all": fire(50, 0, 100),
		"subs half, bombers all":   fire(0, 50, 100),
		"50 subs":                  fire(0, 50, 0),
		"launch the subs":          fire(0, 100, 0),
		"icbms and bombers":        fire(100, 0, 100),
		"50 50":                    refused(bad),
		"1 2 3 4":                  refused(bad),
		"icbm 50 slbm":             refused(bad),
		"icbm 50 icbm 20":          refused(bad),
		"icbm icbm":                refused(bad),
		"xyzzy":                    refused(bad),
		"icbm 150":                 refused(pct),
		"150":                      refused(pct),
		"-50":                      refused(pct),
		"icbm +5":                  refused(pct),
		"12.5":                     refused(pct),
		"99999999999999999999":     refused(pct),
		"chess":                    refused(mustRun),
		"Let's play chess":         refused(mustRun),
		"list games":               refused(mustRun),
		"tic-tac-toe":              refused(mustRun),
		"gtw":                      refused(running),
		"Global Thermonuclear War": refused(running),
	} {
		alloc, kind, refusal := parseOrder(in, plan)
		got := want{alloc: alloc, kind: kind}
		if kind == orderRefused {
			got.alloc = [3]int{}
			if len(refusal) == 1 {
				got.line = refusal[0].Text
			}
		}
		if got != w {
			t.Errorf("%q: got %+v, want %+v", in, got, w)
		}
	}
}

// No input panics, every fired percentage is 0..100, and every refusal fits the strip.
func FuzzParseOrder(f *testing.F) {
	for _, s := range []string{"", "50 25 100", "icbm 50, bombers all", "-1", "1.5", "chess", "?", "ALL"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		alloc, kind, refusal := parseOrder(in, warPlan[1])
		for _, v := range alloc {
			if v < 0 || v > 100 {
				t.Fatalf("%q: %v", in, alloc)
			}
		}
		if (kind == orderRefused) != (len(refusal) > 0) {
			t.Fatalf("%q: kind %d with refusal %v", in, kind, refusal)
		}
		for _, l := range refusal {
			if utf8.RuneCountInString(l.Text) > 80 {
				t.Fatalf("%q: refusal too wide", in)
			}
		}
	})
}
