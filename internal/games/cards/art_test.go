package cards_test

import (
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games/cards"
	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// plain reports a line that is not printable ASCII or has a lower-case letter: screen
// text is capitals, and art is pure ASCII (docs/PLAN.md §4.5).
func plain(t *testing.T, what, line string) {
	t.Helper()
	for _, r := range line {
		if r < 0x20 || r > 0x7e {
			t.Errorf("%s: %q is not printable ASCII", what, line)
			return
		}
	}
	if line != strings.ToUpper(line) {
		t.Errorf("%s: %q has lower case", what, line)
	}
}

// Every face is FaceW by FaceH, ASCII, and shows its index (rank and suit letter) at the
// top left, its suit letter in the middle and its rank at the bottom right.
func TestFaces(t *testing.T) {
	t.Parallel()
	check := func(what string, a cards.Art) {
		for _, row := range a {
			if len(row) != cards.FaceW {
				t.Errorf("%s: %q is not %d wide", what, row, cards.FaceW)
			}
			plain(t, what, row)
		}
	}
	for _, c := range cards.Deck() {
		a := cards.Face(c)
		check(c.String(), a)
		if !strings.HasPrefix(a[1], "|"+c.String()) || !strings.Contains(a[2], c.Suit.Letter()) ||
			!strings.HasSuffix(a[3], c.Rank.String()+"|") {
			t.Errorf("%s:\n%s", c, strings.Join(a[:], "\n"))
		}
	}
	check("back", cards.Back())
}

// A row of cards stands apart while it fits and fans out when it does not, each covered
// card still showing its index.
func TestRow(t *testing.T) {
	t.Parallel()
	hand := mustHand(t, "AS 10H QD 7C")
	var art []cards.Art
	for _, c := range hand {
		art = append(art, cards.Face(c))
	}
	apart := cards.Row(80, art...)
	if got := apart[1]; got != "|AS   | |10H  | |QD   | |7C   |" {
		t.Errorf("apart: %q", got)
	}
	fanned := cards.Row(30, art...)
	want := [cards.FaceH]string{
		".---.---.---.-----.",
		"|AS |10H|QD |7C   |",
		"|   |   |   |  C  |",
		"|   |   |   |    7|",
		"'---'---'---'-----'",
	}
	if fanned != want {
		t.Errorf("fanned:\n%s", strings.Join(fanned[:], "\n"))
	}
	for _, w := range []int{80, 31, 30, 1} {
		row := cards.Row(w, art...)
		if len(row[0]) != cards.RowWidth(len(art), w) {
			t.Errorf("width %d: drawn %d wide, RowWidth says %d", w, len(row[0]), cards.RowWidth(len(art), w))
		}
	}
	if cards.RowWidth(0, 80) != 0 || cards.Row(80) != [cards.FaceH]string{} {
		t.Error("no cards, no row")
	}
}

// The panel drawings, with their styles: a trick around the table (the winning card bold,
// a slot for the seat to play), the trick taken, a fan of backs, a hand's card tops, and
// the single cards.
func TestDrawings(t *testing.T) {
	t.Parallel()
	names := [cards.Seats]string{"SOUTH", "WEST", "NORTH", "EAST"}
	c := proto.NewCanvas(80, 16)
	tr := cards.Trick{Leader: cards.West, Cards: mustHand(t, "10H QH")}
	cards.DrawTrick(c, 0, 0, tr, cards.NoTrump, names, tr.Next())
	taken := cards.Trick{Leader: cards.North, Cards: mustHand(t, "KS 2S AS 9D")}
	cards.DrawTaken(c, 60, 0, taken, names, "LAST TRICK", "TAKEN BY "+names[taken.Winner(cards.NoTrump)])
	cards.DrawFan(c, 0, 7, 10, proto.StyleDim)
	cards.DrawMini(c, 30, 7, mustHand(t, "QS")[0], proto.StyleBright, proto.AttrBold)
	cards.DrawMiniBack(c, 37, 7, proto.StyleDim)
	cards.DrawSlot(c, 44, 7, proto.StyleDim)
	end := cards.DrawTops(c, 0, 11, mustHand(t, "KS 10H 4D AC"), proto.StyleText, 0)
	if end != 4*cards.TopW+1 {
		t.Errorf("DrawTops ends at %d", end)
	}
	if cards.FanWidth(10) != 23 || cards.FanWidth(0) != 0 {
		t.Errorf("FanWidth(10) = %d", cards.FanWidth(10))
	}
	for _, row := range strings.Split(c.String(), "\n") {
		plain(t, "drawing", row)
	}
	golden.AssertString(t, "drawings", c.String()+"\n"+c.StyleMap())
}

func TestSuitStyle(t *testing.T) {
	t.Parallel()
	for s, want := range map[cards.Suit]proto.Style{
		cards.Clubs: proto.StyleSuitBlack, cards.Diamonds: proto.StyleSuitRed,
		cards.Hearts: proto.StyleSuitRed, cards.Spades: proto.StyleSuitBlack,
	} {
		if got := cards.SuitStyle(s); got != want {
			t.Errorf("%s: style %d, want %d", s, got, want)
		}
	}
}
