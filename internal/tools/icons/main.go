// Command icons draws wopr's application icon from the SVG sources in packaging/icons/src and
// writes every icon file the packages and the docs site use:
//
//   - packaging/icons/wopr.ico: Windows (wopr.exe and the installer)
//   - packaging/icons/wopr.icns: macOS (WOPR.app), from the full-bleed wopr-full.svg
//   - packaging/icons/hicolor/: Linux (the .deb, the .rpm and the snap), PNGs and the SVG
//   - packaging/icons/msix/: the MSIX package's visual assets
//   - site/static/favicon.svg, favicon.ico and apple-touch-icon.png: the docs site
//
// Run it from the repository root:
//
//	go run ./internal/tools/icons            # rewrite what is out of date
//	go run ./internal/tools/icons -check     # fail, listing the files, if any is out of date
//
// It draws with its own small SVG renderer, which accepts only the subset listed in svg.go and
// rejects anything else, so a design that would draw differently here and in a browser fails
// at once. Each size is drawn from the vector art, never by shrinking a bigger picture: from
// wopr-<N>.svg at exactly N px when it exists, else from wopr-small.svg at 32 px and below when
// it exists, else from wopr.svg. The same sources give the same bytes on one machine. Pictures
// are compared by their pixels, allowing a difference of 2 in 255 per channel, because floating
// point differs slightly between CPU architectures (Go fuses multiply-adds on arm64). The .ico
// and .icns files around them are compared byte for byte with what the tool writes from their
// own pixels, keeping their PNG data as it is, which takes no floating point: their
// directories, bitmap headers, masks and packed RGB data must be exactly the tool's. -check
// passes, and a rewrite leaves a file alone, when both hold.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
)

// appID is wopr's app ID, which names its icon in the hicolor theme.
const appID = "io.github.ghostofgoes.wopr"

// The sources: the master, with the platforms' usual transparent margin; simplified art for
// small sizes (optional); a full-bleed, opaque square for macOS, which masks icons itself; and,
// also optional, art drawn for one size, wopr-<N>.svg (srcSized).
const (
	srcDir    = "packaging/icons/src"
	srcMaster = srcDir + "/wopr.svg"
	srcSmall  = srcDir + "/wopr-small.svg"
	srcFull   = srcDir + "/wopr-full.svg"
)

// srcSized is the source drawn for the transparent icon at exactly n×n pixels, in place of
// wopr-small.svg or wopr.svg, so that its edges can land on whole pixels at a size where the
// other sources' fall between them. Nothing else is drawn from it.
func srcSized(n int) string { return fmt.Sprintf("%s/wopr-%d.svg", srcDir, n) }

const (
	smallMax    = 32        // the largest size drawn from wopr-small.svg
	maxFileSize = 512 << 10 // check-added-large-files' limit (prek.toml)
	maxDiff     = 2         // per channel, out of 255, between equal pictures
)

// art picks the source of an output.
type art int

const (
	artIcon    art = iota // wopr-<N>.svg at N px, else wopr-small.svg at 32 px and below, else wopr.svg
	artFull               // wopr-full.svg
	artFavicon            // SVG output: wopr-small.svg if there is one, else wopr.svg
)

type format int

const (
	fmtPNG format = iota
	fmtICO
	fmtICNS
	fmtSVG
)

// output is one generated file.
type output struct {
	path   string
	format format
	art    art
	sizes  []int // a PNG's size, or an ICO's entries
}

// generatedDirs hold only this tool's files: any other file there is stale.
var generatedDirs = []string{"packaging/icons/hicolor", "packaging/icons/msix"}

