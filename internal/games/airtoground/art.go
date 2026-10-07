package airtoground

import (
	"fmt"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/script"
)

// artTitle is the title card, drawn for this project: a strike package of three in a
// vic, seen from above, over the game's name. At seven rows the whole start screen fits
// 80x24 with the front panel; the name is on the last row, so anything that does scroll
// off is the jets' noses, not the name.
var artTitle = script.Orig(
	`                                        /\`,
	`                                     __/||\__`,
	`                  /\                /___||___\                /\`,
	`               __/||\__               /_\/_\               __/||\__`,
	`              /___||___\                                  /___||___\`,
	`                /_\/_\                                      /_\/_\`,
	`            ---===[ A I R - T O - G R O U N D   A C T I O N S ]===---`,
)

// The target area, drawn for this project: a sketch map beside the status table, with
// each target's number, its icon (cratered once destroyed) and a ^ for each of its SAM
// sites still standing. The icons and the SAM marks are drawn in at their places.
//
// The status straight after a sortie that destroys a target is the strike camera's view:
// the frame is titled with the hit, and rays burst from the fresh wreck on the map rows
// above and below it (none where that row is the frame). Each burst starts two columns
// left of its icon and clears the icon's width and two columns either side.
var (
	areaMap = script.Orig(
		`.----------- TARGET AREA -----------.`,
		`|  /\   5 \_/ ^^   4 [=-=-=-=-=] ^^ |`,
		`| /  \/\              :             |`,
		`|~~~~~~~~~~~~~~~ 1 =|=|= ^ ~~~~~~~~~|`,
		`|   /\/\              :             |`,
		`| 6 [###] ^^^  3 +-+-+-+ 2 (O)(O) ^ |`,
		`'--- ^ SAM SITE ---- X DESTROYED ---'`,
	)
	areaIcons   = script.Orig(`=|=|=`, `(O)(O)`, `+-+-+-+`, `[=-=-=-=-=]`, `\_/`, `[###]`)
	areaWrecks  = script.Orig(`X|X|X`, `(X)(X)`, `X-X-X-X`, `[X-X-X-X-X]`, `\X/`, `[XXX]`)
	areaHit     = script.Orig(`.--------- DIRECT HIT ON # ---------.`)
	burstsAbove = script.Orig(`  \ | /`, ` \  ||  /`, `  \  |  /`, ``, ``, `  \ | /`)
	burstsBelow = script.Orig(`  / | \`, ``, ``, `  / .  |  . \`, ` / | \`, ``)
)

// areaRows is the map row each target's icon is on, in target order.
var areaRows = [numTargets]int{bridge: 3, fuelDepot: 5, yard: 5, airfield: 1, radar: 1, bunker: 5}

// tableW is the status table's width; the map starts after it.
const tableW = 42

// area draws the target area as it stands. hit is the target the last sortie destroyed,
// which gets the strike camera's burst, or -1.
func (g *Game) area(hit int) []string {
	rows := make([][]byte, len(areaMap))
	for i, l := range areaMap {
		rows[i] = []byte(l.Text)
	}
	full := newTargets()
	for i := range g.targets {
		t, y := &g.targets[i], areaRows[i]
		x := strings.Index(string(rows[y]), areaIcons[i].Text)
		if x < 0 {
			panic(fmt.Sprintf("airtoground: target %d's icon is not on map row %d", i+1, y)) // static art; TestAreaMap keeps it right
		}
		sams := t.sams
		if t.destroyed() {
			copy(rows[y][x:], areaWrecks[i].Text)
			sams = 0
		}
		marks := x + len(areaIcons[i].Text) + 1
		copy(rows[y][marks:], strings.Repeat("^", sams)+strings.Repeat(" ", full[i].sams-sams))
		if i != hit {
			continue
		}
		span := len(areaIcons[i].Text) + 4
		if burst := burstsAbove[i].Text; burst != "" {
			copy(rows[y-1][x-2:], fmt.Sprintf("%-*s", span, burst))
		}
		if burst := burstsBelow[i].Text; burst != "" {
			copy(rows[y+1][x-2:], fmt.Sprintf("%-*s", span, burst))
		}
	}
	if hit >= 0 {
		rows[0] = []byte(fill(areaHit[0].Text, hit+1))
	}
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = string(r)
	}
	return out
}

// beside puts the map to the right of the table, row by row.
func beside(table, area []string) []string {
	out := make([]string, max(len(table), len(area)))
	for i := range out {
		var t, a string
		if i < len(table) {
			t = table[i]
		}
		if i < len(area) {
			a = area[i]
		}
		out[i] = strings.TrimRight(fmt.Sprintf("%-*s%s", tableW, t, a), " ")
	}
	return out
}
