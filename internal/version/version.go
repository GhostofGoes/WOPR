// Package version reports what build of wopr is running.
//
// Release builds set Version, Commit and Date with -ldflags -X (see .goreleaser.yaml).
// Builds without them, such as `go install …@v0.1.0` or a plain `go build`, fall back to
// the module and VCS information the Go toolchain records in the binary.
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
)

// Set at link time by release builds.
var (
	Version = ""
	Commit  = ""
	Date    = ""
)

const unknown = "unknown"

// Info is the resolved build information.
type Info struct {
	Version   string // always starts with "v", or is "(devel)"
	Commit    string // short commit hash, "unknown" if not recorded
	Date      string // build or commit date (YYYY-MM-DD), "unknown" if not recorded
	Dirty     bool   // built from a modified working tree
	GoVersion string
}

// Get resolves the build information for the running binary.
func Get() Info {
	bi, _ := debug.ReadBuildInfo()
	return resolve(bi, Version, Commit, Date)
}

func resolve(bi *debug.BuildInfo, ldVersion, ldCommit, ldDate string) Info {
	info := Info{
		Version:   ldVersion,
		Commit:    ldCommit,
		Date:      ldDate,
		GoVersion: runtime.Version(),
	}
	if bi != nil {
		if info.Version == "" && bi.Main.Version != "" {
			info.Version = bi.Main.Version
		}
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				if info.Commit == "" {
					info.Commit = s.Value
				}
			case "vcs.time":
				if info.Date == "" {
					info.Date = s.Value
				}
			case "vcs.modified":
				info.Dirty = s.Value == "true"
			}
		}
		if bi.GoVersion != "" {
			info.GoVersion = bi.GoVersion
		}
	}
	info.Version = normalizeVersion(info.Version)
	info.Commit = shortCommit(info.Commit)
	info.Date = shortDate(info.Date)
	return info
}

func normalizeVersion(v string) string {
	switch {
	case v == "" || v == "(devel)":
		return "(devel)"
	case strings.HasPrefix(v, "v"):
		return v
	default:
		return "v" + v
	}
}

func shortCommit(c string) string {
	if c == "" {
		return unknown
	}
	if len(c) > 7 {
		return c[:7]
	}
	return c
}

func shortDate(d string) string {
	if d == "" {
		return unknown
	}
	// RFC 3339 timestamps (vcs.time, GoReleaser's CommitDate) keep only the date.
	if i := strings.IndexByte(d, 'T'); i == len("2006-01-02") {
		return d[:i]
	}
	return d
}

// String renders the one-line form printed by `wopr --version`.
func (i Info) String() string {
	commit := i.Commit
	if i.Dirty && commit != unknown {
		commit += "-dirty"
	}
	return fmt.Sprintf("wopr %s (commit %s, built %s, %s)", i.Version, commit, i.Date, i.GoVersion)
}
