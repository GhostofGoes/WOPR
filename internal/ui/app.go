// Package ui is the Bubble Tea application: the only package that imports Bubble Tea.
// It adapts the host runner (internal/proto/host) and the console (internal/ui/console)
// to the terminal: input, the single clock, layout and rendering.
package ui

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"

	"github.com/GhostofGoes/WOPR/internal/debuglog"
	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/movie"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/proto/host"
	"github.com/GhostofGoes/WOPR/internal/theme"
	"github.com/GhostofGoes/WOPR/internal/ui/console"
	"github.com/GhostofGoes/WOPR/internal/wopr"
)

// Minimum terminal size.
const (
	MinWidth  = 80
	MinHeight = 24
)

// Options configure a session.
type Options struct {
	Theme        string
	Instant      bool
	Seed         uint64
	SeedSet      bool
	ReduceMotion bool
	Play         string // game slug to start directly
	Movie        bool   // movie mode: the director replays the film's scenes instead of the persona
	Scene        string // with Movie: the scene to play from (a slug); "" opens the scene menu
	NoColor      bool   // NO_COLOR set to any non-empty value (no-color.org)
	Panel        Panel  // the front-panel row
	Registry     *games.Registry
	Log          *debuglog.Log // nil: no debug log
}

// Panel says whether the front-panel row is shown.
type Panel uint8

// Panel settings.
const (
	PanelDefault Panel = iota // the theme decides: norad shows it
	PanelOn
	PanelOff
)

// Outcome is how a session ended, for the caller's exit code.
type Outcome int

// Outcomes.
const (
	Finished    Outcome = iota // LOGOFF, Ctrl+D or SIGTERM
	Interrupted                // Ctrl+C or SIGINT
	Panicked                   // the terminal is restored and the stack is on stderr
	NoTerminal                 // no terminal could be opened
	Failed                     // any other error
)

// Run runs the TUI until the user leaves.
func Run(opts Options) (Outcome, error) {
	var teaOpts []tea.ProgramOption
	if opts.NoColor { // colorprofile alone parses NO_COLOR with ParseBool and ignores NO_COLOR=yes
		teaOpts = append(teaOpts, tea.WithColorProfile(colorprofile.Ascii))
	}
	_, err := tea.NewProgram(newModel(opts, teaScheduler), teaOpts...).Run()
	return classify(err), err
}

// classify maps Bubble Tea's error. ErrInterrupted comes first: both it and
// ErrProgramPanic wrap ErrProgramKilled.
func classify(err error) Outcome {
	switch {
	case err == nil:
		return Finished
	case errors.Is(err, tea.ErrInterrupted):
		return Interrupted
	case errors.Is(err, tea.ErrProgramPanic):
		return Panicked
	case strings.Contains(err.Error(), "TTY"):
		return NoTerminal
	default:
		return Failed
	}
}

// TerminalProblem explains why the TUI cannot start here, or returns "". A redirected
// stdin is fine: Bubble Tea opens the controlling terminal itself.
func TerminalProblem(getenv func(string) string) string {
	switch {
	case getenv("TERM") == "dumb":
		return "TERM=dumb: this terminal cannot run the full-screen interface (try wopr --games)"
	case !term.IsTerminal(os.Stdout.Fd()):
		return "standard output is not a terminal"
	}
	return ""
}

type thinkDoneMsg struct {
	gen   uint64
	value any
	err   error
}

type model struct {
	opts    Options
	th      *theme.Theme
	profile colorprofile.Profile
	method  ansi.Method
	w, h    int

	runner  *host.Runner
	started bool
	place   host.Placement

	sb         console.Scrollback
	tw         console.Typewriter
	ed         console.Editor
	asking     bool           // the console prompt is active
	prompt     string         // its text
	held       bool           // Enter pressed before the prompt was active
	keyHint    string         // what the keys do, shown in key mode
	heldThinks []thinkDoneMsg // results that arrived while the TOO SMALL card was up

	// The game that just ended, still on screen until its last words are out.
	last      proto.Program
	nextPlace host.Placement
	markID    uint64
	keyMode   bool
	notice    string

	frozen   bool     // a program's Hold: nothing advances, as behind the TOO SMALL card
	drains   []uint64 // Drain markers the typewriter has reached, to report in order
	draining bool     // drain is running; nested calls leave the work to it

	testRoot proto.Program // tests only: run this as the root program instead

	thinks map[uint64]context.CancelFunc
	clk    clock
	phase  time.Duration // animation phase for blink and the thinking indicator
	now    func() time.Time
}

func newModel(opts Options, sched Scheduler) *model {
	th, ok := theme.Get(opts.Theme)
	if !ok {
		th, _ = theme.Get(theme.Default)
	}
	m := &model{
		opts: opts, th: th, profile: colorprofile.TrueColor, method: ansi.WcWidth,
		thinks: map[uint64]context.CancelFunc{}, clk: clock{schedule: sched}, now: time.Now,
	}
	m.tw.SetInstant(opts.Instant)
	return m
}

