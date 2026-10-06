package assets

import (
	"os"
	"path/filepath"
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
	// Places follow the map's projection: the south-west of Las Vegas, Moscow north-east of
	// Kiev, Murmansk on the Arctic coast, Vladivostok at the far east.
	if p := Locate("LAS VEGAS", US); p.X != 10 || p.Y != 9 {
		t.Errorf("Las Vegas: %+v", p)
	}
	moscow, kiev := Locate("MOSCOW", USSR), Locate("KIEV", USSR)
	if moscow.X <= kiev.X || moscow.Y >= kiev.Y {
		t.Errorf("Moscow %+v is not north-east of Kiev %+v", moscow, kiev)
	}
	// The map runs east from the Bering Strait all the way round: Alaska's side is the west
	// edge, Chukotka's tip just west of it wraps to the east edge.
	if x, _ := At(66, -169); x != 0 {
		t.Errorf("the Bering Strait is at column %d, want 0", x)
	}
	if x, _ := At(66, -170); x != MapW-1 {
		t.Errorf("Chukotka's tip is at column %d, want %d", x, MapW-1)
	}
	a, b := Locate("SMALLVILLE", USSR), Locate("SMALLVILLE", USSR)
	if a != b || a.Side != USSR {
		t.Errorf("an unknown target must land in a fixed place on its side: %+v %+v", a, b)
	}
}

// Every city and missile field lies on land: its cell is land fill or coast, never sea.
func TestCitiesAreInsideTheOutline(t *testing.T) {
	t.Parallel()
	for _, c := range append(Cities, Silos[US], Silos[USSR]) {
		if row := Map[c.Y].Text; c.X >= len(row) || row[c.X] == ' ' {
			t.Errorf("%s at %d,%d is in the sea", c.Name, c.X, c.Y)
		}
	}
}
