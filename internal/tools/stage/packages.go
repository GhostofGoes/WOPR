package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"
)

// The Linux packages (.goreleaser.yaml's nfpms): one .deb and one .rpm for each Linux build.
// With -archives, stage checks that each is named as its format names packages and installs what
// it must, reading only what the standard library can: a .deb's control.tar.gz, and an .rpm's
// header. The smoke jobs then install them with dpkg and rpm.

const linuxPackage = "Linux Package"

// pkg reports whether a is a .deb or an .rpm.
func (a artifact) pkg() bool { return a.Type == linuxPackage }

// debArch and rpmArch name Go's architectures as Debian and RPM do.
var (
	debArch = map[string]string{"amd64": "amd64", "arm64": "arm64"}
	rpmArch = map[string]string{"amd64": "x86_64", "arm64": "aarch64"}
)

// debFiles are the files every .deb installs: the program where Debian puts games (Policy
// §11.11), its manual page (§12.1), and in /usr/share/doc/wopr the copyright file (§12.5), the
// Debian changelog and the release notes (§12.7), the README and the notices.
var debFiles = []string{
	"usr/games/wopr",
	"usr/share/man/man6/wopr.6.gz",
	"usr/share/doc/wopr/copyright",
	"usr/share/doc/wopr/changelog.Debian.gz",
	"usr/share/doc/wopr/NEWS.gz",
	"usr/share/doc/wopr/README.md.gz",
	"usr/share/doc/wopr/NOTICE.md.gz",
}

// The RPM file flags that mark documentation (%doc) and licences (%license).
const (
	rpmDoc     = 1 << 1
	rpmLicense = 1 << 7
)

// rpmFiles are the files every .rpm installs, with the flags it must mark them with, and the two
// directories it owns: the program in /usr/bin, as Fedora puts games, the manual page, the
// documents in /usr/share/doc/wopr and the licences in /usr/share/licenses/wopr.
var rpmFiles = map[string]uint32{
	"/usr/bin/wopr":                                    0,
	"/usr/share/man/man6/wopr.6.gz":                    rpmDoc,
	"/usr/share/doc/wopr":                              0,
	"/usr/share/doc/wopr/README.md":                    rpmDoc,
	"/usr/share/doc/wopr/CHANGELOG.md":                 rpmDoc,
	"/usr/share/licenses/wopr":                         0,
	"/usr/share/licenses/wopr/LICENSE":                 rpmLicense,
	"/usr/share/licenses/wopr/NOTICE.md":               rpmLicense,
	"/usr/share/licenses/wopr/THIRD_PARTY_NOTICES.txt": rpmLicense,
}

// checkPackages checks that every Linux build has exactly one .deb and one .rpm, and checks each.
func checkPackages(arts []artifact) error {
	have := map[string]int{} // "<arch> <format>" -> count
	for _, a := range arts {
		if !a.pkg() {
			continue
		}
		var err error
		switch a.Extra.Format {
		case "deb":
			err = checkDeb(a)
		case "rpm":
			err = checkRPM(a)
		default:
			err = fmt.Errorf("unexpected package format %q", a.Extra.Format)
		}
		if err != nil {
			return fmt.Errorf("%s: %w", a.Name, err)
		}
		have[a.Goarch+" "+a.Extra.Format]++
		fmt.Println("package ok:", a.Name)
	}
	for _, a := range arts {
		if a.Type != "Binary" || a.bare() || a.Goos != "linux" {
			continue
		}
		for _, format := range []string{"deb", "rpm"} {
			if n := have[a.Goarch+" "+format]; n != 1 {
				return fmt.Errorf("linux/%s has %d .%s packages, want 1; see nfpms in .goreleaser.yaml", a.Goarch, n, format)
			}
		}
	}
	return nil
}

