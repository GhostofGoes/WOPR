package hearts

import (
	"fmt"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/games/cards"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// Panel geometry.
const (
	tableX   = 8  // the trick table's left edge
	tableY   = 1  // and its top
	asideX   = 62 // the last trick, and the notes beside the hand
	handY    = 7  // the player's hand: two rows of card tops
	numbersY = 9  // the positions under them
)

// View implements proto.Program: the scores; the trick in play around the table, with an
// empty slot where the next card goes (or, while the player passes, three cards face down
// and an arrow to the seat that gets them); the trick just taken; and the player's hand as
// cards with the position under each.
//
//	HEARTS                                            YOU 0  WEST 0  NORTH 0  EAST 0
//	                             .---.                            LAST TRICK (4)
//	                      NORTH  |2S |                            EAST   3H
//	                             '---'         .---.              YOU    QH
//	          WEST               . - .         |9S | EAST         WEST   10H
//	                        YOU  :   :         '---'              NORTH  AH
//	                             ' - '                            TAKEN BY NORTH
//	.---.---.---.---.---.---.---.
//	|JH |5H |2H |QC |8C |7C |4C |                                 YOUR HAND
//	  1   2   3   4   5   6   7                                   HEARTS BROKEN
func (g *Game) View(c *proto.Canvas) {
	c.Put(0, 0, panelTitle[0].Text, proto.StyleLabel, proto.AttrBold)
	var scores []string
	for s, v := range g.score {
		scores = append(scores, fmt.Sprintf("%s %d", seatName(s), v))
	}
	line := strings.Join(scores, "  ")
	c.Put(c.W-len(line), 0, line, proto.StyleText, 0)

	var names [cards.Seats]string
	for s := range names {
		names[s] = seatName(s)
	}
	next := -1
	if g.phase == playing {
		next = g.trick.Next()
	}
	cards.DrawTrick(c, tableX, tableY, g.trick, cards.NoTrump, names, next)
	if g.phase == passing {
		cards.DrawPass(c, tableX, tableY, (you+Pass(g.deal-1))%cards.Seats)
	}
	if g.lastBy >= 0 {
		title := panelLast[0].Text
		if pts := pointsOf(g.last.Cards); pts > 0 {
			title += " " + fill(linePoints, fmt.Sprint(pts))
		}
		cards.DrawTaken(c, asideX, tableY, g.last, names, title, fill(panelTaken, seatName(g.lastBy)))
	}

	cards.DrawTops(c, 0, handY, g.hands[you], proto.StyleText, 0)
	for i := range g.hands[you] {
		num := fmt.Sprint(i + 1)
		c.Put(cards.TopW*i+3-len(num), numbersY, num, proto.StyleDim, 0)
	}
	if len(g.hands[you]) > 0 {
		c.Put(asideX, handY+1, panelHand[0].Text, proto.StyleLabel, 0)
	}
	switch {
	case g.phase == passing:
		c.Put(asideX, numbersY, fill(panelPass, panelDirs[Pass(g.deal-1)].Text), proto.StyleAlert, 0)
	case g.broken:
		c.Put(asideX, numbersY, panelBroken[0].Text, proto.StyleText, 0)
	}
}
