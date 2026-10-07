package falkensmaze

import "math/rand/v2"

// Directions, clockwise from north.
const (
	north = iota
	east
	south
	west
)

var (
	dx = [4]int{0, 1, 0, -1}
	dy = [4]int{-1, 0, 1, 0}
)

// maze is a grid of cells with passages between neighbours. It starts as a perfect maze
// (a spanning tree: exactly one path between any two cells), and re-routing keeps it one.
type maze struct {
	w, h  int
	east  []bool // a passage from cell c to its east neighbour
	south []bool // a passage from cell c to its south neighbour
}

func newMaze(w, h int) *maze {
	return &maze{w: w, h: h, east: make([]bool, w*h), south: make([]bool, w*h)}
}

func (m *maze) cells() int { return m.w * m.h }

// neighbour is the cell in direction d from c, if it is on the grid.
func (m *maze) neighbour(c, d int) (int, bool) {
	x, y := c%m.w+dx[d], c/m.w+dy[d]
	if x < 0 || x >= m.w || y < 0 || y >= m.h {
		return 0, false
	}
	return y*m.w + x, true
}

// open reports a passage from c in direction d.
func (m *maze) open(c, d int) bool {
	n, ok := m.neighbour(c, d)
	if !ok {
		return false
	}
	switch d {
	case east:
		return m.east[c]
	case south:
		return m.south[c]
	case west:
		return m.east[n]
	default:
		return m.south[n]
	}
}

func (m *maze) set(c, d int, v bool) {
	n, ok := m.neighbour(c, d)
	if !ok {
		return
	}
	switch d {
	case east:
		m.east[c] = v
	case south:
		m.south[c] = v
	case west:
		m.east[n] = v
	default:
		m.south[n] = v
	}
}

// generate carves a perfect maze with a randomised depth-first search from cell 0.
func (m *maze) generate(r *rand.Rand) {
	seen := make([]bool, m.cells())
	stack := []int{0}
	seen[0] = true
	for len(stack) > 0 {
		c := stack[len(stack)-1]
		var next []int
		for d := range 4 {
			if n, ok := m.neighbour(c, d); ok && !seen[n] {
				next = append(next, d)
			}
		}
		if len(next) == 0 {
			stack = stack[:len(stack)-1]
			continue
		}
		d := next[r.IntN(len(next))]
		n, _ := m.neighbour(c, d)
		m.set(c, d, true)
		seen[n] = true
		stack = append(stack, n)
	}
}

// path is the route from a to b through open passages (a first, b last), or nil.
func (m *maze) path(a, b int) []int {
	prev := make([]int, m.cells())
	for i := range prev {
		prev[i] = -1
	}
	prev[a] = a
	queue := []int{a}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		if c == b {
			var out []int
			for ; c != a; c = prev[c] {
				out = append(out, c)
			}
			out = append(out, a)
			for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
				out[i], out[j] = out[j], out[i]
			}
			return out
		}
		for d := range 4 {
			if n, ok := m.neighbour(c, d); ok && m.open(c, d) && prev[n] < 0 {
				prev[n] = c
				queue = append(queue, n)
			}
		}
	}
	return nil
}

// component marks the cells reachable from c.
func (m *maze) component(c int) []bool {
	in := make([]bool, m.cells())
	in[c] = true
	queue := []int{c}
	for len(queue) > 0 {
		x := queue[0]
		queue = queue[1:]
		for d := range 4 {
			if n, ok := m.neighbour(x, d); ok && m.open(x, d) && !in[n] {
				in[n] = true
				queue = append(queue, n)
			}
		}
	}
	return in
}

// passages counts the open passages; a perfect maze has cells-1.
func (m *maze) passages() int {
	n := 0
	for c := range m.cells() {
		if m.east[c] {
			n++
		}
		if m.south[c] {
			n++
		}
	}
	return n
}

// dirTo is the direction from cell a to its neighbour b.
func (m *maze) dirTo(a, b int) int {
	for d := range 4 {
		if n, ok := m.neighbour(a, d); ok && n == b {
			return d
		}
	}
	return -1
}

// exits counts the open passages from c.
func (m *maze) exits(c int) int {
	n := 0
	for d := range 4 {
		if m.open(c, d) {
			n++
		}
	}
	return n
}
