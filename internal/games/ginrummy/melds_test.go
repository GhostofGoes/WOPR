package ginrummy

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

func melds(a Arrangement) string {
	var parts []string
	for _, m := range a.Melds {
		parts = append(parts, "["+cards.Format(m)+"]")
	}
	return strings.Join(parts, " ")
}

func TestArrange(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		hand     string
		points   int
		deadwood string
	}{
		{"AS 2S 3S 7D 7C 7H KH QD 9C 4C", 33, "4C 9C QD KH"},
		{"AS 2S 3S 4S 5S 6S 7S 8S 9S 10S", 0, ""},
		{"QS KS AS 2D 3H 4C 5D 6H 8C 9D", 58, "AS 2D 3H 4C 5D 6H 8C 9D QS KS"}, // Q-K-A is no run
		// 7S fits both the set and the run; the run 5S-8S leaves 7H 7D (14), the set leaves
		// 5S 6S 8S (19): the run is better.
		{"5S 6S 7S 8S 7H 7D KC KD KH 2C", 16, "2C 7D 7H"},
		// Four of a kind lends one card to a run.
		{"9S 9H 9D 9C 10S JS 2C 3D 4H 5S", 14, "2C 3D 4H 5S"},
		{"AS AH AD 2S 3S 2H 3H 2D 3D 4C", 4, "4C"}, // three runs beat a set of aces
	} {
		a := Arrange(hand(t, tc.hand))
		if a.Points != tc.points || cards.Format(a.Deadwood) != tc.deadwood {
			t.Errorf("%s: %s deadwood %q (%d), want %q (%d)", tc.hand, melds(a), cards.Format(a.Deadwood), a.Points, tc.deadwood, tc.points)
		}
	}
}

func TestLayOff(t *testing.T) {
	t.Parallel()
	knocker := [][]cards.Card{hand(t, "5H 6H 7H"), hand(t, "KS KD KC")}
	laid, left := LayOff(knocker, hand(t, "4H 3H 8H KH 9C 2C"))
	if cards.Format(laid) != "4H 3H 8H KH" || cards.Format(left) != "9C 2C" {
		t.Fatalf("laid %s, left %s", cards.Format(laid), cards.Format(left))
	}
	_, left = LayOff([][]cards.Card{hand(t, "KS KD KC KH")}, hand(t, "QS"))
	if len(left) != 1 {
		t.Fatal("a set of four takes nothing more")
	}
}

