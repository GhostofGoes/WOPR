// Command snapdir prepares the directory that `snap pack` turns into wopr's snap, for one
// architecture (.github/workflows/snap.yml):
//
//	go run ./internal/tools/snapdir -dist dist/release -version 1.2.3 -arch amd64 -out build/snap/amd64
//
// It writes:
//
//	meta/snap.yaml          packaging/snap/snap.yaml.in, filled in
//	meta/gui/wopr.desktop   the .deb's and the .rpm's menu entry, in the form snapd wants
//	meta/gui/icon.png       the 256 px icon from packaging/icons/hicolor
//	bin/wopr                the release's program, wopr_<version>_linux_<arch>
//	LICENSE, NOTICE.md, THIRD_PARTY_NOTICES.txt   the release's notices
//
// The program and the notices come from -dist and must match its checksums.txt, so the snap holds
// exactly what the release attests: snapdir never builds wopr. The packaging files come from the
// repository, -root. A version, summary or description that snapd or the Snap Store would refuse
// fails here, not at upload. Every file's contents and mode depend only on those inputs.
//
// A Go program rather than a script, like the other tools here, so that its checks are tested
// on any system.
package main

import (
	"bytes"
	"crypto/sha256"
	"debug/elf"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"image"
	_ "image/png" // image.DecodeConfig reads the icon
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

// appID names wopr on every platform (docs/PLAN.md §8): the menu entry, the icon and the snap's
// common-id.
const appID = "io.github.ghostofgoes.wopr"

// The packaging files, in the repository.
const (
	snapTemplate    = "packaging/snap/snap.yaml.in"
	desktopTemplate = "packaging/linux/" + appID + ".desktop.in"
	descriptionFile = "packaging/description.txt"
	iconFile        = "packaging/icons/hicolor/256x256/apps/" + appID + ".png"
)

// iconSize is the store's and snapd's icon, meta/gui/icon.png: 256 by 256 pixels.
const iconSize = 256

// notices are the files the licences require next to the program.
var notices = []string{"LICENSE", "NOTICE.md", "THIRD_PARTY_NOTICES.txt"}

// machines are the architectures the release builds for Linux (Debian's names, which snaps use),
// with their ELF machines.
var machines = map[string]elf.Machine{"amd64": elf.EM_X86_64, "arm64": elf.EM_AARCH64}

func main() {
	dist := flag.String("dist", "", "the release files: wopr_<version>_linux_<arch>, the notices and checksums.txt")
	version := flag.String("version", "", "the version being built, as GoReleaser's {{ .Version }} gives it (1.2.3 or 1.2.4-snapshot.abc1234)")
	arch := flag.String("arch", "", "the architecture to package: amd64 or arm64")
	root := flag.String("root", ".", "the repository, for the packaging files")
	out := flag.String("out", "", "the directory to write, which must not exist yet")
	flag.Parse()
	if *dist == "" || *version == "" || *arch == "" || *out == "" || flag.NArg() > 0 {
		fmt.Fprintln(os.Stderr, "snapdir: use -dist DIR -version VERSION -arch amd64|arm64 [-root DIR] -out DIR")
		os.Exit(2)
	}
	if err := run(options{dist: *dist, version: *version, arch: *arch, root: *root, out: *out}); err != nil {
		fmt.Fprintln(os.Stderr, "snapdir:", err)
		os.Exit(1)
	}
	fmt.Println("snapdir: wrote", *out)
}

type options struct {
	dist, version, arch, root, out string
}

// file is one file of the snap.
type file struct {
	path string // within the snap
	data []byte
	mode fs.FileMode
}

// run writes the directory o.out. On an error it removes what it wrote.
func run(o options) (err error) {
	if err := checkVersion(o.version); err != nil {
		return err
	}
	machine, ok := machines[o.arch]
	if !ok {
		return fmt.Errorf("-arch %q: want amd64 or arm64", o.arch)
	}
	if _, err := os.Lstat(o.out); err == nil {
		return fmt.Errorf("-out %s already exists; remove it first", o.out)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	program := fmt.Sprintf("wopr_%s_linux_%s", o.version, o.arch)
	released, err := readReleased(o.dist, append([]string{program}, notices...))
	if err != nil {
		return err
	}
	if err := checkELF(released[program], machine); err != nil {
		return fmt.Errorf("%s: %w", program, err)
	}

	repo := func(name string) ([]byte, error) {
		return os.ReadFile(filepath.Join(o.root, filepath.FromSlash(name)))
	}
	desc, err := repo(descriptionFile)
	if err != nil {
		return err
	}
	tmpl, err := repo(snapTemplate)
	if err != nil {
		return err
	}
	snapYAML, err := snapMetadata(string(tmpl), string(desc), o.version, o.arch)
	if err != nil {
		return err
	}
	if tmpl, err = repo(desktopTemplate); err != nil {
		return err
	}
	desktop, err := snapDesktop(string(tmpl))
	if err != nil {
		return fmt.Errorf("%s: %w", desktopTemplate, err)
	}
	icon, err := repo(iconFile)
	if err != nil {
		return err
	}
	if err := checkIcon(icon); err != nil {
		return fmt.Errorf("%s: %w", iconFile, err)
	}

	files := []file{
		{"meta/snap.yaml", []byte(snapYAML), 0o644},
		{"meta/gui/wopr.desktop", []byte(desktop), 0o644},
		{"meta/gui/icon.png", icon, 0o644},
		{"bin/wopr", released[program], 0o755},
	}
	for _, n := range notices {
		files = append(files, file{n, released[n], 0o644})
	}

	// snap pack refuses a directory that others cannot read, so the modes are set whatever the
	// umask.
	if err := mkdir(o.out); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(o.out)
		}
	}()
	for _, f := range files {
		path := filepath.Join(o.out, filepath.FromSlash(f.path))
		if err := mkdir(filepath.Dir(path)); err != nil {
			return err
		}
		if err := os.WriteFile(path, f.data, f.mode); err != nil {
			return err
		}
		if err := os.Chmod(path, f.mode); err != nil {
			return err
		}
	}
	return nil
}

