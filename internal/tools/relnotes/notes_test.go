package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestSemver(t *testing.T) {
	t.Parallel()
	for _, bad := range []string{"", "1.2", "1.2.3.4", "01.2.3", "1.2.3+build", "v1.2.3-", "x1.2.3"} {
		if _, err := parseSemver(bad); err == nil {
			t.Errorf("parseSemver(%q) accepted it", bad)
		}
	}
	// In order, as Semantic Versioning sorts them.
	order := []string{
		"0.1.0", "0.2.0", "0.2.1-snapshot.0abc123", "0.2.1-snapshot.b95a117", "0.2.1",
		"1.0.0-alpha", "1.0.0-alpha.1", "1.0.0-alpha.beta", "1.0.0-beta.2", "1.0.0-beta.11", "1.0.0", "v1.10.0",
	}
	for i := range order {
		a, err := parseSemver(order[i])
		if err != nil {
			t.Fatal(err)
		}
		if a.compare(a) != 0 {
			t.Errorf("%s does not equal itself", a)
		}
		for _, later := range order[i+1:] {
			b, err := parseSemver(later)
			if err != nil {
				t.Fatal(err)
			}
			if a.compare(b) >= 0 || b.compare(a) <= 0 {
				t.Errorf("%s must sort before %s", a, b)
			}
		}
	}
	if v, _ := parseSemver("v0.3.1-snapshot.abc"); v.String() != "0.3.1-snapshot.abc" {
		t.Errorf("String() = %q", v)
	}
}

var testKinds = []string{"Added", "Changed", "Deprecated", "Removed", "Fixed", "Security"}

func TestParseSection(t *testing.T) {
	t.Parallel()
	good := "## v0.3.0 - 2026-10-08\r\n\r\n### Added\r\n\r\n- One.\r\n- `Two`.\r\n\r\n### Fixed\r\n\r\n- Fixed a crash in some cases.\r\n"
	s, err := parseSection(good, testKinds)
	if err != nil {
		t.Fatal(err)
	}
	if s.version.String() != "0.3.0" || s.date != "2026-10-08" || len(s.kinds) != 2 {
		t.Fatalf("parsed %+v", s)
	}
	want := "### Added\n\n- One.\n- `Two`.\n\n### Fixed\n\n- Fixed a crash in some cases.\n"
	if got := s.body(); got != want {
		t.Errorf("body() =\n%s\nwant\n%s", got, want)
	}
	if s, err := parseSection("## v0.3.1 - 2026-10-09\n", testKinds); err != nil || s.body() != noChanges+"\n" {
		t.Errorf("an empty version: %v, %q", err, s.body())
	}

	for _, tc := range []struct{ text, want string }{
		{"", "no version header"},
		{"# v0.3.0 - 2026-10-08\n", "want a header"},
		{"## 0.3.0 - 2026-10-08\n", "want a header"},
		{"## v0.3.0 - 2026-13-08\n", "not a date"},
		{"## v0.3.0 - 2026-10-08\n\n### Feature\n\n- x\n", "not a kind"},
		{"## v0.3.0 - 2026-10-08\n\n- x\n", "before any"},
		{"## v0.3.0 - 2026-10-08\n\n### Added\n\nSome words.\n", "only"},
		{"## v0.3.0 - 2026-10-08\n\n### Added\n\n- x\n  more of x\n", "only"},
		{"## v0.3.0 - 2026-10-08\n\n### Added\n\n### Fixed\n\n- x\n", `"Added" has no notes`},
		{"## v0.3.0 - 2026-10-08\n\n### Added\n", `"Added" has no notes`},
		{"## v0.3.0 - 2026-10-08\n\n### Added\n\n- " + strings.Repeat("x", maxNote+1) + "\n", "more than 76"},
	} {
		if _, err := parseSection(tc.text, testKinds); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("parseSection(%q) = %v, want an error with %q", tc.text, err, tc.want)
		}
	}
	if _, err := parseSection("## v0.3.0 - 2026-10-08\n\n### Added\n\n- "+strings.Repeat("é", maxNote)+"\n", testKinds); err != nil {
		t.Errorf("the limit counts characters, not bytes: %v", err)
	}
}

