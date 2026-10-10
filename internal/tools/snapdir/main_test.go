package main

import (
	"bytes"
	"crypto/sha256"
	"debug/elf"
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// repoRoot is the repository, from this package's directory.
var repoRoot = filepath.Join("..", "..", "..")

func readRepo(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// elfProgram returns the header of a 64-bit little-endian ELF executable for machine, which is
// all snapdir reads.
func elfProgram(machine elf.Machine) []byte {
	b := []byte{0x7f, 'E', 'L', 'F', byte(elf.ELFCLASS64), byte(elf.ELFDATA2LSB), byte(elf.EV_CURRENT)}
	b = append(b, make([]byte, 16-len(b))...)
	b = binary.LittleEndian.AppendUint16(b, uint16(elf.ET_EXEC))
	b = binary.LittleEndian.AppendUint16(b, uint16(machine))
	b = binary.LittleEndian.AppendUint32(b, uint32(elf.EV_CURRENT))
	b = append(b, make([]byte, 3*8+4)...) // entry, program and section header offsets, flags
	for _, v := range []uint16{64, 56, 0, 64, 0, 0} {
		b = binary.LittleEndian.AppendUint16(b, v) // header sizes and counts: no headers
	}
	return b
}

// makeDist writes release files as GoReleaser does, with checksums.txt, and returns the
// directory. The program is an ELF header for machine.
func makeDist(t *testing.T, version, arch string, machine elf.Machine) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string][]byte{
		fmt.Sprintf("wopr_%s_linux_%s", version, arch):        elfProgram(machine),
		fmt.Sprintf("wopr_%s_linux_%s.tar.gz", version, arch): []byte("an archive snapdir does not read"),
		"LICENSE":                 []byte("MIT License\n"),
		"NOTICE.md":               []byte("# Notices\n"),
		"THIRD_PARTY_NOTICES.txt": []byte("THIRD-PARTY NOTICES\n"),
	}
	var sums strings.Builder
	for _, name := range slices.Sorted(maps.Keys(files)) {
		if err := os.WriteFile(filepath.Join(dir, name), files[name], 0o644); err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&sums, "%x  %s\n", sha256.Sum256(files[name]), name)
	}
	if err := os.WriteFile(filepath.Join(dir, "checksums.txt"), []byte(sums.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// tree returns each file and directory under dir with its mode.
func tree(t *testing.T, dir string) map[string]fs.FileMode {
	t.Helper()
	got := map[string]fs.FileMode{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		got[filepath.ToSlash(rel)] = info.Mode()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestRun(t *testing.T) {
	for arch, machine := range machines {
		t.Run(arch, func(t *testing.T) {
			const version = "1.2.4-snapshot.abc1234"
			dist := makeDist(t, version, arch, machine)
			out := filepath.Join(t.TempDir(), "snap")
			if err := run(options{dist: dist, version: version, arch: arch, root: repoRoot, out: out}); err != nil {
				t.Fatal(err)
			}

			want := map[string]fs.FileMode{
				".":                       fs.ModeDir | 0o755,
				"bin":                     fs.ModeDir | 0o755,
				"bin/wopr":                0o755,
				"meta":                    fs.ModeDir | 0o755,
				"meta/gui":                fs.ModeDir | 0o755,
				"meta/gui/icon.png":       0o644,
				"meta/gui/wopr.desktop":   0o644,
				"meta/snap.yaml":          0o644,
				"LICENSE":                 0o644,
				"NOTICE.md":               0o644,
				"THIRD_PARTY_NOTICES.txt": 0o644,
			}
			got := tree(t, out)
			if !slices.Equal(slices.Sorted(maps.Keys(got)), slices.Sorted(maps.Keys(want))) {
				t.Fatalf("files %v, want %v", slices.Sorted(maps.Keys(got)), slices.Sorted(maps.Keys(want)))
			}
			for name, mode := range want {
				// Windows has no Unix modes, so only the kinds are checked there.
				if got[name] != mode && (runtime.GOOS != "windows" || got[name].IsDir() != mode.IsDir()) {
					t.Errorf("%s: mode %v, want %v", name, got[name], mode)
				}
			}

			read := func(name string) string {
				data, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(name)))
				if err != nil {
					t.Fatal(err)
				}
				return string(data)
			}
			if !bytes.Equal([]byte(read("bin/wopr")), elfProgram(machine)) {
				t.Error("bin/wopr is not the release's program")
			}
			for _, n := range notices {
				data, err := os.ReadFile(filepath.Join(dist, n))
				if err != nil {
					t.Fatal(err)
				}
				if read(n) != string(data) {
					t.Errorf("%s is not the release's", n)
				}
			}
			if read("meta/gui/icon.png") != readRepo(t, iconFile) {
				t.Errorf("meta/gui/icon.png is not %s", iconFile)
			}
			desktop, err := snapDesktop(readRepo(t, desktopTemplate))
			if err != nil {
				t.Fatal(err)
			}
			if read("meta/gui/wopr.desktop") != desktop {
				t.Error("meta/gui/wopr.desktop is not the snap's form of the menu entry")
			}

			doc := read("meta/snap.yaml")
			summary, body, _ := strings.Cut(readRepo(t, descriptionFile), "\n")
			for _, line := range []string{
				"version: '" + version + "'",
				"summary: " + yamlQuote(summary),
				"  - " + arch,
			} {
				if !slices.Contains(strings.Split(doc, "\n"), line) {
					t.Errorf("meta/snap.yaml has no line %q", line)
				}
			}
			if m := placeholder.FindString(doc); m != "" {
				t.Errorf("meta/snap.yaml still has %s", m)
			}
			desc, ok := literalBlock(doc, "description")
			if !ok || !strings.HasPrefix(desc, strings.TrimSpace(body)+"\n\n") {
				t.Errorf("the description must start with description.txt's, after the summary:\n%s", desc)
			}
		})
	}
}

func TestRunRefuses(t *testing.T) {
	const version = "1.2.3"
	program := "wopr_1.2.3_linux_amd64"
	for _, tc := range []struct {
		name    string
		version string
		arch    string
		change  func(t *testing.T, dist string) // breaks the release files
		want    string
	}{
		{name: "wrong architecture", arch: "amd64", change: func(t *testing.T, dist string) {
			rewrite(t, dist, program, elfProgram(elf.EM_AARCH64))
		}, want: "want ELFCLASS64 EM_X86_64"},
		{name: "not a program", arch: "amd64", change: func(t *testing.T, dist string) {
			rewrite(t, dist, program, []byte("#!/bin/sh\n"))
		}, want: "not an ELF program"},
		{name: "changed after checksums.txt", arch: "amd64", change: func(t *testing.T, dist string) {
			if err := os.WriteFile(filepath.Join(dist, "NOTICE.md"), []byte("changed\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, want: "NOTICE.md: sha256"},
		{name: "not in checksums.txt", arch: "amd64", change: func(t *testing.T, dist string) {
			sums := filepath.Join(dist, "checksums.txt")
			data, err := os.ReadFile(sums)
			if err != nil {
				t.Fatal(err)
			}
			kept := regexp.MustCompile(`(?m)^.*  LICENSE\n`).ReplaceAll(data, nil)
			if err := os.WriteFile(sums, kept, 0o644); err != nil {
				t.Fatal(err)
			}
		}, want: "does not list LICENSE"},
		{name: "unknown architecture", arch: "riscv64", want: "want amd64 or arm64"},
		{name: "tag, not version", version: "v1.2.3", arch: "amd64", want: "is not a version"},
		{name: "too long for a snap", version: "1.2.3-snapshot.0123456789abcdef01", arch: "amd64", want: "is not a snap version"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dist := makeDist(t, version, "amd64", elf.EM_X86_64)
			if tc.change != nil {
				tc.change(t, dist)
			}
			v := cmpOr(tc.version, version)
			out := filepath.Join(t.TempDir(), "snap")
			err := run(options{dist: dist, version: v, arch: tc.arch, root: repoRoot, out: out})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %v, want one containing %q", err, tc.want)
			}
			if _, err := os.Lstat(out); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("-out exists after a refusal: %v", err)
			}
		})
	}

	t.Run("out exists", func(t *testing.T) {
		dist := makeDist(t, version, "amd64", elf.EM_X86_64)
		out := t.TempDir()
		err := run(options{dist: dist, version: version, arch: "amd64", root: repoRoot, out: out})
		if err == nil || !strings.Contains(err.Error(), "already exists") {
			t.Errorf("error %v, want one saying -out exists", err)
		}
	})
}

// rewrite replaces a release file and its line in checksums.txt, as if it had been released so.
func rewrite(t *testing.T, dist, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dist, name), data, 0o644); err != nil {
		t.Fatal(err)
	}
	sums := filepath.Join(dist, "checksums.txt")
	old, err := os.ReadFile(sums)
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`(?m)^[0-9a-f]+  ` + regexp.QuoteMeta(name) + `$`)
	updated := re.ReplaceAll(old, fmt.Appendf(nil, "%x  %s", sha256.Sum256(data), name))
	if err := os.WriteFile(sums, updated, 0o644); err != nil {
		t.Fatal(err)
	}
}

func cmpOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func TestParseChecksums(t *testing.T) {
	sum := strings.Repeat("ab", sha256.Size)
	got, err := parseChecksums([]byte(strings.ToUpper(sum) + "  wopr\n" + sum + "  LICENSE\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got["wopr"] != sum || got["LICENSE"] != sum || len(got) != 2 {
		t.Errorf("got %v", got)
	}
	for _, bad := range []string{
		sum + "\n",
		sum + "  two  names\n",
		sum[:10] + "  wopr\n",
		strings.Repeat("zz", sha256.Size) + "  wopr\n",
		sum + "  wopr\n" + sum + "  wopr\n",
	} {
		if _, err := parseChecksums([]byte(bad)); err == nil {
			t.Errorf("%q: no error", bad)
		}
	}
}

// The snap's menu entry is the Linux packages' entry, with Exec naming the snap's app and Icon a
// file in the snap: every other line is the same, so the two cannot drift apart.
func TestSnapDesktop(t *testing.T) {
	tmpl := readRepo(t, desktopTemplate)
	got, err := snapDesktop(tmpl)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, desktopHeader) {
		t.Error("the entry must start with its own comment")
	}
	body := func(s string) (kept, changed []string) {
		for line := range strings.Lines(s) {
			line = strings.TrimRight(line, "\n")
			switch {
			case strings.HasPrefix(line, "#"):
			case strings.HasPrefix(line, "Exec="), strings.HasPrefix(line, "Icon="):
				changed = append(changed, line)
			default:
				kept = append(kept, line)
			}
		}
		return kept, changed
	}
	wantKept, wantChanged := body(tmpl)
	gotKept, gotChanged := body(got)
	if !slices.Equal(gotKept, wantKept) {
		t.Errorf("the entry's other lines differ from %s:\n%q\nwant\n%q", desktopTemplate, gotKept, wantKept)
	}
	for i, line := range wantChanged {
		want := strings.Replace(line, "Exec=@BINDIR@/wopr", "Exec=wopr", 1)
		if strings.HasPrefix(line, "Icon=") {
			want = "Icon=${SNAP}/meta/gui/icon.png"
		}
		if i >= len(gotChanged) || gotChanged[i] != want {
			t.Errorf("line %d: got %q, want %q", i, gotChanged, want)
		}
	}
	if !slices.Contains(strings.Split(got, "\n"), "Exec=wopr") || !slices.Contains(strings.Split(got, "\n"), "Terminal=true") {
		t.Errorf("the entry must start wopr in a terminal:\n%s", got)
	}

	const entry = "[Desktop Entry]\nType=Application\nName=WOPR\nIcon=x\nExec=@BINDIR@/wopr\nTerminal=true\n"
	for _, tc := range []struct{ name, tmpl, want string }{
		{"a key snapd drops", entry + "TryExec=wopr\n", "snapd drops"},
		{"another program", strings.Replace(entry, "@BINDIR@/wopr", "/usr/bin/wopr", 1), "does not start"},
		{"a longer name", strings.Replace(entry, "@BINDIR@/wopr", "@BINDIR@/woprx", 1), "does not start"},
		{"a placeholder left", entry + "Comment=@COMMENT@\n", "no value for @COMMENT@"},
		{"no icon", strings.Replace(entry, "Icon=x\n", "", 1), "want one Icon="},
		{"no program", strings.Replace(entry, "Exec=@BINDIR@/wopr\n", "", 1), "at least one Exec="},
	} {
		if _, err := snapDesktop(tc.tmpl); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: error %v, want one containing %q", tc.name, err, tc.want)
		}
	}
}

