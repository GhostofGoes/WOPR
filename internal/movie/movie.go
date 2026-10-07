// Package movie is movie mode (docs/PLAN.md §7): wopr --movie replays the film's WOPR terminal
// scenes (internal/movie/scenes) as a self-running show, through the same console, typewriter,
// canvas and game code as interactive play. The Director is the root program in place of the
// persona: no LOGON, no Brain, no network.
package movie

import (
	"errors"
	"strconv"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/movie/scenes"
)

// Seed is movie mode's session seed. The director pins it, whatever --seed says, and runs
// deterministically, so every replay draws the same typing, the same games and the same
// numbers (RF-5).
const Seed uint64 = 1983

// Info describes a scene for --scenes and the menu.
type Info struct {
	Number int // 1-based, in film order
	Slug   string
	Title  string
	Blurb  string
}

// Scenes lists the film's scenes in order.
func Scenes() []Info { return infos(scenes.All()) }

func infos(list []scenes.Scene) []Info {
	out := make([]Info, len(list))
	for i, s := range list {
		out[i] = Info{Number: i + 1, Slug: s.Slug, Title: s.Title, Blurb: s.Blurb}
	}
	return out
}

// ErrNoScene is Resolve's error for a name that matches no scene.
var ErrNoScene = errors.New("no such scene")

// AmbiguousError is Resolve's error for a prefix of more than one scene's name.
type AmbiguousError struct{ Candidates []Info }

func (e *AmbiguousError) Error() string {
	names := make([]string, len(e.Candidates))
	for i, c := range e.Candidates {
		names[i] = c.Slug
	}
	return "could mean " + strings.Join(names, ", ")
}

// Resolve finds a scene the way games.Resolve finds a game: by number, slug, title (spaces
// for hyphens, any case) or a unique prefix of its slug.
func Resolve(arg string) (Info, error) {
	list := Scenes()
	i, err := resolve(list, arg)
	if err != nil {
		return Info{}, err
	}
	return list[i], nil
}

func resolve(list []Info, arg string) (int, error) {
	key := strings.Join(strings.FieldsFunc(strings.ToLower(arg), func(r rune) bool {
		return r == ' ' || r == '-' || r == '_'
	}), "-")
	if key == "" {
		return 0, ErrNoScene
	}
	if n, err := strconv.Atoi(key); err == nil {
		if n >= 1 && n <= len(list) {
			return n - 1, nil
		}
		return 0, ErrNoScene
	}
	var matches []int
	for i, s := range list {
		if s.Slug == key || strings.ToLower(strings.ReplaceAll(s.Title, " ", "-")) == key {
			return i, nil
		}
		if strings.HasPrefix(s.Slug, key) {
			matches = append(matches, i)
		}
	}
	switch len(matches) {
	case 0:
		return 0, ErrNoScene
	case 1:
		return matches[0], nil
	}
	amb := &AmbiguousError{}
	for _, i := range matches {
		amb.Candidates = append(amb.Candidates, list[i])
	}
	return 0, amb
}
