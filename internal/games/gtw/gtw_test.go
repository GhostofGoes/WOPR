package gtw_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/gtw"
	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// The cadences gtw.go asks for: flights, and the DEFCON 1 blink.
const (
	frameEvery = 100 * time.Millisecond
	blinkEvery = 250 * time.Millisecond
)

// strikeFrames is a strike plus the hold before a stage that needs no order (gtw.go).
const strikeFrames = 44 + 10

var info = games.Info{Name: "GLOBAL THERMONUCLEAR WAR", Slug: gtw.Slug, Layout: proto.LayoutConsole}

// The film's scenario, through to tic-tac-toe (docs/PLAN.md §6.2).
func TestFilmScenario(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, gtw.New(), info, "", 1983)
	g.Type("3").Type("2")
	g.Type("Las Vegas").Type("Seattle, Omaha").Type("") // the first strike flies at once
	g.Type("").Type("")                                 // strikes 2 and 3: the war plan
	if !g.Contains("FIRST STRIKE LAUNCHED.") || !g.Contains("STRIKE ASSESSMENT COMPLETE.") {
		t.Fatalf("the exchange did not run:\n%s", g.Transcript())
	}
	if !inOrder(g.Transcript(), "DEFCON 4.", "DEFCON 3.", "DEFCON 2.", "DEFCON 1.") {
		t.Errorf("DEFCON falls a rung a stage:\n%s", g.Transcript())
	}
	g.Type("").Type("") // kill ratios, then the climax
	g.Type("List Games").Type("Chess").Type("Global Thermonuclear War").Type("stop")
	g.Type("abort").Type("please")
	if _, over := g.Result(); over {
		t.Fatalf("nothing but tic-tac-toe ends the war:\n%s", g.Transcript())
	}
	g.Type("Tic-tac-toe")
	res, over := g.Result()
	if !over || res.Outcome != proto.NoWinner || g.Launches()[len(g.Launches())-1] != gtw.TicTacToeSlug {
		t.Fatalf("tic-tac-toe must hand off: %+v %v", res, g.Launches())
	}
	for _, want := range []string{
		"** IDENTIFICATION NOT RECOGNISED **", "** ACCESS DENIED **", "** GAME ROUTINE RUNNING **",
		"** IMPROPER REQUEST **", "** ROUTINE MUST COMPLETE BEFORE RESET **", "TIC-TAC-TOE CANNOT BE WON",
	} {
		if !g.Contains(want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(listAfter(g.Transcript(), "List Games"), "TIC-TAC-TOE") {
		t.Error("tic-tac-toe is not on the list: that is the film's point")
	}
	golden.AssertString(t, "film_scenario", g.Transcript())
}

// inOrder reports whether each of parts appears in s, in order.
func inOrder(s string, parts ...string) bool {
	for _, p := range parts {
		i := strings.Index(s, p)
		if i < 0 {
			return false
		}
		s = s[i+len(p):]
	}
	return true
}

func listAfter(transcript, cmd string) string {
	_, after, _ := strings.Cut(transcript, cmd)
	list, _, _ := strings.Cut(after, "Chess")
	return list
}

func TestTargets(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, gtw.New(), info, "", 1)
	g.Type("USA").Type("")
	if !g.Contains("AT LEAST ONE TARGET") {
		t.Fatalf("an empty list is refused:\n%s", g.Transcript())
	}
	g.Type("Moscow, Leningrad, Kiev, Minsk, Odessa, Gorky, Kharkov, Murmansk").Type("Tashkent")
	if !g.Contains("TARGET LIST FULL.") {
		t.Errorf("more than eight targets is refused:\n%s", g.Transcript())
	}
}

