// Package bridge is minimal contract bridge (docs/PLAN.md §3, decision 14): WOPR bids all
// four hands by point count, the player's side always declares (the table turns when East
// and West hold the cards), and the player plays both declarer's hand and dummy's while
// WOPR defends. One deal is one game: making the contract wins it.
package bridge

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/cards"
	"github.com/GhostofGoes/WOPR/internal/prompt"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// maxDeals bounds redeals of passed-out hands (a passed-out deal is rare; a run of them
// rarer still).
const maxDeals = 50

// Game is bridge as a proto.Program.
type Game struct {
	rng      *rand.Rand
	dealer   int
	hands    [cards.Seats][]cards.Card
	calls    []Bid
	contract Contract
	trick    cards.Trick
	last     cards.Trick
	lastBy   int
	won      [2]int // tricks by side: North-South, East-West
	over     bool
}

// New returns a game.
func New() games.Game { return &Game{lastBy: -1} }

func say(lines ...string) proto.Output { return proto.Say{Lines: lines, Pace: proto.PaceSpeech} }

// Start implements proto.Program: deal until someone can open, turn the table if East and
// West hold more, bid, and lead.
func (g *Game) Start(env proto.Env) []proto.Output {
	g.rng = proto.NewRand(env.Seed, 0)
	outs := []proto.Output{say(lineRules.Texts()...)}
	turned := false
	for range maxDeals {
		turned = g.deal()
		var ok bool
		g.calls, g.contract, ok = Auction(g.hands, g.dealer)
		if ok {
			break
		}
		outs = append(outs, say(fill(lineAuction, cards.SeatNames[g.dealer], g.auctionText()), linePassedOut[0].Text))
		g.dealer = (g.dealer + 1) % cards.Seats
	}
	if turned {
		outs = append(outs, say(lineTurned[0].Text))
	}
	d := g.contract.Declarer
	g.trick = cards.Trick{Leader: (d + 1) % cards.Seats}
	outs = append(outs, say(
		fill(lineAuction, cards.SeatNames[g.dealer], g.auctionText()),
		fill(lineContract, g.contract.String(), cards.SeatNames[d], cards.SeatNames[g.trick.Leader]),
	))
	return g.run(outs)
}

// deal shuffles and deals, then turns the table so that North-South hold the most points.
// It reports whether it turned.
func (g *Game) deal() bool {
	deck := cards.Shuffled(g.rng)
	for s := range g.hands {
		g.hands[s] = slices.Clone(deck[13*s : 13*s+13])
		cards.Sort(g.hands[s])
	}
	ns := HCP(g.hands[cards.North]) + HCP(g.hands[cards.South])
	ew := HCP(g.hands[cards.East]) + HCP(g.hands[cards.West])
	if ew > ns || (ew == ns && side(g.dealer) == 1) {
		var turned [cards.Seats][]cards.Card
		for s, h := range g.hands {
			turned[(s+1)%cards.Seats] = h // West to North, East to South
		}
		g.hands = turned
		g.dealer = (g.dealer + 1) % cards.Seats
		return true
	}
	return false
}

// Contract text: 4S, 3NT.
func (c Contract) String() string { return c.Bid.String() }

func (g *Game) auctionText() string {
	parts := make([]string, len(g.calls))
	for i, b := range g.calls {
		parts[i] = b.String()
	}
	return strings.Join(parts, " ")
}

// View implements proto.Program.
func (g *Game) View(c *proto.Canvas) { g.draw(c) }

// Handle implements proto.Program.
func (g *Game) Handle(ev proto.Event) []proto.Output {
	line, ok := ev.(proto.LineEvent)
	if !ok || g.over {
		return nil
	}
	return g.onPlay(line.Text)
}

func (g *Game) dummy() int { return partner(g.contract.Declarer) }

// ask prompts for the seat to play, which is declarer or dummy.
func (g *Game) ask() proto.Output {
	seat := g.trick.Next()
	if seat == g.dummy() {
		return proto.Prompt{Text: fill(promptDummy, cards.SeatNames[seat])}
	}
	return proto.Prompt{Text: fill(promptHand, cards.SeatNames[seat])}
}

// run plays WOPR's defenders until declarer or dummy is to play, or the deal ends.
func (g *Game) run(outs []proto.Output) []proto.Output {
	trump := g.contract.Strain.Trump()
	for {
		if g.trick.Done() {
			outs = append(outs, g.takeTrick())
			if g.won[0]+g.won[1] == 13 {
				return g.end(outs)
			}
		}
		seat := g.trick.Next()
		if side(seat) == 0 {
			return append(outs, proto.Redraw{}, g.ask())
		}
		g.play(seat, defend(g.hands[seat], g.trick, trump, seat))
	}
}

func (g *Game) play(seat int, c cards.Card) {
	g.hands[seat] = cards.Remove(g.hands[seat], c)
	g.trick.Cards = append(g.trick.Cards, c)
}

