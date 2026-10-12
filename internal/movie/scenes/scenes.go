// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

// Package scenes holds movie mode's scenes as data (docs/PLAN.md §7): the film's WOPR
// terminal scenes, step by step, with a provenance tag on every step. The director in
// internal/movie plays them.
//
// The only film text in the scenes is what WOPR's terminal shows on screen, David's typing
// included; there is no spoken-only dialogue. A line tagged film is that text, character for
// character, checked against the film's transcriptions (docs/PLAN.md §2.3), and it is never
// changed (AGENTS.md). A line only one transcription covers, or that the screens flash by too
// fast to read, is tagged reconstructed. The dial, GTW's strike exchange and tic-tac-toe's
// prompts are this project's own text and are tagged original. David's lines are Type steps in
// mixed case, as on screen (RF-9), except his entries at the NORAD console in the climax,
// which are in capitals as the film and the transcription have them.
package scenes

import (
	"fmt"
	"regexp"
	"strconv"
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

// Clock is the game clock WOPR shows on David's screen: the film's lines, whose two readings
// tick, elapsed up and remaining down, for D, and stay up until the scene ends. The seconds
// start at the film's and this project ticks them on, so Prov is D's: the film shows no
// ticking.
type Clock struct {
	Lines script.Ls
	D     time.Duration // how long the clock runs before the scene goes on
	Prov  script.Prov   // of D
}

// seconds matches a reading's seconds, "50 SEC".
var seconds = regexp.MustCompile(`\b(\d\d) SEC\b`)

// Reading is the clock's lines after it has run for t: the first reading's seconds count up
// from the film's, the second's down, and neither passes a minute (the package's tests check
// that D keeps them within it).
func (c Clock) Reading(t time.Duration) []string {
	n := int(t / time.Second)
	out := make([]string, len(c.Lines))
	k := 0
	for i, l := range c.Lines {
		out[i] = seconds.ReplaceAllStringFunc(l.Text, func(m string) string {
			s, _ := strconv.Atoi(m[:2])
			switch k {
			case 0:
				s = min(s+n, 59)
			case 1:
				s = max(s-n, 0)
			}
			k++
			return fmt.Sprintf("%02d SEC", s)
		})
	}
	return out
}

// Readings are the clock's two seconds readings as the film shows them, in order.
func (c Clock) Readings() []int {
	var out []int
	for _, l := range c.Lines {
		for _, m := range seconds.FindAllStringSubmatch(l.Text, -1) {
			s, _ := strconv.Atoi(m[1])
			out = append(out, s)
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
