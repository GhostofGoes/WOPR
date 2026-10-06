package gtw

import (
	"fmt"
	"math"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/assets"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// Board geometry, 80x19 with the front panel: the map box from the top row, the title in its
// top edge and the sides' names in its bottom edge; the DEFCON ladder to its right; the
// trajectory table and the launch code below.
const (
	mapLeft   = 1                     // the map box's left edge is at mapLeft-1
	mapTop    = 1                     // first map row; the box's top edge is the row above
	boxRight  = mapLeft + assets.MapW // the box's right edge
	boxBottom = mapTop + assets.MapH  // the box's bottom edge
	defconX   = boxRight + 2          // the DEFCON label; the ladder is one column right
	trajRow   = boxBottom + 1         // the trajectory table's heading
	codeRow   = trajRow + 3           // the climax's launch code
	arcLift   = 0.14                  // how far a track rises over the map, in rows per column flown
	arcSteps  = 80                    // points plotted along a track
	usLon     = -98.0                 // the longitude each side's name is centred on
	ussrLon   = 95.0
)

// View implements proto.Program.
func (g *Game) View(c *proto.Canvas) {
	switch g.phase {
	case ratios:
		g.drawRatios(c)
	case exchange, climax:
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

	c.Put(defconX, 0, lineDefcon[0].Text, proto.StyleLabel, 0)
	g.drawDefcon(c)
	g.drawTrajectories(c)
	if g.phase == climax {
		code := lineCode[0].Text
		shown := code[:g.cracked] + strings.Repeat("_", len(code)-g.cracked)
		line := lineCodeLabel[0].Text + shown[:3] + " " + shown[3:7] + " " + shown[7:]
		c.Put((c.W-len(line))/2, codeRow, line, proto.StyleAlert, proto.AttrBold)
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

// drawTrack draws a missile's arc so far: it rises over the top of the map, as a polar
// route does. Outgoing tracks are '+', incoming '*'; an impact is a reversed 'X'.
func (g *Game) drawTrack(c *proto.Canvas, m missile) {
	if g.frame < m.launch {
		return
	}
	progress := min(float64(g.frame-m.launch)/flightFrames, 1)
	glyph, style := "*", proto.StyleIncoming
	if m.ours {
		glyph, style = "+", proto.StyleOutgoing
	}
	for i := 0; i <= int(progress*arcSteps); i++ {
		x, y := arc(m, float64(i)/arcSteps)
		c.Put(mapLeft+x, mapTop+y, glyph, style, 0)
	}
	if progress >= 1 {
		c.Put(mapLeft+m.to.X, mapTop+m.to.Y, "X", style, proto.AttrReverse)
	}
}

// arc is a track's position at s (0 to 1) along its flight: a longer flight rises higher.
func arc(m missile, s float64) (int, int) {
	dx := float64(m.to.X - m.from.X)
	x := float64(m.from.X) + dx*s
	y := float64(m.from.Y) + float64(m.to.Y-m.from.Y)*s - arcLift*math.Abs(dx)*4*s*(1-s)
	return int(x + 0.5), max(int(y+0.5), 0)
}

// drawTrajectories is the film's TRAJECTORY HEADING table: three columns of the player's
// first missiles, two headings each, under an underlined heading. The figures are
// illustrative, drawn from the seed.
func (g *Game) drawTrajectories(c *proto.Canvas) {
	designator := "SS20"
	if g.side == assets.US {
		designator = "MM3"
	}
	col := 0
	for i, m := range g.missiles {
		if !m.ours || col == 3 {
			continue
		}
		x := col * 21
		c.Put(x, trajRow, lineTrajectory[0].Text, proto.StyleLabel, proto.AttrUnderline)
		letter := string(rune('A' + 2*col))
		h := heading(g.env.Seed, i)
		c.Put(x, trajRow+1, fmt.Sprintf("%s-%s-A %03d %03d", letter, designator, h[0], h[1]), proto.StyleText, 0)
		c.Put(x, trajRow+2, fmt.Sprintf("%*s %03d %03d", len(letter)+len(designator)+3, "B", h[2], h[3]), proto.StyleText, 0)
		col++
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
