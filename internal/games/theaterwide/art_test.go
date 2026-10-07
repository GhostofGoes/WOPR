package theaterwide

import (
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/sim"
)

// The pictures are printable ASCII capitals within 80 columns and in Lines, and the intro,
// the first map and the first prompt fit a 23-row console.
func TestArt(t *testing.T) {
	t.Parallel()
	for _, bad := range sim.ArtProblems(Lines, artTitle, artFallout, artRuins) {
		t.Error(bad)
	}
	g := New().(*sim.Game)
	g.Start(proto.Env{Seed: 1, Instant: true})
	if n := g.OpeningRows(); n > 23 {
		t.Errorf("the opening takes %d rows; it must fit 23", n)
	}
}

// Fallout hangs over the whole front from the tactical nuclear rung, drawn in full.
func TestFalloutOnTheMap(t *testing.T) {
	t.Parallel()
	g := New().(*sim.Game)
	g.Start(proto.Env{Seed: 1, Instant: true})
	if m := strings.Join(g.MapLines(), "\n"); strings.Contains(m, artFallout[0].Text) {
		t.Fatalf("fallout before the tactical nuclear rung:\n%s", m)
	}
	g.State().Vars["level"] = tactical
	m := g.MapLines()
	if strings.Count(m[1], artFallout[0].Text) != len(g.State().Regions) {
		t.Errorf("fallout over every region, uncut:\n%s", strings.Join(m, "\n"))
	}
}

// After the strategic exchange every city on the map is in ruins, as WOPR says; until
// then the cities stand.
func TestRuinsOnTheMap(t *testing.T) {
	t.Parallel()
	g := New().(*sim.Game)
	g.Start(proto.Env{Seed: 1, Instant: true})
	cities := 0
	for _, r := range g.State().Regions {
		if r.Terrain == sim.City {
			cities++
		}
	}
	for level, want := range map[int]int{tactical: 0, strategic: cities} {
		g.State().Vars["level"] = level
		m := g.MapLines()
		if got := strings.Count(m[2], artRuins[1].Text); got != want {
			t.Errorf("rung %d: %d cities in ruins, want %d:\n%s", level, got, want, strings.Join(m, "\n"))
		}
	}
}
