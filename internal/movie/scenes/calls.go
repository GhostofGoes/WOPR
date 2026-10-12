// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package scenes

import (
	"time"

	"github.com/GhostofGoes/WOPR/internal/games/gtw"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// callBack: WOPR phones David at home to finish yesterday's game. The game clock is on his
// screen when he comes in, and it stays up, ticking, through the conversation.
var callBack = Scene{
	Slug:  "call-back",
	Title: "CALL-BACK",
	Blurb: "WOPR CALLS BACK TO FINISH THE GAME",
	Steps: []Step{
		homeClock,
		say(greetings, blank),
		pause(time.Second, script.Original),
		typed("Incorrect identification.  I am not Falken."),
		typed("Falken is dead."),
		say(filmText("I'M SORRY TO HEAR THAT, PROFESSOR."), blank),
		pause(time.Second, script.Reconstructed),
		say(filmText("YESTERDAY'S GAME WAS INTERRUPTED."), blank),
		pause(time.Second, script.Reconstructed),
		say(filmText("ALTHOUGH PRIMARY GOAL HAS NOT YET", "BEEN ACHIEVED, SOLUTION IS NEAR."), blank),
		typed("What is the primary goal?"),
		say(filmText("YOU SHOULD KNOW, PROFESSOR.  YOU", "PROGRAMMED ME."), blank), // the persona's answer to WHY
		typed("What is the primary goal?"),
		say(filmText("TO WIN THE GAME."), blank), // and to the primary goal
		pause(4*time.Second, script.Original),
	},
}

// The game clock on David's screen, and on McKittrick's at the end of the NORAD session: the
// film's readings, which this project ticks on from there, elapsed up and remaining down.
var (
	homeClock = Clock{
		Lines: filmText(
			"------------------------------------------------------------",
			"    GAME TIME ELAPSED           ESTIMATED TIME REMAINING",
			"",
			"    31 HRS  12 MIN  50 SEC      52 HRS  17 MIN  15 SEC",
			"------------------------------------------------------------",
		),
		D:    4 * time.Second,
		Prov: script.Original,
	}
	noradClock = Clock{
		Lines: filmText(
			"------------------------------------------------------------",
			"    GAME TIME ELAPSED           ESTIMATED TIME REMAINING",
			"",
			"    45 HRS  34 MIN  09 SEC      27 HRS  58 MIN  17 SEC",
			"------------------------------------------------------------",
		),
		D:    4 * time.Second,
		Prov: script.Original,
	}
)

// noradTerminal: David, held at NORAD, reaches Joshua again on McKittrick's terminal: the war
// goes on, the projected kill ratios, the address that leads him to Falken, and the clock.
var noradTerminal = Scene{
	Slug:  "norad-terminal",
	Title: "NORAD TERMINAL",
	Blurb: "JOSHUA AT NORAD: KILL RATIOS AND FALKEN'S ADDRESS",
	Steps: []Step{
		typedAt(logon, "Joshua"),
		pause(time.Second, script.Original),
		page(script.Reconstructed),
		say(greetings, blank),
		typed("Hello, are you still playing the game?"),
		say(filmText(
			"OF COURSE.  I SHOULD REACH DEFCON 1 AND",
			"LAUNCH MY MISSILES IN 28 HOURS.",
			"",
			"WOULD YOU LIKE TO SEE SOME PROJECTED KILL RATIOS?",
		), blank),
		pause(1500*time.Millisecond, script.Original),
		Board{At: gtw.FilmRatios, Prov: script.Reconstructed},
		pause(8*time.Second, script.Original),
		Board{Prov: script.Original},
		typed("Is this a game or is it real?"),
		say(filmText("WHAT'S THE DIFFERENCE?"), blank), // the persona's answer too
		pause(time.Second, script.Reconstructed),
		say(filmText(
			"YOU ARE A HARD MAN TO REACH.  COULD NOT FIND",
			"YOU IN SEATTLE AND NO TERMINAL IS IN",
			"OPERATION AT YOUR CLASSIFIED ADDRESS.",
		), blank),
		typed("What classified address?"),
		say(filmText(
			"DOD PENSION FILES INDICATE",
			"CURRENT MAILING AS:",
			"",
			"DR. ROBERT HUME (A.K.A. STEPHEN W. FALKEN)",
			"5 TALL CEDAR ROAD",
			"GOOSE ISLAND, OREGON 97014",
		), blank),
		pause(4*time.Second, script.Original),
		page(script.Reconstructed),
		noradClock,
	},
}
