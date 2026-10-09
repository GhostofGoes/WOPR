package main

import (
	"image"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// svgDoc wraps elements in a root with a square viewBox of the given size.
func svgDoc(size int, body string) string {
	n := strconv.Itoa(size)
	return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ` + n + ` ` + n + `">` + body + `</svg>`
}

func draw(t *testing.T, size int, body string) *image.NRGBA {
	t.Helper()
	d, err := parseSVG("test.svg", []byte(svgDoc(size, body)))
	if err != nil {
		t.Fatal(err)
	}
	return render(d, size)
}

func rgbaAt(img *image.NRGBA, x, y int) [4]int {
	p := img.Pix[img.PixOffset(x, y):]
	return [4]int{int(p[0]), int(p[1]), int(p[2]), int(p[3])}
}

func near(a, b [4]int, tol int) bool {
	for i := range a {
		if d := a[i] - b[i]; d > tol || d < -tol {
			return false
		}
	}
	return true
}

// TestUnsupported checks that everything outside the subset is refused, with a message that
// says what and where.
func TestUnsupported(t *testing.T) {
	for _, c := range []struct{ name, svg, want string }{
		{"text", svgDoc(10, `<text x="1" y="5">WOPR</text>`), "<text> is not supported"},
		{"filter", svgDoc(10, `<filter id="f"/>`), "<filter> is not supported"},
		{"mask", svgDoc(10, `<mask id="m"/>`), "<mask> is not supported"},
		{"clipPath", svgDoc(10, `<clipPath id="c"/>`), "<clipPath> is not supported"},
		{"pattern", svgDoc(10, `<defs><pattern id="p"/></defs>`), "<pattern> is not supported"},
		{"image", svgDoc(10, `<image width="1" height="1"/>`), "<image> is not supported"},
		{"use", svgDoc(10, `<use href="#a"/>`), "<use> is not supported"},
		{"style element", svgDoc(10, `<style>rect{fill:red}</style>`), "<style> is not supported"},
		{"style attribute", svgDoc(10, `<rect width="5" height="5" style="fill:red"/>`), "attribute style on <rect>"},
		{"class", svgDoc(10, `<rect width="5" height="5" class="x"/>`), "attribute class on <rect>"},
		{"dashes", svgDoc(10, `<path d="M0 0 L5 5" stroke="red" stroke-dasharray="1 1"/>`), "attribute stroke-dasharray"},
		{"transform on a gradient's stop", svgDoc(10, `<linearGradient id="g"><stop transform="scale(2)"/></linearGradient>`), "attribute transform on <stop>"},
		{"gradient href", svgDoc(10, `<linearGradient id="g" href="#h"/>`), "attribute href on <linearGradient>"},
		{"editor namespace", `<svg xmlns="http://www.w3.org/2000/svg" xmlns:inkscape="http://www.inkscape.org/namespaces/inkscape" viewBox="0 0 10 10"/>`, "namespace declaration"},
		{"editor element", `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><n:view xmlns:n="urn:x"/></svg>`, "<view> in namespace urn:x is not supported"},
		{"xml:space", `<svg xmlns="http://www.w3.org/2000/svg" xml:space="preserve" viewBox="0 0 10 10"/>`, "attribute xml:space on <svg>"},
		{"doctype", `<!DOCTYPE svg><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"/>`, "directives are not supported"},
		{"no namespace", `<svg viewBox="0 0 10 10"/>`, "not in the SVG namespace"},
		{"nested svg", svgDoc(10, `<svg/>`), "<svg> is allowed only as the root"},
		{"stray text", svgDoc(10, `WOPR`), "text \"WOPR\" is not supported"},
		{"stop outside a gradient", svgDoc(10, `<g><stop/></g>`), "<g> may not contain <stop>"},
		{"shape in defs", svgDoc(10, `<defs><rect width="1" height="1"/></defs>`), "<defs> may not contain <rect>"},
		{"no viewBox", `<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"/>`, "needs a viewBox"},
		{"oblong viewBox", `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 20"/>`, "is not a square"},
		{"bad path", svgDoc(10, `<path d="M0 0 L5 5 X"/>`), "path data: unexpected \"X\""},
		{"path not starting with M", svgDoc(10, `<path d="L5 5"/>`), "must start with M"},
		{"bad arc flag", svgDoc(10, `<path d="M0 0 A5 5 0 2 1 5 5"/>`), "arc flag"},
		{"missing gradient", svgDoc(10, `<rect width="5" height="5" fill="url(#nope)"/>`), "names no gradient"},
		{"gradient without id", svgDoc(10, `<linearGradient/>`), "needs an id"},
		{"duplicate id", svgDoc(10, `<g id="a"/><g id="a"/>`), "duplicate id"},
		{"colour name", svgDoc(10, `<rect width="5" height="5" fill="orange"/>`), "colour \"orange\" is not supported"},
		{"percentage", svgDoc(10, `<rect width="50%" height="5"/>`), "px is the only unit"},
		{"unit", svgDoc(10, `<circle r="2mm"/>`), "px is the only unit"},
		{"bad transform", svgDoc(10, `<g transform="spin(3)"/>`), "spin with 1 arguments is not a transform"},
		{"bad fill-rule", svgDoc(10, `<path d="M0 0" fill-rule="odd"/>`), "fill-rule"},
		{"bad linejoin", svgDoc(10, `<path d="M0 0" stroke-linejoin="arcs"/>`), "stroke-linejoin"},
		{"negative size", svgDoc(10, `<rect width="-5" height="5"/>`), "negative size"},
		{"odd points", svgDoc(10, `<polygon points="0,0 5"/>`), "odd number"},
		{"not a number", svgDoc(10, `<circle r="NaN"/>`), "not a number"},
	} {
		_, err := parseSVG("test.svg", []byte(c.svg))
		if err == nil || !strings.Contains(err.Error(), c.want) || !strings.HasPrefix(err.Error(), "test.svg:") {
			t.Errorf("%s: got %v, want an error with %q", c.name, err, c.want)
		}
	}
}

// TestSubset parses a document that uses every supported element and attribute.
func TestSubset(t *testing.T) {
	src := `<?xml version="1.0" encoding="UTF-8"?>
<!-- a comment -->
<svg xmlns="http://www.w3.org/2000/svg" version="1.1" viewBox="-5 -5 110 110" width="110px" height="110" fill="#123" stroke-width="2">
  <title>All of it</title>
  <desc>Every element &amp; attribute</desc>
  <defs>
    <linearGradient id="l" x1="0%" y1="0" x2="100%" y2="1" gradientUnits="objectBoundingBox" spreadMethod="pad" gradientTransform="rotate(5)">
      <stop offset="0" stop-color="#fff" stop-opacity=".5"/>
      <stop offset="100%" stop-color="rgb(10%, 20, 30)"/>
    </linearGradient>
  </defs>
  <radialGradient id="r" cx="50" cy="50" r="40" fx="40" fy="45" gradientUnits="userSpaceOnUse" spreadMethod="repeat">
    <stop offset="0.2" stop-color="teal"/>
    <stop offset="0.1" stop-color="white"/>
  </radialGradient>
  <g id="g" transform="translate(1,2) scale(0.9) skewY(1)" opacity="0.9" fill-opacity="0.8" stroke-opacity="0.7">
    <rect x="1" y="1" width="20" height="20" rx="3" fill="url(#l)" stroke="black" stroke-linejoin="bevel"/>
    <circle cx="50" cy="50" r="10" fill="url(#r)" fill-rule="evenodd"/>
    <ellipse cx="80" cy="20" rx="10" ry="5" transform="matrix(1 0 0 1 0 0)"/>
    <line x1="0" y1="90" x2="30" y2="90" stroke="red" stroke-linecap="round"/>
    <polyline points="40 90,50 80 60 90" fill="none" stroke="lime" stroke-miterlimit="8"/>
    <polygon points="70,90 80,80 90,90"><title>a triangle</title></polygon>
    <path d="m10 30 h5 v5 H10 V30 z M20 30 c1 1 2 2 3 3 s1 1 2 2 q1 1 2 2 t2 2 a2 3 4 1 0 3 3 l1e1-.5.5.5 Z"/>
  </g>
</svg>`
	d, err := parseSVG("all.svg", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if d.vb != [4]float64{-5, -5, 110, 110} || len(d.items) != 1 || len(d.items[0].group) != 7 {
		t.Fatalf("viewBox %v, %d items", d.vb, len(d.items))
	}
	r := d.items[0].group[1].shape.fill.grad
	if r.stops[1].offset != 0.2 || !r.userSpace || r.spread != "repeat" {
		t.Errorf("radial gradient: %+v", r)
	}
	img := render(d, 64)
	if img.Bounds().Dx() != 64 {
		t.Fatal("wrong size")
	}
	small := string(minify(d.root))
	for _, w := range []string{`<title>All of it</title>`, `<desc>Every element &amp; attribute</desc>`, `<stop offset="0" stop-color="#fff" stop-opacity=".5"/>`} {
		if !strings.Contains(small, w) {
			t.Errorf("minified source lacks %s", w)
		}
	}
	if strings.Contains(small, "comment") || strings.Contains(small, "<?xml") {
		t.Error("minified source keeps the comment or declaration")
	}
	if _, err := parseSVG("again.svg", []byte(small)); err != nil {
		t.Errorf("minified source does not parse: %v", err)
	}
}

func TestRender(t *testing.T) {
	black := [4]int{0, 0, 0, 255}
	empty := [4]int{}

	// A full square of colour covers every pixel exactly.
	img := draw(t, 16, `<rect width="16" height="16" fill="#336699"/>`)
	for i := 0; i < len(img.Pix); i += 4 {
		if [4]byte(img.Pix[i:i+4]) != [4]byte{0x33, 0x66, 0x99, 255} {
			t.Fatalf("pixel %d is %v", i/4, img.Pix[i:i+4])
		}
	}

	// Half a pixel is half covered; the scale(s) form scales both ways.
	img = draw(t, 8, `<rect x="2" y="2" width="2.5" height="4"/><rect transform="scale(2)" x="3" y="0" width="1" height="1"/>`)
	if got := rgbaAt(img, 4, 3); got[3] != 128 || rgbaAt(img, 3, 3) != black || rgbaAt(img, 5, 3) != empty {
		t.Errorf("edge pixels: %v %v %v", rgbaAt(img, 3, 3), got, rgbaAt(img, 5, 3))
	}
	if rgbaAt(img, 7, 1) != black || rgbaAt(img, 7, 2) != empty {
		t.Error("scale(2) did not scale both ways")
	}

	// A circle's coverage adds up to its area.
	img = draw(t, 100, `<circle cx="50" cy="50" r="30"/>`)
	var area float64
	for i := 3; i < len(img.Pix); i += 4 {
		area += float64(img.Pix[i]) / 255
	}
	if want := math.Pi * 900; math.Abs(area-want) > want*0.002 {
		t.Errorf("circle area %.1f, want %.1f", area, want)
	}
	// The same circle from arcs.
	arcs := draw(t, 100, `<path d="M20 50 A30 30 0 0 1 80 50 A30 30 0 1 1 20 50 Z"/>`)
	if err := samePicture(img, arcs); err != nil {
		t.Errorf("a circle of arcs: %v", err)
	}

	// Even-odd leaves the inner square empty; non-zero fills it when both run the same way.
	ring := `<path d="M2 2 H14 V14 H2 Z M6 6 H10 V10 H6 Z" fill-rule="`
	if rgbaAt(draw(t, 16, ring+`evenodd"/>`), 8, 8) != empty {
		t.Error("even-odd filled the hole")
	}
	if rgbaAt(draw(t, 16, ring+`nonzero"/>`), 8, 8) != black {
		t.Error("non-zero left a hole")
	}

	// A stroke two units wide, with butt and square caps.
	img = draw(t, 16, `<path d="M2 8 H14" stroke="#000" stroke-width="2" fill="none"/><path d="M2 12 H14" stroke="#000" stroke-width="2" stroke-linecap="square"/>`)
	if rgbaAt(img, 2, 7) != black || rgbaAt(img, 13, 8) != black || rgbaAt(img, 1, 7) != empty || rgbaAt(img, 8, 6) != empty || rgbaAt(img, 8, 9) != empty {
		t.Error("butt-capped stroke covers the wrong pixels")
	}
	if rgbaAt(img, 1, 11) != black || rgbaAt(img, 14, 12) != black || rgbaAt(img, 0, 11) != empty {
		t.Error("square caps do not extend half the width")
	}

	// Joins: a miter fills the corner's tip, a bevel cuts it, a round one rounds it.
	for join, want := range map[string]bool{"miter": true, "bevel": false, "round": false} {
		img = draw(t, 32, `<path d="M4 28 V4 H28" fill="none" stroke="#000" stroke-width="6" stroke-linejoin="`+join+`"/>`)
		if got := rgbaAt(img, 1, 1)[3] > 200; got != want {
			t.Errorf("%s join: corner covered %t", join, got)
		}
		if rgbaAt(img, 4, 4) != black {
			t.Errorf("%s join: the corner itself is not covered", join)
		}
	}
	// A sharp miter beyond the limit is bevelled.
	img = draw(t, 64, `<path d="M20 60 L32 4 L44 60" fill="none" stroke="#000" stroke-width="8"/>`)
	if rgbaAt(img, 32, 1)[3] != 0 {
		t.Error("the miter limit did not bevel a sharp corner")
	}
	// A zero-length subpath with round caps draws a dot.
	img = draw(t, 16, `<path d="M8 8 Z" stroke="#000" stroke-width="6" stroke-linecap="round"/>`)
	if rgbaAt(img, 8, 8) != black || rgbaAt(img, 8, 12) != empty {
		t.Error("a round-capped dot is missing")
	}

	// Opacity on a shape applies to its fill and stroke together.
	img = draw(t, 16, `<rect x="4" y="4" width="8" height="8" fill="#000" stroke="#000" stroke-width="4" opacity="0.5"/>`)
	if got := rgbaAt(img, 4, 8)[3]; got != 128 {
		t.Errorf("fill under stroke at opacity 0.5 has alpha %d, want 128", got)
	}
	img = draw(t, 16, `<g opacity="0.5"><rect width="10" height="10"/><rect x="5" width="10" height="10"/></g>`)
	if got := rgbaAt(img, 7, 5)[3]; got != 128 {
		t.Errorf("overlap in a group at opacity 0.5 has alpha %d, want 128", got)
	}

	// Gradients: the ends of a linear one, the same in user space, and a radial one's centre.
	grads := `<defs><linearGradient id="a"><stop offset="0" stop-color="#ff0000"/><stop offset="1" stop-color="#0000ff"/></linearGradient>
<linearGradient id="u" gradientUnits="userSpaceOnUse" x1="10" x2="90"><stop offset="0" stop-color="#ff0000"/><stop offset="1" stop-color="#0000ff"/></linearGradient>
<radialGradient id="r"><stop offset="0" stop-color="#fff"/><stop offset="1" stop-color="#000"/></radialGradient></defs>`
	img = draw(t, 100, grads+`<rect x="10" y="0" width="80" height="40" fill="url(#a)"/><rect x="10" y="40" width="80" height="20" fill="url(#u)"/><circle cx="50" cy="80" r="20" fill="url(#r)"/>`)
	for _, c := range []struct {
		x, y int
		want [4]int
		tol  int
	}{
		{10, 20, [4]int{254, 0, 1, 255}, 2},
		{89, 20, [4]int{1, 0, 254, 255}, 2},
		{50, 20, [4]int{126, 0, 129, 255}, 2},
		{50, 50, [4]int{126, 0, 129, 255}, 2},
		{50, 80, [4]int{246, 246, 246, 255}, 2},
		{50, 69, [4]int{128, 128, 128, 255}, 8},
	} {
		if got := rgbaAt(img, c.x, c.y); !near(got, c.want, c.tol) {
			t.Errorf("gradient at (%d, %d) is %v, want %v", c.x, c.y, got, c.want)
		}
	}
	// Stops blend straight colour and opacity separately, as browsers do.
	img = draw(t, 100, `<linearGradient id="o"><stop offset="0" stop-color="#ff0000"/><stop offset="1" stop-color="#0000ff" stop-opacity="0"/></linearGradient><rect width="100" height="100" fill="url(#o)"/>`)
	if got := rgbaAt(img, 50, 50); !near(got, [4]int{126, 0, 129, 126}, 2) {
		t.Errorf("half way to a clear stop: %v", got)
	}
}

// FuzzParseSVG feeds the parser and renderer arbitrary documents: they may refuse one, but
// never panic or hang.
func FuzzParseSVG(f *testing.F) {
	for _, src := range repoSources(f) {
		data, err := os.ReadFile(filepath.Join(repoRoot, src))
		if err != nil {
			f.Fatal(err)
		}
		f.Add(data)
	}
	f.Add([]byte(svgDoc(10, `<path d="M1 1 a3 2 30 1 1 5 5 q1 1 2 0 t1 1 s1 1 2 2 z" stroke="red" stroke-width="3" stroke-linejoin="round" stroke-linecap="square"/>`)))
	f.Add([]byte(svgDoc(10, `<radialGradient id="r" fx="2" fy="-1" spreadMethod="reflect"><stop offset=".5"/></radialGradient><circle r="5" fill="url(#r)" opacity=".5" stroke="#fff"/>`)))
	f.Fuzz(func(_ *testing.T, data []byte) {
		d, err := parseSVG("fuzz.svg", data)
		if err != nil {
			return
		}
		render(d, 12)
		minify(d.root)
	})
}