func outputs() []output {
	out := []output{
		{path: "packaging/icons/wopr.ico", format: fmtICO, sizes: []int{16, 20, 24, 32, 40, 48, 64, 96, 128, 256}},
		{path: "packaging/icons/wopr.icns", format: fmtICNS, art: artFull},
		{path: "packaging/icons/hicolor/scalable/apps/" + appID + ".svg", format: fmtSVG},
	}
	for _, n := range []int{16, 22, 24, 32, 48, 64, 128, 256, 512} {
		out = append(out, output{path: fmt.Sprintf("packaging/icons/hicolor/%dx%d/apps/%s.png", n, n, appID), sizes: []int{n}})
	}
	// MSIX visual assets (Microsoft Learn, "Construct your Windows app's icon"): each logo at
	// five scale factors, and the app list icon at fixed target sizes, plain and in the
	// unplated forms Windows shows on dark and light taskbars without a backplate.
	msix := func(name string, n int) {
		out = append(out, output{path: "packaging/icons/msix/" + name + ".png", sizes: []int{n}})
	}
	for _, logo := range []struct {
		name string
		base int
	}{{"StoreLogo", 50}, {"Square44x44Logo", 44}, {"Square150x150Logo", 150}} {
		for _, scale := range []int{100, 125, 150, 200, 400} {
			msix(fmt.Sprintf("%s.scale-%d", logo.name, scale), (logo.base*scale+50)/100) // 62.5 is 63
		}
	}
	for _, n := range []int{16, 20, 24, 30, 32, 36, 40, 48, 60, 64, 72, 80, 96, 256} {
		for _, form := range []string{"", "_altform-unplated", "_altform-lightunplated"} {
			msix(fmt.Sprintf("Square44x44Logo.targetsize-%d%s", n, form), n)
		}
	}
	return append(out,
		output{path: "site/static/favicon.svg", format: fmtSVG, art: artFavicon},
		output{path: "site/static/favicon.ico", format: fmtICO, sizes: []int{16, 32, 48}},
		output{path: "site/static/apple-touch-icon.png", art: artFull, sizes: []int{180}},
	)
}

// iconSizes are the sizes at which the transparent icon is drawn as pixels: those an
// exact-size source may have.
func iconSizes() []int {
	var sizes []int
	for _, o := range outputs() {
		if o.art == artIcon && o.format != fmtSVG {
			sizes = append(sizes, o.sizes...)
		}
	}
	slices.Sort(sizes)
	return slices.Compact(sizes)
}

// sizedSources lists the exact-size sources under root by their size. Any other SVG file in
// the sources' directory is an error, as is a size the icon is never drawn at: either would be
// a source that nothing draws.
func sizedSources(root string) (map[int]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(srcDir)))
	if err != nil {
		return nil, err
	}
	sizes := iconSizes()
	sized := map[int]string{}
	for _, e := range entries {
		path := srcDir + "/" + e.Name()
		if e.IsDir() || !strings.EqualFold(filepath.Ext(path), ".svg") || slices.Contains([]string{srcMaster, srcSmall, srcFull}, path) {
			continue
		}
		digits, ok := strings.CutPrefix(strings.TrimSuffix(e.Name(), ".svg"), "wopr-")
		n, err := strconv.Atoi(digits)
		if !ok || err != nil || strconv.Itoa(n) != digits {
			return nil, fmt.Errorf("%s is not one of the sources (wopr.svg, wopr-small.svg, wopr-full.svg, or wopr-<N>.svg for the N px icon); rename or remove it", path)
		}
		if !slices.Contains(sizes, n) {
			return nil, fmt.Errorf("%s: no icon is drawn at %d px; the sizes are %s", path, n, joinInts(sizes))
		}
		sized[n] = path
	}
	return sized, nil
}

// joinInts lists numbers as "1, 2, 3".
func joinInts(ns []int) string {
	s := make([]string, len(ns))
	for i, n := range ns {
		s[i] = strconv.Itoa(n)
	}
	return strings.Join(s, ", ")
}

// file is a generated file's contents.
type file struct {
	path   string
	format format
	data   []byte
}

func main() {
	check := flag.Bool("check", false, "fail if a file is out of date instead of rewriting it")
	flag.Parse()
	os.Exit(run(".", *check, os.Stdout, os.Stderr))
}

// run generates the icons under root and checks or writes them; it returns the exit status.
func run(root string, check bool, stdout, stderr io.Writer) int {
	files, err := generate(root, png.BestCompression)
	for _, f := range files {
		if len(f.data) > maxFileSize && err == nil {
			err = fmt.Errorf("%s would be %d KB, over the %d KB limit for committed files; simplify the art",
				f.path, (len(f.data)+1023)/1024, maxFileSize>>10)
		}
	}
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "icons:", err)
		return 2 // could not check: not the same as out of date
	}
	return apply(root, files, check, stdout, stderr)
}

