// Package assets holds the art and data the film set pieces draw: the big board's map and
// its cities, GTW's side-choice outlines, and the closing montage's scenario names. Every
// item carries provenance (docs/PLAN.md §2.1).
package assets

import (
	"hash/fnv"
	"math"

	"github.com/GhostofGoes/WOPR/internal/script"
)

// Map size: the big board's map box interior.
const (
	MapW = 71
	MapH = 13
)

// The map's projection: Miller cylindrical, the whole way round the northern hemisphere from
// the Bering Strait (169W) eastwards, 79N to 10N. Missiles arc over the top, as polar routes
// do.
const (
	mapWest  = -169.0
	mapNorth = 79.0
	mapSouth = 10.0
)

// Map is the big board's map, original line-segment art drawn for this project (the film's
// glyphs were custom characters; no fan art is used, L-4). The coastlines were traced from
// Natural Earth's 110m land polygons (public domain) in the projection above, then drawn by
// hand: North America and Greenland on the left, Europe, Africa's north and the Soviet Union
// and Asia on the right, Alaska and Chukotka meeting across the Bering Strait at the edges.
// Every row is at most MapW columns.
var Map = script.Orig(
	`             \___/ \_        |     \__/       /     _/\`,
	`        _.--._   .-. \       |               /   __/   \____`,
	`/-------------. /   \ \    _/        /--.____-/\/           '--------__`,
	`|              | \__/  \  / (_)    _/   \/                          ___`,
	`\_ __.__       /  /'-.  \/        /  /\                      __    /`,
	` \/    \       \_\/  \         /\ \_/_/                     /  \  /`,
	`        \             \        \/ /                         |   \/`,
	`        |          ___/        __\__  .-. /\                | .`,
	`        |         /            |_/ \\ '-' \/            _-\/ )`,
	`         \    ___ |             /--'._._|  _             | _/`,
	`          \\  / \/             /       \\   \--        _/`,
	`            \ \_/\            /         \\  /  \  /\  (`,
	`             \___ \          (           \\-    \/  \ )`,
)

// OutlineUS and OutlineUSSR are the side-choice screen's two large outlines, as in the
// film: the contiguous United States and the Soviet Union, original line-segment art traced
// from Natural Earth's 50m country polygons (public domain; the Soviet Union is the union of
// its fifteen republics) and drawn by hand. They have the same number of rows; the United
// States is 36 columns wide at most and the Soviet Union 42, so the two fit side by side in
// 80 columns.
var (
	OutlineUS = script.Orig(
		`  _________________`,
		` |                 \___       __/\`,
		` |                    \/\    /    |`,
		` |                       \__/  __/`,
		` |                            /`,
		`  \                           |`,
		`   \                         /`,
		`    \_______               _/`,
		`            \             /`,
		`             \_   ------\ |`,
		`               \_/       \/`,
	)
	OutlineUSSR = script.Orig(
		`                   /\`,
		`              ____/  \_____`,
		`   _  _____/\/             ----------_____`,
		`  | \/                                ___)`,
		` _|                            _    _/`,
		` |                            / \  /`,
		`/                _____  /\    |  \/`,
		`\_              /     \/  \  /`,
		`  \__          /           \/`,
		`     \__/\    /`,
		`          \__/`,
	)
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

// miller is the Miller cylindrical projection's northing of a latitude, in radians.
func miller(lat float64) float64 {
	return 1.25 * math.Log(math.Tan(math.Pi/4+0.4*lat*math.Pi/180))
}

// At returns the map cell that holds a latitude and longitude (degrees north and east).
func At(lat, lon float64) (x, y int) {
	if lon < mapWest {
		lon += 360
	}
	x = int((lon - mapWest) * MapW / 360)
	y = int((miller(mapNorth) - miller(lat)) / (miller(mapNorth) - miller(mapSouth)) * MapH)
	return x, y
}

func place(name string, side Side, lat, lon float64) Place {
	x, y := At(lat, lon)
	return Place{Name: name, Side: side, X: x, Y: y}
}

// Cities are the targets the board knows by name, placed from their real latitude and
// longitude. Neighbours less than a column apart share a cell.
var Cities = []Place{
	place("SEATTLE", US, 47.61, -122.33),
	place("SAN FRANCISCO", US, 37.77, -122.42),
	place("LOS ANGELES", US, 34.05, -118.24),
	place("LAS VEGAS", US, 36.17, -115.14),
	place("DENVER", US, 39.74, -104.99),
	place("OMAHA", US, 41.26, -95.94),
	place("CHICAGO", US, 41.88, -87.63),
	place("HOUSTON", US, 29.76, -95.37),
	place("ATLANTA", US, 33.75, -84.39),
	place("MIAMI", US, 25.76, -80.19),
	place("WASHINGTON", US, 38.91, -77.04),
	place("NEW YORK", US, 40.71, -74.01),
	place("BOSTON", US, 42.36, -71.06),
	place("LENINGRAD", USSR, 59.93, 30.34),
	place("MURMANSK", USSR, 68.97, 33.08),
	place("MINSK", USSR, 53.90, 27.57),
	place("MOSCOW", USSR, 55.76, 37.62),
	place("KIEV", USSR, 50.45, 30.52),
	place("KHARKOV", USSR, 49.99, 36.23),
	place("ODESSA", USSR, 46.48, 30.73),
	place("GORKY", USSR, 56.33, 44.00),
	place("SVERDLOVSK", USSR, 56.84, 60.61),
	place("TASHKENT", USSR, 41.30, 69.24),
	place("NOVOSIBIRSK", USSR, 55.01, 82.93),
	place("IRKUTSK", USSR, 52.29, 104.30),
	place("VLADIVOSTOK", USSR, 43.12, 131.89),
}

// Silos are where each side's missiles rise from: the missile fields of the northern Great
// Plains, and of southern Siberia.
var Silos = map[Side]Place{
	US:   place("SILO FIELD", US, 47.5, -104.0),
	USSR: place("SILO FIELD", USSR, 55.3, 89.8),
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
var Lines = []script.Ls{Map, OutlineUS, OutlineUSSR, Scenarios}
