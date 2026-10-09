package main

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type point struct{ x, y float64 }

func (p point) add(q point) point             { return point{p.x + q.x, p.y + q.y} }
func (p point) sub(q point) point             { return point{p.x - q.x, p.y - q.y} }
func (p point) scale(f float64) point         { return point{p.x * f, p.y * f} }
func (p point) dot(q point) float64           { return p.x*q.x + p.y*q.y }
func (p point) cross(q point) float64         { return p.x*q.y - p.y*q.x }
func (p point) length() float64               { return math.Hypot(p.x, p.y) }
func (p point) lerp(q point, t float64) point { return point{p.x + (q.x-p.x)*t, p.y + (q.y-p.y)*t} }

// matrix is an affine transform: x' = a*x + c*y + e, y' = b*x + d*y + f.
type matrix struct{ a, b, c, d, e, f float64 }

var identity = matrix{a: 1, d: 1}

// mul returns m·n, the transform that applies n first.
func (m matrix) mul(n matrix) matrix {
	return matrix{
		a: m.a*n.a + m.c*n.b,
		b: m.b*n.a + m.d*n.b,
		c: m.a*n.c + m.c*n.d,
		d: m.b*n.c + m.d*n.d,
		e: m.a*n.e + m.c*n.f + m.e,
		f: m.b*n.e + m.d*n.f + m.f,
	}
}

func (m matrix) apply(p point) point {
	return point{m.a*p.x + m.c*p.y + m.e, m.b*p.x + m.d*p.y + m.f}
}

func (m matrix) invert() (matrix, bool) {
	det := m.a*m.d - m.b*m.c
	if det == 0 || math.IsNaN(det) || math.IsInf(det, 0) {
		return matrix{}, false
	}
	return matrix{
		a: m.d / det,
		b: -m.b / det,
		c: -m.c / det,
		d: m.a / det,
		e: (m.c*m.f - m.d*m.e) / det,
		f: (m.b*m.e - m.a*m.f) / det,
	}, true
}

// maxScale is how much the transform stretches a length at most (its largest singular value).
func (m matrix) maxScale() float64 {
	s := m.a*m.a + m.b*m.b + m.c*m.c + m.d*m.d
	det := m.a*m.d - m.b*m.c
	return math.Sqrt((s + math.Sqrt(math.Max(0, s*s-4*det*det))) / 2)
}

func translate(x, y float64) matrix { return matrix{a: 1, d: 1, e: x, f: y} }
func scaling(x, y float64) matrix   { return matrix{a: x, d: y} }

// segment is one path command in absolute coordinates: 'M' and 'L' use p[0], 'Q' p[0..1],
// 'C' p[0..2] (the last point is the end point), and 'Z' none.
type segment struct {
	op byte
	p  [3]point
}

// pathBuilder collects segments and remembers the current point.
type pathBuilder struct {
	segs       []segment
	cur, start point
}

func (b *pathBuilder) moveTo(p point) {
	b.segs = append(b.segs, segment{op: 'M', p: [3]point{p}})
	b.cur, b.start = p, p
}

func (b *pathBuilder) lineTo(p point) {
	b.segs = append(b.segs, segment{op: 'L', p: [3]point{p}})
	b.cur = p
}

func (b *pathBuilder) quadTo(c, p point) {
	b.segs = append(b.segs, segment{op: 'Q', p: [3]point{c, p}})
	b.cur = p
}

func (b *pathBuilder) cubeTo(c1, c2, p point) {
	b.segs = append(b.segs, segment{op: 'C', p: [3]point{c1, c2, p}})
	b.cur = p
}

func (b *pathBuilder) close() {
	b.segs = append(b.segs, segment{op: 'Z'})
	b.cur = b.start
}

