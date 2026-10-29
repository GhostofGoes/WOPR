// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package gtw_test

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/GhostofGoes/WOPR/internal/games/gtw"
	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// menuEntry is a numbered entry of the scenario menu, in one of its 27-column columns:
// " 12. CUBAN PROVOCATION".
var menuEntry = regexp.MustCompile(`^ *(\d+)\. (\S.*?) *$`)

// page is the last menu page a transcript shows: its lines, from the heading to the hint or
// the refusal in its place.
func page(t *testing.T, transcript string) []string {
	t.Helper()
	i := strings.LastIndex(transcript, "SCENARIOS ON FILE: PAGE ")
	if i < 0 {
		t.Fatalf("no menu:\n%s", transcript)
	}
	lines := strings.Split(transcript[i:], "\n")
	for j, l := range lines {
		if strings.HasPrefix(l, "A NUMBER OR A NAME PLAYS ONE.") || strings.HasPrefix(l, "NO SUCH SCENARIO, PROFESSOR.") {
			return lines[:j+1]
		}
	}
	t.Fatalf("the menu has no hint:\n%s", transcript[i:])
	return nil
}

// 3 at the side prompt opens the scenario menu: every scenario, numbered once, over pages
// that fit 80x24 with the front panel, which NEXT and PREV turn round. A number or a name in
// any spacing and punctuation briefs one; anything else is refused on the page put up again,
// which still fits, and Enter or BACK returns to the side choice, which still reads 1 and 2 as
// the sides.
func TestScenarioMenu(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, gtw.New(), info, "", 1)
	g.Type("3")
	names := gtw.Scenarios()
	seen := map[int]bool{}
	for p := range 3 {
		lines := page(t, g.Transcript())
		if want := fmt.Sprintf("SCENARIOS ON FILE: PAGE %d OF 3", p+1); lines[0] != want {
			t.Errorf("page %d is headed %q", p+1, lines[0])
		}
		if len(lines)+1 > 23 { // the prompt's row too
			t.Errorf("page %d takes %d rows:\n%s", p+1, len(lines)+1, strings.Join(lines, "\n"))
		}
		for _, l := range lines {
			if len(l) > 80 {
				t.Errorf("%d columns: %q", len(l), l)
			}
			for c := 0; c < len(l); c += 27 {
				m := menuEntry.FindStringSubmatch(l[c:min(c+27, len(l))])
				if m == nil {
					continue
				}
				n, _ := strconv.Atoi(m[1]) // the pattern's digits
				if n < 1 || n > len(names) || seen[n] || names[n-1] != m[2] {
					t.Errorf("page %d lists %q as %d", p+1, m[2], n)
				}
				seen[n] = true
			}
		}
		if asking, prompt := g.Asking(); !asking || prompt != "CHOOSE A SCENARIO: " {
			t.Errorf("the menu asks %q", prompt)
		}
		g.Type("next")
	}
	if len(seen) != len(names) {
		t.Errorf("the pages list %d of %d scenarios", len(seen), len(names))
	}
	if lines := page(t, g.Transcript()); lines[0] != "SCENARIOS ON FILE: PAGE 1 OF 3" {
		t.Errorf("NEXT on the last page turns to %q", lines[0])
	}
	g.Type("prev")
	if lines := page(t, g.Transcript()); lines[0] != "SCENARIOS ON FILE: PAGE 3 OF 3" {
		t.Errorf("PREV on the first page turns to %q", lines[0])
	}
	for _, wrong := range []string{"157", "nowhere"} {
		g.Type(wrong)
		lines := page(t, g.Transcript())
		if lines[0] != "SCENARIOS ON FILE: PAGE 3 OF 3" || lines[len(lines)-1] != "NO SUCH SCENARIO, PROFESSOR. CHOOSE 1 TO 156, OR A NAME ON FILE." ||
			len(lines)+1 > 23 {
			t.Errorf("%q is refused on the page:\n%s", wrong, strings.Join(lines, "\n"))
		}
	}
	for _, back := range []string{"", "back"} {
		before := strings.Count(g.Transcript(), "WHICH SIDE DO YOU WANT?")
		g.Type(back)
		if strings.Count(g.Transcript(), "WHICH SIDE DO YOU WANT?") != before+1 || !g.Contains("  3.    SCENARIOS") {
			t.Errorf("%q goes back to the side choice:\n%s", back, g.Transcript())
		}
		g.Type("scenarios")
	}
	g.Type("")
	g.Type("2")
	if !g.Contains("AWAITING FIRST STRIKE COMMAND") {
		t.Errorf("2 is still the Soviet Union:\n%s", g.Transcript())
	}

	for in, want := range map[string]string{
		"4":                      "FAR EAST STRATEGY",
		"scenario 4":             "FAR EAST STRATEGY",
		"us first strike":        "U.S. FIRST STRIKE",
		"Nato Warsaw Pact":       "NATO / WARSAW PACT",
		"S. African Offensive":   "S.AFRICAN OFFENSIVE",
		"vietnamese retaliation": "VIETNAMESE RETALIATIO",
		"chad alert":             "CHAD ALERT",
	} {
		g := testkit.Game(t, gtw.New(), info, "", 1)
		g.Type("Scenarios").Type(in)
		if !g.Contains("[CLEAR]\n" + want + "\n" + strings.Repeat("-", len(want)) + "\n") {
			t.Errorf("%q does not brief %s:\n%s", in, want, g.Transcript())
		}
	}
}

