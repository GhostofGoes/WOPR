// Package assets holds the art and data the film set pieces draw: the big board's map and
// its cities, and the closing montage's scenario names. Every item carries provenance
// (docs/PLAN.md §2.1).
package assets

import (
	"hash/fnv"

	"github.com/GhostofGoes/WOPR/internal/script"
)

// Map size: the big board's map box interior (docs/PLAN.md Appendix C).
const (
	MapW = 57
	MapH = 7
)

// Map is the big board's map, original line-segment art drawn for this project (the film's
// glyphs were custom characters; no fan art is used, L-4): North America on the left,
// the Soviet Union and Asia on the right. Every row is at most MapW columns.
var Map = script.Orig(
	" _..--''--.__   _.-.       __..--''''''''''''''''--.._  ",
	"(            ''' .'  )  _.-'                          '-.",
	" '-.           '-'   /  /                            _.' ",
	"    \\               |   '-.        _.._        __.--'    ",
	"     '.        _.--'       '-.__.-'    '-.  .-'          ",
	"       '-.__.-'                           '-'            ",
	"            '.                                           ",
)

// Side is a nation on the board.
type Side uint8

// Sides.
const (
	US Side = iota + 1
	USSR
)

// Place is a named point on the map, in map columns and rows.
type Place struct {
	Name string
	Side Side
	X, Y int
}

// Cities are the targets the board knows by name. Positions are approximate, chosen for
// this map.
var Cities = []Place{
	{"SEATTLE", US, 5, 2},
	{"SAN FRANCISCO", US, 5, 3},
	{"LOS ANGELES", US, 7, 4},
	{"LAS VEGAS", US, 8, 3},
	{"DENVER", US, 10, 3},
	{"OMAHA", US, 12, 3},
	{"CHICAGO", US, 14, 2},
	{"HOUSTON", US, 11, 4},
	{"ATLANTA", US, 14, 4},
	{"MIAMI", US, 13, 5},
	{"WASHINGTON", US, 16, 3},
	{"NEW YORK", US, 17, 2},
	{"BOSTON", US, 18, 2},
	{"LENINGRAD", USSR, 30, 2},
	{"MURMANSK", USSR, 33, 1},
	{"MINSK", USSR, 29, 3},
	{"MOSCOW", USSR, 32, 2},
	{"KIEV", USSR, 30, 3},
	{"KHARKOV", USSR, 32, 3},
	{"ODESSA", USSR, 31, 4},
	{"GORKY", USSR, 35, 2},
	{"SVERDLOVSK", USSR, 39, 2},
	{"TASHKENT", USSR, 38, 4},
	{"NOVOSIBIRSK", USSR, 44, 2},
	{"IRKUTSK", USSR, 48, 3},
	{"VLADIVOSTOK", USSR, 53, 2},
}

// Silos are where each side's missiles rise from.
var Silos = map[Side]Place{
	US:   {"SILO FIELD", US, 10, 2},
	USSR: {"SILO FIELD", USSR, 41, 3},
}

// Locate returns where a target lies on side's territory: the named city, or, for a name
// the board does not know, a fixed point chosen from the name, so a target always lands
// in the same place.
func Locate(name string, side Side) Place {
	for _, c := range Cities {
		if c.Name == name && c.Side == side {
			return c
		}
	}
	var own []Place
	for _, c := range Cities {
		if c.Side == side {
			own = append(own, c)
		}
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	near := own[int(h.Sum32()%uint32(len(own)))]
	return Place{Name: name, Side: side, X: near.X, Y: near.Y}
}

// Lines is every script block here, for the provenance test.
var Lines = []script.Ls{Map, Scenarios}
