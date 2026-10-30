// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

// Command deadcheck fails when the module has dead code: a function that nothing calls, or a
// function in a package that ships that only tests call.
//
//	go run ./internal/tools/deadcheck
//
// Run it from the repository root. It builds deadcode (golang.org/x/tools/cmd/deadcode, pinned in
// tools/go.mod) once, then runs it on ./... for linux, darwin and windows, since some files build
// for one system only (cmd/wopr's macOS relaunch). A function is dead only when it is dead on every
// system whose build includes its file: a function in launch.go that only darwin calls is live, and
// one in a _windows.go file that nothing calls is dead. deadcode starts from every main package, so
// each tool in internal/tools counts as a program too.
//
// It runs two analyses on each system:
//
//   - With -test (and the e2e build tag): neither a program nor a test reaches the function. Remove it.
//   - Without -test: only tests reach the function. Move it into a _test.go file, or into a
//     test-only package (testOnly). A helper that tests in other packages share, and that needs
//     its own package's unexported names, may stay where it is when allowFile lists it with the
//     reason.
//
// It fails on either finding, and on an allowFile line that names a function no longer dead.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
)

// module is the module path, which deadcode's package paths start with.
const module = "github.com/GhostofGoes/WOPR"

// allowFile lists the functions that only tests call but that stay in a package that ships.
const allowFile = "tools/deadcheck-allow.txt"

// systems are the GOOS values that wopr ships for.
var systems = []string{"linux", "darwin", "windows"}

// testOnly are the packages that only tests use (G-5). Their code exists for tests, so only the
// -test analysis covers it. The internal/tools programs are not here: their code must serve them.
var testOnly = []string{"internal/golden", "internal/games/gamestest", "internal/games/testkit"}

// fn is one function that deadcode reports.
type fn struct {
	key string // the package's path in the module and the function's name: "internal/script.Validate"
	pos string // file:line
}

// report is deadcode's -json output for one package.
type report struct {
	Path  string
	Funcs []struct {
		Name     string
		Position struct {
			File string
			Line int
		}
	}
}

func main() { os.Exit(realMain()) }

func realMain() int {
	if _, err := os.Stat("go.mod"); err != nil {
		fmt.Fprintln(os.Stderr, "deadcheck: run it from the repository root")
		return 2
	}
	allowed, err := readAllowed(allowFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "deadcheck:", err)
		return 2
	}
	tmp, err := os.MkdirTemp("", "deadcheck")
	if err != nil {
		fmt.Fprintln(os.Stderr, "deadcheck:", err)
		return 2
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	bin := filepath.Join(tmp, "deadcode")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if out, err := exec.Command("go", "build", "-modfile=tools/go.mod", "-o", bin, "golang.org/x/tools/cmd/deadcode").CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "deadcheck: building deadcode: %v\n%s", err, out)
		return 2
	}
	unused, testsOnly, err := analyse(bin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "deadcheck:", err)
		return 2
	}
	problems := check(unused, testsOnly, allowed)
	if len(problems) == 0 {
		fmt.Printf("deadcheck: no dead code on %s\n", strings.Join(systems, ", "))
		return 0
	}
	for _, p := range problems {
		fmt.Fprintln(os.Stderr, p)
	}
	return 1
}

// run is one analysis on one system: what deadcode reports, and the files that system builds.
type run struct {
	goos  string
	test  bool
	out   []fn
	files map[string]bool
	err   error
}

// analyse runs both analyses on every system, at once, and returns the functions that each finds
// dead on every system that builds their file.
func analyse(bin string) (unused, testsOnly []fn, err error) {
	var runs []*run
	for _, goos := range systems {
		runs = append(runs, &run{goos: goos, test: true}, &run{goos: goos})
	}
	var wg sync.WaitGroup
	for _, r := range runs {
		wg.Go(func() {
			if r.files, r.err = sourceFiles(r.goos); r.err == nil {
				r.out, r.err = deadcode(bin, r.goos, r.test)
			}
		})
	}
	wg.Wait()
	var withTests, withoutTests []*run
	for _, r := range runs {
		if r.err != nil {
			return nil, nil, fmt.Errorf("deadcode on %s: %w", r.goos, r.err)
		}
		if r.test {
			withTests = append(withTests, r)
		} else {
			withoutTests = append(withoutTests, r)
		}
	}
	return everywhere(withTests), everywhere(withoutTests), nil
}

// goEnv is the environment for a go command or deadcode on goos: the module stays read-only.
func goEnv(goos string) []string {
	return append(os.Environ(), "GOOS="+goos, "GOFLAGS="+strings.TrimSpace(os.Getenv("GOFLAGS")+" -mod=readonly"))
}

