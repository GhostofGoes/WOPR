// Package host runs proto programs. It is the protocol's only interpreter and it does
// not import Bubble Tea: internal/ui adapts it to the terminal, and games/testkit drives
// the same code in tests.
//
// The runner owns the program stack (the persona at the bottom, games launched on top),
// the input mode, the Esc state machine, Think jobs and Animate cadence. Each call
// returns Effects, in order, for the caller to apply.
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

// Print queues lines on the typewriter.
type Print struct {
	Lines []string
	Pace  proto.Pace
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

// Relayout tells the caller the running program's placement changed.
type Relayout struct{ Placement Placement }

// Redraw asks for the running program's View to be drawn again.
type Redraw struct{}

// Exit ends the session normally (exit code 0).
type Exit struct{}

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
}

// Runner interprets the protocol for a stack of programs.
type Runner struct {
	cfg      Config
	stack    []*frame
	gen      uint64
	escArmed time.Duration // time left in the Esc window; 0 = not armed
	exited   bool
}

// New returns a Runner.
func New(cfg Config) *Runner { return &Runner{cfg: cfg} }

// Start runs the root program (the persona, or the movie director).
func (r *Runner) Start(root proto.Program, place Placement) []Effect {
	f := &frame{prog: root, place: place, isRoot: true}
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

// NeedsTicks reports whether Advance must be called (animation or an armed Esc).
func (r *Runner) NeedsTicks() bool {
	return len(r.stack) > 0 && (r.top().every > 0 || r.escArmed > 0)
}

// hostCommands end the session from anywhere, including inside a game.
var hostCommands = map[string]bool{"LOGOFF": true, "LOG OFF": true, "EXIT": true, "QUIT": true}

// Line delivers a submitted line to the running program.
func (r *Runner) Line(text string) []Effect {
	if r.exited || len(r.stack) == 0 {
		return nil
	}
	effects := r.disarm()
	if hostCommands[strings.Join(strings.Fields(strings.ToUpper(text)), " ")] {
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

// Key delivers a key to the running program in key mode.
func (r *Runner) Key(k proto.Key, ch rune) []Effect {
	if r.exited || len(r.stack) == 0 {
		return nil
	}
	effects := r.disarm()
	f := r.top()
	if f.mode != modeKeys {
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
	if r.escArmed == 0 {
		r.escArmed = EscWindow
		return []Effect{Notice{Text: "** PRESS ESC AGAIN TO END GAME **"}}
	}
	r.escArmed = 0
	return append([]Effect{Notice{}}, r.pop(proto.Result{Outcome: proto.Aborted})...)
}

// Advance moves time forward: Animate ticks and the Esc window.
func (r *Runner) Advance(dt time.Duration) []Effect {
	if r.exited || len(r.stack) == 0 {
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

func (r *Runner) env(f *frame, mode string) proto.Env {
	return proto.Env{
		Seed: r.cfg.Seed, Width: f.areaW, Height: f.areaH,
		Instant: r.cfg.Instant, Deterministic: r.cfg.Deterministic, Mode: mode,
	}
}

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
			effects = append(effects, Print{Lines: o.Lines, Pace: o.Pace})
		case proto.Wait:
			if !r.cfg.Instant {
				effects = append(effects, Pause{D: o.D})
			}
		case proto.Clear:
			effects = append(effects, PageBreak{})
		case proto.Prompt:
			f.mode = modeLine
			effects = append(effects, AskLine{Prompt: o.Text})
		case proto.AwaitKeys:
			f.mode = modeKeys
			effects = append(effects, AskKeys{Hint: o.Hint})
		case proto.Animate:
			f.every, f.acc = o.Every, 0
		case proto.SetLayout:
			f.place = Placement{Layout: o.Layout, PanelRows: o.PanelRows}
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
		}
	}
	return effects
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
	f := &frame{prog: prog, place: place, launchKey: l}
	f.areaW, f.areaH = r.area(place)
	r.stack = append(r.stack, f)
	effects := []Effect{Relayout{Placement: place}}
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
		var effects []Effect
		if f.thinkGen != 0 {
			effects = append(effects, CancelThink{Gen: f.thinkGen})
		}
		r.stack = r.stack[:len(r.stack)-1]
		return append(effects, r.launch(*res.Next)...)
	}
	return r.pop(res)
}

// pop removes the running game and reports its result to the program below.
func (r *Runner) pop(res proto.Result) []Effect {
	f := r.top()
	var effects []Effect
	if f.thinkGen != 0 {
		effects = append(effects, CancelThink{Gen: f.thinkGen})
	}
	r.stack = r.stack[:len(r.stack)-1]
	below := r.top()
	effects = append(effects, Relayout{Placement: below.place})
	return append(effects, r.apply(below, below.prog.Handle(proto.GameOver{Result: res}))...)
}
