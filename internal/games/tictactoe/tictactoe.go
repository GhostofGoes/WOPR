// Package tictactoe is the game that ends the film: perfect minimax, so nobody can win
// against WOPR. In the "climax" launch mode it is the closed state machine of
// docs/PLAN.md §6.2; zero players hands off to the ending, which reuses Best and Draw for
// WOPR's self-play. In the "movie" mode the climax plays itself (docs/PLAN.md §7).
package tictactoe

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/prompt"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// Climax is the launch mode GTW uses at the film's climax.
const Climax = "climax"

// EndingSlug is the program zero players hands off to.
const EndingSlug = "ending"

// CodeMode prefixes the ending's launch mode with the launch code characters already
// cracked: "code:7".
const CodeMode = "code:"

// MovieMode is movie mode's launch mode, alone or with the launch code characters the board
// cracked ("movie:5"). The game plays the film's climax by itself: it types the film's
// answers (one player, a move for X, then zero players) and reads each once it is on screen,
// then hands off to the ending in the same mode, which types the film's last line.
const MovieMode = "movie"

// Script text (docs/PLAN.md Appendix B). Lines lists every block for the provenance test.
var (
	linePlayers      = script.Recon("ONE OR TWO PLAYERS?")
	promptPlayers    = script.Recon("PLEASE LIST NUMBER OF PLAYERS: ")
	lineStalemate    = script.Recon("STALEMATE. WANT TO PLAY AGAIN?")
	lineImproper     = script.Recon("** IMPROPER REQUEST **")
	lineWOPRWins     = script.Orig("WINNER: WOPR.", "WANT TO PLAY AGAIN?")
	linePickSquare   = script.Orig("CHOOSE AN EMPTY SQUARE, 1 TO 9.")
	promptMove       = script.Orig("YOUR MOVE: ")
	promptMoveX      = script.Orig("X TO MOVE: ")
	promptMoveO      = script.Orig("O TO MOVE: ")
	lineWOPRMove     = script.Orig("WOPR: ") // followed by the square WOPR took
	lineYourMarkIsX  = script.Orig("YOU ARE X. WOPR IS O.")
	lineHotseatRules = script.Orig("X MOVES FIRST.")

	// What the user types in the film's climax (movie mode); X's moves are the game's own.
	filmPlayers = script.User(script.Reconstructed, "1")
	filmZero    = script.User(script.Reconstructed, "ZERO")
)

// Lines is every script block, for the provenance test.
var Lines = []script.Ls{
	linePlayers, promptPlayers, lineStalemate, lineImproper, lineWOPRWins, linePickSquare, promptMove,
	promptMoveX, promptMoveO, lineWOPRMove, lineYourMarkIsX, lineHotseatRules, filmPlayers, filmZero,
	artTitle, artBigX, artBigO, artMidX, artMidO, artSmallX, artSmallO, panelTitle, panelYou, panelWOPR,
}

// Board is a tic-tac-toe position: squares 0..8, row by row; each is 0, 'X' or 'O'.
type Board [9]byte

var wins = [8][3]int{{0, 1, 2}, {3, 4, 5}, {6, 7, 8}, {0, 3, 6}, {1, 4, 7}, {2, 5, 8}, {0, 4, 8}, {2, 4, 6}}

// Winner returns 'X' or 'O' if that side has three in a row, or 0.
func (b Board) Winner() byte {
	for _, w := range wins {
		if m := b[w[0]]; m != 0 && m == b[w[1]] && m == b[w[2]] {
			return m
		}
	}
	return 0
}

// Full reports whether every square is taken.
func (b Board) Full() bool {
	for _, m := range b {
		if m == 0 {
			return false
		}
	}
	return true
}

// Over reports whether the game has ended.
func (b Board) Over() bool { return b.Winner() != 0 || b.Full() }

// Other returns the opponent's mark.
func Other(m byte) byte {
	if m == 'X' {
		return 'O'
	}
	return 'X'
}

// Optimal returns every square with the best minimax value for mark, in square order.
// Values prefer a faster win and a slower loss. It returns nil when the game is over.
func Optimal(b Board, mark byte) []int {
	if b.Over() {
		return nil
	}
	memo := map[Board]int{}
	best, moves := -100, []int(nil)
	for sq := range b {
		if b[sq] != 0 {
			continue
		}
		b[sq] = mark
		v := -negamax(b, Other(mark), 1, memo)
		b[sq] = 0
		switch {
		case v > best:
			best, moves = v, []int{sq}
		case v == best:
			moves = append(moves, sq)
		}
	}
	return moves
}