func TestReleaseNotes(t *testing.T) {
	t.Parallel()
	s, err := parseSection("## v0.3.0 - 2026-10-08\n\n### Fixed\n\n- Fixed an issue with the Chess game.\n", testKinds)
	if err != nil {
		t.Fatal(err)
	}
	got := releaseNotes(s, "0.2.0", "v0.3.0")
	want := "### Fixed\n\n- Fixed an issue with the Chess game.\n\n---\n\n" +
		"**Install** with one line on Windows, macOS or Linux: see [Installation](https://ghostofgoes.github.io/WOPR/install/).\n\n" +
		"**Verify.** Every file in this release has a signed record of how it was built. " +
		"[Verifying binaries](https://ghostofgoes.github.io/WOPR/install/#verifying-binaries-attestation) shows how to check it with `gh attestation verify`.\n\n" +
		"**Full changelog:** https://github.com/GhostofGoes/WOPR/compare/v0.2.0...v0.3.0\n"
	if got != want {
		t.Errorf("releaseNotes =\n%s\nwant\n%s", got, want)
	}
	if got := releaseNotes(s, "", "v0.1.0"); !strings.HasSuffix(got, "https://github.com/GhostofGoes/WOPR/commits/v0.1.0\n") {
		t.Errorf("the first release links its commits:\n%s", got)
	}
}

// The package changelogs: every version, newest first, under its package version; dated by its
// commit's time where there is one and at midnight UTC otherwise; the packager on each; urgency
// high for a security fix; Markdown code marks dropped; and a line for an empty version.
func TestChglog(t *testing.T) {
	t.Parallel()
	var sections []section
	for _, text := range []string{
		"## v0.2.0 - 2026-10-07\n\n### Added\n\n- `WOPR_SEED` sets the seed.\n\n### Security\n\n- Fixed a problem with saved games.\n",
		"## v0.1.1 - 2026-10-02\n",
		"## v0.1.0 - 2026-10-01\n\n### Added\n\n- The first release.\n",
	} {
		s, err := parseSection(text, testKinds)
		if err != nil {
			t.Fatal(err)
		}
		sections = append(sections, s)
	}
	times := map[semver]time.Time{
		{0, 2, 0, ""}: time.Date(2026, 10, 7, 10, 4, 5, 0, time.FixedZone("CDT", -5*3600)),
		{9, 9, 9, ""}: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), // no such section
	}
	got, err := chglog(sections, "Packager <packager@example.invalid>", times)
	if err != nil {
		t.Fatal(err)
	}
	want := `[
  {
    "semver": "0.2.0-1",
    "date": "2026-10-07T15:04:05Z",
    "packager": "Packager <packager@example.invalid>",
    "deb": {
      "urgency": "high",
      "distributions": [
        "stable"
      ]
    },
    "changes": [
      {
        "note": "WOPR_SEED sets the seed."
      },
      {
        "note": "Fixed a problem with saved games."
      }
    ]
  },
  {
    "semver": "0.1.1-1",
    "date": "2026-10-02T00:00:00Z",
    "packager": "Packager <packager@example.invalid>",
    "deb": {
      "urgency": "medium",
      "distributions": [
        "stable"
      ]
    },
    "changes": [
      {
        "note": "This release has no changes you will notice."
      }
    ]
  },
  {
    "semver": "0.1.0-1",
    "date": "2026-10-01T00:00:00Z",
    "packager": "Packager <packager@example.invalid>",
    "deb": {
      "urgency": "medium",
      "distributions": [
        "stable"
      ]
    },
    "changes": [
      {
        "note": "The first release."
      }
    ]
  }
]
`
	if string(got) != want {
		t.Errorf("chglog =\n%s\nwant\n%s", got, want)
	}
	var back []chglogEntry
	if err := json.Unmarshal(got, &back); err != nil || len(back) != 3 {
		t.Errorf("does not read back: %v", err)
	}
}

