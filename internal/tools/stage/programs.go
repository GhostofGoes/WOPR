// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package main

import (
	"debug/buildinfo"
	"debug/elf"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
)

// pieBuild is the id of the build in .goreleaser.yaml that makes the Linux packages' program.
// Its programs go into the .deb and the .rpm only, never on their own.
const pieBuild = "wopr-pie"

// linuxPrograms are what each Linux program must be (.goreleaser.yaml, docs/PLAN.md §8): built for
// its machine, and, as a PIE, loaded by the C library's dynamic loader at the path that the build
// names with -I, so that the result does not depend on the build host.
var linuxPrograms = map[string]struct {
	machine elf.Machine
	interp  string
}{
	"amd64": {elf.EM_X86_64, "/lib64/ld-linux-x86-64.so.2"},
	"arm64": {elf.EM_AARCH64, "/lib/ld-linux-aarch64.so.1"},
}

// checkLinuxProgram checks the Linux program at path for goarch. The archives' program (pie
// false) must be static: ET_EXEC with no loader, so that it runs on any distribution. The
// packages' program (pie true) must be a PIE (ET_DYN, DF_1_PIE) with the pinned loader, full RELRO
// (PT_GNU_RELRO and DF_BIND_NOW) and no shared libraries. Neither may have an executable stack,
// and Go must have built both without cgo: a cgo build loses most of the sandbox
// (internal/sandbox).
func checkLinuxProgram(path, goarch string, pie bool) error {
	want, ok := linuxPrograms[goarch]
	if !ok {
		return fmt.Errorf("no rule for linux/%s", goarch)
	}
	f, err := elf.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	if f.Class != elf.ELFCLASS64 || f.Machine != want.machine {
		return fmt.Errorf("an ELF program for %s %s, want %s %s", f.Class, f.Machine, elf.ELFCLASS64, want.machine)
	}
	interp, relro, err := segments(f)
	if err != nil {
		return err
	}
	mode := "exe"
	if pie {
		mode = "pie"
		if err := checkPIE(f, want.interp, interp, relro); err != nil {
			return err
		}
	} else if f.Type != elf.ET_EXEC || interp != "" {
		return fmt.Errorf("ELF type %s, loader %q; want %s and none (a static program)", f.Type, interp, elf.ET_EXEC)
	}
	bi, err := buildinfo.ReadFile(path)
	if err != nil {
		return err
	}
	return checkBuildSettings(bi.Settings, mode)
}

// segments returns the program's loader, if it names one, and whether it has a PT_GNU_RELRO
// segment. It fails if the stack is executable.
func segments(f *elf.File) (interp string, relro bool, err error) {
	for _, p := range f.Progs {
		switch p.Type {
		case elf.PT_INTERP:
			b := make([]byte, p.Filesz)
			if _, err := p.ReadAt(b, 0); err != nil {
				return "", false, fmt.Errorf("PT_INTERP: %w", err)
			}
			interp = strings.TrimRight(string(b), "\x00")
		case elf.PT_GNU_RELRO:
			relro = true
		case elf.PT_GNU_STACK:
			if p.Flags&elf.PF_X != 0 {
				return "", false, errors.New("the stack is executable")
			}
		}
	}
	return interp, relro, nil
}

func checkPIE(f *elf.File, wantInterp, interp string, relro bool) error {
	switch {
	case f.Type != elf.ET_DYN:
		return fmt.Errorf("ELF type %s, want %s (-buildmode=pie)", f.Type, elf.ET_DYN)
	case interp != wantInterp:
		return fmt.Errorf("loader %q, want %q", interp, wantInterp)
	case !relro:
		return errors.New("no PT_GNU_RELRO segment")
	}
	if libs, err := f.ImportedLibraries(); err != nil || len(libs) > 0 {
		return fmt.Errorf("needs shared libraries %v (%v), want none", libs, err)
	}
	if v, _ := f.DynValue(elf.DT_FLAGS_1); len(v) != 1 || v[0]&uint64(elf.DF_1_PIE) == 0 {
		return fmt.Errorf("DT_FLAGS_1 %v has no DF_1_PIE", v)
	}
	if v, _ := f.DynValue(elf.DT_FLAGS); len(v) != 1 || v[0]&uint64(elf.DF_BIND_NOW) == 0 {
		return fmt.Errorf("DT_FLAGS %v has no DF_BIND_NOW (-ldflags=-bindnow)", v)
	}
	return nil
}

// checkBuildSettings checks the go command's record of how it built the program.
func checkBuildSettings(settings []debug.BuildSetting, mode string) error {
	got := map[string]string{}
	for _, s := range settings {
		got[s.Key] = s.Value
	}
	if got["-buildmode"] != mode {
		return fmt.Errorf("built with -buildmode=%s, want %s", got["-buildmode"], mode)
	}
	if got["CGO_ENABLED"] != "0" {
		return fmt.Errorf("built with CGO_ENABLED=%s, want 0", got["CGO_ENABLED"])
	}
	return nil
}
