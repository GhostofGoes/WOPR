package bridge

import (
	"fmt"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/games/cards"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// Script text, all original. A # is filled in when the line is shown.
var (
	lineRules = script.Orig(
		"BRIDGE. WOPR BIDS ALL FOUR HANDS BY POINT COUNT.",
		"YOUR SIDE DECLARES: YOU PLAY YOUR HAND AND DUMMY'S, AND WOPR DEFENDS.",
		"NAME CARDS AS 7H. ONE DEAL IS A GAME: MAKE THE CONTRACT TO WIN.",
	)
	linePassedOut = script.Orig("PASSED OUT. WOPR DEALS AGAIN.")
	lineTurned    = script.Orig("WOPR TURNS THE TABLE SO THAT YOUR SIDE DECLARES.")
	lineAuction   = script.Orig("AUCTION, # DEALING: #")
	lineContract  = script.Orig("CONTRACT: # BY #. # LEADS.")
	promptHand    = script.Orig("# PLAYS: ")
	promptDummy   = script.Orig("DUMMY (#) PLAYS: ")
	lineNotHeld   = script.Orig("# DOES NOT HOLD THAT CARD.")
	lineFollow    = script.Orig("# MUST FOLLOW SUIT: #.")
	linePlayHelp  = script.Orig("NAME A CARD FROM #, SUCH AS 7H.")
	lineFinish    = script.Orig("FINISH THE HAND FIRST.")
	lineTrick     = script.Orig("#. # TAKES IT.")
	lineMade      = script.Orig("CONTRACT MADE WITH # TRICKS. SCORE #.")
	lineDown      = script.Orig("DOWN #. SCORE #.")
	callPass      = script.Orig("PASS")
	strainNames   = script.Orig("C", "D", "H", "S", "NT")

	// Panel text.
	panelTitle    = script.Orig("BRIDGE")
	panelContract = script.Orig("CONTRACT # BY #")
	panelTricks   = script.Orig("N-S #  E-W #")
	panelDummy    = script.Orig("(DUMMY)")
	panelLast     = script.Orig("LAST TRICK:")
	panelVoid     = script.Orig("-")
)

// Lines is every script block, for the provenance test.
var Lines = []script.Ls{
	lineRules, linePassedOut, lineTurned, lineAuction, lineContract, promptHand, promptDummy, lineNotHeld, lineFollow,
	linePlayHelp, lineFinish, lineTrick, lineMade, lineDown, callPass, strainNames, panelTitle, panelContract,
	panelTricks, panelDummy, panelLast, panelVoid,
}

// fill replaces each # in l's first line with the next arg.
func fill(l script.Ls, args ...string) string {
	text := l[0].Text
	for _, a := range args {
		text = strings.Replace(text, "#", a, 1)
	}
	return text
}

// String is the call as the auction shows it: 1S, 3NT, PASS.
func (b Bid) String() string {
	if b.Level == 0 {
		return callPass[0].Text
	}
	return fmt.Sprint(b.Level) + strainNames[b.Strain].Text
}

// bySuit shows a hand a suit at a time, spades first: S A K 4  H Q 10 9  D -  C 8 6 3 2.
func bySuit(hand []cards.Card) string {
	var parts []string
	for s := cards.Spades; ; s-- {
		var ranks []string
		for _, c := range hand {
			if c.Suit == s {
				ranks = append(ranks, c.Rank.String())
			}
		}
		if len(ranks) == 0 {
			ranks = []string{panelVoid[0].Text}
		}
		parts = append(parts, s.Letter()+" "+strings.Join(ranks, " "))
		if s == cards.Clubs {
			break
		}
	}
	return strings.Join(parts, "  ")
}
