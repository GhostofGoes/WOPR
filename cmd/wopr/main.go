// Command wopr is a terminal recreation of the WOPR computer from WarGames (1983).
package main

import (
	"fmt"
	"io"
	"os"

	wopr "github.com/GhostofGoes/WOPR"
	"github.com/GhostofGoes/WOPR/internal/cli"
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
	case cli.PrintLicenses:
		return printed(stderr, writeString(stdout, wopr.License+"\n"+wopr.Notice+"\n"+wopr.ThirdPartyNotices))
	case cli.RunTUI:
	}

	if msg := ui.TerminalProblem(getenv); msg != "" {
		warn(stderr, "%s%s", msg, windowsHint)
		return exitUsage
	}
	outcome, err := ui.Run(ui.Options{
		Theme: cfg.Theme, Instant: cfg.Instant, Seed: cfg.Seed, SeedSet: cfg.SeedSet,
		ReduceMotion: cfg.ReduceMotion, Play: cfg.Play, Movie: cfg.Movie, Scene: cfg.Scene,
		NoColor: getenv("NO_COLOR") != "",
	})
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
