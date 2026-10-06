package hearts

import (
	"fmt"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/games/cards"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// View implements proto.Program: the scores, the trick in play around the table, the
// player's hand with positions, and the trick just taken.
//
//	HEARTS                                          YOU 3  WEST 0  NORTH 13  EAST 10
//
//	                                  NORTH  QS
//	                     WEST  10H                    EAST  2H
//	                                  YOU    --
//
//	KS  QS  4S  AH  9H  3H  JD  8D  2D  KC  10C 6C  3C
//	1   2   3   4   5   6   7   8   9   10  11  12  13
//	PASS LEFT                                                          HEARTS BROKEN
//	LAST TRICK: WEST 10H, NORTH KH, EAST 2S, YOU AD. NORTH TAKES IT.
func (g *Game) View(c *proto.Canvas) {
	c.Put(0, 0, panelTitle[0].Text, proto.StyleLabel, proto.AttrBold)
	var scores []string
	for s, v := range g.score {
		scores = append(scores, fmt.Sprintf("%s %d", seatName(s), v))
	}
	line := strings.Join(scores, "  ")
	c.Put(c.W-len(line), 0, line, proto.StyleText, 0)

	// The table: where each seat's card lands.
	spots := [cards.Seats]struct{ x, y int }{
		cards.South: {34, 4}, cards.West: {21, 3}, cards.North: {34, 2}, cards.East: {50, 3},
	}
	winning := -1
	if len(g.trick.Cards) > 0 {
		winning = g.trick.Winner(cards.NoTrump)
	}
	for s, at := range spots {
		c.Put(at.x, at.y, seatName(s), proto.StyleLabel, 0)
		card, ok := g.trick.Played(s)
		switch {
		case ok:
			var attr proto.Attr
			if s == winning {
				attr = proto.AttrBold
			}
			c.Put(at.x+7, at.y, card.String(), proto.StyleBright, attr)
		case g.phase == playing && g.trick.Next() == s:
			c.Put(at.x+7, at.y, "--", proto.StyleDim, 0)
		}
	}

	c.Put(0, 5, panelHand[0].Text, proto.StyleLabel, 0)
	for i, card := range g.hands[you] {
		c.Put(4*i, 6, card.String(), proto.StyleText, 0)
		c.Put(4*i, 7, fmt.Sprint(i+1), proto.StyleDim, 0)
	}
	if g.phase == passing {
		c.Put(0, 8, fill(panelPass, panelDirs[Pass(g.deal-1)].Text), proto.StyleAlert, 0)
	}
	if g.broken {
		c.Put(c.W-len(panelBroken[0].Text), 8, panelBroken[0].Text, proto.StyleText, 0)
	}
	if g.lastBy >= 0 {
		c.Put(0, 9, panelLast[0].Text+" "+trickText(g.last, g.lastBy), proto.StyleDim, 0)
	}
}
