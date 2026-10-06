// Package poker is heads-up five-card draw against WOPR: an ante of 1, fixed-limit betting
// of 5 before the draw and 10 after (a bet and three raises a round), and up to three cards
// drawn. Each side starts with 100 chips; the game ends when the player leaves between
// hands or either side cannot pay the ante.
package poker

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/cards"
	"github.com/GhostofGoes/WOPR/internal/prompt"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// Table stakes.
const (
	stake       = 100
	ante        = 1
	maxBets     = 4 // a bet and three raises
	maxDiscards = 3
	woprPause   = 600 * time.Millisecond
)

// betSize is the fixed bet before (round 0) and after (round 1) the draw.
var betSize = [2]int{5, 10}

// Seats.
const (
	player = 0
	wopr   = 1
)

type phase uint8

const (
	betting phase = iota
	drawing
	between // waiting for "another hand?"
)

// Game is poker as a proto.Program.
type Game struct {
	deal *rand.Rand // the cards
	ai   *rand.Rand // WOPR's judgement and bluffs

	phase  phase
	deck   []cards.Card
	hands  [2][]cards.Card
	chips  [2]int
	pot    int
	dealer int
	played int // hands started

	round int    // 0 before the draw, 1 after
	in    [2]int // chips each side put in this betting round
	acted [2]bool
	bets  int // bets and raises this round
	toAct int
	drew  [2]int // cards each side drew; -1 before it has drawn
	bluff bool   // WOPR bluffs this hand
}

// New returns a game.
func New() games.Game { return &Game{} }

func say(lines ...string) proto.Output { return proto.Say{Lines: lines, Pace: proto.PaceSpeech} }

func table(lines ...string) proto.Output { return proto.Say{Lines: lines, Pace: proto.PaceTable} }

// Start implements proto.Program.
func (g *Game) Start(env proto.Env) []proto.Output {
	g.deal = proto.NewRand(env.Seed, 0)
	g.ai = proto.NewRand(env.Seed, 1)
	g.chips = [2]int{stake, stake}
	g.dealer = player // the first hand is WOPR's deal
	outs := []proto.Output{table(artTitle.Texts()...), say(lineRules.Texts()...)}
	return g.newHand(outs)
}

// View implements proto.Program; poker is console text only.
func (g *Game) View(*proto.Canvas) {}

// Handle implements proto.Program.
func (g *Game) Handle(ev proto.Event) []proto.Output {
	line, ok := ev.(proto.LineEvent)
	if !ok {
		return nil
	}
	switch g.phase {
	case betting:
		return g.onAct(line.Text)
	case drawing:
		return g.onDraw(line.Text)
	default:
		return g.onAgain(line.Text)
	}
}

// Chips returns each side's chips and the pot, for tests: they always total 200.
func (g *Game) Chips() (you, w, pot int) { return g.chips[player], g.chips[wopr], g.pot }

func (g *Game) newHand(outs []proto.Output) []proto.Output {
	g.dealer = 1 - g.dealer
	g.played++
	g.deck = cards.Shuffled(g.deal)
	g.hands = [2][]cards.Card{}
	for range 5 {
		for _, p := range []int{1 - g.dealer, g.dealer} {
			g.hands[p] = append(g.hands[p], g.deck[0])
			g.deck = g.deck[1:]
		}
	}
	for p := range g.chips {
		g.chips[p] -= ante
		g.pot += ante
	}
	g.bluff = g.ai.Float64() < bluffChance
	g.drew = [2]int{-1, -1}
	deals := lineWOPRDeals
	if g.dealer == player {
		deals = lineYouDeal
	}
	outs = append(outs, say(deals[0].Text), table(g.handRows(lineYourHand, player, true)...))
	g.startRound(0)
	return g.next(outs)
}

func (g *Game) startRound(r int) {
	g.phase, g.round = betting, r
	g.in, g.acted, g.bets = [2]int{}, [2]bool{}, 0
	g.toAct = 1 - g.dealer
	if g.chips[player] == 0 || g.chips[wopr] == 0 { // someone is all in: no more betting
		g.acted = [2]bool{true, true}
	}
}

func (g *Game) roundDone() bool {
	return g.acted[player] && g.acted[wopr] && g.in[player] == g.in[wopr]
}

// raiseAmount is what p may bet or raise by: the round's bet size, limited by p's chips
// after calling and by the chips the other side has left to answer it.
func (g *Game) raiseAmount(p int) int {
	if g.bets >= maxBets {
		return 0
	}
	toCall := g.in[1-p] - g.in[p]
	return max(min(betSize[g.round], g.chips[p]-toCall, g.chips[1-p]), 0)
}

func (g *Game) pay(p, n int) {
	g.chips[p] -= n
	g.pot += n
	g.in[p] += n
}

