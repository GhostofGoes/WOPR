// Command cooldown enforces the dependency cooldown: no module in any build list (wopr's, the
// four tool modules' and the docs site's) may be newer than 14 days at the commit being checked.
//
//	go run ./internal/tools/cooldown [-days 14] [-at 2026-10-10T00:00:00Z | -now]
//
// A module's age is its version's time as the Go module proxy reports it (the commit time of a
// tag or pseudo-version), measured against the committer date of HEAD (or -at; or now, with -now
// or outside a git checkout). Measuring against HEAD rather than the clock means the check fails on the
// commit that bumps a module too early and passes on later commits once the module is old
// enough, so updating a dependency waits out the cooldown, as `prek update` does for hooks
// (prek.toml's cooldown_days).
//
// tools/cooldown-exceptions.txt lists module@version pairs exempt from the wait, one per line,
// each with a reason after '#': a fix for a vulnerability that cannot wait, say. Remove a line
// once its version is old enough.
//
// It exits 0 when every module is old enough, 1 when one is not, and 2 when it could not
// check (go list or git failed).
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// exceptionsFile lists module@version pairs that may skip the cooldown.
const exceptionsFile = "tools/cooldown-exceptions.txt"

// buildList is one module whose dependencies are checked: how to run go list for it.
type buildList struct {
	name string   // as reported
	args []string // go flags before "list -m -json all" (-C dir)
}

// buildLists are wopr's own module, the four tool modules (AGENTS.md, "Commands") and the docs
// site's Hugo modules.
var buildLists = []buildList{
	{"go.mod", nil},
	{"tools/go.mod", []string{"-C", "tools"}},
	{"tools/lint/go.mod", []string{"-C", "tools/lint"}},
	{"tools/release/go.mod", []string{"-C", "tools/release"}},
	{"tools/docs/go.mod", []string{"-C", "tools/docs"}},
	{"site/go.mod", []string{"-C", "site"}},
}

type module struct {
	Path    string
	Version string
	Time    *time.Time
	Main    bool
	Replace *module
}

// finding is a module version still inside the cooldown.
type finding struct {
	list, path, version string
	time                time.Time
}

func main() { os.Exit(realMain()) }

func realMain() int {
	days := flag.Int("days", 14, "the cooldown, in days")
	atFlag := flag.String("at", "", "measure ages at this RFC 3339 time instead of HEAD's committer date")
	now := flag.Bool("now", false, "measure ages now, for a commit not yet made (the pre-commit hook)")
	flag.Parse()

	if *now {
		*atFlag = time.Now().UTC().Format(time.RFC3339)
	}
	at, err := referenceTime(*atFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cooldown: %v\n", err)
		return 2
	}
	allowed, err := readExceptions(exceptionsFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cooldown: %v\n", err)
		return 2
	}
	var found []finding
	for _, bl := range buildLists {
		mods, err := goList(bl.args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cooldown: %s: %v\n", bl.name, err)
			return 2
		}
		f, err := check(bl.name, mods, at, time.Duration(*days)*24*time.Hour, allowed)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cooldown: %v\n", err)
			return 2
		}
		found = append(found, f...)
	}
	if len(found) == 0 {
		fmt.Printf("cooldown: every module is at least %d days old at %s\n", *days, at.Format(time.RFC3339))
		return 0
	}
	fmt.Fprint(os.Stderr, report(found, at, *days))
	return 1
}

// referenceTime is -at, else HEAD's committer date, else now (outside a git checkout).
func referenceTime(at string) (time.Time, error) {
	if at != "" {
		return time.Parse(time.RFC3339, at)
	}
	out, err := exec.Command("git", "log", "-1", "--format=%cI", "HEAD").Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cooldown: no git history; measuring against the current time")
		return time.Now(), nil //nolint:nilerr // outside a checkout, now is the honest reference
	}
	return time.Parse(time.RFC3339, strings.TrimSpace(string(out)))
}

func goList(args []string) ([]module, error) {
	cmd := exec.Command("go", append(args, "list", "-m", "-json", "all")...)
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=readonly")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list: %w: %s", err, stderr.String())
	}
	var mods []module
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var m module
		if err := dec.Decode(&m); errors.Is(err, io.EOF) {
			return mods, nil
		} else if err != nil {
			return nil, err
		}
		mods = append(mods, m)
	}
}

// check returns the modules in one build list that are younger than cooldown at at.
func check(list string, mods []module, at time.Time, cooldown time.Duration, allowed map[string]bool) ([]finding, error) {
	var out []finding
	for _, m := range mods {
		if m.Main {
			continue
		}
		if m.Replace != nil {
			m = *m.Replace // the code that is actually built
		}
		if m.Version == "" {
			continue // a replacement by a local directory has no version and no proxy time
		}
		if m.Time == nil {
			return nil, fmt.Errorf("%s: no time for %s@%s", list, m.Path, m.Version)
		}
		if at.Sub(*m.Time) >= cooldown || allowed[m.Path+"@"+m.Version] {
			continue
		}
		out = append(out, finding{list: list, path: m.Path, version: m.Version, time: *m.Time})
	}
	return out, nil
}

func report(found []finding, at time.Time, days int) string {
	var w strings.Builder
	sort.Slice(found, func(i, j int) bool {
		if found[i].list != found[j].list {
			return found[i].list < found[j].list
		}
		return found[i].path < found[j].path
	})
	fmt.Fprintf(&w, "cooldown: these modules are less than %d days old at %s:\n", days, at.Format(time.RFC3339))
	for _, f := range found {
		ready := f.time.Add(time.Duration(days) * 24 * time.Hour)
		fmt.Fprintf(&w, "  %s: %s@%s, from %s (allowed from %s)\n",
			f.list, f.path, f.version, f.time.Format(time.DateOnly), ready.Format(time.DateOnly))
	}
	fmt.Fprintf(&w, "Use an older version, wait, or add module@version with a reason to %s.\n", exceptionsFile)
	return w.String()
}

// readExceptions reads module@version lines; '#' starts a comment, which a listed pair needs.
func readExceptions(path string) (map[string]bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck // read-only
	return parseExceptions(f)
}

func parseExceptions(r io.Reader) (map[string]bool, error) {
	allowed := map[string]bool{}
	sc := bufio.NewScanner(r)
	for n := 1; sc.Scan(); n++ {
		line, reason, _ := strings.Cut(sc.Text(), "#")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.Contains(line, "@") || strings.ContainsAny(line, " \t") {
			return nil, fmt.Errorf("%s:%d: want module@version, got %q", exceptionsFile, n, line)
		}
		if strings.TrimSpace(reason) == "" {
			return nil, fmt.Errorf("%s:%d: %s needs a reason after '#'", exceptionsFile, n, line)
		}
		allowed[line] = true
	}
	return allowed, sc.Err()
}
