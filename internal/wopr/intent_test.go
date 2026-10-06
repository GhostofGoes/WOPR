package wopr

import (
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
