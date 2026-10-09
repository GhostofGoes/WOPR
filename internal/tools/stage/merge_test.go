package main

import (
	"crypto/sha256"
	"encoding/hex"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func sumOf(body string) string {
	h := sha256.Sum256([]byte(body))
	return hex.EncodeToString(h[:])
}

// releaseDir writes release files and their checksums.txt, as GoReleaser and assemble leave
// them: every file's body is its own name. sums, when set, replaces checksums.txt.
func releaseDir(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	var lines []sumLine
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte(n), 0o644); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, sumLine{sum: sumOf(n), name: n})
	}
	if err := os.WriteFile(filepath.Join(dir, "checksums.txt"), []byte(formatChecksums(lines)), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// snapshot reads every file in dir, to check that a refused merge changed nothing.
func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		files[e.Name()] = string(data)
	}
	return files
}

// installers writes the files to merge, each holding its own name, in a directory of their own.
func installers(t *testing.T, names ...string) []string {
	t.Helper()
	dir := t.TempDir()
	var paths []string
	for _, n := range names {
		p := filepath.Join(dir, n)
		if err := os.WriteFile(p, []byte(n), 0o755); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	return paths
}

// The release files of a real release, in the order GoReleaser's checksums.txt lists them:
// sorted by name, byte by byte, so capitals come first and "-" (0x2d) before "_" (0x5f).
var released = []string{
	"LICENSE", "NOTICE.md", "README.md", "THIRD_PARTY_NOTICES.txt",
	"wopr-1.2.3-1.x86_64.rpm",
	"wopr_1.2.3-1_amd64.deb",
	"wopr_1.2.3_darwin_arm64", "wopr_1.2.3_darwin_arm64.tar.gz",
	"wopr_1.2.3_linux_amd64", "wopr_1.2.3_linux_amd64.tar.gz",
	"wopr_1.2.3_windows_amd64.exe", "wopr_1.2.3_windows_amd64.zip",
}

func TestMerge(t *testing.T) {
	t.Parallel()
	dir := releaseDir(t, released...)
	files := installers(t, "wopr_1.2.3_windows_setup.exe", "wopr_1.2.3_macos.dmg")
	if err := merge(dir, files); err != nil {
		t.Fatalf("merge: %v", err)
	}

	// Each installer's line goes where GoReleaser would have put it: the .dmg after the darwin
	// files and before the Linux ones, setup.exe after the Windows zip.
	order := []string{
		"LICENSE", "NOTICE.md", "README.md", "THIRD_PARTY_NOTICES.txt",
		"wopr-1.2.3-1.x86_64.rpm",
		"wopr_1.2.3-1_amd64.deb",
		"wopr_1.2.3_darwin_arm64", "wopr_1.2.3_darwin_arm64.tar.gz",
		"wopr_1.2.3_linux_amd64", "wopr_1.2.3_linux_amd64.tar.gz",
		"wopr_1.2.3_macos.dmg",
		"wopr_1.2.3_windows_amd64.exe", "wopr_1.2.3_windows_amd64.zip",
		"wopr_1.2.3_windows_setup.exe",
	}
	var want strings.Builder
	for _, n := range order {
		want.WriteString(sumOf(n) + "  " + n + "\n")
	}
	got, err := os.ReadFile(filepath.Join(dir, "checksums.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want.String() {
		t.Errorf("checksums.txt:\n%s\nwant:\n%s", got, want.String())
	}
	for _, n := range []string{"wopr_1.2.3_windows_setup.exe", "wopr_1.2.3_macos.dmg"} {
		data, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil || string(data) != n {
			t.Errorf("%s: %q, %v", n, data, err)
		}
	}
	if err := verifyChecksums(dir); err != nil {
		t.Errorf("the merged files must check: %v", err)
	}

	// Merging the same installer again is refused: it is a release file now.
	before := snapshot(t, dir)
	if err := merge(dir, files[:1]); err == nil || !strings.Contains(err.Error(), "already a release file") {
		t.Errorf("a second merge of the same file: %v", err)
	}
	if after := snapshot(t, dir); !maps.Equal(before, after) {
		t.Error("a refused merge changed the release files")
	}
}

// Every refusal leaves the release files exactly as they were.
func TestMergeRefuses(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func(t *testing.T, dir string) []string // returns the files to merge
		want  string
	}{
		{"no files", func(*testing.T, string) []string { return nil }, "name the files"},
		{"a release file's name", func(t *testing.T, _ string) []string {
			return installers(t, "wopr_1.2.3_macos.dmg", "LICENSE")
		}, "LICENSE is already a release file"},
		{"checksums.txt itself", func(t *testing.T, _ string) []string {
			return installers(t, "checksums.txt")
		}, "already a release file"},
		{"the same name twice", func(t *testing.T, _ string) []string {
			a := installers(t, "wopr_1.2.3_macos.dmg")
			b := installers(t, "wopr_1.2.3_macos.dmg")
			return append(a, b...)
		}, "already a release file"},
		{"a space in the name", func(t *testing.T, _ string) []string {
			return installers(t, "wopr setup.exe")
		}, "spaces"},
		{"a hidden name", func(t *testing.T, _ string) []string {
			return installers(t, ".wopr.dmg")
		}, "dot"},
		{"a missing file", func(t *testing.T, _ string) []string {
			return append(installers(t, "wopr_1.2.3_macos.dmg"), filepath.Join(t.TempDir(), "wopr_1.2.3_windows_setup.exe"))
		}, "wopr_1.2.3_windows_setup.exe"},
		{"a directory", func(t *testing.T, _ string) []string {
			d := filepath.Join(t.TempDir(), "wopr_1.2.3_macos.dmg")
			if err := os.Mkdir(d, 0o755); err != nil {
				t.Fatal(err)
			}
			return []string{d}
		}, "not a regular file"},
		{"a file not in checksums.txt", func(t *testing.T, dir string) []string {
			if err := os.WriteFile(filepath.Join(dir, "stray"), nil, 0o644); err != nil {
				t.Fatal(err)
			}
			return installers(t, "wopr_1.2.3_macos.dmg")
		}, "stray is not in checksums.txt"},
		{"a release file changed after GoReleaser summed it", func(t *testing.T, dir string) []string {
			if err := os.WriteFile(filepath.Join(dir, "NOTICE.md"), []byte("edited"), 0o644); err != nil {
				t.Fatal(err)
			}
			return installers(t, "wopr_1.2.3_macos.dmg")
		}, "NOTICE.md: sha256"},
		{"checksums.txt not sorted", func(t *testing.T, dir string) []string {
			body := sumOf("README.md") + "  README.md\n" + sumOf("LICENSE") + "  LICENSE\n"
			for _, n := range released[1:] {
				if n != "README.md" {
					_ = os.Remove(filepath.Join(dir, n))
				}
			}
			if err := os.WriteFile(filepath.Join(dir, "checksums.txt"), []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
			return installers(t, "wopr_1.2.3_macos.dmg")
		}, "not as GoReleaser writes it"},
		{"one space between sum and name", func(t *testing.T, dir string) []string {
			for _, n := range released[1:] {
				_ = os.Remove(filepath.Join(dir, n))
			}
			if err := os.WriteFile(filepath.Join(dir, "checksums.txt"), []byte(sumOf("LICENSE")+" LICENSE\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			return installers(t, "wopr_1.2.3_macos.dmg")
		}, "not as GoReleaser writes it"},
		{"a name listed twice", func(t *testing.T, dir string) []string {
			path := filepath.Join(dir, "checksums.txt")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			data = append(data, []byte(sumOf("LICENSE")+"  LICENSE\n")...)
			if err := os.WriteFile(path, data, 0o644); err != nil {
				t.Fatal(err)
			}
			return installers(t, "wopr_1.2.3_macos.dmg")
		}, "LICENSE twice"},
		{"a sum that is not SHA-256", func(t *testing.T, dir string) []string {
			for _, n := range released[1:] {
				_ = os.Remove(filepath.Join(dir, n))
			}
			if err := os.WriteFile(filepath.Join(dir, "checksums.txt"), []byte(strings.ToUpper(sumOf("LICENSE"))+"  LICENSE\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			return installers(t, "wopr_1.2.3_macos.dmg")
		}, "not a SHA-256 sum"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := releaseDir(t, released...)
			files := tt.setup(t, dir)
			before := snapshot(t, dir)
			err := merge(dir, files)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("merge: %v, want an error containing %q", err, tt.want)
			}
			if after := snapshot(t, dir); !maps.Equal(before, after) {
				t.Errorf("a refused merge changed the release files:\nbefore %v\nafter  %v", keys(before), keys(after))
			}
		})
	}
}

func keys(m map[string]string) []string { return slices.Sorted(maps.Keys(m)) }

func TestFormatChecksumsRoundTrip(t *testing.T) {
	t.Parallel()
	// The order GoReleaser's checksums pipe gives: by name, compared byte by byte.
	in := sumOf("b") + "  B.txt\n" + sumOf("a") + "  a-b\n" + sumOf("c") + "  a_b\n" + sumOf("d") + "  ab\n"
	lines, err := parseChecksums(in)
	if err != nil {
		t.Fatal(err)
	}
	if got := formatChecksums(lines); got != in {
		t.Errorf("sorted again:\n%s\nwant:\n%s", got, in)
	}
	reversed := []sumLine{lines[3], lines[2], lines[1], lines[0]}
	if got := formatChecksums(reversed); got != in {
		t.Errorf("from reversed lines:\n%s\nwant:\n%s", got, in)
	}
}
