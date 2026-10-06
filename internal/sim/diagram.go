package sim

import (
	"fmt"
	"strings"
)

// The map's picture: the strip drawn as a road across the ground, above the table.
//
//	   _ [] _     /\  /\     .  .  .    .  .  .    /\  /\     _ [] _     _ [] _
//	  |#|##|#|   /  \/  \   . .  .  .  . .  .  .  /  \/  \   |#|##|#|   |#|##|#|
//	 ====(1)========(2)======>>(3)======>>(4)========(5)<<<=====(6)<=======(7)====
//	 <------------------ YOU -------------------><------------ WOPR ------------->
//
// Each region gets a cell of the same width: two rows of ground, its terrain's or the
// scenario's own (a forest, a ruined city), with the scenario's overlay (a gas cloud,
// blowing sand) in place of the top row when there is one; then the road with the
// region's number and its units on it, yours before the number (> each, ~ if hidden) and
// WOPR's after it (< each; its hidden units are not shown), and under the road the
// stretches each side holds. Where your stretch meets WOPR's (><) is the front. The
// picture is always four rows. Nothing rests on colour: the console prints it as plain
// text.

const (
	pictureWidth = 78 // the widest the picture may be, a column clear of each edge
	cellMax      = 13 // the widest a region's cell may be
)

// Indexes into the engine's TextRoad and TextSpan glyphs.
const (
	roadPaving = iota
	roadNumber
	roadYours
	roadHidden
	roadWOPRs
	roadMore
)

const (
	spanLeft = iota
	spanLine
	spanRight
)

// cellWidth is the width of each region's cell for n regions.
func cellWidth(n int) int { return min(cellMax, pictureWidth/max(n, 1)) }

func glyph(k TextKey, i int) string { return engineText[k][i].Text }

// diagram is the map's picture, each line indented so the picture sits centred in 80
// columns.
func (g *Game) diagram() []string {
	s := g.s
	n := len(s.Regions)
	w := cellWidth(n)
	var top, bottom, road strings.Builder
	for r := range s.Regions {
		up, down := g.ground(r)
		pad := strings.Repeat(" ", max(w-max(len(up), len(down)), 0)/2)
		up = fit(pad+up, w)
		if g.sc.Overlay != nil {
			if o := g.sc.Overlay(s, r); o != "" {
				up = centre(o, w)
			}
		}
		top.WriteString(up)
		bottom.WriteString(fit(pad+down, w))
		road.WriteString(roadCell(s, r, w))
	}
	indent := strings.Repeat(" ", max(80-n*w, 0)/2)
	lines := []string{top.String(), bottom.String(), road.String(), spans(s.Control, w)}
	for i, l := range lines {
		lines[i] = strings.TrimRight(indent+l, " ")
	}
	return lines
}

// ground is region r's two rows of ground: the scenario's own, or its terrain's.
func (g *Game) ground(r int) (top, bottom string) {
	if g.sc.Ground != nil {
		if top, bottom = g.sc.Ground(g.s, r); top != "" || bottom != "" {
			return top, bottom
		}
	}
	t := int(g.s.Regions[r].Terrain)
	return engineText[TextGround][2*t].Text, engineText[TextGround][2*t+1].Text
}

// roadCell is region r's stretch of road: its number in the middle, your units before it
// and WOPR's visible units after it, on the paving.
func roadCell(s *State, r, w int) string {
	num := strings.Replace(glyph(TextRoad, roadNumber), "#", fmt.Sprint(r+1), 1)
	left := (w - len(num)) / 2
	right := w - len(num) - left
	// Your hidden units first, then the visible ones next to the number.
	var yours, woprs []string
	for _, hidden := range []bool{true, false} {
		for _, u := range s.In(r, Player) {
			switch {
			case u.Hidden != hidden:
			case hidden:
				yours = append(yours, glyph(TextRoad, roadHidden))
			default:
				yours = append(yours, glyph(TextRoad, roadYours))
			}
		}
	}
	for _, u := range s.In(r, WOPR) {
		if !u.Hidden {
			woprs = append(woprs, glyph(TextRoad, roadWOPRs))
		}
	}
	// More units than fit: as many as fit less one, and a + for the rest.
	more := glyph(TextRoad, roadMore)
	if len(yours) > left {
		yours = append([]string{more}, yours[len(yours)-max(left-1, 0):]...)[:left]
	}
	if len(woprs) > right {
		woprs = append(woprs[:max(right-1, 0)], more)[:right]
	}
	paving := glyph(TextRoad, roadPaving)
	before := strings.Join(yours, "")
	after := strings.Join(woprs, "")
	return strings.Repeat(paving, left-len(before)) + before + num + after + strings.Repeat(paving, right-len(after))
}

// spans marks each run of regions one side holds, <--- YOU --->, under its cells; regions
// nobody holds are left blank.
func spans(control []int, w int) string {
	var b strings.Builder
	for i := 0; i < len(control); {
		j := i
		for j+1 < len(control) && control[j+1] == control[i] {
			j++
		}
		width := (j - i + 1) * w
		i = j + 1
		var label string
		switch control[j] {
		case Player:
			label = " " + engineText[TextYou][0].Text + " "
		case WOPR:
			label = " " + engineText[TextWOPR][0].Text + " "
		default:
			b.WriteString(strings.Repeat(" ", width))
			continue
		}
		line := width - 2 - len(label)
		if line < 0 {
			b.WriteString(centre(label, width))
			continue
		}
		dash := glyph(TextSpan, spanLine)
		b.WriteString(glyph(TextSpan, spanLeft) + strings.Repeat(dash, line/2) + label +
			strings.Repeat(dash, line-line/2) + glyph(TextSpan, spanRight))
	}
	return b.String()
}

// centre centres text in w columns, cutting it to w.
func centre(text string, w int) string {
	text = fit(text, w)
	left := (w - len(text)) / 2
	return strings.Repeat(" ", left) + text + strings.Repeat(" ", w-len(text)-left)
}

// fit pads or cuts text to exactly w columns.
func fit(text string, w int) string {
	if len(text) > w {
		return text[:w]
	}
	return text + strings.Repeat(" ", w-len(text))
}
