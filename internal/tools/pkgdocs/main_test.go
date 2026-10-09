package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode"
)

func gunzip(t *testing.T, data []byte) []byte {
	t.Helper()
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// The header is what `gzip -9n` writes: no file name, no time, the maximum-compression flag (which
// lintian checks for), Unix; and the same input always gives the same bytes.
func TestCompress(t *testing.T) {
	t.Parallel()
	in := []byte(strings.Repeat(".TH WOPR 6\nSHALL WE PLAY A GAME?\n", 200))
	a, err := compress(in)
	if err != nil {
		t.Fatal(err)
	}
	b, err := compress(in)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Error("two runs differ")
	}
	if len(a) < 10 || a[0] != 0x1f || a[1] != 0x8b || a[2] != 8 {
		t.Fatalf("not gzip: % x", a[:min(len(a), 10)])
	}
	if flg := a[3]; flg != 0 {
		t.Errorf("flags = %#x, want none (no name, no comment)", flg)
	}
	if mtime := a[4:8]; !bytes.Equal(mtime, []byte{0, 0, 0, 0}) {
		t.Errorf("time = % x, want zero", mtime)
	}
	if xfl, osb := a[8], a[9]; xfl != 2 || osb != unixOS {
		t.Errorf("XFL %d, OS %d; want 2 (maximum compression) and %d (Unix)", xfl, osb, unixOS)
	}
	if got := gunzip(t, a); !bytes.Equal(got, in) {
		t.Error("does not decompress to its input")
	}
}

// testChangelog is changelog.yml as relnotes writes it for a snapshot of 0.3.0.
const testChangelog = `[
  {"semver": "0.3.0~snapshot.abc1234-1", "date": "2026-10-08T22:49:16Z", "changes": [{"note": "A new game."}]},
  {"semver": "0.2.0-1", "date": "2026-10-07T15:32:04Z", "changes": [{"note": "Fixed a crash."}]}
]
`

