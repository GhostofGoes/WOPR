package ginrummy

import (
	"math/bits"
	"slices"

	"github.com/GhostofGoes/WOPR/internal/games/cards"
)

// Gin counts aces low: A-2-3 is a run, Q-K-A is not.

// low is a card's rank with the ace as 1.
func low(c cards.Card) int {
	if c.Rank == cards.Ace {
		return 1
	}
	return int(c.Rank)
}

// Points is a card's deadwood count: aces 1, faces 10.
func Points(c cards.Card) int { return min(low(c), 10) }

// sum is the deadwood count of cs.
func sum(cs []cards.Card) int {
	n := 0
	for _, c := range cs {
		n += Points(c)
	}
	return n
}

// Arrangement is a hand split into melds and deadwood.
type Arrangement struct {
	Melds    [][]cards.Card
	Deadwood []cards.Card
	Points   int // the deadwood's count
}

// candidates lists every possible meld in hand as a bitmask of hand indices: each set of
// three or four of a rank, and each run of three or more in a suit.
func candidates(hand []cards.Card) []uint32 {
	var out []uint32
	byRank := map[int][]int{}
	bySuit := map[cards.Suit][]int{}
	for i, c := range hand {
		byRank[low(c)] = append(byRank[low(c)], i)
		bySuit[c.Suit] = append(bySuit[c.Suit], i)
	}
	for r := 1; r <= 13; r++ {
		idx := byRank[r]
		switch len(idx) {
		case 3:
			out = append(out, mask(idx...))
		case 4:
			out = append(out, mask(idx...))
			for skip := range idx {
				out = append(out, mask(idx...)&^(1<<idx[skip]))
			}
		}
	}
	for _, s := range cards.Suits {
		idx := bySuit[s]
		slices.SortFunc(idx, func(a, b int) int { return low(hand[a]) - low(hand[b]) })
		for i := range idx {
			for j := i + 2; j < len(idx); j++ {
				if low(hand[idx[j]])-low(hand[idx[i]]) != j-i {
					break // a gap
				}
				out = append(out, mask(idx[i:j+1]...))
			}
		}
	}
	return out
}

func mask(idx ...int) uint32 {
	var m uint32
	for _, i := range idx {
		m |= 1 << i
	}
	return m
}

// Arrange finds the melds that leave the least deadwood. Ties prefer more cards melded.
func Arrange(hand []cards.Card) Arrangement {
	cands := candidates(hand)
	var value [32]int
	for i, c := range hand {
		value[i] = Points(c)
	}
	covered := func(m uint32) int {
		n := 0
		for m != 0 {
			i := bits.TrailingZeros32(m)
			n += value[i]
			m &^= 1 << i
		}
		return n
	}
	bestUsed, bestCover, bestSet := uint32(0), -1, []uint32(nil)
	var set []uint32
	var search func(from int, used uint32)
	search = func(from int, used uint32) {
		cover := covered(used)
		if cover > bestCover || (cover == bestCover && bits.OnesCount32(used) > bits.OnesCount32(bestUsed)) {
			bestUsed, bestCover, bestSet = used, cover, slices.Clone(set)
		}
		for i := from; i < len(cands); i++ {
			if cands[i]&used == 0 {
				set = append(set, cands[i])
				search(i+1, used|cands[i])
				set = set[:len(set)-1]
			}
		}
	}
	search(0, 0)

	var a Arrangement
	for _, m := range bestSet {
		var meld []cards.Card
		for i, c := range hand {
			if m&(1<<i) != 0 {
				meld = append(meld, c)
			}
		}
		sortMeld(meld)
		a.Melds = append(a.Melds, meld)
	}
	for i, c := range hand {
		if bestUsed&(1<<i) == 0 {
			a.Deadwood = append(a.Deadwood, c)
		}
	}
	sortMeld(a.Deadwood)
	a.Points = sum(a.Deadwood)
	return a
}

// sortMeld orders cards low to high, then by suit.
func sortMeld(cs []cards.Card) {
	slices.SortFunc(cs, func(a, b cards.Card) int {
		if low(a) != low(b) {
			return low(a) - low(b)
		}
		return int(a.Suit) - int(b.Suit)
	})
}