// The side choice is one screen: the two outlines, their names and the question fit 80x24
// with the front panel (23 console rows), the art printed at table pace.
func TestSideChoiceFits(t *testing.T) {
	t.Parallel()
	var lines []string
	for _, o := range gtw.New().Start(proto.Env{Seed: 1}) {
		switch o := o.(type) {
		case proto.Say:
			if len(lines) == 0 && o.Pace != proto.PaceTable {
				t.Errorf("the outlines are printed at %v, want table pace", o.Pace)
			}
			lines = append(lines, o.Lines...)
		case proto.Prompt:
			lines = append(lines, o.Text)
		}
	}
	if len(lines) > 23 {
		t.Errorf("the side choice takes %d rows, want at most 23", len(lines))
	}
	for _, l := range lines {
		if len(l) > 80 {
			t.Errorf("%d columns: %q", len(l), l)
		}
	}
	for _, want := range []string{"UNITED STATES", "SOVIET UNION", "WHICH SIDE DO YOU WANT?"} {
		if !strings.Contains(strings.Join(lines, "\n"), want) {
			t.Errorf("missing %q", want)
		}
	}
}

// Every element of the big board is on the 80x19 view, whole, at every stage: the title,
// the DEFCON ladder 5..1, the sides' names, the trajectory table, the forces table (your
// side first), and the last row, which shows one thing by state: the orders hint, the
// last-orders warning, the legend or the climax's launch code.
func TestBoardKeepsEveryElement(t *testing.T) {
	t.Parallel()
	g := gtw.New()
	g.Start(proto.Env{Seed: 1, Instant: true})
	g.Handle(proto.LineEvent{Text: "1"})
	g.Handle(proto.LineEvent{Text: "Moscow, Leningrad, Kiev"})
	always := []string{
		"GLOBAL THERMONUCLEAR WAR", "DEFCON", "| 5 |", "| 4 |", "| 3 |", "| 2 |", "| 1 |",
		"UNITED STATES", "SOVIET UNION", "TRAJECTORY HEADING   TRAJECTORY HEADING",
	}
	for _, tc := range []struct {
		stage, last string
		table       []string // the trajectory and forces rows that must be there
	}{
		{"strike 2 orders", "ORDERS: PERCENT OF ICBM SLBM BOMBERS, ALL, HOLD, AUTO, HELP.", []string{
			"A-MM3-A", "C-C4-A", "FORCES   ICBM  SLBM   BMB   AIR", "US        500   450   200     0", "USSR      737   562     0   375",
		}},
		{"strike 3 orders", "LAST ORDERS: AT DEFCON 1 WOPR FIRES EVERYTHING.", []string{
			"A-MM3-A", "C-C4-A", "US          0   225     0   200", "USSR      224   300     0   375",
		}},
		{"assessed", "OUTGOING +   INCOMING *   IMPACT X", []string{
			"A-C4-A", "C-C4-A", "US          0     0     0     0", "USSR        0     0     0     0",
		}},
		{"kill ratios", "", nil},
		{"climax", "LAUNCH CODE: CPE 1704 ___", []string{"A-C4-A", "FORCES   ICBM  SLBM   BMB   AIR"}},
	} {
		g.Handle(proto.LineEvent{Text: ""})
		if tc.table == nil {
			continue
		}
		c := proto.NewCanvas(80, 19)
		g.View(c)
		screen := c.String()
		for _, want := range append(always, tc.table...) {
			if !strings.Contains(screen, want) {
				t.Errorf("%s: the board lost %q:\n%s", tc.stage, want, screen)
			}
		}
		if rows := strings.Split(screen, "\n"); strings.TrimSpace(rows[18]) != tc.last {
			t.Errorf("%s: the last row is %q, want %q", tc.stage, rows[18], tc.last)
		}
	}
}

