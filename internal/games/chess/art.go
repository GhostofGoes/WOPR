package chess

import (
	"strconv"
	"strings"

	cg "github.com/corentings/chess/v2"

	"github.com/GhostofGoes/WOPR/internal/games/board"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// The panel's title card, drawn for this project: a knight on its base, and the name.
var artTitle = script.Orig(
	`      /\/|`,
	`    _/   '-.`,
	`  _/  O     \`,
	` /           |`,
	`(__.--.      |`,
	`      /      |`,
	`     /       |`,
	`    /________\`,
	`   (__________)`,
	``,
	`   C H E S S`,
)

// Panel labels.
var (
	panelWOPR  = script.Orig("WOPR")
	panelYou   = script.Orig("YOU")
	panelSides = script.Orig("WHITE", "BLACK")
	panelTaken = script.Orig("TAKEN")
	panelLast  = script.Orig("LAST")
	panelMove  = script.Orig("MOVE")
	panelCheck = script.Orig("CHECK")
)

// Panel geometry: the title card at the left, the board in the middle, the two sides and the
// state of play at the right, each side level with its own end of the board. The board's
// squares are two columns by one row, which terminal cells show square (board.DrawCompact).
const (
	titleX = 7 // the card ends seven columns short of the board
	boardX = (80 - board.CompactWidth) / 2
	infoX  = boardX + board.CompactWidth + 3
	valueX = infoX + 7
	infoW  = 80 - valueX
)

// View implements proto.Program: the title card, the board (White in capitals, Black in
// lower case, the last move marked: [] where it left, P< where it landed), and beside it
// each side's captures, the move number and the last move, whichever side made it (as in
// checkers).
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

// drawBoard draws the board and the play beside it: everything the final position needs.
func (g *Game) drawBoard(c *proto.Canvas, x int) {
	pos := g.g.Position()
	b := pos.Board()
	board.DrawCompact(c, x+boardX, 0, func(file, rank int) board.Glyph {
		sq := cg.NewSquare(cg.File(file), cg.Rank(rank))
		gl := board.Glyph{Mark: g.last != cg.NoSquare && (sq == g.last || sq == g.from)}
		if p := b.Piece(sq); p != cg.NoPiece {
			gl.R, gl.S = letter(p), proto.StyleBright
			if p.Color() == cg.Black {
				gl.S = proto.StyleText
			}
		}
		if sq == g.last {
			gl.A |= proto.AttrUnderline
		}
		return gl
	})

	check := false
	if moves := g.g.Moves(); len(moves) > 0 {
		check = moves[len(moves)-1].HasTag(cg.Check)
	}
	side := func(y int, who, colour string, toMove bool) {
		c.Put(x+infoX, y, who, proto.StyleLabel, 0)
		end := c.Put(x+valueX, y, colour, proto.StyleText, 0)
		if toMove && check {
			c.Put(end+2, y, panelCheck[0].Text, proto.StyleAlert, proto.AttrBold)
		}
	}
	turn := pos.Turn()
	side(2, panelWOPR[0].Text, panelSides[1].Text, turn == cg.Black)
	putTaken(c, x, 3, b, cg.White)
	c.Put(x+infoX, 5, panelMove[0].Text, proto.StyleDim, 0)
	c.Put(x+valueX, 5, strconv.Itoa(len(g.g.Moves())/2+1), proto.StyleText, 0)
	if g.lastText != "" && g.last != cg.NoSquare {
		c.Put(x+infoX, 6, panelLast[0].Text, proto.StyleDim, 0)
		c.Put(x+valueX, 6, g.lastText, proto.StyleText, 0)
	}
	putTaken(c, x, 8, b, cg.Black)
	side(9, panelYou[0].Text, panelSides[0].Text, turn == cg.White)
}

// putTaken lists the pieces of colour lost so far on row y, if any.
func putTaken(c *proto.Canvas, x, y int, b *cg.Board, lost cg.Color) {
	pieces := taken(b, lost)
	if len(pieces) == 0 {
		return
	}
	text := strings.Join(pieces, " ")
	if len(text) > infoW {
		text = strings.Join(pieces, "")
	}
	style := proto.StyleBright
	if lost == cg.Black {
		style = proto.StyleText
	}
	c.Put(x+infoX, y, panelTaken[0].Text, proto.StyleDim, 0)
	c.Put(x+valueX, y, text, style, 0)
}

// letter is a piece's letter: capitals for White, lower case for Black.
func letter(p cg.Piece) rune {
	r := rune(strings.ToUpper(p.Type().String())[0])
	if p.Color() == cg.Black {
		r += 'a' - 'A'
	}
	return r
}

// start is how many of each piece a side begins with, most valuable first.
var start = []struct {
	t cg.PieceType
	n int
}{{cg.Queen, 1}, {cg.Rook, 2}, {cg.Bishop, 2}, {cg.Knight, 2}, {cg.Pawn, 8}}

// taken lists the letters of colour's pieces that have been captured, most valuable first:
// the starting set less what is on the board, with each promoted piece counted against the
// pawn it was.
func taken(b *cg.Board, colour cg.Color) []string {
	count := map[cg.PieceType]int{}
	for sq := range cg.Square(64) {
		if p := b.Piece(sq); p != cg.NoPiece && p.Color() == colour {
			count[p.Type()]++
		}
	}
	promoted := 0
	for _, s := range start[:4] {
		promoted += max(count[s.t]-s.n, 0)
	}
	var out []string
	for _, s := range start {
		lost := max(s.n-count[s.t], 0)
		if s.t == cg.Pawn {
			lost = max(lost-promoted, 0)
		}
		for range lost {
			out = append(out, string(letter(cg.NewPiece(s.t, colour))))
		}
	}
	return out
}
