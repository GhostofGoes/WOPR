// Package ginrummy is two-player gin rummy against WOPR: ten cards each, draw from the
// stock or take the discard, then discard; knock with 10 or less deadwood, or go gin with
// none. The defender lays off on the knocker's melds (not after gin); an undercut scores
// 25 more, gin 25. The first to 100 wins. Aces are low.
package ginrummy

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

// Seats.
const (
	player = 0
	wopr   = 1
)

// stockFloor is where the hand ends without a score: the last two cards are never drawn.
const stockFloor = 2

type phase uint8

const (
	drawing phase = iota
	discarding
	between
)

// Game is gin rummy as a proto.Program.
type Game struct {
	rng     *rand.Rand
	phase   phase
	stock   []cards.Card
	pile    []cards.Card // the discard pile; the top is last
	hands   [2][]cards.Card
	score   [2]int
	dealer  int
	took    cards.Card // the card taken from the pile this turn, which may not go straight back
	hasTook bool
	note    string // the last thing WOPR did, for the panel
}

// New returns a game.
func New() games.Game { return &Game{} }

func say(lines ...string) proto.Output { return proto.Say{Lines: lines, Pace: proto.PaceSpeech} }

func table(lines ...string) proto.Output { return proto.Say{Lines: lines, Pace: proto.PaceTable} }

// Start implements proto.Program.
func (g *Game) Start(env proto.Env) []proto.Output {
	g.rng = proto.NewRand(env.Seed, 0)
	g.dealer = player // WOPR deals the first hand
	return g.newHand([]proto.Output{say(lineRules.Texts()...)})
}

// Handle implements proto.Program.
func (g *Game) Handle(ev proto.Event) []proto.Output {
	line, ok := ev.(proto.LineEvent)
	if !ok {
		return nil
	}
	switch g.phase {
	case drawing:
		return g.onDraw(line.Text)
	case discarding:
		return g.onDiscard(line.Text)
	default:
		return g.onNext(line.Text)
	}
}

func (g *Game) newHand(outs []proto.Output) []proto.Output {
	g.dealer = 1 - g.dealer
	deck := cards.Shuffled(g.rng)
	g.hands = [2][]cards.Card{}
	for range 10 {
		for _, p := range []int{1 - g.dealer, g.dealer} {
			g.hands[p] = append(g.hands[p], deck[0])
			deck = deck[1:]
		}
	}
	g.pile, g.stock = deck[:1], deck[1:]
	g.phase, g.hasTook, g.note = drawing, false, ""
	g.order()
	deals := lineWOPRDeals
	if g.dealer == player {
		deals = lineYouDeal
	}
	outs = append(outs, say(deals[0].Text), proto.Redraw{})
	if g.dealer == player {
		outs = append(outs, g.woprTurn()...)
		if g.phase == between {
			return outs
		}
	}
	return g.askDraw(outs)
}

// order sorts the player's hand for display: melds first, then deadwood.
func (g *Game) order() {
	a := Arrange(g.hands[player])
	var out []cards.Card
	for _, m := range a.Melds {
		out = append(out, m...)
	}
	g.hands[player] = append(out, a.Deadwood...)
}

func (g *Game) top() cards.Card { return g.pile[len(g.pile)-1] }

func (g *Game) drawStock(p int) cards.Card {
	c := g.stock[0]
	g.stock = g.stock[1:]
	g.hands[p] = append(g.hands[p], c)
	return c
}

func (g *Game) takePile(p int) cards.Card {
	c := g.top()
	g.pile = g.pile[:len(g.pile)-1]
	g.hands[p] = append(g.hands[p], c)
	g.took, g.hasTook = c, true
	return c
}

func (g *Game) discard(p int, c cards.Card) {
	g.hands[p] = cards.Remove(g.hands[p], c)
	g.pile = append(g.pile, c)
	g.hasTook = false
}

func (g *Game) askDraw(outs []proto.Output) []proto.Output {
	if len(g.stock) <= stockFloor {
		return g.void(outs)
	}
	g.phase = drawing
	return append(outs, proto.Redraw{}, proto.Prompt{Text: promptDraw[0].Text})
}

