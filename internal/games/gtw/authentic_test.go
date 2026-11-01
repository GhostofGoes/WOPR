// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package gtw_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/GhostofGoes/WOPR/internal/games/gtw"
	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/proto/host"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// authentic starts GTW under a host that passes --authentic on, configured by cfg, in launch
// mode mode. It returns the game too, to draw its views.
func authentic(t *testing.T, mode string, cfg host.Config) (*testkit.GameSession, proto.Program) {
	t.Helper()
	g := gtw.New()
	cfg.Authentic = true
	return testkit.GameWith(t, g, info, mode, cfg), g
}

// toClimax runs the clock from the end of the target list until GTW asks for a line, which
// under --authentic is the climax. It returns each view it drew on the way, a frame apart.
func toClimax(t *testing.T, s *testkit.GameSession, g proto.Program) []string {
	t.Helper()
	var views []string
	for range 2000 {
		if asking, _ := s.Asking(); asking {
			return views
		}
		s.Advance(frameEvery)
		c := proto.NewCanvas(80, 19)
		g.View(c)
		views = append(views, c.String())
	}
	t.Fatalf("the war never reached the climax by itself:\n%s", s.Transcript())
	return nil
}

// filmLines are the lines --authentic may show: the film's own text and the side choice's
// picture, the two outlines with their names, which is the film's picture redrawn.
func filmLines() []string {
	lines := script.FilmTexts(gtw.Lines)
	for _, l := range gtw.SideChoice() {
		lines = append(lines, l.Text)
	}
	return lines
}

// Under --authentic, every line GTW prints is the film's, from the side choice to the
// hand-off to tic-tac-toe: no third choice, no word at the targets prompt, no strike prompt,
// no report in the strip, no PRESS ENTER, and at the climax the film's notices alone, however
// often the player is refused. What is typed is echoed after a prompt that is the film's too.
func TestAuthenticShowsOnlyTheFilm(t *testing.T) {
	t.Parallel()
	s, g := authentic(t, "", host.Config{Seed: 1983})
	inputs := []string{"3", "scenarios", "2", "Las Vegas", "Smallville, Moscow, list, ?", "Seattle, Las Vegas", ""}
	for _, in := range inputs {
		s.Type(in)
	}
	if asking, _ := s.Asking(); asking {
		t.Fatalf("the board asks for something after the targets:\n%s", s.Transcript())
	}
	views := toClimax(t, s, g)
	if !slices.ContainsFunc(views, func(v string) bool { return strings.Contains(v, "POPULATION DEATHS") }) {
		t.Error("the kill ratios never showed on the way to the climax")
	}
	climaxInputs := []string{"List Games", "Chess", "GTW", "stop", "cease fire", "abort", "please", "help", "halt", "reset", "end"}
	for _, in := range climaxInputs {
		s.Type(in)
	}
	inputs = append(inputs, climaxInputs...)
	if res, over := s.Result(); over {
		t.Fatalf("the war ended before tic-tac-toe: %+v", res)
	}
	s.Type("Tic-Tac-Toe")
	if res, over := s.Result(); !over || res.Outcome != proto.NoWinner || s.Launches()[len(s.Launches())-1] != gtw.TicTacToeSlug {
		t.Fatalf("tic-tac-toe must hand off, as in the film: %+v %v %v", res, over, s.Launches())
	}
	inputs = append(inputs, "Tic-Tac-Toe")

	// The stand-in host's notice ends the transcript: it builds no tic-tac-toe.
	allowed := map[string]bool{"": true, "[CLEAR]": true, "** GAME ROUTINE NOT AVAILABLE **": true}
	for _, l := range filmLines() {
		allowed[l] = true
	}
	for _, in := range inputs { // the echo of a typed line: a film prompt, or none, then the line
		allowed[in] = true
		allowed["PLEASE CHOOSE ONE:  "+in] = true
	}
	for _, l := range strings.Split(strings.TrimSuffix(s.Transcript(), "\n"), "\n") {
		if !allowed[l] {
			t.Errorf("not the film's: %q", l)
		}
	}
	for _, want := range []string{"** GAME ROUTINE RUNNING **", "** IDENTIFICATION NOT RECOGNIZED **", "** IMPROPER REQUEST **", "FALKEN'S MAZE"} {
		if !s.Contains(want) {
			t.Errorf("missing the film's %q:\n%s", want, s.Transcript())
		}
	}
	golden.AssertString(t, "authentic", s.Transcript())
}

