package ending

import (
	"strings"

	"github.com/GhostofGoes/WOPR/internal/assets"
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

// drawBoards draws the self-play games, five by two, and the launch code below them.
func (g *Game) drawBoards(c *proto.Canvas) {
	const cellW, gap = 11, 4
	left := (c.W - (5*cellW + 4*gap)) / 2
	for i, b := range g.boards {
		x := left + (i%5)*(cellW+gap)
		y := 1 + (i/5)*6
		for r, row := range b.Rows() {
			for j, ch := range row {
				style := proto.StyleDim
				switch ch {
				case 'X':
					style = proto.StyleBright
				case 'O':
					style = proto.StyleText
				case '|', '-', '+':
				default:
					ch = ' ' // an empty square shows nothing: nobody is choosing here
				}
				c.Set(x+j, y+r, proto.Cell{R: ch, S: style})
			}
		}
	}
	code := lineCode[0].Text
	n := g.cracked()
	shown := code[:n] + strings.Repeat("_", len(code)-n)
	line := lineCodeLabel[0].Text + shown[:3] + " " + shown[3:7] + " " + shown[7:]
	c.Put((c.W-len(line))/2, 13, line, proto.StyleAlert, proto.AttrBold)
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
