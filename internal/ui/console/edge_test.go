// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package console

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/GhostofGoes/WOPR/internal/proto"
)

// Typing past MaxInput, one key at a time, stops at MaxInput; Backspace on an empty line does
// nothing.
func TestEditorLimits(t *testing.T) {
	t.Parallel()
	var e Editor
	e.Backspace()
	if !e.Empty() {
		t.Fatal("Backspace on an empty line typed something")
	}
	for range MaxInput + 10 {
		e.Insert("A")
	}
	if n := len(Graphemes(e.Value())); n != MaxInput {
		t.Errorf("%d graphemes after typing past the limit, want %d", n, MaxInput)
	}
	e.Insert(strings.Repeat("B", 5))
	if strings.Contains(e.Value(), "B") {
		t.Error("a paste went past the limit")
	}
}

// A width below one column wraps as one column, so a narrow terminal never loses text.
func TestWrapBelowOneColumn(t *testing.T) {
	t.Parallel()
	for _, w := range []int{0, -3} {
		if got := Wrap("AB C", w, ansi.WcWidth); strings.Join(got, "") != "ABC" || len(got) != 3 {
			t.Errorf("Wrap width %d: %q", w, got)
		}
		if got := HardWrap("AB", w, ansi.WcWidth); strings.Join(got, "") != "AB" {
			t.Errorf("HardWrap width %d: %q", w, got)
		}
	}
}

// A line the typewriter has partly revealed shows only that part.
func TestLineVisible(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		shown int
		want  string
	}{{-1, "SHALL"}, {0, ""}, {3, "SHA"}, {5, "SHALL"}, {9, "SHALL"}} {
		if got := (Line{Text: "SHALL", shown: tc.shown}).Visible(); got != tc.want {
			t.Errorf("shown %d: %q, want %q", tc.shown, got, tc.want)
		}
	}
}

// The scrollback keeps the newest MaxLines lines, and renders nothing at no height.
func TestScrollbackLimits(t *testing.T) {
	t.Parallel()
	var s Scrollback
	for i := range MaxLines + 5 {
		s.Append(strings.Repeat("X", i%7), proto.StyleText)
	}
	if n := len(s.Lines()); n != MaxLines {
		t.Errorf("%d lines kept, want %d", n, MaxLines)
	}
	for _, h := range []int{0, -1} {
		if rows, at := s.RenderAt(40, h, ansi.WcWidth); rows != nil || at != -1 {
			t.Errorf("height %d: %d rows, cursor %d", h, len(rows), at)
		}
	}
}
