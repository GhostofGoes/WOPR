// Package blackjack is Black Jack against WOPR's dealer: one deck, the dealer stands on soft
// 17 (DefaultRules; Rules.StandSoft17 changes it), black jack pays 3 to 2, double down on
// any first two cards, and one split of a pair a round. The player sits down with $100 and
// plays rounds until they leave or run out of money.
package blackjack

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/cards"
	"github.com/GhostofGoes/WOPR/internal/prompt"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// dealerPause is the beat between the dealer's cards.
const dealerPause = 700 * time.Millisecond

// Script text, all original. A # is filled in when the line is shown.
var (
	lineStandS17  = script.Orig("THE DEALER STANDS ON SOFT 17. BLACK JACK PAYS 3 TO 2.")
	lineHitS17    = script.Orig("THE DEALER HITS SOFT 17. BLACK JACK PAYS 3 TO 2.")
	lineLimits    = script.Orig("BETS ARE WHOLE DOLLARS FROM # TO #. TYPE LEAVE TO CASH OUT.")
	lineMoney     = script.Orig("YOU HAVE #.")
	promptBet     = script.Orig("YOUR BET: ")
	lineBadBet    = script.Orig("A BET IS A WHOLE NUMBER OF DOLLARS FROM # TO #.")
	lineShort     = script.Orig("YOU HAVE ONLY #.")
	lineSameBet   = script.Orig("SAME BET: #.")
	lineShuffle   = script.Orig("WOPR SHUFFLES THE DECK.")
	lineDealer    = script.Orig("DEALER:")
	lineYou       = script.Orig("YOU:")
	lineHandN     = script.Orig("HAND #:")
	lineHole      = script.Orig("??")
	lineDescBJ    = script.Orig("BLACK JACK")
	lineDescBust  = script.Orig("BUST")
	lineDescSoft  = script.Orig("SOFT #")
	promptHit     = script.Orig("HIT OR STAND? ")
	promptDouble  = script.Orig("HIT, STAND OR DOUBLE? ")
	promptSplit   = script.Orig("HIT, STAND, DOUBLE OR SPLIT? ")
	lineActions   = script.Orig("H TO HIT, S TO STAND, D TO DOUBLE DOWN, P TO SPLIT A PAIR.")
	lineNoDouble  = script.Orig("YOU CAN DOUBLE ONLY ON YOUR FIRST TWO CARDS.")
	lineNoSplit   = script.Orig("ONLY A PAIR CAN BE SPLIT, ONCE A ROUND.")
	lineNoMoney   = script.Orig("YOU DO NOT HAVE THE MONEY TO COVER IT.")
	lineFinish    = script.Orig("FINISH THE HAND FIRST.")
	lineDealerBJ  = script.Orig("THE DEALER HAS BLACK JACK.")
	lineDraws     = script.Orig("DEALER DRAWS #.")
	lineDealerBst = script.Orig("THE DEALER BUSTS.")
	lineStandsOn  = script.Orig("THE DEALER STANDS ON #.")
	lineWin       = script.Orig("YOU WIN #.")
	lineBJWin     = script.Orig("BLACK JACK. YOU WIN #.")
	lineLose      = script.Orig("YOU LOSE #.")
	linePush      = script.Orig("PUSH.")
	lineBroke     = script.Orig("YOU ARE OUT OF MONEY. THE HOUSE THANKS YOU.")
	lineLeave     = script.Orig("YOU LEAVE THE TABLE WITH #.")
)

// Lines is every script block, for the provenance test.
var Lines = []script.Ls{
	lineStandS17, lineHitS17, lineLimits, lineMoney, promptBet, lineBadBet, lineShort, lineSameBet, lineShuffle,
	lineDealer, lineYou, lineHandN, lineHole, lineDescBJ, lineDescBust, lineDescSoft, promptHit, promptDouble,
	promptSplit, lineActions, lineNoDouble, lineNoSplit, lineNoMoney, lineFinish, lineDealerBJ, lineDraws,
	lineDealerBst, lineStandsOn, lineWin, lineBJWin, lineLose, linePush, lineBroke, lineLeave,
}

