package ginrummy

import (
	"fmt"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/games/cards"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// Panel geometry.
const (
	tableY   = 1  // WOPR's hand, the stock and the discard: three rows of mini cards
	noteY    = 4  // what WOPR did last
	handY    = 5  // the player's hand: two rows of card tops
	numbersY = 7  // the positions under them
	groupGap = 2  // columns between melds, and before the deadwood
	asideX   = 62 // the deadwood count and the melds' positions, right of the hand
)

// View implements proto.Program: the score; WOPR's hand face down, the stock and the
// discard; what WOPR did last; and the player's hand as cards, each meld (bright) apart
// from the next and the deadwood last, with the position a discard can name under each and,
// beside it, the deadwood count and the melds' positions in words.
//
//	GIN RUMMY                                                YOU 34   WOPR 12   TO 100
//	       .-.-.-.-.-.-.-.-.-.---.                  .---.                     .---.
//	WOPR   |/|/|/|/|/|/|/|/|/|/\/| 10 CARDS   STOCK |/\/| 21          DISCARD |7H |
//	       '-'-'-'-'-'-'-'-'-'---'                  '---'                     '---'
//	WOPR DRAWS FROM THE STOCK. WOPR DISCARDS THE 9H.
//	.---.---.---.  .---.---.---.  .---.---.---.---.               YOUR HAND
//	|AS |2S |3S |  |7C |7D |7H |  |KH |QD |9C |4C |               DEADWOOD: 27
//	  1   2   3      4   5   6      7   8   9  10                 MELDS 1-3 4-6
func (g *Game) View(c *proto.Canvas) {
	c.Put(0, 0, panelTitle[0].Text, proto.StyleLabel, proto.AttrBold)
	score := fill(panelScore, fmt.Sprint(g.score[player]), fmt.Sprint(g.score[wopr]))
	c.Put(c.W-len(score), 0, score, proto.StyleText, 0)

	c.Put(0, tableY+1, panelWOPR[0].Text, proto.StyleLabel, 0)
	n := len(g.hands[wopr])
	cards.DrawFan(c, 7, tableY, n, proto.StyleDim)
	c.Put(8+cards.FanWidth(n), tableY+1, fill(panelCards, fmt.Sprint(n)), proto.StyleText, 0)
	c.Put(42, tableY+1, panelStock[0].Text, proto.StyleLabel, 0)
	if len(g.stock) > 0 {
		cards.DrawMiniBack(c, 48, tableY, proto.StyleDim)
	} else {
		cards.DrawSlot(c, 48, tableY, proto.StyleDim)
	}
	c.Put(54, tableY+1, fmt.Sprint(len(g.stock)), proto.StyleText, 0)
	c.Put(66, tableY+1, panelDiscard[0].Text, proto.StyleLabel, 0)
	if len(g.pile) > 0 {
		cards.DrawMini(c, 74, tableY, g.top(), proto.StyleBright, 0)
	} else {
		cards.DrawSlot(c, 74, tableY, proto.StyleDim)
	}

	c.Put(0, noteY, g.note, proto.StyleDim, 0)

	a := Arrange(g.hands[player])
	x, pos := 0, 1
	var melds []string // each meld's positions, 1-3
	for _, run := range g.runs(a) {
		frame := proto.StyleBright
		if run.deadwood {
			frame = proto.StyleText
		} else {
			melds = append(melds, fmt.Sprintf("%d-%d", pos, pos+len(run.cards)-1))
		}
		end := cards.DrawTops(c, x, handY, run.cards, frame, 0)
		for j := range run.cards {
			num := fmt.Sprint(pos)
			c.Put(x+cards.TopW*j+3-len(num), numbersY, num, proto.StyleDim, 0)
			pos++
		}
		x = end + groupGap
	}
	c.Put(asideX, handY, panelHand[0].Text, proto.StyleLabel, 0)
	c.Put(asideX, handY+1, fill(panelDeadwood, fmt.Sprint(a.Points)), proto.StyleText, 0)
	// Which cards are melded is said in words too, not only by the brighter frame.
	meldText := panelMelds[1].Text
	if len(melds) > 0 {
		meldText = fill(panelMelds, strings.Join(melds, " "))
	}
	c.Put(asideX, numbersY, meldText, proto.StyleText, 0)
}

// run is a stretch of the player's hand that the panel draws together: one meld, or the
// deadwood.
type run struct {
	cards    []cards.Card
	deadwood bool
}

// runs splits the player's hand, in its own order (the order positions count in), where
// a card belongs to another meld than the card before it, or to none.
func (g *Game) runs(a Arrangement) []run {
	group := map[cards.Card]int{}
	for i, m := range a.Melds {
		for _, c := range m {
			group[c] = i
		}
	}
	for _, c := range a.Deadwood {
		group[c] = len(a.Melds)
	}
	var out []run
	for i, c := range g.hands[player] {
		if i == 0 || group[c] != group[g.hands[player][i-1]] {
			out = append(out, run{deadwood: group[c] == len(a.Melds)})
		}
		out[len(out)-1].cards = append(out[len(out)-1].cards, c)
	}
	return out
}