// Versions as the packages spell them: dpkg and rpm sort a tilde before anything, so a snapshot
// comes before its release.
func TestPackageVersion(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"0.2.0":                  "0.2.0-1",
		"v1.10.3":                "1.10.3-1",
		"0.2.1-snapshot.b95a117": "0.2.1~snapshot.b95a117-1",
	} {
		v, err := parseSemver(in)
		if err != nil {
			t.Fatal(err)
		}
		if got := packageVersion(v); got != want {
			t.Errorf("packageVersion(%s) = %s, want %s", in, got, want)
		}
	}
}

// The .deb and the .rpm that .goreleaser.yaml builds must match their changelogs: the same
// maintainer as the changelogs' packager, and the same package release as their versions.
func TestPackagingConfig(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join(moduleRoot(t), ".goreleaser.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"maintainer": defaultPackager, "release": packageRelease} {
		re := regexp.MustCompile(`(?m)^[ \t]+` + key + `:[ \t]*"?([^"#\n]*?)"?[ \t]*(?:#.*)?$`)
		found := re.FindAllStringSubmatch(string(data), -1)
		if len(found) != 2 {
			t.Errorf(".goreleaser.yaml sets %s %d times, want 2 (the .deb's and the .rpm's)", key, len(found))
		}
		for _, m := range found {
			if m[1] != want {
				t.Errorf(".goreleaser.yaml sets %s %q, want %q as relnotes writes it", key, m[1], want)
			}
		}
	}
}

func TestConfigKinds(t *testing.T) {
	t.Parallel()
	kinds, err := configKinds(filepath.Join(moduleRoot(t), configFile))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(kinds, testKinds) {
		t.Errorf("kinds = %q, want Keep a Changelog's %q", kinds, testKinds)
	}
}

// The repository's own version files parse; the -check hook also compares CHANGELOG.md.
func TestRepositoryNotes(t *testing.T) {
	t.Parallel()
	sections, err := readSections(filepath.Join(moduleRoot(t), changesDir), testKinds)
	if err != nil {
		t.Fatal(err)
	}
	if len(sections) < 2 || sections[len(sections)-1].version.String() != "0.1.0" {
		t.Errorf("want every release back to v0.1.0, got %d versions", len(sections))
	}
}

func TestUserVisible(t *testing.T) {
	t.Parallel()
	for path, want := range map[string]bool{
		"cmd/wopr/main.go":                          true,
		"internal/games/chess/chess.go":             true,
		"internal/assets/banner.go":                 true,
		"internal/games/chess/chess_test.go":        false,
		"internal/games/chess/testdata/view.golden": false,
		"internal/tools/relnotes/main.go":           false,
		"internal/archtest/archtest_test.go":        false,
		"internal/e2e/e2e_test.go":                  false,
		"internal/golden/golden.go":                 false,
		"internal/games/testkit/testkit.go":         false,
		"internal/games/gamestest/stub.go":          false,
		"README.md":                                 false,
		"docs/PLAN.md":                              false,
		".github/workflows/ci.yml":                  false,
		"go.mod":                                    false,
		".goreleaser.yaml":                          true,
		"packaging/description.txt":                 true,
		"packaging/debian/copyright":                true,
		"docs/man/wopr.6":                           true,
		"docs/manual.md":                            false,
		"site/content/install.md":                   false,
	} {
		if got := userVisible(path); got != want {
			t.Errorf("userVisible(%q) = %v, want %v", path, got, want)
		}
	}
	for path, want := range map[string]bool{
		".changes/unreleased/Fixed-chess.yaml": true,
		".changes/v0.3.0.md":                   true,
		".changes/unreleased/.gitkeep":         false,
		".changes/header.tpl.md":               false,
		"CHANGELOG.md":                         false,
	} {
		if got := isNote(path); got != want {
			t.Errorf("isNote(%q) = %v, want %v", path, got, want)
		}
	}
}
