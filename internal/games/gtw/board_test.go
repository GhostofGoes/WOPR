package gtw

import (
	"strings"
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
			if !assets.Sea(p.X, p.Y) || p.Y >= assets.MapH {
				t.Errorf("side %d's patrol at %d,%d is not at sea on the board: %q", side, p.X, p.Y, assets.Map[min(p.Y, assets.MapH-1)].Text)
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

// Every city the board knows is struck where the assets place it: listed alone, it takes the
// first strike's city-aimed ICBMs and SLBMs, and its impact is a reversed X at its cell
// inside the map box. The film's two targets are pinned on the screen as well: Las Vegas in
// the south-west, a column inland of Los Angeles, and Seattle in the north-west, under the
// Pacific coast's '\'.
func TestStrikesLandOnTheirCities(t *testing.T) {
	t.Parallel()
	screen := map[string][2]int{"LAS VEGAS": {14, 7}, "SEATTLE": {12, 5}}
	for _, city := range assets.Cities {
		side := "1"
		if city.Side == assets.US {
			side = "2"
		}
		g := New().(*Game)
		g.Start(proto.Env{Seed: 1, Instant: true})
		for _, in := range []string{side, city.Name, ""} {
			g.Handle(proto.LineEvent{Text: in})
		}
		hits := 0
		for _, m := range g.missiles {
			if m.ours && m.to.Name == city.Name {
				hits++
				if m.to.X != city.X || m.to.Y != city.Y {
					t.Errorf("%s: a track lands at %d,%d, want %d,%d", city.Name, m.to.X, m.to.Y, city.X, city.Y)
				}
			}
		}
		if hits == 0 {
			t.Errorf("%s: the first strike sent nothing at it", city.Name)
		}
		c := proto.NewCanvas(80, 19)
		g.View(c)
		x, y := mapLeft+city.X, mapTop+city.Y
		if cell := c.At(x, y); cell.R != 'X' || cell.A&proto.AttrReverse == 0 {
			t.Errorf("%s: the board shows %q at %d,%d, want a reversed X:\n%s", city.Name, cell.R, x, y, c.String())
		}
		if want, ok := screen[city.Name]; ok {
			delete(screen, city.Name)
			if x != want[0] || y != want[1] {
				t.Errorf("%s is drawn at %d,%d, want %d,%d", city.Name, x, y, want[0], want[1])
			}
		}
	}
	if len(screen) != 0 {
		t.Errorf("the board does not know the film's targets %v", screen)
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

// Both sides' bombers stay dotted where their routes run together: one track covers the
// other, and they never fill each other's gaps into a solid '+*+*' line.
func TestBombersStayDotted(t *testing.T) {
	t.Parallel()
	for seed := range uint64(8) {
		for _, side := range []string{"1", "2"} {
			for _, script := range [][]string{nil, {"0 0 100", "0 0 0"}, {"all"}} {
				g := New().(*Game)
				g.Start(proto.Env{Seed: seed, Instant: true})
				for _, in := range []string{side, "Vladivostok, Leningrad, Las Vegas, Seattle", ""} {
					g.Handle(proto.LineEvent{Text: in})
				}
				for i := 0; g.phase == orders; i++ {
					in := ""
					if i < len(script) {
						in = script[i]
					}
					g.Handle(proto.LineEvent{Text: in})
				}
				c := proto.NewCanvas(80, 19)
				g.View(c)
				for _, row := range strings.Split(c.String(), "\n") {
					run := 1
					for i := 1; i < len(row); i++ {
						if a, b := row[i-1], row[i]; (a == '+' && b == '*') || (a == '*' && b == '+') {
							run++
						} else {
							run = 1
						}
						if run >= 6 {
							t.Fatalf("seed %d, side %s, %q: the bombers run together:\n%s", seed, side, script, c.String())
						}
					}
				}
			}
		}
	}
}
