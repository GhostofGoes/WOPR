package guerrilla

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
	for _, bad := range sim.ArtProblems(Lines, artTitle, artCamps, artJungle, artRiver) {
		t.Error(bad)
	}
	g := New().(*sim.Game)
	g.Start(proto.Env{Seed: 1, Instant: true})
	if n := g.OpeningRows(); n > 23 {
		t.Errorf("the opening takes %d rows; it must fit 23", n)
	}
}

// The map's road shows hidden cells as ~, as the table marks them: the three cells start
// hidden, one at the border camps and two in the jungle.
func TestHiddenCellsOnTheRoad(t *testing.T) {
	t.Parallel()
	g := New().(*sim.Game)
	g.Start(proto.Env{Seed: 1, Instant: true})
	m := g.MapLines()
	if road := m[3]; !strings.Contains(road, "~(1)") || !strings.Contains(road, "~~(2)") || strings.Contains(road, ">") {
		t.Errorf("hidden cells on the road:\n%s", strings.Join(m, "\n"))
	}
}

// The camps have their tents, the jungle its palms and the valley its river, drawn in
// full; none of them uses the ~ that marks a hidden cell.
func TestGroundOnTheMap(t *testing.T) {
	t.Parallel()
	g := New().(*sim.Game)
	g.Start(proto.Env{Seed: 1, Instant: true})
	m := g.MapLines()
	for _, art := range artGround {
		if !strings.Contains(m[1], art[0].Text) || !strings.Contains(m[2], art[1].Text) {
			t.Errorf("%q not drawn in full:\n%s", art.Texts(), strings.Join(m, "\n"))
		}
		if strings.Contains(strings.Join(art.Texts(), ""), "~") {
			t.Errorf("%q uses ~", art.Texts())
		}
	}
}
