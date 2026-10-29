// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package gtw

import (
	"fmt"
	"slices"
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
		to, ok := assets.Find(tc.target)
		if !ok || to.Side != enemyOf(tc.side) {
			t.Fatalf("%s is not an enemy target of side %d: %+v", tc.target, tc.side, to)
		}
		if got := seaLaunch(tc.side, to); got != patrols[tc.side][tc.want] {
			t.Errorf("side %d at %s fires from %+v, want %+v", tc.side, tc.target, got, patrols[tc.side][tc.want])
		}
	}
}

// Every place the board knows is struck where the assets place it, by its own name or
// another: listed alone, it takes the first strike's city-aimed ICBMs and SLBMs, and its
// impact is a reversed X at its cell inside the map box. The film's two targets are pinned on
// the screen as well: Las Vegas in the south-west, a column inland of Los Angeles, and
// Seattle in the north-west, under the Pacific coast's '\'.
func TestStrikesLandOnTheirCities(t *testing.T) {
	t.Parallel()
	typed := map[string]string{} // what is typed, and the place it names
	for _, p := range append(slices.Clone(assets.Cities), assets.Targets...) {
		typed[p.Name] = p.Name
	}
	for in, name := range map[string]string{
		"St. Petersburg": "LENINGRAD", "Kyiv": "KIEV", "Kharkiv": "KHARKOV", "Odesa": "ODESSA",
		"Yekaterinburg": "SVERDLOVSK", "Nizhny Novgorod": "GORKY", "Washington DC": "WASHINGTON",
		"Washington, D.C.": "WASHINGTON", "NYC": "NEW YORK", "saint louis": "ST LOUIS",
		"Alma-Ata": "ALMA-ATA", "Pearl Harbor": "HONOLULU",
	} {
		typed[in] = name
	}
	screen := map[string][2]int{"LAS VEGAS": {14, 7}, "SEATTLE": {12, 5}}
	for in, name := range typed {
		place, ok := assets.Find(name)
		if !ok {
			t.Fatalf("the board does not know %s", name)
		}
		side := "1"
		if place.Side == assets.US {
			side = "2"
		}
		g := New().(*Game)
		g.Start(proto.Env{Seed: 1, Instant: true})
		for _, line := range []string{side, in, ""} {
			g.Handle(proto.LineEvent{Text: line})
		}
		if len(g.targets) != 1 || g.targets[0] != place {
			t.Errorf("%q lists %+v, want only %+v", in, g.targets, place)
		}
		hits := 0
		for _, m := range g.missiles {
			if m.ours && m.to.Name == place.Name {
				hits++
				if m.to.X != place.X || m.to.Y != place.Y {
					t.Errorf("%q: a track lands at %d,%d, want %d,%d", in, m.to.X, m.to.Y, place.X, place.Y)
				}
			}
		}
		if hits == 0 {
			t.Errorf("%q: the first strike sent nothing at it", in)
		}
		c := proto.NewCanvas(80, 19)
		g.View(c)
		x, y := mapLeft+place.X, mapTop+place.Y
		if cell := c.At(x, y); cell.R != 'X' || cell.A&proto.AttrReverse == 0 {
			t.Errorf("%q: the board shows %q at %d,%d, want a reversed X:\n%s", in, cell.R, x, y, c.String())
		}
		if want, ok := screen[in]; ok {
			delete(screen, in)
			if x != want[0] || y != want[1] {
				t.Errorf("%s is drawn at %d,%d, want %d,%d", in, x, y, want[0], want[1])
			}
		}
	}
	if len(screen) != 0 {
		t.Errorf("the board does not know the film's targets %v", screen)
	}
}

