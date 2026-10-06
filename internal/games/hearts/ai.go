package hearts

import (
	"slices"

	"github.com/GhostofGoes/WOPR/internal/games/cards"
)

// WOPR's three seats play by heuristics: pass danger, duck under the winning card, dump
// the queen and high hearts when void, and lead low from short suits.

func countSuit(hand []cards.Card, s cards.Suit) int {
	n := 0
	for _, c := range hand {
		if c.Suit == s {
			n++
		}
	}
	return n
}

// danger rates a card for passing: the queen, unguarded high spades, high hearts, then
// high cards in short suits.
func danger(hand []cards.Card, c cards.Card) int {
	lowSpades := 0
	for _, x := range hand {
		if x.Suit == cards.Spades && x.Rank < cards.Queen {
			lowSpades++
		}
	}
	switch {
	case c == queenSpade:
		return 100
	case c.Suit == cards.Spades && c.Rank > cards.Queen && lowSpades < 3:
		return 90
	case c.Suit == cards.Hearts:
		return 20 + int(c.Rank)
	}
	short := 0
	if countSuit(hand, c.Suit) <= 2 {
		short = 8 // help void the suit
	}
	return int(c.Rank) + short
}

// choosePass picks three cards to pass.
func choosePass(hand []cards.Card) []cards.Card {
	h := slices.Clone(hand)
	slices.SortStableFunc(h, func(a, b cards.Card) int { return danger(hand, b) - danger(hand, a) })
	return h[:3]
}

// choosePlay picks WOPR's card from the legal ones.
func choosePlay(hand []cards.Card, t cards.Trick, legal []cards.Card) cards.Card {
	if len(legal) == 1 {
		return legal[0]
	}
	led, ok := t.Led()
	if !ok {
		return chooseLead(hand, legal)
	}
	if !cards.HasSuit(hand, led) { // void: dump
		return dump(hand, legal)
	}
	win := t.Cards[t.Winning(cards.NoTrump)]
	// Duck: the highest card that still loses.
	var under []cards.Card
	for _, c := range legal {
		if c.Rank < win.Rank {
			under = append(under, c)
		}
	}
	if len(under) > 0 {
		return highest(under)
	}
	points := pointsOf(t.Cards)
	if len(t.Cards) == cards.Seats-1 && points == 0 { // last to play, nothing to lose: shed the top card
		safe := slices.DeleteFunc(slices.Clone(legal), func(c cards.Card) bool { return c == queenSpade })
		if len(safe) > 0 {
			return highest(safe)
		}
	}
	return lowest(legal)
}

func chooseLead(hand, legal []cards.Card) cards.Card {
	// Lead low from the shortest suit, keeping spades at the queen and above at home.
	best := legal[0]
	score := func(c cards.Card) int {
		s := int(c.Rank)*4 + countSuit(hand, c.Suit)
		if c.Suit == cards.Spades && c.Rank >= cards.Queen {
			s += 100
		}
		if c.Suit == cards.Hearts {
			s += 20
		}
		return s
	}
	for _, c := range legal[1:] {
		if score(c) < score(best) {
			best = c
		}
	}
	return best
}

// dump sheds the most dangerous legal card: the queen, then the highest heart, then the
// highest card of the shortest suit.
func dump(hand, legal []cards.Card) cards.Card {
	if slices.Contains(legal, queenSpade) {
		return queenSpade
	}
	var hearts []cards.Card
	for _, c := range legal {
		if c.Suit == cards.Hearts {
			hearts = append(hearts, c)
		}
	}
	if len(hearts) > 0 {
		return highest(hearts)
	}
	best := legal[0]
	for _, c := range legal[1:] {
		if danger(hand, c) > danger(hand, best) {
			best = c
		}
	}
	return best
}

func highest(cs []cards.Card) cards.Card {
	return slices.MaxFunc(cs, func(a, b cards.Card) int { return int(a.Rank) - int(b.Rank) })
}

func lowest(cs []cards.Card) cards.Card {
	return slices.MinFunc(cs, func(a, b cards.Card) int { return int(a.Rank) - int(b.Rank) })
}
