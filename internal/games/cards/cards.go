// Package cards is the playing cards the card games share: cards and their text, decks,
// shuffling with the session's seed, and tricks (trick.go).
//
// Cards print as a rank and a suit letter, in ASCII so that every terminal shows them at
// one cell a character: AS, 10H, QD, 7C.
package cards

import (
	"math/rand/v2"
	"slices"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/prompt"
)

// Suit is a card's suit, in bridge order (clubs lowest).
type Suit uint8

// Suits.
const (
	Clubs Suit = iota
	Diamonds
	Hearts
	Spades
)

// Suits lists every suit, lowest first.
var Suits = [4]Suit{Clubs, Diamonds, Hearts, Spades}

// Letter is the suit's one-letter name.
func (s Suit) Letter() string { return string("CDHS"[s]) }

// String is the suit's name.
func (s Suit) String() string {
	return [4]string{"CLUBS", "DIAMONDS", "HEARTS", "SPADES"}[s]
}

// Rank is a card's rank: 2 to 10, then jack (11) to ace (14). Aces are high; a game that
// counts them low (Black Jack, Gin Rummy) says so itself.
type Rank uint8

// Ranks.
const (
	Two   Rank = 2
	Ten   Rank = 10
	Jack  Rank = 11
	Queen Rank = 12
	King  Rank = 13
	Ace   Rank = 14
)

// String is the rank as a card shows it: 2 to 10, J, Q, K, A.
func (r Rank) String() string {
	switch r {
	case Jack:
		return "J"
	case Queen:
		return "Q"
	case King:
		return "K"
	case Ace:
		return "A"
	case Ten:
		return "10"
	}
	return string(rune('0' + r))
}

// Card is one playing card.
type Card struct {
	Rank Rank
	Suit Suit
}

// String is the card's short form, such as AS or 10H.
func (c Card) String() string { return c.Rank.String() + c.Suit.Letter() }

// Valid reports whether c is one of the 52 cards.
func (c Card) Valid() bool { return c.Rank >= Two && c.Rank <= Ace && c.Suit <= Spades }

// Deck returns the 52 cards in order: clubs, diamonds, hearts, spades, each two to ace.
func Deck() []Card {
	out := make([]Card, 0, 52)
	for _, s := range Suits {
		for r := Two; r <= Ace; r++ {
			out = append(out, Card{r, s})
		}
	}
	return out
}

// Shuffled returns a new deck in a random order drawn from r.
func Shuffled(r *rand.Rand) []Card {
	d := Deck()
	r.Shuffle(len(d), func(i, j int) { d[i], d[j] = d[j], d[i] })
	return d
}

// Format joins cards with spaces.
func Format(cs []Card) string {
	parts := make([]string, len(cs))
	for i, c := range cs {
		parts[i] = c.String()
	}
	return strings.Join(parts, " ")
}

// Sort orders a hand for display: by suit, spades first, then by rank, highest first.
func Sort(cs []Card) {
	slices.SortFunc(cs, func(a, b Card) int {
		if a.Suit != b.Suit {
			return int(b.Suit) - int(a.Suit)
		}
		return int(b.Rank) - int(a.Rank)
	})
}

// Index returns the position of c in cs, or -1.
func Index(cs []Card, c Card) int { return slices.Index(cs, c) }

// Remove returns cs without its first c, in a new slice.
func Remove(cs []Card, c Card) []Card {
	out := make([]Card, 0, len(cs))
	removed := false
	for _, x := range cs {
		if x == c && !removed {
			removed = true
			continue
		}
		out = append(out, x)
	}
	return out
}

var (
	rankWords = map[string]Rank{
		"A": Ace, "ACE": Ace, "K": King, "KING": King, "Q": Queen, "QUEEN": Queen, "J": Jack, "JACK": Jack,
		"T": Ten, "TEN": Ten, "NINE": 9, "EIGHT": 8, "SEVEN": 7, "SIX": 6, "FIVE": 5, "FOUR": 4, "THREE": 3, "TWO": Two,
	}
	suitWords = map[string]Suit{
		"C": Clubs, "CLUB": Clubs, "CLUBS": Clubs, "D": Diamonds, "DIAMOND": Diamonds, "DIAMONDS": Diamonds,
		"H": Hearts, "HEART": Hearts, "HEARTS": Hearts, "S": Spades, "SPADE": Spades, "SPADES": Spades,
	}
	suitSymbols = strings.NewReplacer("♣", " C", "♧", " C", "♦", " D", "♢", " D", "♥", " H", "♡", " H", "♠", " S", "♤", " S")
)

// Parse reads a card as typed: AS, 10H, TH, 10 H, Q♠, or ACE OF SPADES. Case does not matter.
func Parse(s string) (Card, bool) {
	words := strings.Fields(prompt.Normalize(suitSymbols.Replace(s)))
	words = slices.DeleteFunc(words, func(w string) bool { return w == "OF" })
	switch len(words) {
	case 1: // AS, 10H, TH; not ACES, whose last letter is no suit
		w := words[0]
		if len(w) < 2 || len(w) > 3 {
			return Card{}, false
		}
		words = []string{w[:len(w)-1], w[len(w)-1:]}
	case 2:
	default:
		return Card{}, false
	}
	r, ok := parseRank(words[0])
	if !ok {
		return Card{}, false
	}
	s2, ok := suitWords[words[1]]
	if !ok {
		return Card{}, false
	}
	return Card{r, s2}, true
}

func parseRank(w string) (Rank, bool) {
	if r, ok := rankWords[w]; ok {
		return r, true
	}
	n, ok := prompt.Number(w)
	if !ok || n < 2 || n > 10 {
		return 0, false
	}
	return Rank(n), true
}