// checkDeb reads a .deb's control file and the list of files it installs (md5sums), from
// control.tar.gz, and checks them and the package's file name.
func checkDeb(a artifact) error {
	members, err := readAr(a.Path)
	if err != nil {
		return err
	}
	if string(members["debian-binary"]) != "2.0\n" {
		return errors.New("debian-binary is not 2.0")
	}
	names := slices.Sorted(maps.Keys(members))
	if !slices.ContainsFunc(names, func(n string) bool { return strings.HasPrefix(n, "data.tar") }) {
		return errors.New("no data.tar member")
	}
	ctl, ok := members["control.tar.gz"]
	if !ok {
		return fmt.Errorf("no control.tar.gz (has %v)", names)
	}
	files, err := untarGz(ctl)
	if err != nil {
		return fmt.Errorf("control.tar.gz: %w", err)
	}
	control := parseControl(string(files["control"]))
	want := map[string]string{
		"Package": "wopr", "Architecture": debArch[a.Goarch], "Section": "games", "Priority": "optional",
	}
	for k, v := range want {
		if control[k] != v {
			return fmt.Errorf("control has %s %q, want %q", k, control[k], v)
		}
	}
	for _, k := range []string{"Version", "Maintainer", "Homepage", "Installed-Size"} {
		if control[k] == "" {
			return fmt.Errorf("control has no %s", k)
		}
	}
	if synopsis, extended, _ := strings.Cut(control["Description"], "\n"); synopsis == "" || extended == "" {
		return errors.New("control's Description needs a synopsis and an extended description")
	}
	if name := fmt.Sprintf("wopr_%s_%s.deb", control["Version"], control["Architecture"]); a.Name != name {
		return fmt.Errorf("named %s; Debian's convention (dpkg-name) is %s", a.Name, name)
	}
	var installed []string
	for line := range strings.Lines(string(files["md5sums"])) {
		if _, path, ok := strings.Cut(strings.TrimSpace(line), "  "); ok {
			installed = append(installed, path)
		}
	}
	for _, f := range debFiles {
		if !slices.Contains(installed, f) {
			return fmt.Errorf("does not install /%s (installs %v)", f, installed)
		}
	}
	return nil
}

// readAr reads the members of an ar archive, the container of a .deb.
func readAr(path string) (map[string][]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	const magic = "!<arch>\n"
	if !bytes.HasPrefix(data, []byte(magic)) {
		return nil, errors.New("not an ar archive")
	}
	members := map[string][]byte{}
	for rest := data[len(magic):]; len(rest) > 0; {
		if len(rest) < 60 {
			return nil, errors.New("truncated ar header")
		}
		h := rest[:60]
		var size int
		if _, err := fmt.Sscanf(strings.TrimSpace(string(h[48:58])), "%d", &size); err != nil || size < 0 || 60+size > len(rest) {
			return nil, fmt.Errorf("bad ar member size %q", h[48:58])
		}
		name := strings.TrimSuffix(strings.TrimSpace(string(h[:16])), "/")
		members[name] = rest[60 : 60+size]
		next := min(60+size+size%2, len(rest)) // members are padded to an even length
		rest = rest[next:]
	}
	return members, nil
}

// untarGz returns the regular files in a gzipped tar archive by name, without a leading "./".
func untarGz(data []byte) (map[string][]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)
	files := map[string][]byte{}
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return files, nil
		}
		if err != nil {
			return nil, err
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		body, err := io.ReadAll(tr)
		if err != nil {
			return nil, err
		}
		files[strings.TrimPrefix(h.Name, "./")] = body
	}
}

// parseControl reads a Debian control paragraph. A continuation line (one that starts with a
// space) adds a line to the field before it.
func parseControl(s string) map[string]string {
	fields := map[string]string{}
	last := ""
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, " ") && last != "" {
			fields[last] += "\n" + strings.TrimPrefix(line, " ")
			continue
		}
		if k, v, ok := strings.Cut(line, ":"); ok {
			last = k
			fields[k] = strings.TrimSpace(v)
		}
	}
	return fields
}

// RPM header tags (rpm's include/rpm/rpmtag.h).
const (
	tagName          = 1000
	tagVersion       = 1001
	tagRelease       = 1002
	tagSummary       = 1004
	tagLicense       = 1014
	tagArch          = 1022
	tagFileFlags     = 1037
	tagChangelogName = 1081
	tagDirIndexes    = 1116
	tagBaseNames     = 1117
	tagDirNames      = 1118
)

// checkRPM reads an .rpm's header and checks its name, its file name, the files it installs and
// how it marks them, and that its changelog starts with this version.
func checkRPM(a artifact) error {
	data, err := os.ReadFile(a.Path)
	if err != nil {
		return err
	}
	h, err := readRPMHeader(data)
	if err != nil {
		return err
	}
	name, version, release, arch := h.str(tagName), h.str(tagVersion), h.str(tagRelease), h.str(tagArch)
	if name != "wopr" || arch != rpmArch[a.Goarch] {
		return fmt.Errorf("is %s for %s, want wopr for %s", name, arch, rpmArch[a.Goarch])
	}
	if h.str(tagSummary) == "" || h.str(tagLicense) == "" {
		return errors.New("has no Summary or no License")
	}
	if want := fmt.Sprintf("%s-%s-%s.%s.rpm", name, version, release, arch); a.Name != want {
		return fmt.Errorf("named %s; RPM's convention is %s", a.Name, want)
	}
	dirs, bases, idx, flags := h.strs(tagDirNames), h.strs(tagBaseNames), h.ints(tagDirIndexes), h.ints(tagFileFlags)
	if len(idx) != len(bases) || len(flags) != len(bases) {
		return errors.New("the header's file lists do not line up")
	}
	installed := map[string]uint32{}
	for i, b := range bases {
		if int(idx[i]) >= len(dirs) {
			return errors.New("a file's directory index is out of range")
		}
		installed[dirs[idx[i]]+b] = flags[i]
	}
	for f, want := range rpmFiles {
		got, ok := installed[f]
		switch {
		case !ok:
			return fmt.Errorf("does not install %s (installs %v)", f, slices.Sorted(maps.Keys(installed)))
		case got&want != want:
			return fmt.Errorf("%s has flags %#x, want %#x set (%%doc %#x, %%license %#x)", f, got, want, rpmDoc, rpmLicense)
		}
	}
	changes := h.strs(tagChangelogName)
	if len(changes) == 0 || !strings.HasSuffix(changes[0], " - "+version+"-"+release) {
		return fmt.Errorf("the changelog's newest entry is %q, want one for %s-%s", changes, version, release)
	}
	return nil
}