// Knock and gin scoring.
func TestScore(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name              string
		knocker, defender string
		want              Outcome
	}{
		{
			"knock wins the difference",
			"AS 2S 3S 7D 7C 7H 9C 9D 9H 4C", "KH QD JC 10S 5D 5C 2H 8S 6C 6S",
			Outcome{KnockerWins: true, Points: 72 - 4},
		},
		{
			"gin: 25 plus all the defender's deadwood, no lay-offs",
			"AS 2S 3S 7D 7C 7H 9C 9D 9H 9S", "4S 8D 8H 8S KH QD JC 10C 5D 5C",
			Outcome{KnockerWins: true, Points: 25 + 4 + 10 + 10 + 10 + 10 + 5 + 5, Gin: true}, // the 4S would lay off; gin allows none
		},
		{
			"undercut: the defender scores the difference and 25",
			"AS 2S 3S 7D 7C 7H 9C 9D 9H 8C", "4D 4H 4C JS QS KS 5S 6S 7S 2C",
			Outcome{Points: 8 - 2 + 25, Undercut: true},
		},
		{
			"lay-offs turn a knock into an undercut",
			"AS 2S 3S 7D 7C 7H JC QC KC 5H", "4S 7S 10C 2D 2H 2C 9D 9S 9C 5D",
			Outcome{Points: 0 + UndercutBonus, Undercut: true}, // 5 deadwood against 5
		},
	} {
		got := Score(hand(t, tc.knocker), hand(t, tc.defender))
		if got.KnockerWins != tc.want.KnockerWins || got.Points != tc.want.Points || got.Gin != tc.want.Gin || got.Undercut != tc.want.Undercut {
			t.Errorf("%s: %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

// Over seeded games against a simple player, WOPR's every move is legal: ten cards after
// its turn, never the card it just took back onto the pile, a knock only with 10 or less
// deadwood, and all 52 cards always accounted for.
func TestWOPRPlaysLegally(t *testing.T) {
	t.Parallel()
	for seed := range uint64(60) {
		g := New().(*Game)
		g.Start(proto.Env{Seed: seed, Instant: true, Deterministic: true})
		r := proto.NewRand(seed, 7)
		for step := 0; step < 400; step++ {
			check(t, g, seed)
			var input string
			switch g.phase {
			case drawing:
				input = []string{"s", "d"}[r.IntN(2)]
			case discarding:
				// The last card shown is deadwood (or the highest meld card); knock when allowed.
				h := g.hands[player]
				c := h[len(h)-1]
				if g.hasTook && c == g.took {
					c = h[0]
				}
				input = c.String()
				if Arrange(cards.Remove(h, c)).Points <= KnockLimit {
					input = "knock " + input
				}
			default:
				if g.score[player] >= GameTarget || g.score[wopr] >= GameTarget {
					step = 400
					continue
				}
				input = "y"
			}
			g.Handle(proto.LineEvent{Text: input})
			if g.phase != between && len(g.hands[wopr]) != 10 {
				t.Fatalf("seed %d: WOPR holds %d cards after its turn", seed, len(g.hands[wopr]))
			}
		}
	}
}

func check(t *testing.T, g *Game, seed uint64) {
	t.Helper()
	seen := map[cards.Card]bool{}
	for _, group := range [][]cards.Card{g.stock, g.pile, g.hands[player], g.hands[wopr]} {
		for _, c := range group {
			if seen[c] {
				t.Fatalf("seed %d: %s is in two places", seed, c)
			}
			seen[c] = true
		}
	}
	if len(seen) != 52 {
		t.Fatalf("seed %d: %d cards on the table", seed, len(seen))
	}
}

func TestWOPRNeverDiscardsWhatItTook(t *testing.T) {
	t.Parallel()
	g := New().(*Game)
	g.Start(proto.Env{Seed: 1, Instant: true, Deterministic: true})
	// WOPR holds two of a run and the pile offers the third; its worst card is the one it takes.
	g.hands[wopr] = hand(t, "2S 3S KH KD QC QD JH JD 10C 9H")
	g.pile = hand(t, "4S")
	if !g.wantsTop() {
		t.Fatal("WOPR should take the 4S to finish the run")
	}
	g.takePile(wopr)
	if c := g.woprDiscard(); c == g.took {
		t.Fatalf("WOPR discarded the card it took (%s)", c)
	}
}

// Review cases: the best lay-off is not the greedy one, and the defender may break a meld
// to lay off more.
func TestLayOffChoices(t *testing.T) {
	t.Parallel()
	// 8H fits the eights and the hearts run; on the run, 9H follows: an undercut.
	got := Score(hand(t, "8C 8D 8S 5H 6H 7H 2C 3C 4C AD"), hand(t, "KS KH KD 2S 3S 4S 5S 6S 8H 9H"))
	if got.KnockerWins || !got.Undercut || got.Points != 1+UndercutBonus {
		t.Errorf("greedy lay-off: %+v", got)
	}
	// Meld the three sevens rather than 7H-9H, then lay 8H and 9H on 10H-QH: 3 left.
	got = Score(hand(t, "10H JH QH 2S 3S 4S 5C 6C 7C AC"), hand(t, "7H 7S 7D 8H 9H JS QS KS AD 2D"))
	if !got.KnockerWins || got.Points != 3-1 {
		t.Errorf("breaking a meld: %+v", got)
	}
}

// Discard wording: KNOCK anywhere, filler words, and GIN only with no deadwood.
func TestDiscardWording(t *testing.T) {
	t.Parallel()
	setup := func() *Game {
		g := New().(*Game)
		g.Start(proto.Env{Seed: 3, Instant: true, Deterministic: true})
		// 2C-4C, 7s, 9H-JH melded; 2S and 3D deadwood (5) and KS to throw.
		g.hands[player] = hand(t, "2C 3C 4C 7S 7H 7D 9H 10H JH 2S KS")
		g.phase, g.hasTook = discarding, false
		return g
	}
	said := func(outs []proto.Output) string {
		var b strings.Builder
		for _, o := range outs {
			if s, ok := o.(proto.Say); ok {
				b.WriteString(strings.Join(s.Lines, "\n") + "\n")
			}
		}
		return b.String()
	}
	for _, in := range []string{"KS knock", "knock with the KS", "Knock KS", "discard KS and knock"} {
		if out := said(setup().Handle(proto.LineEvent{Text: in})); !strings.Contains(out, "YOU KNOCK WITH 2 DEADWOOD.") {
			t.Errorf("%q: %q", in, out)
		}
	}
	if out := said(setup().Handle(proto.LineEvent{Text: "gin KS"})); !strings.Contains(out, "GIN NEEDS NONE") {
		t.Errorf("gin with deadwood: %q", out)
	}
	if out := said(setup().Handle(proto.LineEvent{Text: "throw away the KS"})); !strings.Contains(out, "YOU DISCARD THE KS.") {
		t.Errorf("filler words: %q", out)
	}
}
