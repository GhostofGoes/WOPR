package bridge

import (
	"fmt"

	"github.com/GhostofGoes/WOPR/internal/games/cards"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// draw is the panel: the contract and tricks, North's and South's hands (the player holds
// both), the trick around the table, the auction and the last trick.
//
//	BRIDGE                                     CONTRACT 4S BY NORTH    N-S 3  E-W 2
//	NORTH          S A K 4  H Q 10 9  D J 7  C 8 6 3 2
//	                                  NORTH  QS
//	                     WEST   10H                   EAST   2H
//	                                  SOUTH  --
//	SOUTH (DUMMY)  S Q J 3  H A K  D A Q 10 4  C K 5 4
//
//	AUCTION, WEST DEALING: PASS 1S PASS 4S PASS PASS PASS
//	LAST TRICK: WEST 3D, NORTH 7D, EAST KD, SOUTH AD. SOUTH TAKES IT.
func (g *Game) draw(c *proto.Canvas) {
	c.Put(0, 0, panelTitle[0].Text, proto.StyleLabel, proto.AttrBold)
	head := fill(panelContract, g.contract.String(), cards.SeatNames[g.contract.Declarer]) + "    " +
		fill(panelTricks, fmt.Sprint(g.won[0]), fmt.Sprint(g.won[1]))
	c.Put(c.W-len(head), 0, head, proto.StyleText, 0)
	for _, row := range []struct{ seat, y int }{{cards.North, 1}, {cards.South, 5}} {
		label := cards.SeatNames[row.seat]
		if row.seat == g.dummy() {
			label += " " + panelDummy[0].Text
		}
		c.Put(0, row.y, label, proto.StyleLabel, 0)
		style := proto.StyleText
		if !g.over && g.trick.Next() == row.seat {
			style = proto.StyleBright
		}
		c.Put(15, row.y, bySuit(g.hands[row.seat]), style, 0)
	}
	spots := [cards.Seats]struct{ x, y int }{
		cards.South: {34, 4}, cards.West: {21, 3}, cards.North: {34, 2}, cards.East: {50, 3},
	}
	winning := -1
	if len(g.trick.Cards) > 0 {
		winning = g.trick.Winner(g.contract.Strain.Trump())
	}
	for s, at := range spots {
		c.Put(at.x, at.y, cards.SeatNames[s], proto.StyleLabel, 0)
		card, ok := g.trick.Played(s)
		switch {
		case ok:
			var attr proto.Attr
			if s == winning {
				attr = proto.AttrBold
			}
			c.Put(at.x+7, at.y, card.String(), proto.StyleBright, attr)
		case !g.over && g.trick.Next() == s:
			c.Put(at.x+7, at.y, "--", proto.StyleDim, 0)
		}
	}
	c.Put(0, 7, fill(lineAuction, cards.SeatNames[g.dealer], g.auctionText()), proto.StyleDim, 0)
	if g.lastBy >= 0 {
		c.Put(0, 8, panelLast[0].Text+" "+trickText(g.last, g.lastBy), proto.StyleDim, 0)
	}
}
