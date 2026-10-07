package hearts

import (
	"strings"

	"github.com/GhostofGoes/WOPR/internal/script"
)

// Script text, all original. A # is filled in when the line is shown.
var (
	lineRules = script.Orig(
		"HEARTS. YOU ARE SOUTH; WOPR PLAYS WEST, NORTH AND EAST.",
		"EACH HEART IS A POINT AND THE QUEEN OF SPADES 13. TAKE ALL 26 TO SHOOT THE MOON.",
		"LOWEST SCORE WINS WHEN SOMEONE REACHES 100. NAME CARDS AS 7H OR BY POSITION.",
	)
	promptPass     = script.Orig("PASS THREE CARDS TO #: ")
	promptPlay     = script.Orig("YOUR PLAY: ")
	promptNext     = script.Orig("NEXT HAND? ")
	lineHold       = script.Orig("NO PASSING THIS HAND.")
	lineReceived   = script.Orig("YOU RECEIVE #.")
	linePassHelp   = script.Orig("NAME THREE CARDS FROM YOUR HAND, SUCH AS QS AH 2D, OR THEIR POSITIONS.")
	linePlayHelp   = script.Orig("NAME A CARD FROM YOUR HAND, SUCH AS 7H, OR ITS POSITION.")
	lineNotHeld    = script.Orig("YOU DO NOT HOLD THAT CARD.")
	lineFollow     = script.Orig("YOU MUST FOLLOW SUIT: #.")
	lineLeadTwo    = script.Orig("THE 2C LEADS THE FIRST TRICK.")
	lineNotBroken  = script.Orig("HEARTS HAVE NOT BEEN BROKEN.")
	lineNoPoints   = script.Orig("NO POINTS ON THE FIRST TRICK.")
	lineFinish     = script.Orig("FINISH THE HAND FIRST.")
	lineLeads      = script.Orig("# LEADS #.")
	linePlays      = script.Orig("# PLAYS #.")
	lineTrick      = script.Orig("#. # TAKES IT.")
	lineYouTake    = script.Orig("#. YOU TAKE IT.")
	linePoints     = script.Orig("(#)")
	lineMoon       = script.Orig("# SHOOTS THE MOON.")
	lineYouMoon    = script.Orig("YOU SHOT THE MOON.")
	lineHandPoints = script.Orig("THIS HAND: #.")
	lineScores     = script.Orig("SCORES: #.")
	lineYou        = script.Orig("YOU")
	seatNames      = script.Orig("SOUTH", "WEST", "NORTH", "EAST") // by seat, as cards numbers them

	// Panel text.
	panelTitle  = script.Orig("HEARTS")
	panelHand   = script.Orig("YOUR HAND")
	panelLast   = script.Orig("LAST TRICK")
	panelTaken  = script.Orig("TAKEN BY #")
	panelBroken = script.Orig("HEARTS BROKEN")
	panelPass   = script.Orig("PASS #")
	panelDirs   = script.Orig("", "LEFT", "ACROSS", "RIGHT")
)

// Lines is every script block, for the provenance test.
var Lines = []script.Ls{
	lineRules, promptPass, promptPlay, promptNext, lineHold, lineReceived, linePassHelp, linePlayHelp, lineNotHeld,
	lineFollow, lineLeadTwo, lineNotBroken, lineNoPoints, lineFinish, lineLeads, linePlays, lineTrick, lineYouTake, linePoints, lineMoon, lineYouMoon,
	lineHandPoints, lineScores, lineYou, seatNames, panelTitle, panelHand, panelLast, panelTaken, panelBroken, panelPass, panelDirs, artTitle,
}

// fill replaces each # in l's first line with the next arg.
func fill(l script.Ls, args ...string) string {
	text := l[0].Text
	for _, a := range args {
		text = strings.Replace(text, "#", a, 1)
	}
	return text
}