func (m *model) Init() tea.Cmd { return nil }

// tooSmall reports whether the terminal is below the minimum size.
func (m *model) tooSmall() bool { return m.w < MinWidth || m.h < MinHeight }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.opts.Log.Printf("terminal %dx%d", m.w, m.h)
		if !m.tooSmall() {
			if !m.started {
				cmds = append(cmds, m.start())
			} else {
				cmds = append(cmds, m.applyAll(m.runner.Resize()))
			}
			for _, d := range m.heldThinks {
				cmds = append(cmds, m.applyAll(m.runner.ThinkResult(d.gen, d.value, d.err)))
			}
			m.heldThinks = nil
		}
	case tea.ColorProfileMsg:
		m.profile = msg.Profile
		m.opts.Log.Printf("colour profile %v, theme %s", msg.Profile, m.th.Name)
	case tea.KeyPressMsg:
		cmds = append(cmds, m.key(msg))
	case tea.PasteMsg:
		if !m.tooSmall() && !m.keyMode && (m.runner == nil || !m.runner.Shielded()) {
			m.ed.Insert(msg.Content)
		}
	case tea.ModeReportMsg:
		// Bubble Tea switches its renderer to grapheme widths on these answers (tea.go); the
		// wrapper and the editor must measure the same way.
		if v := msg.Value; msg.Mode == ansi.ModeUnicodeCore && (v == ansi.ModeReset || v == ansi.ModeSet || v == ansi.ModePermanentlySet) {
			m.method = ansi.GraphemeWidth
			m.opts.Log.Printf("grapheme widths (mode 2027 %v)", msg.Value)
		}
	case tickMsg:
		if dt, ok := m.clk.accept(msg); ok {
			cmds = append(cmds, m.tick(dt))
		}
	case thinkDoneMsg:
		delete(m.thinks, msg.gen)
		if m.tooSmall() {
			m.heldThinks = append(m.heldThinks, msg) // held until the size is valid again
			break
		}
		if m.runner != nil {
			cmds = append(cmds, m.applyAll(m.runner.ThinkResult(msg.gen, msg.value, msg.err)))
		}
	}
	cmds = append(cmds, m.release(), m.drain(), m.maybeArm())
	return m, tea.Batch(cmds...)
}

// start runs the root program once the terminal has a usable size: the persona, or in movie
// mode the director, which pins its own seed and plays deterministically (docs/PLAN.md §7).
func (m *model) start() tea.Cmd {
	m.started = true
	seed, deterministic := m.opts.Seed, m.opts.SeedSet
	var root proto.Program = wopr.New(m.opts.Registry, nil, wopr.Options{Play: m.opts.Play})
	if m.opts.Movie {
		seed, deterministic = movie.Seed, true
		root = movie.New(movie.Options{Scene: m.opts.Scene})
		m.opts.Log.Printf("movie mode from scene %q, seed %d", m.opts.Scene, seed)
	}
	if m.testRoot != nil {
		root = m.testRoot
	}
	m.runner = host.New(host.Config{
		Seed: seed, Instant: m.opts.Instant, Deterministic: deterministic, ReduceMotion: m.opts.ReduceMotion,
		Resolve: m.resolve, Area: m.area,
	})
	return m.applyAll(m.runner.Start(root, host.Placement{}))
}

func (m *model) resolve(l proto.Launch) (proto.Program, host.Placement, error) {
	if m.opts.Registry == nil {
		return nil, host.Placement{}, games.ErrNotFound
	}
	e, ok := m.opts.Registry.Get(l.Slug)
	if !ok || e.New == nil {
		return nil, host.Placement{}, games.ErrNotFound
	}
	// In movie mode the director's programs (the climax) play through: Esc cannot end them.
	return e.New(), host.Placement{Layout: e.Info.Layout, PanelRows: e.Info.PanelRows, NoAbort: e.Info.Internal || m.opts.Movie}, nil
}

// tick advances the typewriter and the runner.
func (m *model) tick(dt time.Duration) tea.Cmd {
	if m.paused() {
		return nil // nothing advances behind the TOO SMALL card or while a program holds output
	}
	m.phase += dt
	for _, ev := range m.tw.Advance(dt, &m.sb) {
		m.onTypewriter(ev)
	}
	return m.applyAll(m.runner.Advance(dt))
}

func (m *model) onTypewriter(ev console.Event) {
	if ev.Prompt {
		m.asking, m.prompt = true, ev.PromptText
	}
	if ev.Mark != 0 && ev.Mark == m.markID {
		m.place, m.last = m.nextPlace, nil
	}
	if ev.Drain != 0 {
		m.drains = append(m.drains, ev.Drain)
	}
}