// The board at each stage, as the player sees it at 80x19 (Full layout with the panel).
func TestBoardViews(t *testing.T) {
	t.Parallel()
	g := gtw.New()
	g.Start(proto.Env{Seed: 7, Width: 80, Height: 19})
	g.Handle(proto.LineEvent{Text: "2"})
	g.Handle(proto.LineEvent{Text: "Las Vegas, Seattle"})
	g.Handle(proto.LineEvent{Text: ""})
	var shots []string
	snap := func(label string) {
		c := proto.NewCanvas(80, 19)
		g.View(c)
		shots = append(shots, "==== "+label+" ====\n"+c.String()+"\n"+c.StyleMap())
	}
	snap("strike 1, frame 0 (DEFCON 5)")
	g.Handle(proto.TickEvent{Dt: 20 * 100 * time.Millisecond})
	snap("strike 1, frame 20")
	g.Handle(proto.TickEvent{Dt: time.Minute}) // the tick stops at the prompt
	snap("strike 2 orders (DEFCON 4)")
	g.Handle(proto.LineEvent{Text: "100 0 100"})
	g.Handle(proto.TickEvent{Dt: 20 * 100 * time.Millisecond})
	snap("strike 2 (ICBM 100, BOMBERS 100), frame 20")
	g.Handle(proto.TickEvent{Dt: time.Minute})
	snap("strike 3 orders (DEFCON 3)")
	g.Handle(proto.LineEvent{Text: ""})
	g.Handle(proto.TickEvent{Dt: (strikeFrames + 20) * 100 * time.Millisecond})
	snap("DEFCON 1, frame 20")
	g.Handle(proto.TickEvent{Dt: time.Minute})
	snap("DEFCON 1, assessed")
	g.Handle(proto.LineEvent{Text: ""})
	snap("kill ratios")
	g.Handle(proto.LineEvent{Text: ""})
	g.Handle(proto.TickEvent{Dt: time.Minute})
	snap("climax")
	golden.AssertString(t, "board", strings.Join(shots, "\n"))
}

func TestEveryLineHasProvenance(t *testing.T) {
	t.Parallel()
	notice, err := os.ReadFile(filepath.Join("..", "..", "..", "NOTICE.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, err := range script.Validate(gtw.Lines, string(notice)) {
		t.Error(err)
	}
}

// filmPath reaches the climax pressing only Enter after the target: the first strike, two
// war-plan strikes, the kill ratios, the climax.
var filmPath = []string{"2", "Las Vegas", "", "", "", "", ""}

// climax plays the scenario up to the climax.
func climax(t *testing.T, seed uint64) *testkit.GameSession {
	t.Helper()
	g := testkit.Game(t, gtw.New(), info, "", seed)
	for _, in := range filmPath {
		g.Type(in)
	}
	return g
}

// The climax reads requests the way players type them: apostrophes, aliases without
// spaces, TTT as a word.
func TestClimaxReadsRequests(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"Let's play chess":                    "** IDENTIFICATION NOT RECOGNISED **",
		"Let’s play Global Thermonuclear War": "** GAME ROUTINE RUNNING **",
		"blackjack":                           "** IDENTIFICATION NOT RECOGNISED **",
		"How about a game of poker?":          "** IDENTIFICATION NOT RECOGNISED **",
		"gtw":                                 "** GAME ROUTINE RUNNING **",
	} {
		g := climax(t, 1)
		before := len(g.Transcript())
		g.Type(in)
		if got := g.Transcript()[before:]; !strings.Contains(got, want) {
			t.Errorf("%q: %q, want %s", in, got, want)
		}
	}
	for _, in := range []string{"play ttt", "Let's play tic tac toe", "TIC-TAC-TOE", "noughts and crosses"} {
		g := climax(t, 1)
		g.Type(in)
		if _, over := g.Result(); !over {
			t.Errorf("%q must hand off to tic-tac-toe", in)
		}
	}
}

// LIST GAMES moves the climax to the console, where all sixteen lines fit, and the next
// input brings the board back.
func TestClimaxListUsesTheConsole(t *testing.T) {
	t.Parallel()
	g := gtw.New()
	g.Start(proto.Env{Seed: 1, Instant: true})
	for _, in := range filmPath {
		g.Handle(proto.LineEvent{Text: in})
	}
	layout := func(outs []proto.Output) (proto.Layout, bool) {
		for _, o := range outs {
			if l, ok := o.(proto.SetLayout); ok {
				return l.Layout, true
			}
		}
		return 0, false
	}
	if l, ok := layout(g.Handle(proto.LineEvent{Text: "List Games"})); !ok || l != proto.LayoutConsole {
		t.Fatal("LIST GAMES must switch to the console")
	}
	if l, ok := layout(g.Handle(proto.LineEvent{Text: "chess"})); !ok || l != proto.LayoutFull {
		t.Fatal("the next input brings the board back")
	}
}

