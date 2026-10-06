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

// The map's projection, the one its artist drew: equirectangular, 5 degrees of longitude a
// column eastwards from 180W, and 7.5 degrees of latitude a row southwards from 82.5N.
// Missiles arc over the top of the map, as polar routes do.
const (
	mapWest  = -180.0
	mapNorth = 82.5
	degCol   = 5.0
	degRow   = 7.5
)

// World is Matthew Thomas's ASCII world map, exactly as he drew it (his initials
// included), from https://asciiart.website/art/3719, under the terms he gave with it, which
// are MapCredit. NOTICE.md carries that line, and with it every release archive and the
// binary's --licenses text.
var World = script.Tag(script.MatthewThomasMap,
	"           . _..::__:  ,-\"-\"._        |7       ,     _,.__",
	"   _.___ _ _<_>`!(._`.`-.    /         _._     `_ ,_/  '  '-._.---.-.__",
	">.{     \" \" `-==,',._\\{  \\  / {)      / _ \">_,-' `                mt-2_",
	"  \\_.:--.       `._ )`^-. \"'       , [_/(                       __,/-'",
	" '\"'     \\         \"    _L        oD_,--'                )     /. (|",
	"          |           ,'          _)_.\\\\._<> 6              _,' /  '",
	"          `.         /           [_/_'` `\"(                <'}  )",
	"           \\\\    .-. )           /   `-'\"..' `:.#          _)  '",
	"    `        \\  (  `(           /         `:\\  > \\  ,-^.  /' '",
	"              `._,   \"\"         |           \\`'   \\|   ?_)  {\\",
	"                 `=.---.        `._._       ,'     \"`  |' ,- '.",
	"                   |    `-._         |     /          `:`<_|h--._",
	"                   (        >        .     | ,          `=.__.`-'\\",
	"                    `.     /         |     |{|              ,-.,\\     .",
	"                     |   ,'           \\   / `'            ,\"     \\",
	"                     |  /              |_'                |  __  /",
	"                     | |                                  '-'  `-'   \\.",
	"                     |/                                         \"    /",
	"                     \\.                                             '",
	"                      ,/            ______._.--._ _..---.---------._",
	"     ,-----\"-..?----_/ )      __,-'\"             \"                  (",
	"-.._(                  `-----'                                       `-",
)

// MapCredit is the line Matthew Thomas asks to be included with his map.
var MapCredit = script.Tag(script.MatthewThomasMap, "Map (C) 1998 Matthew Thomas. Freely usable if this line is included.")

// Map is the big board's map: World's northern MapH rows, 82.5N to 15S, where the war is
// fought. North America is on the left, Europe and Africa in the middle, the Soviet Union and
// Asia on the right. Every row is at most MapW columns.
var Map = World[:MapH]

// OutlineUS and OutlineUSSR are the side-choice screen's two large outlines, as in the
// film: the contiguous United States and the Soviet Union, original line-segment art made
// for this project from Natural Earth's 50m country polygons (public domain; the Soviet Union
// is the union of its fifteen republics), each in its own Miller window. The polygons were
// filled into a fine grid and each character chosen by marching squares from the land at its
// four corners (_ - | / \ ' .). They have the
// same number of rows; the United States is 36 columns wide at most and the Soviet Union 42,
// so the two fit side by side in 80 columns.
var (
	OutlineUS = script.Orig(
		`  ._________________.`,
		` ./                 \____.      ._.`,
		` |                       |  .___//'`,
		` |                       \__/   /'`,
		` |                            /-'`,
		` '\                          /'`,
		`  '-\                       /'`,
		`    '---\ /\              /-'`,
		`        '-''---\  /------\|`,
		`               '--'      '\.`,
		`                          ''`,
	)
	OutlineUSSR = script.Orig(
		`                  '-\.`,
		`                .___/\_.`,
		`            .___/      \__________.`,
		`  .__.._____/                     \_____.`,
		`  '\ \/                               /-'`,
		` ._/                          /---\/--'`,
		` |                           /'  ./'`,
		` |               /---\  /-\  |   ''`,
		` '---\ /\      /-'   '--' '\/'`,
		`     '\||    /-'           ''`,
		`      '''----'`,
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

// At returns the map cell that holds a latitude and longitude (degrees north and east),
// clamped to the map.
func At(lat, lon float64) (x, y int) {
	x = int(math.Floor((lon - mapWest) / degCol))
	y = int(math.Floor((mapNorth - lat) / degRow))
	return min(max(x, 0), MapW-1), min(max(y, 0), MapH-1)
}

// sea marks the world map's open water: every blank cell reachable from the map's edge, a
// step at a time across or down, without crossing the art's lines. Blank cells the lines
// enclose are land. It is worked out over the whole of World, where every continent closes;
// the board's band alone cuts Africa and South America open.
var sea = func() [][MapW]bool {
	s := make([][MapW]bool, len(World))
	blank := func(x, y int) bool { row := World[y].Text; return x >= len(row) || row[x] == ' ' }
	var stack [][2]int
	for x := range MapW {
		stack = append(stack, [2]int{x, 0}, [2]int{x, len(World) - 1})
	}
	for y := range len(World) {
		stack = append(stack, [2]int{0, y}, [2]int{MapW - 1, y})
	}
	for len(stack) > 0 {
		x, y := stack[len(stack)-1][0], stack[len(stack)-1][1]
		stack = stack[:len(stack)-1]
		if x < 0 || x >= MapW || y < 0 || y >= len(World) || s[y][x] || !blank(x, y) {
			continue
		}
		s[y][x] = true
		stack = append(stack, [2]int{x + 1, y}, [2]int{x - 1, y}, [2]int{x, y + 1}, [2]int{x, y - 1})
	}
	return s
}()

// Sea reports whether a map cell is open water.
func Sea(x, y int) bool { return x >= 0 && x < MapW && y >= 0 && y < MapH && sea[y][x] }

// place puts a named point at its latitude and longitude, on the nearest land if the art
// draws that coast a little differently (Murmansk lies just east of the art's Kola coast).
func place(name string, side Side, lat, lon float64) Place {
	x, y := landNear(At(lat, lon))
	return Place{Name: name, Side: side, X: x, Y: y}
}

// landNear is the nearest cell to x, y that is not sea: the fewest steps across and down,
// then the same row before the one above it, then west before east.
func landNear(x, y int) (int, int) {
	for d := range MapW + MapH {
		for _, dy := range ring(d) {
			for _, dx := range []int{-(d - abs(dy)), d - abs(dy)} {
				if nx, ny := x+dx, y+dy; nx >= 0 && nx < MapW && ny >= 0 && ny < MapH && !Sea(nx, ny) {
					return nx, ny
				}
			}
		}
	}
	return x, y
}

// ring lists the row offsets at a distance of d steps: 0, -1, 1, -2, 2 and so on.
func ring(d int) []int {
	out := []int{0}
	for i := 1; i <= d; i++ {
		out = append(out, -i, i)
	}
	return out
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
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

// Lines is every script block here, for the provenance test. Map is part of World.
var Lines = []script.Ls{World, MapCredit, OutlineUS, OutlineUSSR, Scenarios}
