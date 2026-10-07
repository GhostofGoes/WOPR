package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// driver runs the model synchronously, without a terminal or real time (docs/PLAN.md §9).
// A fake Scheduler records each armed tick; step delivers it after advancing a fake clock.
// Commands run inline: tea.BatchMsg is unwrapped, a Think runs to completion on the spot,
// and Quit or Interrupt ends the session. It lives in package ui because the model is
// unexported.
type driver struct {
	t       *testing.T
	m       *model
	now     time.Time
	pending []armed
	queue   []tea.Msg
	ended   string // "", "quit" or "interrupt"

	slowThinks bool      // hold Think results until finishThinks, as if the search were slow
	thinkDone  []tea.Msg // the held results
}

type armed struct {
	d   time.Duration
	gen uint64
}

// maxSteps bounds settle: a session that never goes idle is a bug.
const maxSteps = 100_000

func newDriver(t *testing.T, opts Options, w, h int) *driver {
	t.Helper()
	d := &driver{t: t, now: time.Date(1983, 6, 3, 9, 0, 0, 0, time.UTC)}
	d.m = newModel(opts, func(dur time.Duration, gen uint64) tea.Cmd {
		d.pending = append(d.pending, armed{dur, gen})
		return nil
	})
	d.m.now = func() time.Time { return d.now }
	d.resize(w, h)
	return d
}

func (d *driver) send(msg tea.Msg) *driver {
	d.t.Helper()
	d.queue = append(d.queue, msg)
	for len(d.queue) > 0 && d.ended == "" {
		next := d.queue[0]
		d.queue = d.queue[1:]
		_, cmd := d.m.Update(next)
		d.exec(cmd)
	}
	return d
}

func (d *driver) exec(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	switch msg := cmd().(type) {
	case nil, tea.SuspendMsg:
	case tea.BatchMsg:
		for _, c := range msg {
			d.exec(c)
		}
	case tea.QuitMsg:
		d.ended = "quit"
	case tea.InterruptMsg:
		d.ended = "interrupt"
	case thinkDoneMsg:
		if d.slowThinks {
			d.thinkDone = append(d.thinkDone, msg)
			return
		}
		d.queue = append(d.queue, msg)
	default:
		d.queue = append(d.queue, msg)
	}
}

// finishThinks delivers the held Think results.
func (d *driver) finishThinks() *driver {
	held := d.thinkDone
	d.thinkDone = nil
	for _, msg := range held {
		d.send(msg)
	}
	return d.settle()
}

func (d *driver) resize(w, h int) *driver {
	return d.send(tea.WindowSizeMsg{Width: w, Height: h})
}

// step delivers the oldest armed tick, if any, and reports whether there was one.
func (d *driver) step() bool {
	if len(d.pending) == 0 || d.ended != "" {
		return false
	}
	a := d.pending[0]
	d.pending = d.pending[1:]
	d.now = d.now.Add(a.d)
	d.send(tickMsg{gen: a.gen, at: d.now})
	return true
}

// steps delivers up to n ticks.
func (d *driver) steps(n int) *driver {
	for range n {
		if !d.step() {
			break
		}
	}
	return d
}

// settle runs the clock until nothing is armed.
func (d *driver) settle() *driver {
	d.t.Helper()
	for i := 0; d.step(); i++ {
		if i > maxSteps {
			d.t.Fatalf("the clock never went idle:\n%s", d.screen())
		}
	}
	return d
}

func (d *driver) key(k tea.KeyPressMsg) *driver { return d.send(k).settle() }

// press types text one key at a time without moving the clock.
func (d *driver) press(text string) *driver {
	for _, r := range text {
		if r == ' ' {
			d.send(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
			continue
		}
		d.send(keyRune(r))
	}
	return d
}

// typ types text, then settles.
func (d *driver) typ(text string) *driver { return d.press(text).settle() }

func keyRune(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

// line types text and presses Enter.
func (d *driver) line(text string) *driver { return d.typ(text).key(enter) }

var (
	enter = tea.KeyPressMsg{Code: tea.KeyEnter}
	esc   = tea.KeyPressMsg{Code: tea.KeyEscape}
	ctrlC = tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	ctrlD = tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl}
	up    = tea.KeyPressMsg{Code: tea.KeyUp}
	pgUp  = tea.KeyPressMsg{Code: tea.KeyPgUp}
	pgDn  = tea.KeyPressMsg{Code: tea.KeyPgDown}
)

// screen is the plain text of the current frame, trailing spaces trimmed, with the cursor.
func (d *driver) screen() string {
	v := d.m.View()
	lines := strings.Split(ansi.Strip(v.Content), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " ")
	}
	cursor := "cursor: hidden"
	if v.Cursor != nil {
		cursor = fmt.Sprintf("cursor: col %d, row %d", v.Cursor.X, v.Cursor.Y)
	}
	return strings.Join(lines, "\n") + "\n" + cursor + "\n"
}

// snapshots collects labelled screens for one golden file.
type snapshots struct{ b strings.Builder }

func (s *snapshots) add(label string, d *driver) {
	fmt.Fprintf(&s.b, "==== %s (%dx%d) ====\n%s", label, d.m.w, d.m.h, d.screen())
}

func (s *snapshots) String() string { return s.b.String() }
