package main

import (
	"image"
	"math"
)

// tolerance is how far, in pixels, flattened curves and round joins may stray from the true
// shapes: well below what anti-aliasing can show.
const tolerance = 0.05

// render draws a source as an n×n picture.
func render(d *doc, n int) *image.NRGBA {
	s := float64(n) / d.vb[2]
	dev := scaling(s, s).mul(translate(-d.vb[0], -d.vb[1]))
	c := newCanvas(n, n)
	drawItems(c, d.items, dev)
	return c.image()
}

func drawItems(c *canvas, items []item, dev matrix) {
	for _, it := range items {
		if it.shape != nil {
			drawShape(c, it.shape, dev)
			continue
		}
		layer := newCanvas(c.w, c.h)
		drawItems(layer, it.group, dev)
		c.composite(layer, it.opacity)
	}
}

func drawShape(c *canvas, s *shape, dev matrix) {
	m := dev.mul(s.ctm)
	scale := m.maxScale()
	if scale == 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		return
	}
	fill := s.fill.at(m, s.bbox)
	var stroke paintFunc
	if s.strokeWidth > 0 {
		stroke = s.stroke.at(m, s.bbox)
	}
	if fill == nil && stroke == nil {
		return
	}
	// A shape's opacity applies to its fill and stroke together, so when it has both they are
	// drawn on a layer of their own first.
	target, opacity := c, s.opacity
	if s.opacity < 1 && fill != nil && stroke != nil {
		target, opacity = newCanvas(c.w, c.h), 1
	}
	polys := flatten(s.path, tolerance/scale)
	if fill != nil {
		rings := make([][]point, len(polys))
		for i, p := range polys {
			rings[i] = p.pts
		}
		target.fill(coverage(transform(rings, m), c.w, c.h, s.evenOdd), fill, opacity)
	}
	if stroke != nil {
		outline := strokeOutline(polys, strokeStyle{
			halfWidth:  s.strokeWidth / 2,
			lineCap:    s.lineCap,
			lineJoin:   s.lineJoin,
			miterLimit: s.miterLimit,
			tol:        tolerance / scale,
		})
		target.fill(coverage(transform(outline, m), c.w, c.h, false), stroke, opacity)
	}
	if target != c {
		c.composite(target, s.opacity)
	}
}

func transform(rings [][]point, m matrix) [][]point {
	out := make([][]point, len(rings))
	for i, r := range rings {
		out[i] = make([]point, len(r))
		for j, p := range r {
			out[i][j] = m.apply(p)
		}
	}
	return out
}

// canvas is a picture being drawn, as premultiplied RGBA from 0 to 1.
type canvas struct {
	w, h int
	px   []float64
}

func newCanvas(w, h int) *canvas {
	return &canvas{w: w, h: h, px: make([]float64, 4*w*h)}
}

// paintFunc gives the premultiplied colour of a paint at a pixel.
type paintFunc func(x, y int) [4]float64

// fill composites a paint over the canvas through a coverage mask (source over).
func (c *canvas) fill(m *mask, p paintFunc, opacity float64) {
	if m == nil {
		return
	}
	for y := range m.h {
		for x := range m.w {
			cov := m.a[y*m.w+x] * opacity
			if cov <= 0 {
				continue
			}
			px, py := x+m.x0, y+m.y0
			s := p(px, py)
			i := 4 * (py*c.w + px)
			k := 1 - s[3]*cov
			for ch := range 4 {
				c.px[i+ch] = s[ch]*cov + c.px[i+ch]*k
			}
		}
	}
}

// composite draws a layer over the canvas at an opacity.
func (c *canvas) composite(layer *canvas, opacity float64) {
	for i := 0; i < len(c.px); i += 4 {
		k := 1 - layer.px[i+3]*opacity
		for ch := range 4 {
			c.px[i+ch] = layer.px[i+ch]*opacity + c.px[i+ch]*k
		}
	}
}

// image converts the canvas to 8-bit straight alpha. Fully transparent pixels are all zero.
func (c *canvas) image() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, c.w, c.h))
	to8 := func(v float64) uint8 { return uint8(math.Floor(math.Max(0, math.Min(1, v))*255 + 0.5)) }
	for i := 0; i < len(c.px); i += 4 {
		a := c.px[i+3]
		if to8(a) == 0 {
			continue
		}
		for ch := range 3 {
			img.Pix[i+ch] = to8(c.px[i+ch] / a)
		}
		img.Pix[i+3] = to8(a)
	}
	return img
}

// mask is the coverage, from 0 to 1, of a shape over a rectangle of the canvas.
type mask struct {
	x0, y0, w, h int
	a            []float64
}

