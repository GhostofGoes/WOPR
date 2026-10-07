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

// Say prints lines through the typewriter. With Open, the last line stays open: the next
// Say continues it on the same row, as a typist's keystrokes do (movie mode, Typed).
type Say struct {
	Lines []string
	Pace  Pace
	Open  bool
}

// Prompt switches to line mode; the next LineEvent answers it.
type Prompt struct{ Text string }

// AwaitKeys switches to key mode; KeyEvents follow until the next Prompt.
//
// Capture is for a root program other than the persona (the movie director): it then
// receives every key, Esc included (KeyEsc), with none of the host's side effects: no key
// skips the typewriter and Esc is not the host's. While a program it launched runs, keys
// other than Ctrl+C do nothing. The host ignores Capture from a launched program.
type AwaitKeys struct {
	Hint    string
	Capture bool
}

// Animate asks for TickEvents every Every; 0 stops them.
type Animate struct{ Every time.Duration }

// Wait pauses output for D. Any key skips it; it is zero under Env.Instant.
type Wait struct{ D time.Duration }

// Clear starts a new page. Scrollback keeps what was above it. The host starts one by
// itself when it launches a program whose layout is not LayoutConsole (see Launch).
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

// Launch pushes another program on top of this one. A program with its own screen (any
// layout but LayoutConsole) starts on a new page, whether it was launched or handed off
// to, so its console strip shows only its own text. Ending a program does not start one:
// its last lines stay above the verdict.
type Launch struct{ Slug, Mode string }

// Done pops this program; the one below receives GameOver.
type Done struct{ Result Result }

// Quit ends the session normally (exit code 0).
type Quit struct{}

// Hold freezes output while On: the typewriter, Wait, Animate and Blink stop where they are,
// as they do behind the TERMINAL TOO SMALL card, until a Hold with On false (movie mode's
// pause). Keys still arrive. A hold ends with the program that set it.
type Hold struct{ On bool }

// Drain asks for a Drained event once everything output before it has been revealed and its
// pauses waited out, so a program can pace itself one step at a time (movie mode). A newer
// Drain replaces a pending one, and only the running program hears it.
type Drain struct{}

// Skip reveals at once everything output before it, as a key press skips the typewriter
// (movie mode's next and previous scene).
type Skip struct{}

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
func (Hold) isOutput()      {}
func (Drain) isOutput()     {}
func (Skip) isOutput()      {}
