// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

// Package scrubenv removes, before any library reads them, the environment variables that
// would make wopr's libraries start a program or write a file (SECURITY.md, "What to expect"):
//
//   - TMUX: colorprofile, which Lip Gloss and Bubble Tea use, runs `tmux info` from the PATH
//     to ask whether tmux passes true colour. InTmux keeps the answer that TMUX gave, so that
//     wopr can draw in true colour inside tmux without asking. tmux itself converts the
//     colours for an outer terminal that cannot show them.
//   - TEA_TRACE: Bubble Tea appends every byte of input and output to the file it names.
//   - TEA_DEBUG: Bubble Tea writes a panic log into the current folder.
//   - UV_DEBUG: Ultraviolet appends its renderer's log to the file it names.
//
// It also removes TERM when TERM is not a terminal's name (ValidTerm). The terminfo loader
// that colorprofile uses joins TERM to a folder's path and reads the whole file it names, so
// TERM=../../../dev/zero would make wopr read forever, before main runs.
//
// The work is done in init, because Lip Gloss detects the colour profile in a package-level
// variable, before main runs. Go initialises packages in the order of their import paths,
// each after its own imports. This package imports only os, and its path sorts before every
// github.com/charmbracelet package, which charm.land's packages import. So it runs before
// all of them. cmd/wopr imports it for that and for InTmux. Keep its only import os
// (TestImportsOnlyOS): any other import could delay it past colorprofile.
package scrubenv

import "os"

// Names are the variables that init removes.
var Names = []string{"TMUX", "TEA_TRACE", "TEA_DEBUG", "UV_DEBUG"}

// InTmux says whether TMUX was set, before init removed it.
var InTmux bool

func init() {
	InTmux = os.Getenv("TMUX") != ""
	Scrub()
}

// Scrub removes Names from the process's environment, and TERM if it is not ValidTerm.
func Scrub() {
	for _, n := range Names {
		_ = os.Unsetenv(n) // fails only on a name with '=' or NUL
	}
	if !ValidTerm(os.Getenv("TERM")) {
		_ = os.Unsetenv("TERM")
	}
}

// ValidTerm says whether term can be a terminfo name: letters, digits and "+-._", not
// starting with a dot. Such a name stays inside the terminfo folder. Empty is valid.
func ValidTerm(term string) bool {
	if term != "" && term[0] == '.' {
		return false
	}
	for _, c := range []byte(term) {
		switch {
		case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9',
			c == '+', c == '-', c == '.', c == '_':
		default:
			return false
		}
	}
	return true
}
