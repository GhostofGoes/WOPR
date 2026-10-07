package desertwarfare

import (
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/sim"
)

// The pictures are printable ASCII capitals within 80 columns and in Lines, and the
// opening (art, map, prompt) fits a 23-row console.
func TestArt(t *testing.T) {
	t.Parallel()
	for _, bad := range sim.ArtProblems(Lines, artTitle, artSand) {
		t.Error(bad)
	}
	g := New().(*sim.Game)
	g.Start(proto.Env{Seed: 1, Instant: true})
	if n := g.OpeningRows(); n > 23 {
		t.Errorf("the opening takes %d rows; it must fit 23", n)
	}
}

// Sand blows over the whole road while a sandstorm halves the attacks, drawn in full.
func TestSandstormOnTheMap(t *testing.T) {
	t.Parallel()
	g := New().(*sim.Game)
	g.Start(proto.Env{Seed: 1, Instant: true})
	if m := strings.Join(g.MapLines(), "\n"); strings.Contains(m, artSand[0].Text) {
		t.Fatalf("sand with no sandstorm:\n%s", m)
	}
	g.State().Vars["sandstorm"] = 1
	if m := g.MapLines(); strings.Count(m[1], artSand[0].Text) != len(g.State().Regions) {
		t.Errorf("sand over every region, uncut:\n%s", strings.Join(m, "\n"))
	}
}
