package main

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/cli"
	"github.com/GhostofGoes/WOPR/internal/games/catalog"
)

// root is the repository root, from this package's directory.
const root = "../../.."

// Every game players can choose has a page in site/data/games that matches the catalog:
// see check.
func TestGameData(t *testing.T) {
	t.Parallel()
	reg := catalog.Registry()
	gs, err := loadGames(root, reg)
	if err != nil {
		t.Fatal(err)
	}
	if len(gs) != len(reg.All()) {
		t.Fatalf("%d game pages for %d games", len(gs), len(reg.All()))
	}
}

// check finds what is wrong with a page, not only that something is.
func TestCheckCatches(t *testing.T) {
	t.Parallel()
	reg := catalog.Registry()
	e, ok := reg.Get("chess")
	if !ok {
		t.Fatal("no chess in the catalog")
	}
	good, err := readGame(filepath.Join(root, gamesDir, "chess.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := check(good, e, reg, root); err != nil {
		t.Fatalf("chess's page refused: %v", err)
	}
	for name, c := range map[string]struct {
		edit func(*game)
		want string
	}{
		"slug":      {func(g *game) { g.Slug = "chess-game" }, `slug "chess-game"`},
		"launch":    {func(g *game) { g.Launch = "wopr 7" }, "launch"},
		"number":    {func(g *game) { n := 8; g.Number = &n }, "number"},
		"no number": {func(g *game) { g.Number = nil }, "number"},
		"name":      {func(g *game) { g.Name = "CHESS" }, "title case"},
		"alias":     {func(g *game) { g.Aliases = append(g.Aliases, "checkers") }, "aliases"},
		"summary":   {func(g *game) { g.Summary = " " }, "summary has an empty entry"},
		"howToPlay": {func(g *game) { g.HowToPlay = g.HowToPlay[:2] }, "howToPlay has 2"},
		"controls":  {func(g *game) { g.Controls = nil }, "controls is empty"},
		"tips":      {func(g *game) { g.Tips = g.Tips[:2] }, "tips has 2"},
		"ascii":     {func(g *game) { g.Tips = append(slices.Clone(g.Tips), "Use the ♔.") }, "printable ASCII"},
		"shot name": {func(g *game) {
			g.Screenshots = append(slices.Clone(g.Screenshots), screenshot{"board.png", "A board."})
		}, "not named chess-<n>.png"},
		"shot file": {func(g *game) {
			g.Screenshots = append(slices.Clone(g.Screenshots), screenshot{"chess-9.png", "A board."})
		}, "chess-9.png"},
		"few shots":  {func(g *game) { g.Screenshots = g.Screenshots[:1] }, "screenshots has 1"},
		"no caption": {func(g *game) { g.Screenshots = []screenshot{{"chess-1.png", ""}, {"chess-2.png", "x"}} }, "screenshots has an empty entry"},
		"unlisted no": {func(g *game) {
			g.Aliases = slices.DeleteFunc(slices.Clone(g.Aliases), func(a string) bool { return a == "7" })
		}, "aliases"},
	} {
		g := good
		g.Number = new(int)
		*g.Number = *good.Number
		c.edit(&g)
		err := check(g, e, reg, root)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: got %v, want an error containing %q", name, err, c.want)
		}
	}
}