// arcTo adds an elliptical arc as cubic Béziers, following the SVG specification's
// implementation notes (endpoint to centre parameterization, out-of-range radii).
func (b *pathBuilder) arcTo(rx, ry, angle float64, large, sweep bool, p point) {
	p0 := b.cur
	if p0 == p {
		return
	}
	rx, ry = math.Abs(rx), math.Abs(ry)
	if rx == 0 || ry == 0 {
		b.lineTo(p)
		return
	}
	sin, cos := math.Sincos(angle * math.Pi / 180)
	dx, dy := (p0.x-p.x)/2, (p0.y-p.y)/2
	x1 := cos*dx + sin*dy
	y1 := -sin*dx + cos*dy
	if l := x1*x1/(rx*rx) + y1*y1/(ry*ry); l > 1 {
		rx, ry = rx*math.Sqrt(l), ry*math.Sqrt(l)
	}
	num := rx*rx*ry*ry - rx*rx*y1*y1 - ry*ry*x1*x1
	den := rx*rx*y1*y1 + ry*ry*x1*x1
	coef := math.Sqrt(math.Max(0, num/den))
	if large == sweep {
		coef = -coef
	}
	cx1 := coef * rx * y1 / ry
	cy1 := -coef * ry * x1 / rx
	centre := point{cos*cx1 - sin*cy1 + (p0.x+p.x)/2, sin*cx1 + cos*cy1 + (p0.y+p.y)/2}
	vecAngle := func(u, v point) float64 { return math.Atan2(u.cross(v), u.dot(v)) }
	u := point{(x1 - cx1) / rx, (y1 - cy1) / ry}
	v := point{(-x1 - cx1) / rx, (-y1 - cy1) / ry}
	theta := vecAngle(point{1, 0}, u)
	delta := vecAngle(u, v)
	if !sweep && delta > 0 {
		delta -= 2 * math.Pi
	} else if sweep && delta < 0 {
		delta += 2 * math.Pi
	}
	n := int(math.Ceil(math.Abs(delta)/(math.Pi/2) - 1e-9))
	n = max(n, 1)
	step := delta / float64(n)
	k := 4.0 / 3 * math.Tan(step/4)
	toUser := func(x, y float64) point {
		return point{centre.x + cos*rx*x - sin*ry*y, centre.y + sin*rx*x + cos*ry*y}
	}
	for i := range n {
		t0 := theta + float64(i)*step
		t1 := t0 + step
		s0, c0 := math.Sincos(t0)
		s1, c1 := math.Sincos(t1)
		end := toUser(c1, s1)
		if i == n-1 {
			end = p
		}
		b.cubeTo(toUser(c0-k*s0, s0+k*c0), toUser(c1+k*s1, s1-k*c1), end)
	}
}

// rect adds a rectangle, with rounded corners when rx and ry are both above zero.
func rect(b *pathBuilder, x, y, w, h, rx, ry float64) {
	if w == 0 || h == 0 {
		return
	}
	if rx == 0 || ry == 0 {
		b.moveTo(point{x, y})
		b.lineTo(point{x + w, y})
		b.lineTo(point{x + w, y + h})
		b.lineTo(point{x, y + h})
		b.close()
		return
	}
	b.moveTo(point{x + rx, y})
	b.lineTo(point{x + w - rx, y})
	b.arcTo(rx, ry, 0, false, true, point{x + w, y + ry})
	b.lineTo(point{x + w, y + h - ry})
	b.arcTo(rx, ry, 0, false, true, point{x + w - rx, y + h})
	b.lineTo(point{x + rx, y + h})
	b.arcTo(rx, ry, 0, false, true, point{x, y + h - ry})
	b.lineTo(point{x, y + ry})
	b.arcTo(rx, ry, 0, false, true, point{x + rx, y})
	b.close()
}