// rpmHeader is an RPM header's index and data store.
type rpmHeader struct {
	entries map[int32]rpmEntry
	store   []byte
}

type rpmEntry struct {
	typ, offset, count int32
}

// RPM header data types.
const (
	rpmInt32       = 4
	rpmString      = 6
	rpmStringArray = 8
	rpmI18NString  = 9
)

// readRPMHeader skips an .rpm's lead and signature header and reads its main header.
func readRPMHeader(data []byte) (*rpmHeader, error) {
	const leadSize = 96
	if len(data) < leadSize || !bytes.Equal(data[:4], []byte{0xed, 0xab, 0xee, 0xdb}) {
		return nil, errors.New("not an rpm")
	}
	sig, n, err := parseRPMHeader(data[leadSize:])
	if err != nil || sig == nil {
		return nil, fmt.Errorf("signature header: %w", err)
	}
	off := leadSize + (n+7)/8*8 // the signature header is padded to 8 bytes
	if off > len(data) {
		return nil, errors.New("truncated rpm")
	}
	h, _, err := parseRPMHeader(data[off:])
	if err != nil {
		return nil, fmt.Errorf("header: %w", err)
	}
	return h, nil
}

// parseRPMHeader parses one header structure and returns it with its length in bytes.
func parseRPMHeader(b []byte) (*rpmHeader, int, error) {
	if len(b) < 16 || !bytes.Equal(b[:4], []byte{0x8e, 0xad, 0xe8, 0x01}) {
		return nil, 0, errors.New("bad header magic")
	}
	nindex, hsize := int(binary.BigEndian.Uint32(b[8:])), int(binary.BigEndian.Uint32(b[12:]))
	end := 16 + 16*nindex + hsize
	if nindex < 0 || hsize < 0 || end > len(b) {
		return nil, 0, errors.New("truncated header")
	}
	h := &rpmHeader{entries: map[int32]rpmEntry{}, store: b[16+16*nindex : end]}
	for i := range nindex {
		e := b[16+16*i:]
		h.entries[int32(binary.BigEndian.Uint32(e))] = rpmEntry{
			typ:    int32(binary.BigEndian.Uint32(e[4:])),
			offset: int32(binary.BigEndian.Uint32(e[8:])),
			count:  int32(binary.BigEndian.Uint32(e[12:])),
		}
	}
	return h, end, nil
}

// strs returns a string, string array or I18N string entry as strings; nil if it is missing.
func (h *rpmHeader) strs(tag int32) []string {
	e, ok := h.entries[tag]
	if !ok || (e.typ != rpmString && e.typ != rpmStringArray && e.typ != rpmI18NString) ||
		e.offset < 0 || int(e.offset) > len(h.store) {
		return nil
	}
	var out []string
	rest := h.store[e.offset:]
	for range e.count {
		s, after, ok := bytes.Cut(rest, []byte{0})
		if !ok {
			return nil
		}
		out = append(out, string(s))
		rest = after
	}
	return out
}

func (h *rpmHeader) str(tag int32) string {
	if s := h.strs(tag); len(s) > 0 {
		return s[0]
	}
	return ""
}

// ints returns an int32 array entry; nil if it is missing.
func (h *rpmHeader) ints(tag int32) []uint32 {
	e, ok := h.entries[tag]
	if !ok || e.typ != rpmInt32 || e.offset < 0 || e.count < 0 || int(e.offset)+4*int(e.count) > len(h.store) {
		return nil
	}
	out := make([]uint32, e.count)
	for i := range out {
		out[i] = binary.BigEndian.Uint32(h.store[int(e.offset)+4*i:])
	}
	return out
}
