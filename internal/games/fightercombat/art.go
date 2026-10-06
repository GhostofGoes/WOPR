package fightercombat

import (
	"strings"

	"github.com/GhostofGoes/WOPR/internal/script"
)

// artTitle is the title card, drawn for this project: a fighter head-on in a gunsight.
var artTitle = script.Orig(
	`                                        |`,
	`                             .-'''''''''+'''''''''-.`,
	`                         .-'                       '-.`,
	`                       .'       \               /       '.`,
	`                     /           \     ___     /           \`,
	`   F I G H T E R ---|    _________\___/   \___/_________    |--- C O M B A T`,
	`                     \   '--+----+===[_] [_]===+----+--'   /`,
	`                       '.   V    V    '---'    V    V   .'`,
	`                         '-.                       .-'`,
	`                             '-.........+.........-'`,
	`                                        |`,
)

// artSplash is a kill, drawn for this project: a burst with the call in the middle, filled
// in from splashCalls.
var artSplash = script.Orig(
	`                              .    *    |    *    .`,
	`                      *    '.    \   \  |  /   /    .'    *`,
	`                .     '.     \    \  \  |  /  /    /     .'     .`,
	`         - -- --- ----=====<<  #  >>=====---- --- -- -`,
	`                '     .'     /    /  /  |  \  \    \     '.     '`,
	`                      *    .'    /   /  |  \   \    '.    *`,
	`                              '    *    |    *    '`,
)

// splashCalls fill artSplash: one aircraft down, or both.
var splashCalls = script.Orig("S P L A S H   O N E", "S P L A S H   T W O")

// artEject is the player's aircraft going down, drawn for this project: the pilot under a
// canopy while the wreck trails smoke into the ground.
var artEject = script.Orig(
	`                _.-'''''''''-._        ~~`,
	`             .-'  .'   |   '.  '-.       ~~ ~`,
	`            /____/_____|_____\____\         ~~ ~`,
	`            \     \    |    /     /            ~~ ~`,
	`              \    \   |   /    /                 ~~ ~`,
	`                '.  \  |  /  .'                      ~~ ~   .  '`,
	`                  '-.\ | /.-'                  -- ---===( * )===--- --`,
	`                     \(O)/                     _________/  |  \_________`,
	`                      /|\       E J E C T`,
)

// The tactical picture, all original: a side view of the fight, height for energy, the gap
// for the range, and which way each aircraft points for the aspect; a damaged one trails
// smoke.
var (
	picScale   = script.Orig("HIGH :", " MID :", " LOW :")
	picPlanes  = script.Orig(`\==>`, `<==/`, `~~`)
	picNames   = script.Orig("YOU", "WOPR")
	picCaption = script.Orig("RANGE #. #.")
	picAspects = script.Orig("WOPR IS ON YOUR TAIL", "HEAD-ON", "YOU ARE ON ITS TAIL")
)

// Picture geometry: the scale's column, where the player's label starts, where WOPR's
// aircraft sits at each range, and the ground line's span.
const (
	picScaleX  = 2
	picYouX    = 9
	picGroundX = 7
	picWidth   = 76
)

var picWoprX = [...]int{rangeClose: 30, 2: 46, rangeLong: 62} // close, medium, long

// band is the picture's row for an energy: high, middle or low.
func band(energy int) int {
	switch {
	case energy >= 5:
		return 0
	case energy >= 2:
		return 1
	}
	return 2
}

// picture draws the fight as it stands: three rows of sky and a ground line captioned with
// the range and the aspect. The player is always at the left, WOPR to the right.
//
//	HIGH :                                                 \==> WOPR
//	 MID :  YOU ~~\==>
//	 LOW :
//	      '-------------- RANGE MEDIUM. YOU ARE ON ITS TAIL. --------------
func (g *Game) picture() []string {
	rows := make([][]byte, 3)
	for i := range rows {
		rows[i] = []byte(strings.Repeat(" ", picWidth))
		copy(rows[i][picScaleX:], picScale[i].Text)
	}
	put := func(row, x int, s string) { copy(rows[row][x:], s) }
	right, left, smoke := picPlanes[0].Text, picPlanes[1].Text, picPlanes[2].Text

	// The player: label, then the aircraft with any smoke behind its tail.
	plane := left
	if g.pos >= 0 {
		plane = right
	}
	unit := plane
	if g.hits[0] > 0 {
		if plane == right {
			unit = smoke + plane
		} else {
			unit = plane + smoke
		}
	}
	put(band(g.energy[0]), picYouX, picNames[0].Text+" "+unit)

	// WOPR: the aircraft at its range, smoke behind it, then the label.
	plane = left
	if g.pos > 0 {
		plane = right
	}
	x, unit := picWoprX[g.dist], plane
	if g.hits[1] > 0 {
		if plane == right {
			x, unit = x-len(smoke), smoke+plane
		} else {
			unit = plane + smoke
		}
	}
	put(band(g.energy[1]), x, unit+" "+picNames[1].Text)

	out := make([]string, 0, 4)
	for _, r := range rows {
		out = append(out, strings.TrimRight(string(r), " "))
	}
	caption := " " + fill(picCaption[0].Text, lineRanges[g.dist].Text, picAspects[g.pos+1].Text) + " "
	ground := []byte("'" + strings.Repeat("-", picWidth-picGroundX-1))
	copy(ground[(len(ground)-len(caption))/2:], caption)
	return append(out, strings.Repeat(" ", picGroundX)+string(ground))
}

// splash is the kill art with its call filled in: both aircraft down, or only WOPR's.
func splash(both bool) []string {
	call := splashCalls[0].Text
	if both {
		call = splashCalls[1].Text
	}
	out := artSplash.Texts()
	for i, l := range out {
		out[i] = fill(l, call) // only the middle row has a #
	}
	return out
}
