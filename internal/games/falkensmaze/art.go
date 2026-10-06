package falkensmaze

import (
	"fmt"

	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// PanelRows is the panel the maze is drawn in: the frame's title row, the maze's 2*mazeH+1
// rows, and the frame's foot, which carries the status. Start asks the host for it.
const PanelRows = 2*mazeH + 3

// The frame's title and the marks beside the maze, all original.
var (
	frameTitle = script.Orig("[ FALKEN'S MAZE ]", "[ FALKEN'S MAZE: EXIT FOUND ]")
	frameMarks = script.Orig("START ->", "<- EXIT", "==> OUT")
)

// artEscaped is the win screen, drawn for this project: ESCAPED in letters built from the
// maze's own walls, and the player walking out through a gap in the east wall.
var artEscaped = script.Orig(
	`      +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+`,
	`      |  +--+  +--+  +--+  +--+  +--+  +--+  +--.                 |`,
	`      |  |     |     |     |  |  |  |  |     |  |         @       |`,
	`      |  +--   +--+  |     +--+  +--+  +--   |  |        /|\  . . . . . . . =>`,
	`      |  |        |  |     |  |  |     |     |  |        / \      |`,
	`      |  +--+  +--+  +--+  +  +  +     +--+  +--'                 |`,
	`      +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+`,
)

// View implements proto.Program: the maze under fog (unseen cells are a dim dot) in a
// frame titled FALKEN'S MAZE, with START and EXIT marked beside it and the status along
// the frame's foot. A panel shorter than PanelRows drops the frame and keeps the rest. Once
// the player is out, the fog lifts, the walls all show and the player's trail is dotted.
func (g *Game) View(c *proto.Canvas) {
	framed := c.H >= PanelRows
	top := 0
	if framed {
		top = 1
	}
	x0 := (c.W - (3*mazeW + 1)) / 2
	g.drawMaze(c, x0, top)
	foot := top + 2*mazeH + 1
	if framed {
		g.drawFrame(c, foot)
	}

	// START points at the first cell, EXIT at the last; once out, the way out is open.
	c.Put(x0-len(frameMarks[0].Text)-1, top+1, frameMarks[0].Text, proto.StyleLabel, 0)
	exitMark := frameMarks[1].Text
	if g.won {
		exitMark = frameMarks[2].Text
	}
	c.Put(x0+3*mazeW+2, top+2*mazeH-1, exitMark, proto.StyleLabel, proto.AttrBold)

	moves := fill(panelMoves[0].Text, fmt.Sprint(g.moves))
	habits := fill(panelHabits[0].Text, fmt.Sprint(g.bias[turnLeft]), fmt.Sprint(g.bias[turnStraight]), fmt.Sprint(g.bias[turnRight]))
	walls := fill(panelWalls[0].Text, fmt.Sprint(g.reroutes))
	left, right := 0, c.W
	if framed { // the status sits in the frame's foot, a space either side of each part
		moves, habits, walls = " "+moves+" ", " "+habits+" ", " "+walls+" "
		left, right = 3, c.W-3
	}
	c.Put(left, foot, moves, proto.StyleText, 0)
	gap := right - len(walls) - (left + len(moves)) // the habits go midway between the others
	c.Put(left+len(moves)+(gap-len(habits))/2, foot, habits, proto.StyleDim, 0)
	c.Put(right-len(walls), foot, walls, proto.StyleText, 0)
}

// drawFrame draws the frame around the panel, with the title in its top edge; foot is the
// row of its bottom edge.
func (g *Game) drawFrame(c *proto.Canvas, foot int) {
	edge := func(x, y int, r rune) { c.Set(x, y, proto.Cell{R: r, S: proto.StyleDim}) }
	for x := 1; x < c.W-1; x++ {
		edge(x, 0, '-')
		edge(x, foot, '-')
	}
	for y := 1; y < foot; y++ {
		edge(0, y, '|')
		edge(c.W-1, y, '|')
	}
	edge(0, 0, '.')
	edge(c.W-1, 0, '.')
	edge(0, foot, '\'')
	edge(c.W-1, foot, '\'')
	title := frameTitle[0].Text
	if g.won {
		title = frameTitle[1].Text
	}
	c.Put((c.W-len(title))/2, 0, title, proto.StyleBright, proto.AttrBold)
}

// drawMaze draws the maze with its top-left corner at x0, y0: three columns and two rows a
// cell, walls only where the player has seen (everywhere, once out).
func (g *Game) drawMaze(c *proto.Canvas, x0, y0 int) {
	at := func(cx, cy int) int { return cy*mazeW + cx }
	seen := func(cx, cy int) bool {
		return cx >= 0 && cx < mazeW && cy >= 0 && cy < mazeH && (g.won || g.seen[at(cx, cy)])
	}
	for gy := 0; gy <= mazeH; gy++ { // the corners
		for gx := 0; gx <= mazeW; gx++ {
			if seen(gx-1, gy-1) || seen(gx, gy-1) || seen(gx-1, gy) || seen(gx, gy) {
				c.Set(x0+3*gx, y0+2*gy, proto.Cell{R: '+', S: proto.StyleDim})
			}
		}
	}
	for cy := range mazeH {
		for cx := range mazeW {
			cell := at(cx, cy)
			x, y := x0+3*cx, y0+2*cy
			if (seen(cx, cy) || seen(cx, cy-1)) && !g.m.open(cell, north) {
				c.Put(x+1, y, "--", proto.StyleText, 0)
			}
			if (seen(cx, cy) || seen(cx-1, cy)) && !g.m.open(cell, west) {
				c.Set(x, y+1, proto.Cell{R: '|', S: proto.StyleText})
			}
			if cx == mazeW-1 && seen(cx, cy) && (!g.won || cell != g.exit) { // the way out opens once the player is through
				c.Set(x+3, y+1, proto.Cell{R: '|', S: proto.StyleText})
			}
			if cy == mazeH-1 && seen(cx, cy) {
				c.Put(x+1, y+2, "--", proto.StyleText, 0)
			}
			switch {
			case cell == g.player:
				c.Set(x+1, y+1, proto.Cell{R: '@', S: proto.StyleBright, A: proto.AttrBold})
			case cell == g.exit:
				c.Put(x+1, y+1, "[]", proto.StyleAlert, proto.AttrBold)
			case g.won && g.visited[cell]:
				c.Set(x+1, y+1, proto.Cell{R: ':', S: proto.StyleAccent})
			case !g.won && !g.seen[cell]:
				c.Set(x+1, y+1, proto.Cell{R: '.', S: proto.StyleDim})
			}
		}
	}
}
