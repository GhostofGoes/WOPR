package bridge

import (
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games/cards"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

func hand(t testing.TB, s string) []cards.Card {
	t.Helper()
	var out []cards.Card
	for _, f := range strings.Fields(s) {
		c, ok := cards.Parse(f)
		if !ok {
			t.Fatalf("bad card %q", f)
		}
		out = append(out, c)
	}
	return out
}

func TestHCP(t *testing.T) {
	t.Parallel()
	if n := HCP(hand(t, "AS KS QS JS 10S 9H 8H 7H 6D 5D 4D 3C 2C")); n != 10 {
		t.Errorf("AKQJ = 10, got %d", n)
	}
	total := 0
	for _, c := range cards.Deck() {
		total += HCP([]cards.Card{c})
	}
	if total != 40 {
		t.Errorf("a deck holds 40 points, got %d", total)
	}
}

func TestScore(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		bid    Bid
		tricks int
		want   int
	}{
		{Bid{4, Strain(cards.Spades)}, 10, 420},
		{Bid{3, NT}, 10, 430},
		{Bid{2, Strain(cards.Hearts)}, 8, 110},
		{Bid{1, NT}, 7, 90},
		{Bid{5, Strain(cards.Clubs)}, 11, 400},
		{Bid{3, Strain(cards.Diamonds)}, 9, 110},
		{Bid{6, Strain(cards.Spades)}, 12, 980},
		{Bid{7, NT}, 13, 1520},
		{Bid{4, Strain(cards.Hearts)}, 8, -100},
		{Bid{3, NT}, 0, -450},
	} {
		if got := Score(Contract{Bid: tc.bid}, tc.tricks); got != tc.want {
			t.Errorf("%s with %d tricks: %d, want %d", tc.bid, tc.tricks, got, tc.want)
		}
	}
}

// Auctions over seeded deals: each bid outranks the one before, three passes end it (four
// when passed out), the contract is the last bid, its declarer opened it, and after the
// table turns the declarer is always North or South.
func TestAuctionsAndRotation(t *testing.T) {
	t.Parallel()
	for seed := range uint64(300) {
		g := New().(*Game)
		g.Start(proto.Env{Seed: seed, Instant: true, Deterministic: true})
		calls := g.calls
		if side(g.contract.Declarer) != 0 {
			t.Fatalf("seed %d: %s declares", seed, cards.SeatNames[g.contract.Declarer])
		}
		var last Bid
		lastAt := -1
		for i, b := range calls {
			if b == Pass {
				continue
			}
			if !b.Above(last) {
				t.Fatalf("seed %d: %s does not outrank %s", seed, b, last)
			}
			last, lastAt = b, i
		}
		if lastAt < 0 || len(calls)-lastAt-1 != 3 {
			t.Fatalf("seed %d: the auction must end with three passes: %v", seed, calls)
		}
		if last != g.contract.Bid {
			t.Fatalf("seed %d: contract %s, last bid %s", seed, g.contract, last)
		}
		first := -1
		for i, b := range calls {
			if b != Pass && b.Strain == last.Strain && side((g.dealer+i)%cards.Seats) == side(g.contract.Declarer) {
				first = (g.dealer + i) % cards.Seats
				break
			}
		}
		if first != g.contract.Declarer {
			t.Fatalf("seed %d: declarer %s, but %d named the strain first", seed, cards.SeatNames[g.contract.Declarer], first)
		}
	}
}

func TestPassedOut(t *testing.T) {
	t.Parallel()
	// Ten points each, an ace, king, queen and jack apiece: nobody opens.
	var hands [cards.Seats][]cards.Card
	honours := [cards.Seats]string{"AS KH QD JC", "AH KD QC JS", "AD KC QS JH", "AC KS QH JD"}
	for s := range hands {
		hands[s] = hand(t, honours[s])
	}
	i := 0
	for _, c := range cards.Deck() {
		if c.Rank <= cards.Ten {
			hands[i%cards.Seats] = append(hands[i%cards.Seats], c)
			i++
		}
	}
	for s := range hands {
		if HCP(hands[s]) != 10 || len(hands[s]) != 13 {
			t.Fatalf("fixture: %s", cards.Format(hands[s]))
		}
	}
	calls, _, ok := Auction(hands, cards.North)
	if ok || len(calls) != 4 {
		t.Fatalf("passed out: %v %v", calls, ok)
	}
}

// Defence and play over seeded deals with every seat played by the defence heuristic:
// every card follows suit and the tricks add up to 13.
func TestPlayLegality(t *testing.T) {
	t.Parallel()
	for seed := range uint64(200) {
		deck := cards.Shuffled(proto.NewRand(seed, 0))
		var hands [cards.Seats][]cards.Card
		for s := range hands {
			hands[s] = append([]cards.Card(nil), deck[13*s:13*s+13]...)
		}
		trump := cards.Suit(seed % 5)
		if seed%5 == 4 {
			trump = cards.NoTrump
		}
		leader := int(seed % 4)
		won := 0
		for range 13 {
			tr := cards.Trick{Leader: leader}
			for range cards.Seats {
				s := tr.Next()
				c := defend(hands[s], tr, trump, s)
				if !cards.Follows(hands[s], tr, c) {
					t.Fatalf("seed %d: %s played %s from %s", seed, cards.SeatNames[s], c, cards.Format(hands[s]))
				}
				hands[s] = cards.Remove(hands[s], c)
				tr.Cards = append(tr.Cards, c)
			}
			leader = tr.Winner(trump)
			won++
		}
		if won != 13 {
			t.Fatal("13 tricks")
		}
	}
}

func TestDefence(t *testing.T) {
	t.Parallel()
	// Third hand wins as cheaply as it can (clockwise: West, North, East).
	tr := cards.Trick{Leader: cards.West, Cards: hand(t, "4S 9S")}
	if c := defend(hand(t, "KS 10S 2S 3H"), tr, cards.Hearts, cards.East); c.String() != "10S" {
		t.Errorf("third hand should beat 9S with 10S: %s", c)
	}
	// Partner winning: play low (East, South, West).
	tr = cards.Trick{Leader: cards.East, Cards: hand(t, "AS 4S")}
	if c := defend(hand(t, "KS 10S 2S"), tr, cards.NoTrump, cards.West); c.String() != "2S" {
		t.Errorf("partner's ace wins: play low, not %s", c)
	}
	// Void: ruff when partner is not winning.
	tr = cards.Trick{Leader: cards.North, Cards: hand(t, "AD")}
	if c := defend(hand(t, "5H 9H 2C"), tr, cards.Hearts, cards.East); c.String() != "5H" {
		t.Errorf("ruff with the lowest trump: %s", c)
	}
	// Lead: ace from ace-king.
	if c := lead(hand(t, "AS KS 4S 3S 9H 2H 7D"), cards.Hearts); c.String() != "AS" {
		t.Errorf("lead the ace from AK: %s", c)
	}
}
