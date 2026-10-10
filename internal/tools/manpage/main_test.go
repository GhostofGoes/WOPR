package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"unicode"

	"github.com/GhostofGoes/WOPR/internal/cli"
	"github.com/GhostofGoes/WOPR/internal/games/catalog"
	"github.com/GhostofGoes/WOPR/internal/movie"
	"github.com/GhostofGoes/WOPR/internal/theme"
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
	got, err := generate(root)
	if err != nil {
		t.Fatal(err)
	}
	env := sectionOf(string(got), "ENVIRONMENT")
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
				if !strings.Contains(env, ".B "+(&page{}).word([]span{{'R', m[1]}})+"\n") {
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
	if got := escape("a-b \\x ~/y ^z `"); got != `a\-b \ex \(ti/y \(haz \(ga` {
		t.Errorf("escape: %q", got)
	}
	if got := textLine(".hidden"); got != `\&.hidden` {
		t.Errorf("textLine: %q", got)
	}
	got := sentences("One. Two? 3 is three! wopr runs. `wopr -m` runs. Not e.g. this (SAY 1 TO 9.), or this. end")
	want := []string{"One.", "Two?", "3 is three!", "wopr runs.", "`wopr -m` runs.", "Not e.g. this (SAY 1 TO 9.), or this. end"}
	if !slices.Equal(got, want) {
		t.Errorf("sentences:\n got %q\nwant %q", got, want)
	}
	for _, l := range wrap(strings.Fields(strings.Repeat("word ", 40)+strings.Repeat("x", 90)), lineWidth) {
		if len(l) > lineWidth && strings.Contains(l, " ") {
			t.Errorf("wrap left a long line: %q", l)
		}
	}
	for in, want := range map[string]string{
		"It cannot be won.": `It cannot be won.\&`, "(SAY 1 TO 9.)": `(SAY 1 TO 9.)\&`, `"Joshua?"`: `"Joshua?"\&`,
		".": `.\&`, "the LOGON:": "the LOGON:", "e.g. this": "e.g. this", "": "",
		`type \fBJoshua.\fR`: `type \fBJoshua.\fR\&`, `(\fISAY\fR.)`: `(\fISAY\fR.)\&`, `\fBwopr\fR`: `\fBwopr\fR`,
	} {
		if got := oneSpace(in); got != want {
			t.Errorf("oneSpace(%q) = %q, want %q", in, got, want)
		}
	}
}

// Words are escaped, set in bold for `...` and in italics for <...>, and led by \% when
// they are names, which groff then never hyphenates.
func TestWords(t *testing.T) {
	t.Parallel()
	p := page{names: map[string]bool{"chess": true}}
	for in, want := range map[string]string{
		"Play chess, or chess's rival.":                     `Play \%chess, or \%chess's rival.`,
		"A film's LOGON: 12 [] ~~ e2e4":                     `A film's \%LOGON: 12 [] \(ti\(ti \%e2e4`,
		"Use `--seed` <n> (`--only`), WOPR_SEED=1 and ~/x.": `Use \%\fB\-\-seed\fR \%\fIn\fR \%(\fB\-\-only\fR), \%WOPR_SEED=1 and \%\(ti/x.`,
		"`wopr --play chess` or `1`.":                       `\%\fBwopr\fR \%\fB\-\-play\fR \%\fBchess\fR or \fB1\fR.`,
		"See https://example.com/a-b/ now.":                 `See \%https://example.com/a\-b/ now.`,
	} {
		if got := strings.Join(p.words(in, true), " "); got != want {
			t.Errorf("words(%q)\n got %s\nwant %s", in, got, want)
		}
	}
	if got := strings.Join(p.words("Press < or > and `.", false), " "); got != `Press < or > and \(ga.` {
		t.Errorf("words without markup: %s", got)
	}
	// A font macro's names are marked the same way. An alternating macro's arguments run
	// together, so \% leads the whole word, not the part of it in the next argument.
	for _, c := range []struct {
		macro string
		args  []string
		want  string
	}{
		{"RB", []string{"[", "--only", "]"}, `.RB \%[ \-\-only ]`},
		{"RI", []string{"[", "options", "]"}, `.RI [ options ]`},
		{"BR", []string{"wopr chess", "."}, `.BR "wopr \%chess" .\&`},
		{"IR", []string{"WarGames", "."}, `.IR \%WarGames .\&`},
		{"B", []string{"1 first-strike"}, `.B "1 \%first\-strike"`},
	} {
		var q page
		q.names = p.names
		q.macro(c.macro, c.args...)
		if got := strings.TrimSuffix(q.String(), "\n"); got != c.want {
			t.Errorf("macro(%q, %q)\n got %s\nwant %s", c.macro, c.args, got, c.want)
		}
	}
	defer func() {
		if recover() == nil {
			t.Error("unclosed markup accepted")
		}
	}()
	p.words("a `b c", true)
}

