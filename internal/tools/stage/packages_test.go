package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
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

// makePkgs writes what pkgdocs writes for the desktop into a new directory: each package's menu
// entry, starting the program from bindir (by format), and the AppStream metadata.
func makePkgs(t *testing.T, bindir map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{appID + ".metainfo.xml": "<component/>\n"}
	for format, bin := range bindir {
		files[filepath.Join(format, appID+".desktop")] = "# A comment.\n[Desktop Entry]\nExec=" + bin + "/wopr\n\n[Desktop Action movie]\nExec=" + bin + "/wopr --movie\n"
	}
	for name, body := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// goodBindirs are where each package puts the program.
var goodBindirs = map[string]string{"deb": "/usr/games", "rpm": "/usr/bin"}

// pkgDigest is the digest a package records for an installed file that comes from pkgdocs, made
// with sum from the file in pkgs; "" for any other file.
func pkgDigest(t *testing.T, pkgs, format, installed string, sum func([]byte) string) string {
	t.Helper()
	src := map[string]string{
		desktopEntry: filepath.Join(pkgs, format, appID+".desktop"),
		metainfo:     filepath.Join(pkgs, appID+".metainfo.xml"),
	}[strings.TrimPrefix(installed, "/")]
	if src == "" {
		return ""
	}
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	return sum(data)
}

func md5Hex(b []byte) string    { s := md5.Sum(b); return hex.EncodeToString(s[:]) }
func sha256Hex(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

// makeDeb writes a .deb with this control file and these installed files (md5sums, each file's
// digest from pkgs when pkgdocs writes it), and data that stage never opens.
func makeDeb(t *testing.T, pkgs, dir, name, control string, files []string, omit string) artifact {
	t.Helper()
	var ctl bytes.Buffer
	gz := gzip.NewWriter(&ctl)
	tw := tar.NewWriter(gz)
	var sums strings.Builder
	for _, f := range files {
		sum := pkgDigest(t, pkgs, "deb", f, md5Hex)
		if sum == "" {
			sum = "d41d8cd98f00b204e9800998ecf8427e"
		}
		sums.WriteString(sum + "  " + f + "\n")
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

// makeRPM writes an .rpm whose header lists files with flags and SHA-256 digests (from pkgs when
// pkgdocs writes the file, as nFPM leaves directories' empty), and a changelog.
func makeRPM(t *testing.T, pkgs, dir, name string, files map[string]uint32, changelog string, edit func([]rpmTag) []rpmTag) artifact {
	t.Helper()
	var dirs, bases, digests []string
	var idx, flags, algo []uint32
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
		digests = append(digests, pkgDigest(t, pkgs, "rpm", f, sha256Hex))
		algo = append(algo, rpmSHA256)
	}
	tags := []rpmTag{
		{tagName, rpmString, "wopr"},
		{tagVersion, rpmString, "1.0.0"},
		{tagRelease, rpmString, "1"},
		{tagSummary, rpmI18NString, []string{"A summary"}},
		{tagLicense, rpmString, "MIT"},
		{tagArch, rpmString, "x86_64"},
		{tagFileDigests, rpmStringArray, digests},
		{tagFileFlags, rpmInt32, flags},
		{tagChangelogName, rpmStringArray, []string{changelog, "Someone - 0.9.0-1"}},
		{tagDirIndexes, rpmInt32, idx},
		{tagBaseNames, rpmStringArray, bases},
		{tagDirNames, rpmStringArray, dirs},
		{tagDigestAlgo, rpmInt32, algo},
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
	dir, pkgs := t.TempDir(), makePkgs(t, goodBindirs)
	const name = "wopr_1.0.0-1_amd64.deb"
	if err := checkDeb(makeDeb(t, pkgs, dir, name, goodControl, debFiles, ""), pkgs); err != nil {
		t.Fatalf("good .deb rejected: %v", err)
	}
	without := func(sub string) []string {
		return slices.DeleteFunc(slices.Clone(debFiles), func(f string) bool { return strings.Contains(f, sub) })
	}
	for _, c := range []struct {
		why, name, control string
		files              []string
		omit, want         string
	}{
		{"without its menu entry", name, goodControl, without("applications"), "", desktopEntry},
		{"without its metadata", name, goodControl, without("metainfo"), "", metainfo},
		{"without an icon size", name, goodControl, without("/512x512/"), "", "512x512"},
		{"without the scalable icon", name, goodControl, without(".svg"), "", "scalable"},
		{"for another architecture", name, strings.Replace(goodControl, "amd64", "arm64", 1), debFiles, "", "Architecture"},
		{"in another section", name, strings.Replace(goodControl, "games", "misc", 1), debFiles, "", "Section"},
		{"without a maintainer", name, strings.Replace(goodControl, "Maintainer: Someone <someone@example.invalid>\n", "", 1), debFiles, "", "Maintainer"},
		{"with a one-line description", name, strings.Replace(goodControl, "\n The extended description.", "", 1), debFiles, "", "extended"},
		{"misnamed", "wopr_1.0.0_amd64.deb", goodControl, debFiles, "", "dpkg-name"},
		{"without its manual page", name, goodControl, without("man6"), "", "wopr.6.gz"},
		{"with the program in /usr/bin", name, goodControl, append([]string{"usr/bin/wopr"}, debFiles[1:]...), "", "usr/games/wopr"},
		{"without control.tar.gz", name, goodControl, debFiles, "control.tar.gz", "control.tar.gz"},
		{"without data", name, goodControl, debFiles, "data.tar.xz", "data.tar"},
	} {
		sub := t.TempDir()
		if err := checkDeb(makeDeb(t, pkgs, sub, c.name, c.control, c.files, c.omit), pkgs); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("a .deb %s: got %v, want an error about %s", c.why, err, c.want)
		}
	}

	// The menu entry must be the one pkgdocs wrote for the .deb, and start /usr/games/wopr.
	if err := checkDeb(makeDeb(t, pkgs, t.TempDir(), name, goodControl, debFiles, ""), makePkgs(t, map[string]string{"deb": "/usr/local/bin"})); err == nil || !strings.Contains(err.Error(), "digest") {
		t.Errorf("a .deb with another menu entry than pkgdocs's: got %v, want an error about its digest", err)
	}
	rpmEntry := makePkgs(t, map[string]string{"deb": "/usr/bin"})
	if err := checkDeb(makeDeb(t, rpmEntry, t.TempDir(), name, goodControl, debFiles, ""), rpmEntry); err == nil || !strings.Contains(err.Error(), "does not start /usr/games/wopr") {
		t.Errorf("a .deb whose menu entry starts /usr/bin/wopr: got %v", err)
	}
	if err := checkDeb(makeDeb(t, pkgs, t.TempDir(), name, goodControl, debFiles, ""), t.TempDir()); err == nil {
		t.Error("a .deb checked without pkgdocs's files accepted")
	}

	bad := filepath.Join(dir, "not.deb")
	if err := os.WriteFile(bad, []byte("!<arch>\ndebian-binary   0           0     0     644     99        `\n2.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkDeb(artifact{Name: "not.deb", Path: bad, Goarch: "amd64"}, pkgs); err == nil {
		t.Error("a truncated ar archive accepted")
	}
}

func TestCheckRPM(t *testing.T) {
	t.Parallel()
	dir, pkgs := t.TempDir(), makePkgs(t, goodBindirs)
	const name = "wopr-1.0.0-1.x86_64.rpm"
	const current = "Someone <someone@example.invalid> - 1.0.0-1"
	if err := checkRPM(makeRPM(t, pkgs, dir, name, rpmFiles, current, nil), pkgs); err != nil {
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
		{"without its menu entry", name, without("/" + desktopEntry), current, nil, desktopEntry},
		{"without an icon size", name, without("/usr/share/icons/hicolor/22x22/apps/" + appID + ".png"), current, nil, "22x22"},
		{"without owning the icons' directories", name, without("/usr/share/icons/hicolor/scalable/apps"), current, nil, "scalable/apps"},
		{"with MD5 file digests", name, rpmFiles, current, func(tags []rpmTag) []rpmTag {
			for i, tv := range tags {
				if tv.tag == tagDigestAlgo {
					tags[i].val = slices.Repeat([]uint32{1}, len(tv.val.([]uint32)))
				}
			}
			return tags
		}, "SHA-256"},
		{"for another architecture", name, rpmFiles, current, func(tags []rpmTag) []rpmTag {
			tags[5].val = "aarch64"
			return tags
		}, "aarch64"},
		{"without a licence", name, rpmFiles, current, func(tags []rpmTag) []rpmTag {
			return slices.DeleteFunc(tags, func(tv rpmTag) bool { return tv.tag == tagLicense })
		}, "License"},
	} {
		sub := t.TempDir()
		err := checkRPM(makeRPM(t, pkgs, sub, c.name, c.files, c.changelog, c.edit), pkgs)
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
		if err := checkRPM(artifact{Name: "junk.rpm", Path: p, Goarch: "amd64"}, pkgs); err == nil {
			t.Errorf("junk accepted: % x", junk)
		}
	}
}

// Every Linux build needs exactly one .deb and one .rpm.
func TestCheckPackages(t *testing.T) {
	t.Parallel()
	dir, pkgs := t.TempDir(), makePkgs(t, goodBindirs)
	deb := makeDeb(t, pkgs, dir, "wopr_1.0.0-1_amd64.deb", goodControl, debFiles, "")
	rpm := makeRPM(t, pkgs, dir, "wopr-1.0.0-1.x86_64.rpm", rpmFiles, "Someone - 1.0.0-1", nil)
	linux := artifact{Name: "wopr", Goos: "linux", Goarch: "amd64", Type: "Binary"}
	darwin := artifact{Name: "wopr", Goos: "darwin", Goarch: "arm64", Type: "Binary"}
	if err := checkPackagesFrom([]artifact{linux, darwin, deb, rpm}, pkgs); err != nil {
		t.Fatalf("a complete set rejected: %v", err)
	}
	if err := checkPackagesFrom([]artifact{linux, darwin, deb}, pkgs); err == nil || !strings.Contains(err.Error(), ".rpm") {
		t.Errorf("a missing .rpm accepted: %v", err)
	}
	if err := checkPackagesFrom([]artifact{linux, deb, deb, rpm}, pkgs); err == nil || !strings.Contains(err.Error(), "2 .deb") {
		t.Errorf("two .debs for one build accepted: %v", err)
	}
	apk := deb
	apk.Extra.Format = "apk"
	if err := checkPackagesFrom([]artifact{linux, deb, rpm, apk}, pkgs); err == nil || !strings.Contains(err.Error(), "apk") {
		t.Errorf("an unexpected format accepted: %v", err)
	}
}

// The packages install the icon in the sizes packaging/icons/hicolor has, the scalable one an SVG
// and the others PNGs, as internal/tools/icons writes them.
func TestHicolorSizes(t *testing.T) {
	t.Parallel()
	icons, err := filepath.Glob(filepath.Join("..", "..", "..", "packaging", "icons", "hicolor", "*", "apps", appID+".*"))
	if err != nil {
		t.Fatal(err)
	}
	var sizes []string
	for _, icon := range icons {
		size := filepath.Base(filepath.Dir(filepath.Dir(icon)))
		if want := map[bool]string{true: ".svg", false: ".png"}[size == "scalable"]; filepath.Ext(icon) != want {
			t.Errorf("%s: want a %s", icon, want)
		}
		sizes = append(sizes, size)
	}
	if !slices.Equal(slices.Sorted(slices.Values(sizes)), slices.Sorted(slices.Values(hicolorSizes))) {
		t.Errorf("packaging/icons/hicolor has the sizes %v; stage expects %v", sizes, hicolorSizes)
	}
}
