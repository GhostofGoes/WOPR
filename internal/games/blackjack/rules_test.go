package blackjack

import (
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games/cards"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// hand parses "AS 7H" into cards.
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

func TestValue(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		hand  string
		total int
		soft  bool
	}{
		{"AS KH", 21, true},
		{"AS 6H", 17, true},
		{"AS 6H KD", 17, false},
		{"AS AH", 12, true},
		{"AS AH 9C", 21, true},
		{"AS AH AC AD", 14, true},
		{"10S 6H", 16, false},
		{"10S 6H 9C", 25, false},
		{"JS QH", 20, false},
		{"2S 3H 4D 5C AS", 15, false},
	} {
		total, soft := Value(hand(t, tc.hand))
		if total != tc.total || soft != tc.soft {
			t.Errorf("Value(%s) = %d soft=%v; want %d soft=%v", tc.hand, total, soft, tc.total, tc.soft)
		}
	}
	if !Natural(hand(t, "AS JH")) || Natural(hand(t, "7S 7H 7D")) || Natural(hand(t, "AS 5H 5D")) {
		t.Error("Natural")
	}
}

// The dealer's fixed rules: hit below 17, stand on hard 17 and above; soft 17 depends on
// the table.
func TestDealerRules(t *testing.T) {
	t.Parallel()
	s17, h17 := DefaultRules, DefaultRules
	h17.StandSoft17 = false
	for _, tc := range []struct {
		hand     string
		s17, h17 bool // hits under each rule
	}{
		{"10S 6H", true, true},
		{"10S 7H", false, false},
		{"AS 6H", false, true}, // soft 17
		{"AS 6H 10C", false, false},
		{"AS 7H", false, false}, // soft 18
		{"AS 5H", true, true},
		{"5S 5H 5D 2C", false, false},
		{"AS AH 5D", false, true}, // soft 17 in three cards
	} {
		h := hand(t, tc.hand)
		if got := s17.DealerHits(h); got != tc.s17 {
			t.Errorf("S17 DealerHits(%s) = %v", tc.hand, got)
		}
		if got := h17.DealerHits(h); got != tc.h17 {
			t.Errorf("H17 DealerHits(%s) = %v", tc.hand, got)
		}
	}
}

// Payouts are exact, in cents: even money, 3 to 2 for a natural, pushes, and a split 21
// that is not a black jack.
func TestSettle(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		h      Hand
		dealer string
		net    int
		v      Verdict
	}{
		{"natural pays 3 to 2", Hand{Cards: hand(t, "AS KH"), Bet: 1000}, "10C 9D", 1500, Blackjack},
		{"odd bet natural", Hand{Cards: hand(t, "AS KH"), Bet: 500}, "10C 9D", 750, Blackjack},
		{"natural against natural", Hand{Cards: hand(t, "AS KH"), Bet: 1000}, "AC QD", 0, Push},
		{"dealer natural beats 21", Hand{Cards: hand(t, "7S 7H 7D"), Bet: 1000}, "AC QD", -1000, Lose},
		{"split 21 is not a natural", Hand{Cards: hand(t, "AS KH"), Bet: 1000, FromSplit: true}, "10C 9D", 1000, Win},
		{"split 21 against a three-card 21", Hand{Cards: hand(t, "AS KH"), Bet: 1000, FromSplit: true}, "10C 5D 6H", 0, Push},
		{"higher wins", Hand{Cards: hand(t, "10S 9H"), Bet: 1000}, "10C 8D", 1000, Win},
		{"lower loses", Hand{Cards: hand(t, "10S 7H"), Bet: 1000}, "10C 8D", -1000, Lose},
		{"tie pushes", Hand{Cards: hand(t, "10S 8H"), Bet: 1000}, "10C 8D", 0, Push},
		{"dealer busts", Hand{Cards: hand(t, "10S 2H"), Bet: 1000}, "10C 6D 9H", 1000, Win},
		{"player bust loses even if the dealer busts", Hand{Cards: hand(t, "10S 6H 9C"), Bet: 1000}, "10C 6D 9H", -1000, Lose},
		{"doubled win", Hand{Cards: hand(t, "5S 6H 10C"), Bet: 2000, Doubled: true}, "10C 9D", 2000, Win},
	} {
		net, v := Settle(tc.h, hand(t, tc.dealer))
		if net != tc.net || v != tc.v {
			t.Errorf("%s: net %d verdict %d; want %d %d", tc.name, net, v, tc.net, tc.v)
		}
	}
}

