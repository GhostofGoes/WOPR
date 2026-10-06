package ginrummy

import (
	"fmt"

	"github.com/GhostofGoes/WOPR/internal/proto"
)

// View implements proto.Program: the score, the table and the player's hand, melds first
// (bright) and deadwood after, with the positions a discard can name.
//
//	GIN RUMMY                                          YOU 34   WOPR 12   TO 100
//	WOPR: 10 CARDS          STOCK: 21                  DISCARD: 7H
//
//	YOUR HAND
//	AS  2S  3S  7C  7D  7H  KH  QD  9C  4C
//	1   2   3   4   5   6   7   8   9   10
//	DEADWOOD: 27
//	WOPR DRAWS FROM THE STOCK. WOPR DISCARDS THE 9H.
func (g *Game) View(c *proto.Canvas) {
	c.Put(0, 0, panelTitle[0].Text, proto.StyleLabel, proto.AttrBold)
	score := fill(panelScore, fmt.Sprint(g.score[player]), fmt.Sprint(g.score[wopr]))
	c.Put(c.W-len(score), 0, score, proto.StyleText, 0)
	c.Put(0, 1, fill(panelWOPR, fmt.Sprint(len(g.hands[wopr]))), proto.StyleText, 0)
	c.Put(24, 1, fill(panelStock, fmt.Sprint(len(g.stock))), proto.StyleText, 0)
	if len(g.pile) > 0 {
		c.Put(51, 1, fill(panelDiscard, g.top().String()), proto.StyleBright, proto.AttrBold)
	}
	c.Put(0, 3, panelHand[0].Text, proto.StyleLabel, 0)
	a := Arrange(g.hands[player])
	melded := len(g.hands[player]) - len(a.Deadwood)
	for i, card := range g.hands[player] {
		style := proto.StyleText
		if i < melded {
			style = proto.StyleBright
		}
		c.Put(4*i, 4, card.String(), style, 0)
		c.Put(4*i, 5, fmt.Sprint(i+1), proto.StyleDim, 0)
	}
	c.Put(0, 6, fill(panelDeadwood, fmt.Sprint(a.Points)), proto.StyleText, 0)
	c.Put(0, 7, g.note, proto.StyleDim, 0)
}
