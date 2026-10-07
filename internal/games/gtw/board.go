package gtw

import (
	"fmt"
	"math"
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
	for _, m := range g.impacts { // earlier stages leave only their impacts
		c.Put(mapLeft+m.to.X, mapTop+m.to.Y, "X", trackStyle(m), proto.AttrReverse)
	}
	for _, m := range g.missiles {
		g.drawTrack(c, m)
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
// colour, since both sides' impacts show.
func (g *Game) drawFoot(c *proto.Canvas) {
	switch {
	case g.phase == climax:
		code := lineCode[0].Text
		shown := code[:g.cracked] + strings.Repeat("_", len(code)-g.cracked)
		line := lineCodeLabel[0].Text + shown[:3] + " " + shown[3:7] + " " + shown[7:]
		c.Put((c.W-len(line))/2, footRow, line, proto.StyleAlert, proto.AttrBold)
	case g.phase == orders && g.stage == strikes-1:
		c.Put(0, footRow, lineLastOrders[0].Text, proto.StyleAlert, proto.AttrBold)
	case g.phase == orders:
		c.Put(0, footRow, lineOrdersHint[0].Text, proto.StyleDim, 0)
	default:
		legend := lineLegend[0].Text
		c.Put(0, footRow, legend, proto.StyleDim, 0)
		c.Put(strings.Index(legend, "+"), footRow, "+", proto.StyleOutgoing, 0)
		c.Put(strings.Index(legend, "*"), footRow, "*", proto.StyleIncoming, 0)
		c.Put(strings.LastIndex(legend, "X"), footRow, "X", proto.StyleText, proto.AttrReverse)
	}
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

// drawDefcon draws the ladder 5..1, a rung per level beside the map; the current level is
// reversed, and blinks at 1.
func (g *Game) drawDefcon(c *proto.Canvas) {
	x := defconX + 1
	c.Put(x, 1, "+---+", proto.StyleDim, 0)
	for i, level := range []int{5, 4, 3, 2, 1} {
		y := 2 + 2*i
		style := [6]proto.Style{0, proto.StyleDefcon1, proto.StyleDefcon2, proto.StyleDefcon3, proto.StyleDefcon4, proto.StyleDefcon5}[level]
		var attr proto.Attr
		if level == g.defcon {
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
// Outgoing tracks are '+', incoming '*'; an impact is a reversed 'X'. Once its stage has
// played out, a track is drawn whole. Every bomber's dots fall on the even columns, so where
// the two sides' routes run together one dotted track covers the other instead of the two
// filling each other's gaps.
func (g *Game) drawTrack(c *proto.Canvas, m missile) {
	if g.phase == flight && g.frame < m.launch {
		return
	}
	progress := 1.0
	if g.phase == flight {
		progress = min(float64(g.frame-m.launch)/float64(flightFrames[m.kind]), 1)
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
	if progress >= 1 {
		c.Put(mapLeft+m.to.X, mapTop+m.to.Y, "X", style, proto.AttrReverse)
	}
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
	r := proto.NewRand(seed, uint64(1000+i))
	return [4]int{r.IntN(1000), r.IntN(1000), r.IntN(1000), r.IntN(1000)}
}

// drawRatios is the projected kill-ratio table: units destroyed on each side, civilian
// losses in percent, people in millions.
func (g *Game) drawRatios(c *proto.Canvas) {
	title := lineRatioTitle[0].Text
	c.Put((c.W-len(title))/2, 0, title, proto.StyleBright, 0)
	player, wopr := "UNITED STATES", "SOVIET UNION"
	if g.side == assets.USSR {
		player, wopr = wopr, player
	}
	head := lineRatioHead.Texts()
	c.Put(4, 2, player, proto.StyleLabel, 0)
	c.Put(54, 2, wopr, proto.StyleLabel, 0)
	c.Put(4, 3, head[0], proto.StyleDim, 0)
	c.Put(54, 3, head[0], proto.StyleDim, 0)

	y := 4
	section := func(name string, rows []string, first int, format func(int) string) {
		c.Put(28, y, name, proto.StyleLabel, proto.AttrUnderline)
		y++
		for i, row := range rows {
			c.Put(28, y, row, proto.StyleText, 0)
			c.Put(8, y, fmt.Sprintf("%9s", format(g.ratios[0][first+i])), proto.StyleOutgoing, 0)
			c.Put(58, y, fmt.Sprintf("%9s", format(g.ratios[1][first+i])), proto.StyleIncoming, 0)
			y++
		}
	}
	count := func(v int) string { return fmt.Sprint(v) }
	section(head[1], lineMilitary.Texts(), 0, count)
	section(head[2], lineCivilian.Texts(), 5, func(v int) string { return fmt.Sprintf("%d%%", v) })
	section(head[3]+" "+lineMillions[0].Text, lineHuman.Texts(), 10, func(v int) string {
		return fmt.Sprintf("%d.%d", v/10, v%10)
	})
}
