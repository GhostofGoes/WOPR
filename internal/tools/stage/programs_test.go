// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
)

// buildTiny links an empty program for linux/goarch with the real toolchain and these go build
// flags, so that the checks meet the linker's real output.
func buildTiny(t *testing.T, goarch string, flags ...string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module tiny\n\ngo 1.27\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "tiny")
	cmd := exec.Command("go", append(append([]string{"build", "-trimpath"}, flags...), "-o", out, ".")...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH="+goarch, "GOFLAGS=")
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build %v: %v\n%s", flags, err, b)
	}
	return out
}

// The checks pass the flags .goreleaser.yaml uses, and fail each flag that is missing or wrong.
func TestCheckLinuxProgram(t *testing.T) {
	if testing.Short() {
		t.Skip("links programs")
	}
	for goarch, want := range linuxPrograms {
		pie := []string{"-buildmode=pie", "-ldflags=-s -w -bindnow -I " + want.interp}
		static := []string{"-ldflags=-s -w"}
		otherArch := map[string]string{"amd64": "arm64", "arm64": "amd64"}[goarch]
		for _, tc := range []struct {
			name    string
			flags   []string
			pie     bool
			arch    string // checked as; goarch when empty
			wantErr string
		}{
			{"PIE", pie, true, "", ""},
			{"static", static, false, "", ""},
			{"PIE wanted, static", static, true, "", "ET_EXEC"},
			{"static wanted, PIE", pie, false, "", "a static program"},
			{"no BIND_NOW", []string{"-buildmode=pie", "-ldflags=-s -w -I " + want.interp}, true, "", "DF_BIND_NOW"},
			{"other loader", []string{"-buildmode=pie", "-ldflags=-s -w -bindnow -I /lib/ld-musl.so.1"}, true, "", "loader"},
			{"other machine", pie, true, otherArch, "an ELF program for"},
		} {
			t.Run(goarch+"/"+tc.name, func(t *testing.T) {
				arch := tc.arch
				if arch == "" {
					arch = goarch
				}
				err := checkLinuxProgram(buildTiny(t, goarch, tc.flags...), arch, tc.pie)
				switch {
				case tc.wantErr == "" && err != nil:
					t.Fatalf("got %v, want no error", err)
				case tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)):
					t.Fatalf("got %v, want an error with %q", err, tc.wantErr)
				}
			})
		}
	}
	if err := checkLinuxProgram("no-such-file", "amd64", true); err == nil {
		t.Error("a missing program accepted")
	}
	if err := checkLinuxProgram("no-such-file", "riscv64", true); err == nil || !strings.Contains(err.Error(), "no rule") {
		t.Errorf("an architecture with no rule accepted: %v", err)
	}
}

func TestCheckBuildSettings(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings []debug.BuildSetting
		mode     string
		ok       bool
	}{
		{"pie, no cgo", []debug.BuildSetting{{Key: "-buildmode", Value: "pie"}, {Key: "CGO_ENABLED", Value: "0"}}, "pie", true},
		{"exe, no cgo", []debug.BuildSetting{{Key: "-buildmode", Value: "exe"}, {Key: "CGO_ENABLED", Value: "0"}}, "exe", true},
		{"cgo", []debug.BuildSetting{{Key: "-buildmode", Value: "pie"}, {Key: "CGO_ENABLED", Value: "1"}}, "pie", false},
		{"exe for pie", []debug.BuildSetting{{Key: "-buildmode", Value: "exe"}, {Key: "CGO_ENABLED", Value: "0"}}, "pie", false},
		{"none", nil, "exe", false},
	} {
		if err := checkBuildSettings(tc.settings, tc.mode); (err == nil) != tc.ok {
			t.Errorf("%s: got %v, want ok=%v", tc.name, err, tc.ok)
		}
	}
}
