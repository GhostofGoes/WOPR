// Command macapp writes WOPR.app, the macOS application bundle that the release's .dmg holds
// (packaging/macos/build-dmg.sh). It runs on any system, so its tests run on Linux:
//
//	go run ./internal/tools/macapp -binary build/wopr -version 1.2.3 [-icon packaging/icons/wopr.icns]
//	    [-notices dist/release] -out build/WOPR.app
//
// It writes:
//
//	Contents/Info.plist              the bundle's identity and version, an XML property list
//	Contents/PkgInfo                 APPL????, the type and creator codes older Finders read
//	Contents/MacOS/wopr              -binary: the program, universal (lipo) in a release
//	Contents/Resources/WOPR.icns     -icon
//	Contents/Resources/LICENSE, NOTICE.md, THIRD_PARTY_NOTICES.txt   from -notices
//
// -notices is the repository by default; build-dmg.sh passes the release files, which it has
// checked against checksums.txt, as it has the programs.
//
// The bundle's executable is wopr itself. Opened from Finder it has no terminal, so it opens
// Terminal running itself (cmd/wopr/launch_darwin.go). The bundle is not signed here: codesign
// must come after every file is in place, so build-dmg.sh signs it.
//
// Info.plist's versions are the numeric X.Y.Z that macOS requires, so a snapshot's suffix is
// dropped. Its minimum macOS is the newest one any of the program's architectures declares in
// its LC_BUILD_VERSION load command, which Go's linker writes, so it follows Go's own minimum.
package main

import (
	"bytes"
	"debug/macho"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
)

// Bundle identity (docs/PLAN.md §8). The identifier is the app ID used on every platform and can
// never change once shipped: macOS keys an app's settings and permissions to it.
const (
	bundleID   = "io.github.ghostofgoes.wopr"
	bundleName = "WOPR"
	executable = "wopr"
	iconName   = "WOPR" // Contents/Resources/WOPR.icns; CFBundleIconFile omits the extension
)

// notices are the files the licences require next to the program.
var notices = []string{"LICENSE", "NOTICE.md", "THIRD_PARTY_NOTICES.txt"}

func main() {
	bin := flag.String("binary", "", "the darwin program to bundle (a universal binary from lipo, in a release)")
	version := flag.String("version", "", "the version being built, as GoReleaser's {{ .Version }} gives it (1.2.3 or 1.2.4-snapshot.abc1234)")
	icon := flag.String("icon", "packaging/icons/wopr.icns", "the app's icon, an .icns file")
	noticeDir := flag.String("notices", ".", "the directory holding LICENSE, NOTICE.md and THIRD_PARTY_NOTICES.txt")
	out := flag.String("out", "", "the bundle to write, a directory named <name>.app that must not exist yet")
	flag.Parse()
	if *bin == "" || *version == "" || *out == "" || flag.NArg() > 0 {
		fmt.Fprintln(os.Stderr, "macapp: use -binary FILE -version VERSION [-icon FILE] [-notices DIR] -out DIR.app")
		os.Exit(2)
	}
	if err := run(*noticeDir, options{binary: *bin, version: *version, icon: *icon, out: *out}); err != nil {
		fmt.Fprintln(os.Stderr, "macapp:", err)
		os.Exit(1)
	}
	fmt.Println("macapp: wrote", *out)
}

type options struct {
	binary, version, icon, out string
}