func TestSnapMetadata(t *testing.T) {
	const tmpl = "name: wopr\nversion: '@VERSION@'\nsummary: @SUMMARY@\ndescription: |\n  @DESCRIPTION@\n\n  Last.\narchitectures:\n  - @ARCH@\n"
	got, err := snapMetadata(tmpl, "Falken's games\nFirst paragraph,\nwrapped.\n\nSecond.\n", "1.2.3", "arm64")
	if err != nil {
		t.Fatal(err)
	}
	want := "name: wopr\nversion: '1.2.3'\nsummary: 'Falken''s games'\ndescription: |\n" +
		"  First paragraph,\n  wrapped.\n\n  Second.\n\n  Last.\narchitectures:\n  - arm64\n"
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}

	for _, tc := range []struct{ name, desc, want string }{
		{"long summary", strings.Repeat("x", maxSummary+1) + "\nBody.\n", "the summary"},
		{"no description", "Summary only\n", "no description"},
		{"long description", "Summary\n" + strings.Repeat("x", maxDescription) + "\n", "snapd allows"},
	} {
		if _, err := snapMetadata(tmpl, tc.desc, "1.2.3", "amd64"); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: error %v, want one containing %q", tc.name, err, tc.want)
		}
	}
}

func TestFill(t *testing.T) {
	got, err := fill("a: @A@\n  @B@\n", map[string][]string{"A": {"1"}, "B": {"x", "", "y"}})
	if err != nil || got != "a: 1\n  x\n\n  y\n" {
		t.Errorf("got %q, %v", got, err)
	}
	for _, tc := range []struct {
		tmpl   string
		values map[string][]string
	}{
		{"@A@\n", nil},
		{"a: @A@\n", map[string][]string{"A": {"1", "2"}}},
		{"a: @A@\n", map[string][]string{"A": {"1\n2"}}},
		{"a\n", map[string][]string{"A": {"1"}}},
	} {
		if _, err := fill(tc.tmpl, tc.values); err == nil {
			t.Errorf("fill(%q, %v): no error", tc.tmpl, tc.values)
		}
	}
}

