package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games/catalog"
	"github.com/GhostofGoes/WOPR/internal/golden"
)

func noEnv(string) string { return "" }

func TestParse(t *testing.T) {
	t.Parallel()
	reg := catalog.Registry()
	tests := []struct {
		args    string
		action  Action
		play    string
		instant bool
		movie   bool
		scene   string
		only    bool
		usage   string // non-empty: expect a UsageError containing this
	}{
		{args: "", action: RunTUI},
		{args: "gtw -i", play: "global-thermonuclear-war", instant: true},
		{args: "-i gtw", play: "global-thermonuclear-war", instant: true},
		{args: "-p gtw -i", play: "global-thermonuclear-war", instant: true},
		{args: "-p=chess", play: "chess"},
		{args: "--play chess", play: "chess"},
		{args: "-play chess", play: "chess"},
		{args: "15", play: "global-thermonuclear-war"},
		{args: "-i=false chess", play: "chess"},
		{args: "-- gtw", play: "global-thermonuclear-war"},
		{args: "-- gtw -i", usage: "too many arguments"},
		{args: "-t -- gtw", usage: "unknown theme"},
		{args: "gtw chess", usage: "too many arguments"},
		{args: "theaterwide", usage: "could mean"},
		{args: "pong", usage: "no game matches"},
		{args: "-p chess gtw", usage: "either with --play"},
		{args: "-pchess", usage: "flag provided but not defined"},
		{args: "-is 1", usage: "flag provided but not defined"},
		{args: "-s x", usage: "--seed"},
		{args: "-m", movie: true},
		{args: "-m 2 -i", movie: true, scene: "joshua", instant: true},
		{args: "--movie joshua", movie: true, scene: "joshua"},
		{args: "-i -m first-c", movie: true, scene: "first-contact", instant: true},
		{args: "-m first", usage: "could mean: first-contact, first-strike"},
		{args: "climax -m", movie: true, scene: "climax"},
		{args: "-m norad terminal", usage: "too many arguments"},
		{args: "-m nowhere", usage: "no scene matches \"nowhere\". The scenes are:\n 1. first-contact"},
		{args: "-m 7", usage: "no scene matches"},
		{args: "-m c", usage: "could mean: call-back, climax"},
		{args: "-S", action: PrintScenes},
		{args: "--scenes", action: PrintScenes},
		{args: "-m --scenes", action: PrintScenes},
		{args: "-m -p chess", usage: "cannot be combined"},
		{args: "-m joshua --only", movie: true, scene: "joshua", only: true},
		{args: "-o -m", movie: true, only: true},
		{args: "-m -o 2", movie: true, scene: "joshua", only: true},
		{args: "climax -o -m -i", movie: true, scene: "climax", only: true, instant: true},
		{args: "--only", usage: "--only needs --movie"},
		{args: "-o gtw", usage: "--only needs --movie"},
		{args: "-o -p chess", usage: "--only needs --movie"},
		{args: "-o=false gtw", play: "global-thermonuclear-war"},
		{args: "-o --scenes", action: PrintScenes},
		{args: "-v", action: PrintVersion},
		{args: "--version gtw", action: PrintVersion},
		{args: "-h", action: PrintHelp},
		{args: "-help", action: PrintHelp},
		{args: "-g", action: PrintGames},
		{args: "-games", action: PrintGames},
		{args: "--games", action: PrintGames},
		{args: "-L", action: PrintLicenses},
	}
	for _, tt := range tests {
		t.Run(tt.args, func(t *testing.T) {
			t.Parallel()
			cfg, act, err := Parse(strings.Fields(tt.args), reg, noEnv)
			if tt.usage != "" {
				var usageErr *UsageError
				if !errors.As(err, &usageErr) || !strings.Contains(err.Error(), tt.usage) {
					t.Fatalf("err = %v, want a usage error containing %q", err, tt.usage)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if act != tt.action || cfg.Play != tt.play || cfg.Instant != tt.instant || cfg.Movie != tt.movie || cfg.Scene != tt.scene ||
				cfg.Only != tt.only {
				t.Errorf("got action=%v play=%q instant=%v movie=%v scene=%q only=%v", act, cfg.Play, cfg.Instant, cfg.Movie, cfg.Scene, cfg.Only)
			}
		})
	}
}

func TestParseSeedAndEnv(t *testing.T) {
	t.Parallel()
	reg := catalog.Registry()
	cfg, _, err := Parse([]string{"-s", "42"}, reg, noEnv)
	if err != nil || !cfg.SeedSet || cfg.Seed != 42 {
		t.Fatalf("seed: %+v %v", cfg, err)
	}
	env := map[string]string{"WOPR_THEME": "norad", "WOPR_INSTANT": "1", "WOPR_REDUCE_MOTION": "yes", "WOPR_SEED": "1983"}
	cfg, _, err = Parse(nil, reg, func(k string) string { return env[k] })
	if err != nil || cfg.Theme != "norad" || !cfg.Instant || !cfg.ReduceMotion || !cfg.SeedSet || cfg.Seed != 1983 {
		t.Fatalf("env fallbacks: %+v %v", cfg, err)
	}
	cfg, _, err = Parse([]string{"-t", "green", "-i=false", "-s", "7"}, reg, func(k string) string { return env[k] })
	if err != nil || cfg.Theme != "green" || cfg.Instant || cfg.Seed != 7 {
		t.Fatalf("flags must override env: %+v %v", cfg, err)
	}
	if _, _, err := Parse(nil, reg, func(k string) string { return map[string]string{"WOPR_THEME": "pink"}[k] }); err == nil {
		t.Error("an unknown theme from the environment must be a usage error")
	}
}

// WOPR_SEED backs --seed: the flag wins, empty means unset, and a bad value is a usage error
// that names where it came from, as a bad --seed or WOPR_THEME is (exit 2), even in movie mode,
// which ignores the seed.
func TestParseSeed(t *testing.T) {
	t.Parallel()
	reg := catalog.Registry()
	tests := []struct {
		args   []string
		env    string // WOPR_SEED
		action Action
		seed   uint64
		set    bool
		usage  string // non-empty: expect a UsageError containing this
	}{
		{},
		{args: []string{"-s", "42"}, seed: 42, set: true},
		{args: []string{"--seed=0"}, seed: 0, set: true},
		{args: []string{"--seed", "18446744073709551615"}, seed: 1<<64 - 1, set: true},
		{env: "1983", seed: 1983, set: true},
		{env: "1983", args: []string{"chess"}, seed: 1983, set: true},
		{env: "1983", args: []string{"-s", "7"}, seed: 7, set: true},
		{env: "x", args: []string{"-s", "7"}, seed: 7, set: true},
		{env: "1983", args: []string{"--seed="}},
		{env: "1983", args: []string{"-s", ""}},
		{env: "x", args: []string{"-v"}, action: PrintVersion},
		{env: "x", usage: `WOPR_SEED needs a non-negative integer, got "x"`},
		{env: "-1", usage: `WOPR_SEED needs a non-negative integer, got "-1"`},
		{env: " 42", usage: "WOPR_SEED needs"},
		{env: "18446744073709551616", usage: "WOPR_SEED needs"},
		{env: "x", args: []string{"-m"}, usage: "WOPR_SEED needs"},
		{args: []string{"-s", "x"}, usage: `--seed needs a non-negative integer, got "x"`},
		{env: "1", args: []string{"--seed", "-1"}, usage: "--seed needs"},
		{env: "x", args: []string{"-s", "x"}, usage: "--seed needs"},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " ")+" WOPR_SEED="+tt.env, func(t *testing.T) {
			t.Parallel()
			cfg, act, err := Parse(tt.args, reg, func(k string) string { return map[string]string{"WOPR_SEED": tt.env}[k] })
			if tt.usage != "" {
				var usageErr *UsageError
				if !errors.As(err, &usageErr) || !strings.Contains(err.Error(), tt.usage) {
					t.Fatalf("err = %v, want a usage error containing %q", err, tt.usage)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if act != tt.action || cfg.Seed != tt.seed || cfg.SeedSet != tt.set {
				t.Errorf("got action=%v seed=%d set=%v, want %v %d %v", act, cfg.Seed, cfg.SeedSet, tt.action, tt.seed, tt.set)
			}
		})
	}
}

// R9: every long flag has a one-letter shorthand, and no letter is used twice.
func TestEveryLongFlagHasAShorthand(t *testing.T) {
	t.Parallel()
	seen := map[string]string{}
	for _, f := range flagPairs {
		if len(f.short) != 1 {
			t.Errorf("--%s has shorthand %q, want one letter", f.long, f.short)
		}
		if other, dup := seen[f.short]; dup {
			t.Errorf("-%s is used by --%s and --%s", f.short, other, f.long)
		}
		seen[f.short] = f.long
	}
	if _, taken := seen["l"]; taken {
		t.Error("-l is reserved for --llm (M7)")
	}
}

func TestRenderGolden(t *testing.T) {
	t.Parallel()
	var games, help, scenes strings.Builder
	if err := RenderGames(&games, catalog.Registry()); err != nil {
		t.Fatal(err)
	}
	golden.AssertString(t, "games", games.String())
	if err := RenderScenes(&scenes); err != nil {
		t.Fatal(err)
	}
	golden.AssertString(t, "scenes", scenes.String())
	if err := RenderHelp(&help); err != nil {
		t.Fatal(err)
	}
	golden.AssertString(t, "help", help.String())
	for _, line := range strings.Split(games.String()+help.String()+scenes.String(), "\n") {
		if len(line) > 80 {
			t.Errorf("line wider than 80 columns: %q", line)
		}
		if strings.ContainsRune(line, '\x1b') {
			t.Errorf("print-and-exit output must be plain text: %q", line)
		}
	}
}