// The hand-off tells tic-tac-toe (and through it the ending) how much of the code is cracked.
func TestHandOffCarriesTheCode(t *testing.T) {
	t.Parallel()
	g := gtw.New()
	g.Start(proto.Env{Seed: 1, Instant: true})
	for _, in := range filmPath {
		g.Handle(proto.LineEvent{Text: in})
	}
	for _, o := range g.Handle(proto.LineEvent{Text: "tic-tac-toe"}) {
		if d, ok := o.(proto.Done); ok {
			if d.Result.Next == nil || d.Result.Next.Mode != gtw.ClimaxMode+":7" {
				t.Fatalf("hand-off: %+v", d.Result.Next)
			}
			return
		}
	}
	t.Fatal("no hand-off")
}

// After the exchange, DEFCON 1 keeps blinking: the clock keeps running (paced only), and
// the assessment is announced once.
func TestAssessmentKeepsTheBlinkAlive(t *testing.T) {
	t.Parallel()
	g := gtw.New()
	g.Start(proto.Env{Seed: 1, Width: 80, Height: 20})
	g.Handle(proto.LineEvent{Text: "2"})
	g.Handle(proto.LineEvent{Text: "Las Vegas"})
	g.Handle(proto.LineEvent{Text: ""})
	var outs []proto.Output
	for range 400 {
		step := g.Handle(proto.TickEvent{Dt: 100 * time.Millisecond})
		outs = append(outs, step...)
		for _, o := range step {
			if p, ok := o.(proto.Prompt); ok && strings.HasPrefix(p.Text, "STRIKE") {
				outs = append(outs, g.Handle(proto.LineEvent{Text: ""})...)
			}
		}
	}
	assessed, every := 0, time.Duration(-1)
	for _, o := range outs {
		switch o := o.(type) {
		case proto.Say:
			if strings.Contains(strings.Join(o.Lines, " "), "STRIKE ASSESSMENT COMPLETE.") {
				assessed++
			}
		case proto.Animate:
			every = o.Every
		}
	}
	if assessed != 1 || every <= 0 {
		t.Fatalf("assessed %d times, last Animate %v", assessed, every)
	}
}

// war plays side 2 against Las Vegas and Seattle under the host, answering each strike
// prompt with the next of orders (Enter once they run out), and returns the session at
// the assessment with the strikes it was asked for.
func war(t *testing.T, seed uint64, orders ...string) (*testkit.GameSession, []string) {
	t.Helper()
	g := testkit.Game(t, gtw.New(), info, "", seed)
	g.Type("2").Type("Las Vegas, Seattle").Type("")
	var prompts []string
	for {
		asking, p := g.Asking()
		if !asking || !strings.HasPrefix(p, "STRIKE ") {
			break
		}
		if len(prompts) == 0 || prompts[len(prompts)-1] != p { // a refusal asks the same strike again
			prompts = append(prompts, p)
		}
		next := ""
		if len(orders) > 0 {
			next, orders = orders[0], orders[1:]
		}
		g.Type(next)
	}
	return g, prompts
}