// Best returns the square mark should take: one of Optimal's moves, chosen with r so games
// vary while a seed reproduces them. It returns -1 when the game is over.
func Best(b Board, mark byte, r *rand.Rand) int {
	moves := Optimal(b, mark)
	if len(moves) == 0 {
		return -1
	}
	return moves[r.IntN(len(moves))]
}

// negamax is the value of b for the side to move, depth plies below the root: 10 − depth
// for a win, 0 for a draw. The board fixes whose move it is, so it is the memo key.
func negamax(b Board, toMove byte, depth int, memo map[Board]int) int {
	if v, ok := memo[b]; ok {
		return v
	}
	var v int
	switch {
	case b.Winner() != 0: // the side that just moved won
		v = -(10 - depth)
	case b.Full():
		v = 0
	default:
		v = -100
		for sq := range b {
			if b[sq] != 0 {
				continue
			}
			b[sq] = toMove
			v = max(v, -negamax(b, Other(toMove), depth+1, memo))
			b[sq] = 0
		}
	}
	memo[b] = v
	return v
}

type state uint8

const (
	askPlayers state = iota
	playing
	askAgain
)

// Game is tic-tac-toe as a proto.Program.
type Game struct {
	env      proto.Env
	climax   bool
	movie    bool   // movie mode: the game types the film's answers itself
	typing   string // the answer being typed, read when it is on screen (Drained)
	typed    uint64 // answers typed, for their keystroke streams
	state    state
	players  int
	b        Board
	toMove   byte
	last     int // the last square played, -1 for none
	thinking bool
	thinkSeq uint64
}

// New returns a game; the launch mode comes in Start.
func New() games.Game { return &Game{last: -1} }

// Start implements proto.Program.
func (g *Game) Start(env proto.Env) []proto.Output {
	g.env = env
	g.movie = env.Mode == MovieMode || strings.HasPrefix(env.Mode, MovieMode+":")
	g.climax = g.movie || env.Mode == Climax || strings.HasPrefix(env.Mode, Climax+":")
	return g.askPlayers()
}

func say(ls ...script.Ls) proto.Output {
	var lines []string
	for _, l := range ls {
		lines = append(lines, l.Texts()...)
	}
	return proto.Say{Lines: lines, Pace: proto.PaceSpeech}
}

// ask prompts for a line. In movie mode the game types the film's answer itself, and reads it
// once it is on screen.
func (g *Game) ask(prompt string) []proto.Output {
	if !g.movie {
		return []proto.Output{proto.Prompt{Text: prompt}}
	}
	g.typing = g.answer()
	g.typed++
	outs := proto.Typed(prompt, g.typing, proto.NewRand(g.env.Seed, proto.DomainMovie|g.typed))
	return append(outs, proto.Drain{})
}

// answer is the film's next line: one player, then X's moves, perfect so that the game ends
// in stalemate, then zero players.
func (g *Game) answer() string {
	switch g.state {
	case askPlayers:
		return filmPlayers[0].Text
	case playing:
		r := proto.NewRand(g.env.Seed, proto.DomainMovie|1<<32|g.typed)
		return fmt.Sprint(Best(g.b, g.toMove, r) + 1)
	case askAgain:
	}
	return filmZero[0].Text
}

func (g *Game) askPlayers() []proto.Output {
	g.state = askPlayers
	return append([]proto.Output{say(linePlayers)}, g.ask(promptPlayers[0].Text)...)
}

// Handle implements proto.Program.
func (g *Game) Handle(ev proto.Event) []proto.Output {
	switch ev := ev.(type) {
	case proto.LineEvent:
		switch g.state {
		case askPlayers:
			return g.onPlayers(ev.Text)
		case playing:
			return g.onMove(ev.Text)
		case askAgain:
			return g.onAgain(ev.Text)
		}
	case proto.ThinkDone:
		return g.onWOPRMove(ev)
	case proto.Drained:
		if line := g.typing; g.movie && line != "" {
			g.typing = ""
			return g.Handle(proto.LineEvent{Text: line})
		}
	}
	return nil
}

func zero(input string) bool {
	n, ok := prompt.Number(input)
	return ok && n == 0
}

// toEnding hands off to the ending. At the climax, GTW's launch mode carries how much of the
// launch code it cracked ("climax:5"); the ending carries on from there.
func (g *Game) toEnding() []proto.Output {
	next := proto.Launch{Slug: EndingSlug}
	_, code, ok := strings.Cut(g.env.Mode, ":")
	switch {
	case g.movie:
		next.Mode = g.env.Mode // the ending plays the film's last line itself
	case ok && g.climax:
		next.Mode = CodeMode + code
	}
	return []proto.Output{proto.Done{Result: proto.Result{Outcome: proto.NoWinner, Next: &next}}}
}

