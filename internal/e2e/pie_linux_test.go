// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

//go:build e2e

package e2e

import (
	"bufio"
	"bytes"
	"debug/elf"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The Linux packages' program is a PIE (the wopr-pie build in .goreleaser.yaml): the kernel loads
// its code at a new random address each run, which a fixed-address program (ET_EXEC, at 0x400000
// on amd64) never gets. Two running sessions, inside the sandbox, must have their code at different
// addresses. The archives' program is static, so the test skips it, unless WOPR_E2E_PIE is set (the
// smoke job's run against the installed .deb). internal/tools/stage checks both programs' headers.
func TestLinuxPIE(t *testing.T) {
	f, err := elf.Open(binary)
	if err != nil {
		t.Fatal(err)
	}
	typ := f.Type
	_ = f.Close()
	switch {
	case typ == elf.ET_DYN:
	case os.Getenv("WOPR_E2E_PIE") != "":
		t.Fatalf("%s is %s, want %s (-buildmode=pie)", binary, typ, elf.ET_DYN)
	default:
		t.Skipf("%s is %s: the archives' program is static", binary, typ)
	}
	if aslr, err := os.ReadFile("/proc/sys/kernel/randomize_va_space"); err == nil && strings.TrimSpace(string(aslr)) == "0" {
		t.Skip("the kernel's ASLR is off (randomize_va_space is 0)")
	}
	path, err := filepath.Abs(binary)
	if err == nil {
		path, err = filepath.EvalSymlinks(path)
	}
	if err != nil {
		t.Fatal(err)
	}
	bases := map[uint64]bool{}
	for range 2 {
		s := start(t, 80, 24, "--instant")
		s.waitFor("LOGON:", 10*time.Second) // the sandbox is on by now
		base := codeBase(t, s.cmd.Process.Pid, path)
		s.send("\x03")
		if code := s.wait(10 * time.Second); code != 130 {
			t.Errorf("exit %d after Ctrl+C, want 130", code)
		}
		if bases[base] {
			t.Errorf("two runs loaded the code at %#x: it is not randomized", base)
		}
		bases[base] = true
	}
}

// codeBase is the start of the first mapping of the program at path in /proc/<pid>/maps.
func codeBase(t *testing.T, pid int, path string) uint64 {
	t.Helper()
	maps, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "maps"))
	if err != nil {
		t.Fatal(err)
	}
	sc := bufio.NewScanner(bytes.NewReader(maps))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) == 6 && fields[5] == path {
			lo, _, _ := strings.Cut(fields[0], "-")
			base, err := strconv.ParseUint(lo, 16, 64)
			if err != nil {
				t.Fatal(err)
			}
			return base
		}
	}
	t.Fatalf("no mapping of %s in /proc/%d/maps:\n%s", path, pid, maps)
	return 0
}
