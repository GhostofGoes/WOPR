package cards

// Card art, drawn for this project in plain ASCII. Every card shows its rank and its suit
// letter, so that no card rests on colour (docs/PLAN.md §4.5).
//
// The console draws faces five rows high, apart when they fit and fanned when they do not.
// A face shows its index in two corners, as a real card does: at the top left, and right
// aligned at the bottom right. A fanned card covered by the next shows only its top index:
//
//	.-----. .-----.       .---.---.---.-----.
//	|10H  | |/\/\/|       |AS |7D |KC |4H   |
//	|  H  | |\/\/\|       |   |   |   |  H  |
//	|  10H| |/\/\/|       |   |   |   |   4H|
//	'-----' '-----'       '---'---'---'-----'
//
// Panels draw mini cards three rows high, face up, face down, as an empty slot or, on a
// trick table, as the card winning the trick so far (a doubled edge, not colour or bold,
// says which), and a held hand as a row of card tops:
//
//	.---.  .---.  . - .  .===.       .---.---.---.---.
//	|QS |  |/\/|  :   :  |AS |       |KS |QS |4S |AH |
//	'---'  '---'  ' - '  '==='

import (
	"fmt"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/proto"
)

// Sizes of the drawn cards.
const (
	FaceW = 7 // a console card's width
	FaceH = 5 // and its height
	MiniW = 5 // a panel card's width
	MiniH = 3 // and its height
	TopW  = 4 // the columns each card takes in DrawTops

	fanStep = 4 // the columns of a covered card that a fanned row shows: its edge and index
)

// Art is a card drawn as rows of text, each FaceW wide.
type Art [FaceH]string

// Face draws c face up: its index (rank and suit) at the top left, its suit letter in the
// middle and its index again at the bottom right, right aligned.
func Face(c Card) Art {
	idx := c.String()
	pad := strings.Repeat(" ", FaceW-2-len(idx))
	return Art{
		".-----.",
		"|" + idx + pad + "|",
		"|  " + c.Suit.Letter() + "  |",
		"|" + pad + idx + "|",
		"'-----'",
	}
}

// Back draws a card face down.
func Back() Art {
	return Art{".-----.", `|/\/\/|`, `|\/\/\|`, `|/\/\/|`, "'-----'"}
}

// RowWidth is the width Row draws n cards in when it may use width columns.
func RowWidth(n, width int) int {
	if n == 0 {
		return 0
	}
	if apart := n*(FaceW+1) - 1; apart <= width {
		return apart
	}
	return (n-1)*fanStep + FaceW
}

// Row lays cards side by side, a column apart. When that is wider than width it fans them,
// each card but the last showing only its edge and index (a face down card, its back).
// Rows are RowWidth wide.
func Row(width int, cs ...Art) [FaceH]string {
	var out [FaceH]string
	if len(cs) == 0 {
		return out
	}
	fan := RowWidth(len(cs), width) < len(cs)*(FaceW+1)-1
	for y := range out {
		var b strings.Builder
		for i, a := range cs {
			switch {
			case i == len(cs)-1:
				b.WriteString(a[y])
			case fan && y > 1 && y < FaceH-1 && a != Back(): // under the next card: the index shows
				b.WriteString(a[y][:1] + strings.Repeat(" ", fanStep-1))
			case fan:
				b.WriteString(a[y][:fanStep])
			default:
				b.WriteString(a[y] + " ")
			}
		}
		out[y] = b.String()
	}
	return out
}

// CornerProblems lists each card face drawn whole in rows (a FaceW by FaceH frame with a
// card's index at its top left) whose bottom right corner does not repeat that index,
// right aligned. Faces covered by the next card in a fan, backs and mini cards are not
// whole faces. Tests check drawn cards and title art with it.
func CornerProblems(rows []string) []string {
	at := func(x, y int) byte {
		if y < 0 || y >= len(rows) || x < 0 || x >= len(rows[y]) {
			return 0
		}
		return rows[y][x]
	}
	edge := func(x, y int) bool { // a frame's side, from its top corner to its bottom one
		return at(x, y-1) == '.' && at(x, y) == '|' && at(x, y+1) == '|' && at(x, y+2) == '|' && at(x, y+3) == '\''
	}
	var bad []string
	for y := 1; y+3 < len(rows); y++ {
		for x := range len(rows[y]) - FaceW + 1 {
			right := x + FaceW - 1
			if !edge(x, y) || !edge(right, y) || rows[y-1][x+1:right] != "-----" || rows[y+3][x+1:right] != "-----" {
				continue
			}
			idx := strings.TrimRight(rows[y][x+1:right], " ")
			if c, ok := Parse(idx); !ok || c.String() != idx {
				continue // a back, or not a card
			}
			if got, want := rows[y+2][x+1:right], strings.Repeat(" ", FaceW-2-len(idx))+idx; got != want {
				bad = append(bad, fmt.Sprintf("the %s at row %d, column %d has %q at its bottom right, not %q", idx, y, x, got, want))
			}
		}
	}
	return bad
}

