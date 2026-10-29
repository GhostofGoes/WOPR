// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func keys(fns []fn) []string {
	out := make([]string, len(fns))
	for i, f := range fns {
		out[i] = f.key
	}
	return out
}

func TestParse(t *testing.T) {
	t.Parallel()
	out := []byte(`[{"Name":"script","Path":"github.com/GhostofGoes/WOPR/internal/script","Funcs":[
		{"Name":"Validate","Position":{"File":"internal/script/script.go","Line":98,"Col":6}},
		{"Name":"Session.Type","Position":{"File":"internal/script/script.go","Line":7,"Col":6}}]},
		{"Name":"main","Path":"github.com/GhostofGoes/WOPR","Funcs":[{"Name":"f","Position":{"File":"legal.go","Line":3}}]}]`)
	got, err := parse(out)
	if err != nil {
		t.Fatal(err)
	}
	want := []fn{
		{"internal/script.Validate", "internal/script/script.go:98"},
		{"internal/script.Session.Type", "internal/script/script.go:7"},
		{"..f", "legal.go:3"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("parse = %v, want %v", got, want)
	}
	if _, err := parse([]byte("unreachable func: f")); err == nil {
		t.Error("text output parsed as JSON")
	}
}

// A function is dead only when every system's analysis reports it.
func TestEverywhere(t *testing.T) {
	t.Parallel()
	linux := []fn{{"a.relaunch", "a.go:1"}, {"b.Z", "b.go:1"}, {"b.A", "b.go:2"}}
	darwin := []fn{{"b.A", "b.go:2"}, {"b.Z", "b.go:1"}, {"b.A", "b.go:2"}}
	if got := keys(everywhere([][]fn{linux, darwin})); !slices.Equal(got, []string{"b.A", "b.Z"}) {
		t.Errorf("everywhere = %v", got)
	}
	if got := everywhere(nil); got != nil {
		t.Errorf("everywhere(nil) = %v", got)
	}
}

func TestCheck(t *testing.T) {
	t.Parallel()
	unused := []fn{{"internal/prompt.Valid", "p.go:1"}, {"internal/games/testkit.Session.Keys", "t.go:1"}}
	testsOnly := []fn{
		{"internal/prompt.Valid", "p.go:1"},               // reported once, as never called
		{"internal/games/testkit.Session.Type", "t.go:2"}, // a test-only package
		{"internal/script.Validate", "s.go:1"},            // listed
		{"internal/theme.Distinct", "th.go:1"},            // not listed
	}
	allowed := map[string]string{"internal/script.Validate": "shared", "internal/sim.Simple": "moved"}
	got := strings.Join(check(unused, testsOnly, allowed), "\n")
	for _, want := range []string{
		"p.go:1: internal/prompt.Valid is never called",
		"t.go:1: internal/games/testkit.Session.Keys is never called",
		"th.go:1: only tests call internal/theme.Distinct",
		allowFile + ": internal/sim.Simple is not a function that only tests call",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in:\n%s", want, got)
		}
	}
	if n := strings.Count(got, "\n") + 1; n != 4 {
		t.Errorf("%d problems, want 4:\n%s", n, got)
	}
	if got := check(nil, []fn{{"internal/script.Validate", "s.go:1"}}, map[string]string{"internal/script.Validate": "shared"}); len(got) != 0 {
		t.Errorf("a listed helper: %v", got)
	}
}

func TestParseAllowed(t *testing.T) {
	t.Parallel()
	got, err := parseAllowed(strings.NewReader("# comment\n\ninternal/script.Validate  # the provenance tests\n"))
	if err != nil || len(got) != 1 || got["internal/script.Validate"] != "the provenance tests" {
		t.Errorf("parseAllowed = %v, %v", got, err)
	}
	for _, bad := range []string{
		"internal/script.Validate\n",
		"Validate  # no package\n",
		"internal/script.Validate extra  # two words\n",
		"a.B  # x\na.B  # y\n",
	} {
		if _, err := parseAllowed(strings.NewReader(bad)); err == nil {
			t.Errorf("parseAllowed(%q) passed", bad)
		}
	}
}

// The committed list parses, and each line names a function that exists.
func TestAllowFile(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..", "..")
	allowed, err := readAllowed(filepath.Join(root, allowFile))
	if err != nil {
		t.Fatal(err)
	}
	for key := range allowed {
		dot := strings.LastIndex(key, ".")
		src, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(key[:dot])))
		if err != nil {
			t.Errorf("%s: %v", key, err)
			continue
		}
		found := false
		for _, e := range src {
			if strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
				b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(key[:dot]), e.Name()))
				if err != nil {
					t.Fatal(err)
				}
				found = found || strings.Contains(string(b), "func "+key[dot+1:]+"(")
			}
		}
		if !found {
			t.Errorf("%s: no such function", key)
		}
	}
}