// mkdir makes dir and its parents, and sets dir's mode to 0755.
func mkdir(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.Chmod(dir, 0o755)
}

// readReleased reads each named file from dist, and checks it against dist/checksums.txt, which
// must list it once.
func readReleased(dist string, names []string) (map[string][]byte, error) {
	data, err := os.ReadFile(filepath.Join(dist, "checksums.txt"))
	if err != nil {
		return nil, err
	}
	sums, err := parseChecksums(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Join(dist, "checksums.txt"), err)
	}
	files := map[string][]byte{}
	for _, name := range names {
		want, ok := sums[name]
		if !ok {
			return nil, fmt.Errorf("checksums.txt does not list %s", name)
		}
		data, err := os.ReadFile(filepath.Join(dist, name))
		if err != nil {
			return nil, err
		}
		if got := sha256.Sum256(data); hex.EncodeToString(got[:]) != want {
			return nil, fmt.Errorf("%s: sha256 %x, but checksums.txt says %s", name, got, want)
		}
		files[name] = data
	}
	return files, nil
}

// parseChecksums reads a checksums.txt as GoReleaser writes it: "<sha256>  <name>" on each line.
func parseChecksums(data []byte) (map[string]string, error) {
	sums := map[string]string{}
	for line := range strings.Lines(string(data)) {
		fields := strings.Fields(line)
		if len(fields) != 2 || len(fields[0]) != 2*sha256.Size {
			return nil, fmt.Errorf("malformed line %q", strings.TrimSpace(line))
		}
		if _, err := hex.DecodeString(fields[0]); err != nil {
			return nil, fmt.Errorf("malformed line %q", strings.TrimSpace(line))
		}
		if _, dup := sums[fields[1]]; dup {
			return nil, fmt.Errorf("%s is listed twice", fields[1])
		}
		sums[fields[1]] = strings.ToLower(fields[0])
	}
	return sums, nil
}

// checkELF checks that data is a 64-bit ELF program for machine.
func checkELF(data []byte, machine elf.Machine) error {
	f, err := elf.NewFile(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("not an ELF program: %w", err)
	}
	if f.Class != elf.ELFCLASS64 || f.Machine != machine {
		return fmt.Errorf("an ELF program for %s %s, want %s %s", f.Class, f.Machine, elf.ELFCLASS64, machine)
	}
	return nil
}

