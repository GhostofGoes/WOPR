// Package golden compares test output with files in the calling package's testdata/
// directory.
//
// Regenerate every golden file in the repository with:
//
//	WOPR_UPDATE_GOLDEN=1 go test ./...
//
// An environment variable is used instead of a -update flag, because `go test ./...`
// passes flags to every package and fails in packages that do not define them.
package golden

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// UpdateEnv is the environment variable that rewrites golden files instead of comparing.
const UpdateEnv = "WOPR_UPDATE_GOLDEN"

// Assert compares got with testdata/<name>.golden. Line endings are normalised to LF on
// both sides, so a checkout with CRLF line endings still compares equal.
func Assert(t testing.TB, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	got = normalize(got)
	if Updating() {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("golden: %v", err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("golden: %v", err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden: %v (run with %s=1 to create it)", err, UpdateEnv)
	}
	want = normalize(want)
	if !bytes.Equal(got, want) {
		t.Errorf("golden %s differs (run with %s=1 to update):\n%s", path, UpdateEnv, diff(string(want), string(got)))
	}
}

// AssertString is Assert for strings.
func AssertString(t testing.TB, name, got string) {
	t.Helper()
	Assert(t, name, []byte(got))
}

// Updating reports whether golden files are being rewritten.
func Updating() bool { return os.Getenv(UpdateEnv) != "" }

func normalize(b []byte) []byte {
	return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
}

// diff is a minimal line diff: it reports the first differing line.
func diff(want, got string) string {
	w := strings.Split(want, "\n")
	g := strings.Split(got, "\n")
	for i := range max(len(w), len(g)) {
		var wl, gl string
		if i < len(w) {
			wl = w[i]
		}
		if i < len(g) {
			gl = g[i]
		}
		if wl != gl {
			return fmt.Sprintf("first difference at line %d:\n  want: %q\n  got:  %q\n(want %d lines, got %d)",
				i+1, wl, gl, len(w), len(g))
		}
	}
	return "(no line differs; check trailing bytes)"
}
