// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package gtw

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/assets"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// Board geometry, 80x19 with the front panel: the map box from the top row, the title in its
// top edge and the sides' names in its bottom edge; the DEFCON ladder to its right; under the
// box, the trajectory table on the left and the forces table on the right; then one row that
// shows one thing at a time (drawFoot).
const (
	mapLeft    = 1                     // the map box's left edge is at mapLeft-1
	mapTop     = 1                     // first map row; the box's top edge is the row above
	boxRight   = mapLeft + assets.MapW // the box's right edge
	boxBottom  = mapTop + assets.MapH  // the box's bottom edge
	defconX    = boxRight + 2          // the DEFCON label; the ladder is one column right
	trajRow    = boxBottom + 1         // the trajectory and forces tables' headings
	trajCols   = 2                     // trajectory columns
	trajW      = 21                    // a trajectory column and the gap after it
	forcesW    = 31                    // the forces table: a 7-column name and four 6-column figures
	forcesX    = 80 - forcesW          // right-aligned to the 80-column board, clear of the trajectories
	footRow    = trajRow + 3           // the orders hint, the last-orders warning, the launch code or the legend
	arcLift    = 0.14                  // how far an ICBM's track rises over the map, in rows per column flown
	bomberLift = 0.07                  // your bombers' route rises half as far; WOPR's fly straight in
	slbmLift   = 1.0                   // an SLBM's hop from the sea rises a row, whatever the distance
	lowerLift  = 0.7                   // WOPR's ICBMs fly lower: both sides' tracks show
	arcSteps   = 80                    // points plotted along a track
	usLon      = -98.0                 // the longitude each side's name is centred on
	ussrLon    = 95.0
	aimGlyph   = "@" // a listed target not yet struck; the map's art has none
)

// View implements proto.Program.
func (g *Game) View(c *proto.Canvas) {
	switch g.phase {
	case ratios:
		g.drawRatios(c)
	case flight, orders, assessed, climax:
		g.drawBoard(c)
	case chooseSide, listTargets:
	}
}

func (g *Game) drawBoard(c *proto.Canvas) {
	// The map box, titled, with each side named under its land.
	edge := "+" + strings.Repeat("-", assets.MapW) + "+"
	c.Put(mapLeft-1, mapTop-1, edge, proto.StyleDim, 0)
	c.Put(mapLeft-1, boxBottom, edge, proto.StyleDim, 0)
	title := " " + lineTitle[0].Text + " "
	c.Put(mapLeft+(assets.MapW-len(title))/2, mapTop-1, title, proto.StyleBright, 0)
	for i, lon := range []float64{usLon, ussrLon} {
		name := " " + lineLabels[i].Text + " "
		x, _ := assets.At(45, lon)
		c.Put(mapLeft+x-len(name)/2, boxBottom, name, proto.StyleLabel, 0)
	}
	for y, row := range assets.Map {
		c.Put(mapLeft-1, mapTop+y, "|", proto.StyleDim, 0)
		c.Put(boxRight, mapTop+y, "|", proto.StyleDim, 0)
		c.Put(mapLeft, mapTop+y, row.Text, proto.StyleLand, 0)
	}
	for _, m := range g.missiles {
		g.drawTrack(c, m)
	}
	// Your targets are marked over the tracks, until an impact covers the mark. The film's
	// board marks nothing.
	if !g.film {
		for _, t := range g.targets {
			c.Put(mapLeft+t.X, mapTop+t.Y, aimGlyph, proto.StyleTarget, 0)
		}
	}
	// Impacts go over every track, so no launch hides where a strike landed: a city beside
	// a silo field (Novosibirsk) would otherwise lose its X under the field's launches.
	// Earlier stages leave only their impacts.
	for _, m := range g.impacts {
		c.Put(mapLeft+m.to.X, mapTop+m.to.Y, "X", trackStyle(m), proto.AttrReverse)
	}
	for _, m := range g.missiles {
		if g.progress(m) >= 1 {
			c.Put(mapLeft+m.to.X, mapTop+m.to.Y, "X", trackStyle(m), proto.AttrReverse)
		}
	}
	if !g.film {
		g.drawNames(c)
	}

	c.Put(defconX, 0, lineDefcon[0].Text, proto.StyleLabel, 0)
	g.drawDefcon(c)
	g.drawTrajectories(c)
	g.drawForces(c)
	g.drawFoot(c)
}

