package main

import "math"

// strokeStyle is what the stroker needs from a shape, in user space.
type strokeStyle struct {
	halfWidth  float64
	lineCap    string
	lineJoin   string
	miterLimit float64
	tol        float64 // how far a round join or cap may stray from the true circle
}

// strokeOutline returns polygons whose union is the stroke of the polylines: one quadrilateral
// per segment, a wedge for each join and a piece for each cap. Neighbouring pieces share their
// edges exactly (the same floating-point corners), and every polygon is wound the same way, so
// filling them together with the non-zero rule gives the union with no seams.
func strokeOutline(polys []poly, st strokeStyle) [][]point {
	var out [][]point
	emit := func(pts ...point) {
		if a := signedArea(pts); a < 0 {
			for i, j := 0, len(pts)-1; i < j; i, j = i+1, j-1 {
				pts[i], pts[j] = pts[j], pts[i]
			}
		} else if a == 0 {
			return
		}
		out = append(out, pts)
	}
	hw := st.halfWidth
	for _, p := range polys {
		// Points closer than a small fraction of the tolerance merge: a segment that short has
		// no reliable direction, and its joins could bulge out in any direction.
		eps := st.tol / 16
		pts, smooth := dedupe(p, eps)
		if p.closed && len(pts) > 1 && pts[0].sub(pts[len(pts)-1]).length() <= eps {
			pts, smooth = pts[:len(pts)-1], smooth[:len(smooth)-1]
			smooth[0] = false
		}
		if len(pts) == 1 {
			if p.drawn {
				dot(pts[0], st, emit)
			}
			continue
		}
		n := len(pts)
		segs := n - 1
		if p.closed {
			segs = n
		}
		// dirs[i] and offs[i] belong to the segment from pts[i] to pts[(i+1)%n]: its unit
		// direction and its left normal times the half width.
		dirs := make([]point, segs)
		offs := make([]point, segs)
		for i := range segs {
			d := pts[(i+1)%n].sub(pts[i])
			d = d.scale(1 / d.length())
			dirs[i] = d
			offs[i] = point{-d.y * hw, d.x * hw}
			a, b := pts[i], pts[(i+1)%n]
			emit(a.add(offs[i]), b.add(offs[i]), b.sub(offs[i]), a.sub(offs[i]))
		}
		for v := range n {
			prev, next := v-1, v
			if !p.closed && (v == 0 || v == n-1) {
				continue
			}
			if v == 0 {
				prev = segs - 1
			}
			join := st.lineJoin
			if smooth[v] {
				join = "round"
			}
			joinWedge(pts[v], dirs[prev], dirs[next], offs[prev], offs[next], join, st, emit)
		}
		if !p.closed {
			capPiece(pts[0], dirs[0].scale(-1), offs[0].scale(-1), st, emit)
			capPiece(pts[n-1], dirs[segs-1], offs[segs-1], st, emit)
		}
	}
	return out
}

// dedupe drops points within eps of the one before.
func dedupe(p poly, eps float64) ([]point, []bool) {
	pts := []point{p.pts[0]}
	smooth := []bool{p.smooth[0]}
	for i := 1; i < len(p.pts); i++ {
		if p.pts[i].sub(pts[len(pts)-1]).length() <= eps {
			smooth[len(smooth)-1] = smooth[len(smooth)-1] && p.smooth[i]
			continue
		}
		pts = append(pts, p.pts[i])
		smooth = append(smooth, p.smooth[i])
	}
	return pts, smooth
}

// joinWedge fills the gap on the outside of a corner at v between a segment coming in along d1
// and one going out along d2 (o1 and o2 are their left offsets).
func joinWedge(v, d1, d2, o1, o2 point, join string, st strokeStyle, emit func(...point)) {
	cross, dot := d1.cross(d2), d1.dot(d2)
	if math.Abs(cross) < 1e-12 && dot > 0 {
		return // straight on
	}
	// A left turn (cross > 0) opens the gap on the right.
	p1, p2, outer := v.add(o1), v.add(o2), 1.0
	if cross > 0 {
		p1, p2, outer = v.sub(o1), v.sub(o2), -1
	}
	switch join {
	case "round":
		arc := []point{v, p1}
		mid := d1 // a U-turn bulges straight ahead
		if math.Abs(cross) >= 1e-12 {
			mid = point{}
		}
		arc = append(arc, arcPoints(v, p1, p2, mid, st)...)
		emit(append(arc, p2)...)
	case "miter":
		// The miter's length over the stroke width is 1/cos(turn/2) = sqrt(2/(1+dot)).
		if 1+dot > 1e-12 && math.Sqrt(2/(1+dot)) <= st.miterLimit {
			n := o1.add(o2).scale(outer / (1 + o1.dot(o2)/(st.halfWidth*st.halfWidth)))
			emit(v, p1, v.add(n), p2)
			return
		}
		emit(v, p1, p2)
	default: // bevel
		emit(v, p1, p2)
	}
}

// capPiece draws the cap at end point e of a segment heading out along d, whose left offset
// is o.
func capPiece(e, d, o point, st strokeStyle, emit func(...point)) {
	switch st.lineCap {
	case "square":
		ext := d.scale(st.halfWidth)
		emit(e.add(o), e.add(o).add(ext), e.sub(o).add(ext), e.sub(o))
	case "round":
		pts := append([]point{e, e.add(o)}, arcPoints(e, e.add(o), e.sub(o), d, st)...)
		emit(append(pts, e.sub(o))...)
	}
}

// dot draws a zero-length subpath: a round cap makes a disc and a square cap a square.
func dot(c point, st strokeStyle, emit func(...point)) {
	hw := st.halfWidth
	switch st.lineCap {
	case "round":
		top, bottom := point{c.x, c.y - hw}, point{c.x, c.y + hw}
		pts := append([]point{top}, arcPoints(c, top, bottom, point{1, 0}, st)...)
		pts = append(pts, bottom)
		emit(append(pts, arcPoints(c, bottom, top, point{-1, 0}, st)...)...)
	case "square":
		emit(point{c.x - hw, c.y - hw}, point{c.x + hw, c.y - hw}, point{c.x + hw, c.y + hw}, point{c.x - hw, c.y + hw})
	}
}

// arcPoints returns the points strictly between a and b on the circle around c through them,
// going the short way, or through the side of dir when dir is not zero (half circles).
func arcPoints(c, a, b, dir point, st strokeStyle) []point {
	u, w := a.sub(c), b.sub(c)
	r := u.length()
	total := math.Atan2(u.cross(w), u.dot(w)) // signed, in (-π, π]
	if dir != (point{}) {
		// Half circle: turn the way that passes through dir.
		total = math.Pi
		if u.cross(dir) < 0 {
			total = -math.Pi
		}
	}
	step := math.Pi / 2
	if st.tol < r {
		step = 2 * math.Acos(1-st.tol/r)
	}
	steps := math.Ceil(math.Abs(total) / step)
	if !(steps <= 256) { // also NaN, from absurd sizes
		steps = 256
	}
	n := int(steps)
	var pts []point
	for i := 1; i < n; i++ {
		s, co := math.Sincos(total * float64(i) / float64(n))
		pts = append(pts, c.add(point{u.x*co - u.y*s, u.x*s + u.y*co}))
	}
	return pts
}

// signedArea is twice the polygon's signed area (the shoelace formula).
func signedArea(pts []point) float64 {
	var a float64
	for i, p := range pts {
		q := pts[(i+1)%len(pts)]
		a += p.cross(q)
	}
	return a
}