// coverage computes exact-area anti-aliased coverage of closed polygons (every ring is closed
// back to its first point) with the non-zero or even-odd rule. It accumulates the signed area
// each edge sweeps in each pixel, the method of font-rs and golang.org/x/image/vector, whose
// running sum along a row is the winding number with fractional edges.
func coverage(rings [][]point, cw, ch int, evenOdd bool) *mask {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, r := range rings {
		for _, p := range r {
			minX, minY = math.Min(minX, p.x), math.Min(minY, p.y)
			maxX, maxY = math.Max(maxX, p.x), math.Max(maxY, p.y)
		}
	}
	for _, v := range []float64{minX, minY, maxX, maxY} {
		if math.IsInf(v, 0) || math.IsNaN(v) {
			return nil
		}
	}
	x0, y0 := max(0, int(math.Floor(minX))), max(0, int(math.Floor(minY)))
	x1, y1 := min(cw, int(math.Ceil(maxX))), min(ch, int(math.Ceil(maxY)))
	if x0 >= x1 || y0 >= y1 {
		return nil
	}
	m := &mask{x0: x0, y0: y0, w: x1 - x0, h: y1 - y0}
	acc := accumulator{w: m.w, h: m.h, stride: m.w + 2}
	acc.a = make([]float64, acc.stride*m.h)
	ox, oy := float64(x0), float64(y0)
	for _, r := range rings {
		for i, p := range r {
			q := r[(i+1)%len(r)]
			acc.edge(p.x-ox, p.y-oy, q.x-ox, q.y-oy)
		}
	}
	m.a = make([]float64, m.w*m.h)
	for y := range m.h {
		sum := 0.0
		for x := range m.w {
			sum += acc.a[y*acc.stride+x]
			v := math.Abs(sum)
			if evenOdd {
				v = math.Mod(v, 2)
				if v > 1 {
					v = 2 - v
				}
			} else {
				v = math.Min(1, v)
			}
			m.a[y*m.w+x] = v
		}
	}
	return m
}

// accumulator collects signed area per pixel; each row has two spare cells on the right, for
// edges on or past the right border.
type accumulator struct {
	a            []float64
	w, h, stride int
}

// edge adds one edge. The part left of the box counts as if on its left border (it still
// changes the winding of every pixel to its right); the part right of the box counts nowhere.
func (acc *accumulator) edge(xa, ya, xb, yb float64) {
	if ya == yb {
		return
	}
	w := float64(acc.w)
	ts := []float64{0}
	for _, xc := range []float64{0, w} {
		if (xa < xc) != (xb < xc) {
			ts = append(ts, (xc-xa)/(xb-xa))
		}
	}
	ts = append(ts, 1)
	if len(ts) == 4 && ts[1] > ts[2] {
		ts[1], ts[2] = ts[2], ts[1]
	}
	at := func(t float64) (float64, float64) {
		switch t {
		case 0:
			return xa, ya
		case 1:
			return xb, yb
		}
		return xa + (xb-xa)*t, ya + (yb-ya)*t
	}
	for i := 0; i+1 < len(ts); i++ {
		px, py := at(ts[i])
		qx, qy := at(ts[i+1])
		switch mid := (px + qx) / 2; {
		case mid >= w:
			continue
		case mid <= 0:
			px, qx = 0, 0
		}
		acc.line(px, py, qx, qy)
	}
}

// line accumulates an edge that lies within the box horizontally.
func (acc *accumulator) line(x0, y0, x1, y1 float64) {
	if y0 == y1 {
		return
	}
	dir := 1.0
	if y0 > y1 {
		dir = -1
		x0, y0, x1, y1 = x1, y1, x0, y0
	}
	h, w := float64(acc.h), float64(acc.w)
	if y1 <= 0 || y0 >= h {
		return
	}
	dxdy := (x1 - x0) / (y1 - y0)
	x := x0
	if y0 < 0 {
		x -= y0 * dxdy
		y0 = 0
	}
	y1 = math.Min(y1, h)
	clamp := func(v float64) float64 { return math.Max(0, math.Min(w, v)) }
	for y := int(y0); float64(y) < y1; y++ {
		dy := math.Min(float64(y+1), y1) - math.Max(float64(y), y0)
		xnext := x + dxdy*dy
		d := dy * dir
		xa, xb := clamp(x), clamp(xnext)
		if xa > xb {
			xa, xb = xb, xa
		}
		row := acc.a[y*acc.stride : (y+1)*acc.stride]
		x0f := math.Floor(xa)
		x0i := int(x0f)
		x1c := math.Ceil(xb)
		x1i := int(x1c)
		if x1i <= x0i+1 {
			// Within one pixel: split the area by where the edge crosses it on average.
			xm := 0.5*(xa+xb) - x0f
			row[x0i] += d - d*xm
			row[x0i+1] += d * xm
		} else {
			s := 1 / (xb - xa)
			xf := xa - x0f
			a0 := 0.5 * s * (1 - xf) * (1 - xf)
			xl := xb - x1c + 1
			am := 0.5 * s * xl * xl
			row[x0i] += d * a0
			if x1i == x0i+2 {
				row[x0i+1] += d * (1 - a0 - am)
			} else {
				a1 := s * (1.5 - xf)
				row[x0i+1] += d * (a1 - a0)
				for xi := x0i + 2; xi < x1i-1; xi++ {
					row[xi] += d * s
				}
				a2 := a1 + float64(x1i-x0i-3)*s
				row[x1i-1] += d * (1 - a2 - am)
			}
			row[x1i] += d * am
		}
		x = xnext
	}
}

