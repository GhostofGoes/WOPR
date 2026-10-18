// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package archtest

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// iosScript is the iOS demo app's screen text (ios/README.md).
const iosScript = "ios/WOPR/Script.swift"

var (
	// A film("...", "...") block in Swift. The film's lines hold no parenthesis.
	swiftFilmCall = regexp.MustCompile(`\bfilm\(\s*("(?:[^"\\]|\\.)*"(?:\s*,\s*"(?:[^"\\]|\\.)*")*)\s*,?\s*\)`)
	swiftString   = regexp.MustCompile(`"((?:[^"\\]|\\.)*)"`)
	swiftFilmOpen = regexp.MustCompile(`\bfilm\(\s*"`)
)

// TestIOSFilmLines checks that every line the iOS app tags film is one of the film's lines that
// the Go program's goldens freeze, character for character (AGENTS.md, "Film text and
// provenance"). The app cannot change a film line, or add one, without a change to a golden.
func TestIOSFilmLines(t *testing.T) {
	root := moduleRoot(t)
	frozen := map[string]bool{}
	goldens, err := filepath.Glob(filepath.Join(root, "internal", "*", "testdata", "film_lines.golden"))
	if err != nil {
		t.Fatal(err)
	}
	more, err := filepath.Glob(filepath.Join(root, "internal", "*", "*", "testdata", "film_lines.golden"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range append(goldens, more...) {
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			frozen[sc.Text()] = true
		}
		if err := sc.Err(); err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if len(frozen) < 20 {
		t.Fatalf("the goldens hold %d film lines, want them all", len(frozen))
	}

	src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(iosScript)))
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, call := range swiftFilmCall.FindAllStringSubmatch(string(src), -1) {
		for _, s := range swiftString.FindAllStringSubmatch(call[1], -1) {
			lines = append(lines, s[1])
		}
	}
	// Every film( call in the file must be one the pattern reads, or a line could hide from it.
	if opened, read := len(swiftFilmOpen.FindAllString(string(src), -1)), len(swiftFilmCall.FindAllString(string(src), -1)); opened != read {
		t.Fatalf("%s has %d film calls; the test reads %d", iosScript, opened, read)
	}
	if len(lines) < 10 {
		t.Fatalf("%s: found %d film lines, want the app's", iosScript, len(lines))
	}
	for _, line := range lines {
		if strings.Contains(line, `\`) {
			t.Errorf("%s: film line %q has an escape; write the film's characters as they are", iosScript, line)
			continue
		}
		if !frozen[line] {
			t.Errorf("%s: film line %q is not in any film_lines.golden", iosScript, line)
		}
	}
	if !slices.Contains(lines, "INDENTIFICATION NOT RECOGNIZED BY SYSTEM") { //nolint:misspell // the film's own spelling
		t.Errorf("%s: the film's LOGON refusal is missing or changed", iosScript)
	}
}
