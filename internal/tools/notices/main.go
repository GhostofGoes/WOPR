// Command notices generates the licence files that ship with wopr:
//
//   - THIRD_PARTY_NOTICES.txt: the licence of every module linked into wopr on any release
//     platform, plus the Go project's own licence, exactly as each ships it. MIT and BSD
//     licences require their notices to accompany binary redistributions, so the file ships in
//     every release archive and the .rpm, and is embedded for `wopr --licenses`.
//
//   - packaging/debian/copyright: the .deb's /usr/share/doc/wopr/copyright, in Debian's
//     machine-readable format (DEP-5): wopr's own licence, the film text it does not cover, the
//     third-party text and art (from NOTICE.md and the provenance tags), and every module linked
//     into the Linux builds (copyright.go).
//
//     go run ./internal/tools/notices            # rewrite both files
//     go run ./internal/tools/notices -check     # fail if either is out of date
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	output        = "THIRD_PARTY_NOTICES.txt"
	copyrightFile = "packaging/debian/copyright"
)

// licenceNames are tried in order in each module's root directory.
var licenceNames = []string{
	"LICENSE", "LICENSE.md", "LICENSE.txt", "LICENCE", "LICENCE.md", "LICENCE.txt",
	"COPYING", "COPYING.md", "COPYING.txt", "License", "license", "license.md",
}

type module struct {
	Path    string
	Version string
	Dir     string
	Main    bool
	Replace *module
}

type pkg struct {
	Standard bool
	Module   *module
}

// file is one generated file and its contents.
type file struct {
	path string
	data []byte
}

func main() {
	check := flag.Bool("check", false, "fail if a file is out of date instead of rewriting it")
	pattern := flag.String("pkg", "./cmd/wopr", "package whose dependencies are listed")
	flag.Parse()
	files, err := generate(*pattern)
	if err != nil {
		fmt.Fprintln(os.Stderr, "notices:", err)
		os.Exit(2) // could not check: not the same as out of date
	}
	if *check {
		stale := false
		for _, f := range files {
			cur, err := os.ReadFile(f.path)
			cur = bytes.ReplaceAll(cur, []byte("\r\n"), []byte("\n"))
			if err != nil || !bytes.Equal(cur, f.data) {
				fmt.Fprintf(os.Stderr, "notices: %s is out of date; run: go run ./internal/tools/notices\n%s", f.path, firstDifference(cur, f.data))
				stale = true
			}
		}
		if stale {
			os.Exit(1)
		}
		return
	}
	for _, f := range files {
		if err := os.WriteFile(f.path, f.data, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "notices:", err)
			os.Exit(1)
		}
	}
}

// targets are the release platforms; dependencies differ per OS, so the notices cover
// the union of all of them. The Linux ones are also the packages' (.deb and .rpm).
var targets = [][2]string{
	{"linux", "amd64"},
	{"linux", "arm64"},
	{"darwin", "amd64"},
	{"darwin", "arm64"},
	{"windows", "amd64"},
	{"windows", "arm64"},
}

func generate(pattern string) ([]file, error) {
	mods, linux := map[string]module{}, map[string]module{}
	for _, t := range targets {
		if err := listModules(pattern, t[0], t[1], mods); err != nil {
			return nil, err
		}
		if t[0] == "linux" {
			if err := listModules(pattern, t[0], t[1], linux); err != nil {
				return nil, err
			}
		}
	}
	out, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		return nil, fmt.Errorf("go env GOROOT: %w", err)
	}
	goroot := strings.TrimSpace(string(out))
	// The release toolchain, not whichever Go runs this: the file must not depend on the
	// host (SL-1), and a newer local Go is allowed.
	goversion, err := toolchain("go.mod")
	if err != nil {
		return nil, err
	}
	notices, err := render(goroot, goversion, mods)
	if err != nil {
		return nil, err
	}
	copyright, err := renderCopyright(".", goroot, goversion, linux)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", copyrightFile, err)
	}
	return []file{{output, notices}, {copyrightFile, copyright}}, nil
}

// toolchain reads go.mod's toolchain line ("go1.27.1").
func toolchain(gomod string) (string, error) {
	data, err := os.ReadFile(gomod)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "toolchain "); ok {
			return strings.TrimSpace(v), nil
		}
	}
	return "", errors.New("go.mod has no toolchain line")
}

// firstDifference shows the first line where the file and the generated text part.
func firstDifference(cur, want []byte) string {
	a, b := strings.Split(string(cur), "\n"), strings.Split(string(want), "\n")
	for i := range max(len(a), len(b)) {
		var x, y string
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if x != y {
			return fmt.Sprintf("line %d:\n  file: %q\n  want: %q\n", i+1, x, y)
		}
	}
	return ""
}

func listModules(pattern, goos, goarch string, mods map[string]module) error {
	cmd := exec.Command("go", "list", "-deps", "-json=Standard,Module", pattern)
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=readonly", "CGO_ENABLED=0", "GOOS="+goos, "GOARCH="+goarch)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("go list (%s/%s): %w: %s", goos, goarch, err, stderr.String())
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var p pkg
		if err := dec.Decode(&p); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return err
		}
		if p.Standard || p.Module == nil || p.Module.Main {
			continue
		}
		m := *p.Module
		if m.Replace != nil {
			m.Dir = m.Replace.Dir
		}
		mods[m.Path] = m
	}
	return nil
}

func render(goroot, goversion string, mods map[string]module) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString("THIRD-PARTY NOTICES\n\n")
	b.WriteString("wopr is built with Go and links the modules below. Their licences follow, unmodified.\n")
	b.WriteString("Generated by `go run ./internal/tools/notices`; do not edit by hand.\n")

	goLicence, err := os.ReadFile(filepath.Join(goroot, "LICENSE"))
	if err != nil {
		return nil, fmt.Errorf("reading the Go licence: %w", err)
	}
	section(&b, "The Go Programming Language ("+goversion+": standard library and runtime)", goLicence)

	paths := make([]string, 0, len(mods))
	for p := range mods {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	var missing []string
	for _, p := range paths {
		m := mods[p]
		text, err := findLicence(m.Dir)
		if err != nil {
			missing = append(missing, p+"@"+m.Version)
			continue
		}
		section(&b, m.Path+" "+m.Version, text)
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("no licence file found for: %s", strings.Join(missing, ", "))
	}
	return b.Bytes(), nil
}

func findLicence(dir string) ([]byte, error) {
	if dir == "" {
		return nil, errors.New("module not downloaded")
	}
	for _, name := range licenceNames {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err == nil {
			return bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n")), nil
		}
	}
	return nil, errors.New("no licence file")
}

func section(b *bytes.Buffer, title string, text []byte) {
	b.WriteString("\n")
	b.WriteString(strings.Repeat("=", 79))
	b.WriteString("\n")
	b.WriteString(title)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("=", 79))
	b.WriteString("\n\n")
	b.Write(bytes.TrimRight(text, "\n"))
	b.WriteString("\n")
}