// ellipse adds an ellipse as four quarter arcs.
func ellipse(b *pathBuilder, cx, cy, rx, ry float64) {
	if rx == 0 || ry == 0 {
		return
	}
	b.moveTo(point{cx + rx, cy})
	b.arcTo(rx, ry, 0, false, true, point{cx, cy + ry})
	b.arcTo(rx, ry, 0, false, true, point{cx - rx, cy})
	b.arcTo(rx, ry, 0, false, true, point{cx, cy - ry})
	b.arcTo(rx, ry, 0, false, true, point{cx + rx, cy})
	b.close()
}

// scanner reads the number lists of path data, points, viewBox and transforms.
type scanner struct {
	s string
	i int
}

// skip passes whitespace and commas.
func (sc *scanner) skip() {
	for sc.i < len(sc.s) && strings.IndexByte(" \t\r\n\f,", sc.s[sc.i]) >= 0 {
		sc.i++
	}
}

// atNumber reports whether a number starts at the current position.
func (sc *scanner) atNumber() bool {
	return sc.i < len(sc.s) && strings.IndexByte("+-.0123456789", sc.s[sc.i]) >= 0
}

// number reads one number in SVG's grammar: no Inf, NaN, hexadecimal or underscores.
func (sc *scanner) number() (float64, error) {
	start, i := sc.i, sc.i
	s := sc.s
	digits := func() int {
		n := 0
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
			n++
		}
		return n
	}
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	n := digits()
	if i < len(s) && s[i] == '.' {
		i++
		n += digits()
	}
	if n == 0 {
		return 0, fmt.Errorf("expected a number at %q", rest(s, start))
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		j := i
		i++
		if i < len(s) && (s[i] == '+' || s[i] == '-') {
			i++
		}
		if digits() == 0 {
			i = j // not an exponent: leave the e for the caller to reject
		}
	}
	f, err := strconv.ParseFloat(s[start:i], 64)
	if err != nil || math.IsInf(f, 0) {
		return 0, fmt.Errorf("bad number %q", s[start:i])
	}
	sc.i = i
	return f, nil
}

// flag reads an arc flag, a single 0 or 1.
func (sc *scanner) flag() (bool, error) {
	sc.skip()
	if sc.i < len(sc.s) && (sc.s[sc.i] == '0' || sc.s[sc.i] == '1') {
		sc.i++
		return sc.s[sc.i-1] == '1', nil
	}
	return false, fmt.Errorf("expected an arc flag (0 or 1) at %q", rest(sc.s, sc.i))
}

func rest(s string, i int) string {
	if len(s)-i > 12 {
		return s[i:i+12] + "..."
	}
	return s[i:]
}

// parseNumber reads a single number.
func parseNumber(v string) (float64, error) {
	sc := scanner{s: strings.TrimSpace(v)}
	f, err := sc.number()
	if err == nil && sc.i != len(sc.s) {
		err = fmt.Errorf("%q is not a number", v)
	}
	return f, err
}

// parseLength reads a number with an optional px unit; other units are not supported.
func parseLength(v string) (float64, error) {
	f, err := parseNumber(strings.TrimSuffix(strings.TrimSpace(v), "px"))
	if err != nil {
		return 0, fmt.Errorf("%q is not a number of user units (px is the only unit allowed)", v)
	}
	return f, nil
}

