package cards

// Trick-taking for Hearts and Bridge: four seats play in turn, clockwise.

// Seats at the table, clockwise. The player sits South.
const (
	South = iota
	West
	North
	East
	Seats
)

// SeatNames are the seats' names, by seat.
var SeatNames = [Seats]string{"SOUTH", "WEST", "NORTH", "EAST"}

// Trick is the cards played to one trick, in play order from the seat that led.
type Trick struct {
	Leader int
	Cards  []Card
}

// Seat returns the seat that played the i-th card.
func (t Trick) Seat(i int) int { return (t.Leader + i) % Seats }

// Next is the seat to play next.
func (t Trick) Next() int { return t.Seat(len(t.Cards)) }

// Done reports that every seat has played.
func (t Trick) Done() bool { return len(t.Cards) == Seats }

// Led is the suit led; ok is false before the lead.
func (t Trick) Led() (s Suit, ok bool) {
	if len(t.Cards) == 0 {
		return 0, false
	}
	return t.Cards[0].Suit, true
}

// Played returns the card seat played, if it has.
func (t Trick) Played(seat int) (Card, bool) {
	i := (seat - t.Leader + Seats) % Seats
	if i >= len(t.Cards) {
		return Card{}, false
	}
	return t.Cards[i], true
}

// NoTrump is the trump argument for a game without trumps.
const NoTrump = Suit(255)

// Winning is the index (in play order) of the card winning so far: the highest trump if
// any was played, else the highest card of the suit led.
func (t Trick) Winning(trump Suit) int {
	best := 0
	for i, c := range t.Cards[1:] {
		b := t.Cards[best]
		switch {
		case c.Suit == b.Suit && c.Rank > b.Rank:
			best = i + 1
		case c.Suit == trump && b.Suit != trump:
			best = i + 1
		}
	}
	return best
}

// Winner is the seat winning the trick so far.
func (t Trick) Winner(trump Suit) int { return t.Seat(t.Winning(trump)) }

// HasSuit reports whether hand holds a card of s.
func HasSuit(hand []Card, s Suit) bool {
	for _, c := range hand {
		if c.Suit == s {
			return true
		}
	}
	return false
}

// Follows reports whether playing c from hand to t follows suit: c is in hand, and it is
// of the suit led unless the hand has none of it.
func Follows(hand []Card, t Trick, c Card) bool {
	if Index(hand, c) < 0 {
		return false
	}
	led, ok := t.Led()
	return !ok || c.Suit == led || !HasSuit(hand, led)
}

// Playable lists the cards of hand that follow suit to t, in hand order.
func Playable(hand []Card, t Trick) []Card {
	var out []Card
	for _, c := range hand {
		if Follows(hand, t, c) {
			out = append(out, c)
		}
	}
	return out
}
