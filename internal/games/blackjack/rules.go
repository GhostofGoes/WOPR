package blackjack

import "github.com/GhostofGoes/WOPR/internal/games/cards"

// Rules are the table's fixed rules (docs/PLAN.md §6.1). Money is in cents, so that a
// black jack's 3 to 2 is exact on any whole-dollar bet.
type Rules struct {
	StandSoft17 bool // the dealer stands on a soft 17; false: the dealer hits it
	MinBet      int  // dollars
	MaxBet      int  // dollars
	Stake       int  // dollars the player sits down with
	Reshuffle   int  // a new deck is shuffled before a round when fewer cards than this remain
}

// DefaultRules are the rules WOPR deals by.
var DefaultRules = Rules{StandSoft17: true, MinBet: 1, MaxBet: 25, Stake: 100, Reshuffle: 15}

// cardValue is a card's count: aces 1 (Value makes one of them 11 when that helps), faces 10.
func cardValue(c cards.Card) int {
	switch {
	case c.Rank == cards.Ace:
		return 1
	case c.Rank >= cards.Ten:
		return 10
	}
	return int(c.Rank)
}

// Value is a hand's best total, and whether it is soft (an ace counted as 11).
func Value(h []cards.Card) (total int, soft bool) {
	aces := false
	for _, c := range h {
		total += cardValue(c)
		aces = aces || c.Rank == cards.Ace
	}
	if aces && total+10 <= 21 {
		return total + 10, true
	}
	return total, false
}

// Natural reports a black jack: an ace and a ten-count card as the first two cards.
func Natural(h []cards.Card) bool {
	total, _ := Value(h)
	return len(h) == 2 && total == 21
}

// DealerHits reports whether the dealer must draw to h.
func (r Rules) DealerHits(h []cards.Card) bool {
	total, soft := Value(h)
	return total < 17 || (total == 17 && soft && !r.StandSoft17)
}

// Peeks reports whether the dealer, showing up, checks the hole card for a black jack.
func Peeks(up cards.Card) bool { return up.Rank == cards.Ace || cardValue(up) == 10 }

// Hand is one of the player's hands. A split makes two.
type Hand struct {
	Cards     []cards.Card
	Bet       int // cents
	Doubled   bool
	FromSplit bool
	Stood     bool
}

// Bust reports a total over 21.
func (h *Hand) Bust() bool {
	total, _ := Value(h.Cards)
	return total > 21
}

// Done reports that the hand takes no more cards: it stood, busted, reached 21, doubled
// (one card only), or is a split ace (one card only).
func (h *Hand) Done() bool {
	total, _ := Value(h.Cards)
	splitAce := h.FromSplit && h.Cards[0].Rank == cards.Ace && len(h.Cards) >= 2
	return h.Stood || total >= 21 || (h.Doubled && len(h.Cards) >= 3) || splitAce
}

// CanDouble reports whether the hand may double down: on its first two cards, unless it is
// done (split aces are).
func (h *Hand) CanDouble() bool { return len(h.Cards) == 2 && !h.Done() }

// CanSplit reports whether the hand is a pair that may be split. One split per round.
func (h *Hand) CanSplit(hands int) bool {
	return hands == 1 && len(h.Cards) == 2 && h.Cards[0].Rank == h.Cards[1].Rank && !h.Done()
}

// Verdict is how one hand settled.
type Verdict uint8

// Verdicts.
const (
	Lose Verdict = iota
	Push
	Win
	Blackjack // a natural, paid 3 to 2
)

// Settle pays one hand against the dealer's final hand and returns the player's net
// winnings in cents. A natural after a split counts as 21, not a black jack.
func Settle(h Hand, dealer []cards.Card) (net int, v Verdict) {
	player, _ := Value(h.Cards)
	house, _ := Value(dealer)
	natural := Natural(h.Cards) && !h.FromSplit
	switch {
	case player > 21:
		return -h.Bet, Lose
	case natural && Natural(dealer):
		return 0, Push
	case natural:
		return h.Bet * 3 / 2, Blackjack
	case Natural(dealer):
		return -h.Bet, Lose
	case house > 21 || player > house:
		return h.Bet, Win
	case player == house:
		return 0, Push
	}
	return -h.Bet, Lose
}