// fill replaces each # in l's text with the next arg.
func fill(l script.Ls, args ...string) string {
	text := l[0].Text
	for _, a := range args {
		text = strings.Replace(text, "#", a, 1)
	}
	return text
}

func say(lines ...string) proto.Output { return proto.Say{Lines: lines, Pace: proto.PaceSpeech} }

func table(lines ...string) proto.Output { return proto.Say{Lines: lines, Pace: proto.PaceTable} }

// money formats cents as dollars: $100, or $112.50 when there are cents.
func money(cents int) string {
	if cents%100 == 0 {
		return fmt.Sprintf("$%d", cents/100)
	}
	return fmt.Sprintf("$%d.%02d", cents/100, cents%100)
}

type phase uint8

const (
	betting phase = iota
	playing
)

// Game is Black Jack as a proto.Program.
type Game struct {
	rules   Rules
	rng     *rand.Rand
	shoe    []cards.Card
	phase   phase
	money   int // cents
	lastBet int // dollars
	hands   []Hand
	cur     int // the hand being played
	dealer  []cards.Card
}

// New returns a game with DefaultRules.
func New() games.Game { return NewWithRules(DefaultRules) }

// NewWithRules returns a game with the given rules.
func NewWithRules(r Rules) *Game { return &Game{rules: r} }

// Start implements proto.Program.
func (g *Game) Start(env proto.Env) []proto.Output {
	g.rng = proto.NewRand(env.Seed, 0)
	g.money = g.rules.Stake * 100
	house := lineStandS17
	if !g.rules.StandSoft17 {
		house = lineHitS17
	}
	return []proto.Output{
		say(house[0].Text, fill(lineLimits, money(g.rules.MinBet*100), money(g.rules.MaxBet*100)), fill(lineMoney, money(g.money))),
		proto.Prompt{Text: promptBet[0].Text},
	}
}

// View implements proto.Program; Black Jack is console text only.
func (g *Game) View(*proto.Canvas) {}

// Handle implements proto.Program.
func (g *Game) Handle(ev proto.Event) []proto.Output {
	line, ok := ev.(proto.LineEvent)
	if !ok {
		return nil
	}
	if g.phase == betting {
		return g.onBet(line.Text)
	}
	return g.onAction(line.Text)
}

// leaving reports a request to cash out. QUIT, EXIT and LOGOFF never get here: they end the
// session from anywhere (the host's commands).
func leaving(norm string) bool {
	switch norm {
	case "LEAVE", "L", "DONE", "STOP", "NO", "CASH OUT", "CASH IN", "I QUIT", "I'M DONE":
		return true
	}
	return false
}

func (g *Game) askBet(outs ...proto.Output) []proto.Output {
	return append(outs, proto.Prompt{Text: promptBet[0].Text})
}

func (g *Game) onBet(input string) []proto.Output {
	norm := prompt.Normalize(input)
	if leaving(norm) {
		return g.leave()
	}
	var outs []proto.Output
	bet, ok := prompt.Number(input)
	if norm == "" {
		if g.lastBet == 0 || g.lastBet*100 > g.money {
			return g.askBet()
		}
		bet, ok = g.lastBet, true
		outs = append(outs, say(fill(lineSameBet, money(bet*100))))
	}
	switch {
	case !ok || bet < g.rules.MinBet || bet > g.rules.MaxBet:
		return g.askBet(say(fill(lineBadBet, money(g.rules.MinBet*100), money(g.rules.MaxBet*100))))
	case bet*100 > g.money:
		return g.askBet(say(fill(lineShort, money(g.money))))
	}
	g.lastBet = bet
	return append(outs, g.deal(bet)...)
}

// draw takes the top card. An empty deck (a long round) is refilled with the cards that are
// not on the table.
func (g *Game) draw() cards.Card {
	if len(g.shoe) == 0 {
		g.shoe = g.offTable(cards.Shuffled(g.rng))
	}
	c := g.shoe[0]
	g.shoe = g.shoe[1:]
	return c
}

