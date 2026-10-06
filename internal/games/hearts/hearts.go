// Package hearts is four-handed Hearts: the player sits South and WOPR plays West, North
// and East. Three cards are passed left, right, across, then held; the two of clubs leads;
// no points on the first trick; hearts cannot be led until broken. Each heart is a point,
// the queen of spades 13, and taking all 26 shoots the moon. The game ends when someone
// reaches 100; the lowest score wins.
package hearts

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

const you = cards.South

type phase uint8

const (
	passing phase = iota
	playing
	between
)

// Game is Hearts as a proto.Program.
type Game struct {
	rng    *rand.Rand
	phase  phase
	deal   int // deals started
	hands  [cards.Seats][]cards.Card
	trick  cards.Trick
	last   cards.Trick // the trick just taken, shown on the panel
	lastBy int
	tricks int // tricks taken this deal
	broken bool
	taken  [cards.Seats]int // points taken this deal
	score  [cards.Seats]int
}

// New returns a game.
func New() games.Game { return &Game{lastBy: -1} }

func say(lines ...string) proto.Output { return proto.Say{Lines: lines, Pace: proto.PaceSpeech} }

// Start implements proto.Program.
func (g *Game) Start(env proto.Env) []proto.Output {
	g.rng = proto.NewRand(env.Seed, 0)
	return g.newDeal([]proto.Output{say(lineRules.Texts()...)})
}

// Handle implements proto.Program.
func (g *Game) Handle(ev proto.Event) []proto.Output {
	line, ok := ev.(proto.LineEvent)
	if !ok {
		return nil
	}
	switch g.phase {
	case passing:
		return g.onPass(line.Text)
	case playing:
		return g.onPlay(line.Text)
	default:
		return g.onNext(line.Text)
	}
}

func (g *Game) newDeal(outs []proto.Output) []proto.Output {
	deck := cards.Shuffled(g.rng)
	for s := range g.hands {
		g.hands[s] = slices.Clone(deck[13*s : 13*s+13])
		cards.Sort(g.hands[s])
	}
	g.trick, g.last, g.lastBy = cards.Trick{}, cards.Trick{}, -1
	g.tricks, g.broken, g.taken = 0, false, [cards.Seats]int{}
	dir := Pass(g.deal)
	g.deal++
	if dir == 0 {
		outs = append(outs, say(lineHold[0].Text))
		return g.startPlay(outs)
	}
	g.phase = passing
	return append(outs, proto.Redraw{}, proto.Prompt{Text: fill(promptPass, cards.SeatNames[(you+dir)%cards.Seats])})
}

// pick finds the card the player named: a position on the panel or the card itself.
func (g *Game) pick(text string) (cards.Card, bool) {
	if n, ok := prompt.MenuChoice(text, len(g.hands[you])); ok {
		return g.hands[you][n-1], true
	}
	c, ok := cards.Parse(text)
	return c, ok && cards.Index(g.hands[you], c) >= 0
}

// onPass reads three cards: positions (1 5 9) or names (QS AH KH).
func (g *Game) onPass(input string) []proto.Output {
	again := func(text string) []proto.Output {
		return []proto.Output{say(text), proto.Prompt{Text: fill(promptPass, cards.SeatNames[(you+Pass(g.deal-1))%cards.Seats])}}
	}
	if leaving(prompt.Normalize(input)) {
		return again(lineFinish[0].Text)
	}
	var chosen []cards.Card
	for _, f := range splitCards(input) {
		c, ok := g.pick(f)
		if !ok {
			return again(linePassHelp[0].Text)
		}
		if !slices.Contains(chosen, c) {
			chosen = append(chosen, c)
		}
	}
	if len(chosen) != 3 {
		return again(linePassHelp[0].Text)
	}
	dir := Pass(g.deal - 1)
	passes := [cards.Seats][]cards.Card{you: chosen}
	for s := cards.West; s < cards.Seats; s++ {
		passes[s] = choosePass(g.hands[s])
	}
	for s, cs := range passes {
		for _, c := range cs {
			g.hands[s] = cards.Remove(g.hands[s], c)
		}
	}
	for s, cs := range passes {
		to := (s + dir) % cards.Seats
		g.hands[to] = append(g.hands[to], cs...)
		cards.Sort(g.hands[to])
	}
	from := passes[(you-dir+cards.Seats)%cards.Seats]
	return g.startPlay([]proto.Output{say(fill(lineReceived, cards.Format(from)))})
}

// splitCards splits input into card or position tokens: "1 5 9", "1,5,9", "QS AH 10H".
func splitCards(input string) []string {
	return strings.Fields(strings.NewReplacer(",", " ", ";", " ").Replace(input))
}

func (g *Game) startPlay(outs []proto.Output) []proto.Output {
	g.phase = playing
	for s, h := range g.hands {
		if cards.Index(h, twoClubs) >= 0 {
			g.trick = cards.Trick{Leader: s}
		}
	}
	return g.run(outs)
}

