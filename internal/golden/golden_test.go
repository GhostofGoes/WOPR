package golden

import (
	"strings"
	"testing"
)

func TestAssertMatchesCRLF(t *testing.T) {
	// testdata/sample.golden is stored with LF; CRLF input must still match.
	if Updating() {
		t.Skip("not meaningful while updating")
	}
	Assert(t, "sample", []byte("SHALL WE PLAY A GAME?\r\nLOVE TO.\r\n"))
}

func TestDiff(t *testing.T) {
	t.Parallel()
	d := diff("A\nB\nC", "A\nX\nC")
	if !strings.Contains(d, "line 2") || !strings.Contains(d, `"X"`) {
		t.Errorf("unexpected diff: %s", d)
	}
}
