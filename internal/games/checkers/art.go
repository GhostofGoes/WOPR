package checkers

import (
	"strconv"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/games/board"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// The panel's title card, drawn for this project: a king, two men stacked under a crown,
// and the name.
var artTitle = script.Orig(
	`     *    *    *`,
	`    / \  / \  / \`,
	` .-/   \/   \/   \-.`,
	`(  |_____________|  )`,
	`|'-._____________.-'|`,
	`|                   |`,
	`|'-._____________.-'|`,
	`|                   |`,
	` '-._____________.-'`,
	``,
	`   C H E C K E R S`,
)

// Panel labels.
var (
	panelWOPR  = script.Orig("WOPR")
	panelYou   = script.Orig("YOU")
	panelSides = script.Orig("BLACK", "WHITE")
	panelMen   = script.Orig("MEN")
	panelKings = script.Orig("KINGS")
	panelLast  = script.Orig("LAST")
	panelMove  = script.Orig("MOVE")
	panelMore  = script.Orig("X..X") // between the first and last squares of a jump too long to show
)

// Panel geometry: the title card at the left, the board in the middle, the two sides and the
// state of play at the right, each side level with its own end of the board.
const (
	titleX = 2
	boardX = (80 - board.Width) / 2
	infoX  = boardX + board.Width + 3
	valueX = infoX + 7
	infoW  = 80 - valueX
)

// View implements proto.Program: the title card, then the board and the play beside it.
func (g *Game) View(c *proto.Canvas) {
	x := max((c.W-80)/2, 0)
	for i, l := range artTitle {
		style := proto.StyleDim
		if i == len(artTitle)-1 {
			style = proto.StyleLabel
		}
		c.Put(x+titleX, i, l.Text, style, 0)
	}
	g.drawBoard(c, x)
}

// drawBoard draws the board (men as discs in lower case, kings in capitals, the last move
// bracketed) and, beside it, what each side has left, the move number and the last move.
func (g *Game) drawBoard(c *proto.Canvas, x int) {
	taken := g.last.taken()
	board.Draw(c, x+boardX, 0, func(file, rank int) board.Glyph {
		if !board.Dark(file, rank) {
			return board.Glyph{}
		}
		sq := at(file, rank)
		gl := board.Glyph{Disc: true, Mark: g.last.n > 0 && (sq == g.last.from() || sq == g.last.to())}
		if taken[sq] && g.pos.sq[sq] == empty {
			return board.Glyph{R: 'x', S: proto.StyleAccent} // a man the last move jumped
		}
		switch g.pos.sq[sq] {
		case blackMan:
			gl.R, gl.S = 'b', proto.StyleText
		case blackKing: // the capital says king; bold would make it look like White without colour
			gl.R, gl.S = 'B', proto.StyleText
		case whiteMan:
			gl.R, gl.S = 'w', proto.StyleBright
		case whiteKing:
			gl.R, gl.S = 'W', proto.StyleBright
		}
		if g.last.n > 0 && sq == g.last.to() {
			gl.A |= proto.AttrUnderline
		}
		return gl
	})

	side := func(y int, who, colour string) {
		c.Put(x+infoX, y, who, proto.StyleLabel, 0)
		c.Put(x+valueX, y, colour, proto.StyleText, 0)
	}
	side(2, panelWOPR[0].Text, panelSides[1].Text)
	g.putCount(c, x, 3, White)
	c.Put(x+infoX, 5, panelMove[0].Text, proto.StyleDim, 0)
	c.Put(x+valueX, 5, strconv.Itoa(g.moves+1), proto.StyleText, 0)
	if g.last.n > 0 {
		c.Put(x+infoX, 6, panelLast[0].Text, proto.StyleDim, 0)
		c.Put(x+valueX, 6, lastText(g.last), proto.StyleText, 0)
	}
	g.putCount(c, x, 8, Black)
	side(9, panelYou[0].Text, panelSides[0].Text)
}

// putCount shows how many men and kings side has left, on row y.
func (g *Game) putCount(c *proto.Canvas, x, y int, side int8) {
	men, kings := 0, 0
	for _, p := range g.pos.sq {
		if sideOf(p) == side {
			if isKing(p) {
				kings++
			} else {
				men++
			}
		}
	}
	c.Put(x+infoX, y, panelMen[0].Text, proto.StyleDim, 0)
	end := c.Put(x+valueX, y, strconv.Itoa(men), proto.StyleText, 0)
	if kings > 0 {
		end = c.Put(end+2, y, panelKings[0].Text, proto.StyleDim, 0)
		c.Put(end+1, y, strconv.Itoa(kings), proto.StyleText, 0)
	}
}

// taken reports the squares a jump passed over: the men it took.
func (m Move) taken() (out [64]bool) {
	if m.n < 2 || !m.jump() {
		return out
	}
	for i := 1; i < int(m.n); i++ {
		out[(int(m.path[i-1])+int(m.path[i]))/2] = true
	}
	return out
}

// lastText is a move as the console writes it, shortened to its ends when a long jump
// would not fit beside the board.
func lastText(m Move) string {
	s := strings.ToUpper(m.String())
	if len(s) <= infoW {
		return s
	}
	return strings.ToUpper(name(m.from())) + panelMore[0].Text + strings.ToUpper(name(m.to()))
}