// checkIcon checks that data is a PNG of iconSize by iconSize pixels.
func checkIcon(data []byte) error {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return err
	}
	if format != "png" || cfg.Width != iconSize || cfg.Height != iconSize {
		return fmt.Errorf("a %dx%d %s, want a %dx%d png", cfg.Width, cfg.Height, format, iconSize, iconSize)
	}
	return nil
}

// isValidVersion is snapd's rule for a snap's version (snap/validate.go): at most 32 characters
// from this set, starting and ending with a letter or digit (or + or ~ at the end).
var isValidVersion = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9:.+~-]{0,30}[a-zA-Z0-9+~])?$`).MatchString

// versionRE is a version as GoReleaser's {{ .Version }} gives it.
var versionRE = regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`)

func checkVersion(v string) error {
	if !versionRE.MatchString(v) {
		return fmt.Errorf("-version %q is not a version like 1.2.3 or 1.2.4-snapshot.abc1234", v)
	}
	if !isValidVersion(v) {
		return fmt.Errorf("-version %q is not a snap version: at most 32 characters, ending in a letter or digit", v)
	}
	return nil
}

// Limits on the metadata. Snapcraft documents a summary of at most 78 characters, which is stricter
// than snapd's 128; snapd allows a description of 4096.
const (
	maxSummary     = 78
	maxDescription = 4096
)

// snapMetadata fills the snap.yaml template: the version, the architecture, and the summary and
// description from description.txt, whose first line is the summary, as for the .deb and the .rpm.
func snapMetadata(tmpl, description, version, arch string) (string, error) {
	summary, body, _ := strings.Cut(strings.TrimSpace(description), "\n")
	if summary == "" || utf8.RuneCountInString(summary) > maxSummary {
		return "", fmt.Errorf("%s: the summary, its first line, must have 1 to %d characters: %q", descriptionFile, maxSummary, summary)
	}
	lines := strings.Split(strings.TrimSpace(body), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return "", fmt.Errorf("%s: no description after the summary", descriptionFile)
	}
	doc, err := fill(tmpl, map[string][]string{
		"VERSION":     {version},
		"ARCH":        {arch},
		"SUMMARY":     {yamlQuote(summary)},
		"DESCRIPTION": lines,
	})
	if err != nil {
		return "", fmt.Errorf("%s: %w", snapTemplate, err)
	}
	desc, ok := literalBlock(doc, "description")
	if !ok {
		return "", fmt.Errorf("%s: no description: | block", snapTemplate)
	}
	if n := utf8.RuneCountInString(desc); n > maxDescription {
		return "", fmt.Errorf("the description has %d characters; snapd allows %d", n, maxDescription)
	}
	return doc, nil
}

// yamlQuote quotes s as a YAML single-quoted scalar, so that no character in it means anything.
func yamlQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// literalBlock returns the value of the top-level literal block scalar "key: |" in doc, the way
// YAML reads it: each line without the two-space indent, and one newline at the end.
func literalBlock(doc, key string) (string, bool) {
	var b strings.Builder
	found := false
	for line := range strings.Lines(doc) {
		switch {
		case !found:
			found = strings.TrimRight(line, "\n") == key+": |"
		case strings.HasPrefix(line, "  "):
			b.WriteString(line[2:])
		case strings.TrimSpace(line) == "":
			b.WriteString("\n")
		default:
			return strings.TrimRight(b.String(), "\n") + "\n", true
		}
	}
	return strings.TrimRight(b.String(), "\n") + "\n", found
}

// placeholder is a value to fill into a template: @NAME@.
var placeholder = regexp.MustCompile(`@([A-Z]+)@`)