func (g *Game) onDraw(input string) []proto.Output {
	norm := prompt.Normalize(input)
	again := func(text string) []proto.Output {
		return []proto.Output{say(text), proto.Prompt{Text: promptDraw[0].Text}}
	}
	if c, ok := cards.Parse(input); ok {
		if c != g.top() {
			return again(fill(lineNotTop, g.top().String()))
		}
		norm = "DISCARD"
	}
	var drew string
	switch norm {
	case "S", "STOCK", "DECK", "DRAW", "THE STOCK", "FROM THE STOCK":
		drew = fill(lineYouDrew, g.drawStock(player).String())
	case "D", "DISCARD", "PILE", "TAKE", "TAKE IT", "THE DISCARD", "UP CARD":
		drew = fill(lineYouTook, g.takePile(player).String())
	default:
		if leaving(norm) {
			return again(lineFinish[0].Text)
		}
		return again(lineDrawHelp[0].Text)
	}
	g.order()
	g.phase = discarding
	return []proto.Output{say(drew), proto.Redraw{}, proto.Prompt{Text: promptDiscard[0].Text}}
}

// onDiscard reads a discard by name or position, optionally with KNOCK or GIN.
func (g *Game) onDiscard(input string) []proto.Output {
	again := func(text string) []proto.Output {
		return []proto.Output{say(text), proto.Prompt{Text: promptDiscard[0].Text}}
	}
	words := strings.Fields(prompt.Normalize(input))
	knock := false
	if len(words) > 0 && (words[0] == "KNOCK" || words[0] == "GIN") {
		knock, words = true, words[1:]
	}
	if len(words) == 0 {
		if knock {
			return again(lineKnockHow[0].Text)
		}
		return again(lineDiscardHelp[0].Text)
	}
	if leaving(strings.Join(words, " ")) {
		return again(lineFinish[0].Text)
	}
	c, ok := g.pick(strings.Join(words, " "))
	switch {
	case !ok:
		return again(lineDiscardHelp[0].Text)
	case g.hasTook && c == g.took:
		return again(lineNotBack[0].Text)
	}
	rest := cards.Remove(g.hands[player], c)
	if knock {
		if dw := Arrange(rest).Points; dw > KnockLimit {
			return again(fill(lineTooMuch, fmt.Sprint(dw)))
		}
	}
	g.discard(player, c)
	g.order()
	outs := []proto.Output{say(fill(lineYouDiscard, c.String()))}
	if knock {
		return g.settle(outs, player)
	}
	outs = append(outs, g.woprTurn()...)
	if g.phase == between {
		return outs
	}
	return g.askDraw(outs)
}

// pick finds the card the player named: a position on the panel (1 to 11) or the card.
func (g *Game) pick(text string) (cards.Card, bool) {
	if n, ok := prompt.MenuChoice(text, len(g.hands[player])); ok {
		return g.hands[player][n-1], true
	}
	c, ok := cards.Parse(text)
	if !ok || cards.Index(g.hands[player], c) < 0 {
		return cards.Card{}, false
	}
	return c, true
}

// woprTurn draws, discards and perhaps knocks for WOPR.
func (g *Game) woprTurn() []proto.Output {
	if len(g.stock) <= stockFloor {
		return g.void(nil)
	}
	var lines []string
	if g.wantsTop() {
		lines = append(lines, fill(lineWOPRTook, g.takePile(wopr).String()))
	} else {
		g.drawStock(wopr)
		lines = append(lines, lineWOPRDrew[0].Text)
	}
	c := g.woprDiscard()
	g.discard(wopr, c)
	lines = append(lines, fill(lineWOPRDiscards, c.String()))
	g.note = strings.Join(lines, " ")
	outs := []proto.Output{say(lines...)}
	if Arrange(g.hands[wopr]).Points <= KnockLimit {
		return g.settle(outs, wopr)
	}
	return outs
}

// wantsTop reports whether WOPR takes the discard: when the card joins a meld and leaves
// less deadwood than WOPR has now.
func (g *Game) wantsTop() bool {
	top := g.top()
	with := append(slices.Clone(g.hands[wopr]), top)
	best, _ := bestDiscard(with, top, true)
	if best.Points >= Arrange(g.hands[wopr]).Points {
		return false
	}
	return !slices.Contains(best.Deadwood, top)
}

// woprDiscard is the discard that leaves the least deadwood; among equals, the highest
// card, so that WOPR sheds points.
func (g *Game) woprDiscard() cards.Card {
	_, c := bestDiscard(g.hands[wopr], g.took, g.hasTook)
	return c
}