// SuitStyle is the style a suit's letters are drawn in: red for hearts and diamonds,
// black for clubs and spades. The letter itself always says which suit it is.
func SuitStyle(s Suit) proto.Style {
	if s == Hearts || s == Diamonds {
		return proto.StyleSuitRed
	}
	return proto.StyleSuitBlack
}

// DrawMini draws c face up as a mini card with its top left corner at x, y: the frame in
// frame, the index in its suit's style, both with attributes a.
func DrawMini(cv *proto.Canvas, x, y int, c Card, frame proto.Style, a proto.Attr) {
	drawMini(cv, x, y, c, frame, a, "---")
}

// DrawMiniWinning draws c as DrawMini does, with the doubled top and bottom edge that marks
// the card winning a trick so far: .===.
func DrawMiniWinning(cv *proto.Canvas, x, y int, c Card, frame proto.Style, a proto.Attr) {
	drawMini(cv, x, y, c, frame, a, "===")
}

func drawMini(cv *proto.Canvas, x, y int, c Card, frame proto.Style, a proto.Attr, edge string) {
	cv.Put(x, y, "."+edge+".", frame, a)
	cv.Put(x, y+1, "|   |", frame, a)
	cv.Put(x+1, y+1, c.String(), SuitStyle(c.Suit), a)
	cv.Put(x, y+2, "'"+edge+"'", frame, a)
}

// DrawMiniBack draws a mini card face down at x, y.
func DrawMiniBack(cv *proto.Canvas, x, y int, st proto.Style) {
	cv.Put(x, y, ".---.", st, 0)
	cv.Put(x, y+1, `|/\/|`, st, 0)
	cv.Put(x, y+2, "'---'", st, 0)
}

// DrawSlot draws the empty place of a card still to come, as a broken outline, at x, y.
func DrawSlot(cv *proto.Canvas, x, y int, st proto.Style) {
	cv.Put(x, y, ". - .", st, 0)
	cv.Put(x, y+1, ":   :", st, 0)
	cv.Put(x, y+2, "' - '", st, 0)
}

// FanWidth is the width DrawFan takes for n cards.
func FanWidth(n int) int {
	if n <= 0 {
		return 0
	}
	return 2*(n-1) + MiniW
}

// DrawFan draws n mini cards face down at x, y, overlapped two columns apart: a hand seen
// from across the table.
func DrawFan(cv *proto.Canvas, x, y, n int, st proto.Style) {
	if n <= 0 {
		return
	}
	for i := range n - 1 {
		cv.Put(x+2*i, y, ".-", st, 0)
		cv.Put(x+2*i, y+1, "|/", st, 0)
		cv.Put(x+2*i, y+2, "'-", st, 0)
	}
	DrawMiniBack(cv, x+2*(n-1), y, st)
}

// The trick table's size, and where each seat's card lies on it.
const (
	TrickW = 46
	TrickH = 6
)

// trickSpots are where each seat's card lies on the trick table: North at the top, West
// and East at the sides, South at the bottom, overlapping in rows as a diamond.
var trickSpots = [Seats]struct{ x, y int }{South: {21, 3}, West: {7, 2}, North: {21, 0}, East: {35, 2}}

// trickLabel is the column of a seat's name: right of East's card, and left of the others.
func trickLabel(seat, cardX int, name string) int {
	switch seat {
	case East:
		return cardX + MiniW + 1
	case West:
		return cardX - 1 - len(name)
	}
	return cardX - 2 - len(name)
}