// A target must be a place the board knows on the enemy's side, by any of its names; anything
// else is refused and strikes nothing. A target listed twice is listed once.
func TestUnknownTargetsAreRefused(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		side, in string
		want     []string // the targets listed
	}{
		{"1", "Smallville, London", nil},
		{"1", "Las Vegas, Seattle", nil},
		{"2", "Moscow, Washington, D.C.", []string{"WASHINGTON"}},
		{"2", "San Diego, Detroit, Portland, Dallas", []string{"SAN DIEGO", "DETROIT", "PORTLAND", "DALLAS"}},
		{"1", "St Petersburg, Kyiv, Kyiv, Kiev", []string{"LENINGRAD", "KIEV"}},
		{"1", "Moscow, Springfield, Minsk", []string{"MOSCOW", "MINSK"}},
	} {
		g := New().(*Game)
		g.Start(proto.Env{Seed: 1, Instant: true})
		g.Handle(proto.LineEvent{Text: tc.side})
		g.Handle(proto.LineEvent{Text: tc.in})
		var got []string
		for _, p := range g.targets {
			got = append(got, p.Name)
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("side %s, %q lists %q, want %q", tc.side, tc.in, got, tc.want)
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

// The forces table and the cost line fit whatever is ordered: every figure keeps a blank
// column before it, a loss mark fits between its figure and the next, and the cost line fits
// 80 columns with either side first. Strike orders run as in TestNoAllocationWins, every
// percent of ICBMs with SLBMs and bombers at the ends and the middle.
func TestStrikeCostFits(t *testing.T) {
	t.Parallel()
	g := &Game{side: assets.USSR} // the longer name first
	ends := []int{0, 50, 100}
	for i2 := 0; i2 <= 100; i2++ {
		for i3 := 0; i3 <= 100; i3++ {
			for _, s2 := range ends {
				for _, b2 := range ends {
					_, rs := wage([3]int{i2, s2, b2}, [3]int{i3, 100 - s2, 100})
					for _, r := range rs {
						g.rep = r
						if line := g.cost(); len(line) > 80 {
							t.Fatalf("the cost line is %d wide: %q", len(line), line)
						}
						for s := range 2 {
							lost := [2]int{r.snap[1][s].ICBM - r.snap[2][s].ICBM, r.snap[1][s].Ground - r.snap[2][s].Ground}
							if lost[0]+lost[1] != r.ground[s] {
								t.Fatalf("stage %d, side %d: the marks %v are not the %d lost on the ground", r.stage, s, lost, r.ground[s])
							}
							for _, f := range r.snap {
								if f[s].ICBM > 9999 || max(f[s].SLBM, f[s].Ground, f[s].Air) > 999 || max(lost[0], lost[1]) > 999 {
									t.Fatalf("stage %d, side %d: %+v losing %v overflows the table", r.stage, s, f[s], lost)
								}
							}
						}
					}
				}
			}
		}
	}
}

// After each strike lands, the forces table shows what it cost: each side's row marks the
// ICBMs and bombers destroyed beside their figures, reversed, the marks add up to the cost
// line's LOST ON THE GROUND, and CITIES is the warheads landed on that side so far. While the
// next strike flies, no marks show and CITIES does not yet count its warheads.
func TestForcesShowWhatEachStrikeCost(t *testing.T) {
	t.Parallel()
	mark := func(c *proto.Canvas, x, y int) int {
		n := 0
		for x++; c.At(x, y).R >= '0' && c.At(x, y).R <= '9'; x++ {
			n = 10*n + int(c.At(x, y).R-'0')
		}
		return n
	}
	figure := func(c *proto.Canvas, end, y int) string {
		var b strings.Builder
		for x := end - 6; x <= end; x++ {
			b.WriteRune(c.At(x, y).R)
		}
		return strings.TrimSpace(b.String())
	}
	for _, side := range []string{"1", "2"} {
		for _, script := range [][]string{nil, {"hold", "hold"}, {"all"}, {"0 0 100", "100 0 0"}} {
			label := side + " " + strings.Join(script, "|")
			g := New().(*Game)
			g.Start(proto.Env{Seed: 1, Instant: true})
			g.Handle(proto.LineEvent{Text: side})
			g.Handle(proto.LineEvent{Text: "Moscow, Las Vegas"})
			g.Handle(proto.LineEvent{Text: ""})
			for i := 0; ; i++ {
				c := proto.NewCanvas(80, 19)
				g.View(c)
				var said []string // the cost line names your side first
				for s := range 2 {
					y := trajRow + 1 + s
					lost := 0
					for _, x := range []int{forcesICBMMark, forcesBMBMark} {
						cell := c.At(forcesX+x, y)
						if cell.R == ' ' {
							continue
						}
						if cell.R != '-' || cell.S != proto.StyleAlert || cell.A&proto.AttrReverse == 0 {
							t.Fatalf("%s: the mark at %d,%d is %q %v %v:\n%s", label, forcesX+x, y, cell.R, cell.S, cell.A, c.String())
						}
						lost += mark(c, forcesX+x, y)
					}
					if lost != g.rep.ground[s] {
						t.Errorf("%s, stage %d: side %d's marks add up to %d, want %d:\n%s", label, g.stage, s, lost, g.rep.ground[s], c.String())
					}
					if got, want := figure(c, 79, y), fmt.Sprint(g.war.t[s].Cities); got != want {
						t.Errorf("%s, stage %d: side %d's CITIES shows %s, want %s", label, g.stage, s, got, want)
					}
					said = append(said, fmt.Sprintf("%s: %d LOST", g.nation(s), lost))
				}
				if cost := g.cost(); !strings.HasPrefix(cost, said[0]+" ON THE GROUND, ") || !strings.Contains(cost, ".  "+said[1]+", ") {
					t.Errorf("%s, stage %d: the cost line %q does not match the marks %q", label, g.stage, cost, said)
				}
				if g.phase != orders {
					break
				}
				order := ""
				if i < len(script) {
					order = script[i]
				}
				g.Handle(proto.LineEvent{Text: order})
			}
			if g.phase != assessed {
				t.Fatalf("%s: the war ended at %v", label, g.phase)
			}
		}
	}
	// In flight, before the strike lands: no marks, and only earlier strikes' warheads.
	g := New().(*Game)
	g.Start(proto.Env{Seed: 1, Width: 80, Height: 19})
	for _, in := range []string{"2", "Las Vegas", ""} {
		g.Handle(proto.LineEvent{Text: in})
	}
	g.Handle(proto.TickEvent{Dt: time.Minute}) // strike 2's prompt
	g.Handle(proto.LineEvent{Text: ""})
	g.Handle(proto.TickEvent{Dt: 20 * frame})
	c := proto.NewCanvas(80, 19)
	g.View(c)
	for s := range 2 {
		y := trajRow + 1 + s
		for _, x := range []int{forcesICBMMark, forcesBMBMark} {
			if r := c.At(forcesX+x, y).R; r != ' ' {
				t.Errorf("side %d shows a mark %q in flight:\n%s", s, r, c.String())
			}
		}
		if got, want := figure(c, 79, y), fmt.Sprint(g.war.t[s].Cities-g.rep.cities[s]); got != want {
			t.Errorf("side %d's CITIES shows %s in flight, want %s:\n%s", s, got, want, c.String())
		}
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
