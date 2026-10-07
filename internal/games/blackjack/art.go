package blackjack

import (
	"fmt"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/games/cards"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// The title card, drawn for this project: the half-moon table seen from above, the chip
// rack on the dealer's side, a black jack on the felt and the betting circles at the rail.
var artTitle = script.Orig(
	` .--------------------------------------------------------------------------.`,
	` |   B L A C K   J A C K                           [$$$|$$$|$$$|$$$|$$$]    |`,
	`  \  = = = = = = = = = =        .-----.-----.                              /`,
	`   \                            |AS   |JS   |                             /`,
	`    '.                          |  S  |  S  |                           .'`,
	`      '-.                       |   AS|   JS|                        .-'`,
	`         '--.   ( )     ( )     '-----'-----'      ( )     ( )   .--'`,
	`             '--------------------------------------------------'`,
)

// Hand blocks. A hand is drawn as its cards with its label beside the middle row and its
// total after them. While the player decides, the table is one block: the dealer's up card
// and hole card at the left and the player's hand beside them, so the up card is on screen
// at every decision. The dealer's turn draws the dealer's hand alone.
const (
	width  = 80                             // the console's width
	labelW = 9                              // the label's column left of the dealer's cards
	sideX  = labelW + 2*(cards.FaceW+1) - 1 // where the player's side starts, right of the dealer's two cards
	sideW  = width - sideX                  // the player's side
	sideL  = len("  HAND 1: ")              // the player's label, set against the cards
	midRow = cards.FaceH / 2                // the row the label and total sit on
)

// handArt is one hand to draw: its label, its total ("" while it is hidden) and its cards.
type handArt struct {
	label, desc string
	art         []cards.Art
}

// after is what follows the cards on the middle row: the total, if there is one.
func (h handArt) after() string {
	if h.desc == "" {
		return ""
	}
	return "   (" + h.desc + ")"
}

// fits reports whether h's cards, apart or fanned, fit w columns after a label of lw.
func (h handArt) fits(w, lw int) bool {
	room := w - lw - len(h.after())
	return cards.RowWidth(len(h.art), room) <= room
}

// draw draws h in w columns: the label in the first lw columns, at the left of them or,
// when against is set, against the cards; then the cards, fanned if they must be.
//
//	         .-----. .-----.
//	         |JH   | |8S   |
//	YOU:     |  H  | |  S  |   (18)
//	         |   JH| |   8S|
//	         '-----' '-----'
func (h handArt) draw(w, lw int, against bool) []string {
	out := make([]string, cards.FaceH)
	for y, r := range cards.Row(w-lw-len(h.after()), h.art...) {
		left, right := "", ""
		if y == midRow {
			left, right = h.label, h.after()
		}
		if against && left != "" {
			left += " "
		}
		pad := fmt.Sprintf("%-*s", lw, left)
		if against {
			pad = fmt.Sprintf("%*s", lw, left)
		}
		out[y] = strings.TrimRight(pad+r+right, " ")
	}
	return out
}

func faces(cs []cards.Card) []cards.Art {
	out := make([]cards.Art, len(cs))
	for i, c := range cs {
		out[i] = cards.Face(c)
	}
	return out
}

// dealerArt is the dealer's hand, with the hole card face down while hidden.
func (g *Game) dealerArt(hidden bool) handArt {
	if hidden {
		return handArt{lineDealer[0].Text, "", []cards.Art{cards.Face(g.dealer[0]), cards.Back()}}
	}
	return handArt{lineDealer[0].Text, describe(g.dealer), faces(g.dealer)}
}

// playerArt is the player's hand i.
func (g *Game) playerArt(i int) handArt {
	label := lineYou[0].Text
	if len(g.hands) > 1 {
		label = fill(lineHandN, fmt.Sprint(i+1))
	}
	h := g.hands[i]
	desc := describe(h.Cards)
	if h.FromSplit && Natural(h.Cards) {
		desc = "21" // not a black jack after a split
	}
	return handArt{label, desc, faces(h.Cards)}
}

// dealerRows draws the dealer's hand alone, for the dealer's turn.
func (g *Game) dealerRows() []string { return g.dealerArt(false).draw(width, labelW, false) }

// tableRows draws the table while the player decides: the dealer's up card and hole card,
// the player's hand beside them and a split's second hand below the first.
//
//	         .-----. .-----.          .-----. .-----.
//	         |6H   | |/\/\/|          |8C   | |3D   |
//	DEALER:  |  H  | |\/\/\|  HAND 1: |  C  | |  D  |   (11)
//	         |   6H| |/\/\/|          |   8C| |   3D|
//	         '-----' '-----'          '-----' '-----'
//	                                  .-----. .-----.
//	                                  |8D   | |KS   |
//	                         HAND 2:  |  D  | |  S  |   (18)
//	                                  |   8D| |   KS|
//	                                  '-----' '-----'
//
// A hand too long to sit beside the dealer (eight to ten cards, by its total) puts every
// hand below the dealer's instead, each label at the left.
func (g *Game) tableRows() []string {
	dealer := g.dealerArt(true)
	hands := make([]handArt, len(g.hands))
	beside := true
	for i := range g.hands {
		hands[i] = g.playerArt(i)
		beside = beside && hands[i].fits(sideW, sideL)
	}
	if !beside {
		out := dealer.draw(width, labelW, false)
		for _, h := range hands {
			out = append(out, h.draw(width, labelW, false)...)
		}
		return out
	}
	left := dealer.draw(sideX, labelW, false)
	var out []string
	for i, h := range hands {
		for y, r := range h.draw(sideW, sideL, true) {
			l := ""
			if i == 0 {
				l = left[y]
			}
			out = append(out, strings.TrimRight(fmt.Sprintf("%-*s%s", sideX, l, r), " "))
		}
	}
	return out
}
