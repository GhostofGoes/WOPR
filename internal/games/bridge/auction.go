package bridge

import (
	"github.com/GhostofGoes/WOPR/internal/games/cards"
)

// Strain is a bid's denomination: a suit, or no trumps (above spades).
type Strain uint8

// NT is no trumps. The suits are cards.Clubs to cards.Spades.
const NT Strain = 4

// Trump is the trump suit for the play, or cards.NoTrump.
func (s Strain) Trump() cards.Suit {
	if s == NT {
		return cards.NoTrump
	}
	return cards.Suit(s)
}

// Bid is a call in the auction. Level 0 is a pass.
type Bid struct {
	Level  int
	Strain Strain
}

// Pass is the call that bids nothing.
var Pass = Bid{}

// Above reports whether b outranks o: a higher level, or the same level in a higher strain.
func (b Bid) Above(o Bid) bool {
	return b.Level > o.Level || (b.Level == o.Level && b.Strain > o.Strain)
}

// Contract is the auction's result.
type Contract struct {
	Bid
	Declarer int
}

// HCP counts high-card points: ace 4, king 3, queen 2, jack 1.
func HCP(hand []cards.Card) int {
	n := 0
	for _, c := range hand {
		if c.Rank >= cards.Jack {
			n += int(c.Rank) - 10
		}
	}
	return n
}

// Opening is the fewest points that open the bidding.
const Opening = 12

// partner is the seat across the table.
func partner(seat int) int { return (seat + 2) % cards.Seats }

// side is 0 for North-South, 1 for East-West.
func side(seat int) int { return seat % 2 }

// target picks the contract for a partnership from its combined points and suit lengths:
// a major with eight cards or more, else no trumps if there is game in it, else the best
// minor fit, else no trumps; the level follows the points.
func target(a, b []cards.Card) Bid {
	points := HCP(a) + HCP(b)
	var length [4]int
	for _, c := range append(append([]cards.Card(nil), a...), b...) {
		length[c.Suit]++
	}
	level := func(steps [5]int) int { // points needed for levels 2, 3, 4, 6, 7
		lv := 1
		for i, need := range steps {
			if points >= need {
				lv = [5]int{2, 3, 4, 6, 7}[i]
			}
		}
		return lv
	}
	major := cards.Spades
	if length[cards.Hearts] > length[cards.Spades] {
		major = cards.Hearts
	}
	minor := cards.Diamonds
	if length[cards.Clubs] > length[cards.Diamonds] {
		minor = cards.Clubs
	}
	switch {
	case length[major] >= 8:
		return Bid{level([5]int{23, 24, 26, 33, 37}), Strain(major)}
	case points >= 25 && points < 33, length[minor] < 8:
		return Bid{min(level([5]int{23, 25, 99, 33, 37}), 7), NT}
	}
	lv := level([5]int{20, 23, 29, 33, 37}) // a minor game needs eleven tricks
	if lv == 4 {
		lv = 5
	}
	return Bid{lv, Strain(minor)}
}

// Auction runs WOPR's bidding for all four seats, starting with dealer. The partnership
// that holds more points (dealer's side on a tie) declares, if one of its players can
// open: that player bids one of the target strain, partner raises to the target, and
// everyone else passes. With no opening hand at the table, the deal is passed out.
// It returns the calls in order and the contract (ok is false when passed out).
func Auction(hands [cards.Seats][]cards.Card, dealer int) (calls []Bid, c Contract, ok bool) {
	ns := HCP(hands[cards.North]) + HCP(hands[cards.South])
	ew := HCP(hands[cards.East]) + HCP(hands[cards.West])
	declaring := side(dealer)
	switch {
	case ns > ew:
		declaring = 0
	case ew > ns:
		declaring = 1
	}
	opener := -1
	for i := range cards.Seats {
		s := (dealer + i) % cards.Seats
		if side(s) == declaring && HCP(hands[s]) >= Opening {
			opener = s
			break
		}
	}
	if opener < 0 {
		return []Bid{Pass, Pass, Pass, Pass}, Contract{}, false
	}
	goal := target(hands[opener], hands[partner(opener)])
	seat := dealer
	for ; seat != opener; seat = (seat + 1) % cards.Seats {
		calls = append(calls, Pass)
	}
	calls = append(calls, Bid{1, goal.Strain})
	if goal.Level > 1 {
		calls = append(calls, Pass, goal) // the opponent on the left passes; partner raises
	}
	calls = append(calls, Pass, Pass, Pass)
	return calls, Contract{Bid: goal, Declarer: opener}, true
}

// Score is the declaring side's score, not vulnerable and undoubled: trick points, the
// part-score or game bonus, slam bonuses and overtricks when made; 50 a trick when down.
func Score(c Contract, tricks int) int {
	need := 6 + c.Level
	if tricks < need {
		return -50 * (need - tricks)
	}
	per := 30
	if c.Strain == Strain(cards.Clubs) || c.Strain == Strain(cards.Diamonds) {
		per = 20
	}
	points := c.Level * per
	if c.Strain == NT {
		points += 10
	}
	bonus := 50
	if points >= 100 {
		bonus = 300
	}
	switch c.Level {
	case 6:
		bonus += 500
	case 7:
		bonus += 1000
	}
	return points + bonus + (tricks-need)*per
}
