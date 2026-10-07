// Package host runs proto programs. It is the protocol's only interpreter and it does
// not import Bubble Tea: internal/ui adapts it to the terminal, and games/testkit drives
// the same code in tests.
//
// The runner owns the program stack (the persona or the movie director at the bottom, games
// launched on top), the input mode, the Esc state machine, Think jobs, Animate cadence and the
// movie-mode hooks (key capture, Hold, Drain). Each call returns Effects, in order, for the
// caller to apply.
package host

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/GhostofGoes/WOPR/internal/proto"
)

// Placement is where a program's View goes.
type Placement struct {
	Layout    proto.Layout
	PanelRows int
	NoAbort   bool // Esc cannot end it: the film's ending (an internal registry entry)
}

// Resolver builds the program for a Launch (from the game registry).
type Resolver func(l proto.Launch) (proto.Program, Placement, error)

// Area returns the layout area for a placement at the current terminal size.
type Area func(Placement) (w, h int)

// Config configures a Runner.
type Config struct {
	Seed          uint64
	Instant       bool
	Deterministic bool
	ReduceMotion  bool
	Resolve       Resolver
	Area          Area
}

// Timing constants.
const (
	EscWindow = 3 * time.Second  // a second Esc within this window aborts a game
	SafetyCap = 60 * time.Second // deadline for deterministic searches bounded by Limit
)

// Effect is something the caller must do, in order.
type Effect interface{ isEffect() }

// Print queues lines on the typewriter. With Open, the last line stays open: the next Print
// continues it on the same row.
type Print struct {
	Lines []string
	Pace  proto.Pace
	Open  bool
}

// Pause queues a skippable pause in the typewriter.
type Pause struct{ D time.Duration }

// PageBreak starts a new console page.
type PageBreak struct{}

// AskLine switches to line mode once the typewriter reaches it.
type AskLine struct{ Prompt string }

// AskKeys switches to key mode.
type AskKeys struct{ Hint string }

// StartThink runs Fn off the UI goroutine; report the outcome with Runner.ThinkResult.
type StartThink struct {
	Gen      uint64
	Fn       func(context.Context) (any, error)
	Deadline time.Duration // 0: none
}

// CancelThink cancels a running Think; its result will be dropped.
type CancelThink struct{ Gen uint64 }

// Notice shows a transient one-line host message; "" clears it.
type Notice struct{ Text string }

// Relayout tells the caller the running program's placement changed. NewProgram is set
// when a different program now runs (a launch, a pop or a hand-off): the old program's
// prompt, and any line typed ahead for it, no longer apply.
type Relayout struct {
	Placement  Placement
	NewProgram bool
	// Last is the program that just ended (on a pop), with its placement, so the caller can
	// keep showing its final View until that program's last output has been revealed.
	Last      proto.Program
	LastPlace Placement
}

// Redraw asks for the running program's View to be drawn again.
type Redraw struct{}

// Exit ends the session normally (exit code 0).
type Exit struct{}

// Hold freezes (On) or releases the typewriter, pauses, Animate and Blink, as the TOO SMALL
// card does. The runner stops Animate itself; the caller stops the rest.
type Hold struct{ On bool }

// Drain queues a marker after everything printed so far. When the typewriter reaches it, the
// caller reports ID with Runner.Drained.
type Drain struct{ ID uint64 }

// Skip reveals everything queued so far at once, as a key press would.
type Skip struct{}

func (Print) isEffect()       {}
func (Pause) isEffect()       {}
func (PageBreak) isEffect()   {}
func (AskLine) isEffect()     {}
func (AskKeys) isEffect()     {}
func (StartThink) isEffect()  {}
func (CancelThink) isEffect() {}
func (Notice) isEffect()      {}
func (Relayout) isEffect()    {}
func (Redraw) isEffect()      {}
func (Exit) isEffect()        {}
func (Hold) isEffect()        {}
func (Drain) isEffect()       {}
func (Skip) isEffect()        {}

