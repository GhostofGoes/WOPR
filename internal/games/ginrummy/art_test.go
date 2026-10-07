package ginrummy

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games/cards"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// checkArt reports art that is not printable ASCII capitals, is taller than 12 rows or
// wider than 80 columns, or is missing from lines (where the provenance test checks it).
func checkArt(t *testing.T, art script.Ls, lines []script.Ls) {
	t.Helper()
	if len(art) == 0 || len(art) > 12 {
		t.Errorf("the art is %d rows; a block on the console is at most 12", len(art))
	}
	listed := false
	for _, block := range lines {
		listed = listed || &block[0] == &art[0]
	}
	if !listed {
		t.Error("the art is not in Lines")
	}
	for _, l := range art {
		for _, r := range l.Text {
			if r < 0x20 || r > 0x7e {
				t.Errorf("%q is not printable ASCII", l.Text)
				break
			}
		}
		if len(l.Text) > 80 || l.Text != strings.ToUpper(l.Text) || l.Text != strings.TrimRight(l.Text, " ") {
			t.Errorf("%q: wider than 80 columns, lower case, or trailing spaces", l.Text)
		}
		if l.Prov != script.Original {
			t.Errorf("%q is tagged %q, not original", l.Text, l.Prov)
		}
	}
}

func TestTitleArt(t *testing.T) {
	t.Parallel()
	checkArt(t, artTitle, Lines)
	for _, bad := range cards.CornerProblems(artTitle.Texts()) { // every whole card shows its index twice
		t.Error(bad)
	}
	for _, block := range Lines {
		for _, l := range block {
			if strings.ContainsFunc(l.Text, func(r rune) bool { return r < 0x20 || r > 0x7e }) {
				t.Errorf("%q is not printable ASCII", l.Text)
			}
		}
	}
}

// panelRows is gin rummy's PanelRows in the catalog.
const panelRows = 8

// fitsPanel reports a view that draws below its rows, or a hand that runs into the
// deadwood count beside it.
func fitsPanel(t *testing.T, g *Game) {
	t.Helper()
	c := proto.NewCanvas(80, panelRows+2)
	g.View(c)
	rows := strings.Split(strings.TrimSuffix(c.String(), "\n"), "\n")
	for _, r := range rows[panelRows:] {
		if r != "" {
			t.Fatalf("the view draws below its %d rows:\n%s", panelRows, c.String())
		}
	}
	if hand := rows[handY+1]; len(hand) > asideX && strings.TrimSpace(hand[asideX-groupGap:asideX]) != "" {
		t.Fatalf("the hand runs into the deadwood count:\n%s", c.String())
	}
}

// The panel fits through seeded games, and with the widest hand: three melds and deadwood.
func TestViewFitsThePanel(t *testing.T) {
	t.Parallel()
	for seed := range uint64(10) {
		g := New().(*Game)
		g.Start(proto.Env{Seed: seed, Instant: true, Deterministic: true})
		for step := 0; step < 300; step++ {
			fitsPanel(t, g)
			input := "y"
			switch {
			case g.phase == drawing:
				input = "s"
			case g.phase == discarding:
				input = fmt.Sprint(len(g.hands[player]))
			case g.score[player] >= GameTarget || g.score[wopr] >= GameTarget:
				step = 300
				continue
			}
			g.Handle(proto.LineEvent{Text: input})
		}
	}
	g := New().(*Game)
	g.Start(proto.Env{Seed: 1, Instant: true, Deterministic: true})
	g.hands[player] = hand(t, "AS 2S 3S 7C 7D 7H JD JC JH QD QH")
	g.order()
	fitsPanel(t, g)
}
