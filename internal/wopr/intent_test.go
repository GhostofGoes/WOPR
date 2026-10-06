package wopr

import (
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games/catalog"
)

// FuzzGameIntent checks that intent detection never panics and only returns real entries.
func FuzzGameIntent(f *testing.F) {
	for _, s := range []string{
		"Love to. How about Global Thermonuclear War?", "Later. Let's play Global Thermonuclear War.",
		"I don't want to play chess", "play", "how about", "LETS PLAY A GAME OF", "chess", "...", "",
	} {
		f.Add(s)
	}
	reg := catalog.Registry()
	f.Fuzz(func(t *testing.T, in string) {
		e, ok := gameIntent(in, reg)
		if !ok {
			return
		}
		if got, found := reg.Get(e.Info.Slug); !found || got.Info.Name != e.Info.Name {
			t.Fatalf("%q: intent returned an entry not in the registry: %+v", in, e.Info)
		}
	})
}

// The intent table: explicit requests start a game; anything that only mentions a game, a
// number that is not the whole request, or a negated request does not.
func TestIntentTable(t *testing.T) {
	t.Parallel()
	reg := catalog.Registry()
	starts := map[string]string{
		"Love to. How about Global Thermonuclear War?": "global-thermonuclear-war",
		"Later. Let's play Global Thermonuclear War.":  "global-thermonuclear-war",
		"Let's play chess now":                         "chess",
		"How about chess instead?":                     "chess",
		"Let's play a game of chess please":            "chess",
		"How about a nice game of chess?":              "chess",
		"I want to play chess with you":                "chess",
		"Shall we play chess":                          "chess",
		"Let\u2019s play chess":                        "chess", // a pasted curly apostrophe
		"Now let's play chess":                         "chess",
		"play 7":                                       "chess",
		"How about 15?":                                "global-thermonuclear-war",
		"chess":                                        "chess",
	}
	for in, want := range starts {
		if e, ok := gameIntent(in, reg); !ok || e.Info.Slug != want {
			t.Errorf("%q: got %q (%v), want %s", in, e.Info.Slug, ok, want)
		}
	}
	for _, in := range []string{
		"Let's play 2 games of chess", "How about 1 more round?", "Play 3 card monte",
		"I don't want to play chess", "Let's not play chess", "Let's play chess and checkers",
		"Chess is a good game.", "I played chess yesterday", "How about that?", "play",
	} {
		if e, ok := gameIntent(in, reg); ok {
			t.Errorf("%q must not start a game, started %s", in, e.Info.Slug)
		}
	}
	if strings.Contains(trimTrailers("CHESS NOW PLEASE"), " ") {
		t.Error("trailers are trimmed repeatedly")
	}
}