// run writes the bundle o.out, with the notices in noticeDir. On an error it removes what it
// wrote.
func run(noticeDir string, o options) (err error) {
	o.out = filepath.Clean(o.out) // WOPR.app/ is WOPR.app
	short, err := bundleVersion(o.version)
	if err != nil {
		return err
	}
	minOS, err := minimumMacOS(o.binary)
	if err != nil {
		return fmt.Errorf("-binary %s: %w", o.binary, err)
	}
	icns, err := os.ReadFile(o.icon)
	if err != nil {
		return fmt.Errorf("-icon: %w", err)
	}
	if err := checkICNS(icns); err != nil {
		return fmt.Errorf("-icon %s: %w", o.icon, err)
	}
	if filepath.Ext(o.out) != ".app" || len(filepath.Base(o.out)) <= len(".app") {
		return fmt.Errorf("-out %s: a bundle's name must end in .app", o.out)
	}
	if _, err := os.Lstat(o.out); err == nil {
		return fmt.Errorf("-out %s already exists; remove it first", o.out)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	plist, err := infoPlist(short, minOS)
	if err != nil {
		return err
	}
	type file struct {
		path string // within the bundle
		data []byte
		src  string // or copied from here
		mode fs.FileMode
	}
	files := []file{
		{path: "Contents/Info.plist", data: plist, mode: 0o644},
		{path: "Contents/PkgInfo", data: []byte("APPL????"), mode: 0o644},
		{path: "Contents/MacOS/" + executable, src: o.binary, mode: 0o755},
		{path: "Contents/Resources/" + iconName + ".icns", data: icns, mode: 0o644},
	}
	for _, n := range notices {
		files = append(files, file{path: "Contents/Resources/" + n, src: filepath.Join(noticeDir, n), mode: 0o644})
	}

	if err := os.MkdirAll(o.out, 0o755); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(o.out)
		}
	}()
	for _, f := range files {
		data := f.data
		if f.src != "" {
			if data, err = os.ReadFile(f.src); err != nil {
				return err
			}
		}
		if err := writeFile(filepath.Join(o.out, filepath.FromSlash(f.path)), data, f.mode); err != nil {
			return err
		}
	}
	return nil
}

// writeFile writes data with exactly mode (and its directories with 0755), whatever the umask.
func writeFile(path string, data []byte, mode fs.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, mode); err != nil {
		return err
	}
	return os.Chmod(path, mode)
}

// versionRE is a version as GoReleaser's {{ .Version }} gives it: no "v", and an optional
// pre-release suffix such as a snapshot's.
var versionRE = regexp.MustCompile(`^(\d+\.\d+\.\d+)(-[0-9A-Za-z.-]+)?$`)

// bundleVersion is the X.Y.Z that CFBundleShortVersionString and CFBundleVersion take: Apple
// allows only digits and periods, so 1.2.4-snapshot.abc1234 becomes 1.2.4.
func bundleVersion(v string) (string, error) {
	m := versionRE.FindStringSubmatch(v)
	if m == nil {
		return "", fmt.Errorf("-version %q is not a version like 1.2.3 or 1.2.4-snapshot.abc1234", v)
	}
	return m[1], nil
}

// Load command and platform numbers from <mach-o/loader.h>, which debug/macho does not name.
const (
	lcBuildVersion = 0x32 // LC_BUILD_VERSION
	platformMacOS  = 1    // PLATFORM_MACOS
)

// minimumMacOS reads the program, a thin or universal Mach-O executable, and returns the newest
// minimum macOS its architectures declare, as "13.0". It refuses anything else.
func minimumMacOS(path string) (string, error) {
	var parts []*macho.File
	fat, err := macho.OpenFat(path)
	switch {
	case err == nil:
		defer func() { _ = fat.Close() }()
		for _, a := range fat.Arches {
			parts = append(parts, a.File)
		}
	case errors.Is(err, macho.ErrNotFat):
		f, err := macho.Open(path)
		if err != nil {
			return "", notMachO(err)
		}
		defer func() { _ = f.Close() }()
		parts = append(parts, f)
	default:
		return "", notMachO(err)
	}
	var newest uint32
	for _, f := range parts {
		if f.Type != macho.TypeExec {
			return "", fmt.Errorf("the %s part is a Mach-O %s, not an executable", f.Cpu, f.Type)
		}
		minos, ok := buildMinOS(f)
		if !ok {
			return "", fmt.Errorf("the %s part has no LC_BUILD_VERSION for macOS", f.Cpu)
		}
		newest = max(newest, minos)
	}
	return formatVersion(newest), nil
}

