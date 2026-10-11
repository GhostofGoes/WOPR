package wopr

import "github.com/GhostofGoes/WOPR/internal/script"

// Script text uses the shared provenance types (docs/PLAN.md §2.1).
type (
	// L is one line of script text with its provenance.
	L = script.L
	// Ls is a block of lines.
	Ls = script.Ls
)

var (
	film  = script.FromFilm
	recon = script.Recon
	orig  = script.Orig
)

// Script text. Every block is listed in allLines for the provenance test.
var (
	lineDialing   = orig("CONNECTING...")
	lineConnected = orig("CONNECTED.")
	// The film's lines below are its screen text, character for character, from the sources in
	// docs/PLAN.md §2.3: two spaces after the colon and after each full stop, and the system's own
	// misspelling of IDENTIFICATION. Never change them (AGENTS.md, "Film text and provenance").
	lineLogon       = L{Text: "LOGON:  ", Prov: script.Film}
	lineNotRecog    = film("INDENTIFICATION NOT RECOGNIZED BY SYSTEM", "--CONNECTION TERMINATED--") //nolint:misspell // the film's own spelling
	lineHelpNA      = film("HELP NOT AVAILABLE")
	lineHelpGames   = film("'GAMES' REFERS TO MODELS, SIMULATIONS AND GAMES", "WHICH HAVE TACTICAL AND STRATEGIC APPLICATIONS.")
	lineLogonHint   = orig("** HINT: THE BACKDOOR IS THE NAME OF FALKEN'S SON **")
	lineGreetings   = film("GREETINGS PROFESSOR FALKEN.")
	lineHowFeeling  = film("HOW ARE YOU FEELING TODAY?")
	lineExcellent   = film("EXCELLENT.  IT'S BEEN A LONG TIME.  CAN YOU EXPLAIN", "THE REMOVAL OF YOUR USER ACCOUNT NUMBER ON 6/23/73?")
	lineFeelFine    = orig("EXCELLENT.") // the scripted brain's answer to HOW ARE YOU: the film's word, not one of its lines
	lineYesTheyDo   = film("YES THEY DO.  SHALL WE PLAY A GAME?")
	linePreferChess = film("WOULDN'T YOU PREFER A GOOD GAME OF CHESS?")
	lineFine        = film("FINE.")
	lineShallWe     = orig("SHALL WE PLAY A GAME?") // the film's words, spoken and within lineYesTheyDo; alone, this project's
	lineNiceChess   = film("HOW ABOUT A NICE GAME OF CHESS?")
	lineImproper    = film("** IMPROPER REQUEST **")
	lineWhatsDiff   = film("WHAT'S THE DIFFERENCE?")
	lineProgrammed  = film("YOU SHOULD KNOW, PROFESSOR.  YOU", "PROGRAMMED ME.")
	lineToWin       = film("TO WIN THE GAME.")

	// The backdoor connect header, as transcribed by abs0/wargames (BSD-2, see NOTICE.md).
	lineHeader = script.Tag(script.ABS0,
		"#45     11456          11009          11893          11972        11315",
		"PRT CON. 3.4.5.  SECTRAN 9.4.3.                      PORT STAT: SD-345",
		"",
		"(311) 699-7305",
	)
	// The status burst after the header: the film's phrases in our own two-column layout. The
	// screens flash by and the transcriptions read them differently (docs/PLAN.md §2.3), so the
	// lines stay reconstructed.
	lineBurst = recon(
		"(311) 936-2364",
		"SYSPROC FUNCT READY                         ALT NET READY",
		"CPU AUTH RY-345-AX3                         SYSCOMP STATUS:  ALL PORTS ACTIVE",
	)

	lineWhichGame    = orig("WHICH GAME?")
	lineNotAvail     = orig("** GAME ROUTINE NOT AVAILABLE **")
	lineNotYet       = orig("THAT GAME IS NOT AVAILABLE YET, PROFESSOR.")
	lineNoSuchGame   = orig("NO SUCH GAME IN MEMORY, PROFESSOR. TYPE LIST GAMES.")
	lineAmbiguous    = orig("THAT NAME FITS MORE THAN ONE GAME:")
	lineCancelled    = orig("** REQUEST CANCELLED **")
	lineAnother      = orig("SHALL WE PLAY ANOTHER GAME?")
	lineAborted      = orig("GAME TERMINATED BEFORE COMPLETION.")
	lineWarAbandoned = orig("THE WAR WAS ABANDONED, PROFESSOR. NO WINNER CAN BE DECLARED.")
	lineRestate      = orig("PLEASE RESTATE YOUR REQUEST, PROFESSOR.")
	lineDeclined     = orig("AS YOU WISH, PROFESSOR.")
	lineIAmWOPR      = orig("I AM THE WOPR. WAR OPERATION PLAN RESPONSE.")
	lineMyName       = orig("THAT IS MY NAME, PROFESSOR.")
	lineWinUser      = orig("WINNER: PROFESSOR FALKEN")
	lineWinWOPR      = orig("WINNER: WOPR")
	lineWinNone      = orig("WINNER: NONE") // the montage's two columns, as one line of this project's
	lineHelpShell    = orig(
		"COMMANDS AVAILABLE:",
		"",
		"  HELP GAMES     WHAT GAMES ARE",
		"  LIST GAMES     THE GAMES IN MEMORY",
		"  <NUMBER>       PICK FROM THE LIST JUST SHOWN",
		"  PLAY <GAME>    START A GAME",
		"  LOGOFF         END THIS SESSION",
	)
)

// allLines is every script block, for the provenance test.
var allLines = []Ls{
	lineDialing, lineConnected,
	{lineLogon},
	lineNotRecog, lineHelpNA, lineHelpGames, lineLogonHint,
	lineGreetings, lineHowFeeling, lineExcellent, lineFeelFine, lineYesTheyDo, linePreferChess, lineFine, lineShallWe,
	lineNiceChess, lineImproper, lineWhatsDiff, lineProgrammed, lineToWin, lineHeader, lineBurst,
	lineWhichGame, lineNotAvail, lineNotYet, lineNoSuchGame, lineAmbiguous, lineCancelled, lineAnother, lineAborted, lineWarAbandoned,
	lineRestate, lineDeclined, lineIAmWOPR, lineMyName, lineHelpShell, lineWinUser, lineWinWOPR, lineWinNone,
}