// next runs the hand until the player has to answer: WOPR's turns, the draw, the showdown.
func (g *Game) next(outs []proto.Output) []proto.Output {
	for {
		switch g.phase {
		case betting:
			switch {
			case g.roundDone() && g.round == 0:
				g.phase = drawing
			case g.roundDone():
				return g.showdown(outs)
			case g.toAct == wopr:
				outs = append(outs, g.woprAct()...)
				if g.phase == between {
					return outs
				}
			default:
				return append(outs, g.askAct())
			}
		case drawing: // the player not dealing draws first
			side := -1
			for _, p := range [2]int{1 - g.dealer, g.dealer} {
				if g.drew[p] < 0 {
					side = p
					break
				}
			}
			switch side {
			case wopr:
				outs = append(outs, g.woprDraw()...)
			case player:
				return append(outs, proto.Prompt{Text: promptDraw[0].Text})
			default:
				g.startRound(1)
			}
		case between:
			return outs
		}
	}
}

func (g *Game) askAct() proto.Output {
	toCall := g.in[wopr] - g.in[player]
	pot := fmt.Sprint(g.pot)
	switch {
	case toCall == 0:
		return proto.Prompt{Text: fill(promptCheck, pot)}
	case g.raiseAmount(player) > 0:
		return proto.Prompt{Text: fill(promptCall, pot, fmt.Sprint(toCall))}
	}
	return proto.Prompt{Text: fill(promptCallOnly, pot, fmt.Sprint(toCall))}
}

// apply plays action a for side p. It reports the chips that moved.
func (g *Game) apply(p int, a action) int {
	toCall := g.in[1-p] - g.in[p]
	g.acted[p] = true
	g.toAct = 1 - p
	switch a {
	case call:
		g.pay(p, toCall)
		return toCall
	case raise:
		n := g.raiseAmount(p)
		g.pay(p, toCall+n)
		g.bets++
		g.acted[1-p] = false
		return n
	}
	return 0
}

func (g *Game) onAct(input string) []proto.Output {
	toCall := g.in[wopr] - g.in[player]
	again := func(text string) []proto.Output { return []proto.Output{say(text), g.askAct()} }
	words := strings.Fields(prompt.Normalize(input))
	if len(words) > 1 && words[0] == "I" {
		words = words[1:] // I CALL, I FOLD
	}
	if len(words) == 0 {
		return again(lineActions[0].Text)
	}
	// A number after the verb must be the fixed amount: BET 5, CALL 10.
	amount := func(want int) bool {
		if len(words) < 2 || words[1] == "IT" {
			return true
		}
		n, ok := prompt.Number(words[1])
		return ok && n == want
	}
	switch words[0] {
	case "C", "CALL", "SEE":
		if !amount(toCall) {
			return again(fill(lineFixed, fmt.Sprint(toCall)))
		}
		g.apply(player, call)
	case "K", "X", "CHECK", "PASS":
		if toCall > 0 {
			return again(fill(lineNoCheck, fmt.Sprint(toCall)))
		}
		g.apply(player, call)
	case "B", "R", "BET", "RAISE", "RERAISE":
		n := g.raiseAmount(player)
		switch {
		case n == 0 && g.bets >= maxBets:
			return again(lineCapped[0].Text)
		case n == 0:
			return again(lineAllIn[0].Text)
		case !amount(n):
			return again(fill(lineFixed, fmt.Sprint(n)))
		}
		g.apply(player, raise)
	case "F", "FOLD":
		return g.award([]proto.Output{say(lineYouFold[0].Text)}, wopr)
	default:
		if leavingWord(strings.Join(words, " ")) {
			return again(lineFinish[0].Text)
		}
		return again(lineActions[0].Text)
	}
	return g.next(nil)
}

func (g *Game) woprAct() []proto.Output {
	a := g.decide()
	toCall := g.in[player] - g.in[wopr]
	var text string
	switch a {
	case fold:
		return g.award([]proto.Output{proto.Wait{D: woprPause}, say(lineWOPRFolds[0].Text)}, player)
	case check:
		text = lineWOPRChecks[0].Text
	case call:
		text = fill(lineWOPRCalls, fmt.Sprint(toCall))
	case raise:
		n := g.apply(wopr, raise)
		line := lineWOPRRaises
		if toCall == 0 {
			line = lineWOPRBets
		}
		return []proto.Output{proto.Wait{D: woprPause}, say(fill(line, fmt.Sprint(n)))}
	}
	g.apply(wopr, a)
	return []proto.Output{proto.Wait{D: woprPause}, say(text)}
}

// onDraw reads the player's discards: positions 1 to 5 or the cards themselves.
func (g *Game) onDraw(input string) []proto.Output {
	again := func(text string) []proto.Output {
		return []proto.Output{say(text), proto.Prompt{Text: promptDraw[0].Text}}
	}
	norm := prompt.Normalize(input)
	var out []int
	switch norm {
	case "", "NONE", "0", "PAT", "STAND PAT", "KEEP", "KEEP ALL", "NO", "NOTHING":
	default:
		if leavingWord(norm) {
			return again(lineFinish[0].Text)
		}
		var ok bool
		if out, ok = g.parseDiscards(input); !ok {
			return again(lineWhichCards[0].Text)
		}
		if len(out) > maxDiscards {
			return again(lineTooMany[0].Text)
		}
	}
	g.replace(player, out)
	outs := []proto.Output{say(lineYouPat[0].Text)}
	if len(out) > 0 {
		outs = []proto.Output{say(fill(lineYouDraw, fmt.Sprint(len(out)))), table(g.handRows(lineYourHand, player, false)...)}
	}
	return g.next(outs)
}

