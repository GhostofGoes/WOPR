package ui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// Black Jack keeps the dealer's up card on the 80x24 screen at every decision, with and
// without the front panel: through hits on one hand, and through a split and hits on both
// of its hands. Each case plays the first seed whose opening deal allows it.
func TestBlackJackKeepsTheUpCardOnScreen(t *testing.T) {
	t.Parallel()
	const seeds = 300
	for _, panel := range []Panel{PanelOff, PanelOn} {
		for _, split := range []bool{false, true} {
			seed := uint64(1)
			for seed <= seeds && !blackJackRound(t, seed, panel, split) {
				seed++
			}
			if seed > seeds {
				t.Errorf("panel %d, split %v: no seed up to %d deals such a round", panel, split, seeds)
			}
		}
	}
}

// upCard finds the dealer's up card beside the face-down hole card.
var upCard = regexp.MustCompile(`\|(\S{2,3}) +\| \|/\\/\\/\|`)

// blackJackRound bets on seed's first deal and hits until the round ends, checking at each
// decision that the dealer's up card is on screen. With split it splits the opening pair
// first. It reports whether the deal allowed the round: three decisions on one hand, or
// two on the split's first hand and one on its second.
func blackJackRound(t *testing.T, seed uint64, panel Panel, split bool) bool {
	t.Helper()
	opts := instant()
	opts.Play, opts.Seed, opts.Panel = "black-jack", seed, panel
	d := newDriver(t, opts, 80, 24).settle().line("5")
	p := promptRow(d)
	m := upCard.FindStringSubmatch(d.screen())
	if !strings.Contains(p, "HIT") || m == nil || split && !strings.Contains(p, "SPLIT?") {
		return false // a natural settled the round, or there is no pair to split
	}
	want := fmt.Sprintf("|%-5s| |/\\/\\/|", m[1])
	if split {
		d.line("P")
	}
	decisions := map[string]int{}
	for p = promptRow(d); strings.Contains(p, "HIT"); p = promptRow(d) {
		hand := "" // the one hand, or a split's HAND 1 or HAND 2
		if strings.HasPrefix(p, "HAND ") {
			hand = p[:len("HAND 1")]
		}
		decisions[hand]++
		if screen := d.screen(); !strings.Contains(screen, want) || !strings.Contains(screen, "DEALER:  |") {
			t.Errorf("seed %d, panel %d: the dealer's %s is off screen at %q:\n%s", seed, panel, m[1], p, screen)
			return true
		}
		d.line("H")
	}
	if split {
		return decisions["HAND 1"] >= 2 && decisions["HAND 2"] >= 1
	}
	return decisions[""] >= 3
}

// promptRow is the screen row the cursor is on: the prompt being answered.
func promptRow(d *driver) string {
	v := d.m.View()
	if v.Cursor == nil {
		return ""
	}
	rows := strings.Split(ansi.Strip(v.Content), "\n")
	if v.Cursor.Y >= len(rows) {
		return ""
	}
	return strings.TrimSpace(rows[v.Cursor.Y])
}
