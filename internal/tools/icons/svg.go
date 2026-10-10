package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"
	"strings"
)

// The SVG subset the icon sources may use. Anything else is an error that names the file and
// line, so a design this renderer would draw differently from a browser fails here rather
// than in a package.
//
//   - Elements: svg (the root only), g, defs (holding gradients), rect, circle, ellipse, line,
//     polyline, polygon, path (every command), linearGradient and radialGradient with stop,
//     and title and desc, which are kept but not drawn. Comments are dropped.
//   - Presentation attributes, on svg, g and the shapes: fill, fill-opacity, fill-rule,
//     stroke, stroke-width, stroke-opacity, stroke-linecap, stroke-linejoin,
//     stroke-miterlimit, and opacity (on a shape or a group, drawn as one layer). transform
//     on g and the shapes. Colours are #rgb, #rrggbb, rgb(...), none, or one of the sixteen
//     basic colour names; a paint may be url(#gradient).
//   - Gradients: x1, y1, x2, y2 or cx, cy, r, fx, fy, with gradientUnits, gradientTransform
//     and spreadMethod; stops with offset, stop-color and stop-opacity.
//   - Numbers are user units, optionally with px. Percentages only in gradients.
//   - No text, filter, mask, clipPath, pattern, image, use, style, class, dashes, other CSS
//     units, or editor metadata (attributes in other namespaces).
//   - The root has a square viewBox.

const svgNS = "http://www.w3.org/2000/svg"

// presentation attributes; all but opacity inherit.
var presentation = []string{
	"fill", "fill-opacity", "fill-rule", "stroke", "stroke-width", "stroke-opacity",
	"stroke-linecap", "stroke-linejoin", "stroke-miterlimit", "opacity",
}

var (
	shapes    = []string{"rect", "circle", "ellipse", "line", "polyline", "polygon", "path"}
	gradients = []string{"linearGradient", "radialGradient"}
	metadata  = []string{"title", "desc"}
	drawable  = slices.Concat([]string{"g", "defs"}, shapes, gradients, metadata)
)

// grammar is each element's own attributes (besides id, and the presentation attributes
// where styled says so) and the elements it may contain.
var grammar = map[string]struct {
	attrs  []string
	kids   []string
	styled bool
}{
	"svg":            {[]string{"viewBox", "width", "height", "version"}, drawable, true},
	"g":              {[]string{"transform"}, drawable, true},
	"defs":           {nil, gradients, false},
	"rect":           {[]string{"transform", "x", "y", "width", "height", "rx", "ry"}, metadata, true},
	"circle":         {[]string{"transform", "cx", "cy", "r"}, metadata, true},
	"ellipse":        {[]string{"transform", "cx", "cy", "rx", "ry"}, metadata, true},
	"line":           {[]string{"transform", "x1", "y1", "x2", "y2"}, metadata, true},
	"polyline":       {[]string{"transform", "points"}, metadata, true},
	"polygon":        {[]string{"transform", "points"}, metadata, true},
	"path":           {[]string{"transform", "d"}, metadata, true},
	"linearGradient": {[]string{"x1", "y1", "x2", "y2", "gradientUnits", "gradientTransform", "spreadMethod"}, []string{"stop"}, false},
	"radialGradient": {[]string{"cx", "cy", "r", "fx", "fy", "gradientUnits", "gradientTransform", "spreadMethod"}, []string{"stop"}, false},
	"stop":           {[]string{"offset", "stop-color", "stop-opacity"}, nil, false},
	"title":          {nil, nil, false},
	"desc":           {nil, nil, false},
}

// element is one parsed element, with its attributes as written.
type element struct {
	name  string
	attrs []xml.Attr
	kids  []*element
	text  string // title and desc only
	line  int
}

func (e *element) attr(name string) (string, bool) {
	for _, a := range e.attrs {
		if a.Name.Local == name {
			return strings.TrimSpace(a.Value), true
		}
	}
	return "", false
}

