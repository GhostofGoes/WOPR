package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	goroot := filepath.Join(root, "go")
	modA := filepath.Join(root, "a")
	modB := filepath.Join(root, "b")
	for _, d := range []string{goroot, modA, modB} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(p, s string) {
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(goroot, "LICENSE"), "Go licence\n")
	write(filepath.Join(modA, "LICENSE"), "A licence\r\n")
	write(filepath.Join(modB, "COPYING"), "B licence\n")

	got, err := render(goroot, "go1.27.1", map[string]module{
		"example.com/b": {Path: "example.com/b", Version: "v1.0.0", Dir: modB},
		"example.com/a": {Path: "example.com/a", Version: "v0.2.0", Dir: modA},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(got)
	ia, ib := strings.Index(s, "example.com/a v0.2.0"), strings.Index(s, "example.com/b v1.0.0")
	if ia < 0 || ib < 0 || ia > ib {
		t.Errorf("modules missing or not sorted:\n%s", s)
	}
	if strings.Contains(s, "\r") {
		t.Error("CRLF must be normalised")
	}
	if !strings.Contains(s, "go1.27.1") || !strings.Contains(s, "Go licence") {
		t.Error("Go licence section missing")
	}

	_, err = render(goroot, "go1.27.1", map[string]module{"example.com/c": {Path: "example.com/c", Dir: t.TempDir()}})
	if err == nil || !strings.Contains(err.Error(), "example.com/c") {
		t.Errorf("a module without a licence must fail, got %v", err)
	}
}

// The header names the release toolchain from go.mod, whatever Go runs the tool.
func TestToolchainFromGoMod(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(path, []byte("module x\n\ngo 1.27\n\ntoolchain go1.27.1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if v, err := toolchain(path); err != nil || v != "go1.27.1" {
		t.Fatalf("toolchain = %q, %v", v, err)
	}
	if err := os.WriteFile(path, []byte("module x\n\ngo 1.27\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := toolchain(path); err == nil {
		t.Fatal("a go.mod without a toolchain line is an error")
	}
}

func TestFirstDifference(t *testing.T) {
	t.Parallel()
	if d := firstDifference([]byte("a\nb\nc"), []byte("a\nB\nc")); !strings.Contains(d, "line 2") {
		t.Errorf("got %q", d)
	}
	if d := firstDifference([]byte("a"), []byte("a")); d != "" {
		t.Errorf("equal: %q", d)
	}
}
