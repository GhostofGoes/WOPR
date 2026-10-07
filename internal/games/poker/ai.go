package poker

import (
	"slices"

	"github.com/GhostofGoes/WOPR/internal/games/cards"
)

// WOPR's style: how often it decides to bluff a hand, and how far its judgement wavers.
const (
	bluffChance = 0.12
	snowChance  = 0.35 // a bluffing WOPR stands pat on nothing this often
	jitter      = 0.10
)

// strength estimates WOPR's hand from 0 to 1. Before the draw, four to a flush or an
// open-ended straight count for something.
func strength(h []cards.Card, drawn bool) float64 {
	s := Evaluate(h)
	top := float64(s.rank(0)-cards.Two) / 12
	var v float64
	switch s.Category() {
	case HighCard:
		v = 0.05 + 0.15*top
	case OnePair:
		v = 0.30 + 0.25*top
	case TwoPair:
		v = 0.62 + 0.10*top
	case Trips:
		v = 0.76 + 0.08*top
	case Straight:
		v = 0.86
	case Flush:
		v = 0.89
	case FullHouse:
		v = 0.93
	case Quads:
		v = 0.97
	default:
		v = 0.99
	}
	if !drawn && s.Category() < TwoPair {
		switch {
		case flushDraw(h) >= 0:
			v = max(v, 0.40)
		case straightDraw(h) >= 0:
			v = max(v, 0.35)
		}
	}
	return v
}

// flushDraw returns the index of the one card off suit when the other four share a suit,
// or -1.
func flushDraw(h []cards.Card) int {
	for i := range h {
		var suit cards.Suit
		n := 0
		for j, c := range h {
			if j == i {
				continue
			}
			if n == 0 {
				suit = c.Suit
			}
			if c.Suit == suit {
				n++
			}
		}
		if n == 4 && h[i].Suit != suit {
			return i
		}
	}
	return -1
}

// straightDraw returns the index of the odd card when the other four are consecutive
// ranks below the ace (open at both ends), or -1.
func straightDraw(h []cards.Card) int {
	for i := range h {
		var rs []cards.Rank
		for j, c := range h {
			if j != i {
				rs = append(rs, c.Rank)
			}
		}
		slices.Sort(rs)
		if len(slices.Compact(slices.Clone(rs))) == 4 && rs[3]-rs[0] == 3 && rs[3] < cards.Ace {
			return i
		}
	}
	return -1
}

// discards chooses the cards WOPR throws away, as indices into h: it keeps any made hand
// and its pairs, draws one to four of a flush or an open straight, and otherwise keeps its
// two highest cards.
func discards(h []cards.Card) []int {
	s := Evaluate(h)
	switch s.Category() {
	case Straight, Flush, FullHouse, Quads, StraightFlush:
		return nil
	case Trips, TwoPair, OnePair:
		var count [15]int
		for _, c := range h {
			count[c.Rank]++
		}
		var out []int
		for i, c := range h {
			if count[c.Rank] == 1 {
				out = append(out, i)
			}
		}
		return out
	}
	if i := flushDraw(h); i >= 0 {
		return []int{i}
	}
	if i := straightDraw(h); i >= 0 {
		return []int{i}
	}
	idx := []int{0, 1, 2, 3, 4}
	slices.SortStableFunc(idx, func(a, b int) int { return int(h[b].Rank) - int(h[a].Rank) })
	out := idx[2:]
	slices.Sort(out)
	return out
}

type action uint8

const (
	check action = iota
	call
	raise // a bet when there is nothing to call
	fold
)

// decide is WOPR's betting heuristic: bet or raise a strong hand (or a bluff), call a fair
// one or a cheap price, and fold the rest. After the draw it respects a player who stood
// pat.
func (g *Game) decide() action {
	s := strength(g.hands[wopr], g.round == 1)
	s += (g.ai.Float64() - 0.5) * jitter
	if g.bluff {
		s = max(s, 0.82)
	} else if g.round == 1 && g.drew[player] == 0 {
		s -= 0.10
	}
	toCall := g.in[player] - g.in[wopr]
	canRaise := g.raiseAmount(wopr) > 0
	if toCall == 0 {
		if s >= 0.55 && canRaise {
			return raise
		}
		return check
	}
	switch {
	case s >= 0.80 && canRaise:
		return raise
	case s >= 0.40, toCall*4 <= g.pot:
		return call
	}
	return fold
}