// play runs scenario n by itself under Instant on an 80x19 board, as --scenario starts it,
// checking every view on the way: nothing past column 79, and the scenario named in the
// board's title. It returns every line printed.
func play(t *testing.T, n int, seed uint64) ([]string, *proto.Canvas) {
	t.Helper()
	g := gtw.New()
	var lines []string
	record := func(outs []proto.Output) []proto.Output {
		for _, o := range outs {
			switch o := o.(type) {
			case proto.Say:
				lines = append(lines, o.Lines...)
			case proto.Prompt:
				lines = append(lines, "> "+o.Text)
			}
		}
		return outs
	}
	name := gtw.Scenarios()[n-1]
	record(g.Start(proto.Env{Seed: seed, Instant: true, Width: 80, Height: 19, Mode: gtw.ScenarioLaunch(n)}))
	record(g.Handle(proto.LineEvent{Text: ""})) // the briefing's Enter
	c := proto.NewCanvas(100, 19)
	g.View(c)
	board := c
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
	record(g.Handle(proto.LineEvent{Text: ""})) // the kill ratios
	record(g.Handle(proto.LineEvent{Text: ""})) // back to the menu
	return lines, board
}

// Every scenario plays itself: its briefing and every line fit 80 columns, no strike prompt
// is asked, DEFCON falls a rung a stage, the kill ratios end with the montage's verdict, and
// Enter returns to the menu at the scenario's page, never to the climax. Under Instant the
// briefing waits for Enter, since it has no pause to be read in.
func TestEveryScenarioPlaysItself(t *testing.T) {
	t.Parallel()
	for i, name := range gtw.Scenarios() {
		lines, _ := play(t, i+1, uint64(i))
		text := strings.Join(lines, "\n")
		var ladder []string
		for _, l := range lines {
			if len(l) > 80 {
				t.Errorf("%s: %d columns: %q", name, len(l), l)
			}
			if strings.HasPrefix(l, "> STRIKE") {
				t.Errorf("%s: a playback asks for an order: %q", name, l)
			}
			if strings.Contains(l, "DEFCON") && (strings.HasPrefix(l, "ENEMY ") || strings.HasPrefix(l, "FULL-SCALE ")) {
				ladder = append(ladder, l[strings.LastIndex(l, "DEFCON"):])
			}
		}
		if got := strings.Join(ladder, " "); got != "DEFCON 4. DEFCON 3. DEFCON 2. DEFCON 1." {
			t.Errorf("%s: the ladder reads %q", name, got)
		}
		if !strings.Contains(text, "WOPR PLAYS BOTH SIDES.\nPRESS ENTER TO OPEN THE BOARD.\n> \n") {
			t.Errorf("%s: the briefing does not wait for Enter:\n%s", name, text)
		}
		_, after, _ := strings.Cut(text, "PROJECTED KILL RATIOS.\n> \n") // the assessment's Enter
		if !strings.HasPrefix(after, "STRATEGY: "+name+"\nWINNER: NONE\nPRESS ENTER TO CONTINUE.\n") {
			t.Errorf("%s: the kill ratios end with:\n%s", name, after)
		}
		page := fmt.Sprintf("SCENARIOS ON FILE: PAGE %d OF 3", i/52+1)
		if !strings.Contains(after, page) || strings.Contains(text, "GAME ROUTINE RUNNING") {
			t.Errorf("%s: Enter goes to %s, not the climax:\n%s", name, page, after)
		}
		if brief, _, _ := strings.Cut(text, "NO ORDERS ARE TAKEN, PROFESSOR."); strings.Count(brief, "\n") > 12 {
			t.Errorf("%s: the briefing takes %d rows", name, strings.Count(brief, "\n"))
		}
	}
}

