package main

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

const mitText = `Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.`

const bsd3Text = `Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Neither the name of Google LLC nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.`

const noticeMD = "# Notices\n\n```text\n(Standard 2 clause BSD licence)\n\nCopyright (c) 2012 A Person\nAll rights reserved.\n\n" +
	"Redistribution and use in source and binary forms are permitted.\n```\n\nText.\n\n" +
	"```text\nMap (C) 1998 Matthew Thomas. Freely usable if this line is included.\n```\n"

func TestSplitLicence(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, in string
		holders  []string
		text     string
	}{
		{"title and (c)", "The MIT License (MIT)\n\nCopyright (c) 2016 Someone\n\nPermission is granted.\n\n", []string{"2016 Someone"}, "Permission is granted."},
		{"Go's", "Copyright 2009 The Go Authors.\r\n\r\nRedistribution is allowed.\r\n", []string{"2009 The Go Authors."}, "Redistribution is allowed."},
		{"two holders, rights reserved", "Copyright (C) 2020 A\nCopyright © 2021 B\nAll rights reserved.\n\nText\n  indented\n", []string{"2020 A", "2021 B"}, "Text\n  indented"},
		{"no copyright line", "MIT License\n\nPermission is granted.\n", nil, ""},
	} {
		l := splitLicence(c.in)
		if !slices.Equal(l.holders, c.holders) || l.text != c.text {
			t.Errorf("%s: got %q %q, want %q %q", c.name, l.holders, l.text, c.holders, c.text)
		}
	}
	if !sameText("a  b\nc", " a b c ") || sameText("a b", "a c") {
		t.Error("sameText must compare words, not wrapping")
	}
	const long = "Go module example.com/a/very/long/module/path, version v0.0.0-20261001125412-878653296cfd."
	if got := wrap(long, 40); got != "Go module\nexample.com/a/very/long/module/path,\nversion\nv0.0.0-20261001125412-878653296cfd." {
		t.Errorf("wrap = %q", got)
	}
}

// NOTICE.md's own licence blocks are found and read; a NOTICE.md without them is refused.
func TestNoticeTexts(t *testing.T) {
	t.Parallel()
	notice, err := os.ReadFile(filepath.Join("..", "..", "..", "NOTICE.md"))
	if err != nil {
		t.Fatal(err)
	}
	abs0, mapTerms, err := noticeTexts(string(notice))
	if err != nil {
		t.Fatal(err)
	}
	if len(abs0.holders) != 1 || !strings.HasPrefix(abs0.holders[0], "2012 David Brownlee") || !strings.HasPrefix(abs0.text, "Redistribution and use") {
		t.Errorf("abs0's licence: %q\n%s", abs0.holders, abs0.text)
	}
	if !slices.Equal(mapTerms.holders, []string{"1998 Matthew Thomas"}) || !strings.Contains(mapTerms.text, "Freely usable") {
		t.Errorf("the map's terms: %q %q", mapTerms.holders, mapTerms.text)
	}
	for _, bad := range []string{
		"# Notices\n",
		strings.Replace(noticeMD, "(C) 1998", "1998", 1),
		strings.Replace(noticeMD, "Copyright (c) 2012 A Person\n", "", 1),
	} {
		if _, _, err := noticeTexts(bad); err == nil {
			t.Errorf("accepted NOTICE.md:\n%s", bad)
		}
	}
}

