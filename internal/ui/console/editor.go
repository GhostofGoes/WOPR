package console

import "strings"

// Editor is the single-line input editor: typed text is echoed as typed (mixed case, as in
// the film); history is on Up/Down; Ctrl+U clears. The cursor is always at the end, as on
// the film's terminal.
type Editor struct {
	text    []string // grapheme clusters
	history []string
	hpos    int    // index into history while browsing; len(history) when not
	draft   string // what was being typed before browsing history
}

// Insert adds sanitised text at the end, respecting MaxInput.
func (e *Editor) Insert(s string) {
	s = SanitizeInput(s)
	for _, g := range Graphemes(s) {
		if len(e.text) >= MaxInput {
			return
		}
		e.text = append(e.text, g)
	}
	e.hpos = len(e.history)
}

// Backspace removes the last grapheme cluster.
func (e *Editor) Backspace() {
	if len(e.text) > 0 {
		e.text = e.text[:len(e.text)-1]
	}
}

// Clear empties the line (Ctrl+U).
func (e *Editor) Clear() { e.text = e.text[:0] }

// Value is the current text.
func (e *Editor) Value() string { return strings.Join(e.text, "") }

// Empty reports whether nothing has been typed.
func (e *Editor) Empty() bool { return len(e.text) == 0 }

// Submit returns the line, records it in history when not blank, and clears the editor.
func (e *Editor) Submit() string {
	v := e.Value()
	if t := strings.TrimSpace(v); t != "" && (len(e.history) == 0 || e.history[len(e.history)-1] != t) {
		e.history = append(e.history, t)
	}
	e.text = e.text[:0]
	e.hpos = len(e.history)
	e.draft = ""
	return strings.TrimSpace(v)
}

// HistoryPrev shows the previous history entry (Up).
func (e *Editor) HistoryPrev() {
	if e.hpos == 0 {
		return
	}
	if e.hpos == len(e.history) {
		e.draft = e.Value()
	}
	e.hpos--
	e.set(e.history[e.hpos])
}

// HistoryNext shows the next history entry, or the draft at the end (Down).
func (e *Editor) HistoryNext() {
	if e.hpos >= len(e.history) {
		return
	}
	e.hpos++
	if e.hpos == len(e.history) {
		e.set(e.draft)
		return
	}
	e.set(e.history[e.hpos])
}

func (e *Editor) set(s string) { e.text = append(e.text[:0], Graphemes(s)...) }
