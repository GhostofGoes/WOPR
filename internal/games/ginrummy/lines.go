package ginrummy

import (
	"strings"

	"github.com/GhostofGoes/WOPR/internal/script"
)

// Script text, all original. A # is filled in when the line is shown.
var (
	lineRules = script.Orig(
		"GIN RUMMY TO 100. TEN CARDS EACH; ACES ARE LOW.",
		"DRAW FROM THE STOCK OR TAKE THE DISCARD, THEN DISCARD. KNOCK WITH 10 OR LESS",
		"DEADWOOD (KNOCK 7H DISCARDS THE 7H AND KNOCKS); GIN WITH NONE.",
	)
	lineWOPRDeals    = script.Orig("WOPR DEALS.")
	lineYouDeal      = script.Orig("YOUR DEAL. WOPR PLAYS FIRST.")
	promptDraw       = script.Orig("STOCK OR DISCARD? ")
	promptDiscard    = script.Orig("DISCARD: ")
	promptNext       = script.Orig("NEXT HAND? ")
	lineDrawHelp     = script.Orig("S TO DRAW FROM THE STOCK, D TO TAKE THE DISCARD.")
	lineNotTop       = script.Orig("THE DISCARD IS THE #.")
	lineDiscardHelp  = script.Orig("NAME A CARD IN YOUR HAND, OR ITS POSITION. ADD KNOCK TO KNOCK.")
	lineKnockHow     = script.Orig("KNOCK WITH THE CARD YOU DISCARD, SUCH AS KNOCK 7H.")
	lineNotBack      = script.Orig("YOU CANNOT DISCARD THE CARD YOU JUST TOOK.")
	lineTooMuch      = script.Orig("THAT LEAVES # DEADWOOD. A KNOCK NEEDS 10 OR LESS.")
	lineFinish       = script.Orig("FINISH THE HAND FIRST.")
	lineYouDrew      = script.Orig("YOU DRAW THE #.")
	lineYouTook      = script.Orig("YOU TAKE THE #.")
	lineYouDiscard   = script.Orig("YOU DISCARD THE #.")
	lineWOPRDrew     = script.Orig("WOPR DRAWS FROM THE STOCK.")
	lineWOPRTook     = script.Orig("WOPR TAKES THE #.")
	lineWOPRDiscards = script.Orig("WOPR DISCARDS THE #.")
	lineWOPRKnocks   = script.Orig("WOPR KNOCKS WITH # DEADWOOD.")
	lineWOPRGin      = script.Orig("WOPR GOES GIN.")
	lineYouKnock     = script.Orig("YOU KNOCK WITH # DEADWOOD.")
	lineYouGin       = script.Orig("GIN.")
	lineWOPRHand     = script.Orig("WOPR:")
	lineYourHand     = script.Orig("YOU:")
	lineYouLay       = script.Orig("YOU LAY OFF #.")
	lineWOPRLays     = script.Orig("WOPR LAYS OFF #.")
	lineUndercut     = script.Orig("UNDERCUT.")
	lineYouScore     = script.Orig("YOU SCORE #.")
	lineWOPRScores   = script.Orig("WOPR SCORES #.")
	lineScore        = script.Orig("SCORE: YOU #, WOPR #.")
	lineVoid         = script.Orig("THE STOCK IS DOWN TO TWO CARDS. NO SCORE THIS HAND.")
	lineLeave        = script.Orig("YOU LEAVE THE TABLE. #")

	// Panel text.
	panelTitle    = script.Orig("GIN RUMMY")
	panelScore    = script.Orig("YOU #   WOPR #   TO 100")
	panelWOPR     = script.Orig("WOPR: # CARDS")
	panelStock    = script.Orig("STOCK: #")
	panelDiscard  = script.Orig("DISCARD: #")
	panelHand     = script.Orig("YOUR HAND")
	panelDeadwood = script.Orig("DEADWOOD: #")
)

// Lines is every script block, for the provenance test.
var Lines = []script.Ls{
	lineRules, lineWOPRDeals, lineYouDeal, promptDraw, promptDiscard, promptNext, lineDrawHelp, lineNotTop,
	lineDiscardHelp, lineKnockHow, lineNotBack, lineTooMuch, lineFinish, lineYouDrew, lineYouTook, lineYouDiscard,
	lineWOPRDrew, lineWOPRTook, lineWOPRDiscards, lineWOPRKnocks, lineWOPRGin, lineYouKnock, lineYouGin,
	lineWOPRHand, lineYourHand, lineYouLay, lineWOPRLays, lineUndercut, lineYouScore, lineWOPRScores, lineScore,
	lineVoid, lineLeave, panelTitle, panelScore, panelWOPR, panelStock, panelDiscard, panelHand, panelDeadwood,
}

// fill replaces each # in l's first line with the next arg.
func fill(l script.Ls, args ...string) string {
	text := l[0].Text
	for _, a := range args {
		text = strings.Replace(text, "#", a, 1)
	}
	return text
}
