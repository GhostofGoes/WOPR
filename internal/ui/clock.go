package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// The single clock (docs/PLAN.md §4.3). This is the only file that may call tea.Tick
// (forbidigo). A tick carries a generation: re-arming bumps it, so stale ticks are
// dropped and at most one is in flight. When nothing is busy, no timer runs.

// tickInterval is the clock period while anything is animating (~30 fps).
const tickInterval = 33 * time.Millisecond

type tickMsg struct {
	gen uint64
	at  time.Time
}

// Scheduler arms one tick. The default uses tea.Tick; tests substitute a fake that
// delivers synthetic ticks without waiting.
type Scheduler func(d time.Duration, gen uint64) tea.Cmd

func teaScheduler(d time.Duration, gen uint64) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return tickMsg{gen: gen, at: t} })
}

type clock struct {
	schedule Scheduler
	gen      uint64
	armed    bool
	last     time.Time
}

// arm schedules a tick unless one is already pending.
func (c *clock) arm(now time.Time) tea.Cmd {
	if c.armed {
		return nil
	}
	c.armed = true
	c.gen++
	if c.last.IsZero() {
		c.last = now
	}
	return c.schedule(tickInterval, c.gen)
}

// accept returns the elapsed time for a tick, or false if the tick is stale.
func (c *clock) accept(m tickMsg) (time.Duration, bool) {
	if m.gen != c.gen || !c.armed {
		return 0, false
	}
	c.armed = false
	dt := m.at.Sub(c.last)
	c.last = m.at
	if dt < 0 {
		dt = 0
	}
	return dt, true
}

// idle forgets the last tick time, so the next busy period does not inherit a huge dt.
func (c *clock) idle() { c.last = time.Time{} }
