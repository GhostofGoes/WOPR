// Package scenes holds movie mode's scenes as data (docs/PLAN.md §7): the film's WOPR
// terminal scenes, step by step, with a provenance tag on every step. The director in
// internal/movie plays them.
//
// Scenes contain only text that appears on the WOPR terminal on screen: no spoken-only
// dialogue. David's lines are Type steps in mixed case, as on screen (RF-9). Lines not yet
// confirmed against the film in the M5 viewing pass are tagged reconstructed.
package scenes

import (
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
		case Run:
			out = append(out, script.Ls{{Prov: st.Prov}})
		}
	}
	return out
}