func (g *Game) onPlayers(input string) []proto.Output {
	n, ok := prompt.Number(input)
	switch {
	case ok && n == 0:
		return g.toEnding()
	case ok && n == 1:
		g.players = 1
		return append([]proto.Output{say(lineYourMarkIsX)}, g.newGame()...)
	case ok && n == 2 && !g.climax:
		g.players = 2
		return append([]proto.Output{say(lineHotseatRules)}, g.newGame()...)
	}
	return append([]proto.Output{say(lineImproper)}, g.askPlayers()...)
}

func (g *Game) newGame() []proto.Output {
	g.state, g.b, g.toMove, g.last = playing, Board{}, 'X', -1
	return append([]proto.Output{proto.Redraw{}}, g.movePrompt()...)
}

func (g *Game) movePrompt() []proto.Output {
	switch {
	case g.players == 1:
		return g.ask(promptMove[0].Text)
	case g.toMove == 'X':
		return g.ask(promptMoveX[0].Text)
	default:
		return g.ask(promptMoveO[0].Text)
	}
}

// parseSquare reads a square: 1-9, or a column letter and row number (A1 is top left).
func parseSquare(input string) (int, bool) {
	if n, ok := prompt.Number(input); ok {
		return n - 1, n >= 1 && n <= 9
	}
	s := strings.ToUpper(strings.TrimSpace(input))
	if len(s) == 2 && s[0] >= 'A' && s[0] <= 'C' && s[1] >= '1' && s[1] <= '3' {
		return int(s[1]-'1')*3 + int(s[0]-'A'), true
	}
	return 0, false
}

func (g *Game) onMove(input string) []proto.Output {
	if g.thinking {
		return nil // no Prompt is pending while WOPR thinks; nothing should arrive
	}
	sq, ok := parseSquare(input)
	if !ok || g.b[sq] != 0 {
		return append([]proto.Output{say(linePickSquare)}, g.movePrompt()...)
	}
	g.play(sq)
	if g.b.Over() {
		return g.gameOver()
	}
	if g.players == 2 {
		return append([]proto.Output{proto.Redraw{}}, g.movePrompt()...)
	}
	return g.think()
}

func (g *Game) play(sq int) {
	g.b[sq] = g.toMove
	g.last = sq
	g.toMove = Other(g.toMove)
}

// think asks for WOPR's move off the UI goroutine. Fn captures the board by value and
// its own seed.
func (g *Game) think() []proto.Output {
	g.thinking = true
	g.thinkSeq++
	b, mark := g.b, g.toMove
	seed, stream := g.env.Seed, proto.DomainAI|g.thinkSeq
	return []proto.Output{proto.Redraw{}, proto.Think{
		Fn: func(context.Context) (any, error) {
			return Best(b, mark, proto.NewRand(seed, stream)), nil
		},
		Limit:  proto.Limit{MaxDepth: 9},
		Budget: time.Second,
	}}
}

func (g *Game) onWOPRMove(done proto.ThinkDone) []proto.Output {
	if !g.thinking {
		return nil
	}
	g.thinking = false
	sq, ok := done.Value.(int)
	if done.Err != nil || !ok || sq < 0 || sq > 8 || g.b[sq] != 0 {
		sq = Best(g.b, g.toMove, proto.NewRand(g.env.Seed, proto.DomainAI)) // never trust a bad result
	}
	g.play(sq)
	outs := []proto.Output{proto.Redraw{}, proto.Say{Lines: []string{lineWOPRMove[0].Text + fmt.Sprint(sq+1)}, Pace: proto.PaceSpeech}}
	if g.b.Over() {
		return append(outs, g.gameOver()...)
	}
	return append(outs, g.movePrompt()...)
}

// gameOver ends a game. Normal mode reports to the persona with the final board; the
// climax asks to play again, as in the film.
func (g *Game) gameOver() []proto.Output {
	w := g.b.Winner()
	if !g.climax {
		res := proto.Result{Outcome: proto.Draw, Lines: g.b.Text(BigRows)}
		switch {
		case w == 0:
		case g.players == 2 || w == 'X':
			res.Outcome = proto.Win // a human won (either player, with two)
		default:
			res.Outcome = proto.Loss
		}
		return []proto.Output{proto.Done{Result: res}}
	}
	g.state = askAgain
	if w == 'O' {
		return append([]proto.Output{say(lineWOPRWins)}, g.ask("")...)
	}
	return append([]proto.Output{say(lineStalemate)}, g.ask("")...)
}

func (g *Game) onAgain(input string) []proto.Output {
	if zero(input) {
		return g.toEnding()
	}
	switch prompt.YesNo(input) {
	case prompt.Yes:
		return g.newGame()
	case prompt.No:
		return g.askPlayers()
	case prompt.Unclear:
	}
	return append([]proto.Output{say(lineImproper)}, g.ask("")...)
}