func (g *Game) offTable(deck []cards.Card) []cards.Card {
	for _, c := range g.dealer {
		deck = cards.Remove(deck, c)
	}
	for _, h := range g.hands {
		for _, c := range h.Cards {
			deck = cards.Remove(deck, c)
		}
	}
	return deck
}

func (g *Game) deal(bet int) []proto.Output {
	var outs []proto.Output
	if len(g.shoe) < g.rules.Reshuffle {
		g.shoe = cards.Shuffled(g.rng)
		outs = append(outs, say(lineShuffle[0].Text))
	}
	g.hands, g.cur, g.dealer = []Hand{{Bet: bet * 100}}, 0, nil
	h := &g.hands[0]
	h.Cards = append(h.Cards, g.draw())
	g.dealer = append(g.dealer, g.draw())
	h.Cards = append(h.Cards, g.draw())
	g.dealer = append(g.dealer, g.draw())
	outs = append(outs, table(g.dealerLine(true), g.handLine(0)))

	if Natural(h.Cards) || (Peeks(g.dealer[0]) && Natural(g.dealer)) {
		return append(outs, g.finish(false)...) // nothing to play: a natural settles at once
	}
	g.phase = playing
	return append(outs, g.ask())
}

// ask prompts for the current hand, offering only the plays it allows.
func (g *Game) ask() proto.Output {
	h := &g.hands[g.cur]
	text := promptHit[0].Text
	switch {
	case h.CanSplit(len(g.hands)):
		text = promptSplit[0].Text
	case h.CanDouble():
		text = promptDouble[0].Text
	}
	if len(g.hands) > 1 {
		text = fill(lineHandN, fmt.Sprint(g.cur+1)) + " " + text
	}
	return proto.Prompt{Text: text}
}

// committed is the money already bet this round.
func (g *Game) committed() int {
	n := 0
	for _, h := range g.hands {
		n += h.Bet
	}
	return n
}

func (g *Game) onAction(input string) []proto.Output {
	h := &g.hands[g.cur]
	switch norm := prompt.Normalize(input); norm {
	case "H", "HIT", "HIT ME", "CARD":
		h.Cards = append(h.Cards, g.draw())
		return g.after(table(g.handLine(g.cur)))
	case "S", "STAND", "STAY", "STICK", "HOLD":
		h.Stood = true
		return g.after()
	case "D", "DOUBLE", "DOUBLE DOWN", "DD":
		switch {
		case !h.CanDouble():
			return []proto.Output{say(lineNoDouble[0].Text), g.ask()}
		case g.committed()+h.Bet > g.money:
			return []proto.Output{say(lineNoMoney[0].Text), g.ask()}
		}
		h.Bet *= 2
		h.Doubled = true
		h.Cards = append(h.Cards, g.draw())
		return g.after(table(g.handLine(g.cur)))
	case "P", "SPLIT":
		switch {
		case !h.CanSplit(len(g.hands)):
			return []proto.Output{say(lineNoSplit[0].Text), g.ask()}
		case g.committed()+h.Bet > g.money:
			return []proto.Output{say(lineNoMoney[0].Text), g.ask()}
		}
		return g.split()
	default:
		if leaving(norm) {
			return []proto.Output{say(lineFinish[0].Text), g.ask()}
		}
		return []proto.Output{say(lineActions[0].Text), g.ask()}
	}
}

func (g *Game) split() []proto.Output {
	first := g.hands[0]
	a := Hand{Cards: []cards.Card{first.Cards[0], g.draw()}, Bet: first.Bet, FromSplit: true}
	b := Hand{Cards: []cards.Card{first.Cards[1], g.draw()}, Bet: first.Bet, FromSplit: true}
	g.hands = []Hand{a, b}
	return g.after(table(g.handLine(0), g.handLine(1)))
}

