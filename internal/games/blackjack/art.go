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
	`      '-.                       |    A|    J|                        .-'`,
	`         '--.   ( )     ( )     '-----'-----'      ( )     ( )   .--'`,
	`             '--------------------------------------------------'`,
)

// Hand blocks: the label beside the middle row of the cards and the total after them.
const (
	labelW = 9                       // the label's column
	descW  = 3 + len("(BLACK JACK)") // the widest total after the cards
	cardsW = 80 - labelW - descW     // the cards' room
	midRow = cards.FaceH / 2         // the row the label and total sit on
)

// block draws a hand as cards, with label at the left and desc (when there is one) after.
//
//	         .-----. .-----.
//	         |JH   | |8S   |
//	YOU:     |  H  | |  S  |   (18)
//	         |    J| |    8|
//	         '-----' '-----'
func block(label, desc string, art ...cards.Art) []string {
	out := make([]string, cards.FaceH)
	for y, r := range cards.Row(cardsW, art...) {
		left, right := "", ""
		if y == midRow {
			left = label
			if desc != "" {
				right = "   (" + desc + ")"
			}
		}
		out[y] = strings.TrimRight(fmt.Sprintf("%-*s%s%s", labelW, left, r, right), " ")
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

// dealerRows shows the dealer's hand, with the hole card face down while hidden.
func (g *Game) dealerRows(hidden bool) []string {
	if hidden {
		return block(lineDealer[0].Text, "", cards.Face(g.dealer[0]), cards.Back())
	}
	return block(lineDealer[0].Text, describe(g.dealer), faces(g.dealer)...)
}

// handRows shows the player's hand i.
func (g *Game) handRows(i int) []string {
	label := lineYou[0].Text
	if len(g.hands) > 1 {
		label = fill(lineHandN, fmt.Sprint(i+1))
	}
	h := g.hands[i]
	if h.FromSplit && Natural(h.Cards) {
		return block(label, "21", faces(h.Cards)...) // not a black jack after a split
	}
	return block(label, describe(h.Cards), faces(h.Cards)...)
}
