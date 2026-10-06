package console

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Wrap breaks a plain-text line into rows no wider than width, measured with method (the
// same method the renderer uses). It breaks at spaces where it can and splits words that
// are wider than a whole row. An empty line yields one empty row.
func Wrap(line string, width int, method ansi.Method) []string {
	if width < 1 {
		width = 1
	}
	if method.StringWidth(line) <= width {
		return []string{line}
	}
	var rows []string
	var row strings.Builder
	rowW := 0
	flush := func() {
		rows = append(rows, strings.TrimRight(row.String(), " "))
		row.Reset()
		rowW = 0
	}
	for i, word := range strings.Split(line, " ") {
		w := method.StringWidth(word)
		sep := 0
		if i > 0 && rowW > 0 {
			sep = 1
		}
		if rowW+sep+w <= width {
			if sep == 1 {
				row.WriteByte(' ')
			}
			row.WriteString(word)
			rowW += sep + w
			continue
		}
		if rowW > 0 {
			flush()
		}
		for _, g := range Graphemes(word) { // a word wider than the row is split
			gw := method.StringWidth(g)
			if rowW+gw > width && rowW > 0 {
				flush()
			}
			row.WriteString(g)
			rowW += gw
		}
	}
	if rowW > 0 || len(rows) == 0 {
		flush()
	}
	return rows
}
