package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	repoURL = "https://github.com/GhostofGoes/WOPR"

	// maxNote keeps a note on one line of a Debian changelog, which shows it as "  * <note>" and
	// keeps to 80 columns. .changie.yaml's body.maxLength says the same to `changie new`.
	maxNote = 76

	// noChanges stands in for an empty version: a Debian changelog entry needs a line.
	noChanges = "This release has no changes you will notice."
)

// semver is a version without its "v": MAJOR.MINOR.PATCH and an optional prerelease. Build
// metadata is not used by this project's versions and is refused.
type semver struct {
	major, minor, patch int
	pre                 string
}

var semverRE = regexp.MustCompile(`^v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)

func parseSemver(s string) (semver, error) {
	m := semverRE.FindStringSubmatch(s)
	if m == nil {
		return semver{}, fmt.Errorf("%q is not a version like 1.2.3 or v1.2.3", s)
	}
	var v semver
	for i, p := range []*int{&v.major, &v.minor, &v.patch} {
		n, err := strconv.Atoi(m[i+1])
		if err != nil {
			return semver{}, fmt.Errorf("%q: %w", s, err)
		}
		*p = n
	}
	v.pre = m[4]
	return v, nil
}

func (v semver) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
	if v.pre != "" {
		s += "-" + v.pre
	}
	return s
}

// compare orders versions as Semantic Versioning does: a prerelease sorts before its release, and
// prerelease identifiers compare numerically when both are numbers.
func (v semver) compare(w semver) int {
	for _, d := range []int{v.major - w.major, v.minor - w.minor, v.patch - w.patch} {
		if d != 0 {
			return d
		}
	}
	switch {
	case v.pre == w.pre:
		return 0
	case v.pre == "":
		return 1
	case w.pre == "":
		return -1
	}
	a, b := strings.Split(v.pre, "."), strings.Split(w.pre, ".")
	for i := range min(len(a), len(b)) {
		if c := compareIdent(a[i], b[i]); c != 0 {
			return c
		}
	}
	return len(a) - len(b)
}

func compareIdent(a, b string) int {
	na, ea := strconv.Atoi(a)
	nb, eb := strconv.Atoi(b)
	switch {
	case ea == nil && eb == nil:
		return na - nb
	case ea == nil:
		return -1 // numeric identifiers sort first
	case eb == nil:
		return 1
	}
	return strings.Compare(a, b)
}

// section is one version's notes, as changie batch writes them into .changes/vX.Y.Z.md:
//
//	## vX.Y.Z - YYYY-MM-DD
//
//	### Added
//
//	- A note.
type section struct {
	version semver
	date    string // YYYY-MM-DD
	kinds   []kindNotes
}

type kindNotes struct {
	kind  string
	notes []string
}

var versionHeaderRE = regexp.MustCompile(`^## v(\S+) - (\d{4}-\d{2}-\d{2})$`)

// parseSection reads a version's notes. It accepts only what changie writes with this
// repository's .changie.yaml: the version header, "### Kind" headings from kinds, and "- note"
// lines of one line each, at most maxNote characters long.
func parseSection(text string, kinds []string) (section, error) {
	var s section
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	header := false
	for i, line := range lines {
		where := fmt.Sprintf("line %d", i+1)
		switch {
		case strings.TrimSpace(line) == "":
		case !header:
			m := versionHeaderRE.FindStringSubmatch(line)
			if m == nil {
				return s, fmt.Errorf("%s: want a header like %q, got %q", where, "## v1.2.3 - 2006-01-02", line)
			}
			v, err := parseSemver(m[1])
			if err != nil {
				return s, fmt.Errorf("%s: %w", where, err)
			}
			if _, err := time.Parse(time.DateOnly, m[2]); err != nil {
				return s, fmt.Errorf("%s: %q is not a date", where, m[2])
			}
			s.version, s.date, header = v, m[2], true
		case strings.HasPrefix(line, "### "):
			kind := strings.TrimPrefix(line, "### ")
			if !slices.Contains(kinds, kind) {
				return s, fmt.Errorf("%s: %q is not a kind in .changie.yaml (%s)", where, kind, strings.Join(kinds, ", "))
			}
			if n := len(s.kinds); n > 0 && len(s.kinds[n-1].notes) == 0 {
				return s, fmt.Errorf("%s: %q has no notes", where, s.kinds[n-1].kind)
			}
			s.kinds = append(s.kinds, kindNotes{kind: kind})
		case strings.HasPrefix(line, "- "):
			if len(s.kinds) == 0 {
				return s, fmt.Errorf("%s: a note before any \"### Kind\" heading", where)
			}
			note := strings.TrimSpace(strings.TrimPrefix(line, "- "))
			if n := utf8.RuneCountInString(note); n > maxNote {
				return s, fmt.Errorf("%s: a note is %d characters, more than %d: %q", where, n, maxNote, note)
			}
			k := &s.kinds[len(s.kinds)-1]
			k.notes = append(k.notes, note)
		default:
			return s, fmt.Errorf("%s: only \"### Kind\" headings and one-line \"- note\" items belong here, got %q", where, line)
		}
	}
	if !header {
		return s, errors.New("no version header")
	}
	if n := len(s.kinds); n > 0 && len(s.kinds[n-1].notes) == 0 {
		return s, fmt.Errorf("%q has no notes", s.kinds[n-1].kind)
	}
	return s, nil
}

// readSections parses every .changes/v*.md in dir, newest first. A file's name must match its
// header's version.
func readSections(changes string, kinds []string) ([]section, error) {
	paths, err := filepath.Glob(filepath.Join(changes, "v*.md"))
	if err != nil {
		return nil, err
	}
	var out []section
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		s, err := parseSection(string(data), kinds)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.ToSlash(p), err)
		}
		if want := "v" + s.version.String() + ".md"; filepath.Base(p) != want {
			return nil, fmt.Errorf("%s: its header is for v%s, so it must be named %s", filepath.ToSlash(p), s.version, want)
		}
		out = append(out, s)
	}
	slices.SortFunc(out, func(a, b section) int { return b.version.compare(a.version) })
	return out, nil
}