// parseNumbers reads a list of numbers separated by whitespace or commas.
func parseNumbers(v string) ([]float64, error) {
	sc := scanner{s: v}
	var out []float64
	for sc.skip(); sc.i < len(sc.s); sc.skip() {
		f, err := sc.number()
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

// parseTransform reads a transform list: matrix, translate, scale, rotate, skewX and skewY.
func parseTransform(v string) (matrix, error) {
	m := identity
	sc := scanner{s: v}
	for sc.skip(); sc.i < len(sc.s); sc.skip() {
		start := sc.i
		for sc.i < len(sc.s) && (sc.s[sc.i] >= 'a' && sc.s[sc.i] <= 'z' || sc.s[sc.i] >= 'A' && sc.s[sc.i] <= 'Z') {
			sc.i++
		}
		name := sc.s[start:sc.i]
		for sc.i < len(sc.s) && strings.IndexByte(" \t\r\n\f", sc.s[sc.i]) >= 0 {
			sc.i++
		}
		if sc.i >= len(sc.s) || sc.s[sc.i] != '(' {
			return m, fmt.Errorf("expected a transform function at %q", rest(sc.s, start))
		}
		sc.i++
		var args []float64
		for sc.skip(); sc.i < len(sc.s) && sc.s[sc.i] != ')'; sc.skip() {
			f, err := sc.number()
			if err != nil {
				return m, err
			}
			args = append(args, f)
		}
		if sc.i >= len(sc.s) {
			return m, fmt.Errorf("%s( has no closing parenthesis", name)
		}
		sc.i++
		t, err := transformFunc(name, args)
		if err != nil {
			return m, err
		}
		m = m.mul(t)
	}
	return m, nil
}

func transformFunc(name string, a []float64) (matrix, error) {
	rad := func(deg float64) float64 { return deg * math.Pi / 180 }
	switch {
	case name == "matrix" && len(a) == 6:
		return matrix{a[0], a[1], a[2], a[3], a[4], a[5]}, nil
	case name == "translate" && len(a) == 1:
		return translate(a[0], 0), nil
	case name == "translate" && len(a) == 2:
		return translate(a[0], a[1]), nil
	case name == "scale" && len(a) == 1:
		return scaling(a[0], a[0]), nil
	case name == "scale" && len(a) == 2:
		return scaling(a[0], a[1]), nil
	case name == "rotate" && (len(a) == 1 || len(a) == 3):
		s, c := math.Sincos(rad(a[0]))
		r := matrix{a: c, b: s, c: -s, d: c}
		if len(a) == 3 {
			r = translate(a[1], a[2]).mul(r).mul(translate(-a[1], -a[2]))
		}
		return r, nil
	case name == "skewX" && len(a) == 1:
		return matrix{a: 1, c: math.Tan(rad(a[0])), d: 1}, nil
	case name == "skewY" && len(a) == 1:
		return matrix{a: 1, b: math.Tan(rad(a[0])), d: 1}, nil
	}
	return identity, fmt.Errorf("%s with %d arguments is not a transform", name, len(a))
}

// parsePathData reads a path's d attribute into absolute segments.
func parsePathData(d string) ([]segment, error) {
	var b pathBuilder
	sc := scanner{s: d}
	var cmd byte
	var lastCtrl point // the last control point, for S and T
	var lastOp byte
	for sc.skip(); sc.i < len(sc.s); sc.skip() {
		if c := sc.s[sc.i]; strings.IndexByte("MmZzLlHhVvCcSsQqTtAa", c) >= 0 {
			cmd = c
			sc.i++
		} else if !sc.atNumber() || cmd == 0 || cmd == 'Z' || cmd == 'z' {
			return nil, fmt.Errorf("path data: unexpected %q", rest(sc.s, sc.i))
		}
		if len(b.segs) == 0 && cmd != 'M' && cmd != 'm' {
			return nil, errors.New("path data must start with M or m")
		}
		rel := cmd >= 'a'
		abs := func(p point) point {
			if rel {
				return p.add(b.cur)
			}
			return p
		}
		var nums [7]float64
		args := map[byte]int{'M': 2, 'L': 2, 'H': 1, 'V': 1, 'C': 6, 'S': 4, 'Q': 4, 'T': 2, 'A': 7, 'Z': 0}[cmd&^0x20]
		for i := range args {
			sc.skip()
			var err error
			if cmd&^0x20 == 'A' && (i == 3 || i == 4) {
				var f bool
				f, err = sc.flag()
				if f {
					nums[i] = 1
				}
			} else {
				nums[i], err = sc.number()
			}
			if err != nil {
				return nil, fmt.Errorf("path data: %w", err)
			}
		}
		// After a Z the next command starts a new subpath at the closed one's start.
		if lastOp == 'Z' && cmd&^0x20 != 'M' && cmd&^0x20 != 'Z' {
			b.moveTo(b.cur)
		}
		ctrl := b.cur
		switch cmd &^ 0x20 {
		case 'M':
			b.moveTo(abs(point{nums[0], nums[1]}))
			// Further pairs after a moveto are lineto commands.
			if rel {
				cmd = 'l'
			} else {
				cmd = 'L'
			}
		case 'Z':
			b.close()
		case 'L':
			b.lineTo(abs(point{nums[0], nums[1]}))
		case 'H':
			x := nums[0]
			if rel {
				x += b.cur.x
			}
			b.lineTo(point{x, b.cur.y})
		case 'V':
			y := nums[0]
			if rel {
				y += b.cur.y
			}
			b.lineTo(point{b.cur.x, y})
		case 'C':
			c1, c2, p := abs(point{nums[0], nums[1]}), abs(point{nums[2], nums[3]}), abs(point{nums[4], nums[5]})
			b.cubeTo(c1, c2, p)
			ctrl = c2
		case 'S':
			c1 := b.cur
			if lastOp == 'C' {
				c1 = b.cur.scale(2).sub(lastCtrl)
			}
			c2, p := abs(point{nums[0], nums[1]}), abs(point{nums[2], nums[3]})
			b.cubeTo(c1, c2, p)
			ctrl = c2
		case 'Q':
			c, p := abs(point{nums[0], nums[1]}), abs(point{nums[2], nums[3]})
			b.quadTo(c, p)
			ctrl = c
		case 'T':
			c := b.cur
			if lastOp == 'Q' {
				c = b.cur.scale(2).sub(lastCtrl)
			}
			b.quadTo(c, abs(point{nums[0], nums[1]}))
			ctrl = c
		case 'A':
			b.arcTo(nums[0], nums[1], nums[2], nums[3] == 1, nums[4] == 1, abs(point{nums[5], nums[6]}))
		}
		lastCtrl = ctrl
		lastOp = map[byte]byte{'C': 'C', 'S': 'C', 'Q': 'Q', 'T': 'Q', 'Z': 'Z'}[cmd&^0x20]
	}
	return b.segs, nil
}

// box is an axis-aligned rectangle.
type box struct{ x0, y0, x1, y1 float64 }

// pathBounds is the tight bounding box of a path: curves count by their extremes, not their
// control points.
func pathBounds(segs []segment) box {
	bb := box{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
	add := func(p point) {
		bb.x0, bb.y0 = math.Min(bb.x0, p.x), math.Min(bb.y0, p.y)
		bb.x1, bb.y1 = math.Max(bb.x1, p.x), math.Max(bb.y1, p.y)
	}
	var cur point
	for _, s := range segs {
		switch s.op {
		case 'M', 'L':
			cur = s.p[0]
			add(cur)
		case 'Q':
			for _, t := range quadExtrema(cur, s.p[0], s.p[1]) {
				add(quadAt(cur, s.p[0], s.p[1], t))
			}
			cur = s.p[1]
			add(cur)
		case 'C':
			for _, t := range cubicExtrema(cur, s.p[0], s.p[1], s.p[2]) {
				add(cubicAt(cur, s.p[0], s.p[1], s.p[2], t))
			}
			cur = s.p[2]
			add(cur)
		}
	}
	return bb
}

func quadAt(p0, p1, p2 point, t float64) point {
	return p0.lerp(p1, t).lerp(p1.lerp(p2, t), t)
}

func cubicAt(p0, p1, p2, p3 point, t float64) point {
	a, b, c := p0.lerp(p1, t), p1.lerp(p2, t), p2.lerp(p3, t)
	return a.lerp(b, t).lerp(b.lerp(c, t), t)
}

// quadExtrema returns the parameters in (0, 1) where a quadratic's x or y turns.
func quadExtrema(p0, p1, p2 point) []float64 {
	var ts []float64
	for _, v := range [][3]float64{{p0.x, p1.x, p2.x}, {p0.y, p1.y, p2.y}} {
		if den := v[0] - 2*v[1] + v[2]; den != 0 {
			if t := (v[0] - v[1]) / den; t > 0 && t < 1 {
				ts = append(ts, t)
			}
		}
	}
	return ts
}

// cubicExtrema returns the parameters in (0, 1) where a cubic's x or y turns.
func cubicExtrema(p0, p1, p2, p3 point) []float64 {
	var ts []float64
	for _, v := range [][4]float64{{p0.x, p1.x, p2.x, p3.x}, {p0.y, p1.y, p2.y, p3.y}} {
		// The derivative divided by 3: a t² + b t + c.
		a := -v[0] + 3*v[1] - 3*v[2] + v[3]
		b := 2 * (v[0] - 2*v[1] + v[2])
		c := v[1] - v[0]
		if math.Abs(a) < 1e-12 {
			if b != 0 {
				ts = append(ts, -c/b)
			}
			continue
		}
		disc := b*b - 4*a*c
		if disc < 0 {
			continue
		}
		sq := math.Sqrt(disc)
		ts = append(ts, (-b+sq)/(2*a), (-b-sq)/(2*a))
	}
	out := ts[:0]
	for _, t := range ts {
		if t > 0 && t < 1 {
			out = append(out, t)
		}
	}
	return out
}

// poly is a subpath flattened to straight lines.
type poly struct {
	pts []point
	// smooth[i] marks a point inside a flattened curve, where a stroke turns smoothly and
	// takes a round join whatever the declared one.
	smooth []bool
	closed bool
	drawn  bool // the subpath has a command after its moveto (a zero-length one draws caps)
}

// flatten turns a path into polylines whose curves stay within tol of the true curves.
func flatten(segs []segment, tol float64) []poly {
	var out []poly
	var cur poly
	var at, start point
	flush := func() {
		if len(cur.pts) > 0 {
			out = append(out, cur)
		}
		cur = poly{}
	}
	add := func(p point, smooth bool) {
		cur.pts = append(cur.pts, p)
		cur.smooth = append(cur.smooth, smooth)
	}
	for _, s := range segs {
		switch s.op {
		case 'M':
			flush()
			at, start = s.p[0], s.p[0]
			add(at, false)
		case 'L':
			add(s.p[0], false)
			at = s.p[0]
			cur.drawn = true
		case 'Q':
			n := subdivisions(0.25*at.sub(s.p[0].scale(2)).add(s.p[1]).length(), tol)
			for i := 1; i <= n; i++ {
				add(quadAt(at, s.p[0], s.p[1], float64(i)/float64(n)), i < n)
			}
			at = s.p[1]
			cur.pts[len(cur.pts)-1] = at // exact end point
			cur.drawn = true
		case 'C':
			m := math.Max(at.sub(s.p[0].scale(2)).add(s.p[1]).length(), s.p[0].sub(s.p[1].scale(2)).add(s.p[2]).length())
			n := subdivisions(0.75*m, tol)
			for i := 1; i <= n; i++ {
				add(cubicAt(at, s.p[0], s.p[1], s.p[2], float64(i)/float64(n)), i < n)
			}
			at = s.p[2]
			cur.pts[len(cur.pts)-1] = at
			cur.drawn = true
		case 'Z':
			cur.closed = true
			cur.drawn = true
			at = start
			flush()
		}
	}
	flush()
	return out
}

// subdivisions is Wang's bound: how many equal steps keep a curve within tol, given the curve's
// largest second difference already scaled by its degree factor.
func subdivisions(m, tol float64) int {
	n := int(math.Ceil(math.Sqrt(m / tol)))
	return min(max(n, 1), 1024)
}
