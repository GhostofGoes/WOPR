package biotoxic

import (
	"github.com/GhostofGoes/WOPR/internal/script"
	"github.com/GhostofGoes/WOPR/internal/sim"
)

// The title picture, drawn for this project: a gas mask, the agent drifting past, and a
// warning sign over the contaminated ground.
var artTitle = script.Orig(
	`           _.-'''''''-._                ~~ ~ ~~~       /\       ~~~ ~ ~~`,
	`         .'             '.                ~~~ ~ ~~    /  \       ~ ~~ ~~~`,
	`        /  .---.   .---.  \             ~ ~~ ~~~     / || \     ~~ ~~~ ~`,
	`    ---|  /  O  \ /  O  \  |---           ~ ~~ ~    /  ||  \     ~ ~~~ ~`,
	`       |  \     / \     /  |             ~~ ~      /   ..   \     ~~ ~`,
	`        \  '---'   '---'  /                       /__________\`,
	`         '.    .-----.    .'            .-------------------------------.`,
	`           '-. |:::::| .-'              |       CONTAMINATED AREA       |`,
	`              '|:::::|'                 |    MASKS ON.  DO NOT ENTER.   |`,
	`               '-----'                  '-------------------------------'`,
)

// artClouds is the agent over a region in the map's picture, thicker with each level of
// contamination (none, then 1 to 3), as the ~ marks after its name in the table count it.
var artClouds = script.Orig(
	``,
	`~    ~    ~`,
	`~ ~~ ~ ~~ ~`,
	`~~~~~~~~~~~`,
)

// overlay draws the contamination on the map.
func overlay(s *sim.State, r int) string { return artClouds[level(s, r)].Text }