// Under --authentic the war runs by itself on the war plan, as under AUTO. The film's side, the Soviet Union, gets the film's kill ratios, which
// hold up for filmRatios frames; the United States gets the war's own. A name the board does
// not know strikes nothing, and a list with no known target takes the enemy's first cities.
func TestAuthenticWarRunsByItself(t *testing.T) {
	t.Parallel()
	ratios := func(side string, targets ...string) (string, int) {
		s, g := authentic(t, "", host.Config{Seed: 7})
		s.Type(side)
		for _, tg := range targets {
			s.Type(tg)
		}
		s.Type("")
		table, frames := "", 0
		for _, v := range toClimax(t, s, g) {
			if strings.Contains(v, "POPULATION DEATHS") {
				table = v
				frames++
			}
		}
		return table, frames
	}
	soviet, frames := ratios("2", "Las Vegas", "Seattle")
	if !strings.Contains(soviet, " 72 MILLION") || !strings.Contains(soviet, "110 MILLION") || !strings.Contains(soviet, "68%") {
		t.Errorf("the film's side does not get the film's figures:\n%s", soviet)
	}
	if frames < 70 || frames > 90 {
		t.Errorf("the kill ratios showed for %d frames, want about 80", frames)
	}
	if us, _ := ratios("1", "Moscow"); strings.Contains(us, "110 MILLION") || !strings.Contains(us, "POPULATION DEATHS") {
		t.Errorf("the United States gets the film's figures, or none:\n%s", us)
	}
	if unknown, _ := ratios("1", "Smallville", "Las Vegas"); !strings.Contains(unknown, "POPULATION DEATHS") {
		t.Errorf("a list of unknown names does not reach the kill ratios:\n%s", unknown)
	}
}

// --authentic wins over what the film does not have, wherever the Env comes from (the command
// line refuses them together): a scenario's launch mode starts at the side choice, the
// launch window never opens, --no-climax still reaches the climax, and an easter-egg seed
// meets a measured response, unnamed. Under Instant the war still stays on the clock.
func TestAuthenticWins(t *testing.T) {
	t.Parallel()
	forces := func(seed uint64) string {
		s, g := authentic(t, gtw.ScenarioLaunch(2), host.Config{Seed: seed, NoClimax: true, LaunchWindow: 5 * time.Second})
		if !s.Contains("WHICH SIDE DO YOU WANT?") || s.Contains("USSR FIRST STRIKE") {
			t.Fatalf("seed %d: a scenario's launch mode under --authentic:\n%s", seed, s.Transcript())
		}
		s.Type("1").Type("Moscow").Type("")
		views := toClimax(t, s, g)
		if s.Contains("DOCTRINE") || s.Contains("WINDOW") || s.Contains("ANOTHER WAR") || !s.Contains("** GAME ROUTINE RUNNING **") {
			t.Errorf("seed %d: --authentic did not win:\n%s", seed, s.Transcript())
		}
		rows := strings.Split(views[len(views)-1], "\n")
		return rows[16][49:] + rows[17][49:] // the forces table, once WOPR has fired everything
	}
	if hair, measured := forces(1962), forces(1983); hair != measured {
		t.Errorf("seed 1962 under --authentic fights another doctrine: %q, want %q", hair, measured)
	}
}

// Paced, the film's war plays on the clock as GTW's does, frame by frame, and says nothing:
// no strip line, no remark or orders hint on the board's last row, and no prompt until the
// climax, which comes about 18 s of flight, 3 s of DEFCON 1 and 8 s of kill ratios later.
func TestAuthenticPlaysOnTheClock(t *testing.T) {
	t.Parallel()
	g := gtw.New()
	g.Start(proto.Env{Seed: 1983, Width: 80, Height: 19, Authentic: true})
	g.Handle(proto.LineEvent{Text: "2"})
	g.Handle(proto.LineEvent{Text: "Las Vegas"})
	outs := g.Handle(proto.LineEvent{Text: ""})
	for frames := 0; ; frames++ {
		for _, o := range outs {
			switch o := o.(type) {
			case proto.Say:
				if frames == 0 || o.Lines[0] != "** GAME ROUTINE RUNNING **" {
					t.Fatalf("frame %d: the strip says %q", frames, o.Lines)
				}
				if frames < 280 || frames > 320 {
					t.Errorf("the climax came after %d frames, want about 300", frames)
				}
				return
			case proto.Prompt:
				t.Fatalf("frame %d: a prompt %q before the climax", frames, o.Text)
			}
		}
		c := proto.NewCanvas(80, 19)
		g.View(c)
		if v := c.String(); strings.Contains(v, "PROFESSOR") || strings.Contains(v, "ORDERS") {
			t.Fatalf("frame %d: a remark or an orders hint on the film's board:\n%s", frames, v)
		}
		if frames > 1000 {
			t.Fatal("the war never reached the climax by itself")
		}
		outs = g.Handle(proto.TickEvent{Dt: frameEvery})
	}
}
