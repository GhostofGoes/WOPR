package ending

import (
	"strings"

	"github.com/GhostofGoes/WOPR/internal/assets"
	"github.com/GhostofGoes/WOPR/internal/games/tictactoe"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// View implements proto.Program.
func (g *Game) View(c *proto.Canvas) {
	switch g.phase {
	case selfPlay:
		g.drawBoards(c)
	case montage:
		g.drawMontage(c)
	case greeting, done:
	}
}

// Self-play geometry: board 0 is the big board in the middle, as on NORAD's main screen;
// the others stand three high down each side, like the screens beside it.
const (
	smallRows = 5 // a small board: marks one row high
	sideGap   = 6 // rows from one small board to the next
	sideInset = 3 // columns from the edge to a side board
	codeGap   = 2 // rows from the big board down to the launch code
	perSide   = (boards - 1) / 2
)

// drawBoards draws the self-play games and, under the big board, the launch code.
func (g *Game) drawBoards(c *proto.Canvas) {
	x0 := max((c.W-80)/2, 0)
	sw, _ := tictactoe.Size(smallRows)
	bw, bh := tictactoe.Size(tictactoe.BigRows)
	sideH := (perSide-1)*sideGap + smallRows // the launch code ends level with the side boards
	top := max((c.H-sideH)/2, 0)
	for i, b := range g.boards {
		x, y, h := x0+(80-bw)/2, top+(sideH-bh)/2, tictactoe.BigRows
		if i > 0 {
			col, row := (i-1)%2, (i-1)/2
			x, y, h = x0+sideInset, top+row*sideGap, smallRows
			if col == 1 {
				x = x0 + 80 - sideInset - sw
			}
		}
		b.Draw(c, x, y, h, g.last[i], false)
	}
	code := lineCode[0].Text
	n := g.cracked()
	shown := code[:n] + strings.Repeat("_", len(code)-n)
	line := lineCodeLabel[0].Text + shown[:3] + " " + shown[3:7] + " " + shown[7:]
	y := top + (sideH-bh)/2 + bh + codeGap
	c.Put((c.W-len(line))/2, y, line, proto.StyleAlert, proto.AttrBold)
}

// drawMontage is the scenario table, scrolling up as WOPR runs each strategy.
func (g *Game) drawMontage(c *proto.Canvas) {
	head := lineStrategy.Texts()
	c.Put(10, 0, head[0], proto.StyleLabel, proto.AttrUnderline)
	c.Put(52, 0, head[1], proto.StyleLabel, proto.AttrUnderline)
	rows := c.H - 2
	first := max(g.shown-rows, 0)
	for i := first; i < g.shown && i < len(assets.Scenarios); i++ {
		y := 2 + i - first
		c.Put(10, y, assets.Scenarios[i].Text, proto.StyleText, 0)
		c.Put(52, y, head[2], proto.StyleDim, 0)
	}
}