// The page puts one space between words and sentences: it turns off groff's stretched lines
// right after .TH, and no input line ends a sentence that groff or mandoc would follow with
// two spaces. No text line is longer than mandoc's lint allows, \& included.
func TestOneSpace(t *testing.T) {
	t.Parallel()
	got, err := generate(root)
	if err != nil {
		t.Fatal(err)
	}
	page := string(got)
	if !regexp.MustCompile(`(?m)^\.TH .*\n\.ad l\n\.ds AD l\n`).MatchString(page) {
		t.Error(".TH is not followed by .ad l and .ds AD l")
	}
	for i, l := range strings.Split(page, "\n") {
		if strings.HasPrefix(l, `.\"`) {
			continue
		}
		if oneSpace(l) != l {
			t.Errorf("line %d ends a sentence without \\&: %q", i+1, l)
		}
		if !strings.HasPrefix(l, ".") && len(l) > lineWidth && strings.Contains(l, " ") {
			t.Errorf("line %d is longer than %d bytes: %q", i+1, lineWidth, l)
		}
	}
}

// A name of several words is quoted, as the shell needs it, so that a list of names reads as
// the names it holds: tic-tac-toe answers to "noughts and crosses", not to "crosses".
func TestShellWords(t *testing.T) {
	t.Parallel()
	got := joinAnd(shellWords([]string{"noughts and crosses", "tic-tac-toe", "tictactoe"}), "and")
	if want := `"noughts and crosses", tic-tac-toe and tictactoe`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

// No name in the page can be split at a line's end. Every option, environment variable,
// path, URL, word in capitals, theme, game, alias and scene, wherever the page names it, starts
// a word marked \%, which groff never hyphenates. The names come from the program's own
// tables, so that one added later is checked too.
func TestNamesNeverSplit(t *testing.T) {
	t.Parallel()
	got, err := generate(root)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"wopr": true, siteURL: true, repoURL: true, issuesURL: true, "~/.cache/wopr/debug.log": true}
	for _, f := range cli.Flags() {
		want["-"+f.Short], want["--"+f.Long] = true, true
		if f.Env != "" {
			want[f.Env] = true
		}
	}
	for _, n := range theme.Names() {
		want[n] = true
	}
	for _, e := range catalog.Registry().All() {
		want[cli.Handle(e.Info)] = true
		for _, a := range e.Info.Aliases {
			want[a] = !strings.Contains(a, " ") // a name of several words is plain words
		}
	}
	for _, s := range movie.Scenes() {
		want[s.Slug] = true
	}
	capitals := regexp.MustCompile(`[A-Z].*[A-Z]`)
	unroff := strings.NewReplacer(`\%`, "", `\&`, "", `\fB`, "", `\fI`, "", `\fR`, "",
		`\-`, "-", `\(ti`, "~", `\(ha`, "^", `\(ga`, "`", `\e`, `\`)
	seen := map[string]bool{}
	section := ""
	for i, l := range strings.Split(string(got), "\n") {
		name, args, _ := strings.Cut(l, " ")
		switch {
		case name == ".SH":
			section = args
			continue
		case section == "NAME" || l == "": // the NAME line is whatis's, read as it stands
			continue
		case strings.HasPrefix(l, "."):
			if !slices.Contains(fontMacros, name[1:]) {
				continue
			}
			sep := " " // .B and .I put a space between their arguments; .BR and the rest run them together
			if len(name) == 3 {
				sep = ""
			}
			l = strings.Join(macroArgs(args), sep)
		}
		for _, w := range strings.Fields(l) {
			if strings.Contains(strings.TrimPrefix(w, `\%`), `\%`) {
				t.Errorf("line %d: %s has \\%% inside it, where groff may hyphenate it", i+1, w)
			}
			core := strings.Trim(unroff.Replace(w), `"'()[]{},.;:!?`)
			name := want[core] || want[strings.TrimSuffix(core, "'s")] || capitals.MatchString(core) ||
				strings.Contains(core, "/") || strings.HasPrefix(core, "-")
			if !name || !strings.ContainsFunc(core, unicode.IsLetter) {
				continue
			}
			seen[core] = true
			if !strings.HasPrefix(w, `\%`) {
				t.Errorf("line %d: %s can be split at a line's end: it does not start with \\%%", i+1, w)
			}
		}
	}
	for n, ok := range want {
		if ok && !seen[n] {
			t.Errorf("the page never names %s", n)
		}
	}
}