func (g *Game) takeTrick() proto.Output {
	w := g.trick.Winner(g.contract.Strain.Trump())
	g.won[side(w)]++
	g.last, g.lastBy = g.trick, w
	g.trick = cards.Trick{Leader: w}
	return proto.Say{Lines: []string{trickText(g.last, w)}, Pace: proto.PaceTable}
}

func trickText(t cards.Trick, winner int) string {
	var parts []string
	for i, c := range t.Cards {
		parts = append(parts, cards.SeatNames[t.Seat(i)]+" "+c.String())
	}
	return fill(lineTrick, strings.Join(parts, ", "), cards.SeatNames[winner])
}

func (g *Game) onPlay(input string) []proto.Output {
	seat := g.trick.Next()
	name := cards.SeatNames[seat]
	again := func(text string) []proto.Output { return []proto.Output{say(text), g.ask()} }
	norm := prompt.Normalize(input)
	switch norm {
	case "LEAVE", "STOP", "DONE", "I'M DONE", "I QUIT":
		return again(lineFinish[0].Text)
	}
	hand := g.hands[seat]
	var c cards.Card
	if n, ok := prompt.MenuChoice(input, len(hand)); ok {
		c = hand[n-1]
	} else if parsed, ok := cards.Parse(input); ok {
		c = parsed
	} else {
		return again(fill(linePlayHelp, name))
	}
	switch {
	case cards.Index(hand, c) < 0:
		return again(fill(lineNotHeld, name))
	case !cards.Follows(hand, g.trick, c):
		led, _ := g.trick.Led()
		return again(fill(lineFollow, name, led.String()))
	}
	g.play(seat, c)
	return g.run(nil)
}

func (g *Game) end(outs []proto.Output) []proto.Output {
	g.over = true
	score := Score(g.contract, g.won[0])
	need := 6 + g.contract.Level
	text := fill(lineMade, fmt.Sprint(g.won[0]), fmt.Sprint(score))
	outcome := proto.Win
	if g.won[0] < need {
		text = fill(lineDown, fmt.Sprint(need-g.won[0]), fmt.Sprint(score))
		outcome = proto.Loss
	}
	return append(outs, proto.Redraw{}, proto.Done{Result: proto.Result{Outcome: outcome, Lines: []string{text}}})
}

// defend is WOPR's play for a defender: second hand low, third and fourth hand win as
// cheaply as they can unless partner already is, ruff when void and partner is not
// winning, and lead the top of an honour sequence or low from the longest suit.
func defend(hand []cards.Card, t cards.Trick, trump cards.Suit, seat int) cards.Card {
	legal := cards.Playable(hand, t)
	if len(legal) == 1 {
		return legal[0]
	}
	led, ok := t.Led()
	if !ok {
		return lead(hand, trump)
	}
	winning := t.Winning(trump)
	partnerWins := t.Seat(winning) == partner(seat)
	follows := cards.HasSuit(hand, led)
	switch {
	case follows && (len(t.Cards) == 1 || partnerWins):
		return lowestOf(legal)
	case !follows && partnerWins:
		return discard(legal, trump)
	}
	best := t.Cards[winning]
	var winners []cards.Card
	for _, c := range legal {
		if (c.Suit == best.Suit && c.Rank > best.Rank) || (c.Suit == trump && best.Suit != trump) {
			winners = append(winners, c)
		}
	}
	switch {
	case len(winners) > 0:
		return lowestOf(winners)
	case follows:
		return lowestOf(legal)
	}
	return discard(legal, trump)
}

// lead picks an opening card: the ace from ace-king, else the top of a sequence of
// honours, else the fourth best (or the lowest) of the longest suit outside trumps.
func lead(hand []cards.Card, trump cards.Suit) cards.Card {
	var best []cards.Card
	for _, s := range cards.Suits {
		var suit []cards.Card
		for _, c := range hand {
			if c.Suit == s {
				suit = append(suit, c)
			}
		}
		if len(suit) == 0 || (s == trump && len(suit) < len(hand)) {
			continue
		}
		if len(suit) > len(best) {
			best = suit
		}
	}
	if len(best) == 0 { // only trumps
		best = hand
	}
	slices.SortFunc(best, func(a, b cards.Card) int { return int(b.Rank) - int(a.Rank) })
	if len(best) >= 2 && best[0].Rank >= cards.Queen && best[1].Rank == best[0].Rank-1 {
		return best[0]
	}
	if len(best) >= 4 {
		return best[3]
	}
	return best[len(best)-1]
}

// discard throws the lowest card outside trumps, or the lowest trump.
func discard(legal []cards.Card, trump cards.Suit) cards.Card {
	var side []cards.Card
	for _, c := range legal {
		if c.Suit != trump {
			side = append(side, c)
		}
	}
	if len(side) > 0 {
		return lowestOf(side)
	}
	return lowestOf(legal)
}

func lowestOf(cs []cards.Card) cards.Card {
	return slices.MinFunc(cs, func(a, b cards.Card) int { return int(a.Rank) - int(b.Rank) })
}