// bestDiscard returns the arrangement left by the best discard from an eleven-card hand,
// and that discard. It never discards not (when avoid is set).
func bestDiscard(hand []cards.Card, not cards.Card, avoid bool) (Arrangement, cards.Card) {
	var best Arrangement
	var bestCard cards.Card
	found := false
	for _, c := range hand {
		if avoid && c == not {
			continue
		}
		a := Arrange(cards.Remove(hand, c))
		better := !found || a.Points < best.Points ||
			(a.Points == best.Points && (Points(c) > Points(bestCard) || (Points(c) == Points(bestCard) && cardLess(bestCard, c))))
		if better {
			best, bestCard, found = a, c, true
		}
	}
	return best, bestCard
}

func cardLess(a, b cards.Card) bool {
	if a.Rank != b.Rank {
		return a.Rank < b.Rank
	}
	return a.Suit < b.Suit
}

// settle scores a knock by side k.
func (g *Game) settle(outs []proto.Output, k int) []proto.Output {
	o := Score(g.hands[k], g.hands[1-k])
	ka := Arrange(g.hands[k])
	switch {
	case k == wopr && o.Gin:
		outs = append(outs, say(lineWOPRGin[0].Text))
	case k == wopr:
		outs = append(outs, say(fill(lineWOPRKnocks, fmt.Sprint(ka.Points))))
	case o.Gin:
		outs = append(outs, say(lineYouGin[0].Text))
	default:
		outs = append(outs, say(fill(lineYouKnock, fmt.Sprint(ka.Points))))
	}
	outs = append(outs, table(g.showLine(lineWOPRHand[0].Text, wopr), g.showLine(lineYourHand[0].Text, player)))
	if len(o.Laid) > 0 {
		who := lineYouLay
		if k == player {
			who = lineWOPRLays
		}
		outs = append(outs, say(fill(who, cards.Format(o.Laid))))
	}
	winner := k
	if !o.KnockerWins {
		winner = 1 - k
		outs = append(outs, say(lineUndercut[0].Text))
	}
	g.score[winner] += o.Points
	scores := lineYouScore
	if winner == wopr {
		scores = lineWOPRScores
	}
	outs = append(outs, say(fill(scores, fmt.Sprint(o.Points)), g.scoreLine()))
	return g.endHand(outs)
}

// void ends a hand that ran out of stock.
func (g *Game) void(outs []proto.Output) []proto.Output {
	return g.endHand(append(outs, say(lineVoid[0].Text, g.scoreLine())))
}

func (g *Game) endHand(outs []proto.Output) []proto.Output {
	g.phase = between
	outs = append(outs, proto.Redraw{})
	if g.score[player] >= GameTarget || g.score[wopr] >= GameTarget {
		return append(outs, proto.Done{Result: g.result()})
	}
	return append(outs, proto.Prompt{Text: promptNext[0].Text})
}

func (g *Game) scoreLine() string {
	return fill(lineScore, fmt.Sprint(g.score[player]), fmt.Sprint(g.score[wopr]))
}

func (g *Game) onNext(input string) []proto.Output {
	norm := prompt.Normalize(input)
	if leaving(norm) || prompt.YesNo(input) == prompt.No {
		return []proto.Output{say(fill(lineLeave, g.scoreLine())), proto.Done{Result: g.result()}}
	}
	return g.newHand(nil)
}

func leaving(norm string) bool {
	switch norm {
	case "LEAVE", "STOP", "DONE", "I'M DONE", "I QUIT", "RESIGN", "I RESIGN":
		return true
	}
	return false
}

// result compares the scores; the final scores travel with it, since the panel is gone
// when the persona speaks.
func (g *Game) result() proto.Result {
	r := proto.Result{Outcome: proto.Draw, Lines: []string{g.scoreLine()}}
	switch {
	case g.score[player] > g.score[wopr]:
		r.Outcome = proto.Win
	case g.score[player] < g.score[wopr]:
		r.Outcome = proto.Loss
	}
	return r
}

// showLine is a hand at a knock: [melds] then deadwood.
func (g *Game) showLine(label string, p int) string {
	a := Arrange(g.hands[p])
	var parts []string
	for _, m := range a.Melds {
		parts = append(parts, "["+cards.Format(m)+"]")
	}
	if len(a.Deadwood) > 0 {
		parts = append(parts, cards.Format(a.Deadwood))
	}
	return fmt.Sprintf("%-6s %s", label, strings.Join(parts, " "))
}
