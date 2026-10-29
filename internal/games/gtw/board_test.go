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

// A line of many targets is named back on lines that fit the console, broken after a comma,
// so every name stays whole and the lines read back as the one line.
func TestLongConfirmationsWrap(t *testing.T) {
	t.Parallel()
	var added []assets.Place
	for _, name := range []string{
		"COLORADO SPRINGS", "SALT LAKE CITY", "SAN FRANCISCO", "OKLAHOMA CITY", "INDIANAPOLIS",
		"PHILADELPHIA", "JACKSONVILLE", "MINNEAPOLIS",
	} {
		p, ok := assets.Find(name)
		if !ok {
			t.Fatalf("the board does not know %s", name)
		}
		added = append(added, p)
	}
	lines := confirmed(added, maxTargets)
	if len(lines) < 2 {
		t.Fatalf("%q: eight long names fit one line?", lines)
	}
	for _, l := range lines {
		if len(l) > consoleW || strings.HasSuffix(l, " ") {
			t.Errorf("%d columns, or a trailing space: %q", len(l), l)
		}
	}
	for _, l := range lines[:len(lines)-1] {
		if !strings.HasSuffix(l, ",") {
			t.Errorf("a line breaks inside a name: %q", l)
		}
	}
	want := "COLORADO SPRINGS, SALT LAKE CITY, SAN FRANCISCO, OKLAHOMA CITY, INDIANAPOLIS, PHILADELPHIA, " +
		"JACKSONVILLE, MINNEAPOLIS. TARGETS 1-8 OF 8."
	if got := strings.Join(lines, " "); got != want {
		t.Errorf("reads %q, want %q", got, want)
	}
}

// screenNames is every name drawNames put on the board: the runs of StyleTarget or
// StyleIncoming letters inside the map box, not reversed (impacts are), with where each
// starts. A name's own spaces are in its style; the gap beside its X is the map's.
func screenNames(c *proto.Canvas) map[string][2]int {
	in := func(x, y int, s proto.Style) bool {
		cell := c.At(x, y)
		return x < boxRight && cell.S == s && cell.A&proto.AttrReverse == 0 &&
			(cell.R >= 'A' && cell.R <= 'Z' || cell.R == '-' || cell.R == ' ')
	}
	out := map[string][2]int{}
	for y := mapTop; y < boxBottom; y++ {
		for x := mapLeft; x < boxRight; x++ {
			s := c.At(x, y).S
			if (s != proto.StyleTarget && s != proto.StyleIncoming) || !in(x, y, s) || c.At(x, y).R == ' ' {
				continue
			}
			var b strings.Builder
			end := x
			for ; in(end, y, s); end++ {
				b.WriteRune(c.At(end, y).R)
			}
			out[strings.TrimRight(b.String(), " ")] = [2]int{x, y}
			x = end
		}
	}
	return out
}

// Your targets are findable on the board: each is marked @ until one of your warheads lands
// on it, and then named beside its X, as the city of yours WOPR hit last is. A name never
// covers an impact or a mark, and keeps two cells clear of them and of other names at either
// end, but for its own X, and it stays inside the map box; whatever the side, the targets and
// the orders.
func TestTargetsAreNamedOnTheBoard(t *testing.T) {
	t.Parallel()
	lists := map[assets.Side]string{
		assets.US:   "Moscow, Leningrad, Kiev, Minsk, Odessa, Gorky, Kharkov, Murmansk",
		assets.USSR: "New York, Chicago, Houston, Denver, Miami, Seattle, Boston, Atlanta",
	}
	named := 0
	r := proto.NewRand(11, 3)
	for seed := range uint64(24) {
		side := []assets.Side{assets.US, assets.USSR}[seed%2]
		g := New().(*Game)
		g.Start(proto.Env{Seed: seed, Width: 80, Height: 19})
		for _, in := range []string{fmt.Sprint(side), lists[side], ""} {
			g.Handle(proto.LineEvent{Text: in})
		}
		for range 400 {
			if g.phase == orders {
				g.Handle(proto.LineEvent{Text: []string{"", "hold", "all", "0 100 0"}[r.IntN(4)]})
			}
			if g.phase != flight && g.phase != orders && g.phase != assessed {
				break
			}
			c := proto.NewCanvas(80, 19)
			g.View(c)
			marks := map[[2]int]bool{}
			for _, m := range append(slices.Clone(g.impacts), g.missiles...) {
				x, y := mapLeft+m.to.X, mapTop+m.to.Y
				if g.progress(m) >= 1 {
					if cell := c.At(x, y); cell.R != 'X' || cell.A&proto.AttrReverse == 0 {
						t.Fatalf("seed %d: the impact at %d,%d is %q:\n%s", seed, x, y, cell.R, c.String())
					}
				}
				marks[[2]int{x, y}] = true
			}
			for _, p := range g.targets {
				x, y := mapLeft+p.X, mapTop+p.Y
				marks[[2]int{x, y}] = true
				if cell := c.At(x, y); !g.struck(p) && cell.R != rune(aimGlyph[0]) && cell.R != 'X' {
					t.Fatalf("seed %d: %s is not marked: %q at %d,%d:\n%s", seed, p.Name, cell.R, x, y, c.String())
				}
			}
			shown := screenNames(c)
			lettered := map[[2]int]string{} // each cell a name covers, and the name
			for name, at := range shown {
				for i := range len(name) {
					lettered[[2]int{at[0] + i, at[1]}] = name
				}
			}
			for name, at := range shown {
				p, ok := assets.Find(name)
				if !ok {
					t.Fatalf("seed %d: %q is not a place:\n%s", seed, name, c.String())
				}
				named++
				own := [2]int{mapLeft + p.X, mapTop + p.Y}
				if d := at[1] - own[1]; d < -1 || d > 1 {
					t.Errorf("seed %d: %s is named %d rows from its X:\n%s", seed, name, d, c.String())
				}
				for x := at[0] - 2; x <= at[0]+len(name)+1; x++ {
					cell := [2]int{x, at[1]}
					if marks[cell] && cell != own {
						t.Fatalf("seed %d: %s at %v covers or is beside another X:\n%s", seed, name, at, c.String())
					}
					if other, ok := lettered[cell]; ok && other != name {
						t.Fatalf("seed %d: %s at %v runs into %s:\n%s", seed, name, at, other, c.String())
					}
				}
				if want := g.struck(p) || p.Side == g.side; !want {
					t.Errorf("seed %d: %s is named before it is struck:\n%s", seed, name, c.String())
				}
			}
			g.Handle(proto.TickEvent{Dt: frame})
		}
	}
	if named == 0 {
		t.Error("no target was ever named")
	}
}

// The film's board, in movie mode, is the film's: no marks, no names, the film's legend.
func TestFilmBoardNamesNothing(t *testing.T) {
	t.Parallel()
	f := NewFilm(1, false)
	for _, stage := range []FilmStage{FilmStrike1, FilmStrike2, FilmClimax} {
		f.Run(stage)
		c := proto.NewCanvas(80, 19)
		f.View(c)
		if strings.Contains(c.String(), aimGlyph) || len(screenNames(c)) > 0 || strings.Contains(c.String(), "TARGET") {
			t.Errorf("stage %d marks or names targets:\n%s", stage, c.String())
		}
	}
}
