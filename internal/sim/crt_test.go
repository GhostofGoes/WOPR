package sim

import "testing"

// The table is pinned, and monotone: better odds or a higher roll is never worse for the
// attacker.
func TestCRT(t *testing.T) {
	t.Parallel()
	rank := map[Result]int{AttackerLoses: 0, NoEffect: 1, Exchange: 2, DefenderRetreats: 3, DefenderLoses: 4}
	for r := range 6 {
		for c := range columns {
			if c > 0 && rank[crtTable[r][c]] < rank[crtTable[r][c-1]] {
				t.Errorf("roll %d: column %d is worse than column %d", r+1, c, c-1)
			}
			if r > 0 && rank[crtTable[r][c]] < rank[crtTable[r-1][c]] {
				t.Errorf("column %d: roll %d is worse than roll %d", c, r+1, r)
			}
		}
	}
	for _, tc := range []struct {
		a, d int
		t    Terrain
		col  int
	}{
		{1, 3, Open, col1to2},
		{3, 3, Open, col1to1},
		{5, 3, Open, col1to1},
		{6, 3, Open, col2to1},
		{9, 3, Open, col3to1},
		{12, 3, Open, col4to1},
		{40, 1, Open, col4to1},
		{6, 3, Rough, col1to1},
		{6, 3, City, col1to2},
		{12, 3, City, col2to1},
	} {
		if got := Column(tc.a, tc.d, tc.t); got != tc.col {
			t.Errorf("Column(%d, %d, %d) = %d, want %d", tc.a, tc.d, tc.t, got, tc.col)
		}
	}
	if CRT(12, 3, Open, 6) != DefenderLoses || CRT(1, 3, Open, 1) != AttackerLoses || CRT(3, 3, Open, 0) != AttackerLoses {
		t.Error("CRT lookups")
	}
}
