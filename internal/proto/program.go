// Package proto is the protocol between the host (internal/ui and internal/proto/host) and
// every "program" it runs: the persona, each game, the ending and the movie director.
//
// A program never touches the terminal. It receives Events, returns Outputs, and draws
// into a Canvas of semantic styles. It must not block: anything slow (an AI search, a
// brain reply) is returned as a Think output and runs off the UI goroutine.
//
// The package depends on the standard library only.
package proto

import "time"

// Program is implemented by the persona, every game, the ending and the movie director.
type Program interface {
	// Start is called once, after the first valid terminal size is known.
	Start(Env) []Output
	// Handle reacts to one event. It must return quickly.
	Handle(Event) []Output
	// View draws the program's layout area. It is never called for LayoutConsole.
	View(c *Canvas)
}

// Env is what a program learns about its surroundings when it starts.
type Env struct {
	Seed          uint64 // derive random streams with NewRand(Seed, StreamID)
	Width, Height int    // the program's layout area, always at least its minimum
	Instant       bool   // no pacing: --instant, tests
	Deterministic bool   // --seed given, tests, movie mode: searches obey Limit, not the clock
	Mode          string // optional launch mode, e.g. "climax"
}

// Layout says where a program's View goes.
type Layout uint8

// Layouts.
const (
	LayoutConsole Layout = iota // console text only
	LayoutPanel                 // View on top, console strip and input line below
	LayoutFull                  // View fills the screen above a 3-row strip and the input line
)

// Limit bounds a search deterministically. Zero fields mean "no limit".
type Limit struct {
	MaxDepth int
	MaxNodes uint64
}

// Outcome is how a program ended.
type Outcome uint8

// Outcomes.
const (
	Win Outcome = iota
	Loss
	Draw
	NoWinner
	Aborted
)

// Result is reported by Done.
type Result struct {
	Outcome   Outcome
	Lines     []string // kill ratios and other detail for the persona's verdict
	Next      *Launch  // hand-off without a verdict, e.g. gtw → tictactoe(climax) → ending
	NoVerdict bool     // the persona adds nothing after this program's last line
}

// Pace is how fast the typewriter reveals text.
type Pace uint8

// Paces.
const (
	PaceSpeech  Pace = iota // WOPR speaking, about 30 characters per second
	PaceTable               // tables and lists, fast
	PaceInstant             // boards and redraws, no delay
	PaceTyping              // a simulated user typing (movie mode), about 8 characters per second
)

// CharsPerSecond returns the typewriter speed for p; 0 means instant.
func (p Pace) CharsPerSecond() float64 {
	switch p {
	case PaceSpeech:
		return 30
	case PaceTable:
		return 240
	case PaceTyping:
		return 8
	default:
		return 0
	}
}

// Event is something the host tells a program. The set is closed.
type Event interface{ isEvent() }

// LineEvent answers the last Prompt. Text is sanitised and trimmed; "" is an empty Enter.
type LineEvent struct{ Text string }

// KeyEvent is a key press, delivered only in key mode (after AwaitKeys).
type KeyEvent struct {
	Key  Key
	Rune rune // set when Key is KeyRune
}

// TickEvent arrives at the cadence a program asked for with Animate.
type TickEvent struct{ Dt time.Duration }

// ResizeEvent reports a change of the program's layout area.
type ResizeEvent struct{ Width, Height int }

// ThinkDone carries the result of the program's Think. A program has at most one Think
// in flight: a new Think replaces the previous one, whose result is dropped. Err is
// ErrCanceled when the user cancelled the Think with Esc and the program keeps running
// (a brain reply at the root); a confirmed abort pops the program, so its result is never
// delivered. For a search, a deadline is not an error (Fn returns its best result); slow
// I/O such as a brain reply returns an error instead.
type ThinkDone struct {
	Value any
	Err   error
}

// GameOver tells a program that the program it launched has finished.
type GameOver struct{ Result Result }

func (LineEvent) isEvent()   {}
func (KeyEvent) isEvent()    {}
func (TickEvent) isEvent()   {}
func (ResizeEvent) isEvent() {}
func (ThinkDone) isEvent()   {}
func (GameOver) isEvent()    {}
