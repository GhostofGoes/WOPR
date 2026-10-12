// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package games

import (
	"errors"
	"testing"
)

func resolveRegistry(t *testing.T) *Registry {
	t.Helper()
	stub := func() Game { return nil }
	r, err := NewRegistry(
		Entry{Info: Info{Number: 1, Listed: true, Name: "FALKEN'S MAZE", Slug: "falkens-maze", Status: Playable}, New: stub},
		Entry{Info: Info{Number: 2, Listed: true, Name: "BLACK JACK", Slug: "blackjack", Aliases: []string{"TWENTY-ONE"}, Status: Playable}, New: stub},
		Entry{Info: Info{Number: 3, Listed: true, Name: "BRIDGE", Slug: "bridge"}},
		Entry{Info: Info{Name: "TIC-TAC-TOE", Slug: "tic-tac-toe", Aliases: []string{"NOUGHTS AND CROSSES"}, Status: Playable}, New: stub},
	)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// Resolve tries the listed number, the slug, an alias, the name, then a unique prefix; it
// ignores case and punctuation, and names every game a shared prefix could mean.
func TestResolve(t *testing.T) {
	t.Parallel()
	r := resolveRegistry(t)
	for in, want := range map[string]string{
		"1": "falkens-maze", "3": "bridge",
		"blackjack": "blackjack", "Twenty One": "blackjack", "falken's maze": "falkens-maze",
		"FALK": "falkens-maze", "noughts": "tic-tac-toe", "tic tac": "tic-tac-toe", "BR": "bridge",
	} {
		e, err := r.Resolve(in)
		if err != nil || e.Info.Slug != want {
			t.Errorf("Resolve(%q) = %q, %v; want %q", in, e.Info.Slug, err, want)
		}
	}
	for _, in := range []string{"", "  ", "?!", "0", "4", "CHESS"} {
		if _, err := r.Resolve(in); !errors.Is(err, ErrNotFound) {
			t.Errorf("Resolve(%q): %v, want ErrNotFound", in, err)
		}
	}
	_, err := r.Resolve("b")
	var amb *AmbiguousError
	if !errors.As(err, &amb) || len(amb.Candidates) != 2 || amb.Error() != `"b" matches several games: blackjack, bridge` {
		t.Errorf("Resolve(\"b\"): %v", err)
	}
}

// Exact matches a whole slug, name or alias only.
func TestExact(t *testing.T) {
	t.Parallel()
	r := resolveRegistry(t)
	for in, want := range map[string]string{"bridge": "bridge", "Noughts and Crosses": "tic-tac-toe", "BLACK JACK": "blackjack"} {
		if e, ok := r.Exact(in); !ok || e.Info.Slug != want {
			t.Errorf("Exact(%q) = %q, %v; want %q", in, e.Info.Slug, ok, want)
		}
	}
	for _, in := range []string{"", "1", "BRI", "chess"} {
		if e, ok := r.Exact(in); ok {
			t.Errorf("Exact(%q) found %q", in, e.Info.Slug)
		}
	}
	if _, ok := r.Get("nope"); ok {
		t.Error("Get found a slug that is not there")
	}
}

func TestValidSlugAndStatus(t *testing.T) {
	t.Parallel()
	for s, want := range map[string]bool{"chess": true, "tic-tac-toe": true, "gtw2": true, "": false, "-a": false, "a-": false, "a--b": false, "A": false, "a b": false} {
		if got := validSlug(s); got != want {
			t.Errorf("validSlug(%q) = %v, want %v", s, got, want)
		}
	}
	if Playable.String() != "playable" || Planned.String() != "planned" {
		t.Errorf("statuses: %s, %s", Playable, Planned)
	}
	if _, err := NewRegistry(Entry{Info: Info{Number: 1, Listed: true, Name: "??", Slug: "x"}}); err == nil {
		t.Error("a name that normalises to nothing was accepted")
	}
}