// drain reports the Drain markers the typewriter has reached, in order. A report can queue
// more output and, under --instant, reach the next marker at once, so it loops; the applyAll
// calls inside it leave any new markers to this loop.
func (m *model) drain() tea.Cmd {
	if m.draining || m.runner == nil {
		return nil
	}
	m.draining = true
	defer func() { m.draining = false }()
	var cmds []tea.Cmd
	for len(m.drains) > 0 {
		id := m.drains[0]
		m.drains = m.drains[1:]
		cmds = append(cmds, m.applyAll(m.runner.Drained(id)))
	}
	return tea.Batch(cmds...)
}

// paused reports whether time stands still: behind the TOO SMALL card, or held by a program.
func (m *model) paused() bool { return m.tooSmall() || m.frozen }

// relayout applies a placement change. When a game with a board ends, its last View stays
// up until its final output has been revealed (docs/PLAN.md §4.2): the change waits for a
// typewriter mark. Any other change applies at once and cancels a waiting one.
func (m *model) relayout(e host.Relayout) {
	m.markID++
	if e.Last != nil && e.LastPlace.Layout != proto.LayoutConsole && !m.opts.Instant {
		m.last, m.nextPlace = e.Last, e.Placement
		m.tw.Mark(m.markID)
		return
	}
	m.place, m.last = e.Placement, nil
}

// release submits a line whose Enter was pressed before the prompt became active. Update
// calls it after every message: the prompt can arrive on a tick, a key's skip, or an
// instant flush.
func (m *model) release() tea.Cmd {
	if m.held && m.asking && !m.tw.Busy() && !m.tooSmall() {
		m.held = false
		return m.submit()
	}
	return nil
}

// busy reports whether the clock must keep running.
func (m *model) busy() bool {
	if !m.started || m.paused() {
		return false
	}
	return m.tw.Busy() || m.runner.NeedsTicks() || m.runner.Thinking() || len(m.thinks) > 0
}

func (m *model) maybeArm() tea.Cmd {
	if m.busy() {
		return m.clk.arm(m.now())
	}
	if !m.clk.armed {
		m.clk.idle()
	}
	return nil
}

// applyAll applies runner effects in order.
func (m *model) applyAll(effects []host.Effect) tea.Cmd {
	var cmds []tea.Cmd
	for _, e := range effects {
		switch e := e.(type) {
		case host.Print:
			if e.Open {
				m.tw.SayOpen(e.Lines, proto.StyleText, e.Pace)
			} else {
				m.tw.Say(e.Lines, proto.StyleText, e.Pace)
			}
		case host.Pause:
			m.tw.Pause(e.D)
		case host.PageBreak:
			if m.tw.Busy() {
				m.tw.Page() // after what is still being revealed
			} else {
				m.sb.PageBreak() // now: a launch's new layout must not frame the old page until the next tick
			}
		case host.AskLine:
			m.keyMode = false
			m.tw.Prompt(e.Prompt)
		case host.AskKeys:
			m.keyMode = true
			m.asking = false
			m.keyHint = e.Hint
		case host.StartThink:
			cmds = append(cmds, m.think(e))
		case host.CancelThink:
			if cancel, ok := m.thinks[e.Gen]; ok {
				cancel()
				delete(m.thinks, e.Gen)
			}
		case host.Notice:
			m.notice = e.Text
		case host.Relayout:
			m.relayout(e)
			if e.NewProgram { // the old program's prompt and typeahead are not the new one's
				m.asking, m.prompt = false, ""
				if m.held {
					m.held = false
					m.ed.Clear()
				}
			}
		case host.Redraw:
		case host.Exit:
			cmds = append(cmds, tea.Quit)
		case host.Hold:
			m.frozen = e.On
		case host.Drain:
			m.tw.Drain(e.ID)
		case host.Skip:
			for _, ev := range m.tw.Flush(&m.sb) {
				m.onTypewriter(ev)
			}
		}
	}
	if m.opts.Instant && !m.frozen { // held output stays held, as behind the TOO SMALL card
		for _, ev := range m.tw.Flush(&m.sb) {
			m.onTypewriter(ev)
		}
	}
	return tea.Batch(append(cmds, m.drain())...)
}

// think runs a program's slow work off the UI goroutine.
func (m *model) think(t host.StartThink) tea.Cmd {
	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	if t.Deadline > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), t.Deadline)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	m.thinks[t.Gen] = cancel
	fn, gen, lg := t.Fn, t.Gen, m.opts.Log
	return func() tea.Msg {
		defer cancel()
		start := time.Now()
		v, err := fn(ctx)
		lg.Printf("think %d: %v, deadline hit %v, err %v", gen, time.Since(start).Round(time.Millisecond), ctx.Err() == context.DeadlineExceeded, err)
		return thinkDoneMsg{gen: gen, value: v, err: err}
	}
}

// area is the layout area for a placement at the current terminal size.
func (m *model) area(p host.Placement) (int, int) {
	g := m.geometry(p)
	return g.width, g.viewRows
}