// sourceError is an error at a line of a source file.
type sourceError struct {
	file string
	line int
	err  error
}

func (e *sourceError) Error() string {
	if e.line == 0 {
		return fmt.Sprintf("%s: %v", e.file, e.err)
	}
	return fmt.Sprintf("%s:%d: %v", e.file, e.line, e.err)
}

func (e *sourceError) Unwrap() error { return e.err }

// parseXML reads an SVG source into elements, enforcing the subset's grammar.
func parseXML(file string, data []byte) (*element, error) {
	d := xml.NewDecoder(bytes.NewReader(data))
	d.Strict = true
	fail := func(err error) error {
		line, _ := d.InputPos()
		return &sourceError{file, line, err}
	}
	var root *element
	var stack []*element
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			var se *xml.SyntaxError
			if errors.As(err, &se) {
				return nil, &sourceError{file, se.Line, errors.New(se.Msg)}
			}
			return nil, fail(err)
		}
		switch t := tok.(type) {
		case xml.ProcInst:
			if t.Target != "xml" || root != nil {
				return nil, fail(fmt.Errorf("processing instruction <?%s?> is not supported", t.Target))
			}
		case xml.Directive:
			return nil, fail(errors.New("<!DOCTYPE> and other directives are not supported"))
		case xml.Comment:
		case xml.CharData:
			if len(stack) > 0 && slices.Contains(metadata, stack[len(stack)-1].name) {
				stack[len(stack)-1].text += string(t)
			} else if len(bytes.TrimSpace(t)) > 0 {
				return nil, fail(fmt.Errorf("text %q is not supported outside title and desc", bytes.TrimSpace(t)))
			}
		case xml.StartElement:
			line, _ := d.InputPos()
			e, err := newElement(t, line, root == nil)
			if err != nil {
				return nil, fail(err)
			}
			switch {
			case root == nil:
				root = e
			case len(stack) == 0:
				return nil, fail(errors.New("only one root element is allowed"))
			default:
				parent := stack[len(stack)-1]
				if !slices.Contains(grammar[parent.name].kids, e.name) {
					return nil, fail(fmt.Errorf("<%s> may not contain <%s>", parent.name, e.name))
				}
				parent.kids = append(parent.kids, e)
			}
			stack = append(stack, e)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		}
	}
	if root == nil {
		return nil, &sourceError{file, 0, errors.New("no <svg> element")}
	}
	return root, nil
}

// newElement checks one start tag against the grammar.
func newElement(t xml.StartElement, line int, isRoot bool) (*element, error) {
	name := t.Name.Local
	if t.Name.Space != svgNS {
		if t.Name.Space == "" {
			return nil, fmt.Errorf("<%s> is not in the SVG namespace; the root needs xmlns=%q", name, svgNS)
		}
		return nil, fmt.Errorf("<%s> in namespace %s is not supported (remove editor metadata)", name, t.Name.Space)
	}
	g, ok := grammar[name]
	if !ok {
		return nil, fmt.Errorf("<%s> is not supported; icons may use only svg, g, defs, %s, %s, stop, title and desc",
			name, strings.Join(shapes, ", "), strings.Join(gradients, ", "))
	}
	if isRoot != (name == "svg") {
		if isRoot {
			return nil, fmt.Errorf("the root element is <%s>, not <svg>", name)
		}
		return nil, errors.New("<svg> is allowed only as the root element")
	}
	e := &element{name: name, line: line}
	for _, a := range t.Attr {
		switch {
		case a.Name.Space == "" && a.Name.Local == "xmlns" && isRoot:
			continue // the decoder already checked it names the SVG namespace
		case a.Name.Space == "xmlns" || a.Name.Space == "" && a.Name.Local == "xmlns":
			return nil, fmt.Errorf("namespace declaration %s on <%s> is not supported (remove editor metadata)", attrName(a), name)
		case a.Name.Space != "":
			return nil, fmt.Errorf("attribute %s on <%s> is not supported (remove editor metadata and xlink)", attrName(a), name)
		case a.Name.Local == "id",
			slices.Contains(g.attrs, a.Name.Local),
			g.styled && slices.Contains(presentation, a.Name.Local):
			e.attrs = append(e.attrs, a)
		default:
			return nil, fmt.Errorf("attribute %s on <%s> is not supported", a.Name.Local, name)
		}
	}
	return e, nil
}

