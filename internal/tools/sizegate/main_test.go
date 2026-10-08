package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestGate(t *testing.T) {
	t.Parallel()
	ok := []row{{"linux/amd64", "a", 4_000_000, false}, {"windows/amd64", "b", 4_100_000, false}}
	if err := gate(ok, 2, 0, new(strings.Builder), new(strings.Builder)); err != nil {
		t.Fatalf("within budget: %v", err)
	}
	var out strings.Builder
	warn := []row{{"linux/amd64", "a", 12_000_000, false}}
	if err := gate(warn, 0, 0, &out, new(strings.Builder)); err != nil {
		t.Fatalf("warning must not fail: %v", err)
	}
	if !strings.Contains(out.String(), "::warning::") {
		t.Errorf("expected a warning annotation, got %q", out.String())
	}
	if err := gate([]row{{"linux/amd64", "a", 15_000_001, false}}, 0, 0, new(strings.Builder), new(strings.Builder)); err == nil {
		t.Error("over the hard limit must fail")
	}
	if err := gate(nil, 0, 0, new(strings.Builder), new(strings.Builder)); err == nil {
		t.Error("no binaries must fail")
	}
	if err := gate(ok, 6, 0, new(strings.Builder), new(strings.Builder)); err == nil {
		t.Error("wrong count must fail")
	}
}

// run sizes each built binary once (not again as its bare release copy) and each Linux package,
// and nothing else; -expect and -packages count them apart.
func TestRunReadsBinariesAndPackages(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	list := filepath.Join(dir, "artifacts.json")
	const js = `[
	 {"name":"metadata.json","path":"dist/metadata.json","type":"Metadata"},
	 {"name":"wopr","path":"dist/wopr_linux_amd64_v1/wopr","goos":"linux","goarch":"amd64","type":"Binary"},
	 {"name":"wopr_0.1.0_linux_amd64","path":"dist/wopr_linux_amd64_v1/wopr","goos":"linux","goarch":"amd64","type":"Binary","extra":{"Format":"binary","ID":"binaries"}},
	 {"name":"wopr_0.1.0_linux_amd64.tar.gz","path":"dist/x.tar.gz","goos":"linux","goarch":"amd64","type":"Archive"},
	 {"name":"wopr_0.1.0-1_amd64.deb","path":"dist/wopr_0.1.0-1_amd64.deb","goos":"linux","goarch":"amd64","type":"Linux Package","extra":{"Format":"deb"}},
	 {"name":"wopr-0.1.0-1.x86_64.rpm","path":"dist/wopr-0.1.0-1.x86_64.rpm","goos":"linux","goarch":"amd64","type":"Linux Package","extra":{"Format":"rpm"}}
	]`
	if err := os.WriteFile(list, []byte(js), 0o644); err != nil {
		t.Fatal(err)
	}
	var seen []string
	size := func(p string) (int64, error) { seen = append(seen, p); return 1000, nil }
	var summary strings.Builder
	if err := run(list, 1, 2, new(strings.Builder), &summary, size); err != nil {
		t.Fatal(err)
	}
	want := []string{"dist/wopr_linux_amd64_v1/wopr", "dist/wopr_0.1.0-1_amd64.deb", "dist/wopr-0.1.0-1.x86_64.rpm"}
	if !slices.Equal(seen, want) {
		t.Errorf("sized %v, want %v", seen, want)
	}
	if !strings.Contains(summary.String(), "| linux/amd64 deb |") || !strings.Contains(summary.String(), "| linux/amd64 rpm |") {
		t.Errorf("the summary must list the packages:\n%s", summary.String())
	}
	if err := run(list, 1, 4, new(strings.Builder), new(strings.Builder), size); err == nil || !strings.Contains(err.Error(), "Linux packages") {
		t.Errorf("a missing package must fail: %v", err)
	}
	big := func(string) (int64, error) { return 15_000_001, nil }
	if err := run(list, 1, 2, new(strings.Builder), new(strings.Builder), big); err == nil {
		t.Error("a package over the hard limit must fail")
	}
}