type inputMode uint8

const (
	modeNone inputMode = iota
	modeLine
	modeKeys
)

type frame struct {
	prog      proto.Program
	place     Placement
	mode      inputMode
	every     time.Duration // Animate cadence; 0 = none
	acc       time.Duration
	thinkGen  uint64 // pending Think, 0 = none
	areaW     int
	areaH     int
	isRoot    bool
	launchKey proto.Launch
	seed      uint64 // Env.Seed for this program
	capture   bool   // the root captures every key (AwaitKeys.Capture)
	drain     uint64 // the pending Drain's marker, 0 = none
}

// Runner interprets the protocol for a stack of programs.
type Runner struct {
	cfg      Config
	stack    []*frame
	gen      uint64
	escArmed time.Duration // time left in the Esc window; 0 = not armed
	exited   bool
	plays    map[string]uint16 // launches so far, by slug
	holder   *frame            // the program holding output frozen (Hold), nil = none
	drains   uint64            // Drain markers issued
}

// New returns a Runner.
func New(cfg Config) *Runner { return &Runner{cfg: cfg} }

// Start runs the root program (the persona, or the movie director).
func (r *Runner) Start(root proto.Program, place Placement) []Effect {
	f := &frame{prog: root, place: place, isRoot: true, seed: r.cfg.Seed}
	r.stack = []*frame{f}
	f.areaW, f.areaH = r.area(place)
	return r.apply(f, root.Start(r.env(f, "")))
}

// Top returns the running program and its placement.
func (r *Runner) Top() (proto.Program, Placement) {
	if len(r.stack) == 0 {
		return nil, Placement{}
	}
	f := r.top()
	return f.prog, f.place
}

// Depth is the number of programs on the stack (1: only the root).
func (r *Runner) Depth() int { return len(r.stack) }

// KeyMode reports whether the running program wants keys rather than lines.
func (r *Runner) KeyMode() bool { return len(r.stack) > 0 && r.top().mode == modeKeys }

// Thinking reports whether the running program has a Think pending.
func (r *Runner) Thinking() bool { return len(r.stack) > 0 && r.top().thinkGen != 0 }

// NeedsTicks reports whether Advance must be called (animation or an armed Esc). Nothing
// moves while output is held.
func (r *Runner) NeedsTicks() bool {
	return len(r.stack) > 0 && r.holder == nil && (r.top().every > 0 || r.escArmed > 0)
}

// Captures reports whether the running program is a root that captures keys: every key goes
// to it with Key, Esc as proto.KeyEsc, and none skips output (docs/PLAN.md §7).
func (r *Runner) Captures() bool {
	if len(r.stack) == 0 {
		return false
	}
	f := r.top()
	return f.isRoot && f.mode == modeKeys && f.capture
}

// Shielded reports whether a capturing root has launched the running program: until it ends,
// keys other than Ctrl+C do nothing (the movie's ending plays through).
func (r *Runner) Shielded() bool { return len(r.stack) > 1 && r.stack[0].capture }

// Held reports whether output is frozen by Hold.
func (r *Runner) Held() bool { return r.holder != nil }

// commandWords upper-cases text and turns every run of characters other than A-Z and
// 0-9 into one space, so "Log off." and "quit!" read as LOG OFF and QUIT. (The host may
// import only proto; prompt.Normalize does the same for the persona.)
func commandWords(text string) string {
	return strings.Join(strings.FieldsFunc(strings.ToUpper(text), func(r rune) bool {
		return (r < 'A' || r > 'Z') && (r < '0' || r > '9')
	}), " ")
}

// hostCommands end the session from anywhere, including inside a game.
var hostCommands = map[string]bool{"LOGOFF": true, "LOG OFF": true, "EXIT": true, "QUIT": true}