// run plays WOPR's seats until it is the player's turn or the deal ends.
func (g *Game) run(outs []proto.Output) []proto.Output {
	for {
		if g.trick.Done() {
			outs = append(outs, g.takeTrick()...)
			if g.tricks == 13 {
				return g.endDeal(outs)
			}
		}
		seat := g.trick.Next()
		if seat == you {
			return append(outs, proto.Redraw{}, proto.Prompt{Text: promptPlay[0].Text})
		}
		legal := Legal(g.hands[seat], g.trick, g.tricks == 0, g.broken)
		g.play(seat, choosePlay(g.hands[seat], g.trick, legal))
	}
}

func (g *Game) play(seat int, c cards.Card) {
	g.hands[seat] = cards.Remove(g.hands[seat], c)
	g.trick.Cards = append(g.trick.Cards, c)
	if c.Suit == cards.Hearts {
		g.broken = true
	}
}

// takeTrick gives the trick to its winner, who leads the next.
func (g *Game) takeTrick() []proto.Output {
	w := g.trick.Winner(cards.NoTrump)
	pts := pointsOf(g.trick.Cards)
	g.taken[w] += pts
	g.tricks++
	g.last, g.lastBy = g.trick, w
	text := trickText(g.trick, w)
	if pts > 0 {
		text += " " + fill(linePoints, fmt.Sprint(pts))
	}
	g.trick = cards.Trick{Leader: w}
	return []proto.Output{proto.Say{Lines: []string{text}, Pace: proto.PaceTable}}
}

// trickText is a taken trick: each seat's card in play order, and who took it.
func trickText(t cards.Trick, winner int) string {
	var parts []string
	for i, c := range t.Cards {
		parts = append(parts, seatName(t.Seat(i))+" "+c.String())
	}
	if winner == you {
		return fill(lineYouTake, strings.Join(parts, ", "))
	}
	return fill(lineTrick, strings.Join(parts, ", "), seatName(winner))
}

func seatName(s int) string {
	if s == you {
		return lineYou[0].Text
	}
	return cards.SeatNames[s]
}

func (g *Game) onPlay(input string) []proto.Output {
	again := func(text string) []proto.Output {
		return []proto.Output{say(text), proto.Prompt{Text: promptPlay[0].Text}}
	}
	norm := prompt.Normalize(input)
	if leaving(norm) {
		return again(lineFinish[0].Text)
	}
	c, ok := g.pick(input)
	if !ok {
		if _, isCard := cards.Parse(input); isCard {
			return again(lineNotHeld[0].Text)
		}
		return again(linePlayHelp[0].Text)
	}
	switch Check(g.hands[you], g.trick, c, g.tricks == 0, g.broken) {
	case MustFollow:
		led, _ := g.trick.Led()
		return again(fill(lineFollow, led.String()))
	case MustLeadTwo:
		return again(lineLeadTwo[0].Text)
	case NotBroken:
		return again(lineNotBroken[0].Text)
	case NoPointsFirst:
		return again(lineNoPoints[0].Text)
	}
	g.play(you, c)
	return g.run(nil)
}

func (g *Game) endDeal(outs []proto.Output) []proto.Output {
	add, moon := Settle(g.taken)
	if moon >= 0 {
		who := fill(lineMoon, seatName(moon))
		if moon == you {
			who = lineYouMoon[0].Text
		}
		outs = append(outs, say(who))
	}
	for s := range g.score {
		g.score[s] += add[s]
	}
	outs = append(outs, say(fill(lineHandPoints, g.scoreText(g.taken)), fill(lineScores, g.scoreText(g.score))), proto.Redraw{})
	g.phase = between
	if slices.Max(g.score[:]) >= GameOver {
		return append(outs, proto.Done{Result: g.result()})
	}
	return append(outs, proto.Prompt{Text: promptNext[0].Text})
}

func (g *Game) scoreText(n [cards.Seats]int) string {
	var parts []string
	for s, v := range n {
		parts = append(parts, fmt.Sprintf("%s %d", seatName(s), v))
	}
	return strings.Join(parts, ", ")
}

func (g *Game) onNext(input string) []proto.Output {
	if leaving(prompt.Normalize(input)) || prompt.YesNo(input) == prompt.No {
		return []proto.Output{say(fill(lineScores, g.scoreText(g.score))), proto.Done{Result: g.result()}}
	}
	return g.newDeal(nil)
}

func leaving(norm string) bool {
	switch norm {
	case "LEAVE", "STOP", "DONE", "I'M DONE", "I QUIT", "RESIGN", "I RESIGN":
		return true
	}
	return false
}

// result: the lowest score wins; sharing the lowest is a draw. The final scores travel
// with it, since the panel is gone when the persona speaks.
func (g *Game) result() proto.Result {
	low := slices.Min(g.score[:])
	r := proto.Result{Outcome: proto.Loss, Lines: []string{fill(lineScores, g.scoreText(g.score))}}
	if g.score[you] == low {
		r.Outcome = proto.Win
		for s, v := range g.score {
			if s != you && v == low {
				r.Outcome = proto.Draw
			}
		}
	}
	return r
}
