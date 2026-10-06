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
}

type itemKind uint8

const (
	itemLine   itemKind = iota // reveal one line at a pace
	itemPause                  // wait, skippable
	itemPage                   // page break
	itemPrompt                 // the input line becomes active
)

// Event is something the typewriter reached, reported by Advance and Flush.
type Event struct {
	Prompt     bool   // an input prompt became active
	PromptText string // its text ("LOGON: ", or "" for free input)
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

	currentPace proto.Pace
}

// SetInstant turns pacing off (--instant, tests) or on.
func (t *Typewriter) SetInstant(on bool) { t.instant = on }

// Say queues lines at a pace.
func (t *Typewriter) Say(lines []string, st proto.Style, pace proto.Pace) {
	for _, l := range lines {
		t.queue = append(t.queue, item{kind: itemLine, text: l, style: st, pace: pace})
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

// Busy reports whether anything is still being revealed or waited for.
func (t *Typewriter) Busy() bool { return t.revealing || t.waiting > 0 || len(t.queue) > 0 }

// Revealing reports whether output is being revealed or paused over, rather than only
// the prompt waiting to appear. That is when a key press counts as "skip".
func (t *Typewriter) Revealing() bool {
	return t.revealing || t.waiting > 0 || (len(t.queue) > 0 && t.queue[0].kind != itemPrompt)
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
		sb.appendHidden(it.text, it.style)
		t.revealing = true
		t.shown = 0
		t.total = len(Graphemes(it.text))
		t.currentPace = it.pace
		t.credit = 0
		if t.total == 0 {
			t.finishLine(sb)
		}
	case itemPause:
		t.waiting = it.pause
	case itemPage:
		sb.PageBreak()
	case itemPrompt:
		return Event{Prompt: true, PromptText: it.prompt}, true
	}
	return Event{}, false
}

func (t *Typewriter) finishLine(sb *Scrollback) {
	if l := sb.last(); l != nil && t.revealing {
		l.shown = -1
	}
	t.revealing = false
}
