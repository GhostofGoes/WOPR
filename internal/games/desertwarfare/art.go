package desertwarfare

import (
	"github.com/GhostofGoes/WOPR/internal/script"
	"github.com/GhostofGoes/WOPR/internal/sim"
)

// The title picture, drawn for this project: the sea and the shore, a tank on the coast
// road (a cupola on the turret, the gun laid east, six road wheels in the track) with dust
// behind it, and a sign pointing on to Benghazi.
var artTitle = script.Orig(
	`  ~~~ ~~  ~~~~ ~~   ~~~  ~~ ~~~~   ~~  ~~~ ~~   ~~~~  ~~ ~~~   ~~  ~~~~ ~~  ~~~`,
	`_____.--.______.---.___________.--._____________.--.___________.---.___________`,
	`                        ____[_]____                        .------------.`,
	`   .  :  .             /           \____________________=  |  BENGHAZI  |`,
	`  .  :. .: .   _______/_____________\________              |   ----->   |`,
	`    . :.  :.  |                               \            '-----+------'`,
	`   .  .: :.   ( (O)  (O)  (O)  (O)  (O)  (O)  )                  |`,
	`               '-----------------------------'                   |`,
	`===============================================================================`,
)

// artSand is blowing sand over every region of the map's picture while a sandstorm halves
// the attacks.
var artSand = script.Orig(`: . : . :`)

// overlay draws the sandstorm on the map.
func overlay(s *sim.State, _ int) string {
	if s.Vars["sandstorm"] > 0 {
		return artSand[0].Text
	}
	return ""
}
