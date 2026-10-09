package main

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
)

// The installers (the Windows setup.exe and the macOS .dmg) are built after GoReleaser, on
// Windows and macOS runners, from the release files it made. -merge adds them to those files: it
// copies each one in and adds its line to checksums.txt, so the release's attestation of
// checksums.txt covers them too. GoReleaser writes checksums.txt as "<sha256>  <name>" lines
// sorted by name (its checksums pipe, v2.18.2); the merged file keeps that form, so it reads as
// if GoReleaser had made the installers itself.

// sumLine is one line of checksums.txt.
type sumLine struct{ sum, name string }

// parseChecksums reads checksums.txt, refusing a malformed line, a sum that is not SHA-256 in
// lower-case hex, and a name listed twice.
func parseChecksums(data string) ([]sumLine, error) {
	var lines []sumLine
	seen := map[string]bool{}
	for line := range strings.Lines(data) {
		f := strings.Fields(line)
		if len(f) != 2 {
			return nil, fmt.Errorf("checksums.txt: malformed line %q", line)
		}
		if !isSHA256(f[0]) {
			return nil, fmt.Errorf("checksums.txt: %s: %q is not a SHA-256 sum", f[1], f[0])
		}
		if seen[f[1]] {
			return nil, fmt.Errorf("checksums.txt lists %s twice", f[1])
		}
		seen[f[1]] = true
		lines = append(lines, sumLine{sum: f[0], name: f[1]})
	}
	return lines, nil
}

func isSHA256(s string) bool {
	return len(s) == 64 && strings.Trim(s, "0123456789abcdef") == ""
}

// formatChecksums writes lines as GoReleaser does: sorted by name, two spaces between the sum
// and the name.
func formatChecksums(lines []sumLine) string {
	lines = slices.Clone(lines)
	slices.SortFunc(lines, func(a, b sumLine) int { return cmp.Compare(a.name, b.name) })
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(l.sum + "  " + l.name + "\n")
	}
	return b.String()
}

// merge adds files to the release files in dir, which must match its checksums.txt exactly, as
// assemble left it: it copies each file in under its own name, adds its line to checksums.txt,
// and checks every file again. A file whose name is already a release file is refused, so
// nothing GoReleaser made can be replaced. On an error dir is left as it was.
func merge(dir string, files []string) (err error) {
	if len(files) == 0 {
		return errors.New("-merge: name the files to add")
	}
	if err := verifyChecksums(dir); err != nil {
		return fmt.Errorf("before merging: %w", err)
	}
	sumsPath := filepath.Join(dir, "checksums.txt")
	data, err := os.ReadFile(sumsPath)
	if err != nil {
		return err
	}
	lines, err := parseChecksums(string(data))
	if err != nil {
		return err
	}
	if formatChecksums(lines) != string(data) {
		return errors.New("checksums.txt is not as GoReleaser writes it (two spaces, sorted by name); refusing to add to it")
	}
	listed := map[string]bool{"checksums.txt": true}
	for _, l := range lines {
		listed[l.name] = true
	}

	var added []string
	defer func() {
		if err == nil {
			return
		}
		for _, p := range added {
			_ = os.Remove(p)
		}
		if werr := os.WriteFile(sumsPath, data, 0o644); werr != nil {
			err = errors.Join(err, fmt.Errorf("restoring checksums.txt: %w", werr))
		}
	}()
	for _, from := range files {
		name := filepath.Base(from)
		if err := checkReleaseName(name); err != nil {
			return err
		}
		if listed[name] {
			return fmt.Errorf("%s is already a release file; refusing to replace it", name)
		}
		listed[name] = true
		to := filepath.Join(dir, name)
		if err := copyNew(from, to); err != nil {
			return err
		}
		added = append(added, to)
		sum, err := sha256File(to)
		if err != nil {
			return err
		}
		lines = append(lines, sumLine{sum: sum, name: name})
		fmt.Printf("merged: %s  %s\n", sum, name)
	}
	if err := os.WriteFile(sumsPath, []byte(formatChecksums(lines)), 0o644); err != nil {
		return err
	}
	return verifyChecksums(dir)
}

// checkReleaseName refuses a name that checksums.txt could not hold on one line, or that would
// be hidden among the release files.
func checkReleaseName(name string) error {
	switch {
	case name == "" || name == "." || name == string(filepath.Separator):
		return fmt.Errorf("%q is not a file name", name)
	case strings.HasPrefix(name, "."):
		return fmt.Errorf("%s: a release file's name must not start with a dot", name)
	case strings.ContainsFunc(name, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }):
		return fmt.Errorf("%q: a release file's name must not hold spaces or control characters", name)
	}
	return nil
}

// copyNew copies a regular file to a path that must not exist yet.
func copyNew(from, to string) error {
	fi, err := os.Lstat(from)
	if err != nil {
		return err
	}
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", from)
	}
	src, err := os.Open(from)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()
	dst, err := os.OpenFile(to, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		_ = os.Remove(to)
		return err
	}
	return dst.Close()
}