// drawFoot fills the row under the tables with one thing, by state: the orders hint at a
// strike prompt (the last-orders warning at the last one), the launch code at the climax, and
// otherwise the legend: + and * in their tracks' colours, and X reversed in the neutral text
// colour, since both sides' impacts show. The film's board (movie mode) has nobody to give
// orders, so it shows the legend at the strike prompts too.
func (g *Game) drawFoot(c *proto.Canvas) {
	switch {
	case g.phase == climax:
		code := lineCode[0].Text
		shown := code[:g.cracked] + strings.Repeat("_", len(code)-g.cracked)
		line := lineCodeLabel[0].Text + shown[:3] + " " + shown[3:7] + " " + shown[7:]
		c.Put((c.W-len(line))/2, footRow, line, proto.StyleAlert, 0)
	case g.phase == orders && g.film:
		g.drawLegend(c)
	case g.phase == orders && g.stage == strikes-1:
		c.Put(0, footRow, lineLastOrders[0].Text, proto.StyleAlert, 0)
	case g.phase == orders:
		c.Put(0, footRow, lineOrdersHint[0].Text, proto.StyleDim, 0)
	default:
		g.drawLegend(c)
	}
}

func (g *Game) drawLegend(c *proto.Canvas) {
	legend := lineLegend[0].Text
	c.Put(0, footRow, legend, proto.StyleDim, 0)
	c.Put(strings.Index(legend, "+"), footRow, "+", proto.StyleOutgoing, 0)
	c.Put(strings.Index(legend, "*"), footRow, "*", proto.StyleIncoming, 0)
	c.Put(strings.LastIndex(legend, "X"), footRow, "X", proto.StyleText, proto.AttrReverse)
	if g.film {
		return
	}
	aim, x := lineLegendAim[0].Text, len(legend)+3
	c.Put(x, footRow, aim, proto.StyleDim, 0)
	c.Put(x+strings.Index(aim, aimGlyph), footRow, aimGlyph, proto.StyleTarget, 0)
}

// drawNames names struck cities beside their impacts: each of your targets once one of your
// warheads has landed on it, in your list's order, then the city of yours that WOPR hit last
// in this stage. A name goes beside its X, to the right or the left, or else on the row above
// or below. It stays inside the map box, over the map's art and tracks only: it never covers
// an impact, a target mark or another name, nor touches one at either end, so that no name
// seems to belong to another X. A name with no room is left out; its X still shows. Every
// impact this stage will make is kept clear from its start, so a name does not move when a
// warhead lands beside it.
func (g *Game) drawNames(c *proto.Canvas) {
	n := names{taken: map[[2]int]bool{}, marks: map[[2]int]bool{}}
	for _, t := range g.targets {
		n.marks[[2]int{mapLeft + t.X, mapTop + t.Y}] = true
	}
	for _, m := range append(slices.Clone(g.impacts), g.missiles...) {
		n.marks[[2]int{mapLeft + m.to.X, mapTop + m.to.Y}] = true
	}
	for _, t := range g.targets {
		if g.struck(t) {
			n.draw(c, t, proto.StyleTarget)
		}
	}
	var last missile
	landed := -1 // the frame the last of WOPR's warheads on your cities landed, in this stage
	for _, m := range g.missiles {
		if !m.ours && g.progress(m) >= 1 && slices.Contains(g.cities, m.to) && m.launch+flightFrames[m.kind] >= landed {
			last, landed = m, m.launch+flightFrames[m.kind]
		}
	}
	if landed >= 0 {
		n.draw(c, last.to, proto.StyleIncoming)
	}
}

// struck reports whether one of your warheads has landed on t.
func (g *Game) struck(t assets.Place) bool {
	for _, m := range g.impacts {
		if m.ours && m.to == t {
			return true
		}
	}
	for _, m := range g.missiles {
		if m.ours && m.to == t && g.progress(m) >= 1 {
			return true
		}
	}
	return false
}

// names is the room drawNames leaves: the cells names take, and the cells of every impact and
// target mark, on the screen.
type names struct{ taken, marks map[[2]int]bool }

// draw puts p's name in the first place beside its X that has room, and takes its cells.
func (n names) draw(c *proto.Canvas, p assets.Place, style proto.Style) {
	x, y, name := mapLeft+p.X, mapTop+p.Y, p.Name
	w := len(name)
	for _, s := range []struct {
		x, y int
		text string
	}{ // right and left of the X, a gap between; then above and below, from its column or to it
		{x + 1, y, " " + name},
		{x - w - 1, y, name + " "},
		{x, y - 1, name},
		{x - w + 1, y - 1, name},
		{x, y + 1, name},
		{x - w + 1, y + 1, name},
	} {
		if !n.room(s.x, s.y, len(s.text), [2]int{x, y}) {
			continue
		}
		c.Put(s.x, s.y, s.text, proto.StyleLand, 0) // the gap beside the X is the map's blank
		c.Put(s.x+strings.Index(s.text, name), s.y, name, style, 0)
		for i := range len(s.text) {
			n.taken[[2]int{s.x + i, s.y}] = true
		}
		return
	}
}

