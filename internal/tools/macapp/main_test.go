package main

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// CPU types from <mach/machine.h>.
const (
	cpuAMD64 = 0x01000007
	cpuARM64 = 0x0100000c
)

// slice describes one architecture of a fake Mach-O program.
type slice struct {
	cpu      uint32
	fileType uint32 // 2 is MH_EXECUTE, 1 MH_OBJECT
	minos    uint32 // 0: no LC_BUILD_VERSION
}

// machO returns a 64-bit little-endian Mach-O file with only an LC_BUILD_VERSION load command,
// which is all macapp reads.
func machO(s slice) []byte {
	var cmds []byte
	ncmds := uint32(0)
	if s.minos != 0 {
		cmds = binary.LittleEndian.AppendUint32(cmds, lcBuildVersion)
		cmds = binary.LittleEndian.AppendUint32(cmds, 24)
		cmds = binary.LittleEndian.AppendUint32(cmds, platformMacOS)
		cmds = binary.LittleEndian.AppendUint32(cmds, s.minos)
		cmds = binary.LittleEndian.AppendUint32(cmds, 26<<16|2<<8) // SDK 26.2
		cmds = binary.LittleEndian.AppendUint32(cmds, 0)
		ncmds = 1
	}
	var b []byte
	for _, v := range []uint32{0xfeedfacf, s.cpu, 0, s.fileType, ncmds, uint32(len(cmds)), 0, 0} {
		b = binary.LittleEndian.AppendUint32(b, v)
	}
	return append(b, cmds...)
}

// fat returns a universal file holding each slice, as lipo writes one: a big-endian header,
// then each architecture at a 4 KiB boundary.
func fat(parts ...slice) []byte {
	const align = 12 // 4 KiB, as a power of two
	var b []byte
	b = binary.BigEndian.AppendUint32(b, 0xcafebabe)
	b = binary.BigEndian.AppendUint32(b, uint32(len(parts)))
	for i, p := range parts {
		offset := uint32(i+1) << align // each slice is far smaller than 4 KiB
		for _, v := range []uint32{p.cpu, 0, offset, uint32(len(machO(p))), align} {
			b = binary.BigEndian.AppendUint32(b, v)
		}
	}
	for i, p := range parts {
		b = append(b, make([]byte, (i+1)<<align-len(b))...)
		b = append(b, machO(p)...)
	}
	return b
}

// icns returns a minimal .icns file: the header and one empty-looking icon element.
func icns() []byte {
	elem := append([]byte("ic10"), binary.BigEndian.AppendUint32(nil, 12)...)
	elem = append(elem, "PNG\x00"...)
	b := append([]byte("icns"), binary.BigEndian.AppendUint32(nil, uint32(8+len(elem)))...)
	return append(b, elem...)
}

