// Command wopr is a terminal recreation of the WOPR computer from WarGames (1983).
package main

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"

	wopr "github.com/GhostofGoes/WOPR"
	"github.com/GhostofGoes/WOPR/internal/cli"
	"github.com/GhostofGoes/WOPR/internal/debuglog"
	"github.com/GhostofGoes/WOPR/internal/games/catalog"
	"github.com/GhostofGoes/WOPR/internal/ui"
	"github.com/GhostofGoes/WOPR/internal/version"
)

// Exit codes.
const (
	exitOK          = 0
	exitError       = 1
	exitUsage       = 2
	exitInterrupted = 130
)

func main() {
	// First: opened from WOPR.app in Finder, wopr has no terminal and reopens itself in Terminal.
	if code, relaunched := relaunchInTerminal(); relaunched {
		os.Exit(code)
	}
	ignoreSIGPIPE()
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, os.Getenv))
}

func run(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	reg := catalog.Registry()
	cfg, action, err := cli.Parse(args, reg, getenv)
	if err != nil {
		warn(stderr, "%v\nRun 'wopr --help' for usage.", err)
		return exitUsage
	}

	switch action {
	case cli.PrintVersion:
		return printed(stderr, writeString(stdout, version.Get().String()+"\n"))
	case cli.PrintHelp:
		return printed(stderr, cli.RenderHelp(stdout))
	case cli.PrintGames:
		return printed(stderr, cli.RenderGames(stdout, reg))
	case cli.PrintScenes:
		return printed(stderr, cli.RenderScenes(stdout))
	case cli.PrintLicenses:
		return printed(stderr, writeString(stdout, wopr.License+"\n"+wopr.Notice+"\n"+wopr.ThirdPartyNotices))
	case cli.RunTUI:
	}

	if msg := ui.TerminalProblem(getenv); msg != "" {
		warn(stderr, "%s%s", msg, windowsHint)
		return exitUsage
	}
	if !cfg.SeedSet {
		cfg.Seed = sessionSeed() // every session differs unless --seed or WOPR_SEED pins it
	}
	lg := openDebugLog(getenv, stderr)
	defer func() {
		if lg != nil {
			_ = lg.Close()
			warn(stderr, "debug log: %s", lg.Path())
		}
	}()
	lg.Printf("wopr %s, seed %d (pinned: %v)", version.Get(), cfg.Seed, cfg.SeedSet)
	outcome, err := ui.Run(ui.Options{
		Log: lg, Theme: cfg.Theme, Instant: cfg.Instant, Seed: cfg.Seed, SeedSet: cfg.SeedSet,
		ReduceMotion: cfg.ReduceMotion, Play: cfg.Play, Movie: cfg.Movie, Scene: cfg.Scene, Only: cfg.Only,
		NoColor: getenv("NO_COLOR") != "", Panel: panelSetting(getenv), Registry: reg,
	})
	lg.Printf("session ended: outcome %d, err %v", outcome, err)
	switch outcome {
	case ui.Finished:
		_ = writeString(stdout, "--CONNECTION TERMINATED--\n")
		return exitOK
	case ui.Interrupted:
		_ = writeString(stdout, "--CONNECTION TERMINATED--\n")
		return exitInterrupted
	case ui.Panicked:
		return exitError // Bubble Tea restored the terminal and printed the stack
	case ui.NoTerminal:
		warn(stderr, "%v%s", err, windowsHint)
		return exitUsage
	default:
		warn(stderr, "%v", err)
		return exitError
	}
}

// sessionSeed draws the session's seed from the operating system's random source.
func sessionSeed() uint64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return uint64(time.Now().UnixNano()) // crypto/rand does not fail on supported systems
	}
	return binary.LittleEndian.Uint64(b[:])
}

// openDebugLog opens the debug log when WOPR_DEBUG is set (docs/PLAN.md §8). Without a cache
// directory, or on any error, logging stays off and stderr says why.
func openDebugLog(getenv func(string) string, stderr io.Writer) *debuglog.Log {
	if !cli.Truthy(getenv("WOPR_DEBUG")) {
		return nil
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		warn(stderr, "WOPR_DEBUG: no cache directory (%v); not logging", err)
		return nil
	}
	lg, err := debuglog.Open(dir)
	if err != nil {
		warn(stderr, "WOPR_DEBUG: %v; not logging", err)
		return nil
	}
	return lg
}

// panelSetting reads WOPR_PANEL: unset leaves the theme's default (norad shows the front
// panel), otherwise it turns the panel on or off in any theme.
func panelSetting(getenv func(string) string) ui.Panel {
	switch v := getenv("WOPR_PANEL"); {
	case v == "":
		return ui.PanelDefault
	case cli.Truthy(v):
		return ui.PanelOn
	default:
		return ui.PanelOff
	}
}

func warn(w io.Writer, format string, a ...any) {
	_, _ = fmt.Fprintf(w, "wopr: "+format+"\n", a...)
}

func writeString(w io.Writer, s string) error {
	_, err := io.WriteString(w, s)
	return err
}

// printed maps the result of a print-and-exit action: a closed pipe (wopr --games | head)
// is success, any other write error is not.
func printed(stderr io.Writer, err error) int {
	if err == nil || isBrokenPipe(err) {
		return exitOK
	}
	warn(stderr, "%v", err)
	return exitError
}