func attrName(a xml.Attr) string {
	switch a.Name.Space {
	case "":
		return a.Name.Local
	case "http://www.w3.org/XML/1998/namespace":
		return "xml:" + a.Name.Local
	}
	return a.Name.Space + ":" + a.Name.Local
}

// minify writes the source again without comments, the XML declaration, or whitespace between
// elements, and with each attribute's whitespace collapsed: the scalable icons, which carry
// nothing an icon validator could object to.
func minify(root *element) []byte {
	var b bytes.Buffer
	var write func(e *element)
	write = func(e *element) {
		b.WriteString("<" + e.name)
		if e == root {
			b.WriteString(` xmlns="` + svgNS + `"`)
		}
		for _, a := range e.attrs {
			b.WriteString(" " + a.Name.Local + `="`)
			_ = xml.EscapeText(&b, []byte(strings.Join(strings.Fields(a.Value), " ")))
			b.WriteString(`"`)
		}
		text := strings.Join(strings.Fields(e.text), " ")
		if len(e.kids) == 0 && text == "" {
			b.WriteString("/>")
			return
		}
		b.WriteString(">")
		_ = xml.EscapeText(&b, []byte(text))
		for _, k := range e.kids {
			write(k)
		}
		b.WriteString("</" + e.name + ">")
	}
	write(root)
	b.WriteString("\n")
	return b.Bytes()
}

// doc is a parsed and resolved source, ready to draw at any size.
type doc struct {
	root  *element
	vb    [4]float64 // viewBox: x, y, width, height (width == height)
	items []item
}

// item is a shape, or a group drawn as one layer because its opacity is below 1.
type item struct {
	shape   *shape
	group   []item
	opacity float64
}

// shape is one drawable element with its style resolved.
type shape struct {
	path        []segment // user space
	ctm         matrix    // user space to viewBox space
	bbox        box       // of the geometry, in user space (objectBoundingBox gradients)
	fill        paint
	evenOdd     bool
	stroke      paint
	strokeWidth float64
	lineCap     string
	lineJoin    string
	miterLimit  float64
	opacity     float64
}

// style is the inherited presentation state.
type style struct {
	fill, stroke               paint
	fillOpacity, strokeOpacity float64
	evenOdd                    bool
	strokeWidth, miterLimit    float64
	lineCap, lineJoin          string
}