// Line delivers a submitted line to the running program.
func (r *Runner) Line(text string) []Effect {
	if r.exited || len(r.stack) == 0 {
		return nil
	}
	effects := r.disarm()
	if hostCommands[commandWords(text)] {
		r.exited = true
		return append(effects, Exit{})
	}
	f := r.top()
	if f.mode != modeLine {
		return effects
	}
	f.mode = modeNone
	return append(effects, r.apply(f, f.prog.Handle(proto.LineEvent{Text: text}))...)
}

// Key delivers a key to the running program in key mode. KeyEsc reaches only a root that
// captures keys; everywhere else Esc is the host's (Runner.Esc).
func (r *Runner) Key(k proto.Key, ch rune) []Effect {
	if r.exited || len(r.stack) == 0 {
		return nil
	}
	effects := r.disarm()
	f := r.top()
	if f.mode != modeKeys || k == proto.KeyEsc && !r.Captures() {
		return effects
	}
	return append(effects, r.apply(f, f.prog.Handle(proto.KeyEvent{Key: k, Rune: ch}))...)
}

// Esc handles the Esc key: it cancels a pending brain reply at the root, and arms then
// confirms aborting a game.
func (r *Runner) Esc() []Effect {
	if r.exited || len(r.stack) == 0 {
		return nil
	}
	f := r.top()
	if f.isRoot {
		if f.thinkGen == 0 {
			return nil
		}
		gen := f.thinkGen
		f.thinkGen = 0
		effects := []Effect{CancelThink{Gen: gen}}
		return append(effects, r.apply(f, f.prog.Handle(proto.ThinkDone{Err: proto.ErrCanceled}))...)
	}
	if f.place.NoAbort {
		return nil // the ending plays out; Ctrl+C still quits
	}
	if r.escArmed == 0 {
		r.escArmed = EscWindow
		return []Effect{Notice{Text: "** PRESS ESC AGAIN TO END GAME **"}}
	}
	r.escArmed = 0
	return append([]Effect{Notice{}}, r.pop(proto.Result{Outcome: proto.Aborted})...)
}

// Advance moves time forward: Animate ticks and the Esc window. While output is held, time
// stands still.
func (r *Runner) Advance(dt time.Duration) []Effect {
	if r.exited || len(r.stack) == 0 || r.holder != nil {
		return nil
	}
	var effects []Effect
	if r.escArmed > 0 {
		r.escArmed -= dt
		if r.escArmed <= 0 {
			r.escArmed = 0
			effects = append(effects, Notice{})
		}
	}
	f := r.top()
	if f.every > 0 {
		f.acc += dt
		if f.acc >= f.every {
			tick := f.acc // one coalesced tick after a late or suspended clock
			f.acc = 0
			effects = append(effects, r.apply(f, f.prog.Handle(proto.TickEvent{Dt: tick}))...)
		}
	}
	return effects
}

// Drained reports that the typewriter reached the Drain marker id. Only the running program's
// latest Drain is answered: an older marker, or one left by a program that has since
// launched another or ended, is dropped.
func (r *Runner) Drained(id uint64) []Effect {
	if r.exited || len(r.stack) == 0 || id == 0 {
		return nil
	}
	f := r.top()
	if f.drain != id {
		return nil
	}
	f.drain = 0
	return r.apply(f, f.prog.Handle(proto.Drained{}))
}

// ThinkResult delivers a finished Think. Results for cancelled or superseded jobs are
// dropped.
func (r *Runner) ThinkResult(gen uint64, value any, err error) []Effect {
	if r.exited {
		return nil
	}
	for _, f := range r.stack {
		if f.thinkGen == gen && gen != 0 {
			f.thinkGen = 0
			if f != r.top() {
				return nil // a program above it is running; this cannot happen in practice
			}
			return r.apply(f, f.prog.Handle(proto.ThinkDone{Value: value, Err: err}))
		}
	}
	return nil
}

