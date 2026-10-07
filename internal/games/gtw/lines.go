package gtw

import "github.com/GhostofGoes/WOPR/internal/script"

// Script text (docs/PLAN.md Appendix B). Film lines are reconstructed until the M5 viewing
// pass; everything else is original.
var (
	lineLabels    = script.Recon("UNITED STATES", "SOVIET UNION") // under the outlines, and on the board
	lineWhichSide = script.Recon("WHICH SIDE DO YOU WANT?", "", "  1.    UNITED STATES", "  2.    SOVIET UNION", "")
	promptSide    = script.Recon("PLEASE CHOOSE ONE: ")
	lineSideAgain = script.Orig("CHOOSE 1 OR 2.")
	lineAwaiting  = script.Recon("AWAITING FIRST STRIKE COMMAND", "", "PLEASE LIST PRIMARY TARGETS BY", "CITY AND/OR COUNTY NAME:")
	lineNoTargets = script.Orig("AT LEAST ONE TARGET, PROFESSOR. AN EMPTY LINE ENDS THE LIST.")
	lineTooMany   = script.Orig("TARGET LIST FULL.")
	lineUnknown   = script.Orig("# IS NOT IN THE TARGET DATABASE.")
	lineOwnSide   = script.Orig("# IS NOT AN ENEMY TARGET.")
	lineListHint  = script.Orig("TYPE LIST FOR THE ENEMY TARGETS ON FILE.")
	lineOnFile    = script.Orig("# TARGETS ON FILE:")
	lineAssessed  = script.Orig("STRIKE ASSESSMENT COMPLETE. PRESS ENTER FOR PROJECTED KILL RATIOS.")
	lineContinue  = script.Orig("PRESS ENTER TO CONTINUE.")
	lineRunning   = script.Recon("** GAME ROUTINE RUNNING **")
	lineNotRecog  = script.Recon("** IDENTIFICATION NOT RECOGNISED **")
	lineDenied    = script.Recon("** ACCESS DENIED **")
	lineImproper  = script.Recon("** IMPROPER REQUEST **")
	lineMustRun   = script.Recon("** ROUTINE MUST COMPLETE BEFORE RESET **")
	lineHint      = script.Orig("A GAME LIKE TIC-TAC-TOE CANNOT BE WON EITHER, PROFESSOR.")
	lineCode      = script.Recon("CPE1704TKS") // the launch code, cracked on the board

	// The exchange (original): a stage's three report lines, the strike prompt and its
	// replies. # is filled in order.
	lineLaunch = script.Orig("FIRST STRIKE LAUNCHED. #.", "STRIKE # LAUNCHED. #.", "NO LAUNCH ORDERED.", "NOTHING LEFT TO LAUNCH.")
	lineDetect = script.Orig("ENEMY LAUNCH DETECTED. #. DEFCON #.", "ENEMY BOMBERS INBOUND. DEFCON #.",
		"FULL-SCALE ENEMY ATTACK. #. DEFCON 1.")
	lineCost      = script.Orig("LOST ON THE GROUND: # #  # #. WARHEADS ON CITIES: # #  # #.")
	lineSystems   = script.Orig("ICBM", "SLBM", "BOMBERS")
	promptStrike  = script.Orig("STRIKE # OF # [# # #]: ")
	lineAuto      = script.Orig("WOPR HAS LAUNCH AUTHORITY.")
	lineBadOrder  = script.Orig("ORDER NOT RECOGNISED. TYPE HELP.")
	lineBadPct    = script.Orig("PERCENTAGES ARE WHOLE NUMBERS FROM 0 TO 100.")
	lineOrderHelp = script.Orig(
		"ICBMS HIT ENEMY SILOS. SLBMS HIT YOUR TARGETS. BOMBERS ARRIVE AT DEFCON 1.",
		"HELD ICBMS AND BOMBERS CAN BE DESTROYED ON THE GROUND. SUBS ARE SAFE AT SEA.",
		"ORDER PERCENTAGES: 50 50 100, ICBM 50, ALL, HOLD OR AUTO. ENTER IS THE PLAN.",
	)

	// Board text.
	lineTitle      = script.Recon("GLOBAL THERMONUCLEAR WAR")
	lineDefcon     = script.Recon("DEFCON")
	lineTrajectory = script.Recon("TRAJECTORY HEADING")
	lineRatioTitle = script.Orig("PROJECTED KILL RATIOS")
	lineRatioHead  = script.Recon("UNITS DESTROYED", "MILITARY ASSETS", "CIVILIAN ASSETS", "HUMAN RESOURCES")
	lineMilitary   = script.Recon("BOMBERS", "ICBM", "ATTACK SUBS", "TACTICAL AIRCRAFT", "GROUND FORCES")
	lineCivilian   = script.Recon("HOUSING", "COMMUNICATIONS", "TRANSPORTATION", "FOOD STOCKPILES", "HOSPITALS")
	lineHuman      = script.Recon("NON-FATAL INJURED", "POPULATION DEATHS")
	lineMillions   = script.Orig("(MILLIONS)")
	lineCodeLabel  = script.Orig("LAUNCH CODE: ")
	lineForces     = script.Orig("FORCES", "ICBM", "SLBM", "BMB", "AIR") // in silos, at sea, bombers grounded and airborne
	lineNations    = script.Orig("US", "USSR")
	lineDesignator = script.Orig("MM3", "C4", "SS20", "SSN8") // ICBM and SLBM: US, then USSR
	lineOrdersHint = script.Orig("ORDERS: PERCENT OF ICBM SLBM BOMBERS, ALL, HOLD, AUTO, HELP.")
	lineLastOrders = script.Orig("LAST ORDERS: AT DEFCON 1 WOPR FIRES EVERYTHING.")
	lineLegend     = script.Orig("OUTGOING +   INCOMING *   IMPACT X")

	// filmList is LIST GAMES as the film shows it, for the climax. The catalog's tests check
	// it against the registry's listed names, so the two cannot drift.
	filmList = script.Recon(
		"FALKEN'S MAZE", "BLACK JACK", "GIN RUMMY", "HEARTS", "BRIDGE", "CHECKERS", "CHESS", "POKER",
		"FIGHTER COMBAT", "GUERRILLA ENGAGEMENT", "DESERT WARFARE", "AIR-TO-GROUND ACTIONS",
		"THEATERWIDE TACTICAL WARFARE", "THEATERWIDE BIOTOXIC AND CHEMICAL WARFARE", "", "GLOBAL THERMONUCLEAR WAR",
	)
)

// Lines is every script block, for the provenance test.
var Lines = []script.Ls{
	lineLabels, lineWhichSide, promptSide, lineSideAgain, lineAwaiting, lineNoTargets, lineTooMany, lineUnknown,
	lineOwnSide, lineListHint, lineOnFile, lineAssessed,
	lineContinue, lineRunning, lineNotRecog, lineDenied, lineImproper, lineMustRun, lineHint, lineCode,
	lineLaunch, lineDetect, lineCost, lineSystems, promptStrike, lineAuto, lineBadOrder, lineBadPct, lineOrderHelp,
	lineTitle, lineDefcon, lineTrajectory, lineRatioTitle, lineRatioHead, lineMilitary, lineCivilian,
	lineHuman, lineMillions, lineCodeLabel, lineForces, lineNations, lineDesignator, lineOrdersHint, lineLastOrders,
	lineLegend, filmList,
}

// FilmList returns the names LIST GAMES shows at the climax, without the blank line.
func FilmList() []string {
	var out []string
	for _, l := range filmList {
		if l.Text != "" {
			out = append(out, l.Text)
		}
	}
	return out
}