// parseSVG reads, checks and resolves an icon source.
func parseSVG(file string, data []byte) (*doc, error) {
	root, err := parseXML(file, data)
	if err != nil {
		return nil, err
	}
	at := func(e *element, err error) error { return &sourceError{file, e.line, err} }
	d := &doc{root: root}
	vb, ok := root.attr("viewBox")
	if !ok {
		return nil, at(root, errors.New("the root <svg> needs a viewBox"))
	}
	nums, err := parseNumbers(vb)
	if err != nil || len(nums) != 4 {
		return nil, at(root, fmt.Errorf("viewBox %q is not four numbers", vb))
	}
	if nums[2] <= 0 || nums[2] != nums[3] {
		return nil, at(root, fmt.Errorf("viewBox %q is not a square", vb))
	}
	d.vb = [4]float64(nums)
	for _, name := range []string{"width", "height"} {
		if v, ok := root.attr(name); ok {
			if _, err := parseLength(v); err != nil {
				return nil, at(root, fmt.Errorf("%s: %w", name, err))
			}
		}
	}

	// Gradients first: a paint may name one defined later in the file.
	grads := map[string]*gradient{}
	ids := map[string]bool{}
	var walkErr error
	var collect func(e *element)
	collect = func(e *element) {
		if id, ok := e.attr("id"); ok && walkErr == nil {
			if ids[id] {
				walkErr = at(e, fmt.Errorf("duplicate id %q", id))
				return
			}
			ids[id] = true
			if slices.Contains(gradients, e.name) {
				g, err := parseGradient(e, d.vb[2])
				if err != nil {
					walkErr = at(e, err)
					return
				}
				grads[id] = g
			}
		} else if slices.Contains(gradients, e.name) && walkErr == nil {
			walkErr = at(e, fmt.Errorf("<%s> needs an id", e.name))
		}
		for _, k := range e.kids {
			collect(k)
		}
	}
	collect(root)
	if walkErr != nil {
		return nil, walkErr
	}

	initial := style{
		fill:          paint{color: rgba{0, 0, 0, 1}},
		fillOpacity:   1,
		strokeOpacity: 1,
		strokeWidth:   1,
		miterLimit:    4,
		lineCap:       "butt",
		lineJoin:      "miter",
	}
	initial.stroke.none = true
	var resolve func(e *element, st style, ctm matrix) ([]item, error)
	resolve = func(e *element, st style, ctm matrix) ([]item, error) {
		opacity := 1.0
		if v, ok := e.attr("transform"); ok {
			t, err := parseTransform(v)
			if err != nil {
				return nil, at(e, fmt.Errorf("transform: %w", err))
			}
			ctm = ctm.mul(t)
		}
		if grammar[e.name].styled {
			var err error
			st, opacity, err = applyStyle(e, st, grads)
			if err != nil {
				return nil, at(e, err)
			}
		}
		switch e.name {
		case "svg", "g":
			var items []item
			for _, k := range e.kids {
				sub, err := resolve(k, st, ctm)
				if err != nil {
					return nil, err
				}
				items = append(items, sub...)
			}
			if opacity < 1 {
				return []item{{group: items, opacity: opacity}}, nil
			}
			return items, nil
		case "defs", "linearGradient", "radialGradient", "stop", "title", "desc":
			return nil, nil
		}
		path, err := shapePath(e)
		if err != nil {
			return nil, at(e, err)
		}
		if len(path) == 0 {
			return nil, nil
		}
		s := &shape{
			path:        path,
			ctm:         ctm,
			bbox:        pathBounds(path),
			fill:        st.fill,
			evenOdd:     st.evenOdd,
			stroke:      st.stroke,
			strokeWidth: st.strokeWidth,
			lineCap:     st.lineCap,
			lineJoin:    st.lineJoin,
			miterLimit:  st.miterLimit,
			opacity:     opacity,
		}
		s.fill.opacity = st.fillOpacity
		s.stroke.opacity = st.strokeOpacity
		return []item{{shape: s}}, nil
	}
	d.items, err = resolve(root, initial, identity)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// applyStyle reads an element's presentation attributes over the inherited style. It returns
// the element's own opacity, which does not inherit.
func applyStyle(e *element, st style, grads map[string]*gradient) (style, float64, error) {
	opacity := 1.0
	for _, a := range e.attrs {
		v := strings.TrimSpace(a.Value)
		var err error
		switch a.Name.Local {
		case "fill":
			st.fill, err = parsePaint(v, grads)
		case "stroke":
			st.stroke, err = parsePaint(v, grads)
		case "fill-opacity":
			st.fillOpacity, err = parseOpacity(v)
		case "stroke-opacity":
			st.strokeOpacity, err = parseOpacity(v)
		case "opacity":
			opacity, err = parseOpacity(v)
		case "fill-rule":
			switch v {
			case "nonzero":
				st.evenOdd = false
			case "evenodd":
				st.evenOdd = true
			default:
				err = fmt.Errorf("fill-rule %q is not nonzero or evenodd", v)
			}
		case "stroke-width":
			st.strokeWidth, err = parseLength(v)
			if err == nil && st.strokeWidth < 0 {
				err = fmt.Errorf("negative stroke-width %q", v)
			}
		case "stroke-miterlimit":
			st.miterLimit, err = parseNumber(v)
			if err == nil && st.miterLimit < 1 {
				err = fmt.Errorf("stroke-miterlimit %q is below 1", v)
			}
		case "stroke-linecap":
			if !slices.Contains([]string{"butt", "round", "square"}, v) {
				err = fmt.Errorf("stroke-linecap %q is not butt, round or square", v)
			}
			st.lineCap = v
		case "stroke-linejoin":
			if !slices.Contains([]string{"miter", "round", "bevel"}, v) {
				err = fmt.Errorf("stroke-linejoin %q is not miter, round or bevel", v)
			}
			st.lineJoin = v
		}
		if err != nil {
			return st, 0, err
		}
	}
	return st, opacity, nil
}

// shapePath turns a shape element into path segments in user space.
func shapePath(e *element) ([]segment, error) {
	num := func(name string) (float64, bool, error) {
		v, ok := e.attr(name)
		if !ok {
			return 0, false, nil
		}
		f, err := parseLength(v)
		if err != nil {
			return 0, true, fmt.Errorf("%s: %w", name, err)
		}
		return f, true, nil
	}
	nums := func(names ...string) ([]float64, error) {
		out := make([]float64, len(names))
		for i, n := range names {
			var err error
			if out[i], _, err = num(n); err != nil {
				return nil, err
			}
		}
		return out, nil
	}
	var b pathBuilder
	switch e.name {
	case "path":
		v, _ := e.attr("d")
		return parsePathData(v)
	case "rect":
		v, err := nums("x", "y", "width", "height")
		if err != nil {
			return nil, err
		}
		rx, hasRX, err := num("rx")
		if err != nil {
			return nil, err
		}
		ry, hasRY, err := num("ry")
		if err != nil {
			return nil, err
		}
		if v[2] < 0 || v[3] < 0 || rx < 0 || ry < 0 {
			return nil, errors.New("negative size")
		}
		switch {
		case hasRX && !hasRY:
			ry = rx
		case hasRY && !hasRX:
			rx = ry
		}
		rect(&b, v[0], v[1], v[2], v[3], min(rx, v[2]/2), min(ry, v[3]/2))
	case "circle":
		v, err := nums("cx", "cy", "r")
		if err != nil {
			return nil, err
		}
		if v[2] < 0 {
			return nil, errors.New("negative radius")
		}
		ellipse(&b, v[0], v[1], v[2], v[2])
	case "ellipse":
		v, err := nums("cx", "cy", "rx", "ry")
		if err != nil {
			return nil, err
		}
		if v[2] < 0 || v[3] < 0 {
			return nil, errors.New("negative radius")
		}
		ellipse(&b, v[0], v[1], v[2], v[3])
	case "line":
		v, err := nums("x1", "y1", "x2", "y2")
		if err != nil {
			return nil, err
		}
		b.moveTo(point{v[0], v[1]})
		b.lineTo(point{v[2], v[3]})
	case "polyline", "polygon":
		v, _ := e.attr("points")
		p, err := parseNumbers(v)
		if err != nil {
			return nil, fmt.Errorf("points: %w", err)
		}
		if len(p)%2 != 0 {
			return nil, errors.New("points has an odd number of coordinates")
		}
		for i := 0; i+1 < len(p); i += 2 {
			if i == 0 {
				b.moveTo(point{p[0], p[1]})
			} else {
				b.lineTo(point{p[i], p[i+1]})
			}
		}
		if e.name == "polygon" && len(p) > 0 {
			b.close()
		}
	}
	return b.segs, nil
}

// rgba is a colour with straight (not premultiplied) components from 0 to 1.
type rgba struct{ r, g, b, a float64 }

// paint is a fill or stroke: none, a colour, or a gradient.
type paint struct {
	none    bool
	color   rgba
	grad    *gradient
	opacity float64 // fill-opacity or stroke-opacity
}

func parsePaint(v string, grads map[string]*gradient) (paint, error) {
	if v == "none" {
		return paint{none: true}, nil
	}
	if ref, ok := strings.CutPrefix(v, "url(#"); ok {
		id, ok := strings.CutSuffix(ref, ")")
		if !ok {
			return paint{}, fmt.Errorf("paint %q: write url(#id) with no fallback", v)
		}
		g, ok := grads[id]
		if !ok {
			return paint{}, fmt.Errorf("paint %q names no gradient", v)
		}
		return paint{grad: g}, nil
	}
	c, err := parseColor(v)
	return paint{color: c}, err
}

// basicColors are CSS's sixteen basic colour names.
var basicColors = map[string]string{
	"black": "#000000", "silver": "#c0c0c0", "gray": "#808080", "white": "#ffffff",
	"maroon": "#800000", "red": "#ff0000", "purple": "#800080", "fuchsia": "#ff00ff",
	"green": "#008000", "lime": "#00ff00", "olive": "#808000", "yellow": "#ffff00",
	"navy": "#000080", "blue": "#0000ff", "teal": "#008080", "aqua": "#00ffff",
}

func parseColor(v string) (rgba, error) {
	bad := fmt.Errorf("colour %q is not supported; write it as #rrggbb", v)
	if hex, ok := basicColors[strings.ToLower(v)]; ok {
		v = hex
	}
	if args, ok := strings.CutPrefix(v, "rgb("); ok {
		args, ok = strings.CutSuffix(args, ")")
		parts := strings.Split(args, ",")
		if !ok || len(parts) != 3 {
			return rgba{}, bad
		}
		var c [3]float64
		for i, p := range parts {
			p = strings.TrimSpace(p)
			pct := strings.HasSuffix(p, "%")
			f, err := parseNumber(strings.TrimSuffix(p, "%"))
			if err != nil {
				return rgba{}, bad
			}
			if pct {
				f = f / 100 * 255
			}
			c[i] = math.Max(0, math.Min(255, f)) / 255
		}
		return rgba{c[0], c[1], c[2], 1}, nil
	}
	hex, ok := strings.CutPrefix(v, "#")
	if !ok || len(hex) != 3 && len(hex) != 6 {
		return rgba{}, bad
	}
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	var c [3]float64
	for i := range 3 {
		var n int
		for _, ch := range hex[2*i : 2*i+2] {
			var d int
			switch {
			case ch >= '0' && ch <= '9':
				d = int(ch - '0')
			case ch >= 'a' && ch <= 'f':
				d = int(ch-'a') + 10
			case ch >= 'A' && ch <= 'F':
				d = int(ch-'A') + 10
			default:
				return rgba{}, bad
			}
			n = n*16 + d
		}
		c[i] = float64(n) / 255
	}
	return rgba{c[0], c[1], c[2], 1}, nil
}

func parseOpacity(v string) (float64, error) {
	f, err := parseNumber(v)
	if err != nil {
		return 0, fmt.Errorf("opacity %q: %w", v, err)
	}
	return math.Max(0, math.Min(1, f)), nil
}

// gradient is a linear or radial gradient. Its coordinates are fractions of the bounding box
// (objectBoundingBox, the default) or user-space values (userSpaceOnUse).
type gradient struct {
	radial    bool
	userSpace bool
	transform matrix
	spread    string // pad, reflect or repeat
	x1, y1    float64
	x2, y2    float64
	cx, cy, r float64
	fx, fy    float64
	stops     []stop
}

type stop struct {
	offset float64
	color  rgba
}

func parseGradient(e *element, size float64) (*gradient, error) {
	g := &gradient{radial: e.name == "radialGradient", transform: identity, spread: "pad"}
	if v, ok := e.attr("gradientUnits"); ok {
		switch v {
		case "objectBoundingBox":
		case "userSpaceOnUse":
			g.userSpace = true
		default:
			return nil, fmt.Errorf("gradientUnits %q is not objectBoundingBox or userSpaceOnUse", v)
		}
	}
	if v, ok := e.attr("spreadMethod"); ok {
		if !slices.Contains([]string{"pad", "reflect", "repeat"}, v) {
			return nil, fmt.Errorf("spreadMethod %q is not pad, reflect or repeat", v)
		}
		g.spread = v
	}
	if v, ok := e.attr("gradientTransform"); ok {
		t, err := parseTransform(v)
		if err != nil {
			return nil, fmt.Errorf("gradientTransform: %w", err)
		}
		g.transform = t
	}
	// coord reads a coordinate, given as a number or a percentage. A percentage is of the
	// bounding box, or in user space of the (square) viewport.
	coord := func(name string, def float64) (float64, bool, error) {
		v, ok := e.attr(name)
		if !ok {
			return def, false, nil
		}
		pct := strings.HasSuffix(v, "%")
		f, err := parseLength(strings.TrimSuffix(v, "%"))
		if err != nil {
			return 0, true, fmt.Errorf("%s: %w", name, err)
		}
		switch {
		case pct && g.userSpace:
			f = f / 100 * size
		case pct:
			f /= 100
		}
		return f, true, nil
	}
	one := 1.0
	if g.userSpace {
		one = size
	}
	var err error
	if g.radial {
		var hasFX, hasFY bool
		for _, c := range []struct {
			name string
			def  float64
			dst  *float64
			set  *bool
		}{
			{"cx", one / 2, &g.cx, nil},
			{"cy", one / 2, &g.cy, nil},
			{"r", one / 2, &g.r, nil},
			{"fx", 0, &g.fx, &hasFX},
			{"fy", 0, &g.fy, &hasFY},
		} {
			var set bool
			if *c.dst, set, err = coord(c.name, c.def); err != nil {
				return nil, err
			}
			if c.set != nil {
				*c.set = set
			}
		}
		if !hasFX {
			g.fx = g.cx
		}
		if !hasFY {
			g.fy = g.cy
		}
		if g.r < 0 {
			return nil, errors.New("negative r")
		}
	} else {
		for _, c := range []struct {
			name string
			def  float64
			dst  *float64
		}{{"x1", 0, &g.x1}, {"y1", 0, &g.y1}, {"x2", one, &g.x2}, {"y2", 0, &g.y2}} {
			if *c.dst, _, err = coord(c.name, c.def); err != nil {
				return nil, err
			}
		}
	}
	last := 0.0
	for _, s := range e.kids {
		st := stop{color: rgba{0, 0, 0, 1}}
		if v, ok := s.attr("offset"); ok {
			pct := strings.HasSuffix(v, "%")
			if st.offset, err = parseNumber(strings.TrimSuffix(v, "%")); err != nil {
				return nil, fmt.Errorf("stop offset %q: %w", v, err)
			}
			if pct {
				st.offset /= 100
			}
		}
		// Each offset is clamped to 0..1 and to at least the one before it.
		st.offset = max(last, math.Min(1, st.offset))
		last = st.offset
		if v, ok := s.attr("stop-color"); ok {
			if st.color, err = parseColor(v); err != nil {
				return nil, err
			}
		}
		if v, ok := s.attr("stop-opacity"); ok {
			if st.color.a, err = parseOpacity(v); err != nil {
				return nil, err
			}
		}
		g.stops = append(g.stops, st)
	}
	return g, nil
}