// at returns the paint's colour function for a shape drawn with transform m (user space to
// pixels) whose geometry has bounding box bb, or nil when it paints nothing.
func (p paint) at(m matrix, bb box) paintFunc {
	if p.none {
		return nil
	}
	premul := func(c rgba, o float64) [4]float64 {
		a := c.a * o
		return [4]float64{c.r * a, c.g * a, c.b * a, a}
	}
	solid := func(c [4]float64) paintFunc {
		if c[3] == 0 {
			return nil
		}
		return func(int, int) [4]float64 { return c }
	}
	g := p.grad
	if g == nil {
		return solid(premul(p.color, p.opacity))
	}
	switch len(g.stops) {
	case 0:
		return nil
	case 1:
		return solid(premul(g.stops[0].color, p.opacity))
	}
	last := premul(g.stops[len(g.stops)-1].color, p.opacity)
	t := m
	if !g.userSpace {
		w, h := bb.x1-bb.x0, bb.y1-bb.y0
		if !(w > 0 && h > 0) {
			return nil // a bounding box with no area takes no objectBoundingBox paint
		}
		t = t.mul(translate(bb.x0, bb.y0)).mul(scaling(w, h))
	}
	inv, ok := t.mul(g.transform).invert()
	if !ok {
		return nil
	}
	// Stops blend with straight colour and opacity, as SVG 1.1 says and browsers draw them.
	cols := make([][4]float64, len(g.stops))
	for i, s := range g.stops {
		cols[i] = [4]float64{s.color.r, s.color.g, s.color.b, s.color.a}
	}
	blend := func(v float64) [4]float64 {
		switch g.spread {
		case "repeat":
			v -= math.Floor(v)
		case "reflect":
			v -= 2 * math.Floor(v/2)
			if v > 1 {
				v = 2 - v
			}
		}
		if v <= g.stops[0].offset {
			return cols[0]
		}
		for i := 1; i < len(g.stops); i++ {
			lo, hi := g.stops[i-1].offset, g.stops[i].offset
			if v >= hi {
				continue
			}
			f := (v - lo) / (hi - lo)
			var c [4]float64
			for ch := range 4 {
				c[ch] = cols[i-1][ch] + (cols[i][ch]-cols[i-1][ch])*f
			}
			return c
		}
		return cols[len(cols)-1]
	}
	lookup := func(v float64) [4]float64 {
		c := blend(v)
		return premul(rgba{c[0], c[1], c[2], c[3]}, p.opacity)
	}
	if !g.radial {
		p1, p2 := point{g.x1, g.y1}, point{g.x2, g.y2}
		d := p2.sub(p1)
		dd := d.dot(d)
		if dd == 0 {
			return solid(last) // a gradient of no length paints its last stop
		}
		return func(x, y int) [4]float64 {
			q := inv.apply(point{float64(x) + 0.5, float64(y) + 0.5})
			return lookup(q.sub(p1).dot(d) / dd)
		}
	}
	if g.r == 0 {
		return solid(last)
	}
	c, f := point{g.cx, g.cy}, point{g.fx, g.fy}
	// SVG 1.1: a focal point outside the circle moves onto it; keep it just inside so every
	// ray from it meets the circle once.
	if e := f.sub(c); e.length() > 0.999*g.r {
		f = c.add(e.scale(0.999 * g.r / e.length()))
	}
	e := f.sub(c)
	ee := e.dot(e) - g.r*g.r
	return func(x, y int) [4]float64 {
		q := inv.apply(point{float64(x) + 0.5, float64(y) + 0.5})
		d := q.sub(f)
		a := d.dot(d)
		if a == 0 {
			return lookup(0)
		}
		// q = f + d/t, where f + s·d is on the circle: solve |e + s·d| = r for s > 0.
		b := e.dot(d)
		s := (-b + math.Sqrt(b*b-a*ee)) / a
		return lookup(1 / s)
	}
}
