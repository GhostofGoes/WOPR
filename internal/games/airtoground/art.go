package airtoground

import (
	"fmt"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/script"
)

// artTitle is the title card, drawn for this project: a strike package of three, bombs
// under the wings, seen from above.
var artTitle = script.Orig(
	`                                      /\`,
	`      A I R - T O - G R O U N D      /  \      A C T I O N S`,
	`                                    |(  )|`,
	`                    /\              |    |              /\`,
	`                   /  \            /|    |\            /  \`,
	`                  |(  )|        /   |    |   \        |(  )|`,
	`                  |    |     /______|    |______\     |    |`,
	`                 /|    |\         V | /\ | V         /|    |\`,
	`              /   |    |   \       /_/  \_\       /   |    |   \`,
	`           /______|    |______\                /______|    |______\`,
	`                V | /\ | V                          V | /\ | V`,
	`                 /_/  \_\                            /_/  \_\`,
)

// artImpact is a target going up, drawn for this project: the strike camera's frame on
// a blast over the target.
var artImpact = script.Orig(
	`            .----                       |                       ----.`,
	`            |          .     .    '     |     '    .     .          |`,
	`                   '       *   '.   \   |   /   .'   *       '`,
	`              .  ---  ---  --   (  (  *****  )  )   --  ---  ---  .`,
	`                   .       *   .'   /   |   \   '.   *       .`,
	`                  __      ____.'   /    |    \   '.____      __`,
	`            |  __|  |____|    |___/     |     \___|    |____|  |__  |`,
	`            '----              D I R E C T   H I T              ----'`,
)

// The target area, drawn for this project: a sketch map beside the status table, with
// each target's number, its icon (cratered once destroyed) and a ^ for each of its SAM
// sites still standing. The icons and the SAM marks are drawn in at their places.
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
	areaIcons  = script.Orig(`=|=|=`, `(O)(O)`, `+-+-+-+`, `[=-=-=-=-=]`, `\_/`, `[###]`)
	areaWrecks = script.Orig(`X|X|X`, `(X)(X)`, `X-X-X-X`, `[X-X-X-X-X]`, `\X/`, `[XXX]`)
)

// areaRows is the map row each target's icon is on, in target order.
var areaRows = [numTargets]int{bridge: 3, fuelDepot: 5, yard: 5, airfield: 1, radar: 1, bunker: 5}

// tableW is the status table's width; the map starts after it.
const tableW = 42

// area draws the target area as it stands.
func (g *Game) area() []string {
	rows := make([][]byte, len(areaMap))
	for i, l := range areaMap {
		rows[i] = []byte(l.Text)
	}
	full := newTargets()
	for i := range g.targets {
		t, row := &g.targets[i], rows[areaRows[i]]
		x := strings.Index(string(row), areaIcons[i].Text)
		if x < 0 {
			panic(fmt.Sprintf("airtoground: target %d's icon is not on map row %d", i+1, areaRows[i])) // static art; TestArea keeps it right
		}
		sams := t.sams
		if t.destroyed() {
			copy(row[x:], areaWrecks[i].Text)
			sams = 0
		}
		marks := x + len(areaIcons[i].Text) + 1
		copy(row[marks:], strings.Repeat("^", sams)+strings.Repeat(" ", full[i].sams-sams))
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