// fill replaces each @NAME@ in a template with values[NAME], as internal/tools/pkgdocs does. A
// placeholder alone on its line stands for any number of lines, each indented as the placeholder
// is; anywhere else its value must be one line. Every placeholder needs a value and every value
// must be used.
func fill(tmpl string, values map[string][]string) (string, error) {
	used := map[string]bool{}
	var b strings.Builder
	var err error
	for line := range strings.Lines(tmpl) {
		if m := placeholder.FindStringSubmatch(line); m != nil && strings.TrimSpace(line) == m[0] {
			v, ok := values[m[1]]
			if !ok {
				return "", fmt.Errorf("no value for %s", m[0])
			}
			used[m[1]] = true
			indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
			for _, l := range v {
				if l != "" {
					b.WriteString(indent + l)
				}
				b.WriteString("\n")
			}
			continue
		}
		b.WriteString(placeholder.ReplaceAllStringFunc(line, func(p string) string {
			name := strings.Trim(p, "@")
			v, ok := values[name]
			if !ok || len(v) != 1 || strings.Contains(v[0], "\n") {
				err = errors.Join(err, fmt.Errorf("%s needs a one-line value", p))
				return p
			}
			used[name] = true
			return v[0]
		}))
	}
	if err != nil {
		return "", err
	}
	for _, name := range slices.Sorted(maps.Keys(values)) {
		if !used[name] {
			return "", fmt.Errorf("no @%s@ to fill", name)
		}
	}
	return b.String(), nil
}

// The snap's menu entry changes two keys of the Linux packages' one: Exec names the snap's app,
// which snapd turns into /snap/bin/wopr (in the actions too), and Icon is a file in the snap,
// whose ${SNAP} snapd replaces with /snap/wopr/current (wrappers/desktop.go).
const (
	snapExec = "wopr"
	snapIcon = "${SNAP}/meta/gui/icon.png"
)

// desktopHeader replaces the template's own comment, which is about the .deb and the .rpm.
const desktopHeader = `# wopr's menu entry in its snap, which snapd installs as
# /var/lib/snapd/desktop/applications/wopr_wopr.desktop. internal/tools/snapdir writes it from the
# .deb's and the .rpm's (packaging/linux/io.github.ghostofgoes.wopr.desktop.in), with Exec naming
# the snap's app and Icon a file in the snap, as snapd wants.
`

// isSnapdLine matches the lines snapd keeps in a snap's desktop file (isValidDesktopFileLine in
// snapd's wrappers/desktop.go). It drops any other without a word, so snapDesktop refuses them
// rather than lose them.
var isSnapdLine = regexp.MustCompile(`^(\s*$|\s*#|\[Desktop Entry\]$|\[Desktop Action [0-9A-Za-z-]+\]$|` +
	`(Type|Version|NoDisplay|Icon|Hidden|OnlyShowIn|NotShowIn|Exec|Terminal|Actions|MimeType|Categories|` +
	`StartupNotify|StartupWMClass|PrefersNonDefaultGPU|SingleMainWindow|X-Ayatana-Desktop-Shortcuts|TargetEnvironment)=|` +
	`(Name|GenericName|Comment|Keywords)(\[[a-z]+(_[A-Z]+)?(\.[0-9A-Z-]+)?(@[a-z]+)?\])?=)`).MatchString

// snapDesktop turns the Linux packages' menu entry template into the snap's entry.
func snapDesktop(tmpl string) (string, error) {
	const program = "Exec=@BINDIR@/wopr"
	var b strings.Builder
	b.WriteString(desktopHeader)
	header := true
	icons, execs := 0, 0
	for line := range strings.Lines(tmpl) {
		line = strings.TrimRight(line, "\n")
		if header && strings.HasPrefix(line, "#") {
			continue
		}
		header = false
		switch {
		case strings.HasPrefix(line, "Exec="):
			args, ok := strings.CutPrefix(line, program)
			if !ok || (args != "" && !strings.HasPrefix(args, " ")) {
				return "", fmt.Errorf("%q does not start %s", line, program)
			}
			line = "Exec=" + snapExec + args
			execs++
		case strings.HasPrefix(line, "Icon="):
			line = "Icon=" + snapIcon
			icons++
		}
		if m := placeholder.FindString(line); m != "" {
			return "", fmt.Errorf("%q: the snap's entry has no value for %s", line, m)
		}
		if !isSnapdLine(line) {
			return "", fmt.Errorf("%q: snapd drops this line from a snap's menu entry", line)
		}
		b.WriteString(line + "\n")
	}
	if icons != 1 || execs == 0 {
		return "", fmt.Errorf("want one Icon= and at least one Exec=, found %d and %d", icons, execs)
	}
	return b.String(), nil
}