// body is the section's notes as Markdown, without the version header.
func (s section) body() string {
	var b strings.Builder
	for i, k := range s.kinds {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "### %s\n\n", k.kind)
		for _, n := range k.notes {
			fmt.Fprintf(&b, "- %s\n", n)
		}
	}
	if b.Len() == 0 {
		b.WriteString(noChanges + "\n")
	}
	return b.String()
}

// releaseNotes is the GitHub Release body: the version's notes, a pointer to the attestation check,
// and the comparison with the previous version (prev is "" for the first release). ref is what to
// compare against: the tag, or a commit for a snapshot.
func releaseNotes(s section, prev, ref string) string {
	var b strings.Builder
	b.WriteString(s.body())
	b.WriteString("\n---\n\n")
	b.WriteString("**Verify before you run.** Every file in this release has a signed record of how it was built. ")
	fmt.Fprintf(&b, "[Install](%s#install) shows how to check it with `gh attestation verify`.\n\n", repoURL)
	if prev == "" {
		fmt.Fprintf(&b, "**Full changelog:** %s/commits/%s\n", repoURL, ref)
	} else {
		fmt.Fprintf(&b, "**Full changelog:** %s/compare/v%s...%s\n", repoURL, prev, ref)
	}
	return b.String()
}

// packageRelease is the Debian revision and RPM release of every package: .goreleaser.yaml's
// nfpms set release to the same (a test checks).
const packageRelease = "1"

// packageVersion is a version as the .deb and .rpm spell it, which their changelogs repeat: a
// prerelease follows a tilde, which sorts before the release in dpkg and rpm alike, and the
// package release follows a hyphen. nFPM spells GoReleaser's version the same way: 0.2.1 becomes
// 0.2.1-1, and 0.2.1-snapshot.abc1234 becomes 0.2.1~snapshot.abc1234-1. (In an rpm it would also
// turn a hyphen inside the prerelease into an underscore; this project's only prereleases, the
// snapshots, have none.)
func packageVersion(v semver) string {
	s := fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
	if v.pre != "" {
		s += "~" + v.pre
	}
	return s + "-" + packageRelease
}

// chglog is nFPM's changelog format (github.com/goreleaser/chglog), which it turns into the .deb's
// changelog.Debian.gz and the .rpm's %changelog. It is YAML; JSON is YAML too, so encoding/json
// writes it.
type chglogEntry struct {
	Semver   string         `json:"semver"`
	Date     string         `json:"date"`
	Packager string         `json:"packager"`
	Deb      chglogDeb      `json:"deb"`
	Changes  []chglogChange `json:"changes"`
}

type chglogDeb struct {
	Urgency       string   `json:"urgency"`
	Distributions []string `json:"distributions"`
}

type chglogChange struct {
	Note string `json:"note"`
}

// chglog lists every version, newest first, under its package version. Each is dated by its time
// in times (its tagged commit's), or else at midnight UTC on its version file's date. A Security
// note makes the Debian urgency high, as Debian does for security fixes; otherwise it is medium,
// dch's default. Notes lose their Markdown code marks, which a package changelog would show as
// they are.
func chglog(sections []section, packager string, times map[semver]time.Time) ([]byte, error) {
	entries := make([]chglogEntry, 0, len(sections))
	for _, s := range sections {
		date := s.date + "T00:00:00Z"
		if t, ok := times[s.version]; ok {
			date = t.UTC().Format(time.RFC3339)
		}
		e := chglogEntry{
			Semver:   packageVersion(s.version),
			Date:     date,
			Packager: packager,
			Deb:      chglogDeb{Urgency: "medium", Distributions: []string{"stable"}},
		}
		for _, k := range s.kinds {
			if k.kind == "Security" {
				e.Deb.Urgency = "high"
			}
			for _, n := range k.notes {
				e.Changes = append(e.Changes, chglogChange{Note: strings.ReplaceAll(n, "`", "")})
			}
		}
		if len(e.Changes) == 0 {
			e.Changes = []chglogChange{{Note: noChanges}}
		}
		entries = append(entries, e)
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false) // keep the packager's <...> readable
	enc.SetIndent("", "  ")
	if err := enc.Encode(entries); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

var kindRE = regexp.MustCompile(`(?m)^\s*-\s+label:\s*["']?([^"'\r\n#]+?)["']?\s*(?:#.*)?$`)

// configKinds reads the kinds' labels from .changie.yaml, in order.
func configKinds(config string) ([]string, error) {
	data, err := os.ReadFile(config)
	if err != nil {
		return nil, err
	}
	var kinds []string
	for _, m := range kindRE.FindAllStringSubmatch(string(data), -1) {
		kinds = append(kinds, strings.TrimSpace(m[1]))
	}
	if len(kinds) == 0 {
		return nil, fmt.Errorf("%s lists no kinds", config)
	}
	return kinds, nil
}
