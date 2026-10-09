package debuglog

import (
	"os"
	"path/filepath"
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

func TestOpenRefusesLinks(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symbolic links need privileges on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("keep\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "wopr"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "wopr", "debug.log")); err != nil {
		t.Fatal(err)
	}
	if l, err := Open(dir); err == nil {
		_ = l.Close()
		t.Fatal("Open followed a link at debug.log")
	}

	linked := t.TempDir()
	if err := os.Symlink(t.TempDir(), filepath.Join(linked, "wopr")); err != nil {
		t.Fatal(err)
	}
	if l, err := Open(linked); err == nil {
		_ = l.Close()
		t.Fatal("Open followed a link at the log's directory")
	}
	if data, err := os.ReadFile(target); err != nil || string(data) != "keep\n" {
		t.Fatalf("target changed to %q, %v", data, err)
	}
}

func TestOpenTightensModes(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no Unix modes")
	}
	dir := t.TempDir()
	d := filepath.Join(dir, "wopr")
	if err := os.Mkdir(d, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(d, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "debug.log"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(d, "debug.log"), 0o666); err != nil {
		t.Fatal(err)
	}
	l, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	_ = l.Close()
	for path, want := range map[string]os.FileMode{d: 0o700, l.Path(): 0o600} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != want {
			t.Errorf("%s: mode %v, want %v", path, got, want)
		}
	}
}
