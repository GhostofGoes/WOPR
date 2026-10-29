// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package gtw_test

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/assets"
	"github.com/GhostofGoes/WOPR/internal/games/gtw"
	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// listed is a numbered entry of the scenario list: "  1.  U.S. FIRST STRIKE".
var listed = regexp.MustCompile(`(\d+)\.  (\S+(?: \S+)*)`)

// scenarioNames is the scenario list as the player sees it, in order.
func scenarioNames(t *testing.T) []string {
	t.Helper()
	g := testkit.Game(t, gtw.New(), info, "", 1)
	g.Type("scenarios")
	_, list, ok := strings.Cut(g.Transcript(), "SCENARIOS ON FILE:")
	if !ok {
		t.Fatalf("SCENARIOS lists nothing:\n%s", g.Transcript())
	}
	found := listed.FindAllStringSubmatch(list, -1)
	names := make([]string, len(found))
	for _, m := range found { // down the first column, then the second
		n, _ := strconv.Atoi(m[1])
		if n < 1 || n > len(names) || names[n-1] != "" {
			t.Fatalf("the list numbers %s twice, or out of 1 to %d:\n%s", m[1], len(names), list)
		}
		names[n-1] = m[2]
	}
	return names
}

// SCENARIOS lists the montage's strategies GTW can play, numbered, on one screen; the
// scenario prompt takes a number or a name, refuses anything else, and Enter goes back to
// the side choice, which still reads 1 and 2 as the sides.
func TestScenarioList(t *testing.T) {
	t.Parallel()
	names := scenarioNames(t)
	if len(names) < 10 {
		t.Fatalf("%d scenarios: %q", len(names), names)
	}
	montage := assets.Scenarios.Texts()
	for _, n := range names {
		if !slices.Contains(montage, n) {
			t.Errorf("%q is not a montage name", n)
		}
	}
	for _, want := range []string{"U.S. FIRST STRIKE", "USSR FIRST STRIKE", "NATO / WARSAW PACT", "FAR EAST STRATEGY"} {
		if !slices.Contains(names, want) {
			t.Errorf("the list lacks %s", want)
		}
	}

	g := testkit.Game(t, gtw.New(), info, "", 1)
	g.Type("3")
	if !g.Contains("CHOOSE 1 OR 2, OR TYPE SCENARIOS.") {
		t.Errorf("a wrong side names the scenarios:\n%s", g.Transcript())
	}
	before := g.Transcript()
	g.Type("list scenarios")
	out := strings.TrimPrefix(g.Transcript(), before)
	if rows := strings.Count(out, "\n"); rows > 23 {
		t.Errorf("the list takes %d rows:\n%s", rows, out)
	}
	if wide := g.Wide(80); len(wide) > 0 {
		t.Errorf("lines wider than 80: %q", wide)
	}
	if asking, p := g.Asking(); !asking || p != "CHOOSE A SCENARIO: " {
		t.Errorf("the list asks %q", p)
	}
	g.Type("99")
	if !g.Contains(fmt.Sprintf("CHOOSE 1 TO %d, PROFESSOR.", len(names))) {
		t.Errorf("99 is refused:\n%s", g.Transcript())
	}
	g.Type("")
	if strings.Count(g.Transcript(), "WHICH SIDE DO YOU WANT?") != 2 {
		t.Errorf("Enter goes back to the side choice:\n%s", g.Transcript())
	}
	g.Type("2")
	if !g.Contains("AWAITING FIRST STRIKE COMMAND") {
		t.Errorf("2 is still the Soviet Union:\n%s", g.Transcript())
	}
}

// A scenario is named by its number or its name, at the side prompt or the list's, in any
// spacing and punctuation; the briefing then says what is different, and SCENARIOS goes back
// to the list.
func TestScenarioByName(t *testing.T) {
	t.Parallel()
	for _, in := range [][]string{
		{"Far East Strategy"},
		{"scenario far east strategy"},
		{"scenario 4"},
		{"scenarios", "4"},
		{"scenarios", "FAR-EAST STRATEGY"},
		{"scenario nato warsaw pact", "scenarios", "scenario 4"},
	} {
		g := testkit.Game(t, gtw.New(), info, "", 1)
		for _, line := range in {
			g.Type(line)
		}
		_, brief, ok := strings.Cut(g.Transcript(), "FAR EAST STRATEGY\n-----------------")
		for _, want := range []string{
			"YOUR SIDE: UNITED STATES.", "YOUR FORCES:  ICBM 1000  SLBM 600  BOMBERS 300.",
			"ENEMY FORCES: ICBM 1250  SLBM 750  BOMBERS 375.", "WAR PLAN: 10 25 50, THEN 50 50 100, THEN 100 100 100.",
			"TARGETS: VLADIVOSTOK, KHABAROVSK,", "PRESS ENTER TO BEGIN.",
		} {
			if !ok || !strings.Contains(brief, want) {
				t.Errorf("%q: the briefing lacks %q:\n%s", in, want, g.Transcript())
			}
		}
	}
	g := testkit.Game(t, gtw.New(), info, "", 1)
	g.Type("scenario 404")
	if !g.Contains("SCENARIOS ON FILE:") {
		t.Errorf("an unknown scenario shows the list:\n%s", g.Transcript())
	}
}

