// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package scenes

import (
	"time"

	"github.com/GhostofGoes/WOPR/internal/script"
)

// firstContact: David's first connections. A number at LOGON: is refused and the line drops;
// after the re-dial, HELP LOGON, HELP GAMES, and LIST GAMES on an open line; then, from the
// library, two names at LOGON:, each refused. The persona answers each line the same way.
var firstContact = Scene{
	Slug:        "first-contact",
	Title:       "FIRST CONTACT",
	Blurb:       "LOGON ATTEMPTS, HELP AND THE LIST OF GAMES",
	Interactive: true,
	Steps: steps(
		dial(dialPause),
		[]Step{typedAt(logon, "000001")},
		refused(),
		[]Step{
			typedAt(logon, "Help Logon"),
			say(filmText("HELP NOT AVAILABLE"), blank),
			typedAt(logon, "Help Games"),
			say(filmText("'GAMES' REFERS TO MODELS, SIMULATIONS AND GAMES", "WHICH HAVE TACTICAL AND STRATEGIC APPLICATIONS."), blank),
			typed("List Games"),
			table(gamesList, blank),
			pause(2*time.Second, script.Original),
			typedAt(logon, "Falkens-Maze"),
		},
		refused(),
		[]Step{
			typedAt(logon, "Armageddon"),
			say(notRecognized),
			pause(2*time.Second, script.Original),
		},
	),
}

// joshua: the backdoor. The connect header and the status burst, the greeting, the small
// talk, and Global Thermonuclear War asked for twice, as the persona plays it.
var joshua = Scene{
	Slug:        "joshua",
	Title:       "JOSHUA",
	Blurb:       "THE BACKDOOR, THE GREETING AND A GAME OF CHOICE",
	Interactive: true,
	Steps: steps(
		dial(dialPause),
		[]Step{
			typedAt(logon, "Joshua"),
			pause(beat, script.Original),
			page(script.Reconstructed),
			table(script.Tag(script.ABS0, // the connect header, as abs0/wargames transcribes it (NOTICE.md)
				"#45     11456          11009          11893          11972        11315",
				"PRT CON. 3.4.5.  SECTRAN 9.4.3.                      PORT STAT: SD-345",
				"",
				"(311) 699-7305",
			)),
			pause(1500*time.Millisecond, script.Reconstructed),
			page(script.Reconstructed),
			table(recon( // the film's status phrases in the persona's layout
				"(311) 936-2364",
				"SYSPROC FUNCT READY                         ALT NET READY",
				"CPU AUTH RY-345-AX3                         SYSCOMP STATUS:  ALL PORTS ACTIVE",
			)),
			pause(1500*time.Millisecond, script.Reconstructed),
			page(script.Reconstructed),
			say(greetings, blank),
			typed("Hello."),
			say(filmText("HOW ARE YOU FEELING TODAY?"), blank),
			typed("I'm fine.  How are you?"),
			say(filmText("EXCELLENT.  IT'S BEEN A LONG TIME.  CAN YOU EXPLAIN", "THE REMOVAL OF YOUR USER ACCOUNT NUMBER ON 6/23/73?"), blank),
			typed("People sometimes make mistakes."),
			say(filmText("YES THEY DO.  SHALL WE PLAY A GAME?"), blank),
			typed("Love to.  How about Global Thermonuclear War?"),
			say(filmText("WOULDN'T YOU PREFER A GOOD GAME OF CHESS?"), blank),
			typed("Later.  Let's play Global Thermonuclear War."),
			say(filmText("FINE."), blank),
			pause(2*time.Second, script.Original),
		},
	),
}
