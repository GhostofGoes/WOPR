package console

import (
	"github.com/charmbracelet/x/ansi"

	"github.com/GhostofGoes/WOPR/internal/proto"
)

// MaxLines caps the scrollback, in logical lines.
const MaxLines = 2000

// Line is one logical line of console text.
type Line struct {
	Text  string
	Style proto.Style
	shown int // graphemes revealed so far; -1 means all
}

// Visible returns the part of the line the typewriter has revealed.
func (l Line) Visible() string {
	if l.shown < 0 {
		return l.Text
	}
	gs := Graphemes(l.Text)
	if l.shown >= len(gs) {
		return l.Text
	}
	var out string
	for _, g := range gs[:l.shown] {
		out += g
	}
	return out
}

// Row is one rendered row.
type Row struct {
	Text  string
	Style proto.Style
}

// Scrollback is the console's history, split into pages by Clear.
type Scrollback struct {
	lines     []Line
	pageStart int // index of the first line of the current page
	offset    int // rows scrolled up from the bottom; 0 follows new output
}

// Append adds a fully revealed line.
func (s *Scrollback) Append(text string, st proto.Style) {
	s.add(Line{Text: text, Style: st, shown: -1})
}

// appendHidden adds a line the typewriter will reveal; it is always the newest line.
func (s *Scrollback) appendHidden(text string, st proto.Style) {
	s.add(Line{Text: text, Style: st, shown: 0})
}

func (s *Scrollback) add(l Line) {
	s.lines = append(s.lines, l)
	if over := len(s.lines) - MaxLines; over > 0 {
		s.lines = append(s.lines[:0], s.lines[over:]...)
		s.pageStart = max(s.pageStart-over, 0)
	}
	s.offset = 0
}

// last returns the most recent line, for the typewriter to reveal into.
func (s *Scrollback) last() *Line {
	if len(s.lines) == 0 {
		return nil
	}
	return &s.lines[len(s.lines)-1]
}

// PageBreak starts a new page: the screen restarts at the top; earlier pages stay
// reachable by scrolling up.
func (s *Scrollback) PageBreak() {
	s.pageStart = len(s.lines)
	s.offset = 0
}

// Scroll moves the view by delta rows (positive = back in history). It is clamped when
// rendering.
func (s *Scrollback) Scroll(delta int) { s.offset = max(s.offset+delta, 0) }

// Following reports whether the view follows new output.
func (s *Scrollback) Following() bool { return s.offset == 0 }

// Lines returns the logical lines (for tests and transcripts).
func (s *Scrollback) Lines() []Line { return s.lines }

// Render returns exactly h rows for a w-column area. extra rows (the active input line)
// follow the current page. A page that fits starts at the top of the screen; otherwise the
// newest rows are shown, or older ones when the view is scrolled up. Lines are wrapped
// lazily from the newest backwards, so rendering cost does not grow with history.
func (s *Scrollback) Render(w, h int, method ansi.Method, extra ...Row) []Row {
	if h <= 0 {
		return nil
	}
	out := make([]Row, h)
	need := h + s.offset
	// Rows of the current page, newest last, collected backwards.
	var rev []Row
	for i := len(extra) - 1; i >= 0; i-- {
		rev = append(rev, extra[i])
	}
	pageRows := len(extra)
	i := len(s.lines) - 1
	for ; i >= s.pageStart && len(rev) < need+1; i-- {
		rows := wrapRows(s.lines[i], w, method)
		for j := len(rows) - 1; j >= 0; j-- {
			rev = append(rev, rows[j])
		}
		pageRows += len(rows)
	}
	pageComplete := i < s.pageStart
	if s.offset == 0 && pageComplete && pageRows <= h {
		for k := range pageRows { // the page starts at the top
			out[k] = rev[pageRows-1-k]
		}
		return out
	}
	for ; i >= 0 && len(rev) < need; i-- { // continue into earlier pages
		rows := wrapRows(s.lines[i], w, method)
		for j := len(rows) - 1; j >= 0; j-- {
			rev = append(rev, rows[j])
		}
	}
	offset := min(s.offset, max(len(rev)-h, 0))
	s.offset = offset
	// rev[offset] is the bottom row of the view.
	for k := range h {
		src := offset + (h - 1 - k)
		if src < len(rev) {
			out[k] = rev[src]
		}
	}
	return out
}

func wrapRows(l Line, w int, method ansi.Method) []Row {
	text := l.Visible()
	var rows []Row
	for _, r := range Wrap(text, w, method) {
		rows = append(rows, Row{Text: r, Style: l.Style})
	}
	return rows
}