// Every scenario plays the whole war on its own terms: the briefing fits, the board carries
// the scenario's name and fits 80x19, DEFCON falls a rung a stage, Enter at every prompt
// reaches the assessment, and only tic-tac-toe ends it.
func TestEveryScenarioPlays(t *testing.T) {
	t.Parallel()
	for i, name := range scenarioNames(t) {
		g := gtw.New()
		g.Start(proto.Env{Seed: uint64(i), Instant: true, Width: 80, Height: 19})
		var lines []string
		handle := func(in string) []proto.Output {
			outs := g.Handle(proto.LineEvent{Text: in})
			for _, o := range outs {
				if s, ok := o.(proto.Say); ok {
					lines = append(lines, s.Lines...)
				}
			}
			return outs
		}
		handle(fmt.Sprintf("scenario %d", i+1))
		if len(lines) > 22 || !slices.Contains(lines, name) {
			t.Errorf("%s: the briefing takes %d rows:\n%s", name, len(lines), strings.Join(lines, "\n"))
		}
		handle("")
		c := proto.NewCanvas(100, 19)
		g.View(c)
		if !strings.Contains(c.String(), " GLOBAL THERMONUCLEAR WAR: "+name+" ") {
			t.Errorf("%s: the board's title:\n%s", name, c.String())
		}
		for y := range c.H {
			for x := 80; x < c.W; x++ {
				if c.At(x, y).R != ' ' {
					t.Fatalf("%s: drawn at %d,%d:\n%s", name, x, y, c.String())
				}
			}
		}
		for range 3 {
			handle("") // the strikes, then the kill ratios
		}
		var ladder []string
		for _, l := range lines {
			if len(l) > 80 {
				t.Errorf("%s: %d columns: %q", name, len(l), l)
			}
			if (strings.HasPrefix(l, "ENEMY ") || strings.HasPrefix(l, "FULL-SCALE ")) && strings.Contains(l, "DEFCON") {
				ladder = append(ladder, l[strings.LastIndex(l, "DEFCON"):])
			}
		}
		if got := strings.Join(ladder, " "); got != "DEFCON 4. DEFCON 3. DEFCON 2. DEFCON 1." {
			t.Errorf("%s: the ladder reads %q", name, got)
		}
		handle("")
		var res proto.Result
		for _, o := range handle("tic-tac-toe") {
			if d, ok := o.(proto.Done); ok {
				res = d.Result
			}
		}
		if res.Next == nil || res.Next.Slug != gtw.TicTacToeSlug {
			t.Errorf("%s: only tic-tac-toe ends it: %+v\n%s", name, res, strings.Join(lines, "\n"))
		}
	}
}

// When WOPR strikes first its tracks leave first, and yours (if any) launch on warning: the
// strip says WOPR's launch before yours, and the first prompt is the answer at strike 2.
func TestWOPRStrikesFirst(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, gtw.New(), info, "", 1)
	g.Type("USSR first strike").Type("")
	if !g.Contains("ENEMY LAUNCH DETECTED. ICBM 500  SLBM 150  BOMBERS 375. DEFCON 4.\nNO LAUNCH ORDERED.\nLOST ON THE GROUND:") {
		t.Errorf("USSR FIRST STRIKE:\n%s", g.Transcript())
	}
	if asking, p := g.Asking(); !asking || p != "STRIKE 2 OF 3 [50 50 100]: " {
		t.Errorf("the first prompt is %q", p)
	}
	g = testkit.Game(t, gtw.New(), info, "", 1)
	g.Type("Cuban Provocation").Type("")
	if !g.Contains("ENEMY LAUNCH DETECTED. SLBM 120. DEFCON 4.\nLAUNCH ON WARNING. ICBM 250  SLBM 150.\n") {
		t.Errorf("CUBAN PROVOCATION:\n%s", g.Transcript())
	}

	// Paced, WOPR's tracks are in the air before yours.
	b := gtw.New()
	b.Start(proto.Env{Seed: 1, Width: 80, Height: 19})
	b.Handle(proto.LineEvent{Text: "cuban provocation"})
	b.Handle(proto.LineEvent{Text: ""})
	// tracks is the map box's interior, where the tracks fly.
	tracks := func() string {
		c := proto.NewCanvas(80, 19)
		b.View(c)
		var rows []string
		for _, row := range strings.Split(c.String(), "\n")[1:14] {
			rows = append(rows, row[1:72])
		}
		return strings.Join(rows, "\n")
	}
	b.Handle(proto.TickEvent{Dt: 4 * frameEvery})
	if m := tracks(); !strings.Contains(m, "*") || strings.Contains(m, "+") {
		t.Errorf("at frame 4 only WOPR's tracks fly:\n%s", m)
	}
	b.Handle(proto.TickEvent{Dt: 10 * frameEvery})
	if m := tracks(); !strings.Contains(m, "+") {
		t.Errorf("by frame 14 yours fly too:\n%s", m)
	}
}

// A scenario's whole war with Enter at every prompt, through to tic-tac-toe: WOPR's first
// strike, your answer, and the climax.
func TestScenarioTranscript(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, gtw.New(), info, "", 1983)
	g.Type("scenarios").Type("2").Type("").Type("").Type("").Type("").Type("").Type("Tic-tac-toe")
	if _, over := g.Result(); !over {
		t.Fatalf("the scenario did not reach tic-tac-toe:\n%s", g.Transcript())
	}
	golden.AssertString(t, "scenario", g.Transcript())
}