// apply compares the generated files with those under root, then reports the differences
// (check) or writes the files that differ and removes the extra ones.
func apply(root string, files []file, check bool, stdout, stderr io.Writer) int {
	stale := compareAll(root, files)
	if check {
		if len(stale) == 0 {
			return 0
		}
		_, _ = fmt.Fprintln(stderr, "icons: out of date; run: go run ./internal/tools/icons")
		for _, s := range stale {
			_, _ = fmt.Fprintf(stderr, "  %s: %s\n", s.path, s.why)
		}
		return 1
	}
	want := map[string][]byte{}
	for _, f := range files {
		want[f.path] = f.data
	}
	for _, s := range stale {
		var err error
		p := filepath.Join(root, filepath.FromSlash(s.path))
		if data, ok := want[s.path]; ok {
			if err = os.MkdirAll(filepath.Dir(p), 0o755); err == nil {
				err = os.WriteFile(p, data, 0o644)
			}
			_, _ = fmt.Fprintln(stdout, "wrote", s.path)
		} else {
			err = os.Remove(p)
			_, _ = fmt.Fprintln(stdout, "removed", s.path)
		}
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "icons:", err)
			return 1
		}
	}
	return 0
}

// generate draws every output from the sources under root, compressing PNGs at level: the
// tool uses the best, for room under the size limit; tests, which compare pixels, the fastest.
// run checks the size limit.
func generate(root string, level png.CompressionLevel) ([]file, error) {
	read := func(path string, optional bool) (*doc, error) {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if optional && errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		return parseSVG(path, data)
	}
	master, err := read(srcMaster, false)
	if err != nil {
		return nil, err
	}
	small, err := read(srcSmall, true)
	if err != nil {
		return nil, err
	}
	full, err := read(srcFull, false)
	if err != nil {
		return nil, err
	}
	paths, err := sizedSources(root)
	if err != nil {
		return nil, err
	}
	sized := map[int]*doc{}
	for _, n := range slices.Sorted(maps.Keys(paths)) { // the first broken one, every time
		if sized[n], err = read(paths[n], false); err != nil {
			return nil, err
		}
	}
	// The outputs are put together in parallel. Each picture is drawn, and each PNG encoded,
	// once however many outputs share it (the unplated MSIX forms, the 256 px icons), and the
	// result of each depends only on its inputs, so the order of the work does not matter.
	limit := make(chan struct{}, runtime.GOMAXPROCS(0))
	work := func(f func()) {
		limit <- struct{}{}
		defer func() { <-limit }()
		f()
	}
	type key struct {
		d *doc
		n int
	}
	var pictures memo[key, *image.NRGBA]
	pic := func(a art, n int) (*image.NRGBA, error) {
		d := master
		switch {
		case a == artFull:
			d = full
		case sized[n] != nil:
			d = sized[n]
		case n <= smallMax && small != nil:
			d = small
		}
		return pictures.get(key{d, n}, func() (img *image.NRGBA, err error) {
			work(func() { img = render(d, n) })
			if d == full {
				if x, y, ok := transparentPixel(img); ok {
					err = fmt.Errorf("%s must be opaque, but at %d px its pixel (%d, %d) is not", srcFull, n, x, y)
				}
			}
			return img, err
		})
	}
	var pngs memo[*image.NRGBA, []byte]
	encode := func(img *image.NRGBA) ([]byte, error) {
		return pngs.get(img, func() (data []byte, err error) {
			work(func() { data, err = encodePNG(img, level) })
			return data, err
		})
	}
	build := func(o output) ([]byte, error) {
		var err error
		switch o.format {
		case fmtSVG:
			d := master
			if o.art == artFavicon && small != nil {
				d = small
			}
			return minify(d.root), nil
		case fmtPNG:
			img, err := pic(o.art, o.sizes[0])
			if err != nil {
				return nil, err
			}
			return encode(img)
		case fmtICO:
			imgs := make([]*image.NRGBA, len(o.sizes))
			for i, n := range o.sizes {
				if imgs[i], err = pic(o.art, n); err != nil {
					return nil, err
				}
			}
			return encodeICO(imgs, encode)
		default:
			imgs := map[int]*image.NRGBA{}
			for _, t := range icnsTypes {
				if imgs[t.size], err = pic(o.art, t.size); err != nil {
					return nil, err
				}
			}
			return encodeICNS(func(n int) *image.NRGBA { return imgs[n] }, encode)
		}
	}
	outs := outputs()
	files := make([]file, len(outs))
	errs := make([]error, len(outs))
	var wg sync.WaitGroup
	for i, o := range outs {
		wg.Go(func() {
			data, err := build(o)
			files[i], errs[i] = file{path: o.path, format: o.format, data: data}, err
		})
	}
	wg.Wait()
	for i, f := range files {
		if err := errs[i]; err != nil {
			return nil, fmt.Errorf("%s: %w", f.path, err)
		}
	}
	return files, nil
}

