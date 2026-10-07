package poker

import (
	"slices"

	"github.com/GhostofGoes/WOPR/internal/games/cards"
)

// Category is a poker hand's class, lowest first.
type Category uint8

// Categories.
const (
	HighCard Category = iota
	OnePair
	TwoPair
	Trips
	Straight
	Flush
	FullHouse
	Quads
	StraightFlush
)

// Score ranks a five-card hand: the higher Score wins, and equal Scores split the pot.
// The category is in bits 20 and up; below it, up to five ranks of four bits each, in
// order of importance (the pair before the kickers, the trips before the pair).
type Score uint32

// Category is the hand's class.
func (s Score) Category() Category { return Category(s >> 20) }

// rank returns the i-th rank that decides the hand (0 is the most important).
func (s Score) rank(i int) cards.Rank { return cards.Rank(s >> (16 - 4*i) & 0xf) }

// Evaluate scores a five-card hand.
func Evaluate(h []cards.Card) Score {
	if len(h) != 5 {
		panic("poker: Evaluate needs five cards")
	}
	var count [15]int
	flush := true
	for _, c := range h {
		count[c.Rank]++
		flush = flush && c.Suit == h[0].Suit
	}
	// Ranks ordered by how many of each, then by rank: [K K 7 7 2] for two pair.
	type group struct {
		r cards.Rank
		n int
	}
	var groups []group
	for r := cards.Ace; r >= cards.Two; r-- {
		if count[r] > 0 {
			groups = append(groups, group{r, count[r]})
		}
	}
	slices.SortStableFunc(groups, func(a, b group) int { return b.n - a.n })

	straight, high := false, cards.Rank(0)
	if len(groups) == 5 {
		switch {
		case groups[0].r-groups[4].r == 4:
			straight, high = true, groups[0].r
		case groups[0].r == cards.Ace && groups[1].r == 5: // the wheel, A-2-3-4-5: five high
			straight, high = true, 5
		}
	}

	var cat Category
	switch {
	case straight && flush:
		cat = StraightFlush
	case groups[0].n == 4:
		cat = Quads
	case groups[0].n == 3 && groups[1].n == 2:
		cat = FullHouse
	case flush:
		cat = Flush
	case straight:
		cat = Straight
	case groups[0].n == 3:
		cat = Trips
	case groups[0].n == 2 && groups[1].n == 2:
		cat = TwoPair
	case groups[0].n == 2:
		cat = OnePair
	}
	s := Score(cat) << 20
	if straight {
		return s | Score(high)<<16
	}
	for i, g := range groups {
		s |= Score(g.r) << (16 - 4*i)
	}
	return s
}