// room reports whether w cells from x, y are inside the map box, free, and with nothing but
// the map's art, a track or own (the name's own X) on either side.
func (n names) room(x, y, w int, own [2]int) bool {
	if y < mapTop || y >= boxBottom || x < mapLeft || x+w > boxRight {
		return false
	}
	for i := -1; i <= w; i++ {
		cell := [2]int{x + i, y}
		if n.taken[cell] || (n.marks[cell] && (cell != own || i >= 0 && i < w)) {
			return false
		}
	}
	return true
}

func trackStyle(m missile) proto.Style {
	if m.ours {
		return proto.StyleOutgoing
	}
	return proto.StyleIncoming
}

// shown is the forces the table shows: each launch counts when it leaves, losses when the
// stage lands.
func (g *Game) shown() [2]forces {
	if g.war == nil {
		return startForces
	}
	snap := g.rep.snap
	switch {
	case g.phase != flight:
		return snap[2]
	case g.stage > strikes && g.frame < finalEnd:
		return snap[1]
	case g.stage > strikes:
		return snap[2]
	case g.frame < detectAt:
		return snap[0]
	case g.frame < strikeEnd:
		return snap[1]
	}
	return snap[2]
}

// drawForces is the table of each side's launchers, yours first as in the kill ratios: ICBMs
// in silos, SLBMs at sea, bombers on the ground and in the air. The nations' names say whose
// figures are whose; the colours only repeat it.
func (g *Game) drawForces(c *proto.Canvas) {
	labels := lineForces.Texts()
	f := g.shown()
	head := fmt.Sprintf("%-7s%6s%6s%6s%6s", labels[0], labels[1], labels[2], labels[3], labels[4])
	c.Put(forcesX, trajRow, head, proto.StyleLabel, 0)
	for i, s := range []int{you, wopr} {
		y := trajRow + 1 + i
		style := proto.StyleOutgoing
		if s == wopr {
			style = proto.StyleIncoming
		}
		c.Put(forcesX, y, g.nation(s), proto.StyleLabel, 0)
		c.Put(forcesX+7, y, fmt.Sprintf("%6d%6d%6d%6d", f[s].ICBM, f[s].SLBM, f[s].Ground, f[s].Air), style, 0)
	}
}

// drawDefcon draws the ladder 5..1, a rung per level beside the map. The current level is
// pointed at (>) and reversed, and blinks at 1. The pointer is what says it in every theme:
// the monochrome themes always reverse the DEFCON 1 rung (their stand-in for white on red),
// and so does norad without colour, so reversal alone would show two current levels.
func (g *Game) drawDefcon(c *proto.Canvas) {
	x := defconX + 1
	c.Put(x, 1, "+---+", proto.StyleDim, 0)
	for i, level := range []int{5, 4, 3, 2, 1} {
		y := 2 + 2*i
		style := [6]proto.Style{0, proto.StyleDefcon1, proto.StyleDefcon2, proto.StyleDefcon3, proto.StyleDefcon4, proto.StyleDefcon5}[level]
		var attr proto.Attr
		if level == g.defcon {
			c.Put(x-1, y, ">", proto.StyleBright, 0)
			attr = proto.AttrReverse
			if level == 1 {
				attr |= proto.AttrBlink
			}
		}
		c.Put(x, y, "|", proto.StyleDim, 0)
		c.Put(x+1, y, fmt.Sprintf(" %d ", level), style, attr)
		c.Put(x+4, y, "|", proto.StyleDim, 0)
		c.Put(x, y+1, "+---+", proto.StyleDim, 0)
	}
}

// drawTrack draws a track's arc so far: an ICBM's rises over the top of the map, as a polar
// route does, a bomber's is lower and dotted, and an SLBM's is a short hop from the sea.
// Outgoing tracks are '+', incoming '*'; drawBoard puts a reversed 'X' over each impact.
// Once its stage has played out, a track is drawn whole. Every bomber's dots fall on the
// even columns, so where the two sides' routes run together one dotted track covers the
// other instead of the two filling each other's gaps.
func (g *Game) drawTrack(c *proto.Canvas, m missile) {
	progress := g.progress(m)
	if progress < 0 {
		return
	}
	glyph, style := "*", trackStyle(m)
	if m.ours {
		glyph = "+"
	}
	for i := 0; i <= int(progress*arcSteps); i++ {
		x, y := arc(m, float64(i)/arcSteps)
		if m.kind == kindBomber && x%2 != 0 {
			continue
		}
		c.Put(mapLeft+x, mapTop+y, glyph, style, 0)
	}
}

