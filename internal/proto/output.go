package proto

import (
	"context"
	"errors"
	"time"
)

// ErrCanceled is ThinkDone.Err when the user cancelled the request (Esc while a brain
// reply was pending).
var ErrCanceled = errors.New("cancelled by the user")

// Output is something a program asks the host to do. The set is closed: every output a
// program can produce is declared in this file.
type Output interface{ isOutput() }

// Say prints lines through the typewriter.
type Say struct {
	Lines []string
	Pace  Pace
}

// Prompt switches to line mode; the next LineEvent answers it.
type Prompt struct{ Text string }

// AwaitKeys switches to key mode; KeyEvents follow until the next Prompt.
type AwaitKeys struct{ Hint string }

// Animate asks for TickEvents every Every; 0 stops them.
type Animate struct{ Every time.Duration }

// Wait pauses output for D. Any key skips it; it is zero under Env.Instant.
type Wait struct{ D time.Duration }

// Clear starts a new page. Scrollback keeps what was above it.
type Clear struct{}

// SetLayout changes where the program's View goes.
type SetLayout struct {
	Layout    Layout
	PanelRows int // rows for LayoutPanel; ignored otherwise
}

// Redraw asks the host to call View again.
type Redraw struct{}

// Think runs Fn off the UI goroutine and delivers its result as a ThinkDone event.
//
// Fn must use only values it captured (copy positions, e.g. as a FEN string): the program
// keeps running while Fn does. With Env.Deterministic, Fn must stop at Limit; Budget is a
// safety cap. Otherwise Budget is the main bound: when the context's deadline passes, Fn
// returns the best result it has, without an error.
type Think struct {
	Fn     func(ctx context.Context) (any, error)
	Limit  Limit
	Budget time.Duration
}

// Launch pushes another program on top of this one.
type Launch struct{ Slug, Mode string }

// Done pops this program; the one below receives GameOver.
type Done struct{ Result Result }

// Quit ends the session normally (exit code 0).
type Quit struct{}

func (Say) isOutput()       {}
func (Prompt) isOutput()    {}
func (AwaitKeys) isOutput() {}
func (Animate) isOutput()   {}
func (Wait) isOutput()      {}
func (Clear) isOutput()     {}
func (SetLayout) isOutput() {}
func (Redraw) isOutput()    {}
func (Think) isOutput()     {}
func (Launch) isOutput()    {}
func (Done) isOutput()      {}
func (Quit) isOutput()      {}