// fixture writes a repository root with the notices, a program and an icon, and returns options
// for a bundle in a directory of its own.
func fixture(t *testing.T, program []byte) (root string, o options) {
	t.Helper()
	root = t.TempDir()
	for _, n := range notices {
		if err := os.WriteFile(filepath.Join(root, n), []byte("text of "+n+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	o = options{
		binary:  filepath.Join(root, "wopr"),
		version: "1.2.3",
		icon:    filepath.Join(root, "wopr.icns"),
		out:     filepath.Join(t.TempDir(), "dmg", "WOPR.app"),
	}
	if err := os.WriteFile(o.binary, program, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(o.icon, icns(), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, o
}

var universal = fat(slice{cpuAMD64, 2, 13 << 16}, slice{cpuARM64, 2, 13 << 16})

func TestBundleVersion(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"1.2.3":                  "1.2.3",
		"0.10.0":                 "0.10.0",
		"1.2.4-snapshot.abc1234": "1.2.4",
		"2.0.0-rc.1":             "2.0.0",
	} {
		if got, err := bundleVersion(in); err != nil || got != want {
			t.Errorf("bundleVersion(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "v1.2.3", "1.2", "1.2.3.4", "1.2.3+build", "1.2.3-", "one.two.three"} {
		if got, err := bundleVersion(in); err == nil {
			t.Errorf("bundleVersion(%q) = %q, want an error", in, got)
		}
	}
}

func TestFormatVersion(t *testing.T) {
	t.Parallel()
	for v, want := range map[uint32]string{13 << 16: "13.0", 26<<16 | 2<<8: "26.2", 10<<16 | 15<<8 | 7: "10.15.7"} {
		if got := formatVersion(v); got != want {
			t.Errorf("formatVersion(%#x) = %q, want %q", v, got, want)
		}
	}
}

func TestMinimumMacOS(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		data []byte
		want string
	}{
		"thin arm64":         {machO(slice{cpuARM64, 2, 13 << 16}), "13.0"},
		"universal":          {universal, "13.0"},
		"universal, newest":  {fat(slice{cpuAMD64, 2, 13 << 16}, slice{cpuARM64, 2, 14<<16 | 2<<8}), "14.2"},
		"universal, patched": {fat(slice{cpuAMD64, 2, 13<<16 | 1}, slice{cpuARM64, 2, 13 << 16}), "13.0.1"},
	} {
		path := filepath.Join(t.TempDir(), "wopr")
		if err := os.WriteFile(path, tc.data, 0o755); err != nil {
			t.Fatal(err)
		}
		if got, err := minimumMacOS(path); err != nil || got != tc.want {
			t.Errorf("%s: minimumMacOS = %q, %v; want %q", name, got, err, tc.want)
		}
	}
}

// plistValues decodes an XML property list's top-level dict into its keys, in order, and their
// values: a string, or true or false.
func plistValues(t *testing.T, data []byte) (keys []string, values map[string]any) {
	t.Helper()
	values = map[string]any{}
	dec := xml.NewDecoder(bytes.NewReader(data))
	var key string
	var path []string
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("Info.plist does not parse: %v", err)
		}
		switch tok := tok.(type) {
		case xml.StartElement:
			path = append(path, tok.Name.Local)
			if strings.Join(path, "/") == "plist/dict/true" || strings.Join(path, "/") == "plist/dict/false" {
				values[key] = tok.Name.Local == "true"
			}
		case xml.EndElement:
			path = path[:len(path)-1]
		case xml.CharData:
			switch strings.Join(path, "/") {
			case "plist/dict/key":
				key = string(tok)
				if _, dup := values[key]; dup || slices.Contains(keys, key) {
					t.Errorf("key %s appears twice", key)
				}
				keys = append(keys, key)
			case "plist/dict/string":
				values[key] = string(tok)
			}
		}
	}
	if len(keys) != len(values) {
		t.Errorf("%d keys but %d values", len(keys), len(values))
	}
	return keys, values
}

func TestInfoPlist(t *testing.T) {
	t.Parallel()
	data, err := infoPlist("1.2.4", "13.0")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte(`<?xml version="1.0" encoding="UTF-8"?>`+"\n"+`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">`+"\n"+`<plist version="1.0">`)) {
		t.Errorf("Info.plist does not start as an XML property list:\n%s", data)
	}
	keys, values := plistValues(t, data)
	want := map[string]any{
		"CFBundleDevelopmentRegion":     "en",
		"CFBundleDisplayName":           "WOPR",
		"CFBundleExecutable":            "wopr",
		"CFBundleIconFile":              "WOPR",
		"CFBundleIdentifier":            "io.github.ghostofgoes.wopr",
		"CFBundleInfoDictionaryVersion": "6.0",
		"CFBundleName":                  "WOPR",
		"CFBundlePackageType":           "APPL",
		"CFBundleShortVersionString":    "1.2.4",
		"CFBundleVersion":               "1.2.4",
		"LSApplicationCategoryType":     "public.app-category.strategy-games",
		"LSMinimumSystemVersion":        "13.0",
		"LSUIElement":                   true,
	}
	for k, v := range want {
		if values[k] != v {
			t.Errorf("%s = %#v, want %#v", k, values[k], v)
		}
	}
	for _, k := range keys {
		if _, ok := want[k]; !ok && k != "NSHumanReadableCopyright" {
			t.Errorf("unexpected key %s", k)
		}
	}
	if c, _ := values["NSHumanReadableCopyright"].(string); !strings.HasPrefix(c, "Copyright (c) 2026 GhostofGoes.") {
		t.Errorf("NSHumanReadableCopyright = %q", c)
	}
	if !slices.IsSorted(keys) {
		t.Errorf("keys are not in order: %v", keys)
	}
	// CFBundleName is at most 15 characters; the executable must be the program's name.
	if n := values["CFBundleName"].(string); len(n) > 15 {
		t.Errorf("CFBundleName %q is longer than 15 characters", n)
	}
	again, err := infoPlist("1.2.4", "13.0")
	if err != nil || !bytes.Equal(again, data) {
		t.Error("two runs differ")
	}
}

func TestRunWritesBundle(t *testing.T) {
	t.Parallel()
	root, o := fixture(t, universal)
	o.version = "1.2.4-snapshot.abc1234"
	if err := run(root, options{binary: o.binary, version: o.version, icon: o.icon, out: o.out + string(filepath.Separator)}); err != nil {
		t.Fatal(err)
	}
	want := map[string]fs.FileMode{
		"Contents":                                   fs.ModeDir | 0o755,
		"Contents/Info.plist":                        0o644,
		"Contents/PkgInfo":                           0o644,
		"Contents/MacOS":                             fs.ModeDir | 0o755,
		"Contents/MacOS/wopr":                        0o755,
		"Contents/Resources":                         fs.ModeDir | 0o755,
		"Contents/Resources/WOPR.icns":               0o644,
		"Contents/Resources/LICENSE":                 0o644,
		"Contents/Resources/NOTICE.md":               0o644,
		"Contents/Resources/THIRD_PARTY_NOTICES.txt": 0o644,
	}
	got := map[string]fs.FileMode{}
	err := filepath.WalkDir(o.out, func(path string, d fs.DirEntry, err error) error {
		if err != nil || path == o.out {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(o.out, path)
		got[filepath.ToSlash(rel)] = info.Mode()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for name, mode := range want {
		m, ok := got[name]
		switch {
		case !ok:
			t.Errorf("%s is missing", name)
		case runtime.GOOS != "windows" && m != mode:
			t.Errorf("%s has mode %v, want %v", name, m, mode)
		}
	}
	for name := range got {
		if _, ok := want[name]; !ok {
			t.Errorf("unexpected %s", name)
		}
	}

	read := func(name string) []byte {
		data, err := os.ReadFile(filepath.Join(o.out, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	if got := read("Contents/PkgInfo"); string(got) != "APPL????" {
		t.Errorf("PkgInfo = %q, want APPL????", got)
	}
	if !bytes.Equal(read("Contents/MacOS/wopr"), universal) {
		t.Error("Contents/MacOS/wopr is not the program")
	}
	if !bytes.Equal(read("Contents/Resources/WOPR.icns"), icns()) {
		t.Error("Contents/Resources/WOPR.icns is not the icon")
	}
	for _, n := range notices {
		if got := read("Contents/Resources/" + n); string(got) != "text of "+n+"\n" {
			t.Errorf("%s = %q", n, got)
		}
	}
	_, values := plistValues(t, read("Contents/Info.plist"))
	for k, v := range map[string]string{
		"CFBundleShortVersionString": "1.2.4",
		"CFBundleVersion":            "1.2.4",
		"LSMinimumSystemVersion":     "13.0",
		"CFBundleExecutable":         "wopr",
		"CFBundleIconFile":           "WOPR",
	} {
		if values[k] != v {
			t.Errorf("%s = %#v, want %q", k, values[k], v)
		}
	}
}

func TestRunRefuses(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		program []byte
		change  func(root string, o *options)
		want    string
	}{
		"missing program": {universal, func(_ string, o *options) { o.binary += ".missing" }, "wopr.missing"},
		"a script":        {[]byte("#!/bin/sh\nexec wopr\n"), nil, "not a Mach-O program"},
		"a Linux program": {append([]byte("\x7fELF\x02\x01\x01"), make([]byte, 64)...), nil, "not a Mach-O program"},
		"an object file":  {machO(slice{cpuARM64, 1, 13 << 16}), nil, "not an executable"},
		"no minimum":      {fat(slice{cpuAMD64, 2, 13 << 16}, slice{cpuARM64, 2, 0}), nil, "no LC_BUILD_VERSION"},
		"bad version":     {universal, func(_ string, o *options) { o.version = "v1.2.3" }, "not a version"},
		"missing icon":    {universal, func(_ string, o *options) { o.icon += ".missing" }, "-icon"},
		"bad icon": {universal, func(_ string, o *options) {
			if err := os.WriteFile(o.icon, []byte("\x89PNG\r\n\x1a\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, "not an .icns file"},
		"not .app":       {universal, func(_ string, o *options) { o.out = filepath.Join(filepath.Dir(o.out), "WOPR") }, "must end in .app"},
		"only .app":      {universal, func(_ string, o *options) { o.out = filepath.Join(filepath.Dir(o.out), ".app") }, "must end in .app"},
		"out exists":     {universal, func(_ string, o *options) { mkdir(t, o.out) }, "already exists"},
		"missing notice": {universal, func(root string, _ *options) { _ = os.Remove(filepath.Join(root, "NOTICE.md")) }, "NOTICE.md"},
	} {
		root, o := fixture(t, tc.program)
		if tc.change != nil {
			tc.change(root, &o)
		}
		_, existed := os.Lstat(o.out)
		err := run(root, o)
		if err == nil || !strings.Contains(err.Error(), tc.want) || strings.HasPrefix(name, "missing") != errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s: run = %v, want an error containing %q", name, err, tc.want)
			continue
		}
		if _, statErr := os.Lstat(o.out); existed != nil && statErr == nil {
			t.Errorf("%s: a failed run left %s behind", name, o.out)
		}
	}
}

func mkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

// Go's own linker output, for both architectures: macapp must read what the release builds.
func TestMinimumMacOSOfGoBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a program for darwin")
	}
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module hello\n\ngo 1.27\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, arch := range []string{"amd64", "arm64"} {
		bin := filepath.Join(dir, "hello_"+arch)
		cmd := exec.Command("go", "build", "-trimpath", "-ldflags=-s -w", "-o", bin, ".")
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=darwin", "GOARCH="+arch, "GOFLAGS=")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go build for darwin/%s: %v\n%s", arch, err, out)
		}
		got, err := minimumMacOS(bin)
		if err != nil {
			t.Fatalf("darwin/%s: %v", arch, err)
		}
		if !regexp.MustCompile(`^(1[1-9]|[2-9]\d)\.\d+(\.\d+)?$`).MatchString(got) {
			t.Errorf("darwin/%s: minimum macOS %q, want 11.0 or later", arch, got)
		}
		t.Logf("darwin/%s: LSMinimumSystemVersion %s", arch, got)
	}
}
