package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const goodControl = `Package: wopr
Version: 1.0.0-1
Section: games
Priority: optional
Architecture: amd64
Maintainer: Someone <someone@example.invalid>
Installed-Size: 6000
Homepage: https://example.invalid/wopr
Description: a synopsis
 The extended description.
`

// makeDeb writes a .deb with this control file and these installed files (md5sums), and data
// that stage never opens.
func makeDeb(t *testing.T, dir, name, control string, files []string, omit string) artifact {
	t.Helper()
	var ctl bytes.Buffer
	gz := gzip.NewWriter(&ctl)
	tw := tar.NewWriter(gz)
	var sums strings.Builder
	for _, f := range files {
		sums.WriteString("d41d8cd98f00b204e9800998ecf8427e  " + f + "\n")
	}
	for _, f := range []struct{ name, body string }{{"./control", control}, {"./md5sums", sums.String()}} {
		if err := tw.WriteHeader(&tar.Header{Name: f.name, Mode: 0o644, Size: int64(len(f.body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(f.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	var ar bytes.Buffer
	ar.WriteString("!<arch>\n")
	for _, m := range []struct {
		name string
		body []byte
	}{{"debian-binary", []byte("2.0\n")}, {"control.tar.gz", ctl.Bytes()}, {"data.tar.xz", []byte("odd")}} {
		if m.name == omit {
			continue
		}
		fmt.Fprintf(&ar, "%-16s%-12d%-6d%-6d%-8o%-10d`\n", m.name, 0, 0, 0, 0o644, len(m.body))
		ar.Write(m.body)
		if len(m.body)%2 == 1 {
			ar.WriteByte('\n')
		}
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, ar.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	a := artifact{Name: name, Path: p, Goos: "linux", Goarch: "amd64", Type: linuxPackage}
	a.Extra.Format = "deb"
	return a
}

// rpmTag is one entry of an RPM header: a string, a string array or an int32 array.
type rpmTag struct {
	tag, typ int32
	val      any
}

func rpmHeaderBytes(tags []rpmTag) []byte {
	var index, store bytes.Buffer
	for _, tv := range tags {
		if tv.typ == rpmInt32 {
			for store.Len()%4 != 0 {
				store.WriteByte(0)
			}
		}
		off, count := store.Len(), 0
		switch v := tv.val.(type) {
		case string:
			store.WriteString(v + "\x00")
			count = 1
		case []string:
			for _, s := range v {
				store.WriteString(s + "\x00")
			}
			count = len(v)
		case []uint32:
			for _, n := range v {
				_ = binary.Write(&store, binary.BigEndian, n)
			}
			count = len(v)
		}
		_ = binary.Write(&index, binary.BigEndian, [4]int32{tv.tag, tv.typ, int32(off), int32(count)})
	}
	var b bytes.Buffer
	b.Write([]byte{0x8e, 0xad, 0xe8, 0x01, 0, 0, 0, 0})
	_ = binary.Write(&b, binary.BigEndian, [2]uint32{uint32(len(tags)), uint32(store.Len())})
	b.Write(index.Bytes())
	b.Write(store.Bytes())
	return b.Bytes()
}

// makeRPM writes an .rpm whose header lists files with flags, and a changelog.
func makeRPM(t *testing.T, dir, name string, files map[string]uint32, changelog string, edit func([]rpmTag) []rpmTag) artifact {
	t.Helper()
	var dirs, bases []string
	var idx, flags []uint32
	for _, f := range slices.Sorted(maps.Keys(files)) {
		d := path.Dir(f) + "/"
		i := slices.Index(dirs, d)
		if i < 0 {
			dirs = append(dirs, d)
			i = len(dirs) - 1
		}
		bases = append(bases, path.Base(f))
		idx = append(idx, uint32(i))
		flags = append(flags, files[f])
	}
	tags := []rpmTag{
		{tagName, rpmString, "wopr"},
		{tagVersion, rpmString, "1.0.0"},
		{tagRelease, rpmString, "1"},
		{tagSummary, rpmI18NString, []string{"A summary"}},
		{tagLicense, rpmString, "MIT"},
		{tagArch, rpmString, "x86_64"},
		{tagFileFlags, rpmInt32, flags},
		{tagChangelogName, rpmStringArray, []string{changelog, "Someone - 0.9.0-1"}},
		{tagDirIndexes, rpmInt32, idx},
		{tagBaseNames, rpmStringArray, bases},
		{tagDirNames, rpmStringArray, dirs},
	}
	if edit != nil {
		tags = edit(tags)
	}
	lead := make([]byte, 96)
	copy(lead, []byte{0xed, 0xab, 0xee, 0xdb})
	sig := rpmHeaderBytes([]rpmTag{{1000, rpmInt32, []uint32{1}}})
	data := slices.Concat(lead, sig)
	for len(data)%8 != 0 {
		data = append(data, 0)
	}
	data = append(data, rpmHeaderBytes(tags)...)
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	a := artifact{Name: name, Path: p, Goos: "linux", Goarch: "amd64", Type: linuxPackage}
	a.Extra.Format = "rpm"
	return a
}

func TestCheckDeb(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	const name = "wopr_1.0.0-1_amd64.deb"
	if err := checkDeb(makeDeb(t, dir, name, goodControl, debFiles, "")); err != nil {
		t.Fatalf("good .deb rejected: %v", err)
	}
	for _, c := range []struct {
		why, name, control string
		files              []string
		omit, want         string
	}{
		{"for another architecture", name, strings.Replace(goodControl, "amd64", "arm64", 1), debFiles, "", "Architecture"},
		{"in another section", name, strings.Replace(goodControl, "games", "misc", 1), debFiles, "", "Section"},
		{"without a maintainer", name, strings.Replace(goodControl, "Maintainer: Someone <someone@example.invalid>\n", "", 1), debFiles, "", "Maintainer"},
		{"with a one-line description", name, strings.Replace(goodControl, "\n The extended description.", "", 1), debFiles, "", "extended"},
		{"misnamed", "wopr_1.0.0_amd64.deb", goodControl, debFiles, "", "dpkg-name"},
		{"without its manual page", name, goodControl, slices.DeleteFunc(slices.Clone(debFiles), func(f string) bool { return strings.Contains(f, "man6") }), "", "wopr.6.gz"},
		{"with the program in /usr/bin", name, goodControl, append([]string{"usr/bin/wopr"}, debFiles[1:]...), "", "usr/games/wopr"},
		{"without control.tar.gz", name, goodControl, debFiles, "control.tar.gz", "control.tar.gz"},
		{"without data", name, goodControl, debFiles, "data.tar.xz", "data.tar"},
	} {
		sub := t.TempDir()
		if err := checkDeb(makeDeb(t, sub, c.name, c.control, c.files, c.omit)); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("a .deb %s: got %v, want an error about %s", c.why, err, c.want)
		}
	}
	bad := filepath.Join(dir, "not.deb")
	if err := os.WriteFile(bad, []byte("!<arch>\ndebian-binary   0           0     0     644     99        `\n2.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkDeb(artifact{Name: "not.deb", Path: bad, Goarch: "amd64"}); err == nil {
		t.Error("a truncated ar archive accepted")
	}
}

func TestCheckRPM(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	const name = "wopr-1.0.0-1.x86_64.rpm"
	const current = "Someone <someone@example.invalid> - 1.0.0-1"
	if err := checkRPM(makeRPM(t, dir, name, rpmFiles, current, nil)); err != nil {
		t.Fatalf("good .rpm rejected: %v", err)
	}
	without := func(f string) map[string]uint32 {
		m := maps.Clone(rpmFiles)
		delete(m, f)
		return m
	}
	with := func(f string, flags uint32) map[string]uint32 {
		m := maps.Clone(rpmFiles)
		m[f] = flags
		return m
	}
	for _, c := range []struct {
		why, name string
		files     map[string]uint32
		changelog string
		edit      func([]rpmTag) []rpmTag
		want      string
	}{
		{"misnamed", "wopr-1.0.0.x86_64.rpm", rpmFiles, current, nil, "convention"},
		{"without its licence flagged", name, with("/usr/share/licenses/wopr/LICENSE", 0), current, nil, "LICENSE has flags"},
		{"without owning its documentation directory", name, without("/usr/share/doc/wopr"), current, nil, "/usr/share/doc/wopr"},
		{"with a file more", name, with("/usr/share/doc/wopr/extra.txt", rpmDoc), current, nil, ""},
		{"with an old changelog", name, rpmFiles, "Someone - 0.9.0-1", nil, "changelog"},
		{"for another architecture", name, rpmFiles, current, func(tags []rpmTag) []rpmTag {
			tags[5].val = "aarch64"
			return tags
		}, "aarch64"},
		{"without a licence", name, rpmFiles, current, func(tags []rpmTag) []rpmTag {
			return slices.DeleteFunc(tags, func(tv rpmTag) bool { return tv.tag == tagLicense })
		}, "License"},
	} {
		sub := t.TempDir()
		err := checkRPM(makeRPM(t, sub, c.name, c.files, c.changelog, c.edit))
		if c.want == "" {
			if err != nil {
				t.Errorf("an .rpm %s: %v", c.why, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("an .rpm %s: got %v, want an error about %s", c.why, err, c.want)
		}
	}
	for _, junk := range [][]byte{nil, []byte("not an rpm"), append([]byte{0xed, 0xab, 0xee, 0xdb}, make([]byte, 120)...)} {
		p := filepath.Join(dir, "junk.rpm")
		if err := os.WriteFile(p, junk, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := checkRPM(artifact{Name: "junk.rpm", Path: p, Goarch: "amd64"}); err == nil {
			t.Errorf("junk accepted: % x", junk)
		}
	}
}

// Every Linux build needs exactly one .deb and one .rpm.
func TestCheckPackages(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	deb := makeDeb(t, dir, "wopr_1.0.0-1_amd64.deb", goodControl, debFiles, "")
	rpm := makeRPM(t, dir, "wopr-1.0.0-1.x86_64.rpm", rpmFiles, "Someone - 1.0.0-1", nil)
	linux := artifact{Name: "wopr", Goos: "linux", Goarch: "amd64", Type: "Binary"}
	darwin := artifact{Name: "wopr", Goos: "darwin", Goarch: "arm64", Type: "Binary"}
	if err := checkPackages([]artifact{linux, darwin, deb, rpm}); err != nil {
		t.Fatalf("a complete set rejected: %v", err)
	}
	if err := checkPackages([]artifact{linux, darwin, deb}); err == nil || !strings.Contains(err.Error(), ".rpm") {
		t.Errorf("a missing .rpm accepted: %v", err)
	}
	if err := checkPackages([]artifact{linux, deb, deb, rpm}); err == nil || !strings.Contains(err.Error(), "2 .deb") {
		t.Errorf("two .debs for one build accepted: %v", err)
	}
	apk := deb
	apk.Extra.Format = "apk"
	if err := checkPackages([]artifact{linux, deb, rpm, apk}); err == nil || !strings.Contains(err.Error(), "apk") {
		t.Errorf("an unexpected format accepted: %v", err)
	}
}
