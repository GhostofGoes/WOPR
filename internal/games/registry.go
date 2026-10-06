// Package games defines what a game is and how games are found. It holds types only: the
// concrete games live in sub-packages, and internal/games/catalog wires them into a
// Registry (a game package imports this one, so this one cannot import them).
package games

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/prompt"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// Game is a program the persona can launch.
type Game interface{ proto.Program }

// Status says whether a game can be played yet.
type Status uint8

// Statuses.
const (
	Planned  Status = iota // listed, but WOPR declines in character
	Playable               // has a constructor
)

func (s Status) String() string {
	if s == Playable {
		return "playable"
	}
	return "planned"
}

// Info describes a game. The registry is the only source of it.
type Info struct {
	Number    int    // position on LIST GAMES, 1..n; 0 when not Listed
	Listed    bool   // shown by LIST GAMES (tic-tac-toe is not, as in the film)
	Name      string // the film's spelling, upper case
	Slug      string // lower-case words joined by '-'
	Aliases   []string
	Layout    proto.Layout
	PanelRows int // rows of View for LayoutPanel
	Status    Status
	Blurb     string // one line for --games and the README
}

// Entry pairs a game's Info with its constructor (nil while Planned).
type Entry struct {
	Info Info
	New  func() Game
}

// Registry is the ordered, validated set of games.
type Registry struct {
	entries []Entry
}

// NewRegistry validates entries and returns a registry in the given order.
func NewRegistry(entries ...Entry) (*Registry, error) {
	var errs []error
	keys := map[string]string{} // normalised slug/alias/name -> owning slug
	claim := func(key, slug, what string) {
		n := prompt.Normalize(key)
		if n == "" {
			errs = append(errs, fmt.Errorf("%s: empty %s", slug, what))
			return
		}
		if owner, dup := keys[n]; dup && owner != slug {
			errs = append(errs, fmt.Errorf("%s: %s %q collides with %s", slug, what, key, owner))
		}
		keys[n] = slug
	}
	var numbers []int
	slugs := map[string]bool{}
	for _, e := range entries {
		in := e.Info
		if slugs[in.Slug] {
			errs = append(errs, fmt.Errorf("%s: slug collides with another entry", in.Slug))
		}
		slugs[in.Slug] = true
		if !validSlug(in.Slug) {
			errs = append(errs, fmt.Errorf("invalid slug %q", in.Slug))
		}
		if in.Name == "" || in.Name != strings.ToUpper(in.Name) {
			errs = append(errs, fmt.Errorf("%s: name %q must be upper case", in.Slug, in.Name))
		}
		claim(in.Slug, in.Slug, "slug")
		claim(in.Name, in.Slug, "name")
		for _, a := range in.Aliases {
			claim(a, in.Slug, "alias")
			if _, isNumber := prompt.Number(a); isNumber {
				errs = append(errs, fmt.Errorf("%s: alias %q is a number, which Resolve reads as a list position", in.Slug, a))
			}
		}
		switch {
		case in.Listed && in.Number < 1:
			errs = append(errs, fmt.Errorf("%s: listed games need a Number from 1", in.Slug))
		case !in.Listed && in.Number != 0:
			errs = append(errs, fmt.Errorf("%s: unlisted games must have Number 0", in.Slug))
		case in.Listed:
			numbers = append(numbers, in.Number)
		}
		if in.Status == Playable && e.New == nil {
			errs = append(errs, fmt.Errorf("%s: playable without a constructor", in.Slug))
		}
		if in.Layout == proto.LayoutPanel && in.PanelRows < 1 {
			errs = append(errs, fmt.Errorf("%s: panel layout needs PanelRows", in.Slug))
		}
	}
	slices.Sort(numbers)
	for i, n := range numbers {
		if n != i+1 {
			errs = append(errs, fmt.Errorf("listed numbers must be 1..%d, each once; got %v", len(numbers), numbers))
			break
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return &Registry{entries: slices.Clone(entries)}, nil
}

func validSlug(s string) bool {
	if s == "" || s[0] == '-' || s[len(s)-1] == '-' || strings.Contains(s, "--") {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

// All returns every entry in registry order.
func (r *Registry) All() []Entry { return slices.Clone(r.entries) }

// Listed returns the entries shown by LIST GAMES, in Number order.
func (r *Registry) Listed() []Entry {
	var out []Entry
	for _, e := range r.entries {
		if e.Info.Listed {
			out = append(out, e)
		}
	}
	slices.SortFunc(out, func(a, b Entry) int { return a.Info.Number - b.Info.Number })
	return out
}

// Get returns the entry with the given slug.
func (r *Registry) Get(slug string) (Entry, bool) {
	for _, e := range r.entries {
		if e.Info.Slug == slug {
			return e, true
		}
	}
	return Entry{}, false
}

// Exact finds a game whose slug, alias or name equals input after normalisation. Unlike
// Resolve it never matches numbers or prefixes, so conversation cannot start a game by
// accident.
func (r *Registry) Exact(input string) (Entry, bool) {
	in := prompt.Normalize(input)
	if in == "" {
		return Entry{}, false
	}
	for _, e := range r.entries {
		keys := append([]string{e.Info.Slug, e.Info.Name}, e.Info.Aliases...)
		for _, k := range keys {
			if prompt.Normalize(k) == in {
				return e, true
			}
		}
	}
	return Entry{}, false
}

// ErrNotFound is returned by Resolve when nothing matches.
var ErrNotFound = errors.New("no such game")

// AmbiguousError is returned by Resolve when a prefix matches several games.
type AmbiguousError struct {
	Input      string
	Candidates []Entry
}

func (e *AmbiguousError) Error() string {
	names := make([]string, len(e.Candidates))
	for i, c := range e.Candidates {
		names[i] = c.Info.Slug
	}
	return fmt.Sprintf("%q matches several games: %s", e.Input, strings.Join(names, ", "))
}

// Resolve finds a game by, in order: listed number (1..n), slug, alias, exact name, then
// a unique prefix of a slug, alias or name. Matching ignores case and punctuation.
func (r *Registry) Resolve(input string) (Entry, error) {
	in := prompt.Normalize(input)
	if in == "" {
		return Entry{}, ErrNotFound
	}
	if n, ok := prompt.Number(input); ok {
		for _, e := range r.entries {
			if e.Info.Listed && e.Info.Number == n {
				return e, nil
			}
		}
		return Entry{}, ErrNotFound
	}
	exact := func(key func(Info) []string) (Entry, bool) {
		for _, e := range r.entries {
			for _, k := range key(e.Info) {
				if prompt.Normalize(k) == in {
					return e, true
				}
			}
		}
		return Entry{}, false
	}
	for _, key := range []func(Info) []string{
		func(i Info) []string { return []string{i.Slug} },
		func(i Info) []string { return i.Aliases },
		func(i Info) []string { return []string{i.Name} },
	} {
		if e, ok := exact(key); ok {
			return e, nil
		}
	}
	var matches []Entry
	for _, e := range r.entries {
		keys := append([]string{e.Info.Slug, e.Info.Name}, e.Info.Aliases...)
		for _, k := range keys {
			if strings.HasPrefix(prompt.Normalize(k), in) {
				matches = append(matches, e)
				break
			}
		}
	}
	switch len(matches) {
	case 0:
		return Entry{}, ErrNotFound
	case 1:
		return matches[0], nil
	default:
		return Entry{}, &AmbiguousError{Input: input, Candidates: matches}
	}
}
