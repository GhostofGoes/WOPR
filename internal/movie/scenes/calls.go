package scenes

import (
	"time"

	"github.com/GhostofGoes/WOPR/internal/games/gtw"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// callBack: WOPR phones David at home to finish yesterday's game, and shows how long it has
// left. (The plan's provisional table put "Is this a game or is it real?" here; the
// transcriptions put it in the NORAD session, which is where it now is. M5 settles it.)
var callBack = Scene{
	Slug:  "call-back",
	Title: "CALL-BACK",
	Blurb: "WOPR CALLS BACK TO FINISH THE GAME",
	Steps: []Step{
		say(greetings, blank),
		pause(time.Second, script.Original),
		typed("Incorrect identification. I am not Falken."),
		typed("Falken is dead."),
		say(recon("I'M SORRY TO HEAR THAT, PROFESSOR."), blank),
		pause(time.Second, script.Reconstructed),
		say(recon("YESTERDAY'S GAME WAS INTERRUPTED."), blank),
		pause(time.Second, script.Reconstructed),
		say(recon("ALTHOUGH PRIMARY GOAL HAS NOT YET BEEN ACHIEVED, SOLUTION IS NEAR."), blank),
		typed("What is the primary goal?"),
		say(recon("YOU SHOULD KNOW PROFESSOR. YOU PROGRAMMED ME."), blank),
		typed("What is the primary goal?"),
		say(recon("TO WIN THE GAME."), blank),
		pause(2*time.Second, script.Original),
		page(script.Reconstructed),
		table(recon(
			"GAME TIME ELAPSED:           31 HRS 12 MIN 36 SEC",
			"ESTIMATED TIME REMAINING:    52 HRS 17 MIN 26 SEC",
		)),
		pause(4*time.Second, script.Original),
	},
}

// noradTerminal: David, held at NORAD, reaches Joshua again: the war goes on, the projected
// kill ratios, and the address that leads him to Falken.
var noradTerminal = Scene{
	Slug:  "norad-terminal",
	Title: "NORAD TERMINAL",
	Blurb: "JOSHUA AT NORAD: KILL RATIOS AND FALKEN'S ADDRESS",
	Steps: []Step{
		typedAt(logon, "Joshua"),
		pause(time.Second, script.Original),
		page(script.Reconstructed),
		say(greetings, blank),
		typed("Are you still playing the game?"),
		say(recon(
			"OF COURSE. I SHOULD REACH DEFCON 1 AND LAUNCH MY MISSILES IN 28 HOURS.",
			"WOULD YOU LIKE TO SEE SOME PROJECTED KILL RATIOS?",
		), blank),
		pause(1500*time.Millisecond, script.Original),
		Board{At: gtw.FilmRatios, Prov: script.Reconstructed},
		pause(8*time.Second, script.Original),
		Board{Prov: script.Original},
		typed("Is this a game or is it real?"),
		say(recon("WHAT'S THE DIFFERENCE?"), blank),
		pause(time.Second, script.Reconstructed),
		say(recon(
			"YOU ARE A HARD MAN TO REACH. COULD NOT FIND YOU",
			"IN SEATTLE AND NO TERMINAL IS IN OPERATION AT",
			"YOUR CLASSIFIED ADDRESS.",
		), blank),
		typed("What classified address?"),
		say(recon(
			"DOD PENSION FILES INDICATE CURRENT MAILING AS:",
			"",
			"DR. ROBERT HUME (A.K.A. STEPHEN W. FALKEN)",
			"5 TALL CEDAR ROAD",
			"GOOSE ISLAND, OREGON 97014",
		), blank),
		pause(4*time.Second, script.Original),
	},
}
