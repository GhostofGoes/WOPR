package assets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/script"
)

func TestEveryLineHasProvenance(t *testing.T) {
	t.Parallel()
	notice, err := os.ReadFile(filepath.Join("..", "..", "NOTICE.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, err := range script.Validate(Lines, string(notice)) {
		t.Error(err)
	}
	// Matthew Thomas's map is freely usable if his line is included: NOTICE.md carries it.
	if !strings.Contains(string(notice), MapCredit[0].Text) {
		t.Errorf("NOTICE.md must include the map's credit line %q", MapCredit[0].Text)
	}
	if len(Scenarios) != 157 {
		t.Errorf("abs0's montage has 157 names, have %d", len(Scenarios))
	}
}

// checkArt checks a block of art: at most rows rows and width columns, printable ASCII.
func checkArt(t *testing.T, name string, art script.Ls, width, rows int) {
	t.Helper()
	if len(art) > rows {
		t.Errorf("%s has %d rows, want at most %d", name, len(art), rows)
	}
	for i, row := range art {
		if len(row.Text) > width {
			t.Errorf("%s row %d is %d wide, want at most %d", name, i, len(row.Text), width)
		}
		for _, r := range row.Text {
			if r < 0x20 || r > 0x7e {
				t.Errorf("%s row %d: non-ASCII %q", name, i, r)
			}
		}
	}
}

func TestArtFits(t *testing.T) {
	t.Parallel()
	if len(Map) != MapH {
		t.Fatalf("map has %d rows, want %d", len(Map), MapH)
	}
	checkArt(t, "map", Map, MapW, MapH)
	checkArt(t, "world map", World, MapW, 22)
	// The side-choice outlines sit side by side in 80 columns, with the side prompt below
	// them on a 24-row screen.
	if len(OutlineUS) != len(OutlineUSSR) {
		t.Errorf("the outlines have %d and %d rows", len(OutlineUS), len(OutlineUSSR))
	}
	checkArt(t, "US outline", OutlineUS, 36, 12)
	checkArt(t, "USSR outline", OutlineUSSR, 42, 12)
}

func TestMapAndCities(t *testing.T) {
	t.Parallel()
	for _, c := range append(Cities, Silos[US], Silos[USSR]) {
		if c.X < 0 || c.X >= MapW || c.Y < 0 || c.Y >= MapH {
			t.Errorf("%s at %d,%d is off the map", c.Name, c.X, c.Y)
		}
		// North America lies west of the mid-Atlantic, the Soviet Union east of it.
		if (c.Side == US) != (c.X < 30) {
			t.Errorf("%s at x=%d is on the wrong continent", c.Name, c.X)
		}
	}
	// Places follow the art's projection: Las Vegas in the south-west, Moscow north-east of
	// Kiev, Vladivostok at the far east, and the map's corners at 180W, 82.5N.
	if p := Locate("LAS VEGAS", US); p.X != 13 || p.Y != 6 {
		t.Errorf("Las Vegas: %+v", p)
	}
	moscow, kiev := Locate("MOSCOW", USSR), Locate("KIEV", USSR)
	if moscow.X <= kiev.X || moscow.Y >= kiev.Y {
		t.Errorf("Moscow %+v is not north-east of Kiev %+v", moscow, kiev)
	}
	for _, c := range []struct {
		lat, lon float64
		x, y     int
	}{{82, -180, 0, 0}, {0, 0, 36, 11}, {-15, 179.9, MapW - 1, MapH - 1}, {90, -200, 0, 0}} {
		if x, y := At(c.lat, c.lon); x != c.x || y != c.y {
			t.Errorf("At(%v, %v) = %d,%d, want %d,%d", c.lat, c.lon, x, y, c.x, c.y)
		}
	}
	a, b := Locate("SMALLVILLE", USSR), Locate("SMALLVILLE", USSR)
	if a != b || a.Side != USSR {
		t.Errorf("an unknown target must land in a fixed place on its side: %+v %+v", a, b)
	}
}

// The art's lines enclose the land: the oceans are sea, the continents' interiors are not,
// and every city and missile field lies on land or its coast. Murmansk, a little north of
// the art's Kola coast, is moved south onto that coast.
func TestCitiesAreInsideTheOutline(t *testing.T) {
	t.Parallel()
	for _, c := range append(Cities, Silos[US], Silos[USSR]) {
		if Sea(c.X, c.Y) {
			t.Errorf("%s at %d,%d is in the sea", c.Name, c.X, c.Y)
		}
	}
	for _, p := range []struct {
		name     string
		lat, lon float64
		sea      bool
	}{
		{"the mid-Atlantic", 35, -40, true},
		{"the North Pacific", 40, -160, true},
		{"the Indian Ocean", -5, 75, true},
		{"Kansas", 38, -98, false},
		{"Siberia", 60, 100, false},
		{"the Sahara", 22, 10, false},
	} {
		if x, y := At(p.lat, p.lon); Sea(x, y) != p.sea {
			t.Errorf("%s (%d,%d): sea %v, want %v", p.name, x, y, Sea(x, y), p.sea)
		}
	}
	m := Locate("MURMANSK", USSR)
	if x, y := At(68.97, 33.08); !Sea(x, y) || Sea(m.X, m.Y) || m.X != x || m.Y != y+1 {
		t.Errorf("Murmansk is at %d,%d; its true cell %d,%d should be sea, a row north of the coast", m.X, m.Y, x, y)
	}
}

// cityCells is every city's real position and its cell on the big board, pinned so that a
// change to the map, its projection or the placing rules cannot move a city unseen. A
// city's cell is the one that holds its position; or, where that is open water, the land
// cell nearest it (Murmansk); or, where the art or a neighbour says so, a cell beside it
// (moved, with the reason in Cities). Checked against the art: every city in its own
// country, coastal cities on or beside their coast, inland ones not across it.
var cityCells = []struct {
	name     string
	side     Side
	lat, lon float64
	x, y     int
	moved    bool
}{
	{"SEATTLE", US, 47.61, -122.33, 11, 4, false},
	{"SAN FRANCISCO", US, 37.77, -122.42, 11, 5, false},
	{"LOS ANGELES", US, 34.05, -118.24, 12, 6, false},
	{"LAS VEGAS", US, 36.17, -115.14, 13, 6, true},
	{"DENVER", US, 39.74, -104.99, 15, 5, false},
	{"OMAHA", US, 41.26, -95.94, 16, 5, false},
	{"CHICAGO", US, 41.88, -87.63, 18, 5, false},
	{"HOUSTON", US, 29.76, -95.37, 16, 7, false},
	{"ATLANTA", US, 33.75, -84.39, 19, 6, false},
	{"MIAMI", US, 25.76, -80.19, 20, 7, true},
	{"WASHINGTON", US, 38.91, -77.04, 20, 5, false},
	{"NEW YORK", US, 40.71, -74.01, 21, 5, false},
	{"BOSTON", US, 42.36, -71.06, 22, 5, true},
	{"LENINGRAD", USSR, 59.93, 30.34, 42, 3, false},
	{"MURMANSK", USSR, 68.97, 33.08, 42, 2, false},
	{"MINSK", USSR, 53.90, 27.57, 41, 3, false},
	{"MOSCOW", USSR, 55.76, 37.62, 43, 3, false},
	{"KIEV", USSR, 50.45, 30.52, 41, 4, true},
	{"KHARKOV", USSR, 49.99, 36.23, 43, 4, false},
	{"ODESSA", USSR, 46.48, 30.73, 42, 4, false},
	{"GORKY", USSR, 56.33, 44.00, 44, 3, false},
	{"SVERDLOVSK", USSR, 56.84, 60.61, 48, 3, false},
	{"TASHKENT", USSR, 41.30, 69.24, 49, 5, false},
	{"NOVOSIBIRSK", USSR, 55.01, 82.93, 52, 3, false},
	{"IRKUTSK", USSR, 52.29, 104.30, 56, 4, false},
	{"VLADIVOSTOK", USSR, 43.12, 131.89, 62, 5, false},
}

func TestCityCells(t *testing.T) {
	t.Parallel()
	if len(Cities) != len(cityCells) {
		t.Fatalf("%d cities, %d pinned", len(Cities), len(cityCells))
	}
	held := map[[2]int]string{}
	for i, want := range cityCells {
		c := Cities[i]
		if c.Name != want.name || c.Side != want.side || c.X != want.x || c.Y != want.y {
			t.Errorf("city %d is %s (side %d) at %d,%d, want %s (side %d) at %d,%d",
				i, c.Name, c.Side, c.X, c.Y, want.name, want.side, want.x, want.y)
		}
		tx, ty := At(want.lat, want.lon)
		steps := max(c.X-tx, tx-c.X) + max(c.Y-ty, ty-c.Y)
		switch {
		case want.moved && steps != 1:
			t.Errorf("%s is moved %d steps from its true cell %d,%d, want 1", c.Name, steps, tx, ty)
		case !want.moved && !Sea(tx, ty) && steps != 0:
			t.Errorf("%s is %d steps from its true cell %d,%d, which is land", c.Name, steps, tx, ty)
		case !want.moved && Sea(tx, ty) && steps != 1:
			t.Errorf("%s is %d steps from its true cell %d,%d, which is sea; want the coast beside it", c.Name, steps, tx, ty)
		}
		if other, ok := held[[2]int{c.X, c.Y}]; ok {
			t.Errorf("%s and %s share cell %d,%d", other, c.Name, c.X, c.Y)
		}
		held[[2]int{c.X, c.Y}] = c.Name
	}
	// The missile fields: the northern Great Plains, and Uzhur in southern Siberia. Their
	// cells, and the cells two columns east that a second track hits, hold no city.
	for side, want := range map[Side][2]int{US: {15, 4}, USSR: {53, 3}} {
		s := Silos[side]
		if s.X != want[0] || s.Y != want[1] {
			t.Errorf("side %d's silo field is at %d,%d, want %d,%d", side, s.X, s.Y, want[0], want[1])
		}
		for _, x := range []int{s.X, s.X + 2} {
			if city, ok := held[[2]int{x, s.Y}]; ok {
				t.Errorf("side %d's silo strikes land on %s at %d,%d", side, city, x, s.Y)
			}
		}
	}
}

// The art around the cities placed by its coastlines: if the map is redrawn, these say which
// placements to check again. Each is a glyph at an offset from the city, or sea there.
func TestCitiesAgainstTheArt(t *testing.T) {
	t.Parallel()
	const sea = 0
	for _, c := range []struct {
		name   string
		dx, dy int
		glyph  byte
		why    string
	}{
		{"SEATTLE", -2, 0, '\\', "the Pacific coast of the north-west"},
		{"SAN FRANCISCO", -1, 0, '|', "on the Pacific coast"},
		{"LOS ANGELES", -1, 0, '.', "on the Pacific coast"},
		{"LAS VEGAS", -1, 0, ' ', "inland, beside Los Angeles"},
		{"HOUSTON", 1, 0, '.', "at the Gulf coast's west end"},
		{"MIAMI", -1, 0, '.', "the Gulf coast is to its west"},
		{"MIAMI", 1, 0, ')', "the Atlantic coast is to its east"},
		{"BOSTON", 0, 0, ',', "on the New England coast"},
		{"BOSTON", 1, 0, '\'', "on the New England coast"},
		{"MURMANSK", 0, 0, '"', "on the Kola coast"},
		{"MURMANSK", 0, -1, sea, "the Barents Sea is north of it"},
		{"MURMANSK", -1, -1, '_', "Norway's North Cape is north-west of it"},
		{"MINSK", -1, 0, '(', "east of the Baltic's shore"},
		{"ODESSA", 0, 1, '<', "on the Black Sea's north shore"},
		{"KHARKOV", 0, 1, '>', "north of the Black Sea's east end"},
		{"TASHKENT", -4, 0, '6', "east of the Caspian"},
		{"IRKUTSK", 1, 0, ')', "beside Lake Baikal"},
		{"VLADIVOSTOK", 0, 0, '\'', "on the Pacific coast"},
		{"VLADIVOSTOK", 1, 0, sea, "the Sea of Japan is east of it"},
		{"VLADIVOSTOK", 2, 0, '/', "Japan is across that sea"},
	} {
		var p Place
		for _, city := range Cities {
			if city.Name == c.name {
				p = city
			}
		}
		if p.Name == "" {
			t.Fatalf("no city %s", c.name)
		}
		x, y := p.X+c.dx, p.Y+c.dy
		row := Map[y].Text
		got := byte(' ')
		if x < len(row) {
			got = row[x]
		}
		if c.glyph == sea && !Sea(x, y) || c.glyph != sea && got != c.glyph {
			t.Errorf("%s (%d,%d): %d,%d is %q, want %q: %s", c.name, p.X, p.Y, x, y, got, c.glyph, c.why)
		}
	}
}
