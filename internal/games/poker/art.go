package poker

import (
	"fmt"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/games/cards"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// The title card, drawn for this project: an oval table with a royal flush fanned on the
// felt and the pot stacked beside it.
var artTitle = script.Orig(
	`         .------------------------------------------------------------.`,
	`      .-'  .---.---.---.---.-----.                              ___    '-.`,
	`    .'     |10S|JS |QS |KS |AS   |    P O K E R                (___) ___  '.`,
	`   |       |   |   |   |   |  S  |    = = = = =            ___ (___)(___)   |`,
	`   |       |   |   |   |   |    A|    WOPR HAS NO TELLS.  (___)(___)(___)   |`,
	`    '.     '---'---'---'---'-----'                        (___)(___)(___) .'`,
	`      '-.                                                              .-'`,
	`         '------------------------------------------------------------'`,
)

// handRows shows side p's hand: label and the hand's name, the cards, and, for the
// player's own hand before the draw, the position under each card that a discard names.
//
//	YOUR HAND: KING HIGH
//	.-----. .-----. .-----. .-----. .-----.
//	|KD   | |5H   | |QC   | |4H   | |3C   |
//	|  D  | |  H  | |  C  | |  H  | |  C  |
//	|    K| |    5| |    Q| |    4| |    3|
//	'-----' '-----' '-----' '-----' '-----'
//	   1       2       3       4       5
func (g *Game) handRows(label script.Ls, p int, positions bool) []string {
	art := make([]cards.Art, len(g.hands[p]))
	for i, c := range g.hands[p] {
		art[i] = cards.Face(c)
	}
	row := cards.Row(80, art...)
	out := append([]string{label[0].Text + " " + Evaluate(g.hands[p]).Name()}, row[:]...)
	if positions {
		var n strings.Builder
		for i := range art {
			fmt.Fprintf(&n, "%*d%*s", cards.FaceW/2+1, i+1, cards.FaceW/2+1, "")
		}
		out = append(out, strings.TrimRight(n.String(), " "))
	}
	return out
}
