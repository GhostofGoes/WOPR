// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package gtw_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games/gtw"
	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/proto/host"
)

const promptAnother = "ANOTHER WAR, PROFESSOR? "

// noClimax starts GTW under a host that passes --no-climax on, in a session seeded seed. It
// returns the game too, to draw its board.
func noClimax(t *testing.T, seed uint64) (*testkit.GameSession, proto.Program) {
	t.Helper()
	g := gtw.New()
	return testkit.GameWith(t, g, info, "", host.Config{Seed: seed, NoClimax: true}), g
}

// fightOne plays a war on the war plan, from the side choice to the question after the kill
// ratios: the side, a target, the end of the list, both strikes, the kill ratios and Enter.
func fightOne(t *testing.T, s *testkit.GameSession, side, target string) {
	t.Helper()
	s.Type(side).Type(target).Type("").Type("").Type("").Type("").Type("")
	if asking, p := s.Asking(); !asking || p != promptAnother {
		t.Fatalf("after the kill ratios, asks %q, want %q:\n%s", p, promptAnother, s.Transcript())
	}
}

// wars cuts a transcript into its wars' exchanges, from the targets prompt to the assessment.
func wars(transcript string) []string {
	var out []string
	for _, part := range strings.Split(transcript, "AWAITING FIRST STRIKE COMMAND")[1:] {
		exchange, _, _ := strings.Cut(part, "STRIKE ASSESSMENT COMPLETE.")
		out = append(out, exchange)
	}
	return out
}

// With --no-climax, Enter after the kill ratios asks for another war, over the table, and
// never reaches the climax. The after-action report comes again above the question, since
// that Enter scrolls it away. Anything but yes or no asks again. No ends the game with no
// winner, and nothing is handed on: no tic-tac-toe, no ending.
func TestNoClimaxOffersAnotherWar(t *testing.T) {
	t.Parallel()
	s, g := noClimax(t, 1983)
	fightOne(t, s, "2", "Las Vegas")
	if s.Contains("** GAME ROUTINE RUNNING **") {
		t.Errorf("the climax started:\n%s", s.Transcript())
	}
	c := proto.NewCanvas(80, 19)
	g.View(c)
	if !strings.Contains(c.String(), "POPULATION DEATHS") {
		t.Errorf("the kill ratios are not on the board under the question:\n%s", c.String())
	}
	transcript := s.Transcript()
	_, report, _ := strings.Cut(transcript, "KILL RATIOS.\n\n\n")
	report, _, _ = strings.Cut(report, "PRESS ENTER TO CONTINUE.\n")
	if !strings.HasPrefix(report, "AGAINST HOLDING FIRE: ") || !strings.HasSuffix(transcript, "CONTINUE.\n\n\n"+report) {
		t.Errorf("the question must follow the report again:\n%s", transcript)
	}
	s.Type("perhaps")
	if asking, p := s.Asking(); !strings.HasSuffix(s.Transcript(), "YES FOR ANOTHER WAR. NO ENDS THE GAME.\n") || !asking || p != promptAnother {
		t.Errorf("an unclear answer must be asked again:\n%s", s.Transcript())
	}
	if _, over := s.Result(); over {
		t.Fatal("the game ended before the answer")
	}
	s.Type("no")
	res, over := s.Result()
	if !over || res.Outcome != proto.NoWinner || res.Next != nil || !slices.Equal(s.Launches(), []string{gtw.Slug}) {
		t.Errorf("no must end the game with no winner and no hand-off: %+v %v %v", res, over, s.Launches())
	}
}

// Yes, or Enter alone, starts a new war at the side choice, with every force back and the
// board cleared: the same side and targets fight the same exchange again, and another side
// fights the exchange a fresh game would.
func TestAnotherWarStartsAfresh(t *testing.T) {
	t.Parallel()
	s, _ := noClimax(t, 1983)
	fightOne(t, s, "2", "Las Vegas")
	s.Type("yes")
	if asking, p := s.Asking(); !asking || p != "PLEASE CHOOSE ONE:  " || strings.Count(s.Transcript(), "WHICH SIDE DO YOU WANT?") != 2 {
		t.Fatalf("yes asks %q, want the side choice:\n%s", p, s.Transcript())
	}
	fightOne(t, s, "1", "Moscow")
	s.Type("")
	fightOne(t, s, "2", "Las Vegas")
	s.Type("n")
	if _, over := s.Result(); !over {
		t.Fatalf("no did not end the game:\n%s", s.Transcript())
	}

	fresh, _ := noClimax(t, 1983)
	fightOne(t, fresh, "1", "Moscow")
	got, want := wars(s.Transcript()), wars(fresh.Transcript())
	if len(got) != 3 || got[0] != got[2] || got[1] != want[0] {
		t.Errorf("each war must start afresh:\n%s", s.Transcript())
	}
}

// An easter egg's doctrine is the launch's, so every war of it meets that doctrine, and WOPR
// names it as each board opens.
func TestAnotherWarKeepsTheEgg(t *testing.T) {
	t.Parallel()
	s, _ := noClimax(t, 1962)
	fightOne(t, s, "2", "Las Vegas")
	s.Type("yes")
	fightOne(t, s, "2", "Las Vegas")
	if n := strings.Count(s.Transcript(), "WOPR IS USING DOCTRINE S-1962 (HAIR TRIGGER)\nFIRST STRIKE LAUNCHED."); n != 2 {
		t.Errorf("the doctrine is named %d times in two wars, want 2:\n%s", n, s.Transcript())
	}
}

// A scenario's playback still returns to the menu after its kill ratios.
func TestNoClimaxPlaybackReturnsToTheMenu(t *testing.T) {
	t.Parallel()
	s, _ := noClimax(t, 1983)
	s.Type("3").Type("USSR First Strike").Type("").Type("") // the kill ratios, then Enter
	if asking, p := s.Asking(); !asking || p != "CHOOSE A SCENARIO: " {
		t.Errorf("the playback asks %q, want the menu:\n%s", p, s.Transcript())
	}
}
