package ui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/proto/host"
	"github.com/GhostofGoes/WOPR/internal/ui/console"
)

// layoutWidth is the width of WOPR's screen; wider terminals centre it.
const layoutWidth = 80

// blinkPeriod toggles Blink cells and the thinking indicator at 2 Hz (under the 3 Hz
// flash limit).
const blinkPeriod = 500 * time.Millisecond

type geometry struct {
	width, margin int
	viewRows      int // rows for the running program's View (0 in Console layout)
	consoleRows   int // rows for console text and the input line
	panel         bool
}

func (m *model) showPanel() bool {
	switch m.opts.Panel {
	case PanelOn:
		return true
	case PanelOff:
		return false
	case PanelDefault:
	}
	return m.th.Name == "norad"
}

func (m *model) geometry(p host.Placement) geometry {
	g := geometry{width: min(m.w, layoutWidth)}
	g.margin = max((m.w-g.width)/2, 0)
	rows := m.h
	if m.showPanel() {
		g.panel = true
		rows--
	}
	switch p.Layout {
	case proto.LayoutPanel:
		g.viewRows = max(min(p.PanelRows, rows-8), 0)
	case proto.LayoutFull:
		g.viewRows = max(rows-4, 0)
	case proto.LayoutConsole:
	}
	g.consoleRows = rows - g.viewRows
	return g
}

func (m *model) consoleRows() int { return m.geometry(m.place).consoleRows }

func (m *model) View() tea.View {
	v := tea.NewView("")
	v.AltScreen = true
	if m.w <= 0 || m.h <= 0 {
		return v
	}
	if m.tooSmall() {
		v.SetContent(m.renderTooSmall())
		return v
	}
	if !m.started {
		return v
	}
	g := m.geometry(m.place)
	bg := m.th.Lip(proto.StyleText, 0, m.profile)
	var lines []string

	if g.viewRows > 0 {
		prog, _ := m.runner.Top()
		if m.last != nil {
			prog = m.last // a game that just ended, until its last words are out
		}
		if prog != nil {
			canvas := proto.NewCanvas(g.width, g.viewRows)
			prog.View(canvas)
			for y := range g.viewRows {
				lines = append(lines, m.pad(m.canvasRow(canvas, y), g, bg))
			}
		}
	}

	extra := m.inputRows(g.width)
	rows, at := m.sb.RenderAt(g.width, g.consoleRows, m.method, extra...)
	for _, r := range rows {
		lines = append(lines, m.pad(m.th.Lip(r.Style, 0, m.profile).Render(r.Text), g, bg))
	}
	if g.panel {
		lines = append(lines, m.pad(m.frontPanel(g.width), g, bg))
	}
	v.SetContent(strings.Join(lines, "\n"))
	v.Cursor = m.cursor(g, rows, at)
	return v
}

// pad paints the theme background across the whole terminal row.
func (m *model) pad(styled string, g geometry, bg lipgloss.Style) string {
	w := lipgloss.Width(styled)
	left := bg.Render(strings.Repeat(" ", g.margin))
	right := bg.Render(strings.Repeat(" ", max(m.w-g.margin-w, 0)))
	return left + styled + right
}

// inputRows are the console's last rows: the thinking indicator or a host notice, then the
// input line (prompt and what has been typed, including typeahead), wrapped to width.
func (m *model) inputRows(width int) []console.Row {
	var rows []console.Row
	switch {
	case m.notice != "":
		rows = append(rows, console.Row{Text: m.notice, Style: proto.StyleAlert})
	case m.thinking() && !m.asking:
		dots := 1 + int(m.phase/blinkPeriod)%3
		if m.opts.ReduceMotion {
			dots = 3
		}
		rows = append(rows, console.Row{Text: "PROCESSING " + strings.Repeat(".", dots), Style: proto.StyleDim})
	}
	if m.asking || !m.ed.Empty() {
		for _, part := range console.HardWrap(m.prompt+m.ed.Value(), width, m.method) {
			rows = append(rows, console.Row{Text: part, Style: proto.StyleText})
		}
	}
	return rows
}

func (m *model) thinking() bool { return m.runner != nil && (m.runner.Thinking() || len(m.thinks) > 0) }

// cursor puts the native cursor at the end of the input line, or after the text being
// revealed. It stops blinking while WOPR is thinking.
func (m *model) cursor(g geometry, rows []console.Row, at int) *tea.Cursor {
	if m.keyMode || len(rows) == 0 {
		return nil
	}
	row := -1
	if at >= 0 && at < len(rows) { // the last extra row: the input line, or the notice
		row = at
	} else {
		for i := len(rows) - 1; i >= 0; i-- {
			if rows[i].Text != "" {
				row = i
				break
			}
		}
		if row < 0 {
			row = 0
		}
	}
	col := min(m.method.StringWidth(rows[row].Text), g.width-1)
	c := tea.NewCursor(g.margin+col, g.viewRows+row)
	c.Blink = !m.thinking() && !m.opts.ReduceMotion
	return c
}

// canvasRow renders one canvas row as styled runs.
func (m *model) canvasRow(c *proto.Canvas, y int) string {
	var b strings.Builder
	blinkOff := !m.opts.ReduceMotion && int(m.phase/blinkPeriod)%2 == 1
	x := 0
	for x < c.W {
		cell := c.At(x, y)
		run := []rune{}
		for x < c.W {
			next := c.At(x, y)
			if next.S != cell.S || next.A != cell.A {
				break
			}
			r := next.R
			if next.A&proto.AttrBlink != 0 && blinkOff {
				r = ' '
			}
			run = append(run, r)
			x++
		}
		b.WriteString(m.th.Lip(cell.S, cell.A&^proto.AttrBlink, m.profile).Render(string(run)))
	}
	return b.String()
}

// frontPanel is the optional WOPR cabinet row; its lights move while WOPR thinks.
func (m *model) frontPanel(width int) string {
	const lights = 8
	var l strings.Builder
	step := int(m.phase / (blinkPeriod / 2))
	for i := range lights {
		on := i%2 == 0
		if m.thinking() && !m.opts.ReduceMotion {
			on = (i+step)%3 == 0
		}
		if on {
			l.WriteByte('*')
		} else {
			l.WriteByte('.')
		}
		l.WriteByte(' ')
	}
	left := " W.O.P.R.   LINE 1200 BAUD   ONLINE"
	right := strings.TrimSpace(l.String()) + " "
	gap := max(width-len(left)-len(right), 1)
	return m.th.Lip(proto.StyleDim, 0, m.profile).Render(left + strings.Repeat(" ", gap) + right)
}

func (m *model) renderTooSmall() string {
	bg := m.th.Lip(proto.StyleText, 0, m.profile)
	msg := []string{"TERMINAL TOO SMALL", fmt.Sprintf("NEED %dx%d, HAVE %dx%d", MinWidth, MinHeight, m.w, m.h)}
	lines := make([]string, m.h)
	top := max((m.h-len(msg))/2, 0)
	for y := range lines {
		text := ""
		if i := y - top; i >= 0 && i < len(msg) {
			text = msg[i]
		}
		pad := max((m.w-len(text))/2, 0)
		line := strings.Repeat(" ", pad) + text
		if len(line) > m.w {
			line = line[:m.w]
		}
		lines[y] = bg.Render(line + strings.Repeat(" ", max(m.w-len(line), 0)))
	}
	return strings.Join(lines, "\n")
}
