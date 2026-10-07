package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGate(t *testing.T) {
	t.Parallel()
	ok := []row{{"linux/amd64", "a", 4_000_000}, {"windows/amd64", "b", 4_100_000}}
	if err := gate(ok, 2, new(strings.Builder), new(strings.Builder)); err != nil {
		t.Fatalf("within budget: %v", err)
	}
	var out strings.Builder
	warn := []row{{"linux/amd64", "a", 12_000_000}}
	if err := gate(warn, 0, &out, new(strings.Builder)); err != nil {
		t.Fatalf("warning must not fail: %v", err)
	}
	if !strings.Contains(out.String(), "::warning::") {
		t.Errorf("expected a warning annotation, got %q", out.String())
	}
	if err := gate([]row{{"linux/amd64", "a", 15_000_001}}, 0, new(strings.Builder), new(strings.Builder)); err == nil {
		t.Error("over the hard limit must fail")
	}
	if err := gate(nil, 0, new(strings.Builder), new(strings.Builder)); err == nil {
		t.Error("no binaries must fail")
	}
	if err := gate(ok, 6, new(strings.Builder), new(strings.Builder)); err == nil {
		t.Error("wrong count must fail")
	}
}

func TestRunReadsOnlyBinaries(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	list := filepath.Join(dir, "artifacts.json")
	const js = `[
	 {"name":"metadata.json","path":"dist/metadata.json","type":"Metadata"},
	 {"name":"wopr","path":"dist/wopr_linux_amd64_v1/wopr","goos":"linux","goarch":"amd64","type":"Binary"},
	 {"name":"wopr_0.1.0_linux_amd64.tar.gz","path":"dist/x.tar.gz","goos":"linux","goarch":"amd64","type":"Archive"}
	]`
	if err := os.WriteFile(list, []byte(js), 0o644); err != nil {
		t.Fatal(err)
	}
	var seen []string
	size := func(p string) (int64, error) { seen = append(seen, p); return 1000, nil }
	if err := run(list, 1, new(strings.Builder), new(strings.Builder), size); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 1 || seen[0] != "dist/wopr_linux_amd64_v1/wopr" {
		t.Errorf("sized %v, want only the binary", seen)
	}
}