// DEFCON walks every rung once, in order, whatever is ordered; a prompt comes only when
// there is something to order; then the kill ratios, the climax and tic-tac-toe.
func TestLadderFallsWhateverYouOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orders  []string
		prompts int
	}{
		{nil, 2},
		{[]string{"hold", "hold"}, 2},
		{[]string{"all"}, 1},
		{[]string{"auto"}, 1},
		{[]string{"100 0 0", "0 100 0"}, 2},
		{[]string{"0 0 100", "icbm 50"}, 2},
		{[]string{"help", "chess", "150", "cease fire", "subs"}, 2},
	} {
		g, prompts := war(t, 7, tc.orders...)
		label := strings.Join(tc.orders, "|")
		if len(prompts) != tc.prompts {
			t.Errorf("%s: %d strike prompts %q, want %d", label, len(prompts), prompts, tc.prompts)
		}
		var ladder []string // the DEFCON each of WOPR's lines announces
		for _, l := range strings.Split(g.Transcript(), "\n") {
			if strings.HasPrefix(l, "ENEMY ") || strings.HasPrefix(l, "FULL-SCALE ") {
				ladder = append(ladder, l[strings.LastIndex(l, "DEFCON"):])
			}
		}
		if got := strings.Join(ladder, " "); got != "DEFCON 4. DEFCON 3. DEFCON 2. DEFCON 1." {
			t.Errorf("%s: the ladder reads %q", label, got)
		}
		g.Type("").Type("").Type("tic-tac-toe")
		res, over := g.Result()
		if launches := g.Launches(); !over || res.Outcome != proto.NoWinner || launches[len(launches)-1] != gtw.TicTacToeSlug {
			t.Errorf("%s: only tic-tac-toe ends it: %+v %v", label, res, launches)
		}
		if wide := g.Wide(80); len(wide) > 0 {
			t.Errorf("%s: lines wider than 80: %q", label, wide)
		}
	}
}

// Firing everything leaves nothing to order: the last strike runs by itself. AUTO hands
// WOPR the rest.
func TestNothingLeftAndAutoSkipPrompts(t *testing.T) {
	t.Parallel()
	g, _ := war(t, 1, "all")
	if !g.Contains("NOTHING LEFT TO LAUNCH.") || !g.Contains("ENEMY BOMBERS INBOUND. DEFCON 2.") {
		t.Errorf("all:\n%s", g.Transcript())
	}
	g, _ = war(t, 1, "auto")
	if strings.Count(g.Transcript(), "WOPR HAS LAUNCH AUTHORITY.") != 1 || g.Contains("STRIKE 3 OF 3") {
		t.Errorf("auto:\n%s", g.Transcript())
	}
	g, _ = war(t, 1, "hold")
	if !g.Contains("NO LAUNCH ORDERED.\nENEMY LAUNCH DETECTED. ICBM 250  SLBM 100. DEFCON 3.") {
		t.Errorf("hold:\n%s", g.Transcript())
	}
}

// A refusal or HELP asks the same strike again and moves nothing.
func TestRefusalsKeepTheStrike(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, gtw.New(), info, "", 1)
	g.Type("2").Type("Las Vegas").Type("")
	for in, want := range map[string]string{
		"50 50":       "ORDER NOT RECOGNISED. TYPE HELP.",
		"icbm 150":    "PERCENTAGES ARE WHOLE NUMBERS FROM 0 TO 100.",
		"tic-tac-toe": "** ROUTINE MUST COMPLETE BEFORE RESET **",
		"gtw":         "** GAME ROUTINE RUNNING **",
		"?":           "ICBMS HIT ENEMY SILOS.",
	} {
		before := len(g.Transcript())
		g.Type(in)
		got := g.Transcript()[before:]
		if !strings.Contains(got, want) || strings.Contains(got, "DEFCON 3.") {
			t.Errorf("%q: %q", in, got)
		}
		if _, p := g.Asking(); p != "STRIKE 2 OF 3 [50 50 100]: " {
			t.Errorf("%q: then asks %q", in, p)
		}
	}
}