// Over many seeded rounds, the dealer's final hand obeys its rules: each card it drew was
// a required hit, and it stopped as soon as it had to.
func TestDealerFollowsRulesOverSeededShoes(t *testing.T) {
	t.Parallel()
	for _, standSoft := range []bool{true, false} {
		rules := DefaultRules
		rules.StandSoft17 = standSoft
		for seed := range uint64(300) {
			g := NewWithRules(rules)
			g.Start(proto.Env{Seed: seed, Instant: true, Deterministic: true})
			for range 5 {
				if g.phase != betting {
					t.Fatal("not betting")
				}
				outs := g.Handle(proto.LineEvent{Text: "1"})
				for g.phase == playing {
					outs = g.Handle(proto.LineEvent{Text: "S"})
				}
				if done(outs) {
					break
				}
				if !Natural(g.hands[0].Cards) { // a player's natural settles before the dealer plays
					checkDealer(t, rules, g.dealer)
				}
			}
		}
	}
}

func checkDealer(t *testing.T, r Rules, dealer []cards.Card) {
	t.Helper()
	if Natural(dealer) {
		return
	}
	for n := 2; n < len(dealer); n++ {
		if !r.DealerHits(dealer[:n]) {
			t.Fatalf("the dealer drew to %s (soft 17 stands: %v)", cards.Format(dealer[:n]), r.StandSoft17)
		}
	}
	if total, _ := Value(dealer); total <= 21 && r.DealerHits(dealer) {
		t.Fatalf("the dealer stood on %s", cards.Format(dealer))
	}
}

func done(outs []proto.Output) bool {
	for _, o := range outs {
		if _, ok := o.(proto.Done); ok {
			return true
		}
	}
	return false
}

// stacked starts a game whose deck deals the given cards in order (player, dealer, player,
// dealer, then hits).
func stacked(t *testing.T, deal string) *Game {
	t.Helper()
	rules := DefaultRules
	rules.Reshuffle = 0
	g := NewWithRules(rules)
	g.Start(proto.Env{Seed: 1, Instant: true, Deterministic: true})
	g.shoe = hand(t, deal)
	return g
}

// text joins every Say and Prompt in outs, one per line.
func text(outs []proto.Output) string {
	var b strings.Builder
	for _, o := range outs {
		switch o := o.(type) {
		case proto.Say:
			for _, l := range o.Lines {
				b.WriteString(l + "\n")
			}
		case proto.Prompt:
			b.WriteString("> " + o.Text + "\n")
		}
	}
	return b.String()
}

