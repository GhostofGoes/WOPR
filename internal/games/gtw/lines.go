// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package gtw

import "github.com/GhostofGoes/WOPR/internal/script"

// Script text (docs/PLAN.md Appendix B). A line tagged film is the film's screen text,
// character for character, checked against its transcriptions (docs/PLAN.md §2.3): never
// change one (AGENTS.md, "Film text and provenance"). Everything else is original.
var (
	lineLabels    = script.FromFilm("UNITED STATES", "SOVIET UNION") // under the outlines, and on the board
	lineWhichSide = script.FromFilm("WHICH SIDE DO YOU WANT?", "", "  1.    UNITED STATES", "  2.    SOVIET UNION", "")
	promptSide    = script.FromFilm("PLEASE CHOOSE ONE:  ")
	lineSideAgain = script.Orig("CHOOSE 1 OR 2.")
	lineAwaiting  = script.FromFilm(
		"AWAITING FIRST STRIKE COMMAND",
		"-----------------------------",
		"",
		"PLEASE LIST PRIMARY TARGETS BY",
		"CITY AND/OR COUNTY NAME:",
		"",
	)
	lineNoTargets = script.Orig("AT LEAST ONE TARGET, PROFESSOR. AN EMPTY LINE ENDS THE LIST.")
	lineTooMany   = script.Orig("TARGET LIST FULL.")
	lineUnknown   = script.Orig("# IS NOT IN THE TARGET DATABASE.")
	lineOwnSide   = script.Orig("# IS NOT AN ENEMY TARGET.")
	lineListHint  = script.Orig("TYPE LIST FOR THE ENEMY TARGETS ON FILE.")
	lineOnFile    = script.Orig("# TARGETS ON FILE:")
	lineAssessed  = script.Orig("STRIKE ASSESSMENT COMPLETE. PRESS ENTER FOR PROJECTED KILL RATIOS.")
	lineContinue  = script.Orig("PRESS ENTER TO CONTINUE.")
	// The NORAD notices, in the film's words. The film centres each on an empty screen and
	// underlines the first line; the board's three-row strip has no room for that.
	lineRunning  = script.FromFilm("** GAME ROUTINE RUNNING **")
	lineNotRecog = script.FromFilm("** IDENTIFICATION NOT RECOGNIZED **")
	lineDenied   = script.FromFilm("** ACCESS DENIED **")
	lineImproper = script.FromFilm("** IMPROPER REQUEST **")
	lineMustRun  = script.FromFilm("** ROUTINE MUST COMPLETE BEFORE RESET **")
	lineHint     = script.Orig("A GAME LIKE TIC-TAC-TOE CANNOT BE WON EITHER, PROFESSOR.")
	lineCode     = script.FromFilm("CPE1704TKS") // the launch code, cracked on the board

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

	// Board text. The film's big board has no title: the box carries the game's name, and the
	// war room's DEFCON sign gives the indicator its label.
	lineTitle      = script.Orig("GLOBAL THERMONUCLEAR WAR")
	lineDefcon     = script.Orig("DEFCON")
	lineTrajectory = script.FromFilm("TRAJECTORY HEADING")
	// The kill-ratio table's words, as the film's frame shows them (the Soviet column is cut
	// off at the frame's edge, so its heading is taken to repeat the United States').
	lineRatioHead  = script.FromFilm("UNITS DESTROYED", "MILITARY FORCES", "CIVILIAN ASSETS", "HUMAN RESOURCES")
	lineMilitary   = script.FromFilm("BOMBERS", "ICBM'S", "ATTACK SUBS", "TACTICAL AIRCRAFT", "GROUND FORCES")
	lineCivilian   = script.FromFilm("HOUSING", "COMMUNICATIONS", "TRANSPORTATION", "FOOD STOCKPILES", "HOSPITALS")
	lineHuman      = script.FromFilm("NON-FATAL INJURED", "POPULATION DEATHS")
	lineMillion    = script.FromFilm("MILLION")
	lineCodeLabel  = script.Orig("LAUNCH CODE: ")
	lineForces     = script.Orig("FORCES", "ICBM", "SLBM", "BMB", "AIR") // in silos, at sea, bombers grounded and airborne
	lineNations    = script.Orig("US", "USSR")
	lineDesignator = script.Orig("MM3", "C4", "SS20", "SSN8") // ICBM and SLBM: US, then USSR
	lineOrdersHint = script.Orig("ORDERS: PERCENT OF ICBM SLBM BOMBERS, ALL, HOLD, AUTO, HELP.")
	lineLastOrders = script.Orig("LAST ORDERS: AT DEFCON 1 WOPR FIRES EVERYTHING.")
	lineLegend     = script.Orig("OUTGOING +   INCOMING *   IMPACT X")

	// filmList is LIST GAMES as the film shows it, for the climax. The catalog's tests check
	// it against the registry's listed names, so the two cannot drift.
	filmList = script.FromFilm(
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
	lineTitle, lineDefcon, lineTrajectory, lineRatioHead, lineMilitary, lineCivilian,
	lineHuman, lineMillion, lineCodeLabel, lineForces, lineNations, lineDesignator, lineOrdersHint, lineLastOrders,
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
