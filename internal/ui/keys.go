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
	switch k.String() {
	case "pgup":
		m.sb.Scroll(m.consoleRows() - 1)
		return nil
	case "pgdown":
		m.sb.Scroll(-(m.consoleRows() - 1))
		return nil
	}

	// While text is being revealed, the first key skips it.
	if m.tw.Revealing() {
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
		return nil
	}
	return m.edit(k)
}

// edit handles keys for the line editor. Typing is kept even while WOPR is busy
// (typeahead); Enter submits only once the prompt is active.
func (m *model) edit(k tea.KeyPressMsg) tea.Cmd {
	switch k.String() {
	case "enter":
		if m.asking && !m.tw.Busy() {
			return m.submit()
		}
		m.held = true
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
	case "space":
		return proto.KeyRune, ' ', true
	}
	if r := []rune(k.Text); len(r) == 1 && !strings.ContainsRune("\x00\x1b", r[0]) {
		return proto.KeyRune, r[0], true
	}
	return 0, 0, false
}
