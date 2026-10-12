// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package main

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"
)

// gobco's output for a package, as v1.3 prints it with -branch.
const gobcoOutput = `ok  	github.com/GhostofGoes/WOPR/internal/cli	0.007s

Branch coverage: 72/80
internal/cli/cli.go:136:6: condition "errors.Is(err, flag.ErrHelp)" was 2 times false but never true
internal/cli/render.go:37:6: condition "first" was once true but never false
`

func TestParse(t *testing.T) {
	t.Parallel()
	r, err := parse("internal/cli", []byte(gobcoOutput))
	if err != nil {
		t.Fatal(err)
	}
	if r.covered != 72 || r.total != 80 || len(r.uncovered) != 2 || !strings.HasPrefix(r.uncovered[1], "internal/cli/render.go:37:6") {
		t.Errorf("parsed %+v", r)
	}
	crlf := strings.ReplaceAll(gobcoOutput, "\n", "\r\n")
	if r, err := parse("internal/cli", []byte(crlf)); err != nil || r.covered != 72 || r.total != 80 {
		t.Errorf("Windows line ends: %+v, %v", r, err)
	}
	// A package with no conditions has no summary.
	if r, err := parse("internal/x", []byte("ok  \tx\t0.001s\n")); err != nil || r.total != 0 || r.covered != 0 {
		t.Errorf("no conditions: %+v, %v", r, err)
	}
	if _, err := parse("x", []byte("Branch coverage: 1/2\nBranch coverage: 1/2\n")); err == nil {
		t.Error("two summaries were accepted")
	}
	if _, err := parse("x", []byte("Branch coverage: 3/2\n")); err == nil {
		t.Error("more covered than total was accepted")
	}
}

func TestShips(t *testing.T) {
	t.Parallel()
	for pkg, want := range map[string]bool{
		".": true, "cmd/wopr": true, "internal/ui": true, "internal/games/chess": true,
		"internal/tools": false, "internal/tools/branchcov": false, "internal/archtest": false,
		"internal/e2e": false, "internal/golden": false, "internal/games/testkit": false,
		"internal/games/gamestest": false, "internal/toolsmith": true, "..": false, "../x": false,
	} {
		if got := ships(pkg); got != want {
			t.Errorf("ships(%q) = %v, want %v", pkg, got, want)
		}
	}
}

func TestPercent(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		covered, total int
		want           float64
	}{{0, 0, 100}, {0, 4, 0}, {3, 4, 75}, {4, 4, 100}} {
		if got := percent(tc.covered, tc.total); got != tc.want {
			t.Errorf("percent(%d, %d) = %v, want %v", tc.covered, tc.total, got, tc.want)
		}
	}
}

func TestReportAndMarkdown(t *testing.T) {
	t.Parallel()
	results := []result{
		{pkg: "internal/ui", covered: 3, total: 4, uncovered: []string{"internal/ui/a.go:1:1: condition"}},
		{pkg: "cmd/wopr", covered: 1, total: 4},
	}
	var plain strings.Builder
	report(results, &plain, false)
	want := "cmd/wopr        1/4       25.0%\n" +
		"internal/ui     3/4       75.0%\n" +
		"total           4/8       50.0%\n"
	if plain.String() != want {
		t.Errorf("report:\n%s\nwant:\n%s", plain.String(), want)
	}
	var verbose strings.Builder
	report(results, &verbose, true)
	if !strings.Contains(verbose.String(), "\n    internal/ui/a.go:1:1: condition\n") {
		t.Errorf("-v does not list the branches never taken:\n%s", verbose.String())
	}
	if results[0].pkg != "internal/ui" {
		t.Error("report sorted its caller's slice")
	}
	var md strings.Builder
	markdown(results, &md)
	for _, line := range []string{"| `cmd/wopr` | 4 | 1 | 25.0 |", "| **total** | 8 | 4 | **50.0** |"} {
		if !strings.Contains(md.String(), line+"\n") {
			t.Errorf("markdown lacks %q:\n%s", line, md.String())
		}
	}
}

// measure runs every package, at most jobs at a time, keeps the packages' order, and reports
// every error.
func TestMeasure(t *testing.T) {
	t.Parallel()
	pkgs := []pkg{{rel: "a"}, {rel: "b"}, {rel: "c"}, {rel: "d"}}
	var running, most atomic.Int32
	results, err := measure(pkgs, 2, func(p pkg) (result, error) {
		n := running.Add(1)
		defer running.Add(-1)
		for {
			m := most.Load()
			if n <= m || most.CompareAndSwap(m, n) {
				break
			}
		}
		if p.rel == "c" {
			return result{}, errors.New("c failed")
		}
		return result{pkg: p.rel, covered: 1, total: 2}, nil
	})
	if err == nil || !strings.Contains(err.Error(), "c failed") {
		t.Errorf("error %v, want c's", err)
	}
	if most.Load() > 2 {
		t.Errorf("%d packages at once, want at most 2", most.Load())
	}
	if results[0].pkg != "a" || results[3].pkg != "d" {
		t.Errorf("results out of order: %+v", results)
	}
}
