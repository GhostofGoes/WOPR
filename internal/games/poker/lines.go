package poker

import (
	"strings"

	"github.com/GhostofGoes/WOPR/internal/games/cards"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// Script text, all original. A # is filled in when the line is shown.
var (
	lineRules = script.Orig(
		"FIVE-CARD DRAW, HEADS UP. EACH HAND COSTS AN ANTE OF 1.",
		"BETS ARE 5 BEFORE THE DRAW AND 10 AFTER; A BET AND THREE RAISES A ROUND.",
		"DISCARD UP TO THREE CARDS. TYPE LEAVE BETWEEN HANDS TO CASH OUT.",
	)
	lineChips      = script.Orig("YOU HAVE # CHIPS. WOPR HAS #.")
	lineWOPRDeals  = script.Orig("WOPR DEALS. ANTE 1 EACH.")
	lineYouDeal    = script.Orig("YOUR DEAL. ANTE 1 EACH.")
	lineYourHand   = script.Orig("YOUR HAND:")
	lineYouHad     = script.Orig("YOU SHOW:")
	lineWOPRShows  = script.Orig("WOPR SHOWS:")
	linePot        = script.Orig("POT #.")
	promptCheck    = script.Orig("CHECK OR BET? ")
	promptCall     = script.Orig("CALL #, RAISE OR FOLD? ")
	promptCallOnly = script.Orig("CALL # OR FOLD? ")
	promptDraw     = script.Orig("DISCARD (1-5, OR NONE): ")
	promptAgain    = script.Orig("ANOTHER HAND? ")
	lineActions    = script.Orig("C TO CHECK OR CALL, B TO BET OR RAISE, F TO FOLD.")
	lineNoCheck    = script.Orig("THERE IS A BET OF # TO CALL.")
	lineCapped     = script.Orig("NO MORE RAISES THIS ROUND.")
	lineFinish     = script.Orig("FINISH THE HAND FIRST.")
	lineTooMany    = script.Orig("YOU MAY DISCARD UP TO THREE CARDS.")
	lineWhichCards = script.Orig("NAME CARDS BY POSITION, 1 TO 5, OR AS THEY SHOW, SUCH AS 7H.")
	lineAgainHelp  = script.Orig("YES TO DEAL, LEAVE TO CASH OUT.")
	lineYouDraw    = script.Orig("YOU DRAW #.")
	lineYouPat     = script.Orig("YOU STAND PAT.")
	lineYouFold    = script.Orig("YOU FOLD.")
	lineWOPRChecks = script.Orig("WOPR CHECKS.")
	lineWOPRBets   = script.Orig("WOPR BETS #.")
	lineWOPRCalls  = script.Orig("WOPR CALLS #.")
	lineWOPRRaises = script.Orig("WOPR RAISES #.")
	lineWOPRFolds  = script.Orig("WOPR FOLDS.")
	lineWOPRDraws  = script.Orig("WOPR DRAWS #.")
	lineWOPRPat    = script.Orig("WOPR STANDS PAT.")
	lineYouWin     = script.Orig("YOU WIN THE POT OF #.")
	lineWOPRWins   = script.Orig("WOPR WINS THE POT OF #.")
	lineSplit      = script.Orig("SPLIT POT.")
	lineBroke      = script.Orig("YOU ARE OUT OF CHIPS.")
	lineWOPRBroke  = script.Orig("WOPR IS OUT OF CHIPS. THAT IS UNEXPECTED.")
	lineLeave      = script.Orig("YOU LEAVE WITH # CHIPS.")

	// Hand names.
	nameHigh     = script.Orig("# HIGH")
	namePair     = script.Orig("A PAIR OF #")
	nameTwoPair  = script.Orig("TWO PAIR, # AND #")
	nameTrips    = script.Orig("THREE #")
	nameStraight = script.Orig("A STRAIGHT, # HIGH")
	nameFlush    = script.Orig("A FLUSH, # HIGH")
	nameFull     = script.Orig("A FULL HOUSE, # OVER #")
	nameQuads    = script.Orig("FOUR #")
	nameSF       = script.Orig("A STRAIGHT FLUSH, # HIGH")
	nameRoyal    = script.Orig("A ROYAL FLUSH")
	rankOne      = script.Orig("", "", "TWO", "THREE", "FOUR", "FIVE", "SIX", "SEVEN", "EIGHT", "NINE", "TEN", "JACK", "QUEEN", "KING", "ACE")
	rankMany     = script.Orig("", "", "TWOS", "THREES", "FOURS", "FIVES", "SIXES", "SEVENS", "EIGHTS", "NINES", "TENS", "JACKS", "QUEENS", "KINGS", "ACES")
)

// Lines is every script block, for the provenance test.
var Lines = []script.Ls{
	lineRules, lineChips, lineWOPRDeals, lineYouDeal, lineYourHand, lineYouHad, lineWOPRShows, linePot, promptCheck,
	promptCall, promptCallOnly, promptDraw, promptAgain, lineActions, lineNoCheck, lineCapped, lineFinish, lineTooMany,
	lineWhichCards, lineAgainHelp, lineYouDraw, lineYouPat, lineYouFold, lineWOPRChecks, lineWOPRBets, lineWOPRCalls,
	lineWOPRRaises, lineWOPRFolds, lineWOPRDraws, lineWOPRPat, lineYouWin, lineWOPRWins, lineSplit, lineBroke,
	lineWOPRBroke, lineLeave, nameHigh, namePair, nameTwoPair, nameTrips, nameStraight, nameFlush, nameFull, nameQuads,
	nameSF, nameRoyal, rankOne, rankMany,
}

// fill replaces each # in l's first line with the next arg.
func fill(l script.Ls, args ...string) string {
	text := l[0].Text
	for _, a := range args {
		text = strings.Replace(text, "#", a, 1)
	}
	return text
}

func one(r cards.Rank) string  { return rankOne[r].Text }
func many(r cards.Rank) string { return rankMany[r].Text }

// Name is a hand's name as WOPR says it: A PAIR OF SEVENS, A FLUSH, KING HIGH.
func (s Score) Name() string {
	a, b := s.rank(0), s.rank(1)
	switch s.Category() {
	case OnePair:
		return fill(namePair, many(a))
	case TwoPair:
		return fill(nameTwoPair, many(a), many(b))
	case Trips:
		return fill(nameTrips, many(a))
	case Straight:
		return fill(nameStraight, one(a))
	case Flush:
		return fill(nameFlush, one(a))
	case FullHouse:
		return fill(nameFull, many(a), many(b))
	case Quads:
		return fill(nameQuads, many(a))
	case StraightFlush:
		if a == cards.Ace {
			return nameRoyal[0].Text
		}
		return fill(nameSF, one(a))
	}
	return fill(nameHigh, one(a))
}
