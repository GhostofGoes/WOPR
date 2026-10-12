// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

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
		say(filmText("WHICH SIDE DO YOU WANT?", "", "  1.    UNITED STATES", "  2.    SOVIET UNION", "")),
		typedAt(filmText("PLEASE CHOOSE ONE:  "), "2"),
		pause(beat, script.Original),
		page(script.Reconstructed),
		say(filmText(
			"AWAITING FIRST STRIKE COMMAND",
			"-----------------------------",
			"",
			"PLEASE LIST PRIMARY TARGETS BY",
			"CITY AND/OR COUNTY NAME:",
			"",
		)),
		typed("Las Vegas"),
		typed("Seattle"),
		typedAs(script.Original, nil, ""), // the Enter that ends GTW's own target list
		pause(beat, script.Original),
		page(script.Reconstructed),
		Board{At: gtw.FilmStrike1, Play: true, Prov: script.Reconstructed},
		pause(1500*time.Millisecond, script.Original),
		typedAs(script.Original, orig("STRIKE 2 OF 3 [50 50 100]: "), ""), // Enter at GTW's own prompt
		Board{At: gtw.FilmStrike2, Play: true, Prov: script.Original},
		pause(4*time.Second, script.Original),
	},
}

// climax: DEFCON 1 at NORAD while WOPR searches for the launch code. LIST GAMES, a game it
// will not switch to, the war it is running, then tic-tac-toe: one player and a stalemate,
// then zero players, self-play, the montage, and the film's last words. The NORAD notices are
// GTW's own replies to the same lines (the consistency test), in the words the film shows:
// the board's three-row strip has no room for the film's rules and blank lines between them.
// Tic-tac-toe and the ending are the real programs, in movie mode. LIST GAMES is in capitals
// on the film's screen; the other names David enters there are in capitals as abs0's
// transcription of the scene has them (tic-tac-toe's ZERO too).
var climax = Scene{
	Slug:  "climax",
	Title: "CLIMAX",
	Blurb: "DEFCON 1, TIC-TAC-TOE AND A STRANGE GAME",
	Steps: []Step{
		Board{At: gtw.FilmClimax, Prov: script.Reconstructed},
		say(gameRunning),
		pause(3*time.Second, script.Original),
		typed("LIST GAMES"),
		Board{Prov: script.Original}, // sixteen lines do not fit under the board: GTW lists them in the console
		table(gamesList),
		pause(3*time.Second, script.Original),
		typedRecon("CHESS"),
		Board{At: gtw.FilmClimax, Prov: script.Original},
		say(filmText("** IDENTIFICATION NOT RECOGNIZED **", "** ACCESS DENIED **")),
		pause(2*time.Second, script.Original),
		typedRecon("GTW"),
		say(gameRunning),
		pause(3*time.Second, script.Original),
		typedRecon("TIC-TAC-TOE"),
		pause(beat, script.Original),
		Run{Slug: gtw.TicTacToeSlug, Mode: ending.MovieMode, Prov: script.Reconstructed},
		pause(4*time.Second, script.Original),
	},
}