// A misspelt field is refused rather than read as empty.
func TestReadGameStrict(t *testing.T) {
	t.Parallel()
	p := filepath.Join(t.TempDir(), "x.json")
	if err := os.WriteFile(p, []byte(`{"slug": "x", "tip": ["one"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readGame(p); err == nil || !strings.Contains(err.Error(), "tip") {
		t.Errorf("unknown field accepted: %v", err)
	}
}

// Every option in internal/cli has a description here, and none is left over.
func TestOptionDetails(t *testing.T) {
	t.Parallel()
	var longs []string
	for _, f := range cli.Flags() {
		longs = append(longs, f.Long)
		if _, ok := optionDetails[f.Long]; !ok && f.Long != "theme" {
			t.Errorf("--%s has no entry in optionDetails", f.Long)
		}
	}
	for k := range optionDetails {
		if !slices.Contains(longs, k) {
			t.Errorf("optionDetails describes --%s, which internal/cli does not have", k)
		}
	}
}

// Every environment variable the program reads is in the page's ENVIRONMENT section.
func TestEnvironmentDocumented(t *testing.T) {
	t.Parallel()
	page, err := generate(root)
	if err != nil {
		t.Fatal(err)
	}
	env := sectionOf(string(page), "ENVIRONMENT")
	read := regexp.MustCompile(`[gG]etenv\("([A-Z_]+)"\)`)
	found := 0
	for _, dir := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			switch {
			case err != nil:
				return err
			case d.IsDir() && (d.Name() == "tools" || d.Name() == "testdata"):
				return filepath.SkipDir // the developer tools read CI's variables
			case d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go"):
				return nil
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, m := range read.FindAllStringSubmatch(string(src), -1) {
				found++
				if !strings.Contains(env, ".B "+escape(m[1])+"\n") {
					t.Errorf("%s reads %s, which ENVIRONMENT does not describe", path, m[1])
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if found < 5 {
		t.Fatalf("found only %d environment reads; has the pattern gone stale?", found)
	}
}

// sectionOf is the text of one .SH section.
func sectionOf(page, name string) string {
	_, after, ok := strings.Cut(page, ".SH "+name+"\n")
	if !ok {
		return ""
	}
	if i := strings.Index(after, "\n.SH "); i >= 0 {
		return after[:i+1]
	}
	return after
}

// docs/man/wopr.6 is what the tool generates from this tree.
func TestPageIsCurrent(t *testing.T) {
	t.Parallel()
	got, err := generate(root)
	if err != nil {
		t.Fatal(err)
	}
	cur, err := os.ReadFile(filepath.Join(root, output))
	if err != nil {
		t.Fatal(err)
	}
	cur = bytes.ReplaceAll(cur, []byte("\r\n"), []byte("\n"))
	if !bytes.Equal(cur, got) {
		t.Errorf("%s is out of date; run: go run ./internal/tools/manpage\n%s", output, firstDifference(cur, got))
	}
	for _, sec := range []string{
		"NAME", "SYNOPSIS", "DESCRIPTION", "OPTIONS", "GAMES", "MOVIE SCENES", "EXIT STATUS",
		"ENVIRONMENT", "FILES", "NOTES", "BUGS", "EXAMPLES", "SEE ALSO",
	} {
		if sectionOf(string(got), sec) == "" && sectionOf(string(got), `"`+sec+`"`) == "" {
			t.Errorf("no %s section", sec)
		}
	}
}

func TestRelease(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if v, d, err := release(filepath.Join(dir, "missing")); err != nil || v != "" || d != fallbackDate {
		t.Errorf("no .changes: %q %q %v", v, d, err)
	}
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("header.tpl.md", "# Changelog\n")
	if v, d, err := release(dir); err != nil || v != "" || d != fallbackDate {
		t.Errorf("no version files: %q %q %v", v, d, err)
	}
	write("v0.2.0.md", "## v0.2.0 - 2026-10-07\n")
	write("v0.10.0.md", "## v0.10.0 - 2027-01-02\n\n### Added\n")
	write("v0.9.1.md", "## v0.9.1 - 2026-12-31\n")
	if v, d, err := release(dir); err != nil || v != "0.10.0" || d != "2027-01-02" {
		t.Errorf("got %q %q %v, want 0.10.0 dated 2027-01-02 (versions compare as numbers)", v, d, err)
	}
	write("v1.0.0.md", "## v1.0.0\n")
	if _, _, err := release(dir); err == nil {
		t.Error("a version file with no date accepted")
	}
}

func TestRoff(t *testing.T) {
	t.Parallel()
	if got := escape(`a-b \x`); got != `a\-b \ex` {
		t.Errorf("escape: %q", got)
	}
	if got := textLine(".hidden"); got != `\&.hidden` {
		t.Errorf("textLine: %q", got)
	}
	got := sentences("One. Two? 3 is three! wopr runs. Not e.g. this (SAY 1 TO 9.), or this. end")
	want := []string{"One.", "Two?", "3 is three!", "wopr runs.", "Not e.g. this (SAY 1 TO 9.), or this. end"}
	if !slices.Equal(got, want) {
		t.Errorf("sentences:\n got %q\nwant %q", got, want)
	}
	for _, l := range wrap(strings.Repeat("word ", 40)+strings.Repeat("x", 90), lineWidth) {
		if len(l) > lineWidth && strings.Contains(l, " ") {
			t.Errorf("wrap left a long line: %q", l)
		}
	}
}
