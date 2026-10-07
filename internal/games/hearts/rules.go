package hearts

import (
	"github.com/GhostofGoes/WOPR/internal/games/cards"
)

// The two of clubs leads the first trick; the queen of spades is worth 13.
var (
	twoClubs   = cards.Card{Rank: cards.Two, Suit: cards.Clubs}
	queenSpade = cards.Card{Rank: cards.Queen, Suit: cards.Spades}
)

// Scoring.
const (
	GameOver = 100 // the game ends after the hand in which someone reaches this
	Moon     = 26  // all the points: the taker scores none, everyone else 26
)

// Points is a card's penalty: each heart 1, the queen of spades 13.
func Points(c cards.Card) int {
	switch {
	case c == queenSpade:
		return 13
	case c.Suit == cards.Hearts:
		return 1
	}
	return 0
}

func pointsOf(cs []cards.Card) int {
	n := 0
	for _, c := range cs {
		n += Points(c)
	}
	return n
}

// Pass is a hand's passing direction, by the seats to the left: 1 left, 3 right,
// 2 across, 0 hold. It cycles left, right, across, hold.
func Pass(deal int) int { return [4]int{1, 3, 2, 0}[deal%4] }

// Refusal is why a play is refused.
type Refusal uint8

// Refusals.
const (
	OK Refusal = iota
	NotHeld
	MustFollow
	MustLeadTwo
	NotBroken
	NoPointsFirst
)

// Check says whether c may be played from hand to t. first is the deal's first trick;
// broken is whether hearts have been played.
func Check(hand []cards.Card, t cards.Trick, c cards.Card, first, broken bool) Refusal {
	switch {
	case cards.Index(hand, c) < 0:
		return NotHeld
	case !cards.Follows(hand, t, c):
		return MustFollow
	}
	if len(t.Cards) == 0 { // leading
		switch {
		case first && cards.Index(hand, twoClubs) >= 0 && c != twoClubs:
			return MustLeadTwo
		case !broken && c.Suit == cards.Hearts && !onlyHearts(hand):
			return NotBroken
		}
		return OK
	}
	if first && Points(c) > 0 && hasNonPoint(cards.Playable(hand, t)) {
		return NoPointsFirst
	}
	return OK
}

func onlyHearts(hand []cards.Card) bool {
	for _, c := range hand {
		if c.Suit != cards.Hearts {
			return false
		}
	}
	return true
}

func hasNonPoint(cs []cards.Card) bool {
	for _, c := range cs {
		if Points(c) == 0 {
			return true
		}
	}
	return false
}

// Legal lists the cards of hand that may be played to t.
func Legal(hand []cards.Card, t cards.Trick, first, broken bool) []cards.Card {
	var out []cards.Card
	for _, c := range hand {
		if Check(hand, t, c, first, broken) == OK {
			out = append(out, c)
		}
	}
	return out
}

// Settle turns one hand's points into score changes: shooting the moon gives the taker 0
// and everyone else 26.
func Settle(taken [cards.Seats]int) (add [cards.Seats]int, moon int) {
	for s, n := range taken {
		if n == Moon {
			for o := range add {
				if o != s {
					add[o] = Moon
				}
			}
			return add, s
		}
	}
	return taken, -1
}
