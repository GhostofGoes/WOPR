// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package main

import (
	"debug/buildinfo"
	"debug/elf"
	"errors"
	"fmt"
	"runtime/debug"
)

// linuxMachines is the ELF machine of each Linux program (.goreleaser.yaml).
var linuxMachines = map[string]elf.Machine{
	"amd64": elf.EM_X86_64,
	"arm64": elf.EM_AARCH64,
}

// checkLinuxProgram checks the Linux program at path for goarch. It must be static (ET_EXEC with
// no loader), so that it runs on any distribution, with no executable stack. Go must have built it
// without cgo: a cgo build loses most of the sandbox (internal/sandbox).
func checkLinuxProgram(path, goarch string) error {
	want, ok := linuxMachines[goarch]
	if !ok {
		return fmt.Errorf("no rule for linux/%s", goarch)
	}
	f, err := elf.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	if f.Class != elf.ELFCLASS64 || f.Machine != want {
		return fmt.Errorf("an ELF program for %s %s, want %s %s", f.Class, f.Machine, elf.ELFCLASS64, want)
	}
	if f.Type != elf.ET_EXEC {
		return fmt.Errorf("ELF type %s, want %s (a static program)", f.Type, elf.ET_EXEC)
	}
	for _, p := range f.Progs {
		switch {
		case p.Type == elf.PT_INTERP:
			return errors.New("names a dynamic loader, want a static program")
		case p.Type == elf.PT_GNU_STACK && p.Flags&elf.PF_X != 0:
			return errors.New("the stack is executable")
		}
	}
	bi, err := buildinfo.ReadFile(path)
	if err != nil {
		return err
	}
	return checkBuildSettings(bi.Settings)
}

// checkBuildSettings checks the go command's record of how it built the program.
func checkBuildSettings(settings []debug.BuildSetting) error {
	got := map[string]string{}
	for _, s := range settings {
		got[s.Key] = s.Value
	}
	if got["-buildmode"] != "exe" {
		return fmt.Errorf("built with -buildmode=%s, want exe", got["-buildmode"])
	}
	if got["CGO_ENABLED"] != "0" {
		return fmt.Errorf("built with CGO_ENABLED=%s, want 0", got["CGO_ENABLED"])
	}
	return nil
}