// macroArgs splits a macro's arguments as roff does: at spaces, except inside double quotes,
// where "" stands for one quote.
func macroArgs(s string) []string {
	var out []string
	for s = strings.TrimLeft(s, " "); s != ""; s = strings.TrimLeft(s, " ") {
		if s[0] != '"' {
			a, rest, _ := strings.Cut(s, " ")
			out, s = append(out, a), rest
			continue
		}
		var b strings.Builder
		i := 1
		for ; i < len(s) && (s[i] != '"' || strings.HasPrefix(s[i:], `""`)); i++ {
			b.WriteByte(s[i])
			if s[i] == '"' {
				i++
			}
		}
		out, s = append(out, b.String()), s[min(i+1, len(s)):]
	}
	return out
}

// Where groff is installed, it renders the page at every width from 60 to 140 columns with no
// warning, splits no word but plain ones, and prints ~ as ~. It runs with an empty man.local,
// so that groff's own character mappings apply, not a distribution's.
func TestGroffRendering(t *testing.T) {
	t.Parallel()
	groff, err := exec.LookPath("groff")
	if err != nil {
		t.Skip("groff is not on the PATH")
	}
	got, err := generate(root)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "wopr.6")
	if err := os.WriteFile(file, got, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "man.local"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	prose := regexp.MustCompile(`^[A-Z]?[a-z]*(?:['’][a-z]+)?$`) // groff 1.23 prints ' as ’
	for width := 60; width <= 140; width += 5 {
		var stderr bytes.Buffer
		cmd := exec.Command(groff, "-M", dir, "-man", "-Tutf8", "-ww", "-P-cbou",
			fmt.Sprintf("-rLL=%dn", width), fmt.Sprintf("-rLT=%dn", width), file)
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil || stderr.Len() > 0 {
			t.Fatalf("groff at %d columns: %v\n%s", width, err, stderr.String())
		}
		lines := strings.Split(string(out), "\n")
		for i, l := range lines[:len(lines)-1] {
			l = strings.TrimRight(l, " ")
			next := strings.Fields(strings.TrimLeft(lines[i+1], " •"))
			if !strings.HasSuffix(l, "‐") || len(next) == 0 { // U+2010, groff's hyphen
				continue
			}
			fields := strings.Fields(l)
			word := strings.TrimSuffix(fields[len(fields)-1], "‐") + next[0]
			if !prose.MatchString(strings.Trim(word, `"'()[]{},.;:!?`)) {
				t.Errorf("at %d columns, groff splits %s at a line's end", width, word)
			}
		}
		if !strings.Contains(string(out), " ~/.cache/wopr/debug.log") || strings.ContainsAny(string(out), "˜ˆ") {
			t.Errorf("at %d columns, groff does not print ~ as ~", width)
		}
	}
}
