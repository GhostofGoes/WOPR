package theaterwide

import (
	"github.com/GhostofGoes/WOPR/internal/script"
	"github.com/GhostofGoes/WOPR/internal/sim"
)

// The title picture, drawn for this project: a situation map of the central front. Two
// corps a side (XXX over the frame; an X for infantry, a track for armour), the rivers
// behind them, attacks and air wings closing on the front line in the middle, a scale,
// and the sides as the map below has them, yours on the left.
var artTitle = script.Orig(
	`.--[ CENTRAL FRONT ]-------------------------------[ 0 |----|----| 200 KM ]--.`,
	`|  )  .--XXX--.   .--XXX--.           \            .--XXX--.   .--XXX--.   ( |`,
	`| (   |.-----.|   | \   / |   =====>  /            | \   / |   |.-----.|  )  |`,
	`|  )  |'-----'|   | /   \ |           \  <=====    | /   \ |   |'-----'|   ( |`,
	`| (   '-------'   '-------'           /            '-------'   '-------'  )  |`,
	`|  )                        \         \          /                         ( |`,
	`| (                       -=#>        /         <#=-                      )  |`,
	`|  )                        /         \          \                         ( |`,
	`'--[ YOU ]---------------------------------------------------------[ WOPR ]--'`,
)

// artFallout drifts over every region of the map's picture at the tactical nuclear rung.
var artFallout = script.Orig(`* ' * ' *`)

// overlay draws the fallout on the map.
func overlay(s *sim.State, _ int) string {
	if s.Vars["level"] >= 2 {
		return artFallout[0].Text
	}
	return ""
}