// sourceFiles lists the .go files that goos's build of ./... includes, tests and the e2e tag
// included, relative to the repository root with slashes, as deadcode names them.
func sourceFiles(goos string) (map[string]bool, error) {
	cmd := exec.Command("go", "list", "-e", "-tags", "e2e", "-json=Dir,GoFiles,CgoFiles,TestGoFiles,XTestGoFiles", "./...")
	cmd.Env = goEnv(goos)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list: %w: %s", err, stderr.String())
	}
	root, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	files := map[string]bool{}
	dec := json.NewDecoder(bytes.NewReader(out))
	for dec.More() {
		var p struct {
			Dir                                          string
			GoFiles, CgoFiles, TestGoFiles, XTestGoFiles []string
		}
		if err := dec.Decode(&p); err != nil {
			return nil, err
		}
		for _, f := range slices.Concat(p.GoFiles, p.CgoFiles, p.TestGoFiles, p.XTestGoFiles) {
			rel, err := filepath.Rel(root, filepath.Join(p.Dir, f))
			if err != nil {
				return nil, err
			}
			files[filepath.ToSlash(rel)] = true
		}
	}
	return files, nil
}

// deadcode runs deadcode on ./... for goos and returns what it reports.
func deadcode(bin, goos string, test bool) ([]fn, error) {
	args := []string{"-json"}
	if test {
		args = append(args, "-test", "-tags", "e2e")
	}
	cmd := exec.Command(bin, append(args, "./...")...)
	cmd.Env = goEnv(goos)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, stderr.String())
	}
	return parse(out)
}

// parse reads deadcode's -json output.
func parse(out []byte) ([]fn, error) {
	var pkgs []report
	if err := json.Unmarshal(out, &pkgs); err != nil {
		return nil, fmt.Errorf("reading deadcode's output: %w", err)
	}
	var fns []fn
	for _, p := range pkgs {
		rel := strings.TrimPrefix(strings.TrimPrefix(p.Path, module), "/")
		if rel == "" {
			rel = "."
		}
		for _, f := range p.Funcs {
			fns = append(fns, fn{key: rel + "." + f.Name, pos: fmt.Sprintf("%s:%d", filepath.ToSlash(f.Position.File), f.Position.Line)})
		}
	}
	return fns, nil
}

// file is the file part of a function's position.
func (f fn) file() string { return f.pos[:strings.LastIndexByte(f.pos, ':')] }

// everywhere returns, sorted by key, the functions that every run reports whose system builds
// their file. A function in a file that one system alone builds needs only that system's report.
func everywhere(runs []*run) []fn {
	reported := map[string]int{}
	first := map[string]fn{}
	for _, r := range runs {
		seen := map[string]bool{}
		for _, f := range r.out {
			if !seen[f.key] {
				seen[f.key] = true
				reported[f.key]++
				if _, ok := first[f.key]; !ok {
					first[f.key] = f
				}
			}
		}
	}
	var out []fn
	for key, f := range first {
		builds := 0
		for _, r := range runs {
			if r.files[f.file()] {
				builds++
			}
		}
		if reported[key] >= builds {
			out = append(out, f)
		}
	}
	slices.SortFunc(out, func(a, b fn) int { return strings.Compare(a.key, b.key) })
	return out
}

// check returns one line for each problem: a function nothing calls, a function in a package
// that ships that only tests call and that allowed does not list, and an allowed function that is
// no longer test-only.
func check(unused, testsOnly []fn, allowed map[string]string) []string {
	var out []string
	for _, f := range unused {
		also := ""
		if _, ok := allowed[f.key]; ok {
			also = ", and its line in " + allowFile
		}
		out = append(out, fmt.Sprintf("%s: %s is never called, not even by a test: remove it%s", f.pos, f.key, also))
	}
	dead := map[string]bool{}
	for _, f := range unused {
		dead[f.key] = true
	}
	stillTestOnly := map[string]bool{}
	for _, f := range testsOnly {
		stillTestOnly[f.key] = true
		if dead[f.key] || inTestOnly(f.key) {
			continue
		}
		if _, ok := allowed[f.key]; !ok {
			out = append(out, fmt.Sprintf("%s: only tests call %s: move it into a _test.go file, or list it in %s with the reason", f.pos, f.key, allowFile))
		}
	}
	keys := make([]string, 0, len(allowed))
	for k := range allowed {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		if !stillTestOnly[k] {
			out = append(out, fmt.Sprintf("%s: %s is not a function that only tests call: remove its line", allowFile, k))
		}
	}
	return out
}

// inTestOnly reports whether key names a function in a test-only package.
func inTestOnly(key string) bool {
	return slices.ContainsFunc(testOnly, func(p string) bool { return strings.HasPrefix(key, p+".") })
}

// readAllowed reads allowFile: one function per line, as deadcheck names it, with the reason
// after '#'. Blank lines and lines that start with '#' are comments.
func readAllowed(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck // read-only
	return parseAllowed(f)
}

func parseAllowed(r io.Reader) (map[string]string, error) {
	out := map[string]string{}
	sc := bufio.NewScanner(r)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, reason, _ := strings.Cut(line, "#")
		key, reason = strings.TrimSpace(key), strings.TrimSpace(reason)
		switch {
		case strings.ContainsAny(key, " \t") || !strings.Contains(key, "."):
			return nil, fmt.Errorf("%s:%d: %q is not a function, such as internal/script.Validate", allowFile, n, key)
		case reason == "":
			return nil, fmt.Errorf("%s:%d: %s has no reason after '#'", allowFile, n, key)
		case out[key] != "":
			return nil, fmt.Errorf("%s:%d: %s is listed twice", allowFile, n, key)
		}
		out[key] = reason
	}
	return out, sc.Err()
}