// snapYAMLKeys are the top-level keys a snap.yaml may have: the Snap Store's review tools check it
// against a schema that allows no others (review-tools, reviewtools/schemas/snap.json). Links such
// as website go under links:, unlike in a snapcraft.yaml.
var snapYAMLKeys = []string{
	"apps", "architectures", "assumes", "base", "components", "confinement", "description",
	"environment", "epoch", "grade", "hooks", "layout", "license", "links", "name", "plugs",
	"provenance", "slots", "snapd-info", "summary", "system-usernames", "title", "type", "version",
}

// snapdLicences are the licences wopr's snap names, each of which is on snapd's SPDX list
// (snapd's spdx/licenses.go). snapd refuses any other identifier, LicenseRef- ones included, so
// check that list before adding one here.
var snapdLicences = []string{"MIT", "BSD-2-Clause", "BSD-3-Clause"}

// The template's settings that the snap depends on.
func TestTemplate(t *testing.T) {
	tmpl := readRepo(t, snapTemplate)
	top := map[string]string{}
	for line := range strings.Lines(tmpl) {
		if line[0] == ' ' || line[0] == '#' || line[0] == '\n' {
			continue
		}
		key, value, ok := strings.Cut(strings.TrimRight(line, "\n"), ":")
		if !ok || !slices.Contains(snapYAMLKeys, key) {
			t.Errorf("%q: a snap.yaml may have no top-level key %q", line, key)
		}
		top[key] = strings.TrimSpace(value)
	}
	for key, want := range map[string]string{
		"name": "wopr", "title": "WOPR", "base": "core24", "confinement": "strict", "grade": "stable",
	} {
		if top[key] != want {
			t.Errorf("%s: %q, want %q", key, top[key], want)
		}
	}
	// No plugs: wopr needs no access beyond its terminal, and a plug can need the store's review.
	if regexp.MustCompile(`(?m)^\s*plugs:`).MatchString(tmpl) {
		t.Error("the snap must have no plugs")
	}
	for _, line := range []string{"    command: bin/wopr", "    common-id: " + appID} {
		if !slices.Contains(strings.Split(tmpl, "\n"), line) {
			t.Errorf("no line %q", line)
		}
	}

	// The licence is the .rpm's, but for the LicenseRef that snapd cannot read.
	for term := range strings.SplitSeq(top["license"], " AND ") {
		if !slices.Contains(snapdLicences, term) {
			t.Errorf("license %q: %q is not on snapd's list as far as this test knows", top["license"], term)
		}
	}
	rpm := regexp.MustCompile(`(?m)^\s+license: (.+)$`).FindAllStringSubmatch(readRepo(t, ".goreleaser.yaml"), -1)
	if len(rpm) == 0 {
		t.Fatal(".goreleaser.yaml has no license:")
	}
	for _, m := range rpm {
		var terms []string
		for term := range strings.SplitSeq(m[1], " AND ") {
			if !strings.HasPrefix(term, "LicenseRef-") {
				terms = append(terms, term)
			}
		}
		if want := strings.Join(terms, " AND "); top["license"] != want {
			t.Errorf("license %q, want the .rpm's %q without its LicenseRef: %q", top["license"], m[1], want)
		}
	}
}

func TestLiteralBlock(t *testing.T) {
	doc := "a: 1\ndescription: |\n  one\n\n  two\n\nb: 2\n"
	if got, ok := literalBlock(doc, "description"); !ok || got != "one\n\ntwo\n" {
		t.Errorf("got %q, %v", got, ok)
	}
	if _, ok := literalBlock(doc, "summary"); ok {
		t.Error("found a block that is not there")
	}
}