// memo computes each key's value once, however many goroutines ask for it.
type memo[K comparable, V any] struct {
	mu sync.Mutex
	m  map[K]*memoEntry[V]
}

type memoEntry[V any] struct {
	once sync.Once
	v    V
	err  error
}

func (m *memo[K, V]) get(k K, f func() (V, error)) (V, error) {
	m.mu.Lock()
	if m.m == nil {
		m.m = map[K]*memoEntry[V]{}
	}
	e, ok := m.m[k]
	if !ok {
		e = &memoEntry[V]{}
		m.m[k] = e
	}
	m.mu.Unlock()
	e.once.Do(func() { e.v, e.err = f() })
	return e.v, e.err
}

func transparentPixel(img *image.NRGBA) (x, y int, found bool) {
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] != 255 {
			p := i / 4
			return p % img.Rect.Dx(), p / img.Rect.Dx(), true
		}
	}
	return 0, 0, false
}

// staleFile is a file that differs from what the sources give, and how.
type staleFile struct {
	path, why string
}

// compareAll lists the files under root that are missing or differ from the generated ones,
// and any other file in the generated directories.
func compareAll(root string, files []file) []staleFile {
	var stale []staleFile
	known := map[string]bool{}
	for _, f := range files {
		known[f.path] = true
		cur, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f.path)))
		if errors.Is(err, fs.ErrNotExist) {
			stale = append(stale, staleFile{f.path, "missing"})
			continue
		}
		if err == nil && !bytes.Equal(cur, f.data) {
			err = same(f.format, cur, f.data)
		}
		if err != nil {
			stale = append(stale, staleFile{f.path, err.Error()})
		}
	}
	for _, dir := range generatedDirs {
		_ = filepath.WalkDir(filepath.Join(root, filepath.FromSlash(dir)), func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil // a missing directory has no extra files
			}
			rel, err := filepath.Rel(root, p)
			if err == nil && !known[filepath.ToSlash(rel)] {
				stale = append(stale, staleFile{filepath.ToSlash(rel), "not generated from the sources; it will be removed"})
			}
			return nil
		})
	}
	slices.SortFunc(stale, func(a, b staleFile) int { return strings.Compare(a.path, b.path) })
	return stale
}

