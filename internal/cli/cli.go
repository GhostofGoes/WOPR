// Package cli parses wopr's command line and renders the print-and-exit outputs.
//
// It uses the standard library flag package. Every long flag has a one-letter shorthand
// (enforced by TestEveryLongFlagHasAShorthand), and a bare positional argument means
// --play (or a scene with --movie). Flags may come before or after the positional.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/theme"
)

// Action is what main should do.
type Action int

// Actions.
const (
	RunTUI Action = iota
	PrintVersion
	PrintHelp
	PrintGames
	PrintLicenses
)

// Config is the parsed command line.
type Config struct {
	Play         string // resolved game slug, "" for none
	Movie        bool
	Scene        string // the scene argument as typed, with Movie
	Theme        string
	Instant      bool
	Seed         uint64
	SeedSet      bool
	ReduceMotion bool
}

// UsageError is a command-line mistake; main prints it with the usage hint and exits 2.
type UsageError struct{ Msg string }

func (e *UsageError) Error() string { return e.Msg }

func usagef(format string, a ...any) error { return &UsageError{Msg: fmt.Sprintf(format, a...)} }

// flagPair is one option: its long name, its shorthand, and its help text.
type flagPair struct {
	long, short, arg, help, env string
}

var flagPairs = []flagPair{
	{"version", "v", "", "print the version and exit", ""},
	{"help", "h", "", "print this help and exit", ""},
	{"games", "g", "", "list the games and exit", ""},
	{"licenses", "L", "", "print licence notices and exit", ""},
	{"play", "p", "game", "start a game: number, name, alias or prefix", ""},
	{"movie", "m", "", "replay the film's WOPR scenes [scene]", ""},
	{"theme", "t", "name", strings.Join(theme.Names(), ", "), "WOPR_THEME"},
	{"instant", "i", "", "no typewriter pacing", "WOPR_INSTANT"},
	{"seed", "s", "n", "deterministic run with this seed", ""},
	{"reduce-motion", "r", "", "no blinking or motion", "WOPR_REDUCE_MOTION"},
}

// Parse parses args (without the program name). getenv supplies environment fallbacks.
func Parse(args []string, reg *games.Registry, getenv func(string) string) (Config, Action, error) {
	var (
		cfg                                Config
		version, help, list, licences, mov bool
		play, seed                         string
	)
	cfg.Theme = theme.Default
	if v := getenv("WOPR_THEME"); v != "" {
		cfg.Theme = v
	}
	cfg.Instant = truthy(getenv("WOPR_INSTANT"))
	cfg.ReduceMotion = truthy(getenv("WOPR_REDUCE_MOTION"))

	fs := flag.NewFlagSet("wopr", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	boolVar := func(p *bool, f flagPair) {
		fs.BoolVar(p, f.long, *p, f.help)
		fs.BoolVar(p, f.short, *p, f.help)
	}
	strVar := func(p *string, f flagPair) {
		fs.StringVar(p, f.long, *p, f.help)
		fs.StringVar(p, f.short, *p, f.help)
	}
	byName := map[string]flagPair{}
	for _, f := range flagPairs {
		byName[f.long] = f
	}
	boolVar(&version, byName["version"])
	boolVar(&help, byName["help"])
	boolVar(&list, byName["games"])
	boolVar(&licences, byName["licenses"])
	strVar(&play, byName["play"])
	boolVar(&mov, byName["movie"])
	strVar(&cfg.Theme, byName["theme"])
	boolVar(&cfg.Instant, byName["instant"])
	strVar(&seed, byName["seed"])
	boolVar(&cfg.ReduceMotion, byName["reduce-motion"])

	positionals, err := parseInterspersed(fs, args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return cfg, PrintHelp, nil
		}
		return cfg, RunTUI, usagef("%v", err)
	}
	switch {
	case help:
		return cfg, PrintHelp, nil
	case version:
		return cfg, PrintVersion, nil
	case list:
		return cfg, PrintGames, nil
	case licences:
		return cfg, PrintLicenses, nil
	}

	if !theme.ValidName(cfg.Theme) {
		return cfg, RunTUI, usagef("unknown theme %q (choose from %s)", cfg.Theme, strings.Join(theme.Names(), ", "))
	}
	if seed != "" {
		n, err := strconv.ParseUint(seed, 10, 64)
		if err != nil {
			return cfg, RunTUI, usagef("--seed needs a non-negative integer, got %q", seed)
		}
		cfg.Seed, cfg.SeedSet = n, true
	}
	if len(positionals) > 1 {
		return cfg, RunTUI, usagef("too many arguments: %s", strings.Join(positionals, " "))
	}
	cfg.Movie = mov
	if mov {
		if play != "" {
			return cfg, RunTUI, usagef("--movie and --play cannot be combined")
		}
		if len(positionals) == 1 {
			cfg.Scene = positionals[0]
		}
		return cfg, RunTUI, nil
	}
	if len(positionals) == 1 {
		if play != "" {
			return cfg, RunTUI, usagef("give the game either with --play or as an argument, not both")
		}
		play = positionals[0]
	}
	if play != "" {
		e, err := reg.Resolve(play)
		var amb *games.AmbiguousError
		switch {
		case errors.As(err, &amb):
			names := make([]string, len(amb.Candidates))
			for i, c := range amb.Candidates {
				names[i] = c.Info.Slug
			}
			return cfg, RunTUI, usagef("%q could mean: %s", play, strings.Join(names, ", "))
		case err != nil:
			return cfg, RunTUI, usagef("no game matches %q (try wopr --games)", play)
		}
		cfg.Play = e.Info.Slug
	}
	return cfg, RunTUI, nil
}

// parseInterspersed lets flags follow positionals ("wopr gtw -i"). The standard flag
// package stops at the first positional, so parsing resumes after each one. A "--"
// terminator ends flag parsing for good.
func parseInterspersed(fs *flag.FlagSet, args []string) ([]string, error) {
	var positionals []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		rest := fs.Args()
		consumed := len(args) - len(rest)
		if consumed > 0 && args[consumed-1] == "--" {
			return append(positionals, rest...), nil
		}
		if len(rest) == 0 {
			return positionals, nil
		}
		positionals = append(positionals, rest[0])
		args = rest[1:]
	}
}

func truthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "0", "false", "no", "off":
		return false
	}
	return true
}
