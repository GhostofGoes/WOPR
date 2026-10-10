package main

import (
	"strings"
	"testing"
	"time"
)

func at(s string) *time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return &t
}

func TestCheck(t *testing.T) {
	t.Parallel()
	now := *at("2026-10-15")
	mods := []module{
		{Path: "example.com/main", Main: true},
		{Path: "example.com/old", Version: "v1.0.0", Time: at("2026-09-01")},
		{Path: "example.com/edge", Version: "v1.0.0", Time: at("2026-10-01")}, // exactly 14 days
		{Path: "example.com/new", Version: "v0.0.0-20261010000000-abcdefabcdef", Time: at("2026-10-10")},
		{Path: "example.com/allowed", Version: "v2.0.0", Time: at("2026-10-14")},
		{
			Path: "example.com/replaced", Version: "v1.0.0", Time: at("2026-01-01"),
			Replace: &module{Path: "example.com/fork", Version: "v1.0.1", Time: at("2026-10-12")},
		},
		{Path: "example.com/local", Version: "v1.0.0", Replace: &module{Path: "../local"}},
	}
	allowed := map[string]bool{"example.com/allowed@v2.0.0": true}
	got, err := check("go.mod", mods, now, 14*24*time.Hour, allowed)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, f := range got {
		paths = append(paths, f.path)
	}
	if want := "example.com/new example.com/fork"; strings.Join(paths, " ") != want {
		t.Errorf("findings %v, want %s", paths, want)
	}

	if _, err := check("go.mod", []module{{Path: "example.com/x", Version: "v1.0.0"}}, now, time.Hour, nil); err == nil {
		t.Error("a module without a time passed")
	}
}

func TestParseExceptions(t *testing.T) {
	t.Parallel()
	got, err := parseExceptions(strings.NewReader("# header\n\nexample.com/a@v1.2.3  # a CVE fix\n"))
	if err != nil || len(got) != 1 || !got["example.com/a@v1.2.3"] {
		t.Errorf("got %v, %v", got, err)
	}
	for _, bad := range []string{"example.com/a@v1\n", "example.com/a # no version\n", "a@v1 b@v2 # two\n"} {
		if _, err := parseExceptions(strings.NewReader(bad)); err == nil {
			t.Errorf("%q was accepted", bad)
		}
	}
}

// The checked-in exceptions file must parse.
func TestExceptionsFileParses(t *testing.T) {
	t.Parallel()
	if _, err := readExceptions("../../../" + exceptionsFile); err != nil {
		t.Fatal(err)
	}
}
