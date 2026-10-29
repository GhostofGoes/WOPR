// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package scrubenv

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestScrub(t *testing.T) {
	for _, n := range Names {
		t.Setenv(n, "1")
	}
	t.Setenv("TERM", "xterm-256color")
	Scrub()
	for _, n := range Names {
		if v, ok := os.LookupEnv(n); ok {
			t.Errorf("%s is still set, to %q", n, v)
		}
	}
	if os.Getenv("TERM") != "xterm-256color" {
		t.Error("Scrub removed TERM")
	}
}

func TestScrubInvalidTerm(t *testing.T) {
	t.Setenv("TERM", "../../../dev/zero")
	Scrub()
	if v, ok := os.LookupEnv("TERM"); ok {
		t.Errorf("TERM is still set, to %q", v)
	}
}

func TestValidTerm(t *testing.T) {
	for term, want := range map[string]bool{
		"":                      true,
		"xterm-256color":        true,
		"screen.xterm-256color": true,
		"foot+base":             true,
		"tmux_direct":           true,
		"dumb":                  true,
		"../../../dev/zero":     false,
		"./x":                   false,
		".hidden":               false,
		"..":                    false,
		`..\..\x`:               false,
		"xterm/../../x":         false,
		"xterm 256":             false,
		"xterm\x1b":             false,
		"xterm\u00e9":           false,
	} {
		if got := ValidTerm(term); got != want {
			t.Errorf("ValidTerm(%q) = %v, want %v", term, got, want)
		}
	}
}

// The package must initialise before colorprofile, which needs os too. Another import could
// make it wait (see the package comment).
func TestImportsOnlyOS(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") || strings.HasPrefix(f, "gobco_") { // gobco adds its own files
			continue
		}
		ast, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range ast.Imports {
			if p, _ := strconv.Unquote(imp.Path.Value); p != "os" {
				t.Errorf("%s imports %q; scrubenv may import only os", f, p)
			}
		}
	}
}