// Resize recomputes layout areas and tells the running program when its area changed.
func (r *Runner) Resize() []Effect {
	if r.exited || len(r.stack) == 0 {
		return nil
	}
	f := r.top()
	w, h := r.area(f.place)
	if w == f.areaW && h == f.areaH {
		return nil
	}
	f.areaW, f.areaH = w, h
	return r.apply(f, f.prog.Handle(proto.ResizeEvent{Width: w, Height: h}))
}

func (r *Runner) top() *frame { return r.stack[len(r.stack)-1] }

func (r *Runner) area(p Placement) (int, int) {
	if r.cfg.Area == nil {
		return 80, 24
	}
	return r.cfg.Area(p)
}

// launchSeed derives a launched program's Env.Seed from the session seed, the game and
// how many times it has been played, so every game and every replay draws its own
// sequence (docs/PLAN.md §4.6, streams). Programs derive their own streams from it.
func (r *Runner) launchSeed(slug string) uint64 {
	if r.plays == nil {
		r.plays = map[string]uint16{}
	}
	n := r.plays[slug]
	r.plays[slug] = n + 1
	return proto.NewRand(r.cfg.Seed, proto.GameStream(slug, n)).Uint64()
}

func (r *Runner) env(f *frame, mode string) proto.Env {
	return proto.Env{
		Seed: f.seed, Width: f.areaW, Height: f.areaH,
		Instant: r.cfg.Instant, Deterministic: r.cfg.Deterministic, ReduceMotion: r.cfg.ReduceMotion, Mode: mode,
	}
}

// Disarm closes the Esc window: any key other than Esc does (docs/PLAN.md §4.3). The UI
// calls it for keys that only edit the input line.
func (r *Runner) Disarm() []Effect { return r.disarm() }

func (r *Runner) disarm() []Effect {
	if r.escArmed == 0 {
		return nil
	}
	r.escArmed = 0
	return []Effect{Notice{}}
}

// apply turns a program's outputs into effects, in order.
func (r *Runner) apply(f *frame, outs []proto.Output) []Effect {
	var effects []Effect
	for i, o := range outs {
		if r.exited {
			return effects
		}
		switch o := o.(type) {
		case proto.Say:
			effects = append(effects, Print{Lines: o.Lines, Pace: o.Pace, Open: o.Open})
		case proto.Wait:
			if !r.cfg.Instant {
				effects = append(effects, Pause{D: o.D})
			}
		case proto.Clear:
			effects = append(effects, PageBreak{})
		case proto.Prompt:
			f.mode, f.capture = modeLine, false
			effects = append(effects, AskLine{Prompt: o.Text})
		case proto.AwaitKeys:
			f.mode, f.capture = modeKeys, o.Capture && f.isRoot // a launched game never captures Esc
			effects = append(effects, AskKeys{Hint: o.Hint})
		case proto.Animate:
			f.every, f.acc = o.Every, 0
		case proto.SetLayout:
			f.place.Layout, f.place.PanelRows = o.Layout, o.PanelRows // NoAbort stays with the program
			f.areaW, f.areaH = r.area(f.place)
			effects = append(effects, Relayout{Placement: f.place})
			effects = append(effects, r.apply(f, f.prog.Handle(proto.ResizeEvent{Width: f.areaW, Height: f.areaH}))...)
		case proto.Redraw:
			effects = append(effects, Redraw{})
		case proto.Think:
			if f.thinkGen != 0 {
				effects = append(effects, CancelThink{Gen: f.thinkGen})
			}
			r.gen++
			f.thinkGen = r.gen
			effects = append(effects, StartThink{Gen: r.gen, Fn: o.Fn, Deadline: r.deadline(o)})
		case proto.Launch:
			effects = append(effects, r.launch(o)...)
			return append(effects, r.apply(f, outs[i+1:])...) // outputs after Launch still apply to f
		case proto.Done:
			return append(effects, r.finish(f, o.Result)...)
		case proto.Quit:
			r.exited = true
			return append(effects, Exit{})
		case proto.Hold:
			effects = append(effects, r.hold(f, o.On)...)
		case proto.Drain:
			r.drains++
			f.drain = r.drains
			effects = append(effects, Drain{ID: f.drain})
		case proto.Skip:
			effects = append(effects, Skip{})
		}
	}
	return effects
}

