package wopr

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/catalog"
	"github.com/GhostofGoes/WOPR/internal/games/gamestest"
	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/proto/host"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// start runs the persona under the host with reg, resolving launches from it.
func start(t *testing.T, reg *games.Registry, opts Options) *testkit.Session {
	t.Helper()
	cfg := host.Config{
		Seed: 1,
		Resolve: func(l proto.Launch) (proto.Program, host.Placement, error) {
			e, ok := reg.Get(l.Slug)
			if !ok || e.New == nil {
				return nil, host.Placement{}, games.ErrNotFound
			}
			return e.New(), host.Placement{Layout: e.Info.Layout, PanelRows: e.Info.PanelRows}, nil
		},
	}
	return testkit.Start(t, New(reg, nil, opts), host.Placement{}, cfg)
}

func TestFilmGreetingPath(t *testing.T) {
	t.Parallel()
	s := start(t, catalog.Registry(), Options{})
	s.Type("Joshua").
		Type("Hello.").
		Type("I'm fine. How are you?").
		Type("People sometimes make mistakes.").
		Type("Love to. How about Global Thermonuclear War?").
		Type("Later. Let's play Global Thermonuclear War.").
		Type("LOGOFF")
	if !s.Exited() {
		t.Error("LOGOFF must end the session")
	}
	golden.AssertString(t, "film_greeting", s.Transcript())
	for _, want := range []string{"GREETINGS PROFESSOR FALKEN.", "WOULDN'T YOU PREFER A GOOD GAME OF CHESS?", "FINE."} {
		if !s.Contains(want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestLogonFailuresAndHint(t *testing.T) {
	t.Parallel()
	s := start(t, catalog.Registry(), Options{})
	s.Type("000001").Type("Falkens-Maze").Type("Armageddon")
	golden.AssertString(t, "logon_fail_x3_hint", s.Transcript())
	if strings.Count(s.Transcript(), "IDENTIFICATION NOT RECOGNIZED BY SYSTEM") != 3 || !s.Contains("FALKEN'S SON") {
		t.Error("three terminations then a hint expected")
	}
	if s.Runner().Depth() != 1 {
		t.Error("a game name at LOGON must not start a game")
	}
}

func TestLogonCommandsDoNotArmSelection(t *testing.T) {
	t.Parallel()
	s := start(t, catalog.Registry(), Options{})
	s.Type("Help Logon").Type("Help Games").Type("List Games").Type("7")
	golden.AssertString(t, "logon_list_then_number", s.Transcript())
	if !s.Contains("HELP NOT AVAILABLE") || !s.Contains("TACTICAL AND STRATEGIC") || !s.Contains("GLOBAL THERMONUCLEAR WAR") {
		t.Error("LOGON commands missing")
	}
	if strings.Count(s.Transcript(), "IDENTIFICATION NOT RECOGNIZED") != 1 {
		t.Error("'7' at LOGON is a failed log-on, not a game choice")
	}
}

// loggedOn skips the greeting with a command, landing in the shell.
func loggedOn(t *testing.T, reg *games.Registry) *testkit.Session {
	t.Helper()
	s := start(t, reg, Options{})
	s.Type("joshua").Type("help")
	return s
}

func TestIntentFalsePositivesGoToTheBrain(t *testing.T) {
	t.Parallel()
	s := loggedOn(t, gamestest.Registry())
	for _, in := range []string{
		"I don't want to play stub", "the golden gate bridge", "my heart's not in it",
		"poker face", "we should not play stub",
	} {
		s.Type(in)
		if s.Runner().Depth() != 1 {
			t.Fatalf("%q started a game", in)
		}
	}
	golden.AssertString(t, "intent_false_positives", s.Transcript())
}

func TestIntentStartsGames(t *testing.T) {
	t.Parallel()
	for _, in := range []string{"play stub", "Let's play stub", "lets play a game of stub", "OK. How about stub?", "stub", "Stub Game"} {
		s := loggedOn(t, gamestest.Registry())
		s.Type(in)
		if s.Runner().Depth() != 2 {
			t.Errorf("%q did not start the stub", in)
		}
	}
}

func TestOfferAndListSelection(t *testing.T) {
	t.Parallel()
	s := start(t, catalog.Registry(), Options{})
	s.Type("Joshua").Type("Hello.").Type("Fine.").Type("Sorry.") // SHALL WE PLAY A GAME?
	s.Type("yes")
	if !s.Contains("WHICH GAME?") {
		t.Fatal("accepting the offer must ask which game")
	}
	s.Type("7")
	if !s.Contains("** GAME ROUTINE NOT AVAILABLE **") {
		t.Error("choosing 7 (chess, not built yet) must decline in character")
	}
	s.Type("7")
	if strings.Count(s.Transcript(), "GAME ROUTINE NOT AVAILABLE") != 1 {
		t.Error("a number is a choice only right after a list")
	}
	golden.AssertString(t, "offer_accept", s.Transcript())
}

func TestShellHelpAndListGames(t *testing.T) {
	t.Parallel()
	s := start(t, catalog.Registry(), Options{})
	s.Type("Joshua").Type("HELP").Type("Help Games").Type("List Games").Type("15")
	golden.AssertString(t, "help_list_games", s.Transcript())
	for _, want := range []string{"COMMANDS AVAILABLE:", "TACTICAL AND STRATEGIC", "FALKEN'S MAZE", "GLOBAL THERMONUCLEAR WAR"} {
		if !s.Contains(want) {
			t.Errorf("missing %q", want)
		}
	}
	if !s.Contains("WOULDN'T YOU PREFER A GOOD GAME OF CHESS?") {
		t.Error("15 after LIST GAMES picks GTW, which first draws the film's counter-offer")
	}
}

func TestStubGameThinkWinAndAbort(t *testing.T) {
	t.Parallel()
	s := loggedOn(t, gamestest.Registry())
	s.Type("play stub").Type("THINK").Type("WIN")
	if !s.Contains("THOUGHT.") || !s.Contains("WINNER: PROFESSOR FALKEN") || !s.Contains("MOVES: 1") {
		t.Fatalf("game flow:\n%s", s.Transcript())
	}
	if !s.Contains("SHALL WE PLAY ANOTHER GAME?") || s.Runner().Depth() != 1 {
		t.Fatal("must return to the shell with an offer")
	}
	s.Type("yes")
	if !s.Contains("WHICH GAME?") {
		t.Error("the post-game offer must be accepted")
	}
	s.Type("stub").Esc().Esc()
	if !s.Contains("GAME TERMINATED BEFORE COMPLETION.") || s.Runner().Depth() != 1 {
		t.Errorf("Esc twice must abort:\n%s", s.Transcript())
	}
	golden.AssertString(t, "stub_game", s.Transcript())
}

func TestPlayFlagStartsTheGame(t *testing.T) {
	t.Parallel()
	s := start(t, gamestest.Registry(), Options{Play: "stub"})
	if s.Runner().Depth() != 2 || !s.Contains("STUB READY.") || s.Contains("LOGON") {
		t.Fatalf("--play must start the game directly:\n%s", s.Transcript())
	}
	s.Type("LOSE")
	if !s.Contains("WINNER: WOPR") {
		t.Error("loss verdict")
	}
	cat := start(t, catalog.Registry(), Options{Play: "chess"})
	if !cat.Contains("NOT AVAILABLE") || cat.Runner().Depth() != 1 {
		t.Error("--play of a planned game must decline in character")
	}
}

func TestScriptedBrainIsDeterministic(t *testing.T) {
	t.Parallel()
	b := NewScripted()
	snap := Snapshot{Phase: PhaseShell, Seed: 42, Turn: 3, Said: map[string]bool{}}
	a, _ := b.Reply(context.Background(), snap, "zzz")
	c, _ := b.Reply(context.Background(), snap, "zzz")
	if strings.Join(a.Lines, "|") != strings.Join(c.Lines, "|") {
		t.Error("same seed and turn must give the same reply")
	}
	r, _ := b.Reply(context.Background(), snap, "Is this a game or is it real?")
	if r.Lines[0] != "WHAT'S THE DIFFERENCE?" {
		t.Errorf("film line: %v", r.Lines)
	}
}

func TestEveryLineHasProvenance(t *testing.T) {
	t.Parallel()
	notice, err := os.ReadFile(filepath.Join("..", "..", "NOTICE.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, err := range script.Validate(allLines, string(notice)) {
		t.Error(err)
	}
}
