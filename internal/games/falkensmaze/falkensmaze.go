// Package falkensmaze is Falken's Maze: a maze under fog of war, walked with the arrow
// keys. WOPR watches which way the player turns at junctions and, every few moves, moves a
// wall the player has not seen yet: it closes the passage the player's habit would take
// and opens another, so the maze stays a perfect maze and the exit stays reachable from
// wherever the player stands (docs/PLAN.md §6.1, G-6).
package falkensmaze

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// Size and pace.
const (
	mazeW        = 19 // cells; drawn 3 columns and 2 rows a cell, so 58x11 characters
	mazeH        = 5
	rerouteEvery = 6  // moves between WOPR's re-routes
	maxReroutes  = 12 // then the walls stay put: always choosing the longest way, WOPR could stall a player forever
	learnAfter   = 8  // junction choices before WOPR names the player's habit (if it has one)
)

// Relative turns, as the bias counts them.
const (
	turnLeft = iota
	turnStraight
	turnRight
)

// Script text, all original. A # is filled in when the line is shown.
var (
	lineRules   = script.Orig("FIND THE EXIT. WOPR IS WATCHING HOW YOU TURN.")
	lineHint    = script.Orig("ARROW KEYS OR WASD TO MOVE. Q GIVES UP.")
	lineMoved   = script.Orig("WOPR HAS MOVED A WALL.", "THE MAZE IS NOT WHERE YOU LEFT IT.", "PREDICTABLE, PROFESSOR.")
	lineBias    = script.Orig("YOU FAVOUR # TURNS, PROFESSOR. SO NOTED.")
	lineTurns   = script.Orig("LEFT", "STRAIGHT", "RIGHT")
	lineFound   = script.Orig("YOU FOUND THE EXIT IN # MOVES. WOPR MOVED # WALLS.", "YOU FOUND THE EXIT IN # MOVES. WOPR MOVED ONE WALL.")
	lineGiveUp  = script.Orig("RETREAT ACCEPTED. THE EXIT WAS # MOVES AWAY.", "RETREAT ACCEPTED. THE EXIT WAS ONE MOVE AWAY.")
	panelMoves  = script.Orig("MOVES #")
	panelWalls  = script.Orig("WALLS MOVED #")
	panelHabits = script.Orig("YOUR TURNS: # LEFT  # STRAIGHT  # RIGHT")
)

// Lines is every script block, for the provenance test.
var Lines = []script.Ls{
	lineRules, lineHint, lineMoved, lineBias, lineTurns, lineFound, lineGiveUp, panelMoves, panelWalls, panelHabits,
	frameTitle, frameMarks, artEscaped,
}

func fill(text string, args ...string) string {
	for _, a := range args {
		text = strings.Replace(text, "#", a, 1)
	}
	return text
}

// Game is Falken's Maze as a proto.Program.
type Game struct {
	rng      *rand.Rand
	m        *maze
	player   int
	exit     int
	heading  int
	seen     []bool
	moves    int
	reroutes int
	bias     [3]int
	remarked bool
	visited  []bool // the cells the player has stood on: their trail, shown once out
	won      bool
}

// New returns a game.
func New() games.Game { return &Game{} }

func say(lines ...string) proto.Output { return proto.Say{Lines: lines, Pace: proto.PaceSpeech} }

func table(lines ...string) proto.Output { return proto.Say{Lines: lines, Pace: proto.PaceTable} }

// Start implements proto.Program.
func (g *Game) Start(env proto.Env) []proto.Output {
	g.rng = proto.NewRand(env.Seed, 0)
	g.m = newMaze(mazeW, mazeH)
	g.m.generate(g.rng)
	g.exit = g.m.cells() - 1
	g.heading = east
	g.seen = make([]bool, g.m.cells())
	g.visited = make([]bool, g.m.cells())
	g.visited[g.player] = true
	g.reveal()
	return []proto.Output{
		proto.SetLayout{Layout: proto.LayoutPanel, PanelRows: PanelRows},
		say(lineRules[0].Text),
		proto.Redraw{},
		proto.AwaitKeys{Hint: lineHint[0].Text},
	}
}

// reveal shows the player's cell and the cells its passages lead to.
func (g *Game) reveal() {
	g.seen[g.player] = true
	for d := range 4 {
		if n, ok := g.m.neighbour(g.player, d); ok && g.m.open(g.player, d) {
			g.seen[n] = true
		}
	}
}

// Handle implements proto.Program.
func (g *Game) Handle(ev proto.Event) []proto.Output {
	k, ok := ev.(proto.KeyEvent)
	if !ok {
		return nil
	}
	d := -1
	switch {
	case k.Key == proto.KeyUp || k.Rune == 'w' || k.Rune == 'W' || k.Rune == 'k':
		d = north
	case k.Key == proto.KeyRight || k.Rune == 'd' || k.Rune == 'D' || k.Rune == 'l':
		d = east
	case k.Key == proto.KeyDown || k.Rune == 's' || k.Rune == 'S' || k.Rune == 'j':
		d = south
	case k.Key == proto.KeyLeft || k.Rune == 'a' || k.Rune == 'A' || k.Rune == 'h':
		d = west
	case k.Rune == 'q' || k.Rune == 'Q':
		left := len(g.m.path(g.player, g.exit)) - 1
		line := lineGiveUp[0]
		if left == 1 {
			line = lineGiveUp[1]
		}
		return []proto.Output{say(fill(line.Text, fmt.Sprint(left))), proto.Done{Result: proto.Result{Outcome: proto.Loss}}}
	}
	if d < 0 || !g.m.open(g.player, d) {
		return nil // a wall, or a key that means nothing here
	}
	return g.step(d)
}