// readHands replaces each hand drawn as cards in out with a line that reads it back: its
// label, the indices of its cards (?? for one face down) and its total, as HAND 1: 8S 3C (11).
func readHands(t *testing.T, out string) string {
	t.Helper()
	lines := strings.Split(out, "\n")
	var b strings.Builder
	for i := 0; i < len(lines); i++ {
		if !strings.HasPrefix(strings.TrimSpace(lines[i]), ".-") {
			b.WriteString(lines[i] + "\n")
			continue
		}
		if i+cards.FaceH > len(lines) {
			t.Fatalf("a cut-off hand at line %d:\n%s", i, out)
		}
		var idx []string
		for _, f := range strings.Split(lines[i+1][labelW:], "|") {
			switch f = strings.TrimSpace(f); {
			case f == "":
			case strings.Trim(f, `/\`) == "":
				idx = append(idx, "??")
			default:
				idx = append(idx, f)
			}
		}
		mid := lines[i+midRow]
		label := strings.TrimSpace(mid[:labelW])
		desc := strings.TrimSpace(mid[strings.LastIndex(mid, "|")+1:])
		b.WriteString(strings.TrimSpace(label+" "+strings.Join(idx, " ")+" "+desc) + "\n")
		i += cards.FaceH - 1
	}
	return b.String()
}

func play(t *testing.T, g *Game, lines ...string) string {
	t.Helper()
	var all []proto.Output
	for _, l := range lines {
		all = append(all, g.Handle(proto.LineEvent{Text: l})...)
	}
	return text(all)
}

func TestSplitAndDouble(t *testing.T) {
	t.Parallel()
	// Player 8S 8H, dealer 6C (hole 10D). Split: hand 1 gets 3C, hand 2 gets 10S.
	// Hand 1 doubles on 11 and draws 10H (21); hand 2 stands on 18. Dealer 16 draws 9C: bust.
	g := stacked(t, "8S 6C 8H 10D 3C 10S 10H 9C")
	out := play(t, g, "10", "split", "d", "s")
	out = readHands(t, out)
	for _, want := range []string{
		"HIT, STAND, DOUBLE OR SPLIT?",
		"HAND 1: 8S 3C (11)",
		"HAND 2: 8H 10S (18)",
		"HAND 1: HIT, STAND OR DOUBLE?",
		"HAND 1: 8S 3C 10H (21)",
		"HAND 2: HIT, STAND OR DOUBLE?",
		"DEALER DRAWS 9C.",
		"THE DEALER BUSTS.",
		"HAND 1: YOU WIN $20.",
		"HAND 2: YOU WIN $10.",
		"YOU HAVE $130.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

// The longest hand one deck allows, eleven cards to 21, fans out within 80 columns and
// still shows every card; the dealer's hole card is face down until it turns.
func TestLongHandFans(t *testing.T) {
	t.Parallel()
	g := stacked(t, "AS 9C AH 8D AD AC 2S 2H 2D 2C 3S 3H 3D")
	out := play(t, g, "10", "h", "h", "h", "h", "h", "h", "h", "h", "h")
	for _, l := range strings.Split(out, "\n") {
		if len(l) > 80 {
			t.Errorf("%d columns: %q", len(l), l)
		}
	}
	read := readHands(t, out)
	for _, want := range []string{"DEALER: 9C ??", "YOU: AS AH AD AC 2S 2H 2D 2C 3S 3H 3D (21)", "DEALER: 9C 8D (17)"} {
		if !strings.Contains(read, want) {
			t.Errorf("missing %q in:\n%s", want, read)
		}
	}
	if !strings.Contains(out, ".---.---.---.---.---.---.---.---.---.---.-----.") {
		t.Errorf("eleven cards fan out:\n%s", out)
	}
}

func TestSplitAcesTakeOneCardEach(t *testing.T) {
	t.Parallel()
	g := stacked(t, "AS 9C AH 8D KS 5C")
	out := readHands(t, play(t, g, "10", "p"))
	for _, want := range []string{"HAND 1: AS KS (21)", "HAND 2: AH 5C (SOFT 16)", "THE DEALER STANDS ON 17.", "HAND 1: YOU WIN $10.", "HAND 2: YOU LOSE $10.", "YOU HAVE $100."} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestNaturals(t *testing.T) {
	t.Parallel()
	g := stacked(t, "AS 9C KH 8D")
	if out := play(t, g, "5"); !strings.Contains(out, "BLACK JACK. YOU WIN $7.50.") || !strings.Contains(out, "YOU HAVE $107.50.") {
		t.Errorf("player natural:\n%s", out)
	}
	g = stacked(t, "9S AC KH KD")
	if out := play(t, g, "5"); !strings.Contains(out, "THE DEALER HAS BLACK JACK.") || !strings.Contains(out, "YOU LOSE $5.") || g.phase != betting {
		t.Errorf("dealer natural is checked before the player plays:\n%s", out)
	}
}

func TestRefusals(t *testing.T) {
	t.Parallel()
	g := stacked(t, "9S 7C 5H 10D 4C 2S 3S")
	out := play(t, g, "0", "26", "lots", "10", "p", "leave", "x", "h", "d")
	for _, want := range []string{
		"A BET IS A WHOLE NUMBER OF DOLLARS FROM $1 TO $25.",
		"ONLY A PAIR CAN BE SPLIT, ONCE A ROUND.",
		"FINISH THE HAND FIRST.",
		"H TO HIT, S TO STAND, D TO DOUBLE DOWN, P TO SPLIT A PAIR.",
		"YOU CAN DOUBLE ONLY ON YOUR FIRST TWO CARDS.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestMoneyLimits(t *testing.T) {
	t.Parallel()
	g := stacked(t, "8S 9C 8H 10D")
	g.money = 1500 // $15
	out := play(t, g, "20", "10", "d", "p")
	if !strings.Contains(out, "YOU HAVE ONLY $15.") || strings.Count(out, "YOU DO NOT HAVE THE MONEY TO COVER IT.") != 2 {
		t.Errorf("money limits:\n%s", out)
	}
}

func TestBrokeEndsTheGame(t *testing.T) {
	t.Parallel()
	g := stacked(t, "10S 9C 6H 10D 10C")
	g.money = 500
	var outs []proto.Output
	for _, l := range []string{"5", "h"} {
		outs = append(outs, g.Handle(proto.LineEvent{Text: l})...)
	}
	if !done(outs) || !strings.Contains(text(outs), "YOU ARE OUT OF MONEY.") {
		t.Fatalf("broke:\n%s", text(outs))
	}
	if g.result().Outcome != proto.Loss {
		t.Fatal("going broke is a loss")
	}
}

func TestMoneyFormat(t *testing.T) {
	t.Parallel()
	for cents, want := range map[int]string{0: "$0", 100: "$1", 750: "$7.50", 10705: "$107.05"} {
		if got := money(cents); got != want {
			t.Errorf("money(%d) = %q", cents, got)
		}
	}
}

// Review fixes: the prompt offers only what the player can afford; a split with the deck
// running out never duplicates a card; a soft 21 is 21; a bare Enter says why it did nothing.
func TestReviewFixes(t *testing.T) {
	t.Parallel()
	g := stacked(t, "8S 9C 8H 10D")
	g.money = 1500
	if out := play(t, g, "10"); !strings.Contains(out, "> HIT OR STAND?") {
		t.Errorf("$15 cannot cover a split or a double of $10:\n%s", out)
	}

	g = stacked(t, "8S 6C 8H 10D 3C") // the deck runs out on the second hand's card
	play(t, g, "10", "p")
	seen := map[cards.Card]bool{}
	for _, c := range append(append(append([]cards.Card{}, g.dealer...), g.hands[0].Cards...), g.hands[1].Cards...) {
		if seen[c] {
			t.Fatalf("%s is on the table twice", c)
		}
		seen[c] = true
	}
	for _, c := range g.shoe {
		if seen[c] {
			t.Fatalf("%s is both on the table and in the deck", c)
		}
	}

	if got := describe(hand(t, "AS 5H 5D")); got != "21" {
		t.Errorf("soft 21 shows as %q", got)
	}

	g = stacked(t, "8S 9C 8H 10D")
	if out := play(t, g, ""); !strings.Contains(out, "A BET IS A WHOLE NUMBER") {
		t.Errorf("Enter before any bet:\n%s", out)
	}
}