// A small source tree: wopr's licence, NOTICE.md, Go files carrying each kind of tag, and modules
// under the two licences the file knows; then one under a licence it does not, and a stale pattern.
func TestRenderCopyright(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write := func(p, s string) {
		t.Helper()
		p = filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("LICENSE", "MIT License\n\nCopyright (c) 2026 Christopher Goes\n\n"+mitText+"\n")
	write("NOTICE.md", noticeMD)
	write("go/LICENSE", "Copyright 2009 The Go Authors.\n\n"+bsd3Text+"\n")
	write("cmd/wopr/main.go", "package main\n")
	write("internal/wopr/lines.go", "var a = script.Tag(script.ABS0, \"X\")\nvar b = script.Recon(\"Y\")\n")
	write("internal/games/gtw/film.go", "var c = script.Tag(script.Film, \"Z\")\n")
	write("internal/assets/gtwmap.go", "var d = script.Tag(script.MatthewThomasMap, \"M\")\n")
	write("internal/games/gtw/film_test.go", "var e = script.Film\n")
	write("internal/x/testdata/f.go", "var f = script.Film\n")
	write("internal/script/script.go", "Film Prov = \"film\"\n")
	for _, p := range filmOutsideCode {
		write(strings.ReplaceAll(p, "*", "x"), "")
	}
	// The same MIT text, wrapped differently and under a title; and Go's BSD text.
	write("mods/a/LICENSE", "The MIT License (MIT)\n\nCopyright (c) 2016 Someone\n\n"+strings.Join(strings.Fields(mitText), " ")+"\n")
	write("mods/b/LICENSE", "Copyright 2009 The Go Authors.\n\n"+bsd3Text+"\n")
	mods := map[string]module{
		"example.com/b": {Path: "example.com/b", Version: "v1.0.0", Dir: filepath.Join(root, "mods", "b")},
		"example.com/a": {Path: "example.com/a", Version: "v0.2.0", Dir: filepath.Join(root, "mods", "a")},
	}
	goroot := filepath.Join(root, "go")
	got, err := renderCopyright(root, goroot, "go1.27.1", mods)
	if err != nil {
		t.Fatal(err)
	}
	s := string(got)
	for _, want := range []string{
		"Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/\nUpstream-Name: wopr\n",
		"\nFiles: *\nCopyright: 2026 Christopher Goes\nLicense: Expat\n\n",
		"\nFiles: internal/games/gtw/film.go\nCopyright: 2026 Christopher Goes\nLicense: Expat\nComment: Besides code",
		"\nFiles: docs/man/wopr.6\n docs/screenshots/*\n",
		"\nFiles: internal/wopr/lines.go\nCopyright: 2026 Christopher Goes\n 2012 A Person\nLicense: Expat and BSD-2-clause\n",
		"\nFiles: internal/assets/gtwmap.go\nCopyright: 2026 Christopher Goes\n 1998 Matthew Thomas\nLicense: Expat and Matthew-Thomas-map\n",
		"\nFiles: std/*\nCopyright: 2009 The Go Authors.\nLicense: BSD-3-clause\nComment: The Go standard library and runtime, go1.27.1.\n",
		"\nFiles: example.com/a/*\nCopyright: 2016 Someone\nLicense: Expat\n",
		"\nFiles: example.com/b/*\nCopyright: 2009 The Go Authors.\nLicense: BSD-3-clause\n",
		"\nLicense: Expat\n Permission is hereby granted",
		" copies or substantial portions of the Software.\n .\n THE SOFTWARE",
		"\nLicense: BSD-2-clause\n Redistribution and use in source and binary forms are permitted.\n",
		"\nLicense: BSD-3-clause\n Redistribution",
		"\nLicense: Matthew-Thomas-map\n Map (C) 1998 Matthew Thomas. Freely usable if this line is included.\n",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
	if strings.Index(s, "example.com/a/*") > strings.Index(s, "example.com/b/*") {
		t.Error("modules must be sorted")
	}
	if strings.Contains(s, "film_test.go") || strings.Contains(s, "testdata") || strings.Contains(s, "script.go") {
		t.Errorf("tests, test data and package script's own definitions must not be listed:\n%s", s)
	}
	// A file with a film tag and abs0's is listed once, under abs0's licence.
	if strings.Count(s, "internal/wopr/lines.go") != 1 {
		t.Errorf("internal/wopr/lines.go must appear once:\n%s", s)
	}
	// Every line is a field, a continuation or the blank line between paragraphs.
	fieldLine := regexp.MustCompile(`^[A-Z][A-Za-z-]*: \S`)
	for i, line := range strings.Split(strings.TrimSuffix(s, "\n"), "\n") {
		ok := line == "" || (strings.HasPrefix(line, " ") && strings.TrimSpace(line) != "") || fieldLine.MatchString(line)
		if !ok || strings.TrimRight(line, " \t") != line {
			t.Errorf("line %d is not DEP-5: %q", i+1, line)
		}
	}

	write("mods/c/LICENSE", "Copyright 2020 Someone Else\n\nApache License, Version 2.0\n")
	mods["example.com/c"] = module{Path: "example.com/c", Version: "v3.0.0", Dir: filepath.Join(root, "mods", "c")}
	if _, err := renderCopyright(root, goroot, "go1.27.1", mods); err == nil || !strings.Contains(err.Error(), "example.com/c@v3.0.0") {
		t.Errorf("a module under a licence the file does not know must stop the tool, got %v", err)
	}
	delete(mods, "example.com/c")
	if err := os.RemoveAll(filepath.Join(root, "site")); err != nil {
		t.Fatal(err)
	}
	if _, err := renderCopyright(root, goroot, "go1.27.1", mods); err == nil || !strings.Contains(err.Error(), "site/") {
		t.Errorf("a pattern that matches nothing must stop the tool, got %v", err)
	}
}
