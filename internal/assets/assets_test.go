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
	if p := Locate("LAS VEGAS", US); p.X != 12 || p.Y != 6 {
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
// and every city and missile field lies on land or its coast. Murmansk, a few degrees east of
// the art's Kola coast, is moved onto that coast.
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
	if x, y := At(68.97, 33.08); !Sea(x, y) || Sea(m.X, m.Y) || abs(m.X-x)+abs(m.Y-y) != 1 {
		t.Errorf("Murmansk is at %d,%d; its true cell %d,%d should be sea, one step off the coast", m.X, m.Y, x, y)
	}
}
