package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/GhostofGoes/WOPR/internal/proto"
)

// key routes a key press (docs/PLAN.md §4.3, skip policy and Esc machine).
func (m *model) key(k tea.KeyPressMsg) tea.Cmd {
	switch k.String() {
	case "ctrl+c":
		return tea.Interrupt
	case "ctrl+z":
		return tea.Suspend
	}
	if m.tooSmall() || !m.started {
		return nil // dropped: the user cannot see what they would be typing
	}
	var disarm tea.Cmd
	if k.String() != "esc" {
		disarm = m.applyAll(m.runner.Disarm()) // any other key closes the Esc window
	}
	switch k.String() {
	case "pgup":
		m.sb.Scroll(m.consoleRows() - 1)
		return disarm
	case "pgdown":
		m.sb.Scroll(-(m.consoleRows() - 1))
		return disarm
	}

	// A root that captures keys (the movie director) gets every key, Esc included, and no
	// key skips its output; while a program it launched runs, keys do nothing but bring up a
	// notice saying so.
	if m.runner.Captures() {
		if key, r, ok := protoKey(k); ok {
			return tea.Batch(disarm, m.applyAll(m.runner.Key(key, r)))
		}
		return disarm
	}
	if m.runner.Shielded() {
		return tea.Batch(disarm, m.applyAll(m.runner.Refused())) // say why the key did nothing
	}

	// While text is being revealed, the first key skips it, unless a program holds it.
	if m.tw.Revealing() && !m.frozen {
		for _, ev := range m.tw.Flush(&m.sb) {
			m.onTypewriter(ev)
		}
		switch {
		case k.String() == "esc":
			return nil // Esc only skips here
		case m.keyMode:
			// fall through: the key also goes to the game
		case k.String() == "enter" || k.String() == "space":
			if m.ed.Empty() {
				return nil // a pure skip
			}
		}
	}

	if k.String() == "esc" {
		return m.applyAll(m.runner.Esc())
	}
	if m.keyMode {
		if key, r, ok := protoKey(k); ok {
			return m.applyAll(m.runner.Key(key, r))
		}
		return disarm
	}
	return tea.Batch(disarm, m.edit(k))
}

// edit handles keys for the line editor. Typing is kept even while WOPR is busy
// (typeahead); Enter submits only once the prompt is active.
func (m *model) edit(k tea.KeyPressMsg) tea.Cmd {
	switch k.String() {
	case "enter":
		if m.asking && !m.tw.Busy() {
			return m.submit()
		}
		if !m.ed.Empty() {
			m.held = true // typeahead; a bare Enter is only impatience, never an answer
		}
		return nil
	case "backspace":
		m.ed.Backspace()
	case "ctrl+u":
		m.ed.Clear()
	case "up":
		m.ed.HistoryPrev()
	case "down":
		m.ed.HistoryNext()
	case "ctrl+d":
		if m.ed.Empty() {
			return tea.Quit
		}
	case "space":
		m.ed.Insert(" ")
	default:
		if t := k.Text; t != "" {
			m.ed.Insert(t)
		}
	}
	return nil
}

// submit echoes the line the way the film's terminal shows it and hands it to the runner.
func (m *model) submit() tea.Cmd {
	text := m.ed.Submit()
	m.sb.Append(m.prompt+text, proto.StyleText)
	m.sb.Append("", proto.StyleText)
	m.asking, m.prompt = false, ""
	return m.applyAll(m.runner.Line(text))
}

func protoKey(k tea.KeyPressMsg) (proto.Key, rune, bool) {
	switch k.String() {
	case "up":
		return proto.KeyUp, 0, true
	case "down":
		return proto.KeyDown, 0, true
	case "left":
		return proto.KeyLeft, 0, true
	case "right":
		return proto.KeyRight, 0, true
	case "enter":
		return proto.KeyEnter, 0, true
	case "backspace":
		return proto.KeyBackspace, 0, true
	case "esc":
		return proto.KeyEsc, 0, true // only a capturing root receives it (Runner.Key)
	case "space":
		return proto.KeyRune, ' ', true
	}
	if r := []rune(k.Text); len(r) == 1 && !strings.ContainsRune("\x00\x1b", r[0]) {
		return proto.KeyRune, r[0], true
	}
	return 0, 0, false
}
