// Package scenes holds movie mode's scenes as data (docs/PLAN.md §7): the film's WOPR
// terminal scenes, step by step, with a provenance tag on every step. The director in
// internal/movie plays them.
//
// The only film text in the scenes is what WOPR's terminal shows on screen, David's typing
// included; there is no spoken-only dialogue. The dial, GTW's strike exchange, tic-tac-toe's
// prompts and the game clock's seconds are this project's own text and are tagged original.
// David's lines are Type steps in mixed case, as on screen (RF-9), except his entries at the
// NORAD console in the climax, which are in capitals as the transcription has them. Film lines
// not yet confirmed against the film in the M5 viewing pass are tagged reconstructed.
package scenes

import (
	"fmt"
	"strings"
	"time"

	"github.com/GhostofGoes/WOPR/internal/games/gtw"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// Step is one beat of a scene.
type Step interface{ isStep() }

// Type is a line the user types: the prompt it answers shows at once, then the text follows at
// typing pace, a few seeded keystrokes at a time, and a blank line, as the console echoes an
// answer (proto.Typed).
type Type struct {
	Prompt script.Ls // the prompt ("LOGON: "), or none
	Text   script.L  // a User line, in mixed case as on screen
}

// Say is WOPR's output at Pace.
type Say struct {
	Lines script.Ls
	Pace  proto.Pace
}

// Clear starts a new page.
type Clear struct{ Prov script.Prov }

// Wait pauses for D (none under --instant).
type Wait struct {
	D    time.Duration
	Prov script.Prov
}

// Board shows the big board at a stage of the film's war, drawn by games/gtw's film renderer.
// With Play, the war plays out on the board from where it stands to At, printing the game's
// strip lines; without, the board jumps there. At 0 puts the board away (the console layout).
type Board struct {
	At   gtw.FilmStage
	Play bool
	Prov script.Prov
}

// Clock is the game clock WOPR shows on David's screen: two readings that tick, elapsed up and
// remaining down, for D, and stay up until the scene ends. Lines are the readings with # where
// the seconds go; Start is the seconds they start at. The labels, hours and minutes are the
// film's; the seconds are this project's own until the M5 viewing pass reads them off the film.
type Clock struct {
	Lines script.Ls
	Start [2]int        // the seconds of each reading at the start
	D     time.Duration // how long the clock runs before the scene goes on
	Prov  script.Prov   // of Start and D
}

// Reading is the clock's lines after it has run for t: the first reading's seconds count up from
// its start, the second's down, and neither passes a minute (the package's tests check that D
// keeps them within it).
func (c Clock) Reading(t time.Duration) []string {
	n := int(t / time.Second)
	secs := [2]int{min(c.Start[0]+n, 59), max(c.Start[1]-n, 0)}
	out := make([]string, len(c.Lines))
	for i, l := range c.Lines {
		out[i] = l.Text
		if i < len(secs) {
			out[i] = strings.Replace(l.Text, "#", fmt.Sprintf("%02d", secs[i]), 1)
		}
	}
	return out
}

// Run launches a real program over the director, as proto.Launch{Slug, Mode} does; the scene
// goes on once it ends. The board makes way for it. In movie mode the program supplies its own
// scripted input (the climax's tic-tac-toe and the ending).
type Run struct {
	Slug, Mode string
	Prov       script.Prov
}

func (Type) isStep()  {}
func (Say) isStep()   {}
func (Clear) isStep() {}
func (Wait) isStep()  {}
func (Board) isStep() {}
func (Clock) isStep() {}
func (Run) isStep()   {}

// Scene is one of the film's terminal scenes.
type Scene struct {
	Slug  string // the name to play it by: wopr -m first-contact
	Title string // as the menu shows it
	Blurb string // one line for --scenes and the menu
	// Interactive marks a scene whose WOPR lines the persona also prints when it is fed the
	// scene's Type steps: the movie's consistency test checks they agree.
	Interactive bool
	Steps       []Step
}

// Lines is every line a scene shows, and every step's provenance as an empty line, for
// script.Validate.
func (s Scene) Lines() []script.Ls {
	var out []script.Ls
	for _, st := range s.Steps {
		switch st := st.(type) {
		case Type:
			out = append(out, st.Prompt, script.Ls{st.Text})
		case Say:
			out = append(out, st.Lines)
		case Clear:
			out = append(out, script.Ls{{Prov: st.Prov}})
		case Wait:
			out = append(out, script.Ls{{Prov: st.Prov}})
		case Board:
			out = append(out, script.Ls{{Prov: st.Prov}})
		case Clock:
			out = append(out, st.Lines, script.Ls{{Prov: st.Prov}})
		case Run:
			out = append(out, script.Ls{{Prov: st.Prov}})
		}
	}
	return out
}
