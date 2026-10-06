package version

import (
	"runtime/debug"
	"testing"
)

func TestResolve(t *testing.T) {
	t.Parallel()
	vcs := &debug.BuildInfo{
		GoVersion: "go1.27.1",
		Main:      debug.Module{Version: "v0.1.0"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abcdef0123456789"},
			{Key: "vcs.time", Value: "2026-10-06T04:38:16Z"},
			{Key: "vcs.modified", Value: "false"},
		},
	}
	tests := []struct {
		name                string
		bi                  *debug.BuildInfo
		ldV, ldC, ldD, want string
	}{
		{
			"release build", nil, "0.1.0", "abcdef0123456789", "2026-10-06T04:38:16Z",
			"wopr v0.1.0 (commit abcdef0, built 2026-10-06, go1.27.1)",
		},
		{
			"go install", &debug.BuildInfo{GoVersion: "go1.27.1", Main: debug.Module{Version: "v0.1.0"}}, "", "", "",
			"wopr v0.1.0 (commit unknown, built unknown, go1.27.1)",
		},
		{
			"local clone", vcs, "", "", "",
			"wopr v0.1.0 (commit abcdef0, built 2026-10-06, go1.27.1)",
		},
		{
			"devel", &debug.BuildInfo{GoVersion: "go1.27.1", Main: debug.Module{Version: "(devel)"}}, "", "", "",
			"wopr (devel) (commit unknown, built unknown, go1.27.1)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := resolve(tt.bi, tt.ldV, tt.ldC, tt.ldD)
			if tt.bi == nil {
				got.GoVersion = "go1.27.1" // runtime.Version() varies; pin it for the comparison
			}
			if s := got.String(); s != tt.want {
				t.Errorf("String() = %q, want %q", s, tt.want)
			}
		})
	}
}

func TestDirty(t *testing.T) {
	t.Parallel()
	bi := &debug.BuildInfo{GoVersion: "go1.27.1", Settings: []debug.BuildSetting{
		{Key: "vcs.revision", Value: "1234567890"},
		{Key: "vcs.modified", Value: "true"},
	}}
	if got, want := resolve(bi, "", "", "").String(), "wopr (devel) (commit 1234567-dirty, built unknown, go1.27.1)"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
