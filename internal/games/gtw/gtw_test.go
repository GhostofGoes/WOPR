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
