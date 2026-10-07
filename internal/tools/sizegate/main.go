// Command sizegate enforces the binary size budget on GoReleaser output.
//
//	go run ./internal/tools/sizegate [-expect 6] [dist/artifacts.json]
//
// It reads GoReleaser's artifact list, checks every Binary artifact against the budget
// (warn above 10 MB, fail above 15 MB; decimal megabytes), writes a Markdown table to
// $GITHUB_STEP_SUMMARY when set, and fails when no binaries are listed.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

const (
	warnBytes = 10_000_000
	failBytes = 15_000_000
)

type artifact struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Goos   string `json:"goos"`
	Goarch string `json:"goarch"`
	Type   string `json:"type"`
	Extra  struct {
		Format string `json:"Format"`
	} `json:"extra"`
}

type row struct {
	target string
	path   string
	size   int64
}

func main() { os.Exit(realMain()) }

func realMain() int {
	expect := flag.Int("expect", 0, "fail unless exactly this many binaries are listed (0 = any, at least one)")
	flag.Parse()
	path := "dist/artifacts.json"
	if flag.NArg() > 0 {
		path = flag.Arg(0)
	}
	var out, summary strings.Builder
	err := run(path, *expect, &out, &summary, statSize)
	fmt.Print(out.String())
	if p := os.Getenv("GITHUB_STEP_SUMMARY"); p != "" && summary.Len() > 0 {
		if werr := appendFile(p, summary.String()); werr != nil {
			fmt.Fprintln(os.Stderr, "sizegate:", werr)
			return 1
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "sizegate:", err)
		return 1
	}
	return 0
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

func statSize(p string) (int64, error) {
	fi, err := os.Stat(p)
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}

func run(path string, expect int, out, summary *strings.Builder, size func(string) (int64, error)) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var arts []artifact
	if err := json.Unmarshal(data, &arts); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	var rows []row
	for _, a := range arts {
		// A binary-format archive lists a built binary again, under its release name.
		if a.Type != "Binary" || a.Extra.Format == "binary" {
			continue
		}
		n, err := size(a.Path)
		if err != nil {
			return err
		}
		rows = append(rows, row{target: a.Goos + "/" + a.Goarch, path: a.Path, size: n})
	}
	return gate(rows, expect, out, summary)
}

func gate(rows []row, expect int, out, summary *strings.Builder) error {
	if len(rows) == 0 {
		return errors.New("no Binary artifacts listed; refusing to pass an empty build")
	}
	if expect > 0 && len(rows) != expect {
		return fmt.Errorf("expected %d binaries, found %d", expect, len(rows))
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].target < rows[j].target })
	fmt.Fprintln(summary, "| target | bytes | MB | budget |")
	fmt.Fprintln(summary, "|---|---:|---:|---|")
	var failed []string
	for _, r := range rows {
		status := "ok"
		switch {
		case r.size > failBytes:
			status = "FAIL (> 15 MB)"
			failed = append(failed, r.target)
			fmt.Fprintf(out, "::error::%s is %d bytes (> 15 MB hard limit)\n", r.path, r.size)
		case r.size > warnBytes:
			status = "warn (> 10 MB target)"
			fmt.Fprintf(out, "::warning::%s is %d bytes (> 10 MB target)\n", r.path, r.size)
		default:
			fmt.Fprintf(out, "%-14s %10d bytes  ok\n", r.target, r.size)
		}
		fmt.Fprintf(summary, "| %s | %d | %.2f | %s |\n", r.target, r.size, float64(r.size)/1e6, status)
	}
	if len(failed) > 0 {
		return fmt.Errorf("over the 15 MB hard limit: %v", failed)
	}
	return nil
}