// progress is how far along its flight a track is drawn, 0 to 1 (1 has landed), or -1
// before its launch.
func (g *Game) progress(m missile) float64 {
	switch {
	case g.phase != flight:
		return 1
	case g.frame < m.launch:
		return -1
	}
	return min(float64(g.frame-m.launch)/float64(flightFrames[m.kind]), 1)
}

// arc is a track's position at s (0 to 1) along its flight: a longer ICBM or bomber flight
// rises higher. WOPR's ICBMs rise less high than yours, since both sides' ICBMs fly between
// the same two silo fields, and its bombers fly straight in, under yours, since both sides'
// bombers cross the same stretch of the map.
func arc(m missile, s float64) (int, int) {
	dx := float64(m.to.X - m.from.X)
	var lift float64
	switch m.kind {
	case kindSLBM:
		lift = slbmLift
	case kindBomber:
		if m.ours {
			lift = bomberLift * math.Abs(dx)
		}
	case kindICBM:
		lift = arcLift * math.Abs(dx)
		if !m.ours {
			lift *= lowerLift
		}
	}
	x := float64(m.from.X) + dx*s
	y := float64(m.from.Y) + float64(m.to.Y-m.from.Y)*s - lift*4*s*(1-s)
	return int(x + 0.5), max(int(y+0.5), 0)
}

// drawTrajectories is the film's TRAJECTORY HEADING table: both columns, each under an
// underlined heading, show your latest missiles (lay), two headings each. The figures are
// illustrative, drawn from the seed.
func (g *Game) drawTrajectories(c *proto.Canvas) {
	for col := range trajCols {
		c.Put(col*trajW, trajRow, lineTrajectory[0].Text, proto.StyleLabel, proto.AttrUnderline)
	}
	for col, m := range g.traj[:min(len(g.traj), trajCols)] {
		designator := lineDesignator[2*int(g.side-1)+int(m.kind)].Text
		x := col * trajW
		letter := string(rune('A' + 2*col))
		h := heading(g.env.Seed, m.id)
		c.Put(x, trajRow+1, fmt.Sprintf("%s-%s-A %03d %03d", letter, designator, h[0], h[1]), proto.StyleText, 0)
		c.Put(x, trajRow+2, fmt.Sprintf("%*s %03d %03d", len(letter)+len(designator)+3, "B", h[2], h[3]), proto.StyleText, 0)
	}
}

func heading(seed uint64, i int) [4]int {
	r := proto.NewRand(seed, uint64(1000+i)) //nolint:gosec // a stream ID; i is a non-negative index
	return [4]int{r.IntN(1000), r.IntN(1000), r.IntN(1000), r.IntN(1000)}
}

// drawRatios is the projected kill-ratio table in the film's layout: the United States on the
// left and the Soviet Union on the right, whichever side you took; units destroyed in percent
// for the military forces and the civilian assets; and the human cost in millions. The film's
// frame repeats the nations' names above each section and leaves blank lines under the rules,
// which 19 rows have no room for.
func (g *Game) drawRatios(c *proto.Canvas) {
	us, ussr := you, wopr
	if g.side == assets.USSR {
		us, ussr = wopr, you
	}
	style := [2]proto.Style{proto.StyleOutgoing, proto.StyleIncoming} // yours, WOPR's
	rule := strings.Repeat("-", c.W)
	head := lineRatioHead.Texts()
	y := 0
	// A section: its heading row, its rule, and a row per asset with both sides' figures.
	section := func(left, name, right string, rows []string, first int, format func(int) string) {
		c.Put(0, y, left, proto.StyleDim, 0)
		c.Put(27, y, name, proto.StyleLabel, 0)
		c.Put(54, y, right, proto.StyleDim, 0)
		c.Put(0, y+1, rule, proto.StyleDim, 0)
		y += 2
		for i, row := range rows {
			c.Put(0, y, format(g.ratios[us][first+i]), style[us], 0)
			c.Put(27, y, row, proto.StyleText, 0)
			c.Put(54, y, format(g.ratios[ussr][first+i]), style[ussr], 0)
			y++
		}
	}
	percent := func(v int) string { return fmt.Sprintf("     %3d%%", v) }
	millions := func(v int) string { return fmt.Sprintf("%3d %s", v, lineMillion[0].Text) }
	c.Put(1, y, lineLabels[0].Text, proto.StyleLabel, 0)
	c.Put(55, y, lineLabels[1].Text, proto.StyleLabel, 0)
	y++
	section(head[0], head[1], head[0], lineMilitary.Texts(), 0, percent)
	section(head[0], head[2], head[0], lineCivilian.Texts(), 5, percent)
	section(lineLabels[0].Text, head[3], lineLabels[1].Text, lineHuman.Texts(), 10, millions)
}