// DrawTrick draws t around a table at x, y, TrickW by TrickH: each seat's card where the
// seat sits, labelled with its name from names, with the card winning so far edged in =
// (its style is already bold, so bold could not say it). The seat next (or none, when
// next < 0) gets an empty slot until it plays.
//
//	                   .===.
//	            NORTH  |QS |
//	     .---.         '==='         .---.
//	WEST |10H|         . - .         |2H | EAST
//	     '---'  SOUTH  :   :         '---'
//	                   ' - '
func DrawTrick(cv *proto.Canvas, x, y int, t Trick, trump Suit, names [Seats]string, next int) {
	winning := -1
	if len(t.Cards) > 0 {
		winning = t.Winner(trump)
	}
	for s, at := range trickSpots {
		cx, cy := x+at.x, y+at.y
		cv.Put(trickLabel(s, cx, names[s]), cy+1, names[s], proto.StyleLabel, 0)
		card, ok := t.Played(s)
		switch {
		case ok && s == winning:
			DrawMiniWinning(cv, cx, cy, card, proto.StyleBright, 0)
		case ok:
			DrawMini(cv, cx, cy, card, proto.StyleBright, 0)
		case s == next:
			DrawSlot(cv, cx, cy, proto.StyleDim)
		}
	}
}

// DrawPass draws a pass on the empty trick table at x, y (placed as DrawTrick places it):
// three cards face down in the middle of the table and an arrow from them to the seat to,
// which gets them. South passes, so to is West, North or East; South draws nothing. A
// pass to the left:
//
//	            NORTH
//	                   .-.-.---.
//	WEST <------------ |/|/|/\/|          EAST
//	              SOUTH'-'-'---'
func DrawPass(cv *proto.Canvas, x, y, to int) {
	if to == South {
		return
	}
	south := trickSpots[South]
	fx, fy := x+south.x, y+trickSpots[West].y // the fan, level with the side seats' cards
	DrawFan(cv, fx, fy, 3, proto.StyleDim)
	mid := fy + MiniH/2
	switch to {
	case West:
		from := x + trickSpots[West].x
		cv.Put(from, mid, "<"+strings.Repeat("-", fx-2-from), proto.StyleAlert, 0)
	case East:
		from, end := fx+FanWidth(3)+1, x+trickSpots[East].x+MiniW-1
		cv.Put(from, mid, strings.Repeat("-", end-from)+">", proto.StyleAlert, 0)
	case North:
		nx := x + trickSpots[North].x + MiniW/2
		cv.Put(nx, y+trickSpots[North].y, "^", proto.StyleAlert, 0)
		for row := y + trickSpots[North].y + 1; row < fy; row++ {
			cv.Put(nx, row, "|", proto.StyleAlert, 0)
		}
	}
}

// DrawTaken lists a taken trick at x, y: title, each seat's name and card in play order,
// then taken (who took it).
//
//	LAST TRICK
//	WEST   10H
//	NORTH  KH
//	EAST   2S
//	SOUTH  AD
//	TAKEN BY NORTH
func DrawTaken(cv *proto.Canvas, x, y int, t Trick, names [Seats]string, title, taken string) {
	cv.Put(x, y, title, proto.StyleLabel, 0)
	for i, c := range t.Cards {
		s := t.Seat(i)
		cv.Put(x, y+1+i, names[s], proto.StyleDim, 0)
		cv.Put(x+7, y+1+i, c.String(), SuitStyle(c.Suit), 0)
	}
	cv.Put(x, y+1+len(t.Cards), taken, proto.StyleDim, 0)
}

// DrawTops draws cs as the tops of a held hand at x, y: two rows, each card TopW columns
// wide and touching the next, its index in its suit's style. It returns the x just after
// the last card's right edge.
//
//	.---.---.---.
//	|KS |10H|4D |
func DrawTops(cv *proto.Canvas, x, y int, cs []Card, frame proto.Style, a proto.Attr) int {
	for i, c := range cs {
		at := x + TopW*i
		cv.Put(at, y, ".---", frame, a)
		cv.Put(at, y+1, "|", frame, a)
		cv.Put(at+1, y+1, c.String(), SuitStyle(c.Suit), a)
	}
	end := x + TopW*len(cs)
	if len(cs) > 0 {
		cv.Put(end, y, ".", frame, a)
		cv.Put(end, y+1, "|", frame, a)
		end++
	}
	return end
}