// isSet reports three or four cards of one rank.
func isSet(m []cards.Card) bool {
	for _, c := range m {
		if c.Rank != m[0].Rank {
			return false
		}
	}
	return len(m) >= 3
}

// fits reports whether c can be laid off on meld m: a fourth card of a set, or the next
// card at either end of a run.
func fits(m []cards.Card, c cards.Card) bool {
	if isSet(m) {
		return c.Rank == m[0].Rank && len(m) < 4
	}
	if c.Suit != m[0].Suit {
		return false
	}
	return low(c) == low(m[0])-1 || low(c) == low(m[len(m)-1])+1
}

// LayOff plays the defender's deadwood onto the knocker's melds so that as little as
// possible is left: it tries every card on every meld it fits, in every order, since laying
// one card can make room for the next and a card may fit a set and a run.
func LayOff(knocker [][]cards.Card, deadwood []cards.Card) (laid, left []cards.Card) {
	melds := make([][]cards.Card, len(knocker))
	for i, m := range knocker {
		melds[i] = slices.Clone(m)
	}
	best, bestLaid := slices.Clone(deadwood), []cards.Card(nil)
	var search func(melds [][]cards.Card, left, laid []cards.Card)
	search = func(melds [][]cards.Card, left, laid []cards.Card) {
		if sum(left) < sum(best) {
			best, bestLaid = slices.Clone(left), slices.Clone(laid)
		}
		for i, c := range left {
			for j, m := range melds {
				if !fits(m, c) {
					continue
				}
				next := slices.Clone(melds)
				next[j] = append(slices.Clone(m), c)
				sortMeld(next[j])
				search(next, slices.Delete(slices.Clone(left), i, i+1), append(slices.Clone(laid), c))
			}
		}
	}
	search(melds, slices.Clone(deadwood), nil)
	return bestLaid, best
}

// arrangements lists every way to meld a hand (each set of disjoint melds, none included),
// for a defender who may break a meld to lay off more.
func arrangements(hand []cards.Card) []Arrangement {
	cands := candidates(hand)
	var out []Arrangement
	var set []uint32
	var search func(from int, used uint32)
	search = func(from int, used uint32) {
		var a Arrangement
		for _, m := range set {
			var meld []cards.Card
			for i, c := range hand {
				if m&(1<<i) != 0 {
					meld = append(meld, c)
				}
			}
			a.Melds = append(a.Melds, meld)
		}
		for i, c := range hand {
			if used&(1<<i) == 0 {
				a.Deadwood = append(a.Deadwood, c)
			}
		}
		a.Points = sum(a.Deadwood)
		out = append(out, a)
		for i := from; i < len(cands); i++ {
			if cands[i]&used == 0 {
				set = append(set, cands[i])
				search(i+1, used|cands[i])
				set = set[:len(set)-1]
			}
		}
	}
	search(0, 0)
	return out
}

// Scoring.
const (
	GinBonus      = 25
	UndercutBonus = 25
	GameTarget    = 100
	KnockLimit    = 10 // the most deadwood a knock allows
)

// Outcome of a knock: who scores, how much, and how it went.
type Outcome struct {
	KnockerWins bool
	Points      int
	Gin         bool
	Undercut    bool
	Laid        []cards.Card // cards the defender laid off
}

// Score settles a knock. knocker is the knocker's ten cards after the discard; defender is
// the other hand. Gin allows no lay-offs.
func Score(knocker, defender []cards.Card) Outcome {
	k := Arrange(knocker)
	d := Arrange(defender)
	if k.Points == 0 {
		return Outcome{KnockerWins: true, Points: GinBonus + d.Points, Gin: true}
	}
	// The defender chooses its melds too: whichever leaves the least after laying off.
	laid, left := LayOff(k.Melds, d.Deadwood)
	for _, a := range arrangements(defender) { // more deadwood before laying off can mean less after
		if l, rest := LayOff(k.Melds, a.Deadwood); sum(rest) < sum(left) {
			laid, left = l, rest
		}
	}
	dPoints := sum(left)
	if dPoints <= k.Points {
		return Outcome{Points: k.Points - dPoints + UndercutBonus, Undercut: true, Laid: laid}
	}
	return Outcome{KnockerWins: true, Points: dPoints - k.Points, Laid: laid}
}
