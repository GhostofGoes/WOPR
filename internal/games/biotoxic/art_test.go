package biotoxic

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
	for _, bad := range sim.ArtProblems(Lines, artTitle, artClouds) {
		t.Error(bad)
	}
	if len(artClouds) != maxLevel+1 || artClouds[0].Text != "" {
		t.Errorf("a cloud for each level of contamination, and none for clean ground: %q", artClouds.Texts())
	}
	g := New().(*sim.Game)
	g.Start(proto.Env{Seed: 1, Instant: true})
	if n := g.OpeningRows(); n > 23 {
		t.Errorf("the opening takes %d rows; it must fit 23", n)
	}
}

// Each contaminated region has its cloud over it, drawn in full and thicker with the
// level; clean ground has none, and with nothing contaminated the row is left out.
func TestCloudsOnTheMap(t *testing.T) {
	t.Parallel()
	g := New().(*sim.Game)
	g.Start(proto.Env{Seed: 1, Instant: true})
	clean := g.MapLines()
	s := g.State()
	for r, c := range []int{0, 1, 2, 3, 0, 1} {
		setLevel(s, r, c)
	}
	m := g.MapLines()
	if len(m) != len(clean)+1 {
		t.Fatalf("the cloud row: %d rows contaminated, %d clean", len(m), len(clean))
	}
	for c := 1; c <= maxLevel; c++ {
		want := 1
		if c == 1 {
			want = 2
		}
		if got := strings.Count(m[1], artClouds[c].Text); got != want {
			t.Errorf("level %d clouds: %d, want %d:\n%s", c, got, want, strings.Join(m, "\n"))
		}
	}
}
