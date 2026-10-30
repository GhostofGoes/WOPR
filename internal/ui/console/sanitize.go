// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

// Package console holds the pure parts of WOPR's text console: sanitising, wrapping,
// scrollback, the typewriter and the line editor. It does not import Bubble Tea, so it
// is tested without a terminal.
package console

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"
)

// MaxInput is the longest input line, in grapheme clusters.
const MaxInput = 256

// SanitizeInput cleans text the user typed or pasted into the single-line editor.
// Order matters: tabs and line breaks are turned into spaces before control characters
// are dropped, and escape sequences are removed whole (ansi.Strip) before that, so no
// sequence body is left behind as visible junk.
func SanitizeInput(s string) string {
	s = strings.ToValidUTF8(s, "�")
	s = ansi.Strip(s)
	s = strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ", "\t", " ").Replace(s)
	s = dropControls(s)
	return capGraphemes(s, MaxInput)
}

func dropControls(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r < 0xa0) {
			return -1
		}
		return r
	}, s)
}

func capGraphemes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	g := uniseg.NewGraphemes(s)
	count, end := 0, 0
	for g.Next() {
		if count == n {
			break
		}
		_, end = g.Positions()
		count++
	}
	return s[:end]
}

// Graphemes splits s into grapheme clusters.
func Graphemes(s string) []string {
	var out []string
	g := uniseg.NewGraphemes(s)
	for g.Next() {
		out = append(out, g.Str())
	}
	return out
}