// hold freezes or releases output for f. Only a change is reported.
func (r *Runner) hold(f *frame, on bool) []Effect {
	switch {
	case on && r.holder == nil:
		r.holder = f
	case !on && r.holder != nil:
		r.holder = nil
	default:
		return nil
	}
	return []Effect{Hold{On: on}}
}

// release ends f's hold when f stops running: a hold never outlives its program.
func (r *Runner) release(f *frame) []Effect {
	if r.holder != f {
		return nil
	}
	return r.hold(f, false)
}

// deadline is a Think's wall-clock bound. Deterministic runs bound searches by Limit, so
// the clock is only a safety cap there; otherwise Budget is the bound.
func (r *Runner) deadline(t proto.Think) time.Duration {
	limited := t.Limit.MaxDepth > 0 || t.Limit.MaxNodes > 0
	if r.cfg.Deterministic && limited {
		return SafetyCap
	}
	return t.Budget
}

func (r *Runner) launch(l proto.Launch) []Effect {
	if r.cfg.Resolve == nil {
		return []Effect{Print{Lines: []string{"** GAME ROUTINE NOT AVAILABLE **"}, Pace: proto.PaceSpeech}}
	}
	prog, place, err := r.cfg.Resolve(l)
	if err != nil {
		return []Effect{Print{Lines: []string{"** GAME ROUTINE NOT AVAILABLE **"}, Pace: proto.PaceSpeech}}
	}
	f := &frame{prog: prog, place: place, launchKey: l, seed: r.launchSeed(l.Slug)}
	f.areaW, f.areaH = r.area(place)
	r.stack = append(r.stack, f)
	effects := []Effect{Relayout{Placement: place, NewProgram: true}}
	if place.Layout != proto.LayoutConsole {
		// A program with its own screen starts on a new page, so its console strip shows
		// only its own text, not the end of the last game and the menu that chose this one.
		effects = append(effects, PageBreak{})
	}
	return append(effects, r.apply(f, prog.Start(r.env(f, l.Mode)))...)
}

// finish handles Done from f.
func (r *Runner) finish(f *frame, res proto.Result) []Effect {
	if f.isRoot {
		r.exited = true
		return []Effect{Exit{}}
	}
	if f != r.top() {
		panic(fmt.Sprintf("host: Done from a program that is not running (%T)", f.prog))
	}
	if res.Next != nil { // hand-off: replace this program with the next, without a verdict
		effects := r.release(f)
		if f.thinkGen != 0 {
			effects = append(effects, CancelThink{Gen: f.thinkGen})
		}
		r.stack = r.stack[:len(r.stack)-1]
		depth := len(r.stack)
		effects = append(effects, r.launch(*res.Next)...)
		if len(r.stack) == depth { // the next program could not be built: report to the program below
			res.Next = nil
			below := r.top()
			effects = append(effects, Relayout{Placement: below.place, NewProgram: true})
			return append(effects, r.apply(below, below.prog.Handle(proto.GameOver{Result: res}))...)
		}
		return effects
	}
	return r.pop(res)
}

// pop removes the running game and reports its result to the program below.
func (r *Runner) pop(res proto.Result) []Effect {
	f := r.top()
	effects := r.release(f)
	if f.thinkGen != 0 {
		effects = append(effects, CancelThink{Gen: f.thinkGen})
	}
	r.stack = r.stack[:len(r.stack)-1]
	below := r.top()
	effects = append(effects, Relayout{Placement: below.place, NewProgram: true, Last: f.prog, LastPlace: f.place})
	return append(effects, r.apply(below, below.prog.Handle(proto.GameOver{Result: res}))...)
}
