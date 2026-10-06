package cards_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games/cards"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

func TestDeck(t *testing.T) {
	t.Parallel()
	d := cards.Deck()
	if len(d) != 52 {
		t.Fatalf("a deck has 52 cards, not %d", len(d))
	}
	seen := map[cards.Card]bool{}
	for _, c := range d {
		if !c.Valid() || seen[c] {
			t.Fatalf("bad or repeated card %v", c)
		}
		seen[c] = true
	}
}

func TestShuffledIsSeeded(t *testing.T) {
	t.Parallel()
	a := cards.Shuffled(proto.NewRand(7, 1))
	b := cards.Shuffled(proto.NewRand(7, 1))
	c := cards.Shuffled(proto.NewRand(8, 1))
	if !slices.Equal(a, b) {
		t.Fatal("the same seed must deal the same deck")
	}
	if slices.Equal(a, c) {
		t.Fatal("another seed should deal another deck")
	}
	sorted := slices.Clone(a)
	cards.Sort(sorted)
	want := cards.Deck()
	cards.Sort(want)
	if !slices.Equal(sorted, want) {
		t.Fatal("a shuffled deck holds the 52 cards")
	}
}

func TestStringAndParse(t *testing.T) {
	t.Parallel()
	for _, c := range cards.Deck() {
		got, ok := cards.Parse(c.String())
		if !ok || got != c {
			t.Errorf("Parse(%q) = %v, %v", c.String(), got, ok)
		}
	}
	for in, want := range map[string]cards.Card{
		"as":             {Rank: cards.Ace, Suit: cards.Spades},
		"10h":            {Rank: cards.Ten, Suit: cards.Hearts},
		"th":             {Rank: cards.Ten, Suit: cards.Hearts},
		"10 H":           {Rank: cards.Ten, Suit: cards.Hearts},
		"Q♠":             {Rank: cards.Queen, Suit: cards.Spades},
		"ace of spades":  {Rank: cards.Ace, Suit: cards.Spades},
		"seven of clubs": {Rank: 7, Suit: cards.Clubs},
		"2d":             {Rank: cards.Two, Suit: cards.Diamonds},
	} {
		if got, ok := cards.Parse(in); !ok || got != want {
			t.Errorf("Parse(%q) = %v, %v; want %v", in, got, ok, want)
		}
	}
	for _, in := range []string{"", "1H", "11H", "AX", "10", "H", "ACE", "QUEEN OF HEARTS AND", "5 OF"} {
		if c, ok := cards.Parse(in); ok {
			t.Errorf("Parse(%q) = %v; want no card", in, c)
		}
	}
}

func TestSortAndRemove(t *testing.T) {
	t.Parallel()
	h := []cards.Card{{Rank: 3, Suit: cards.Clubs}, {Rank: cards.Ace, Suit: cards.Hearts}, {Rank: cards.King, Suit: cards.Spades}, {Rank: 9, Suit: cards.Hearts}}
	cards.Sort(h)
	if got := cards.Format(h); got != "KS AH 9H 3C" {
		t.Fatalf("sorted: %s", got)
	}
	out := cards.Remove(h, cards.Card{Rank: cards.Ace, Suit: cards.Hearts})
	if cards.Format(out) != "KS 9H 3C" || cards.Format(h) != "KS AH 9H 3C" {
		t.Fatalf("Remove: %s (the original is %s)", cards.Format(out), cards.Format(h))
	}
	if cards.Index(out, cards.Card{Rank: 9, Suit: cards.Hearts}) != 1 || cards.Index(out, cards.Card{Rank: 2, Suit: cards.Hearts}) != -1 {
		t.Fatal("Index")
	}
}

func FuzzParse(f *testing.F) {
	for _, s := range []string{"AS", "10H", "ace of spades", "Q♠", "1", "  ", "OF OF"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		c, ok := cards.Parse(s)
		if ok && !c.Valid() {
			t.Fatalf("Parse(%q) returned an invalid card %v", s, c)
		}
	})
}

func mustHand(t *testing.T, s string) []cards.Card {
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

func TestTrick(t *testing.T) {
	t.Parallel()
	tr := cards.Trick{Leader: cards.North, Cards: mustHand(t, "10H KH 2S AD")}
	if tr.Seat(1) != cards.East || tr.Seat(3) != cards.West || !tr.Done() {
		t.Fatal("seats run clockwise from the leader")
	}
	if w := tr.Winner(cards.NoTrump); w != cards.East {
		t.Errorf("no trump: %s wins, want EAST", cards.SeatNames[w])
	}
	if w := tr.Winner(cards.Spades); w != cards.South {
		t.Errorf("spades trump: %s wins, want SOUTH", cards.SeatNames[w])
	}
	if c, ok := tr.Played(cards.West); !ok || c.String() != "AD" {
		t.Errorf("West played %v", c)
	}
	open := cards.Trick{Leader: cards.West, Cards: mustHand(t, "5C")}
	if open.Next() != cards.North {
		t.Fatal("North plays after West")
	}
	if _, ok := open.Played(cards.East); ok {
		t.Fatal("East has not played")
	}
	h := mustHand(t, "AS 3C 9D")
	if !cards.Follows(h, open, mustHand(t, "3C")[0]) || cards.Follows(h, open, mustHand(t, "AS")[0]) {
		t.Error("must follow suit when able")
	}
	void := mustHand(t, "AS 9D")
	if got := cards.Format(cards.Playable(void, open)); got != "AS 9D" {
		t.Errorf("a void may play anything: %s", got)
	}
	if cards.Follows(void, open, mustHand(t, "3C")[0]) {
		t.Error("a card not in hand is never playable")
	}
}
