// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

// Command branchcov measures the branch coverage of every package that ships, with gobco.
//
//	go run ./internal/tools/branchcov [-j 2] [-min 80] [-v] [package...]
//
// Run it from the repository root.
//
// Go's own -cover counts statements. gobco (pinned in tools/go.mod) counts each branch of each
// condition: an if, a for, a switch case and each operand of && and ||, true and false. It
// instruments one package at a time, so branchcov builds gobco once and runs it on each package,
// -j at a time. The packages default to ./..., less the ones only tests and tools use (testOnly).
//
// gobco type-checks every .go file in a package's directory, whatever its build constraints, so
// it fails on a package with files for other systems (debuglog_other.go beside debuglog_unix.go).
// branchcov therefore runs gobco on a copy of the module (the files git tracks, and new files git
// does not ignore) that leaves out the files the go command ignores for this system and the
// package.
//
// It prints a table of covered and total branches per package and the total, writes the same as
// Markdown to $GITHUB_STEP_SUMMARY when set, and fails when gobco fails or, with -min, when the
// total is below that percentage. -v also prints each branch that the tests never take.
package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
)

// testOnly are the packages that only tests and tools use, as relnotes counts them. They do not
// ship, so their coverage is not counted.
var testOnly = []string{
	"internal/tools", "internal/archtest", "internal/e2e", "internal/golden",
	"internal/games/testkit", "internal/games/gamestest",
}

// pkg is a package to measure.
type pkg struct {
	rel     string   // relative to the module root, with slashes: ".", "cmd/wopr"
	ignored []string // its files that build constraints exclude here, relative to the module root
}

// result is one package's coverage.
type result struct {
	pkg       string // relative to the module root: ".", "cmd/wopr", "internal/ui"
	covered   int
	total     int
	uncovered []string // gobco's lines for the branches never taken
}

func main() { os.Exit(realMain()) }

func realMain() int {
	jobs := flag.Int("j", max(1, runtime.NumCPU()/2), "packages to measure at once")
	minPct := flag.Float64("min", 0, "fail when the total branch coverage is below this percentage (0 = report only)")
	verbose := flag.Bool("v", false, "print each branch that the tests never take")
	flag.Parse()

	err := run(flag.Args(), *jobs, *minPct, *verbose)
	if err != nil {
		fmt.Fprintln(os.Stderr, "branchcov:", err)
		return 1
	}
	return 0
}

func run(patterns []string, jobs int, minPct float64, verbose bool) error {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	pkgs, err := listPackages(patterns)
	if err != nil {
		return err
	}
	if len(pkgs) == 0 {
		return errors.New("no packages to measure")
	}
	dir, err := os.MkdirTemp("", "branchcov")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	gobco := filepath.Join(dir, "gobco")
	if runtime.GOOS == "windows" {
		gobco += ".exe"
	}
	build := exec.Command("go", "build", "-modfile=tools/go.mod", "-o", gobco, "github.com/rillig/gobco")
	build.Stdout, build.Stderr = os.Stderr, os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("building gobco: %w", err)
	}

	files, err := moduleFiles()
	if err != nil {
		return err
	}
	results, err := measure(pkgs, max(1, jobs), func(p pkg) (result, error) {
		mod, err := os.MkdirTemp(dir, "module")
		if err != nil {
			return result{}, err
		}
		defer func() { _ = os.RemoveAll(mod) }()
		if err := copyFiles(mod, files, p.ignored); err != nil {
			return result{}, err
		}
		cmd := exec.Command(gobco, "-branch", "./"+p.rel)
		cmd.Dir = mod
		out, err := cmd.CombinedOutput()
		if err != nil {
			return result{}, fmt.Errorf("gobco %s: %w\n%s", p.rel, err, out)
		}
		return parse(p.rel, out)
	})
	if err != nil {
		return err
	}

	var table strings.Builder
	report(results, &table, verbose)
	fmt.Print(table.String())
	if p := os.Getenv("GITHUB_STEP_SUMMARY"); p != "" {
		var md strings.Builder
		markdown(results, &md)
		if err := appendFile(p, md.String()); err != nil {
			return err
		}
	}
	covered, total := sum(results)
	if pct := percent(covered, total); minPct > 0 && pct < minPct {
		return fmt.Errorf("branch coverage %.1f%% is below -min %.1f%%", pct, minPct)
	}
	return nil
}

// listPackages lists the packages that patterns name, less testOnly.
func listPackages(patterns []string) ([]pkg, error) {
	args := append([]string{"list", "-f", "{{.Dir}}{{range .IgnoredGoFiles}}\t{{.}}{{end}}"}, patterns...)
	out, err := exec.Command("go", args...).Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return nil, fmt.Errorf("go list: %w\n%s", err, ee.Stderr)
		}
		return nil, fmt.Errorf("go list: %w", err)
	}
	root, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	var pkgs []pkg
	for line := range strings.Lines(string(out)) {
		fields := strings.Split(strings.TrimRight(line, "\r\n"), "\t")
		rel, err := filepath.Rel(root, fields[0])
		if err != nil {
			return nil, err
		}
		p := pkg{rel: filepath.ToSlash(rel)}
		if !ships(p.rel) {
			continue
		}
		for _, f := range fields[1:] {
			p.ignored = append(p.ignored, path.Join(p.rel, f))
		}
		pkgs = append(pkgs, p)
	}
	return pkgs, nil
}