// A playback is the same in every session: whatever the seed, an easter egg's included, the
// same lines and the same board.
func TestPlaybackIgnoresTheSeed(t *testing.T) {
	t.Parallel()
	for _, n := range []int{1, 2, 12, 104} {
		a, boardA := play(t, n, 1)
		b, boardB := play(t, n, proto.LaunchSeed(1962, gtw.Slug, 0))
		if strings.Join(a, "\n") != strings.Join(b, "\n") || boardA.String() != boardB.String() {
			t.Errorf("scenario %d differs with the seed:\n%s\n\n%s", n, strings.Join(a, "\n"), strings.Join(b, "\n"))
		}
		if strings.Contains(strings.Join(b, "\n"), "WOPR IS USING DOCTRINE") {
			t.Errorf("scenario %d names an easter egg's doctrine", n)
		}
	}
}

// Paced, a briefing waits a few seconds once it has been read, and then the board opens by
// itself. The war flies on the clock, with no prompt, until the strike assessment.
func TestPlaybackRunsOnTheClock(t *testing.T) {
	t.Parallel()
	g := gtw.New()
	outs := g.Start(proto.Env{Seed: 1, Width: 80, Height: 19, Mode: gtw.ScenarioLaunch(2)})
	if w, ok := outs[len(outs)-2].(proto.Wait); !ok || w.D < 2*time.Second {
		t.Errorf("the briefing ends %#v", outs[len(outs)-2:])
	}
	if _, ok := outs[len(outs)-1].(proto.Drain); !ok {
		t.Errorf("the briefing ends %#v", outs[len(outs)-1])
	}
	prompts := 0
	for _, o := range g.Handle(proto.Drained{}) {
		if a, ok := o.(proto.Animate); ok && a.Every != frameEvery {
			t.Errorf("the board animates every %v", a.Every)
		}
	}
	for range 4 * strikeFrames {
		for _, o := range g.Handle(proto.TickEvent{Dt: frameEvery}) {
			if _, ok := o.(proto.Prompt); ok {
				prompts++
			}
			if s, ok := o.(proto.Say); ok && strings.HasPrefix(strings.Join(s.Lines, "\n"), "STRIKE ASSESSMENT COMPLETE.") && prompts != 0 {
				t.Errorf("%d prompts before the assessment", prompts)
			}
		}
	}
	if prompts != 1 {
		t.Errorf("%d prompts: want the assessment's alone", prompts)
	}
}

// A scenario's whole playback from the side choice, through the menu and back: the Soviet
// Union's first strike, your answer, the kill ratios and the verdict.
func TestScenarioTranscript(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, gtw.New(), info, "", 1983)
	g.Type("3").Type("USSR First Strike").Type("").Type("").Type("").Type("")
	if !g.Contains("WINNER: NONE") || g.Contains("STRIKE 2 OF 3") {
		t.Fatalf("the scenario did not play itself:\n%s", g.Transcript())
	}
	golden.AssertString(t, "scenario", g.Transcript())
}
