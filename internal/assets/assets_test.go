package assets

import (
	"os"
	"path/filepath"
	"slices"
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

// mustFind is the place name names, which the board must know.
func mustFind(t *testing.T, name string) Place {
	t.Helper()
	p, ok := Find(name)
	if !ok {
		t.Fatalf("the board does not know %s", name)
	}
	return p
}

// everyPlace is every city, target and missile field.
func everyPlace() []Place {
	return append(append(slices.Clone(Cities), Targets...), Silos[US], Silos[USSR])
}

func TestMapAndCities(t *testing.T) {
	t.Parallel()
	for _, c := range everyPlace() {
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
	if p := mustFind(t, "LAS VEGAS"); p.X != 13 || p.Y != 6 || p.Side != US {
		t.Errorf("Las Vegas: %+v", p)
	}
	moscow, kiev := mustFind(t, "MOSCOW"), mustFind(t, "KIEV")
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
}

// The art's lines enclose the land: the oceans are sea, the continents' interiors are not,
// and every city and missile field lies on land or its coast. Murmansk, a little north of
// the art's Kola coast, is moved south onto that coast.
func TestCitiesAreInsideTheOutline(t *testing.T) {
	t.Parallel()
	for _, c := range everyPlace() {
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
	m := mustFind(t, "MURMANSK")
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
type cell struct {
	name     string
	side     Side
	lat, lon float64
	x, y     int
	moved    bool
}

var cityCells = []cell{
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

// checkCells checks places against their pinned cells, in order: each where its table says,
// and that cell the one holding its true position, or the land beside it when that is sea, or
// one step away when the table says it was moved.
func checkCells(t *testing.T, places []Place, cells []cell) {
	t.Helper()
	if len(places) != len(cells) {
		t.Fatalf("%d places, %d pinned", len(places), len(cells))
	}
	for i, want := range cells {
		c := places[i]
		if c.Name != want.name || c.Side != want.side || c.X != want.x || c.Y != want.y {
			t.Errorf("place %d is %s (side %d) at %d,%d, want %s (side %d) at %d,%d",
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
	}
}

func TestCityCells(t *testing.T) {
	t.Parallel()
	checkCells(t, Cities, cityCells)
	held := map[[2]int]string{}
	for _, c := range Cities {
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

// targetCells pins the other targets the same way. They may share a cell with a city: at 5
// degrees by 7.5 a cell, Philadelphia's is Washington's. Checked against the art like the
// cities; those it puts on a coast glyph are in TestCitiesAgainstTheArt.
var targetCells = []cell{
	{"SAN DIEGO", US, 32.72, -117.16, 12, 6, false},
	{"SAN JOSE", US, 37.34, -121.89, 11, 6, false},
	{"SACRAMENTO", US, 38.58, -121.49, 11, 5, false},
	{"PORTLAND", US, 45.52, -122.68, 11, 4, false},
	{"SPOKANE", US, 47.66, -117.43, 12, 4, false},
	{"BOISE", US, 43.62, -116.20, 12, 5, false},
	{"RENO", US, 39.53, -119.81, 12, 5, false},
	{"SALT LAKE CITY", US, 40.76, -111.89, 13, 5, false},
	{"PHOENIX", US, 33.45, -112.07, 13, 6, false},
	{"TUCSON", US, 32.22, -110.97, 13, 6, false},
	{"ALBUQUERQUE", US, 35.08, -106.65, 14, 6, false},
	{"EL PASO", US, 31.76, -106.49, 14, 6, false},
	{"COLORADO SPRINGS", US, 38.83, -104.82, 15, 5, false},
	{"CHEYENNE", US, 41.14, -104.82, 15, 5, false},
	{"GREAT FALLS", US, 47.51, -111.30, 13, 4, false},
	{"MINOT", US, 48.23, -101.29, 15, 4, false},
	{"GRAND FORKS", US, 47.93, -97.03, 16, 4, false},
	{"DALLAS", US, 32.78, -96.80, 16, 6, false},
	{"FORT WORTH", US, 32.76, -97.33, 16, 6, false},
	{"SAN ANTONIO", US, 29.42, -98.49, 16, 7, false},
	{"AUSTIN", US, 30.27, -97.74, 16, 6, false},
	{"OKLAHOMA CITY", US, 35.47, -97.52, 16, 6, false},
	{"WICHITA", US, 37.69, -97.33, 16, 5, false},
	{"KANSAS CITY", US, 39.10, -94.58, 17, 5, false},
	{"ST LOUIS", US, 38.63, -90.20, 17, 5, false},
	{"MINNEAPOLIS", US, 44.98, -93.27, 17, 5, false},
	{"MILWAUKEE", US, 43.04, -87.91, 18, 5, false},
	{"DETROIT", US, 42.33, -83.05, 19, 5, false},
	{"CLEVELAND", US, 41.50, -81.69, 19, 5, false},
	{"COLUMBUS", US, 39.96, -83.00, 19, 5, false},
	{"CINCINNATI", US, 39.10, -84.51, 19, 5, false},
	{"INDIANAPOLIS", US, 39.77, -86.16, 18, 5, false},
	{"LOUISVILLE", US, 38.25, -85.76, 18, 5, false},
	{"PITTSBURGH", US, 40.44, -80.00, 20, 5, false},
	{"BUFFALO", US, 42.89, -78.88, 20, 5, false},
	{"PHILADELPHIA", US, 39.95, -75.17, 20, 5, false},
	{"BALTIMORE", US, 39.29, -76.61, 20, 5, false},
	{"NORFOLK", US, 36.85, -76.29, 20, 6, false},
	{"CHARLOTTE", US, 35.23, -80.84, 19, 6, false},
	{"CHARLESTON", US, 32.78, -79.93, 20, 6, false},
	{"NASHVILLE", US, 36.16, -86.78, 18, 6, false},
	{"MEMPHIS", US, 35.15, -90.05, 17, 6, false},
	{"LITTLE ROCK", US, 34.75, -92.29, 17, 6, false},
	{"NEW ORLEANS", US, 29.95, -90.07, 17, 7, false},
	{"JACKSONVILLE", US, 30.33, -81.66, 20, 6, true},
	{"TAMPA", US, 27.95, -82.46, 19, 7, false},
	{"ORLANDO", US, 28.54, -81.38, 20, 7, true},
	{"ANCHORAGE", US, 61.22, -149.90, 6, 2, false},
	{"HONOLULU", US, 21.31, -157.86, 4, 8, false},
	{"TALLINN", USSR, 59.44, 24.75, 40, 3, false},
	{"RIGA", USSR, 56.95, 24.11, 40, 3, false},
	{"VILNIUS", USSR, 54.69, 25.28, 41, 3, false},
	{"KALININGRAD", USSR, 54.71, 20.51, 40, 3, false},
	{"KISHINEV", USSR, 47.01, 28.86, 41, 4, false},
	{"SEVASTOPOL", USSR, 44.62, 33.53, 42, 4, true},
	{"DNEPROPETROVSK", USSR, 48.46, 35.05, 43, 4, false},
	{"DONETSK", USSR, 48.02, 37.80, 43, 4, false},
	{"ROSTOV", USSR, 47.24, 39.70, 43, 4, false},
	{"VOLGOGRAD", USSR, 48.71, 44.51, 44, 4, false},
	{"KUYBYSHEV", USSR, 53.20, 50.10, 46, 3, false},
	{"KAZAN", USSR, 55.80, 49.11, 45, 3, false},
	{"ARKHANGELSK", USSR, 64.54, 40.52, 44, 2, false},
	{"SEVEROMORSK", USSR, 69.07, 33.42, 42, 2, false},
	{"TBILISI", USSR, 41.72, 44.83, 44, 5, false},
	{"YEREVAN", USSR, 40.18, 44.50, 44, 5, false},
	{"BAKU", USSR, 40.41, 49.87, 45, 5, false},
	{"ASHKHABAD", USSR, 37.96, 58.33, 47, 5, false},
	{"SAMARKAND", USSR, 39.63, 66.98, 49, 5, false},
	{"DUSHANBE", USSR, 38.56, 68.79, 49, 5, false},
	{"FRUNZE", USSR, 42.87, 74.57, 50, 5, false},
	{"ALMA-ATA", USSR, 43.22, 76.85, 51, 5, false},
	{"BAIKONUR", USSR, 45.96, 63.31, 48, 4, false},
	{"SEMIPALATINSK", USSR, 50.41, 80.23, 52, 4, false},
	{"PERM", USSR, 58.01, 56.25, 47, 3, false},
	{"UFA", USSR, 54.74, 55.97, 47, 3, false},
	{"CHELYABINSK", USSR, 55.16, 61.44, 48, 3, false},
	{"OMSK", USSR, 54.99, 73.32, 50, 3, false},
	{"TOMSK", USSR, 56.50, 84.97, 52, 3, false},
	{"KRASNOYARSK", USSR, 56.02, 92.89, 54, 3, false},
	{"NORILSK", USSR, 69.36, 88.19, 53, 1, false},
	{"YAKUTSK", USSR, 62.04, 129.68, 61, 2, false},
	{"CHITA", USSR, 52.05, 113.47, 58, 4, false},
	{"KHABAROVSK", USSR, 48.48, 135.07, 62, 4, true},
	{"MAGADAN", USSR, 59.56, 150.83, 66, 3, false},
	{"PETROPAVLOVSK", USSR, 53.05, 158.65, 67, 3, false},
}

func TestTargetCells(t *testing.T) {
	t.Parallel()
	checkCells(t, Targets, targetCells)
}

// Find knows every place by its own name and its aliases, whatever the case, spacing and
// punctuation, and nothing else: an unknown name is not struck somewhere in its stead.
func TestFind(t *testing.T) {
	t.Parallel()
	for _, p := range append(slices.Clone(Cities), Targets...) {
		if got, ok := Find(strings.ToLower(p.Name)); !ok || got != p {
			t.Errorf("Find(%q) = %+v, %v; want %+v", strings.ToLower(p.Name), got, ok, p)
		}
	}
	for in, want := range map[string]string{
		"St. Petersburg": "LENINGRAD", "Saint Petersburg": "LENINGRAD", "ST PETERSBURG": "LENINGRAD",
		"Kyiv": "KIEV", "Kharkiv": "KHARKOV", "Odesa": "ODESSA", "Nizhny Novgorod": "GORKY",
		"Yekaterinburg": "SVERDLOVSK", "Stalingrad": "VOLGOGRAD", "Samara": "KUYBYSHEV",
		"Rostov-on-Don": "ROSTOV", "Alma-Ata": "ALMA-ATA", "alma ata": "ALMA-ATA", "Almaty": "ALMA-ATA",
		"Bishkek": "FRUNZE", "Petropavlovsk-Kamchatsky": "PETROPAVLOVSK",
		"Washington DC": "WASHINGTON", "Washington, D.C.": "WASHINGTON", "D.C.": "WASHINGTON", "dc": "WASHINGTON",
		"New York City": "NEW YORK", "NYC": "NEW YORK", "  new   york ": "NEW YORK", "Vegas": "LAS VEGAS",
		"St. Louis": "ST LOUIS", "Saint Louis": "ST LOUIS", "NORAD": "COLORADO SPRINGS",
		"Cheyenne Mountain": "COLORADO SPRINGS", "Pearl Harbor": "HONOLULU",
		"Smallville": "", "London": "", "Peking": "", "": "", "List": "", "St": "",
	} {
		got, ok := Find(in)
		if want == "" && ok || want != "" && (!ok || got.Name != want) {
			t.Errorf("Find(%q) = %q, %v; want %q", in, got.Name, ok, want)
		}
	}
	// Every name is its own, on one side; every alias is written as key writes it, names a
	// place, and does not hide one.
	sides := map[string]Side{}
	for _, p := range append(slices.Clone(Cities), Targets...) {
		if _, dup := sides[key(p.Name)]; dup {
			t.Errorf("two places are called %s", p.Name)
		}
		sides[key(p.Name)] = p.Side
	}
	for alias, name := range aliases {
		if key(alias) != alias || key(name) != name {
			t.Errorf("alias %q -> %q is not written as key writes it", alias, name)
		}
		if _, ok := known[name]; !ok {
			t.Errorf("alias %s names %s, which the board does not know", alias, name)
		}
		if _, ok := known[alias]; ok {
			t.Errorf("alias %s hides a place of that name", alias)
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
		{"SAN JOSE", 0, 0, '.', "on the Pacific coast, between San Francisco and Los Angeles"},
		{"NEW ORLEANS", 0, 0, '.', "on the Gulf coast"},
		{"TAMPA", 0, 0, '.', "on Florida's Gulf coast"},
		{"ORLANDO", -1, 0, '.', "the Gulf coast is to its west"},
		{"ORLANDO", 1, 0, ')', "the Atlantic coast is to its east"},
		{"JACKSONVILLE", 1, 0, '/', "on the Atlantic coast"},
		{"ANCHORAGE", 0, 1, '-', "Alaska's south coast is below it"},
		{"HONOLULU", 0, 0, '`', "the art's one mark in the mid-Pacific, Hawaii"},
		{"TALLINN", 0, 0, '(', "on the Baltic shore"},
		{"RIGA", 0, 0, '(', "on the Baltic shore"},
		{"KALININGRAD", 0, 0, '(', "on the Baltic shore"},
		{"SEVASTOPOL", 0, 1, '<', "on the Black Sea's north shore"},
		{"TBILISI", -1, 0, '>', "between the Black Sea"},
		{"TBILISI", 1, 0, '6', "and the Caspian"},
		{"BAKU", 0, 0, '6', "on the Caspian"},
		{"ARKHANGELSK", 0, 0, '_', "on the White Sea coast"},
		{"ARKHANGELSK", 0, -1, sea, "the sea is north of it"},
		{"CHITA", -1, 0, ')', "east of Lake Baikal"},
		{"KHABAROVSK", 1, 0, '/', "inland of the Pacific coast"},
		{"MAGADAN", 0, 0, ',', "on the Sea of Okhotsk's north coast"},
		{"MAGADAN", -1, 1, sea, "the Sea of Okhotsk is south of it"},
		{"PETROPAVLOVSK", 0, 0, '/', "on Kamchatka"},
		{"PETROPAVLOVSK", 0, 1, '|', "Kamchatka runs south of it"},
	} {
		p := mustFind(t, c.name)
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
