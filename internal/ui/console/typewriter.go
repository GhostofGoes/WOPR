package console

import (
	"time"

	"github.com/GhostofGoes/WOPR/internal/proto"
)

// item is one step of the typewriter queue.
type item struct {
	kind   itemKind
	text   string
	style  proto.Style
	pace   proto.Pace
	pause  time.Duration
	prompt string
	mark   uint64
	open   bool // the line stays open: the next line continues it
}

type itemKind uint8

const (
	itemLine   itemKind = iota // reveal one line at a pace
	itemPause                  // wait, skippable
	itemPage                   // page break
	itemPrompt                 // the input line becomes active
	itemMark                   // a caller's marker, reported when reached
	itemDrain                  // a program's Drain marker, reported when reached
)

// Event is something the typewriter reached, reported by Advance and Flush.
type Event struct {
	Prompt     bool   // an input prompt became active
	PromptText string // its text ("LOGON: ", or "" for free input)
	Mark       uint64 // a marker queued with Mark was reached (0: none)
	Drain      uint64 // a marker queued with Drain was reached (0: none)
}

// Typewriter reveals queued output at modem speed. Items run strictly in order, so a
// program's Say, Wait, Clear and Prompt appear exactly as sequenced.
type Typewriter struct {
	queue     []item
	revealing bool    // the newest scrollback line is being revealed
	shown     int     // graphemes of it shown so far
	total     int     // graphemes in it
	credit    float64 // graphemes owed but not yet shown
	waiting   time.Duration
	instant   bool
	open      bool   // the newest line was left open (SayOpen); the next line continues it
	openAt    uint64 // the scrollback's line count when it was opened

	currentPace proto.Pace
}

// SetInstant turns pacing off (--instant, tests) or on.
func (t *Typewriter) SetInstant(on bool) { t.instant = on }

// Say queues lines at a pace.
func (t *Typewriter) Say(lines []string, st proto.Style, pace proto.Pace) {
	t.say(lines, st, pace, false)
}

// SayOpen is Say with the last line left open: the next line queued continues it on the same
// row (a simulated user's keystrokes, movie mode).
func (t *Typewriter) SayOpen(lines []string, st proto.Style, pace proto.Pace) {
	t.say(lines, st, pace, true)
}

func (t *Typewriter) say(lines []string, st proto.Style, pace proto.Pace, open bool) {
	for i, l := range lines {
		t.queue = append(t.queue, item{kind: itemLine, text: l, style: st, pace: pace, open: open && i == len(lines)-1})
	}
}

// Pause queues a skippable pause.
func (t *Typewriter) Pause(d time.Duration) {
	t.queue = append(t.queue, item{kind: itemPause, pause: d})
}

// Page queues a page break.
func (t *Typewriter) Page() { t.queue = append(t.queue, item{kind: itemPage}) }

// Prompt queues the moment the input line becomes active.
func (t *Typewriter) Prompt(text string) {
	t.queue = append(t.queue, item{kind: itemPrompt, prompt: text})
}

// Mark queues a marker (non-zero), reported as an Event when everything queued before it
// has been shown: the UI changes the layout there, after the old program's last words.
func (t *Typewriter) Mark(id uint64) {
	t.queue = append(t.queue, item{kind: itemMark, mark: id})
}

// Drain queues a program's marker (non-zero), reported as an Event when everything queued
// before it has been shown and waited out.
func (t *Typewriter) Drain(id uint64) {
	t.queue = append(t.queue, item{kind: itemDrain, mark: id})
}

// Busy reports whether anything is still being revealed or waited for.
func (t *Typewriter) Busy() bool { return t.revealing || t.waiting > 0 || len(t.queue) > 0 }

// Revealing reports whether output is being revealed or paused over, rather than only
// the prompt waiting to appear. That is when a key press counts as "skip".
func (t *Typewriter) Revealing() bool {
	if t.revealing || t.waiting > 0 || len(t.queue) == 0 {
		return t.revealing || t.waiting > 0
	}
	switch t.queue[0].kind {
	case itemPrompt, itemMark, itemDrain:
		return false
	case itemLine, itemPause, itemPage:
	}
	return true
}

// Advance moves the typewriter forward by dt and returns the events reached.
func (t *Typewriter) Advance(dt time.Duration, sb *Scrollback) []Event {
	if t.instant {
		return t.Flush(sb)
	}
	var events []Event
	budget := dt
	for budget > 0 || !t.revealing && len(t.queue) > 0 && t.queue[0].kind != itemPause {
		if t.revealing {
			cps := t.currentPace.CharsPerSecond()
			if cps == 0 {
				t.finishLine(sb)
				continue
			}
			t.credit += budget.Seconds() * cps
			budget = 0
			n := int(t.credit)
			t.credit -= float64(n)
			t.shown = min(t.shown+n, t.total)
			if t.shown >= t.total {
				t.finishLine(sb)
				continue // leftover credit is dropped; the next line starts fresh
			}
			if l := sb.last(); l != nil {
				l.shown = t.shown
			}
			break
		}
		if t.waiting > 0 {
			if budget >= t.waiting {
				budget -= t.waiting
				t.waiting = 0
				continue
			}
			t.waiting -= budget
			budget = 0
			break
		}
		if len(t.queue) == 0 {
			break
		}
		if ev, ok := t.start(sb); ok {
			events = append(events, ev)
		}
	}
	return events
}

// Flush reveals everything queued immediately (a key press skips the typewriter) and
// returns the events reached.
func (t *Typewriter) Flush(sb *Scrollback) []Event {
	var events []Event
	for t.Busy() {
		if t.revealing {
			t.finishLine(sb)
			continue
		}
		t.waiting = 0
		if len(t.queue) == 0 {
			break
		}
		if ev, ok := t.start(sb); ok {
			events = append(events, ev)
		}
	}
	return events
}

// start begins the next queued item.
func (t *Typewriter) start(sb *Scrollback) (Event, bool) {
	it := t.queue[0]
	t.queue = t.queue[1:]
	switch it.kind {
	case itemLine:
		if l := sb.last(); t.open && l != nil && sb.added == t.openAt { // continue the open line
			t.shown = len(Graphemes(l.Text))
			l.Text += it.text
			l.shown = t.shown
		} else {
			sb.appendHidden(it.text, it.style)
			t.shown = 0
		}
		t.open, t.openAt = it.open, sb.added
		t.revealing = true
		t.total = len(Graphemes(sb.last().Text))
		t.currentPace = it.pace
		t.credit = 0
		if t.shown >= t.total {
			t.finishLine(sb)
		}
	case itemPause:
		t.waiting = it.pause
	case itemPage:
		sb.PageBreak()
		t.open = false
	case itemPrompt:
		return Event{Prompt: true, PromptText: it.prompt}, true
	case itemMark:
		return Event{Mark: it.mark}, true
	case itemDrain:
		return Event{Drain: it.mark}, true
	}
	return Event{}, false
}

func (t *Typewriter) finishLine(sb *Scrollback) {
	if l := sb.last(); l != nil && t.revealing {
		l.shown = -1
	}
	t.revealing = false
}
