// Package assets holds the art and data the film set pieces draw: the big board's map and
// its cities, GTW's side-choice outlines, and the closing montage's scenario names. Every
// item carries provenance (docs/PLAN.md §2.1).
package assets

import (
	"math"
	"slices"
	"strings"

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

// place puts a named point at its latitude and longitude: in the cell that holds it, or, where
// the art draws that coast a little differently and the cell is open water, on the land cell
// whose centre is nearest the true point. Murmansk lies just off the art's Kola coast, '">'
// below it, and goes there, not a column west onto the North Cape's "_._", which is Norway.
func place(name string, side Side, lat, lon float64) Place {
	x, y := At(lat, lon)
	if Sea(x, y) {
		x, y = landNear(lat, lon)
	}
	return Place{Name: name, Side: side, X: x, Y: y}
}

// landNear is the land cell whose centre is nearest a latitude and longitude, measured in
// cells across and down, as the map shows them; a tie goes to the first in reading order.
func landNear(lat, lon float64) (int, int) {
	px, py := (lon-mapWest)/degCol, (mapNorth-lat)/degRow
	bx, by, best := 0, 0, math.Inf(1)
	for y := range MapH {
		for x := range MapW {
			dx, dy := float64(x)+0.5-px, float64(y)+0.5-py
			if d := dx*dx + dy*dy; d < best && !Sea(x, y) {
				bx, by, best = x, y, d
			}
		}
	}
	return bx, by
}

// moved is p moved dx columns east and dy rows south: for a city whose own cell another city
// holds, or that the art draws a column off its true coast. Each use says why, and the
// assets tests pin every city's cell.
func moved(p Place, dx, dy int) Place {
	p.X, p.Y = p.X+dx, p.Y+dy
	return p
}

// Cities are the board's principal targets, and WOPR's aim points: placed from their real
// latitude and longitude, each in a cell of its own. A cell is 5 degrees across and 7.5
// down, so a few share one, or fall where the art draws a coast a column off its true line;
// those are moved a cell, each the way that keeps it in its own country and on the right
// side of its coast.
var Cities = []Place{
	place("SEATTLE", US, 47.61, -122.33),
	place("SAN FRANCISCO", US, 37.77, -122.42),
	place("LOS ANGELES", US, 34.05, -118.24),
	// Las Vegas shares Los Angeles's cell, 0.14 degrees west of the next column east, which
	// holds Arizona; there it stands inland, east of Los Angeles, not on top of it.
	moved(place("LAS VEGAS", US, 36.17, -115.14), 1, 0),
	place("DENVER", US, 39.74, -104.99),
	place("OMAHA", US, 41.26, -95.94),
	place("CHICAGO", US, 41.88, -87.63),
	place("HOUSTON", US, 29.76, -95.37),
	place("ATLANTA", US, 33.75, -84.39),
	// Miami's cell is the art's Gulf coast, the "." of ".-."; the art draws the Florida
	// peninsula a column east, between that coast and the Atlantic's ")", and Miami lies
	// 0.19 degrees short of that column.
	moved(place("MIAMI", US, 25.76, -80.19), 1, 0),
	place("WASHINGTON", US, 38.91, -77.04),
	place("NEW YORK", US, 40.71, -74.01),
	// Boston shares New York's cell; a column east is the art's New England coast, the ","
	// of ",'", and Boston stands there, on the coast north-east of New York.
	moved(place("BOSTON", US, 42.36, -71.06), 1, 0),
	place("LENINGRAD", USSR, 59.93, 30.34),
	place("MURMANSK", USSR, 68.97, 33.08),
	place("MINSK", USSR, 53.90, 27.57),
	place("MOSCOW", USSR, 55.76, 37.62),
	// Kiev and Odessa share a cell, the one above the Black Sea's "<". Odessa keeps it, on
	// the sea's north shore; Kiev, inland and 0.52 degrees east of the column to the west,
	// moves there.
	moved(place("KIEV", USSR, 50.45, 30.52), -1, 0),
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

// Targets are the other places a target list may name, placed the same way from their real
// latitude and longitude: more cities, and the bases a war plan would name. They are not
// WOPR's aim points, and at 5 degrees by 7.5 a cell most share one with a city nearby, as
// they do on a board this coarse: Philadelphia and Baltimore strike Washington's cell, which
// holds them. Each was checked against the art: in its own country, and on the right side of
// its coast. Four are moved a cell, each with its reason.
var Targets = []Place{
	place("SAN DIEGO", US, 32.72, -117.16),
	place("SAN JOSE", US, 37.34, -121.89),
	place("SACRAMENTO", US, 38.58, -121.49),
	place("PORTLAND", US, 45.52, -122.68),
	place("SPOKANE", US, 47.66, -117.43),
	place("BOISE", US, 43.62, -116.20),
	place("RENO", US, 39.53, -119.81),
	place("SALT LAKE CITY", US, 40.76, -111.89),
	place("PHOENIX", US, 33.45, -112.07),
	place("TUCSON", US, 32.22, -110.97),
	place("ALBUQUERQUE", US, 35.08, -106.65),
	place("EL PASO", US, 31.76, -106.49),
	place("COLORADO SPRINGS", US, 38.83, -104.82), // NORAD, in Cheyenne Mountain
	place("CHEYENNE", US, 41.14, -104.82),         // the Minuteman fields: Warren,
	place("GREAT FALLS", US, 47.51, -111.30),      // Malmstrom,
	place("MINOT", US, 48.23, -101.29),            // Minot
	place("GRAND FORKS", US, 47.93, -97.03),       // and Grand Forks
	place("DALLAS", US, 32.78, -96.80),
	place("FORT WORTH", US, 32.76, -97.33),
	place("SAN ANTONIO", US, 29.42, -98.49),
	place("AUSTIN", US, 30.27, -97.74),
	place("OKLAHOMA CITY", US, 35.47, -97.52),
	place("WICHITA", US, 37.69, -97.33),
	place("KANSAS CITY", US, 39.10, -94.58),
	place("ST LOUIS", US, 38.63, -90.20),
	place("MINNEAPOLIS", US, 44.98, -93.27),
	place("MILWAUKEE", US, 43.04, -87.91),
	place("DETROIT", US, 42.33, -83.05),
	place("CLEVELAND", US, 41.50, -81.69),
	place("COLUMBUS", US, 39.96, -83.00),
	place("CINCINNATI", US, 39.10, -84.51),
	place("INDIANAPOLIS", US, 39.77, -86.16),
	place("LOUISVILLE", US, 38.25, -85.76),
	place("PITTSBURGH", US, 40.44, -80.00),
	place("BUFFALO", US, 42.89, -78.88),
	place("PHILADELPHIA", US, 39.95, -75.17),
	place("BALTIMORE", US, 39.29, -76.61),
	place("NORFOLK", US, 36.85, -76.29),
	place("CHARLOTTE", US, 35.23, -80.84),
	place("CHARLESTON", US, 32.78, -79.93),
	place("NASHVILLE", US, 36.16, -86.78),
	place("MEMPHIS", US, 35.15, -90.05),
	place("LITTLE ROCK", US, 34.75, -92.29),
	place("NEW ORLEANS", US, 29.95, -90.07),
	// Jacksonville is on the Atlantic, where the art draws the coast a column east of its
	// true line (as at Miami): its own cell is inland, Atlanta's, and the column east, 1.66
	// degrees away, is the coast's.
	moved(place("JACKSONVILLE", US, 30.33, -81.66), 1, 0),
	place("TAMPA", US, 27.95, -82.46),
	// Orlando, like Miami, falls on the art's Gulf coast, the "." of ".-."; it lies inland,
	// 1.38 degrees short of the column east, where the art draws the Florida peninsula.
	moved(place("ORLANDO", US, 28.54, -81.38), 1, 0),
	place("ANCHORAGE", US, 61.22, -149.90),
	place("HONOLULU", US, 21.31, -157.86), // the art's "`" in the mid-Pacific is Hawaii
	place("TALLINN", USSR, 59.44, 24.75),
	place("RIGA", USSR, 56.95, 24.11),
	place("VILNIUS", USSR, 54.69, 25.28),
	place("KALININGRAD", USSR, 54.71, 20.51),
	place("KISHINEV", USSR, 47.01, 28.86),
	// Sevastopol is on the Crimea, which the art does not draw: its cell is the Black Sea's
	// "<". It lies 0.38 degrees south of the row above, the sea's north shore, and stands
	// there, with Odessa.
	moved(place("SEVASTOPOL", USSR, 44.62, 33.53), 0, -1),
	place("DNEPROPETROVSK", USSR, 48.46, 35.05),
	place("DONETSK", USSR, 48.02, 37.80),
	place("ROSTOV", USSR, 47.24, 39.70),
	place("VOLGOGRAD", USSR, 48.71, 44.51),
	place("KUYBYSHEV", USSR, 53.20, 50.10),
	place("KAZAN", USSR, 55.80, 49.11),
	place("ARKHANGELSK", USSR, 64.54, 40.52),
	place("SEVEROMORSK", USSR, 69.07, 33.42), // the Northern Fleet, beside Murmansk
	place("TBILISI", USSR, 41.72, 44.83),
	place("YEREVAN", USSR, 40.18, 44.50),
	place("BAKU", USSR, 40.41, 49.87), // a port on the art's Caspian, "6"; across it is Turkmenia
	place("ASHKHABAD", USSR, 37.96, 58.33),
	place("SAMARKAND", USSR, 39.63, 66.98),
	place("DUSHANBE", USSR, 38.56, 68.79),
	place("FRUNZE", USSR, 42.87, 74.57),
	place("ALMA-ATA", USSR, 43.22, 76.85),
	place("BAIKONUR", USSR, 45.96, 63.31),
	place("SEMIPALATINSK", USSR, 50.41, 80.23),
	place("PERM", USSR, 58.01, 56.25),
	place("UFA", USSR, 54.74, 55.97),
	place("CHELYABINSK", USSR, 55.16, 61.44),
	place("OMSK", USSR, 54.99, 73.32),
	place("TOMSK", USSR, 56.50, 84.97),
	place("KRASNOYARSK", USSR, 56.02, 92.89),
	place("NORILSK", USSR, 69.36, 88.19),
	place("YAKUTSK", USSR, 62.04, 129.68),
	place("CHITA", USSR, 52.05, 113.47),
	// Khabarovsk stands on the Amur, inland; its cell is the art's Pacific coast, the "/" of
	// "/.", and it lies 0.07 degrees east of the column to the west, which is inland.
	moved(place("KHABAROVSK", USSR, 48.48, 135.07), -1, 0),
	place("MAGADAN", USSR, 59.56, 150.83),
	place("PETROPAVLOVSK", USSR, 53.05, 158.65),
}

// aliases are other names a target list may use for a place the board knows, both sides
// written as key makes them: today's names for the 1983 ones the board uses, and short
// forms. A name that is neither a place's nor an alias is not found; the board strikes
// nothing in its stead.
var aliases = map[string]string{
	"WASHINGTON DC":        "WASHINGTON",
	"DC":                   "WASHINGTON",
	"DISTRICT OF COLUMBIA": "WASHINGTON",
	"PENTAGON":             "WASHINGTON",
	"THE PENTAGON":         "WASHINGTON",
	"NEW YORK CITY":        "NEW YORK",
	"NYC":                  "NEW YORK",
	"MANHATTAN":            "NEW YORK",
	"VEGAS":                "LAS VEGAS",
	"NORAD":                "COLORADO SPRINGS",
	"CHEYENNE MOUNTAIN":    "COLORADO SPRINGS",
	"PEARL HARBOR":         "HONOLULU",

	"ST PETERSBURG":            "LENINGRAD",
	"PETERSBURG":               "LENINGRAD",
	"PETROGRAD":                "LENINGRAD",
	"MOSKVA":                   "MOSCOW",
	"KYIV":                     "KIEV",
	"KHARKIV":                  "KHARKOV",
	"ODESA":                    "ODESSA",
	"NIZHNY NOVGOROD":          "GORKY",
	"NIZHNI NOVGOROD":          "GORKY",
	"GORKI":                    "GORKY",
	"GORKIY":                   "GORKY",
	"YEKATERINBURG":            "SVERDLOVSK",
	"EKATERINBURG":             "SVERDLOVSK",
	"STALINGRAD":               "VOLGOGRAD",
	"SAMARA":                   "KUYBYSHEV",
	"KUIBYSHEV":                "KUYBYSHEV",
	"ROSTOV ON DON":            "ROSTOV",
	"CHISINAU":                 "KISHINEV",
	"DNIPRO":                   "DNEPROPETROVSK",
	"DNIPROPETROVSK":           "DNEPROPETROVSK",
	"SEBASTOPOL":               "SEVASTOPOL",
	"KONIGSBERG":               "KALININGRAD",
	"TIFLIS":                   "TBILISI",
	"ASHGABAT":                 "ASHKHABAD",
	"BISHKEK":                  "FRUNZE",
	"ALMATY":                   "ALMA ATA",
	"TYURATAM":                 "BAIKONUR",
	"LENINSK":                  "BAIKONUR",
	"SEMEY":                    "SEMIPALATINSK",
	"PETROPAVLOVSK KAMCHATSKY": "PETROPAVLOVSK",
}

// key is a name as Find compares it: in capitals, its words single-spaced, full stops and
// apostrophes dropped (D.C., ST. LOUIS) and other marks taken as spaces (ALMA-ATA), and SAINT
// written ST.
func key(name string) string {
	var b strings.Builder
	space := false
	for _, r := range strings.ToUpper(name) {
		switch {
		case r == '.' || r == '\'' || r == '\u2019':
		case r >= 'A' && r <= 'Z' || r >= '0' && r <= '9':
			if space && b.Len() > 0 {
				b.WriteByte(' ')
			}
			space = false
			b.WriteRune(r)
		default:
			space = true
		}
	}
	k := b.String()
	if rest, ok := strings.CutPrefix(k, "SAINT "); ok {
		k = "ST " + rest
	}
	return k
}

// known is every place the board knows, Cities and Targets, by key.
var known = func() map[string]Place {
	m := map[string]Place{}
	for _, p := range append(slices.Clone(Cities), Targets...) {
		m[key(p.Name)] = p
	}
	return m
}()

// Find returns the place a target list names, on either side: a city or base the board
// knows, by the name it had in 1983 (LENINGRAD, GORKY) or another in use (ST PETERSBURG,
// NIZHNY NOVGOROD, WASHINGTON D.C.), in any case and spacing. A name it does not know is not
// found, so a target is only ever struck where it is.
func Find(name string) (Place, bool) {
	k := key(name)
	if a, ok := aliases[k]; ok {
		k = a
	}
	p, ok := known[k]
	return p, ok
}

// placeNames is every place's name, as a target list (LIST) prints it: real place names,
// spelled as the board spells them.
var placeNames = func() script.Ls {
	var names []string
	for _, p := range append(slices.Clone(Cities), Targets...) {
		names = append(names, p.Name)
	}
	return script.Orig(names...)
}()

// Lines is every script block here, for the provenance test. Map is part of World.
var Lines = []script.Ls{World, MapCredit, OutlineUS, OutlineUSSR, Scenarios, placeNames}