// notMachO keeps a missing file's error as it is, and says what is wrong with any other.
func notMachO(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return fmt.Errorf("not a Mach-O program: %w", err)
}

// buildMinOS returns the minimum OS version from f's LC_BUILD_VERSION for macOS:
// struct build_version_command { cmd, cmdsize, platform, minos, sdk, ntools uint32 }.
func buildMinOS(f *macho.File) (uint32, bool) {
	for _, l := range f.Loads {
		raw := l.Raw()
		if len(raw) < 24 || f.ByteOrder.Uint32(raw) != lcBuildVersion {
			continue
		}
		if f.ByteOrder.Uint32(raw[8:]) == platformMacOS {
			return f.ByteOrder.Uint32(raw[12:]), true
		}
	}
	return 0, false
}

// formatVersion turns a Mach-O version, xxxx.yy.zz in nibbles (16, 8 and 8 bits), into "13.0" or
// "13.0.1".
func formatVersion(v uint32) string {
	s := fmt.Sprintf("%d.%d", v>>16, v>>8&0xff)
	if patch := v & 0xff; patch != 0 {
		s += fmt.Sprintf(".%d", patch)
	}
	return s
}

// checkICNS checks that data is an .icns file: the magic "icns", then the whole file's length.
func checkICNS(data []byte) error {
	if len(data) < 8 || string(data[:4]) != "icns" {
		return errors.New("not an .icns file")
	}
	if n := binary.BigEndian.Uint32(data[4:8]); int(n) != len(data) {
		return fmt.Errorf("the .icns header says %d bytes, the file has %d", n, len(data))
	}
	return nil
}

// entry is one Info.plist key and its value: a string or a bool.
type entry struct {
	key   string
	value any
}

// plistEntries is Info.plist's contents, in the order it is written: sorted by key, as Xcode
// writes them, so a diff of two builds shows only what changed.
func plistEntries(version, minOS string) []entry {
	return []entry{
		{"CFBundleDevelopmentRegion", "en"},
		{"CFBundleDisplayName", bundleName},
		{"CFBundleExecutable", executable},
		{"CFBundleIconFile", iconName},
		{"CFBundleIdentifier", bundleID},
		{"CFBundleInfoDictionaryVersion", "6.0"},
		{"CFBundleName", bundleName},
		{"CFBundlePackageType", "APPL"},
		{"CFBundleShortVersionString", version},
		{"CFBundleVersion", version},
		// Most of WOPR's games are strategy and war games, as the film's computer was built for.
		{"LSApplicationCategoryType", "public.app-category.strategy-games"},
		{"LSMinimumSystemVersion", minOS},
		// An agent app has no Dock icon. The bundle's own process only opens Terminal and exits
		// (cmd/wopr/launch_darwin.go), so without this its icon would flash in the Dock. The game
		// runs in Terminal, whose icon the Dock shows instead.
		{"LSUIElement", true},
		{"NSHumanReadableCopyright", "Copyright (c) 2026 GhostofGoes. The code is under the MIT License; see NOTICE.md."},
	}
}

// infoPlist writes Info.plist as an XML property list, the format Xcode writes.
func infoPlist(version, minOS string) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString(xml.Header)
	b.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	b.WriteString(`<plist version="1.0">` + "\n<dict>\n")
	for _, e := range plistEntries(version, minOS) {
		b.WriteString("\t<key>")
		if err := xml.EscapeText(&b, []byte(e.key)); err != nil {
			return nil, err
		}
		b.WriteString("</key>\n\t")
		switch v := e.value.(type) {
		case string:
			b.WriteString("<string>")
			if err := xml.EscapeText(&b, []byte(v)); err != nil {
				return nil, err
			}
			b.WriteString("</string>")
		case bool:
			if v {
				b.WriteString("<true/>")
			} else {
				b.WriteString("<false/>")
			}
		default:
			return nil, fmt.Errorf("key %s: unsupported value %T", e.key, v)
		}
		b.WriteString("\n")
	}
	b.WriteString("</dict>\n</plist>\n")
	return b.Bytes(), nil
}
