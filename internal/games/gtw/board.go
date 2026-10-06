package gtw

import (
	"fmt"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/assets"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// Board geometry, from Appendix C's big-board screen.
const (
	mapLeft   = 4 // the map box's left edge is at mapLeft-1
	mapTop    = 2 // first map row; the box's top edge is the row above
	defconX   = 68
	labelsRow = mapTop + assets.MapH + 1
	trajRow   = labelsRow + 2
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
	c.Put((c.W-len(lineTitle[0].Text))/2, 0, lineTitle[0].Text, proto.StyleBright, 0)
	c.Put(defconX, 0, lineDefcon[0].Text, proto.StyleLabel, 0)
	g.drawDefcon(c)

	// The map box and the land.
	edge := "+" + strings.Repeat("-", assets.MapW) + "+"
	c.Put(mapLeft-1, mapTop-1, edge, proto.StyleDim, 0)
	c.Put(mapLeft-1, mapTop+assets.MapH, edge, proto.StyleDim, 0)
	for y, row := range assets.Map {
		c.Put(mapLeft-1, mapTop+y, "|", proto.StyleDim, 0)
		c.Put(mapLeft+assets.MapW, mapTop+y, "|", proto.StyleDim, 0)
		c.Put(mapLeft, mapTop+y, row.Text, proto.StyleLand, 0)
	}
	for _, m := range g.missiles {
		g.drawTrack(c, m)
	}
	c.Put(0, labelsRow, lineLabels[0].Text, proto.StyleLabel, 0)
	g.drawTrajectories(c)
	if g.phase == climax {
		code := lineCode[0].Text
		shown := code[:g.cracked] + strings.Repeat("_", len(code)-g.cracked)
		line := lineCodeLabel[0].Text + shown[:3] + " " + shown[3:7] + " " + shown[7:]
		c.Put((c.W-len(line))/2, trajRow+5, line, proto.StyleAlert, proto.AttrBold)
	}
}

// drawDefcon draws the ladder 5..1; the current level is reversed, and blinks at 1.
func (g *Game) drawDefcon(c *proto.Canvas) {
	x := defconX + 1
	c.Put(x, 1, "+---+", proto.StyleDim, 0)
	for i, level := range []int{5, 4, 3, 2, 1} {
		y := 2 + i
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
	}
	c.Put(x, 7, "+---+", proto.StyleDim, 0)
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
	const steps = 40
	for i := 0; i <= int(progress*steps); i++ {
		x, y := arc(m, float64(i)/steps)
		c.Put(mapLeft+x, mapTop+y, glyph, style, 0)
	}
	if progress >= 1 {
		c.Put(mapLeft+m.to.X, mapTop+m.to.Y, "X", style, proto.AttrReverse)
	}
}

func arc(m missile, s float64) (int, int) {
	const lift = 3.0
	x := float64(m.from.X) + float64(m.to.X-m.from.X)*s
	y := float64(m.from.Y) + float64(m.to.Y-m.from.Y)*s - lift*4*s*(1-s)
	return int(x + 0.5), max(int(y+0.5), 0)
}

// drawTrajectories is the film's TRAJECTORY HEADING table: three columns of the player's
// first missiles, two headings each. The figures are illustrative, drawn from the seed.
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
		c.Put(x, trajRow, lineTrajectory[0].Text, proto.StyleLabel, 0)
		c.Put(x, trajRow+1, strings.Repeat("-", len(lineTrajectory[0].Text)), proto.StyleDim, 0)
		letter := string(rune('A' + 2*col))
		h := heading(g.env.Seed, i)
		c.Put(x, trajRow+2, fmt.Sprintf("%s-%s-A %03d %03d", letter, designator, h[0], h[1]), proto.StyleText, 0)
		c.Put(x, trajRow+3, fmt.Sprintf("%*s %03d %03d", len(letter)+len(designator)+3, "B", h[2], h[3]), proto.StyleText, 0)
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
