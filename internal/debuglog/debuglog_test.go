package debuglog

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestLog(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	l, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	l.Printf("seed %d", 42)
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(l.Path())
	if err != nil || !strings.Contains(string(data), "seed 42") {
		t.Fatalf("log contents %q, %v", data, err)
	}
	if runtime.GOOS != "windows" {
		if info, err := os.Stat(l.Path()); err != nil || info.Mode().Perm() != 0o600 {
			t.Errorf("file mode %v, want 0600", info.Mode().Perm())
		}
	}
	var none *Log
	none.Printf("discarded")
	if none.Path() != "" || none.Close() != nil {
		t.Error("a nil log discards")
	}
}
