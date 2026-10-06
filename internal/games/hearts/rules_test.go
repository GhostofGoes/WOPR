package hearts

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

func card(t testing.TB, s string) cards.Card { return hand(t, s)[0] }

func TestCheck(t *testing.T) {
	t.Parallel()
	h := hand(t, "2C 9C QS AH 5D")
	empty := cards.Trick{Leader: cards.South}
	clubLed := cards.Trick{Leader: cards.West, Cards: hand(t, "KC")}
	diamondLed := cards.Trick{Leader: cards.West, Cards: hand(t, "KD")}
	spadeLed := cards.Trick{Leader: cards.West, Cards: hand(t, "3S")}
	for _, tc := range []struct {
		name          string
		hand          []cards.Card
		trick         cards.Trick
		play          string
		first, broken bool
		want          Refusal
	}{
		{"2C leads the first trick", h, empty, "9C", true, false, MustLeadTwo},
		{"2C may lead", h, empty, "2C", true, false, OK},
		{"hearts not broken", h, empty, "AH", false, false, NotBroken},
		{"hearts broken", h, empty, "AH", false, true, OK},
		{"only hearts left", hand(t, "AH 3H"), empty, "3H", false, false, OK},
		{"follow suit", h, clubLed, "5D", false, false, MustFollow},
		{"following", h, clubLed, "9C", true, false, OK},
		{"not held", h, clubLed, "8C", false, false, NotHeld},
		{"void may discard", hand(t, "9C QS AH"), diamondLed, "9C", true, false, OK},
		{"no points on the first trick", hand(t, "9C QS AH"), diamondLed, "AH", true, false, NoPointsFirst},
		{"the queen counts too", hand(t, "9C QS AH"), diamondLed, "QS", true, false, NoPointsFirst},
		{"points when nothing else", hand(t, "QS AH"), diamondLed, "AH", true, false, OK},
		{"queen on a later spade trick", h, spadeLed, "QS", false, false, OK},
	} {
		if got := Check(tc.hand, tc.trick, card(t, tc.play), tc.first, tc.broken); got != tc.want {
			t.Errorf("%s: %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestSettleAndPass(t *testing.T) {
	t.Parallel()
	add, moon := Settle([4]int{3, 13, 10, 0})
	if moon != -1 || add != [4]int{3, 13, 10, 0} {
		t.Errorf("plain hand: %v %d", add, moon)
	}
	add, moon = Settle([4]int{0, 0, 26, 0})
	if moon != cards.North || add != [4]int{26, 26, 0, 26} {
		t.Errorf("moon: %v %d", add, moon)
	}
	var dirs [8]int
	for d := range dirs {
		dirs[d] = Pass(d)
	}
	if dirs != [8]int{1, 3, 2, 0, 1, 3, 2, 0} {
		t.Errorf("passing cycles left, right, across, hold: %v", dirs)
	}
}

// Legality over 200 seeded deals: every seat played by WOPR's heuristics, every card
// checked against the rules, 13 tricks, 26 points.
func TestLegalityOverSeededDeals(t *testing.T) {
	t.Parallel()
	for seed := range uint64(200) {
		deck := cards.Shuffled(proto.NewRand(seed, 0))
		var hands [cards.Seats][]cards.Card
		for s := range hands {
			hands[s] = append([]cards.Card(nil), deck[13*s:13*s+13]...)
		}
		dir := Pass(int(seed))
		var passes [cards.Seats][]cards.Card
		for s := range hands {
			passes[s] = choosePass(hands[s])
			if len(passes[s]) != 3 {
				t.Fatalf("seed %d: passed %d cards", seed, len(passes[s]))
			}
		}
		if dir != 0 {
			for s := range hands {
				for _, c := range passes[s] {
					hands[s] = cards.Remove(hands[s], c)
				}
			}
			for s := range hands {
				hands[(s+dir)%cards.Seats] = append(hands[(s+dir)%cards.Seats], passes[s]...)
			}
		}
		leader := -1
		for s := range hands {
			if cards.Index(hands[s], twoClubs) >= 0 {
				leader = s
			}
		}
		var taken [cards.Seats]int
		broken := false
		for n := range 13 {
			tr := cards.Trick{Leader: leader}
			for range cards.Seats {
				s := tr.Next()
				legal := Legal(hands[s], tr, n == 0, broken)
				if len(legal) == 0 {
					t.Fatalf("seed %d trick %d: %s has no legal card in %s", seed, n, cards.SeatNames[s], cards.Format(hands[s]))
				}
				c := choosePlay(hands[s], tr, legal)
				if why := Check(hands[s], tr, c, n == 0, broken); why != OK {
					t.Fatalf("seed %d trick %d: %s played %s illegally (%d)", seed, n, cards.SeatNames[s], c, why)
				}
				hands[s] = cards.Remove(hands[s], c)
				tr.Cards = append(tr.Cards, c)
				broken = broken || c.Suit == cards.Hearts
			}
			leader = tr.Winner(cards.NoTrump)
			taken[leader] += pointsOf(tr.Cards)
		}
		if taken[0]+taken[1]+taken[2]+taken[3] != 26 {
			t.Fatalf("seed %d: %v points", seed, taken)
		}
	}
}

func TestAIChoices(t *testing.T) {
	t.Parallel()
	if got := cards.Format(choosePass(hand(t, "QS KS 3S 2C 4C 5C 6D 7D 8D 9H 2H JD 10C"))); !strings.Contains(got, "QS") || !strings.Contains(got, "KS") {
		t.Errorf("passing should shed the queen and an unguarded king: %s", got)
	}
	// Following spades under the king with the queen: dump the queen.
	tr := cards.Trick{Leader: cards.West, Cards: hand(t, "KS")}
	h := hand(t, "QS 3S 9D")
	if c := choosePlay(h, tr, Legal(h, tr, false, false)); c.String() != "QS" {
		t.Errorf("ducking under the king should play the queen: %s", c)
	}
	// Void in diamonds: the queen goes.
	tr = cards.Trick{Leader: cards.West, Cards: hand(t, "KD")}
	h = hand(t, "QS AH 9C")
	if c := choosePlay(h, tr, Legal(h, tr, false, false)); c.String() != "QS" {
		t.Errorf("void: dump the queen: %s", c)
	}
}