// same reports how a committed file differs from the generated one, or nil if it does not. It
// compares pictures by their pixels; an .ico or .icns around them must also be byte for byte
// what the tool writes from the committed pixels, with the committed PNG data kept as it is.
func same(f format, cur, want []byte) error {
	switch f {
	case fmtSVG:
		if !bytes.Equal(bytes.ReplaceAll(cur, []byte("\r\n"), []byte("\n")), want) {
			return errors.New("differs")
		}
		return nil
	case fmtPNG:
		a, err := png.Decode(bytes.NewReader(cur))
		if err != nil {
			return err
		}
		b, err := png.Decode(bytes.NewReader(want))
		if err != nil {
			return err
		}
		return samePicture(a, b)
	case fmtICO:
		a, err := decodeICO(cur)
		if err != nil {
			return err
		}
		b, err := decodeICO(want)
		if err != nil {
			return err
		}
		if len(a) != len(b) {
			return fmt.Errorf("has %d pictures, want %d", len(a), len(b))
		}
		for i := range a {
			if a[i].size != b[i].size || a[i].isPNG != b[i].isPNG {
				return fmt.Errorf("picture %d is %d px (PNG %t), want %d px (PNG %t)", i, a[i].size, a[i].isPNG, b[i].size, b[i].isPNG)
			}
			if err := samePicture(a[i].img, b[i].img); err != nil {
				return fmt.Errorf("the %d px picture: %w", a[i].size, err)
			}
		}
		canon, err := canonicalICO(a)
		if err != nil {
			return err
		}
		if i := firstDiff(cur, canon); i >= 0 {
			return fmt.Errorf("%s differs from what the tool writes from its pictures (byte %d)", icoPart(canon, i), i)
		}
		return nil
	default:
		a, err := decodeICNS(cur)
		if err != nil {
			return err
		}
		b, err := decodeICNS(want)
		if err != nil {
			return err
		}
		types := func(es []icnsElem) []string {
			var s []string
			for _, e := range es {
				s = append(s, e.typ)
			}
			return s
		}
		if !slices.Equal(types(a), types(b)) {
			return fmt.Errorf("holds %v, want %v", types(a), types(b))
		}
		for i := range a {
			if err := samePicture(a[i].img, b[i].img); err != nil {
				return fmt.Errorf("%s: %w", a[i].typ, err)
			}
		}
		canon, err := canonicalICNS(a)
		if err != nil {
			return err
		}
		for i, e := range a {
			if j := firstDiff(e.raw, canon[i].raw); j >= 0 {
				where := fmt.Sprintf("byte %d", j)
				if len(e.raw) != len(canon[i].raw) {
					where = fmt.Sprintf("%d bytes, want %d", len(e.raw), len(canon[i].raw))
				}
				return fmt.Errorf("%s: its data differs from what the tool writes from its pixels (%s)", e.typ, where)
			}
		}
		if !bytes.Equal(cur, icnsFile(canon)) {
			return errors.New("its headers differ from what the tool writes")
		}
		return nil
	}
}

// firstDiff is the first index at which a and b differ, or -1 if they are equal.
func firstDiff(a, b []byte) int {
	n := min(len(a), len(b))
	for i := range n {
		if a[i] != b[i] {
			return i
		}
	}
	if len(a) == len(b) {
		return -1
	}
	return n
}

// samePicture compares premultiplied 8-bit pixels within maxDiff per channel.
func samePicture(a, b image.Image) error {
	if a.Bounds().Size() != b.Bounds().Size() {
		return fmt.Errorf("is %v, want %v", a.Bounds().Size(), b.Bounds().Size())
	}
	pa, pb := premultiplied(a), premultiplied(b)
	w := a.Bounds().Dx()
	for i := range pa {
		if diff := int(pa[i]) - int(pb[i]); diff > maxDiff || diff < -maxDiff {
			p := i / 4
			return fmt.Errorf("pixel (%d, %d) differs by %d in 255", p%w, p/w, max(diff, -diff))
		}
	}
	return nil
}

// premultiplied returns a picture's pixels as premultiplied 8-bit RGBA, row by row.
func premultiplied(img image.Image) []uint8 {
	r := img.Bounds()
	out := make([]uint8, 0, 4*r.Dx()*r.Dy())
	switch m := img.(type) {
	case *image.NRGBA:
		for y := r.Min.Y; y < r.Max.Y; y++ {
			row := m.Pix[m.PixOffset(r.Min.X, y):m.PixOffset(r.Max.X, y)]
			for i := 0; i < len(row); i += 4 {
				a := uint32(row[i+3])
				for ch := range 3 {
					out = append(out, uint8((uint32(row[i+ch])*a+127)/255))
				}
				out = append(out, row[i+3])
			}
		}
	case *image.RGBA:
		for y := r.Min.Y; y < r.Max.Y; y++ {
			out = append(out, m.Pix[m.PixOffset(r.Min.X, y):m.PixOffset(r.Max.X, y)]...)
		}
	default:
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				cr, cg, cb, ca := img.At(x, y).RGBA()
				out = append(out, uint8(cr>>8), uint8(cg>>8), uint8(cb>>8), uint8(ca>>8))
			}
		}
	}
	return out
}
