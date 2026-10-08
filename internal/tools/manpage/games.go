package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/cli"
	"github.com/GhostofGoes/WOPR/internal/games"
)

// Where the per-game pages and their screenshots live, from the repository root. The docs
// site renders its game pages from the same files.
const (
	gamesDir = "site/data/games"
	shotsDir = "site/static/img/games"
)

// game is one site/data/games/<slug>.json file.
type game struct {
	Slug        string       `json:"slug"`   // the shortest name wopr --games shows (cli.Handle)
	Number      *int         `json:"number"` // on LIST GAMES; null for a game not on it
	Name        string       `json:"name"`   // the catalog's name in title case
	Launch      string       `json:"launch"` // "wopr <slug>"
	Aliases     []string     `json:"aliases"`
	Summary     string       `json:"summary"`
	HowToPlay   []string     `json:"howToPlay"`
	Controls    []control    `json:"controls"`
	Tips        []string     `json:"tips"` // the manual page shows the first manTips
	Screenshots []screenshot `json:"screenshots"`
}

type control struct {
	Input string `json:"input"`
	Does  string `json:"does"`
}

type screenshot struct {
	File    string `json:"file"`
	Caption string `json:"caption"`
}

// manTips is how many tips the manual page gives for each game; every file has at least
// this many.
const manTips = 3

// loadGames reads the page of every game players can choose, in the catalog's order, and
// checks each against the catalog: see check.
func loadGames(root string, reg *games.Registry) ([]game, error) {
	var out []game
	var errs []error
	want := map[string]bool{}
	for _, e := range reg.All() {
		slug := cli.Handle(e.Info)
		want[slug+".json"] = true
		g, err := readGame(filepath.Join(root, gamesDir, slug+".json"))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if err := check(g, e, reg, root); err != nil {
			errs = append(errs, fmt.Errorf("%s/%s.json: %w", gamesDir, slug, err))
			continue
		}
		out = append(out, g)
	}
	files, err := os.ReadDir(filepath.Join(root, gamesDir))
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if !want[f.Name()] {
			errs = append(errs, fmt.Errorf("%s/%s: no game in the catalog has this slug", gamesDir, f.Name()))
		}
	}
	return out, errors.Join(errs...)
}

func readGame(path string) (game, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return game{}, err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields() // a misspelt field is an error, not a silently empty one
	var g game
	if err := dec.Decode(&g); err != nil {
		return game{}, fmt.Errorf("%s: %w", path, err)
	}
	return g, nil
}

var shotName = regexp.MustCompile(`^[a-z0-9-]+-[1-9][0-9]*\.png$`)

// check holds a game's page to the catalog entry it describes: the slug and launch command
// wopr --games shows, the number, the name, aliases that are exactly the other names wopr
// accepts for it, every text field filled in, at least manTips tips, and screenshots that
// exist. Text is printable ASCII, as on the screen.
func check(g game, e games.Entry, reg *games.Registry, root string) error {
	var errs []error
	fail := func(format string, a ...any) { errs = append(errs, fmt.Errorf(format, a...)) }
	in := e.Info
	if want := cli.Handle(in); g.Slug != want {
		fail("slug %q, want %q (as wopr --games shows it)", g.Slug, want)
	}
	if want := "wopr " + g.Slug; g.Launch != want {
		fail("launch %q, want %q", g.Launch, want)
	}
	switch {
	case in.Listed && (g.Number == nil || *g.Number != in.Number):
		fail("number %v, want %d", g.Number, in.Number)
	case !in.Listed && g.Number != nil:
		fail("number %d for a game not on LIST GAMES; want null", *g.Number)
	}
	if !strings.EqualFold(g.Name, in.Name) || g.Name == strings.ToUpper(g.Name) {
		fail("name %q, want %q in title case", g.Name, in.Name)
	}

	// The aliases and the slug together are every other name the catalog gives the game.
	want := append([]string{in.Slug}, in.Aliases...)
	if in.Listed {
		want = append(want, strconv.Itoa(in.Number))
	}
	have := append([]string{g.Slug}, g.Aliases...)
	slices.Sort(want)
	slices.Sort(have)
	if !slices.Equal(slices.Compact(want), slices.Compact(have)) {
		fail("slug and aliases %q, want %q", have, want)
	}
	for _, a := range g.Aliases {
		if r, err := reg.Resolve(a); err != nil || r.Info.Slug != in.Slug {
			fail("alias %q does not start this game", a)
		}
	}

	var controls, shots []string
	for _, c := range g.Controls {
		controls = append(controls, c.Input, c.Does)
	}
	for _, s := range g.Screenshots {
		shots = append(shots, s.File, s.Caption)
	}
	texts := map[string][]string{
		"summary": {g.Summary}, "howToPlay": g.HowToPlay, "tips": g.Tips, "aliases": g.Aliases,
		"controls": controls, "screenshots": shots,
	}
	for _, field := range slices.Sorted(maps.Keys(texts)) {
		if len(texts[field]) == 0 {
			fail("%s is empty", field)
		}
		for _, t := range texts[field] {
			if strings.TrimSpace(t) == "" {
				fail("%s has an empty entry", field)
			}
			if i := strings.IndexFunc(t, func(r rune) bool { return r < ' ' || r > '~' }); i >= 0 {
				fail("%s: %q is not printable ASCII at byte %d", field, t, i)
			}
		}
	}
	if n := len(g.HowToPlay); n < 3 || n > 6 {
		fail("howToPlay has %d entries, want 3 to 6", n)
	}
	if len(g.Tips) < manTips {
		fail("tips has %d entries, want at least %d", len(g.Tips), manTips)
	}
	if len(g.Screenshots) < 2 {
		fail("screenshots has %d entries, want at least 2", len(g.Screenshots))
	}
	for _, s := range g.Screenshots {
		if !shotName.MatchString(s.File) || !strings.HasPrefix(s.File, g.Slug+"-") {
			fail("screenshot %q is not named %s-<n>.png", s.File, g.Slug)
		}
		if _, err := os.Stat(filepath.Join(root, shotsDir, s.File)); err != nil {
			fail("screenshot %s: %w", s.File, err)
		}
	}
	return errors.Join(errs...)
}
