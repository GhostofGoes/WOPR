package gtw_test

import (
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

var info = games.Info{Name: "GLOBAL THERMONUCLEAR WAR", Slug: gtw.Slug, Layout: proto.LayoutConsole}

// The film's scenario, through to tic-tac-toe (docs/PLAN.md §6.2).
func TestFilmScenario(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, gtw.New(), info, "", 1983)
	g.Type("3").Type("2")
	g.Type("Las Vegas").Type("Seattle, Omaha").Type("")
	if !g.Contains("FIRST STRIKE LAUNCHED.") || !g.Contains("STRIKE ASSESSMENT COMPLETE.") {
		t.Fatalf("the exchange did not run:\n%s", g.Transcript())
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

// Every element of the big board is on the 80x19 view, whole: the title, the DEFCON ladder
// 5..1, the sides' names, the trajectory table and the climax's launch code.
func TestBoardKeepsEveryElement(t *testing.T) {
	t.Parallel()
	g := gtw.New()
	g.Start(proto.Env{Seed: 1, Instant: true})
	for _, in := range []string{"1", "Moscow, Leningrad, Kiev", "", "", ""} {
		g.Handle(proto.LineEvent{Text: in})
	}
	c := proto.NewCanvas(80, 19)
	g.View(c)
	screen := c.String()
	for _, want := range []string{
		"GLOBAL THERMONUCLEAR WAR", "DEFCON", "| 5 |", "| 4 |", "| 3 |", "| 2 |", "| 1 |",
		"UNITED STATES", "SOVIET UNION", "TRAJECTORY HEADING", "A-MM3-A", "E-MM3-A", "LAUNCH CODE: CPE 1704 ___",
	} {
		if !strings.Contains(screen, want) {
			t.Errorf("the board lost %q:\n%s", want, screen)
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
	g.Handle(proto.TickEvent{Dt: 20 * 100 * time.Millisecond})
	snap("exchange, frame 20")
	g.Handle(proto.TickEvent{Dt: time.Minute})
	snap("exchange, done")
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

// climax plays the scenario up to the climax.
func climax(t *testing.T, seed uint64) *testkit.GameSession {
	t.Helper()
	g := testkit.Game(t, gtw.New(), info, "", seed)
	g.Type("2").Type("Las Vegas").Type("").Type("").Type("")
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
	for _, in := range []string{"2", "Las Vegas", "", "", ""} {
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
	for _, in := range []string{"2", "Las Vegas", "", "", ""} {
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
	for range 200 {
		outs = append(outs, g.Handle(proto.TickEvent{Dt: 100 * time.Millisecond})...)
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
