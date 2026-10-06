package poker

import (
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/proto"
)

// lastPrompt is the text of the last Prompt in outs, or "" when there is none.
func lastPrompt(outs []proto.Output) (string, bool) {
	for i := len(outs) - 1; i >= 0; i-- {
		if p, ok := outs[i].(proto.Prompt); ok {
			return p.Text, true
		}
	}
	return "", false
}

func ended(outs []proto.Output) bool {
	for _, o := range outs {
		if _, ok := o.(proto.Done); ok {
			return true
		}
	}
	return false
}

// Pot accounting over seeded games played by a random player: the chips always total 200,
// nobody goes below zero, the pot is empty between hands, the betting never passes a bet
// and three raises, and both sides put in the same amount when a round closes.
func TestPotAccountingOverSeededGames(t *testing.T) {
	t.Parallel()
	inputs := map[string][]string{
		"CHECK OR BET": {"c", "b", "check", "bet", "f", "x", "bet 5", "pass", "q"},
		"OR FOLD":      {"c", "r", "f", "call", "raise", "check", "x", "call it", "i fold"},
		"DISCARD":      {"", "1", "1 2", "2 4 5", "135", "1 2 3 4", "AS", "9"},
		"ANOTHER":      {"y", "", "y", "y", "maybe"},
	}
	for seed := range uint64(150) {
		g := New().(*Game)
		outs := g.Start(proto.Env{Seed: seed, Instant: true, Deterministic: true})
		r := proto.NewRand(seed, 99)
		for step := 0; step < 400; step++ {
			you, w, pot := g.Chips()
			if you+w+pot != 2*stake || you < 0 || w < 0 || pot < 0 {
				t.Fatalf("seed %d step %d: chips %d + %d + pot %d", seed, step, you, w, pot)
			}
			if g.bets > maxBets {
				t.Fatalf("seed %d: %d bets in a round", seed, g.bets)
			}
			if ended(outs) {
				break
			}
			text, ok := lastPrompt(outs)
			if !ok {
				t.Fatalf("seed %d: nothing asks for input", seed)
			}
			var choices []string
			for key, c := range inputs {
				if strings.Contains(text, key) {
					choices = c
				}
			}
			if strings.HasPrefix(text, "DISCARD") && g.in[0] != g.in[1] {
				t.Fatalf("seed %d: the betting round closed with %d against %d", seed, g.in[0], g.in[1])
			}
			if choices == nil {
				t.Fatalf("seed %d: unknown prompt %q", seed, text)
			}
			if strings.HasPrefix(text, "ANOTHER") && pot != 0 {
				t.Fatalf("seed %d: %d chips left in the pot between hands", seed, pot)
			}
			outs = g.Handle(proto.LineEvent{Text: choices[r.IntN(len(choices))]})
		}
	}
}

func TestPlayerFoldGivesWOPRThePot(t *testing.T) {
	t.Parallel()
	g := New().(*Game)
	g.Start(proto.Env{Seed: 3, Instant: true, Deterministic: true})
	if g.phase != betting || g.toAct != player {
		t.Fatal("WOPR deals the first hand, so the player acts first")
	}
	g.pay(wopr, 5) // WOPR has bet
	g.acted[wopr], g.bets = true, 1
	pot := g.pot
	you, w, _ := g.Chips()
	g.Handle(proto.LineEvent{Text: "fold"})
	you2, w2, pot2 := g.Chips()
	if you2 != you || w2 != w+pot || pot2 != 0 || g.phase != between {
		t.Fatalf("fold: you %d→%d, wopr %d→%d, pot %d→%d", you, you2, w, w2, pot, pot2)
	}
}

