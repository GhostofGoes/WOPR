package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
)

// repoRoot is the repository, from this package's directory.
const repoRoot = "../../.."

// fixture makes a tree holding the repository's icon sources (all but those dropped) and
// returns its root.
func fixture(t *testing.T, drop ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, src := range []string{srcMaster, srcSmall, srcFull} {
		if slices.Contains(drop, src) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(repoRoot, src))
		if err != nil {
			t.Fatal(err)
		}
		writeFile(t, root, src, data)
	}
	return root
}

func writeFile(t *testing.T, root, path string, data []byte) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// generated is every output from the repository's sources, drawn once for all tests. They
// use the fastest PNG compression: the tests compare pixels, not bytes.
var generated = sync.OnceValues(func() ([]file, error) {
	return generate(repoRoot, png.BestSpeed)
})

func generatedFiles(t *testing.T) []file {
	t.Helper()
	files, err := generated()
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func fileData(t *testing.T, files []file, path string) []byte {
	t.Helper()
	for _, f := range files {
		if f.path == path {
			return f.data
		}
	}
	t.Fatalf("%s was not generated", path)
	return nil
}

// TestCommittedIconsAreCurrent fails when the committed icons differ from what the sources
// give: run go run ./internal/tools/icons after changing a source.
func TestCommittedIconsAreCurrent(t *testing.T) {
	files := generatedFiles(t)
	for _, s := range compareAll(repoRoot, files) {
		t.Errorf("%s: %s; run: go run ./internal/tools/icons", s.path, s.why)
	}
	for _, f := range files {
		info, err := os.Stat(filepath.Join(repoRoot, f.path))
		if err == nil && info.Size() > maxFileSize {
			t.Errorf("%s is %d bytes, over the %d limit", f.path, info.Size(), maxFileSize)
		}
	}
}

// TestOutputs checks that every file the packages and the site expect is generated, at the
// right sizes.
func TestOutputs(t *testing.T) {
	files := generatedFiles(t)
	var paths []string
	for _, f := range files {
		paths = append(paths, f.path)
	}
	want := []string{
		"packaging/icons/wopr.ico",
		"packaging/icons/wopr.icns",
		"packaging/icons/hicolor/scalable/apps/io.github.ghostofgoes.wopr.svg",
		"packaging/icons/hicolor/48x48/apps/io.github.ghostofgoes.wopr.png",
		"packaging/icons/hicolor/256x256/apps/io.github.ghostofgoes.wopr.png",
		"packaging/icons/msix/StoreLogo.scale-100.png",
		"packaging/icons/msix/Square44x44Logo.scale-100.png",
		"packaging/icons/msix/Square44x44Logo.targetsize-24_altform-unplated.png",
		"packaging/icons/msix/Square44x44Logo.targetsize-256_altform-lightunplated.png",
		"packaging/icons/msix/Square150x150Logo.scale-100.png",
		"site/static/favicon.svg",
		"site/static/favicon.ico",
		"site/static/apple-touch-icon.png",
	}
	for _, w := range want {
		if !slices.Contains(paths, w) {
			t.Errorf("%s is not generated", w)
		}
	}
	var hicolor, msix int
	for _, p := range paths {
		switch {
		case strings.HasPrefix(p, "packaging/icons/hicolor/") && strings.HasSuffix(p, ".png"):
			hicolor++
		case strings.HasPrefix(p, "packaging/icons/msix/"):
			msix++
		}
	}
	if hicolor != 9 || msix != 3*5+14*3 {
		t.Errorf("%d hicolor PNGs and %d MSIX assets, want 9 and 57", hicolor, msix)
	}

	// Every PNG is square and as big as its name says.
	sizes := map[string]int{
		"packaging/icons/msix/StoreLogo.scale-125.png":         63,
		"packaging/icons/msix/StoreLogo.scale-400.png":         200,
		"packaging/icons/msix/Square44x44Logo.scale-150.png":   66,
		"packaging/icons/msix/Square150x150Logo.scale-125.png": 188,
		"packaging/icons/msix/Square150x150Logo.scale-400.png": 600,
		"site/static/apple-touch-icon.png":                     180,
	}
	for _, o := range outputs() {
		if o.format != fmtPNG {
			continue
		}
		cfg, err := png.DecodeConfig(bytes.NewReader(fileData(t, files, o.path)))
		if err != nil {
			t.Fatalf("%s: %v", o.path, err)
		}
		want := o.sizes[0]
		if n, ok := sizes[o.path]; ok && n != want {
			t.Errorf("%s is listed at %d px, want %d", o.path, want, n)
		}
		// The size in a hicolor directory's or a target size's name.
		var dim int
		named := filepath.Base(filepath.Dir(filepath.Dir(o.path)))
		if _, after, ok := strings.Cut(o.path, "targetsize-"); ok {
			named = after
		}
		if _, err := fmt.Sscanf(named, "%d", &dim); err == nil && dim != want {
			t.Errorf("%s is listed at %d px", o.path, want)
		}
		if cfg.Width != want || cfg.Height != want {
			t.Errorf("%s is %dx%d, want %dx%d", o.path, cfg.Width, cfg.Height, want, want)
		}
	}
	apple, err := png.Decode(bytes.NewReader(fileData(t, files, "site/static/apple-touch-icon.png")))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, a := pixel(apple, 0, 0); a != 255 {
		t.Error("apple-touch-icon.png is not opaque at its corner")
	}
	hi, err := png.Decode(bytes.NewReader(fileData(t, files, "packaging/icons/hicolor/256x256/apps/io.github.ghostofgoes.wopr.png")))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, a := pixel(hi, 0, 0); a != 0 {
		t.Error("the hicolor icon's corner is not transparent; the master needs its margin")
	}

	// The scalable icons are the sources, minified.
	for path, src := range map[string]string{
		"packaging/icons/hicolor/scalable/apps/io.github.ghostofgoes.wopr.svg": srcMaster,
		"site/static/favicon.svg": srcSmall,
	} {
		got := string(fileData(t, files, path))
		if strings.Contains(got, "<!--") || strings.Contains(got, "\n  ") || !strings.HasPrefix(got, `<svg xmlns="http://www.w3.org/2000/svg"`) {
			t.Errorf("%s is not minified:\n%.200s", path, got)
		}
		data, err := os.ReadFile(filepath.Join(repoRoot, src))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := parseSVG(path, []byte(got)); err != nil {
			t.Errorf("%s does not parse again: %v", path, err)
		}
		d, err := parseSVG(src, data)
		if err != nil {
			t.Fatal(err)
		}
		if got != string(minify(d.root)) {
			t.Errorf("%s is not %s minified", path, src)
		}
	}
}

func pixel(img image.Image, x, y int) (r, g, a uint8) {
	cr, cg, _, ca := img.At(x, y).RGBA()
	return uint8(cr >> 8), uint8(cg >> 8), uint8(ca >> 8)
}

// TestICOLayout reads wopr.ico byte by byte, independently of decodeICO.
func TestICOLayout(t *testing.T) {
	files := generatedFiles(t)
	data := fileData(t, files, "packaging/icons/wopr.ico")
	le := binary.LittleEndian
	if le.Uint16(data) != 0 || le.Uint16(data[2:]) != 1 {
		t.Fatalf("header % x is not an icon's", data[:4])
	}
	want := []int{16, 20, 24, 32, 40, 48, 64, 96, 128, 256}
	if n := int(le.Uint16(data[4:])); n != len(want) {
		t.Fatalf("%d entries, want %d", n, len(want))
	}
	next := 6 + 16*len(want)
	for i, n := range want {
		e := data[6+16*i : 6+16*(i+1)]
		w, h := int(e[0]), int(e[1])
		if n == 256 && (w != 0 || h != 0) || n < 256 && (w != n || h != n) {
			t.Errorf("entry %d is %dx%d, want %d", i, w, h, n)
		}
		if e[2] != 0 || e[3] != 0 || le.Uint16(e[4:]) != 1 || le.Uint16(e[6:]) != 32 {
			t.Errorf("entry %d: colours %d, reserved %d, planes %d, bits %d", i, e[2], e[3], le.Uint16(e[4:]), le.Uint16(e[6:]))
		}
		size, offset := int(le.Uint32(e[8:])), int(le.Uint32(e[12:]))
		if offset != next {
			t.Errorf("entry %d starts at %d, want %d (right after the one before)", i, offset, next)
		}
		next = offset + size
		pic := data[offset : offset+size]
		if n == 256 {
			img, err := png.Decode(bytes.NewReader(pic))
			if err != nil || img.Bounds().Dx() != 256 || img.Bounds().Dy() != 256 {
				t.Errorf("the 256 px entry is not a 256 px PNG: %v", err)
			}
			continue
		}
		maskStride := (n + 31) / 32 * 4
		if size != 40+4*n*n+maskStride*n {
			t.Errorf("the %d px bitmap is %d bytes, want %d", n, size, 40+4*n*n+maskStride*n)
			continue
		}
		hdr := []uint32{le.Uint32(pic), le.Uint32(pic[4:]), le.Uint32(pic[8:]), uint32(le.Uint16(pic[12:])), uint32(le.Uint16(pic[14:])), le.Uint32(pic[16:])}
		if !slices.Equal(hdr, []uint32{40, uint32(n), uint32(2 * n), 1, 32, 0}) {
			t.Errorf("the %d px BITMAPINFOHEADER starts %v", n, hdr)
		}
		// Compare with the hicolor-sized picture where there is one: rows run bottom-up in BGRA,
		// and the mask marks the transparent corner.
		img := render(mustDoc(t, n), n)
		x, y := n/2, n/4
		p := pic[40+4*((n-1-y)*n+x):]
		q := img.Pix[img.PixOffset(x, y):]
		if !bytes.Equal([]byte{p[2], p[1], p[0], p[3]}, q[:4]) {
			t.Errorf("the %d px bitmap's pixel (%d, %d) is % x, want % x", n, x, y, p[:4], q[:4])
		}
		mask := pic[40+4*n*n:]
		corner := mask[(n-1)*maskStride] & 0x80 // top-left pixel, last row stored
		centre := mask[(n-1-n/2)*maskStride+n/2/8] & (0x80 >> (n / 2 % 8))
		if corner == 0 || centre != 0 {
			t.Errorf("the %d px mask: corner bit %#x (want set), centre bit %#x (want clear)", n, corner, centre)
		}
	}
	if next != len(data) {
		t.Errorf("the entries end at %d, the file at %d", next, len(data))
	}
}

// mustDoc parses the source the generator uses at size n for the transparent icon.
func mustDoc(t *testing.T, n int) *doc {
	t.Helper()
	src := srcMaster
	if n <= smallMax {
		src = srcSmall
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, src))
	if err != nil {
		t.Fatal(err)
	}
	d, err := parseSVG(src, data)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// TestICNSLayout reads wopr.icns byte by byte, independently of decodeICNS.
func TestICNSLayout(t *testing.T) {
	data := fileData(t, generatedFiles(t), "packaging/icons/wopr.icns")
	be := binary.BigEndian
	if string(data[:4]) != "icns" || int(be.Uint32(data[4:])) != len(data) {
		t.Fatalf("header % x, length %d", data[:8], len(data))
	}
	want := []struct {
		typ  string
		size int
	}{
		{"is32", 16},
		{"s8mk", 16},
		{"il32", 32},
		{"l8mk", 32},
		{"ic11", 32},
		{"ic12", 64},
		{"ic07", 128},
		{"ic13", 256},
		{"ic08", 256},
		{"ic14", 512},
		{"ic09", 512},
		{"ic10", 1024},
	}
	i := 8
	for _, w := range want {
		if i+8 > len(data) {
			t.Fatalf("the file ends before %s", w.typ)
		}
		typ, n := string(data[i:i+4]), int(be.Uint32(data[i+4:]))
		if typ != w.typ {
			t.Fatalf("element %q, want %q", typ, w.typ)
		}
		body := data[i+8 : i+n]
		switch typ {
		case "is32", "il32":
			// Three channels of packBits data, each the picture's size, then one spare zero.
			total, j := 0, 0
			for total < 3*w.size*w.size {
				h := int(body[j])
				if h < 128 {
					total += h + 1
					j += h + 2
				} else {
					total += h - 125
					j += 2
				}
			}
			if total != 3*w.size*w.size || len(body) != j+1 || body[j] != 0 {
				t.Errorf("%s holds %d values in %d of %d bytes, want %d values and one zero byte", typ, total, j, len(body), 3*w.size*w.size)
			}
		case "s8mk", "l8mk":
			if len(body) != w.size*w.size {
				t.Errorf("%s is %d bytes, want %d", typ, len(body), w.size*w.size)
			}
		default:
			cfg, err := png.DecodeConfig(bytes.NewReader(body))
			if err != nil || cfg.Width != w.size || cfg.Height != w.size {
				t.Errorf("%s is not a %d px PNG (%v, %dx%d)", typ, w.size, err, cfg.Width, cfg.Height)
			}
		}
		i += n
	}
	if i != len(data) {
		t.Errorf("%d bytes after the last element", len(data)-i)
	}
}

// TestCheck runs the tool's check and write modes on a tree of generated files.
func TestCheck(t *testing.T) {
	files := generatedFiles(t)
	root := t.TempDir()
	check := func() (int, string) {
		var stderr bytes.Buffer
		code := apply(root, files, true, io.Discard, &stderr)
		return code, stderr.String()
	}
	if code, out := check(); code != 1 || !strings.Contains(out, "packaging/icons/wopr.ico: missing") {
		t.Fatalf("check on an empty tree: %d\n%s", code, out)
	}
	if code := apply(root, files, false, io.Discard, io.Discard); code != 0 {
		t.Fatalf("write: %d", code)
	}
	if code, out := check(); code != 0 {
		t.Fatalf("check after writing: %d\n%s", code, out)
	}

	// Change one pixel of a PNG, then one of a bitmap inside the .ico.
	path := "packaging/icons/hicolor/64x64/apps/io.github.ghostofgoes.wopr.png"
	orig := fileData(t, files, path)
	for _, delta := range []int{maxDiff, maxDiff + 1} {
		img, err := png.Decode(bytes.NewReader(orig))
		if err != nil {
			t.Fatal(err)
		}
		n := img.(*image.NRGBA)
		i := n.PixOffset(32, 32)
		n.Pix[i] = byte(int(n.Pix[i]) + delta*sign(n.Pix[i]))
		data, err := encodePNG(n, png.BestSpeed)
		if err != nil {
			t.Fatal(err)
		}
		writeFile(t, root, path, data)
		code, out := check()
		if delta <= maxDiff && code != 0 {
			t.Errorf("a change of %d failed the check:\n%s", delta, out)
		}
		if delta > maxDiff && (code != 1 || !strings.Contains(out, path+": pixel (32, 32) differs by 3")) {
			t.Errorf("a change of %d: %d\n%s", delta, code, out)
		}
	}
	writeFile(t, root, path, orig)

	ico := slices.Clone(fileData(t, files, "packaging/icons/wopr.ico"))
	off := int(binary.LittleEndian.Uint32(ico[6+12:])) // the 16 px bitmap
	ico[off+40+4*(8*16+8)+1] ^= 0x80                   // green of pixel (8, 7)
	writeFile(t, root, "packaging/icons/wopr.ico", ico)
	if code, out := check(); code != 1 || !strings.Contains(out, "wopr.ico: the 16 px picture: pixel (8, 7)") {
		t.Errorf("a changed .ico passed: %d\n%s", code, out)
	}

	// A missing file, and one the tool does not write.
	if err := os.Remove(filepath.Join(root, "packaging/icons/wopr.icns")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "packaging/icons/msix/Old.png", orig)
	code, out := check()
	for _, w := range []string{"wopr.icns: missing", "msix/Old.png: not generated", "wopr.ico"} {
		if code != 1 || !strings.Contains(out, w) {
			t.Errorf("check: %d, want it to report %q:\n%s", code, w, out)
		}
	}
	var stdout bytes.Buffer
	if code := apply(root, files, false, &stdout, io.Discard); code != 0 {
		t.Fatalf("write: %d", code)
	}
	if want := "removed packaging/icons/msix/Old.png\nwrote packaging/icons/wopr.icns\nwrote packaging/icons/wopr.ico\n"; stdout.String() != want {
		t.Errorf("write said:\n%s\nwant:\n%s", stdout.String(), want)
	}
	if code, out := check(); code != 0 {
		t.Errorf("check after rewriting: %d\n%s", code, out)
	}
}

func sign(v byte) int {
	if v > 127 {
		return -1
	}
	return 1
}

// TestRun runs the command itself on a fixture, as a contributor would: it writes the icons,
// then checks them. It compresses at the tool's own level, which is slow.
func TestRun(t *testing.T) {
	if testing.Short() || raceDetector {
		t.Skip("compressing every icon at the tool's level is slow, and much slower under -race")
	}
	root := fixture(t)
	var stdout, stderr bytes.Buffer
	if code := run(root, false, &stdout, &stderr); code != 0 {
		t.Fatalf("write: %d\n%s", code, stderr.String())
	}
	if code := run(root, true, &stdout, &stderr); code != 0 {
		t.Errorf("-check after writing: %d\n%s", code, stderr.String())
	}
	writeFile(t, root, srcFull, []byte("<svg/>"))
	if code := run(root, true, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "icons: "+srcFull) {
		t.Errorf("-check with a broken source: %d, want 2\n%s", code, stderr.String())
	}
}

// TestDeterministic generates twice: the parallel work must not change a byte.
func TestDeterministic(t *testing.T) {
	a := generatedFiles(t)
	b, err := generate(repoRoot, png.BestSpeed)
	if err != nil {
		t.Fatal(err)
	}
	for i := range a {
		if a[i].path != b[i].path || !bytes.Equal(a[i].data, b[i].data) {
			t.Errorf("%s differs between two runs", a[i].path)
		}
	}
}

// TestSources covers the optional small art and the full-bleed art's opacity.
func TestSources(t *testing.T) {
	// Without wopr-small.svg everything is drawn from wopr.svg.
	root := fixture(t, srcSmall)
	files, err := generate(root, png.BestSpeed)
	if err != nil {
		t.Fatal(err)
	}
	master := mustDoc(t, 64)
	if got := string(fileData(t, files, "site/static/favicon.svg")); got != string(minify(master.root)) {
		t.Error("without wopr-small.svg, favicon.svg is not wopr.svg minified")
	}
	small, err := png.Decode(bytes.NewReader(fileData(t, files, "packaging/icons/hicolor/32x32/apps/"+appID+".png")))
	if err != nil {
		t.Fatal(err)
	}
	if err := samePicture(small, render(master, 32)); err != nil {
		t.Errorf("without wopr-small.svg, the 32 px icon is not wopr.svg drawn at 32 px: %v", err)
	}
	// With it, 32 px and below come from it and 48 px from the master.
	files = generatedFiles(t)
	for n, d := range map[int]*doc{32: mustDoc(t, 32), 48: master} {
		img, err := png.Decode(bytes.NewReader(fileData(t, files, fmt.Sprintf("packaging/icons/hicolor/%dx%d/apps/%s.png", n, n, appID))))
		if err != nil {
			t.Fatal(err)
		}
		if err := samePicture(img, render(d, n)); err != nil {
			t.Errorf("the %d px icon is drawn from the wrong source: %v", n, err)
		}
	}

	// A full-bleed source with a transparent corner is refused.
	root = fixture(t)
	writeFile(t, root, srcFull, []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><rect x="1" width="9" height="10"/></svg>`))
	if _, err := generate(root, png.BestSpeed); err == nil || !strings.Contains(err.Error(), "wopr-full.svg must be opaque") {
		t.Errorf("a transparent wopr-full.svg: %v", err)
	}
	// A broken source names itself and the line.
	root = fixture(t)
	writeFile(t, root, srcMaster, []byte("<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 10 10\">\n<text>WOPR</text></svg>"))
	if _, err := generate(root, png.BestSpeed); err == nil || !strings.Contains(err.Error(), "packaging/icons/src/wopr.svg:2: <text> is not supported") {
		t.Errorf("a source with text: %v", err)
	}
}

func TestPackBits(t *testing.T) {
	var src []byte
	for i := range 300 {
		src = append(src, byte(i*7)) // literals, more than 128 in a row
	}
	src = append(src, bytes.Repeat([]byte{9}, 300)...) // a run longer than 130
	src = append(src, 1, 1, 2, 2, 2, 3)                // short runs among literals
	packed := packBits(src)
	got, n, err := unpackBits(packed, len(src))
	if err != nil || n != len(packed) || !bytes.Equal(got, src) {
		t.Errorf("round trip: %v, read %d of %d bytes, equal %t", err, n, len(packed), bytes.Equal(got, src))
	}
	if len(packed) > 300+3+6+7 { // 300 literals and their 3 headers, 3 runs, the tail
		t.Errorf("%d bytes packed to %d: runs are not compressed", len(src), len(packed))
	}
}

// TestDecoders reads back what the encoders write, including the masks' alpha.
func TestDecoders(t *testing.T) {
	files := generatedFiles(t)
	ico, err := decodeICO(fileData(t, files, "packaging/icons/wopr.ico"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ico {
		if e.img.Bounds().Dx() != e.size || e.isPNG != (e.size == 256) {
			t.Errorf("ico entry %d px: %v, PNG %t", e.size, e.img.Bounds(), e.isPNG)
		}
	}
	icns, err := decodeICNS(fileData(t, files, "packaging/icons/wopr.icns"))
	if err != nil {
		t.Fatal(err)
	}
	full, err := os.ReadFile(filepath.Join(repoRoot, srcFull))
	if err != nil {
		t.Fatal(err)
	}
	d, err := parseSVG(srcFull, full)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range icns {
		if err := samePicture(e.img, render(d, e.img.Bounds().Dx())); err != nil && !strings.HasSuffix(e.typ, "mk") {
			t.Errorf("%s: %v", e.typ, err)
		}
	}
	if _, err := decodeICNS([]byte("icns\x00\x00\x00\x09x")); err == nil {
		t.Error("a truncated .icns decodes")
	}
	if _, err := decodeICO([]byte{0, 0, 1, 0, 1, 0}); err == nil {
		t.Error("a truncated .ico decodes")
	}
}
