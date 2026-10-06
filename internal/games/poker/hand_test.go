package poker

import (
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games/cards"
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

// Every five-card hand, classified: the counts are the textbook ones.
func TestCategoryCountsOverEveryHand(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("2,598,960 hands")
	}
	want := map[Category]int{
		StraightFlush: 40, Quads: 624, FullHouse: 3744, Flush: 5108, Straight: 10200,
		Trips: 54912, TwoPair: 123552, OnePair: 1098240, HighCard: 1302540,
	}
	got := map[Category]int{}
	d := cards.Deck()
	h := make([]cards.Card, 5)
	for a := 0; a < 52; a++ {
		for b := a + 1; b < 52; b++ {
			for c := b + 1; c < 52; c++ {
				for e := c + 1; e < 52; e++ {
					for f := e + 1; f < 52; f++ {
						h[0], h[1], h[2], h[3], h[4] = d[a], d[b], d[c], d[e], d[f]
						got[Evaluate(h).Category()]++
					}
				}
			}
		}
	}
	for cat, n := range want {
		if got[cat] != n {
			t.Errorf("category %d: %d hands, want %d", cat, got[cat], n)
		}
	}
}

func TestOrdering(t *testing.T) {
	t.Parallel()
	// Each hand beats the next.
	ladder := []string{
		"AS KS QS JS 10S", // royal flush
		"9H 8H 7H 6H 5H",  // straight flush
		"5D 4D 3D 2D AD",  // steel wheel: the lowest straight flush
		"7S 7H 7D 7C 2S",  // quads
		"6S 6H 6D 6C AS",  // lower quads, higher kicker
		"KS KH KD 2C 2S",  // full house
		"QS QH QD AC AS",  // lower trips over higher pair
		"AD JD 9D 4D 3D",  // flush
		"AH JH 9H 4H 2H",  // a lower fifth card
		"KS QH JD 10C 9S", // straight
		"6S 5H 4D 3C 2S",  // six-high straight
		"5S 4H 3D 2C AS",  // the wheel: five high
		"8S 8H 8D AC KS",  // trips
		"JS JH 4D 4C AS",  // two pair
		"JS JH 3D 3C AS",  // lower second pair
		"JS JH 3D 3C KS",  // lower kicker
		"AS AH KD QC JS",  // a pair of aces
		"AS AH KD QC 10S", // lower third kicker
		"2S 2H 5D 4C 3S",  // the lowest pair
		"AS KH QD JC 9S",  // ace high
		"7S 5H 4D 3C 2S",  // the lowest hand
	}
	for i := 0; i+1 < len(ladder); i++ {
		a, b := Evaluate(hand(t, ladder[i])), Evaluate(hand(t, ladder[i+1]))
		if a <= b {
			t.Errorf("%s (%s) should beat %s (%s)", ladder[i], a.Name(), ladder[i+1], b.Name())
		}
	}
	if Evaluate(hand(t, "AS KH QD JC 9S")) != Evaluate(hand(t, "AH KD QC JS 9D")) {
		t.Error("suits never break a tie")
	}
}

func TestNames(t *testing.T) {
	t.Parallel()
	for h, want := range map[string]string{
		"AS KS QS JS 10S": "A ROYAL FLUSH",
		"5D 4D 3D 2D AD":  "A STRAIGHT FLUSH, FIVE HIGH",
		"7S 7H 7D 7C 2S":  "FOUR SEVENS",
		"KS KH KD 2C 2S":  "A FULL HOUSE, KINGS OVER TWOS",
		"AH JH 9H 4H 2H":  "A FLUSH, ACE HIGH",
		"5S 4H 3D 2C AS":  "A STRAIGHT, FIVE HIGH",
		"8S 8H 8D AC KS":  "THREE EIGHTS",
		"JS JH 4D 4C AS":  "TWO PAIR, JACKS AND FOURS",
		"6S 6H KD 4C AS":  "A PAIR OF SIXES",
		"QS 10H 7D 4C 2S": "QUEEN HIGH",
	} {
		if got := Evaluate(hand(t, h)).Name(); got != want {
			t.Errorf("%s: %q, want %q", h, got, want)
		}
	}
}

func TestDraws(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		hand    string
		discard []int
	}{
		{"AS KS QS JS 10S", nil},
		{"KS KH KD 2C 2S", nil},
		{"8S 8H 8D AC KS", []int{3, 4}},
		{"JS JH 4D 4C AS", []int{4}},
		{"6S 6H KD 4C AS", []int{2, 3, 4}},
		{"AH JH 9H 4H 2S", []int{4}},       // four to a flush
		{"9S 8H 7D 6C 2S", []int{4}},       // open-ended straight
		{"AS KH QD JC 2S", []int{2, 3, 4}}, // A-K-Q-J is not open at both ends: keep the top two
		{"QS 10H 7D 4C 2S", []int{2, 3, 4}},
	} {
		got := discards(hand(t, tc.hand))
		if len(got) != len(tc.discard) {
			t.Errorf("%s: discards %v, want %v", tc.hand, got, tc.discard)
			continue
		}
		for i := range got {
			if got[i] != tc.discard[i] {
				t.Errorf("%s: discards %v, want %v", tc.hand, got, tc.discard)
				break
			}
		}
	}
}
