package bridge

import (
	"fmt"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/games/cards"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// Panel geometry.
const (
	northY   = 1  // North's hand
	tableX   = 8  // the trick table's left edge
	tableY   = 2  // and its top
	southY   = 8  // South's hand
	auctionY = 9  // the auction
	asideX   = 62 // the last trick
	handX    = 16 // where a hand's suits start, after its label and the marker of the hand to play
)

// draw is the panel: the contract and tricks; North's and South's hands, a suit at a time
// (the player holds both; the one to play is marked); the trick around the table; the
// trick just taken; and the auction.
//
//	BRIDGE                                      CONTRACT 3H BY NORTH    N-S 3  E-W 2
//	NORTH           S 3  H Q 3  D K J 7  C A 2
//	                                                              LAST TRICK
//	                      NORTH                                   NORTH  4D
//	                                           .===.              EAST   QD
//	          WEST               . - .         |QC | EAST         SOUTH  9D
//	                      SOUTH  :   :         '==='              WEST   3D
//	                             ' - '                            TAKEN BY EAST
//	SOUTH (DUMMY) > S -  H A K J 9 7  D -  C 7 6 5
//	AUCTION, WEST DEALING: PASS 1H PASS 3H PASS PASS PASS
func (g *Game) draw(c *proto.Canvas) {
	c.Put(0, 0, panelTitle[0].Text, proto.StyleLabel, proto.AttrBold)
	head := fill(panelContract, g.contract.String(), seatNames[g.contract.Declarer].Text) + "    " +
		fill(panelTricks, fmt.Sprint(g.won[0]), fmt.Sprint(g.won[1]))
	c.Put(c.W-len(head), 0, head, proto.StyleText, 0)

	next := -1
	if !g.over {
		next = g.trick.Next()
	}
	for _, row := range []struct{ seat, y int }{{cards.North, northY}, {cards.South, southY}} {
		label := seatNames[row.seat].Text
		if row.seat == g.dummy() {
			label += " " + panelDummy[0].Text
		}
		c.Put(0, row.y, label, proto.StyleLabel, 0)
		style := proto.StyleText
		if row.seat == next {
			style = proto.StyleBright
			c.Put(handX-2, row.y, panelToPlay[0].Text, proto.StyleBright, 0)
		}
		drawSuits(c, handX, row.y, g.hands[row.seat], style)
	}

	var names [cards.Seats]string
	for s := range names {
		names[s] = seatNames[s].Text
	}
	cards.DrawTrick(c, tableX, tableY, g.trick, g.contract.Strain.Trump(), names, next)
	if g.lastBy >= 0 {
		cards.DrawTaken(c, asideX, tableY, g.last, names, panelLast[0].Text, fill(panelTaken, seatNames[g.lastBy].Text))
	}
	c.Put(0, auctionY, fill(lineAuction, seatNames[g.dealer].Text, g.auctionText()), proto.StyleDim, 0)
}

// drawSuits draws a hand a suit at a time, spades first, each suit's letter in its suit's
// style and the ranks in style: S A K 4  H Q 10 9  D -  C 8 6 3 2.
func drawSuits(c *proto.Canvas, x, y int, hand []cards.Card, style proto.Style) {
	for i, part := range strings.Split(bySuit(hand), "  ") {
		if i > 0 {
			x += 2
		}
		s := cards.Suit(len(cards.Suits) - 1 - i) // spades first
		x = c.Put(x, y, part[:1], cards.SuitStyle(s), 0)
		x = c.Put(x, y, part[1:], style, 0)
	}
}