func TestRun(t *testing.T) {
	t.Parallel()
	root, notes, out := t.TempDir(), t.TempDir(), filepath.Join(t.TempDir(), "pkg")
	const page = ".\\\" generated\n.TH WOPR 6 2026-10-07 \"wopr 0.2.0\" \"Games Manual\"\n.SH NAME\nwopr \\- WOPR\n"
	sources := map[string]string{
		filepath.Join(root, "docs", "man", "wopr.6"):        page,
		filepath.Join(root, "README.md"):                    "# wopr\n",
		filepath.Join(root, "NOTICE.md"):                    "# Notices\n",
		filepath.Join(notes, "CHANGELOG.md"):                "# Changelog\n",
		filepath.Join(notes, "changelog.yml"):               testChangelog,
		filepath.Join(root, "packaging", "description.txt"): "A summary\nThe description.\n",
	}
	// The real templates, which linux_test.go checks in detail.
	for _, tmpl := range []string{desktopTemplate, metainfoTemplate} {
		data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(tmpl)))
		if err != nil {
			t.Fatal(err)
		}
		sources[filepath.Join(root, filepath.FromSlash(tmpl))] = string(data)
	}
	for p, s := range sources {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := run(root, notes, out, "0.3.0-snapshot.abc1234"); err != nil {
		t.Fatal(err)
	}
	// Every package and archive names the version being built, whatever the tree's page says.
	stamped := strings.Replace(page, `"wopr 0.2.0"`, `"wopr 0.3.0\-snapshot.abc1234"`, 1)
	for name, want := range map[string]string{
		"wopr.6": stamped, "wopr.6.gz": stamped, "NEWS.gz": "# Changelog\n", "README.md.gz": "# wopr\n",
		"NOTICE.md.gz": "# Notices\n", "changelog.yml": testChangelog,
	} {
		data, err := os.ReadFile(filepath.Join(out, name))
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(name, ".gz") {
			data = gunzip(t, data)
		}
		if got := string(data); got != want {
			t.Errorf("%s holds %q, want %q", name, got, want)
		}
	}
	// The menu entries name the program where each package puts it; the metadata has the
	// description and the releases.
	for format, want := range map[string]string{"deb": "Exec=/usr/games/wopr\n", "rpm": "Exec=/usr/bin/wopr\n"} {
		data, err := os.ReadFile(filepath.Join(out, format, appID+".desktop"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), want) {
			t.Errorf("the %s menu entry has no %q:\n%s", format, want, data)
		}
	}
	metainfo, err := os.ReadFile(filepath.Join(out, appID+".metainfo.xml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"\n    <p>The description.</p>\n",
		`<release version="0.3.0-snapshot.abc1234" date="2026-10-08" type="snapshot">`,
		"<li>A new game.</li>",
		`<release version="0.2.0" date="2026-10-07">`,
	} {
		if !strings.Contains(string(metainfo), want) {
			t.Errorf("the metadata has no %q:\n%s", want, metainfo)
		}
	}
	if err := run(root, notes, out, "0.3.1"); err == nil {
		t.Error("notes for another version must fail")
	}
	if err := run(root, notes, out, "v0.3.0"); err == nil {
		t.Error("a version with a v must fail: GoReleaser's {{ .Version }} has none")
	}
	if err := os.Remove(filepath.Join(notes, "changelog.yml")); err != nil {
		t.Fatal(err)
	}
	if err := run(root, notes, out, "0.3.0-snapshot.abc1234"); err == nil {
		t.Error("a missing package changelog must fail")
	}
	if err := os.Remove(filepath.Join(notes, "CHANGELOG.md")); err != nil {
		t.Fatal(err)
	}
	if err := run(root, notes, out, "0.3.0-snapshot.abc1234"); err == nil {
		t.Error("missing release notes must fail")
	}
}

func TestStampVersion(t *testing.T) {
	t.Parallel()
	got, err := stampVersion([]byte(".TH WOPR 6 2026-10-07 \"wopr\" \"Games Manual\"\n.TH OTHER 1 x \"y\" \"z\"\n"), "1.0.0")
	if want := ".TH WOPR 6 2026-10-07 \"wopr 1.0.0\" \"Games Manual\"\n.TH OTHER 1 x \"y\" \"z\"\n"; err != nil || string(got) != want {
		t.Errorf("stampVersion = %q, %v; want %q (only the first title line changes)", got, err, want)
	}
	if _, err := stampVersion([]byte(".SH NAME\n"), "1.0.0"); err == nil {
		t.Error("a page with no title line must fail")
	}
}

// The manual page in the tree has the title line stampVersion rewrites.
func TestRepositoryPage(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(manPage)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := stampVersion(data, "1.0.0"); err != nil {
		t.Error(err)
	}
}

// packaging/description.txt is both packages' description: its first line the .deb's synopsis and
// the .rpm's summary, the rest the extended description. It must follow both formats' rules
// (Debian Policy §3.4, Fedora's "Summary and Description"; lintian and rpmlint check them).
func TestDescription(t *testing.T) {
	t.Parallel()
	root := repoRoot
	data, err := os.ReadFile(filepath.Join(root, "packaging", "description.txt"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) < 2 || strings.TrimSpace(lines[1]) == "" {
		t.Fatal("description.txt needs a synopsis line, then the extended description")
	}
	summary := lines[0]
	switch {
	case len(summary) >= 80:
		t.Errorf("the synopsis is %d characters; Debian and Fedora want under 80", len(summary))
	case strings.HasSuffix(summary, "."):
		t.Error("the synopsis must not end with a period")
	case !unicode.IsUpper(rune(summary[0])):
		t.Error("the synopsis must start with a capital letter (rpmlint: summary-not-capitalized)")
	case strings.Contains(strings.ToLower(summary), "wopr"):
		t.Error("the synopsis must not repeat the package name (Debian Policy §3.4.1; rpmlint: name-repeated-in-summary)")
	case regexp.MustCompile(`(?i)^(a|an|the) `).MatchString(summary):
		t.Error("the synopsis must not start with an article (lintian: description-synopsis-starts-with-article)")
	}
	for i, line := range lines {
		if len(line) > 79 {
			t.Errorf("line %d is %d characters; a .deb indents it by one, and both formats want 80 at most", i+1, len(line))
		}
		if line != strings.TrimSpace(line) {
			t.Errorf("line %d has leading or trailing space, which nFPM drops", i+1)
		}
		for _, r := range line {
			if r < ' ' || r > '~' {
				t.Errorf("line %d has %q; keep the description plain ASCII", i+1, r)
			}
		}
	}
	// The .rpm's summary is spelled out in .goreleaser.yaml, which cannot read it from the file.
	cfg, err := os.ReadFile(filepath.Join(root, ".goreleaser.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^[ \t]+summary:[ \t]*(.*?)[ \t]*(?:#.*)?$`).FindAllSubmatch(cfg, -1)
	if len(m) != 1 || string(m[0][1]) != summary {
		t.Errorf(".goreleaser.yaml's rpm summary must be description.txt's first line %q; found %q", summary, m)
	}
}