func (g *Game) step(d int) []proto.Output {
	if t := relTurn(g.heading, d); g.m.exits(g.player) >= 3 && t >= 0 { // a junction: note the turn
		g.bias[t]++
	}
	g.player, _ = g.m.neighbour(g.player, d)
	g.heading = d
	g.moves++
	g.visited[g.player] = true
	g.reveal()
	if g.player == g.exit {
		g.won = true
		found := lineFound[0]
		if g.reroutes == 1 {
			found = lineFound[1]
		}
		text := fill(found.Text, fmt.Sprint(g.moves), fmt.Sprint(g.reroutes))
		return []proto.Output{proto.Redraw{}, table(artEscaped.Texts()...), say(text), proto.Done{Result: proto.Result{Outcome: proto.Win}}}
	}
	outs := []proto.Output{proto.Redraw{}}
	if g.moves%rerouteEvery == 0 && g.reroute() && g.reroutes%3 == 1 { // a remark now and then, not every time
		outs = append(outs, say(lineMoved[(g.reroutes/3)%len(lineMoved)].Text))
	}
	if total := g.bias[0] + g.bias[1] + g.bias[2]; !g.remarked && total >= learnAfter && 2*g.bias[g.favourite()] > total {
		g.remarked = true
		outs = append(outs, say(fill(lineBias[0].Text, lineTurns[g.favourite()].Text)))
	}
	return outs
}

// relTurn is the turn from heading in to heading out: left, straight or right, or -1 for
// going back.
func relTurn(in, out int) int {
	switch (out - in + 4) % 4 {
	case 0:
		return turnStraight
	case 1:
		return turnRight
	case 3:
		return turnLeft
	}
	return -1
}

// favourite is the turn the player takes most at junctions.
func (g *Game) favourite() int {
	best := turnStraight
	for t, n := range g.bias {
		if n > g.bias[best] {
			best = t
		}
	}
	return best
}

// reroute closes a passage on the way to the exit that the player has not seen, preferring
// a junction where the way on is the player's favourite turn, and opens another unseen wall
// that joins the two halves again, choosing the one that makes the way longest. It reports
// whether a wall moved.
func (g *Game) reroute() bool {
	if g.reroutes >= maxReroutes {
		return false
	}
	path := g.m.path(g.player, g.exit)
	type edge struct{ a, d int }
	var preferred, others []edge
	for i := 1; i+1 < len(path); i++ {
		a, b := path[i], path[i+1]
		if g.seen[a] || g.seen[b] {
			continue
		}
		out := g.m.dirTo(a, b)
		e := edge{a, out}
		if g.m.exits(a) >= 3 && relTurn(g.m.dirTo(path[i-1], a), out) == g.favourite() {
			preferred = append(preferred, e)
		} else {
			others = append(others, e)
		}
	}
	for _, e := range append(preferred, others...) {
		g.m.set(e.a, e.d, false)
		if g.openAnother(e.a, e.d) {
			g.reroutes++
			return true
		}
		g.m.set(e.a, e.d, true) // nothing unseen joins the halves: leave this passage be
	}
	return false
}

// openAnother opens an unseen wall between the player's half of the maze and the exit's,
// other than the one just closed, choosing the longest resulting way to the exit.
func (g *Game) openAnother(closedA, closedD int) bool {
	mine := g.m.component(g.player)
	bestLen, bestCells, bestDirs := -1, []int(nil), []int(nil)
	for c := range g.m.cells() {
		if !mine[c] || g.seen[c] {
			continue
		}
		for d := range 4 {
			n, ok := g.m.neighbour(c, d)
			if !ok || mine[n] || g.seen[n] || g.m.open(c, d) {
				continue
			}
			if c == closedA && d == closedD {
				continue
			}
			if back, _ := g.m.neighbour(closedA, closedD); n == closedA && c == back {
				continue // the same wall, seen from the other side
			}
			g.m.set(c, d, true)
			l := len(g.m.path(g.player, g.exit))
			g.m.set(c, d, false)
			switch {
			case l > bestLen:
				bestLen, bestCells, bestDirs = l, []int{c}, []int{d}
			case l == bestLen:
				bestCells, bestDirs = append(bestCells, c), append(bestDirs, d)
			}
		}
	}
	if bestLen < 0 {
		return false
	}
	i := g.rng.IntN(len(bestCells))
	g.m.set(bestCells[i], bestDirs[i], true)
	return true
}