// moduleFiles lists the files to copy, relative to the module root, with slashes: those git
// tracks, and new ones it does not ignore, so that a test not yet committed counts.
func moduleFiles() ([]string, error) {
	out, err := exec.Command("git", "ls-files", "-z", "--cached", "--others", "--exclude-standard").Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}
	var files []string
	for f := range strings.SplitSeq(strings.TrimSuffix(string(out), "\x00"), "\x00") {
		if _, err := os.Lstat(f); err == nil { // a file deleted but not yet staged is still listed
			files = append(files, f)
		}
	}
	slices.Sort(files)
	return slices.Compact(files), nil
}

// copyFiles copies files from the working directory into dst, less skip.
func copyFiles(dst string, files, skip []string) error {
	for _, f := range files {
		if slices.Contains(skip, f) {
			continue
		}
		to := filepath.Join(dst, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
			return err
		}
		if err := copyFile(to, filepath.FromSlash(f)); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(dst, src string) error {
	fi, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if !fi.Mode().IsRegular() {
		return nil // a symbolic link or a submodule: nothing the tests read
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, fi.Mode().Perm())
}

// ships reports whether a package, relative to the module root, is in the program.
func ships(pkg string) bool {
	if pkg == ".." || strings.HasPrefix(pkg, "../") {
		return false // outside the module: run branchcov from the repository root
	}
	for _, p := range testOnly {
		if pkg == p || strings.HasPrefix(pkg, p+"/") {
			return false
		}
	}
	return true
}

// measure runs fn on each package, jobs at a time, and returns the results in pkgs's order and
// every error.
func measure(pkgs []pkg, jobs int, fn func(pkg) (result, error)) ([]result, error) {
	results := make([]result, len(pkgs))
	errs := make([]error, len(pkgs))
	next := make(chan int)
	var wg sync.WaitGroup
	for range min(jobs, len(pkgs)) {
		wg.Go(func() {
			for i := range next {
				results[i], errs[i] = fn(pkgs[i])
			}
		})
	}
	for i := range pkgs {
		next <- i
	}
	close(next)
	wg.Wait()
	return results, errors.Join(errs...)
}

// coverageLine is gobco's summary line: "Branch coverage: 72/80".
var coverageLine = regexp.MustCompile(`^Branch coverage: (\d+)/(\d+)$`)

// parse reads gobco's output for one package. A package with no conditions has no summary line.
func parse(pkg string, out []byte) (result, error) {
	r := result{pkg: pkg}
	found := false
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if m := coverageLine.FindStringSubmatch(line); m != nil {
			if found {
				return r, fmt.Errorf("%s: gobco printed two summaries", pkg)
			}
			found = true
			r.covered, _ = strconv.Atoi(m[1])
			r.total, _ = strconv.Atoi(m[2])
			continue
		}
		if found && strings.Contains(line, ": condition ") {
			r.uncovered = append(r.uncovered, line)
		}
	}
	if err := sc.Err(); err != nil {
		return r, err
	}
	if r.covered > r.total {
		return r, fmt.Errorf("%s: gobco counted %d of %d branches", pkg, r.covered, r.total)
	}
	return r, nil
}

func sum(results []result) (covered, total int) {
	for _, r := range results {
		covered += r.covered
		total += r.total
	}
	return covered, total
}

// percent is covered/total as a percentage. No branches count as fully covered.
func percent(covered, total int) float64 {
	if total == 0 {
		return 100
	}
	return 100 * float64(covered) / float64(total)
}

// report writes the plain table, sorted by package, then the total.
func report(results []result, out *strings.Builder, verbose bool) {
	rs := slices.Clone(results)
	slices.SortFunc(rs, func(a, b result) int { return strings.Compare(a.pkg, b.pkg) })
	width := len("total")
	for _, r := range rs {
		width = max(width, len(r.pkg))
	}
	for _, r := range rs {
		fmt.Fprintf(out, "%-*s %5d/%-5d %6.1f%%\n", width, r.pkg, r.covered, r.total, percent(r.covered, r.total))
		if verbose {
			for _, u := range r.uncovered {
				fmt.Fprintf(out, "    %s\n", u)
			}
		}
	}
	covered, total := sum(rs)
	fmt.Fprintf(out, "%-*s %5d/%-5d %6.1f%%\n", width, "total", covered, total, percent(covered, total))
}

// markdown writes the same table for a GitHub job summary.
func markdown(results []result, out *strings.Builder) {
	rs := slices.Clone(results)
	slices.SortFunc(rs, func(a, b result) int { return strings.Compare(a.pkg, b.pkg) })
	fmt.Fprintln(out, "### Branch coverage (gobco)")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "| package | branches | covered | % |")
	fmt.Fprintln(out, "|---|---:|---:|---:|")
	for _, r := range rs {
		fmt.Fprintf(out, "| `%s` | %d | %d | %.1f |\n", r.pkg, r.total, r.covered, percent(r.covered, r.total))
	}
	covered, total := sum(rs)
	fmt.Fprintf(out, "| **total** | %d | %d | **%.1f** |\n", total, covered, percent(covered, total))
}

func appendFile(path, text string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(text); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