// parseDiscards reads positions (1 3 5, 135) or cards in the hand (7H KD).
func (g *Game) parseDiscards(input string) ([]int, bool) {
	fields := strings.Fields(strings.NewReplacer(",", " ", ";", " ").Replace(input))
	if len(fields) == 1 && len(fields[0]) > 1 && strings.Trim(fields[0], "12345") == "" { // 135
		fields = strings.Split(fields[0], "")
	}
	seen := map[int]bool{}
	var out []int
	for i := 0; i < len(fields); {
		if strings.EqualFold(fields[i], "and") {
			i++
			continue
		}
		at, used := -1, 1
		if n, ok := prompt.MenuChoice(fields[i], 5); ok {
			at = n - 1
		} else {
			for _, k := range []int{3, 2, 1} { // TEN OF CLUBS, 10 C, 10C
				if i+k > len(fields) {
					continue
				}
				if c, ok := cards.Parse(strings.Join(fields[i:i+k], " ")); ok {
					at, used = cards.Index(g.hands[player], c), k
					break
				}
			}
		}
		if at < 0 {
			return nil, false
		}
		if !seen[at] {
			seen[at] = true
			out = append(out, at)
		}
		i += used
	}
	return out, true
}

// replace swaps the cards at idx for new ones from the deck.
func (g *Game) replace(p int, idx []int) {
	for _, i := range idx {
		g.hands[p][i] = g.deck[0]
		g.deck = g.deck[1:]
	}
	g.drew[p] = len(idx)
}

func (g *Game) woprDraw() []proto.Output {
	idx := discards(g.hands[wopr])
	if g.bluff && Evaluate(g.hands[wopr]).Category() < Straight && g.ai.Float64() < snowChance {
		idx = nil // stand pat on nothing
	}
	g.replace(wopr, idx)
	if len(idx) == 0 {
		return []proto.Output{proto.Wait{D: woprPause}, say(lineWOPRPat[0].Text)}
	}
	return []proto.Output{proto.Wait{D: woprPause}, say(fill(lineWOPRDraws, fmt.Sprint(len(idx))))}
}

// showdown compares the hands; ties split the pot, the odd chip to the player not dealing.
func (g *Game) showdown(outs []proto.Output) []proto.Output {
	ws, ps := Evaluate(g.hands[wopr]), Evaluate(g.hands[player])
	outs = append(outs,
		table(g.handRows(lineWOPRShows, wopr, false)...),
		table(g.handRows(lineYouHad, player, false)...),
	)
	switch {
	case ps > ws:
		return g.award(outs, player)
	case ws > ps:
		return g.award(outs, wopr)
	}
	half := g.pot / 2
	g.chips[1-g.dealer] += g.pot - half
	g.chips[g.dealer] += half
	g.pot = 0
	return g.endHand(append(outs, say(lineSplit[0].Text)))
}

func (g *Game) award(outs []proto.Output, p int) []proto.Output {
	line := lineYouWin
	if p == wopr {
		line = lineWOPRWins
	}
	outs = append(outs, say(fill(line, fmt.Sprint(g.pot))))
	g.chips[p] += g.pot
	g.pot = 0
	return g.endHand(outs)
}

func (g *Game) endHand(outs []proto.Output) []proto.Output {
	g.phase = between
	outs = append(outs, say(fill(lineChips, fmt.Sprint(g.chips[player]), fmt.Sprint(g.chips[wopr])), ""))
	switch {
	case g.chips[player] < ante:
		return append(outs, say(lineBroke[0].Text), proto.Done{Result: g.result()})
	case g.chips[wopr] < ante:
		return append(outs, say(lineWOPRBroke[0].Text), proto.Done{Result: g.result()})
	}
	return append(outs, proto.Prompt{Text: promptAgain[0].Text})
}

// leavingWord reports a request to cash out. QUIT, EXIT and LOGOFF are the host's: they end
// the session from anywhere.
func leavingWord(norm string) bool {
	switch norm {
	case "LEAVE", "L", "DONE", "STOP", "CASH OUT", "CASH IN", "I QUIT", "I'M DONE":
		return true
	}
	return false
}

func (g *Game) onAgain(input string) []proto.Output {
	norm := prompt.Normalize(input)
	switch {
	case norm == "" || norm == "DEAL" || prompt.YesNo(input) == prompt.Yes:
		return g.newHand(nil)
	case leavingWord(norm) || prompt.YesNo(input) == prompt.No:
		return []proto.Output{say(fill(lineLeave, fmt.Sprint(g.chips[player]))), proto.Done{Result: g.result()}}
	}
	return []proto.Output{say(lineAgainHelp[0].Text), proto.Prompt{Text: promptAgain[0].Text}}
}

func (g *Game) result() proto.Result {
	switch {
	case g.chips[player] > stake:
		return proto.Result{Outcome: proto.Win}
	case g.chips[player] < stake:
		return proto.Result{Outcome: proto.Loss}
	}
	return proto.Result{Outcome: proto.Draw}
}
