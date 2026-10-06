package gtw

import (
	"testing"
	"time"

	"github.com/GhostofGoes/WOPR/internal/assets"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// Every track lands before its stage ends, whatever is ordered, and the board's rows fit
// the 19 the Full layout leaves with the front panel.
func TestTracksLandInTime(t *testing.T) {
	t.Parallel()
	if footRow > 18 {
		t.Fatalf("the board needs %d rows; the Full layout has 19", footRow+1)
	}
	r := proto.NewRand(7, 8)
	for range 300 {
		g := New().(*Game)
		g.Start(proto.Env{Seed: r.Uint64(), Width: 80, Height: 19})
		g.Handle(proto.LineEvent{Text: []string{"1", "2"}[r.IntN(2)]})
		g.Handle(proto.LineEvent{Text: "Moscow, Las Vegas, Kiev"})
		g.Handle(proto.LineEvent{Text: ""})
		stages := map[int]bool{}
		for g.phase != assessed {
			if g.phase == orders {
				g.Handle(proto.LineEvent{Text: []string{"", "all", "hold", "100 0 100", "0 100 0"}[r.IntN(5)]})
			}
			stages[g.stage] = true
			end := strikeEnd
			if g.stage > strikes {
				end = finalEnd
			}
			for _, m := range g.missiles {
				if m.launch+flightFrames[m.kind] > end {
					t.Fatalf("stage %d: %+v lands after frame %d", g.stage, m, end)
				}
			}
			g.Handle(proto.TickEvent{Dt: 100 * time.Millisecond})
		}
		if len(stages) != strikes+1 {
			t.Fatalf("saw stages %v", stages)
		}
	}
}

// Submarines fire from the sea: every patrol area is a blank cell of the map, and each
// target is reached from its side's nearer area.
func TestPatrolsAreAtSea(t *testing.T) {
	t.Parallel()
	for side, areas := range patrols {
		if len(areas) == 0 {
			t.Fatalf("side %d has no patrol area", side)
		}
		for _, p := range areas {
			row := assets.Map[p.Y].Text
			if p.X < 0 || p.X >= assets.MapW || (p.X < len(row) && row[p.X] != ' ') {
				t.Errorf("side %d's patrol at %d,%d is not at sea: %q", side, p.X, p.Y, row)
			}
		}
	}
	for _, tc := range []struct {
		side   assets.Side
		target string
		want   int // index into patrols[side]
	}{
		{assets.USSR, "SEATTLE", 1},
		{assets.USSR, "NEW YORK", 0},
		{assets.US, "MOSCOW", 0},
		{assets.US, "VLADIVOSTOK", 1},
	} {
		enemy := enemyOf(tc.side)
		if got := seaLaunch(tc.side, assets.Locate(tc.target, enemy)); got != patrols[tc.side][tc.want] {
			t.Errorf("side %d at %s fires from %+v, want %+v", tc.side, tc.target, got, patrols[tc.side][tc.want])
		}
	}
}

// The tables under the map do not overlap: the trajectory columns end before the forces
// table starts, and the forces table ends at the board's right edge.
func TestTablesDoNotOverlap(t *testing.T) {
	t.Parallel()
	if end := (trajCols-1)*trajW + len(lineTrajectory[0].Text); end >= forcesX-1 {
		t.Errorf("the trajectory table reaches column %d; the forces table starts at %d", end, forcesX)
	}
	if forcesX+forcesW != 80 {
		t.Errorf("the forces table ends at column %d", forcesX+forcesW-1)
	}
}