// after moves on once the current hand is done: to the next hand, or to the dealer.
func (g *Game) after(outs ...proto.Output) []proto.Output {
	for g.cur < len(g.hands) && g.hands[g.cur].Done() {
		g.cur++
	}
	if g.cur < len(g.hands) {
		return append(outs, g.ask())
	}
	return append(outs, g.finish(true)...)
}

// finish turns the hole card, plays the dealer's hand if any of the player's hands still
// stands, and settles every hand.
func (g *Game) finish(dealerPlays bool) []proto.Output {
	outs := []proto.Output{table(g.dealerLine(false))}
	if Natural(g.dealer) {
		outs = append(outs, say(lineDealerBJ[0].Text))
		dealerPlays = false
	}
	live := false
	for i := range g.hands {
		live = live || !g.hands[i].Bust()
	}
	if dealerPlays && live {
		for g.rules.DealerHits(g.dealer) {
			c := g.draw()
			g.dealer = append(g.dealer, c)
			outs = append(outs, proto.Wait{D: dealerPause}, say(fill(lineDraws, c.String())), table(g.dealerLine(false)))
		}
		if total, _ := Value(g.dealer); total > 21 {
			outs = append(outs, say(lineDealerBst[0].Text))
		} else {
			outs = append(outs, say(fill(lineStandsOn, describe(g.dealer))))
		}
	}
	for i, h := range g.hands {
		net, v := Settle(h, g.dealer)
		g.money += net
		text := ""
		switch v {
		case Blackjack:
			text = fill(lineBJWin, money(net))
		case Win:
			text = fill(lineWin, money(net))
		case Lose:
			text = fill(lineLose, money(-net))
		case Push:
			text = linePush[0].Text
		}
		if len(g.hands) > 1 {
			text = fill(lineHandN, fmt.Sprint(i+1)) + " " + text
		}
		outs = append(outs, say(text))
	}
	outs = append(outs, say(fill(lineMoney, money(g.money)), ""))
	g.phase = betting
	if g.money < g.rules.MinBet*100 {
		return append(outs, say(lineBroke[0].Text), proto.Done{Result: g.result()})
	}
	return g.askBet(outs...)
}

func (g *Game) leave() []proto.Output {
	return []proto.Output{say(fill(lineLeave, money(g.money))), proto.Done{Result: g.result()}}
}

// result compares what the player leaves with to the stake.
func (g *Game) result() proto.Result {
	switch {
	case g.money > g.rules.Stake*100:
		return proto.Result{Outcome: proto.Win}
	case g.money < g.rules.Stake*100:
		return proto.Result{Outcome: proto.Loss}
	}
	return proto.Result{Outcome: proto.Draw}
}

// describe is a hand's total as the table says it: 18, SOFT 18, BLACK JACK or BUST.
func describe(h []cards.Card) string {
	total, soft := Value(h)
	switch {
	case Natural(h):
		return lineDescBJ[0].Text
	case total > 21:
		return lineDescBust[0].Text
	case soft:
		return fill(lineDescSoft, fmt.Sprint(total))
	}
	return fmt.Sprint(total)
}

func row(label string, hand string, desc string) string {
	if desc == "" {
		return fmt.Sprintf("%-8s %s", label, hand)
	}
	return fmt.Sprintf("%-8s %-20s (%s)", label, hand, desc)
}

// dealerLine shows the dealer's hand, with the hole card face down while hidden.
func (g *Game) dealerLine(hidden bool) string {
	if hidden {
		return row(lineDealer[0].Text, g.dealer[0].String()+" "+lineHole[0].Text, "")
	}
	return row(lineDealer[0].Text, cards.Format(g.dealer), describe(g.dealer))
}

func (g *Game) handLine(i int) string {
	label := lineYou[0].Text
	if len(g.hands) > 1 {
		label = fill(lineHandN, fmt.Sprint(i+1))
	}
	h := g.hands[i]
	if h.FromSplit && Natural(h.Cards) {
		return row(label, cards.Format(h.Cards), "21") // not a black jack after a split
	}
	return row(label, cards.Format(h.Cards), describe(h.Cards))
}