func TestSplitPotOddChip(t *testing.T) {
	t.Parallel()
	g := New().(*Game)
	g.Start(proto.Env{Seed: 1, Instant: true, Deterministic: true})
	g.hands[player] = hand(t, "AS KH QD JC 9S")
	g.hands[wopr] = hand(t, "AH KD QC JS 9D")
	g.chips = [2]int{90, 99}
	g.pot = 11
	g.dealer = wopr
	g.showdown(nil)
	if g.chips != [2]int{96, 104} || g.pot != 0 {
		t.Fatalf("split: %v pot %d; the odd chip goes to the player not dealing", g.chips, g.pot)
	}
}

func TestDiscardParsing(t *testing.T) {
	t.Parallel()
	g := New().(*Game)
	g.Start(proto.Env{Seed: 1, Instant: true, Deterministic: true})
	g.hands[player] = hand(t, "AS KH QD JC 9S")
	for in, want := range map[string]string{
		"1 3":     "0 2",
		"135":     "0 2 4",
		"1,1,2":   "0 1",
		"QD 9S":   "2 4",
		"jc":      "3",
		"5 as":    "4 0",
		"6":       "bad",
		"2H":      "bad",
		"maybe 2": "bad",
	} {
		idx, ok := g.parseDiscards(in)
		got := "bad"
		if ok {
			var parts []string
			for _, i := range idx {
				parts = append(parts, string(rune('0'+i)))
			}
			got = strings.Join(parts, " ")
		}
		if got != want {
			t.Errorf("parseDiscards(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestWOPRNeverFoldsForFree(t *testing.T) {
	t.Parallel()
	for seed := range uint64(200) {
		g := New().(*Game)
		g.Start(proto.Env{Seed: seed, Instant: true, Deterministic: true})
		g.in = [2]int{}
		if g.decide() == fold {
			t.Fatalf("seed %d: WOPR folded with nothing to call", seed)
		}
	}
}

// Betting input as players type it, and the right refusal when a raise is impossible.
func TestBettingInput(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		in   string
		want string // a line the reply must contain, or "" when the action is taken
	}{
		{"bet 5", ""}, {"BET 10", "BETS ARE FIXED: THAT IS 5."}, {"x", ""}, {"pass", ""}, {"i check", ""},
	} {
		g := New().(*Game)
		g.Start(proto.Env{Seed: 3, Instant: true, Deterministic: true})
		outs := g.Handle(proto.LineEvent{Text: tc.in})
		var said string
		for _, o := range outs {
			if s, ok := o.(proto.Say); ok {
				said += strings.Join(s.Lines, "\n") + "\n"
			}
		}
		if tc.want == "" && g.acted[player] == false && g.phase == betting && g.toAct == player {
			t.Errorf("%q was not taken:\n%s", tc.in, said)
		}
		if tc.want != "" && !strings.Contains(said, tc.want) {
			t.Errorf("%q: %q, want %q", tc.in, said, tc.want)
		}
	}
	g := New().(*Game)
	g.Start(proto.Env{Seed: 3, Instant: true, Deterministic: true})
	g.chips = [2]int{0, g.chips[1] + g.chips[0]} // the player is all in
	outs := g.Handle(proto.LineEvent{Text: "raise"})
	if s, ok := outs[0].(proto.Say); !ok || !strings.Contains(s.Lines[0], "ALL IN") {
		t.Errorf("all in: %v", outs)
	}
}

func TestDiscardWords(t *testing.T) {
	t.Parallel()
	g := New().(*Game)
	g.Start(proto.Env{Seed: 1, Instant: true, Deterministic: true})
	g.hands[player] = hand(t, "AS 10C QD JC 9S")
	for in, want := range map[string]string{"10 c": "1", "ten of clubs": "1", "1 and 2": "0 1", "as and 10 c": "0 1"} {
		idx, ok := g.parseDiscards(in)
		var got []string
		for _, i := range idx {
			got = append(got, string(rune('0'+i)))
		}
		if !ok || strings.Join(got, " ") != want {
			t.Errorf("parseDiscards(%q) = %v %v, want %s", in, idx, ok, want)
		}
	}
	if _, ok := g.parseDiscards("tens"); ok {
		t.Error("TENS is not the ten of spades")
	}
}