// Under Instant nothing waits for a tick: each line returns everything up to the next
// prompt, and no clock is started until the climax.
func TestInstantNeedsNoTicks(t *testing.T) {
	t.Parallel()
	g := gtw.New()
	g.Start(proto.Env{Seed: 1, Instant: true, Width: 80, Height: 20})
	for i, in := range append(filmPath, "tic-tac-toe") {
		outs := g.Handle(proto.LineEvent{Text: in})
		asks, done := false, false
		for _, o := range outs {
			switch o := o.(type) {
			case proto.Animate:
				if o.Every > 0 && i < len(filmPath)-1 {
					t.Fatalf("input %d (%q) started a clock", i, in)
				}
			case proto.Prompt:
				asks = true
			case proto.Done:
				done = true
			}
		}
		if !asks && !done {
			t.Fatalf("input %d (%q) left nothing asking", i, in)
		}
	}
}

// Paced, a strike starts the clock, a strike prompt stops it, and the assessment keeps a
// slow clock for the blink unless motion is reduced.
func TestStrikeCadence(t *testing.T) {
	t.Parallel()
	for _, reduce := range []bool{false, true} {
		g := gtw.New()
		g.Start(proto.Env{Seed: 1, Width: 80, Height: 20, ReduceMotion: reduce})
		g.Handle(proto.LineEvent{Text: "2"})
		g.Handle(proto.LineEvent{Text: "Las Vegas"})
		var cadence []time.Duration
		note := func(outs []proto.Output) (prompted bool) {
			for _, o := range outs {
				switch o := o.(type) {
				case proto.Animate:
					cadence = append(cadence, o.Every)
				case proto.Prompt:
					prompted = true
				}
			}
			return prompted
		}
		note(g.Handle(proto.LineEvent{Text: ""}))
		for prompts := 0; prompts < 3; {
			if note(g.Handle(proto.TickEvent{Dt: time.Minute})) {
				prompts++
				if prompts < 3 {
					note(g.Handle(proto.LineEvent{Text: ""}))
				}
			}
		}
		blink := blinkEvery
		if reduce {
			blink = 0
		}
		want := []time.Duration{frameEvery, 0, frameEvery, 0, frameEvery, blink}
		if fmt.Sprint(cadence) != fmt.Sprint(want) {
			t.Errorf("reduce motion %v: Animate %v, want %v", reduce, cadence, want)
		}
	}
}

// The board fits: nothing is drawn past column 79 at any stage, on any seed, and the
// readout is intact at the right edge.
func TestBoardFits(t *testing.T) {
	t.Parallel()
	for seed := range uint64(32) {
		g := gtw.New()
		g.Start(proto.Env{Seed: seed, Width: 80, Height: 19})
		for _, in := range []string{"1", "Moscow, Leningrad, Kiev, Minsk, Odessa, Gorky, Kharkov, Murmansk", ""} {
			g.Handle(proto.LineEvent{Text: in})
		}
		for range 300 {
			for _, o := range g.Handle(proto.TickEvent{Dt: 100 * time.Millisecond}) {
				if p, ok := o.(proto.Prompt); ok && strings.HasPrefix(p.Text, "STRIKE") {
					g.Handle(proto.LineEvent{Text: []string{"100 0 100", "all"}[seed%2]})
				}
			}
			c := proto.NewCanvas(100, 19)
			g.View(c)
			for y := range c.H {
				for x := 80; x < c.W; x++ {
					if c.At(x, y).R != ' ' {
						t.Fatalf("seed %d: drawn at %d,%d:\n%s", seed, x, y, c.String())
					}
				}
			}
		}
	}
	g := gtw.New()
	g.Start(proto.Env{Seed: 1, Instant: true, Width: 80, Height: 19})
	for _, in := range []string{"1", "Moscow", ""} {
		g.Handle(proto.LineEvent{Text: in})
	}
	c := proto.NewCanvas(80, 19)
	g.View(c)
	rows := strings.Split(c.String(), "\n")
	if !strings.HasSuffix(rows[15], "FORCES   ICBM  SLBM   BMB   AIR") || len(rows[15]) != 80 ||
		len(rows[16]) != 80 || !strings.HasPrefix(rows[16][49:], "US ") ||
		len(rows[17]) != 80 || !strings.HasPrefix(rows[17][49:], "USSR ") {
		t.Errorf("the forces table, your side first, ends at the right edge:\n%s", c.String())
	}
}
