// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package ginrummy

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games/cards"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

func saidText(outs []proto.Output) string {
	var b strings.Builder
	for _, o := range outs {
		if s, ok := o.(proto.Say); ok {
			b.WriteString(strings.Join(s.Lines, "\n") + "\n")
		}
	}
	return b.String()
}

// startDrawing starts a game and returns it waiting for the player's draw.
func startDrawing(t *testing.T, seed uint64) *Game {
	t.Helper()
	g := New().(*Game)
	g.Start(proto.Env{Seed: seed, Deterministic: true, Instant: true})
	if g.phase != drawing {
		t.Fatalf("seed %d: the game starts in phase %d, not drawing", seed, g.phase)
	}
	return g
}

// The draw prompt takes the stock or the discard pile in many words, refuses a card that is
// not on top of the pile, and explains itself for anything else.
func TestDrawInput(t *testing.T) {
	t.Parallel()
	if g := startDrawing(t, 1); g.Handle(proto.KeyEvent{Key: proto.KeyEnter}) != nil {
		t.Error("a key event was answered")
	}
	for in, want := range map[string]string{
		"HELLO":  lineDrawHelp[0].Text,
		"i quit": lineFinish[0].Text,
	} {
		g := startDrawing(t, 1)
		if got := saidText(g.Handle(proto.LineEvent{Text: in})); !strings.Contains(got, want) || g.phase != drawing {
			t.Errorf("%q: %q, phase %d", in, got, g.phase)
		}
	}
	g := startDrawing(t, 1)
	notTop := g.hands[player][0]
	if got := saidText(g.Handle(proto.LineEvent{Text: "take the " + notTop.String()})); !strings.Contains(got, fill(lineNotTop, g.top().String())) {
		t.Errorf("a card not on the pile: %q", got)
	}
	for _, in := range []string{"S", "stock", "deck", "draw", "the stock", "from the stock"} {
		g := startDrawing(t, 1)
		top := g.stock[0]
		if got := saidText(g.Handle(proto.LineEvent{Text: in})); !strings.Contains(got, top.String()) || g.phase != discarding {
			t.Errorf("%q drew %q, phase %d", in, got, g.phase)
		}
	}
	for _, in := range []string{"D", "discard", "pile", "take", "take it", "the discard", "up card", "TOP"} {
		g := startDrawing(t, 1)
		top := g.top()
		if in == "TOP" {
			in = "take the " + top.String()
		}
		g.Handle(proto.LineEvent{Text: in})
		if g.phase != discarding || !g.hasTook || g.took != top {
			t.Errorf("%q: phase %d, took %v (%v), want %v", in, g.phase, g.took, g.hasTook, top)
		}
	}
}

// The discard prompt reads a card or a position, with KNOCK or GIN anywhere, and refuses
// what the rules forbid.
func TestDiscardInput(t *testing.T) {
	t.Parallel()
	waitDiscard := func(t *testing.T, take bool) *Game {
		t.Helper()
		g := startDrawing(t, 3)
		in := "stock"
		if take {
			in = "pile"
		}
		g.Handle(proto.LineEvent{Text: in})
		return g
	}
	for in, want := range map[string]string{
		"":           lineDiscardHelp[0].Text,
		"throw away": lineDiscardHelp[0].Text,
		"KNOCK":      lineKnockHow[0].Text,
		"resign":     lineFinish[0].Text,
		"ZZ":         lineDiscardHelp[0].Text,
		"99":         lineDiscardHelp[0].Text,
	} {
		g := waitDiscard(t, false)
		if got := saidText(g.Handle(proto.LineEvent{Text: in})); !strings.Contains(got, want) || g.phase != discarding {
			t.Errorf("%q: %q, phase %d", in, got, g.phase)
		}
	}

	g := waitDiscard(t, true)
	if got := saidText(g.Handle(proto.LineEvent{Text: g.took.String()})); !strings.Contains(got, lineNotBack[0].Text) {
		t.Errorf("throwing back the card just taken: %q", got)
	}

	// GIN and KNOCK with too much deadwood are refused; the opening hand of seed 3 has plenty.
	g = waitDiscard(t, false)
	c := g.hands[player][0]
	dw := Arrange(cards.Remove(g.hands[player], c)).Points
	if dw <= KnockLimit {
		t.Fatalf("seed 3's hand has only %d deadwood without %s; pick another seed", dw, c)
	}
	if got := saidText(g.Handle(proto.LineEvent{Text: "gin " + c.String()})); !strings.Contains(got, fill(lineNotGin, fmt.Sprint(dw))) {
		t.Errorf("GIN with deadwood: %q", got)
	}
	if got := saidText(g.Handle(proto.LineEvent{Text: "knock with the " + c.String()})); !strings.Contains(got, fill(lineTooMuch, fmt.Sprint(dw))) {
		t.Errorf("KNOCK with too much deadwood: %q", got)
	}
	if got := saidText(g.Handle(proto.LineEvent{Text: "1"})); !strings.Contains(got, fill(lineYouDiscard, c.String())) {
		t.Errorf("discard by position: %q", got)
	}
}

func TestResult(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		you, wopr int
		want      proto.Outcome
	}{{30, 10, proto.Win}, {10, 30, proto.Loss}, {20, 20, proto.Draw}} {
		g := &Game{score: [2]int{tc.you, tc.wopr}}
		if got := g.result().Outcome; got != tc.want {
			t.Errorf("%d to %d: %v, want %v", tc.you, tc.wopr, got, tc.want)
		}
	}
}
