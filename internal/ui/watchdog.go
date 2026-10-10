package ui

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"

	"github.com/GhostofGoes/WOPR/internal/debuglog"
)

// A program's Handle runs on Bubble Tea's event loop, so one that never returns (a bug)
// would freeze wopr for good: Ctrl+C is only a key in raw mode, and SIGINT and SIGTERM
// become messages queued behind the stuck Update. The watchdog notices an Update or View
// that runs too long, or a signal that arrives while one is running, and ends the process
// itself with the terminal restored. Nothing legitimate is slow there: slow work is a
// Think, off the event loop.

const (
	stuckAfter  = 10 * time.Second // an Update or View this long is a bug
	signalGrace = time.Second      // after SIGINT or SIGTERM, how long a running Update may finish
	killGrace   = 2 * time.Second  // how long Bubble Tea's own Kill may take before the fallback
)

// watchdog times each Update and View. Its zero value is not usable; see newWatchdog.
type watchdog struct {
	start, end chan struct{}
	signals    <-chan os.Signal
	stuck      time.Duration
	grace      time.Duration
	trip       func(reason string) // ends the process; tests replace it
}

func newWatchdog(signals <-chan os.Signal, trip func(string)) *watchdog {
	return &watchdog{
		start: make(chan struct{}), end: make(chan struct{}), signals: signals,
		stuck: stuckAfter, grace: signalGrace, trip: trip,
	}
}

// watch runs until it trips; start it on its own goroutine.
func (w *watchdog) watch() {
	for {
		select {
		case <-w.start:
		case <-w.signals: // idle: Bubble Tea handles the signal itself
			continue
		}
		if reason := w.busy(); reason != "" {
			w.trip(reason)
			return
		}
	}
}

// busy waits for the running Update or View to end, and says why not if it does not.
func (w *watchdog) busy() string {
	ctx, cancel := context.WithTimeout(context.Background(), w.stuck)
	defer cancel()
	select {
	case <-w.end:
		return ""
	case <-ctx.Done():
		return fmt.Sprintf("it stopped responding for %v", w.stuck)
	case s := <-w.signals:
		grace, cancelGrace := context.WithTimeout(context.Background(), w.grace)
		defer cancelGrace()
		select {
		case <-w.end:
			return "" // Bubble Tea has the signal queued and will act on it
		case <-grace.Done():
			return fmt.Sprintf("it was not responding when it got %v", s)
		}
	}
}

// do runs f as one timed step of the event loop.
func (w *watchdog) do(f func()) {
	w.start <- struct{}{}
	defer func() { w.end <- struct{}{} }()
	f()
}

// watched is the model Bubble Tea runs: the real one, timed by the watchdog.
type watched struct {
	m tea.Model
	w *watchdog
}

func (v watched) Init() tea.Cmd {
	var cmd tea.Cmd
	v.w.do(func() { cmd = v.m.Init() })
	return cmd
}

func (v watched) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	v.w.do(func() { v.m, cmd = v.m.Update(msg) })
	return v, cmd
}

func (v watched) View() tea.View {
	var view tea.View
	v.w.do(func() { view = v.m.View() })
	return view
}

// runWatched runs p with m under a watchdog that, if it trips, restores the terminal,
// says why on stderr and exits with status 1. Run cannot return then: its goroutine is
// the one that is stuck.
func runWatched(m tea.Model, lg *debuglog.Log, opts ...tea.ProgramOption) (tea.Model, error) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sig)

	var p *tea.Program
	saved := saveTerminal()
	w := newWatchdog(sig, func(reason string) {
		lg.Printf("watchdog: %s", reason)
		restoreAfterHang(p, saved, os.Stdout)
		_, _ = fmt.Fprintf(os.Stderr, "\r\nwopr: %s, so it quit. This is a bug; please report it.\r\n", reason)
		_ = lg.Close()
		os.Exit(1)
	})
	p = tea.NewProgram(watched{m: m, w: w}, opts...)
	go w.watch()
	out, err := p.Run()
	if v, ok := out.(watched); ok {
		out = v.m
	}
	return out, err
}

// savedTerminal is the terminal's state before Bubble Tea made it raw, for the fallback.
type savedTerminal struct {
	fd    uintptr
	state *term.State
}

func saveTerminal() savedTerminal {
	fd := os.Stdin.Fd()
	if !term.IsTerminal(fd) {
		return savedTerminal{}
	}
	st, err := term.GetState(fd)
	if err != nil {
		return savedTerminal{}
	}
	return savedTerminal{fd: fd, state: st}
}

// restoreAfterHang asks Bubble Tea to kill the program, which restores the terminal. Kill
// can block behind the stuck event loop (a signal handler waiting to deliver its
// message), so after killGrace it undoes raw mode and the screen modes wopr uses itself.
func restoreAfterHang(p *tea.Program, saved savedTerminal, out io.Writer) {
	done := make(chan struct{})
	go func() {
		p.Kill()
		close(done)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), killGrace)
	defer cancel()
	select {
	case <-done:
		return
	case <-ctx.Done():
	}
	if saved.state != nil {
		_ = term.Restore(saved.fd, saved.state)
	}
	_, _ = io.WriteString(out, ansi.ResetModifyOtherKeys+ansi.PopKittyKeyboard(1)+
		ansi.ResetModeAltScreenSaveCursor+ansi.ShowCursor+ansi.ResetModeBracketedPaste)
}
