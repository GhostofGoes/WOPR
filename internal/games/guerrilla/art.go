package guerrilla

import (
	"github.com/GhostofGoes/WOPR/internal/script"
	"github.com/GhostofGoes/WOPR/internal/sim"
)

// The title picture, drawn for this project: the hills, a patrol helicopter over the
// jungle, and a cell of three in the undergrowth between the palms (fronds drooping from
// the crown) and the broadleaf trees.
var artTitle = script.Orig(
	`              _.--._                          -------------+-------------`,
	`         _.--'      '--._        _.--._             _______|______`,
	`    _.--'                '--.__.'      '--._       /  []  []  |   \________+`,
	`_.-'                                       '--._   \._________|___/        '`,
	`   _.-\ /-._                   _.-\ /-._                         _.-\ /-._`,
	`  '  .-|-.  '    .-.    .-.   '  .-|-.  '   .-.        .-.      '  .-|-.  '`,
	`    '  |  '     (   )  (   )    '  |  '    (   )      (   )       '  |  '`,
	`       |         '-'    '-'        |        '-'        '-'           |`,
	`       |     O    |      | O       |    O    |          |            |`,
	`____________/|\___________/|\__________/|\_____________________________________`,
)

// The ground of the regions that are more than their terrain, in the map's picture: the
// tents of the border camps, palms over the jungle's undergrowth, and the river through
// its valley. The highlands keep the hills and the towns their roofs.
var (
	artCamps  = script.Orig(` /\  /\`, `/__\/__\`)
	artJungle = script.Orig(`_\|/_ _\|/_`, `@@|@@@@|@@`)
	artRiver  = script.Orig(` .  .  .`, `_.-'-._.-'`)
	// By region index: BORDER CAMPS, JUNGLE, RIVER VALLEY.
	artGround = map[int]script.Ls{0: artCamps, 1: artJungle, 3: artRiver}
)

// ground draws the regions' own ground on the map.
func ground(_ *sim.State, r int) (string, string) {
	if art, ok := artGround[r]; ok {
		return art[0].Text, art[1].Text
	}
	return "", ""
}
