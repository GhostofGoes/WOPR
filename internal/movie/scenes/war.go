package scenes

import (
	"time"

	"github.com/GhostofGoes/WOPR/internal/games/ending"
	"github.com/GhostofGoes/WOPR/internal/games/gtw"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// firstStrike: Global Thermonuclear War. The side choice, the targets, and the big board: the
// first strike needs no order, and the scene goes on with an empty line at the strike prompt,
// which carries out WOPR's war plan. The console text is GTW's own (the consistency test), and
// the board is GTW's film renderer.
var firstStrike = Scene{
	Slug:  "first-strike",
	Title: "FIRST STRIKE",
	Blurb: "A SIDE, TWO TARGETS AND THE BIG BOARD",
	Steps: []Step{
		table(gtw.SideChoice(), blank),
		say(recon("WHICH SIDE DO YOU WANT?", "", "  1.    UNITED STATES", "  2.    SOVIET UNION", "")),
		typedAt(recon("PLEASE CHOOSE ONE: "), "2"),
		page(script.Reconstructed),
		say(recon("AWAITING FIRST STRIKE COMMAND", "", "PLEASE LIST PRIMARY TARGETS BY", "CITY AND/OR COUNTY NAME:")),
		typed("Las Vegas"),
		typed("Seattle"),
		typed(""),
		page(script.Reconstructed),
		Board{At: gtw.FilmStrike1, Play: true, Prov: script.Reconstructed},
		pause(1500*time.Millisecond, script.Original),
		typedAt(orig("STRIKE 2 OF 3 [50 50 100]: "), ""), // GTW's own prompt
		Board{At: gtw.FilmStrike2, Play: true, Prov: script.Original},
		pause(4*time.Second, script.Original),
	},
}

// climax: DEFCON 1 at NORAD while WOPR searches for the launch code. LIST GAMES, a game it
// will not switch to, the war it is running, then tic-tac-toe: one player and a stalemate,
// then zero players, self-play, the montage, and the film's last words. The NORAD notices are
// GTW's own replies to the same lines (the consistency test); tic-tac-toe and the ending are
// the real programs, in movie mode.
var climax = Scene{
	Slug:  "climax",
	Title: "CLIMAX",
	Blurb: "DEFCON 1, TIC-TAC-TOE AND A STRANGE GAME",
	Steps: []Step{
		Board{At: gtw.FilmClimax, Prov: script.Reconstructed},
		say(gameRunning),
		pause(3*time.Second, script.Original),
		typed("List Games"),
		Board{Prov: script.Original}, // sixteen lines do not fit under the board: GTW lists them in the console
		table(gamesList),
		pause(3*time.Second, script.Original),
		typed("CHESS"),
		Board{At: gtw.FilmClimax, Prov: script.Original},
		say(recon("** IDENTIFICATION NOT RECOGNISED **", "** ACCESS DENIED **")),
		pause(2*time.Second, script.Original),
		typed("GTW"),
		say(gameRunning),
		pause(3*time.Second, script.Original),
		typed("TIC-TAC-TOE"),
		Run{Slug: gtw.TicTacToeSlug, Mode: ending.MovieMode, Prov: script.Reconstructed},
		pause(4*time.Second, script.Original),
	},
}
