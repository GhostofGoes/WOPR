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

func TestMapAndCities(t *testing.T) {
	t.Parallel()
	if len(Map) != MapH {
		t.Fatalf("map has %d rows, want %d", len(Map), MapH)
	}
	for i, row := range Map {
		if len(row.Text) > MapW {
			t.Errorf("row %d is %d wide", i, len(row.Text))
		}
		for _, r := range row.Text {
			if r < 0x20 || r > 0x7e {
				t.Errorf("row %d: non-ASCII %q", i, r)
			}
		}
	}
	for _, c := range append(Cities, Silos[US], Silos[USSR]) {
		if c.X < 0 || c.X >= MapW || c.Y < 0 || c.Y >= MapH {
			t.Errorf("%s at %d,%d is off the map", c.Name, c.X, c.Y)
		}
		if (c.Side == US) != (c.X < 26) {
			t.Errorf("%s at x=%d is on the wrong continent", c.Name, c.X)
		}
	}
	if p := Locate("LAS VEGAS", US); p.X != 8 || p.Y != 3 {
		t.Errorf("Las Vegas: %+v", p)
	}
	a, b := Locate("SMALLVILLE", USSR), Locate("SMALLVILLE", USSR)
	if a != b || a.Side != USSR {
		t.Errorf("an unknown target must land in a fixed place on its side: %+v %+v", a, b)
	}
}
