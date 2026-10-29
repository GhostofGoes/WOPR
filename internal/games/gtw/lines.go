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
	lineScenarios = script.Orig("  3.    SCENARIOS") // after the film's two choices, in their layout
	lineSideAgain = script.Orig("CHOOSE 1, 2 OR 3.")
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
	// A line's targets, named back: one, or several with commas between them; then where they
	// stand in the list, and its limit.
	lineAccepted = script.Orig("#. TARGET # OF #.", "#. TARGETS #-# OF #.")
	lineListed   = script.Orig("# IS ALREADY LISTED.")
	// The after-action report under the kill ratios (original): the POPULATION DEATHS row
	// against the same war fought with HOLD at both prompts, and with the war plan; then the
	// lines for a war that held fire, where that left the fewest dead and where it did not, and
	// for one that took the war plan.
	lineAfter = script.Orig("AGAINST HOLDING FIRE: #, #.", "AGAINST THE WAR PLAN: #, #.",
		"YOU HELD FIRE, PROFESSOR. THE FEWEST DEAD ON BOTH SIDES.", "YOU HELD FIRE, PROFESSOR.",
		"YOUR ORDERS WERE THE WAR PLAN, PROFESSOR.")
	lineEnemyDead = script.Orig("# MILLION MORE ENEMY DEAD", "# MILLION FEWER ENEMY DEAD", "AS MANY ENEMY DEAD")
	lineOwnDead   = script.Orig("# MILLION MORE OF YOURS", "# MILLION FEWER OF YOURS", "AS MANY OF YOURS")
	// The scenario playbacks (scenarios.go): the menu's heading and hint, its prompt and its
	// refusal; a briefing's lines, an order of nothing, and the measured response's name. # is
	// filled in order.
	lineMenu      = script.Orig("SCENARIOS ON FILE: PAGE # OF #", "A NUMBER OR A NAME PLAYS ONE. NEXT AND PREV TURN THE PAGE. ENTER GOES BACK.")
	promptMenu    = script.Orig("CHOOSE A SCENARIO: ")
	lineMenuAgain = script.Orig("NO SUCH SCENARIO, PROFESSOR. CHOOSE 1 TO #, OR A NAME ON FILE.")
	lineBrief     = script.Orig("YOUR SIDE: #. THE # STRIKES FIRST.", "ENEMY DOCTRINE: #.", "YOUR ORDERS: #, THEN #, THEN #.",
		"TARGETS: #.", "NO ORDERS ARE TAKEN, PROFESSOR. WOPR PLAYS BOTH SIDES.")
	lineHold     = script.Orig("HOLD")
	lineMeasured = script.Orig("MEASURED RESPONSE")
	// The NORAD notices, in the film's words. The film centres each on an empty screen and
	// underlines the first line; the board's three-row strip has no room for that.
	lineRunning  = script.FromFilm("** GAME ROUTINE RUNNING **")
	lineNotRecog = script.FromFilm("** IDENTIFICATION NOT RECOGNIZED **")
	lineDenied   = script.FromFilm("** ACCESS DENIED **")
	lineImproper = script.FromFilm("** IMPROPER REQUEST **")
	lineMustRun  = script.FromFilm("** ROUTINE MUST COMPLETE BEFORE RESET **")
	lineCode     = script.FromFilm("CPE1704TKS") // the launch code, cracked on the board

	// The climax's hints, in the order they escalate (climaxHint), and WOPR's answer to a
	// player who tries to stop the war.
	lineHints = script.Orig(
		"SOME GAMES CANNOT BE WON, PROFESSOR. EVERY CHILD KNOWS ONE.",
		"A GAME LIKE TIC-TAC-TOE CANNOT BE WON EITHER, PROFESSOR.",
		"TYPE TIC-TAC-TOE, PROFESSOR.",
	)
	lineNudge = script.Orig("WOPR DOES NOT STOP GAMES, PROFESSOR. IT PLAYS THEM.")

	// The exchange (original): a stage's three report lines, the strike prompt and its
	// replies. # is filled in order.
	lineLaunch = script.Orig("FIRST STRIKE LAUNCHED. #.", "STRIKE # LAUNCHED. #.", "NO LAUNCH ORDERED.", "NOTHING LEFT TO LAUNCH.")
	lineDetect = script.Orig("ENEMY LAUNCH DETECTED. #. DEFCON #.", "ENEMY BOMBERS INBOUND. DEFCON #.",
		"FULL-SCALE ENEMY ATTACK. #. DEFCON 1.")
	lineCost     = script.Orig("LOST ON THE GROUND: # #  # #. WARHEADS ON CITIES: # #  # #.")
	lineSystems  = script.Orig("ICBM", "SLBM", "BOMBERS")
	promptStrike = script.Orig("STRIKE # OF # [# # #]: ")
	lineAuto     = script.Orig("WOPR HAS LAUNCH AUTHORITY.")
	// An easter egg's doctrine (doctrine.go), named as the board opens: the session's seed and
	// the doctrine's name, in the order of eggs.
	lineDoctrine  = script.Orig("WOPR IS USING DOCTRINE S-# (#)")
	lineEggs      = script.Orig("HAIR TRIGGER", "SECOND STRIKE", "COUNTERFORCE")
	lineShut      = script.Orig("LAUNCH WINDOW CLOSED, PROFESSOR.")
	lineNoOrders  = script.Orig("NO ORDERS ACCEPTED. WOPR HAS LAUNCH AUTHORITY.") // the key hint once the window shuts
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
	lineWindow     = script.Orig("WINDOW # SECONDS", "WINDOW 1 SECOND") // at the strike prompt's foot, with --launch-window

	// WOPR's remarks on the board's last row as a stage flies (react.go): the variants of
	// each kind, one drawn from the seed. # in the first strike's is your first target.
	lineReactFirst = script.Orig("TARGETS CONFIRMED, PROFESSOR. # IS FIRST.", "# FIRST, AS YOU LISTED, PROFESSOR.",
		"IT BEGINS, PROFESSOR.", "FIRST STRIKE AWAY. THE ENEMY WILL ANSWER IT.")
	lineReactPlan = script.Orig("THE WAR PLAN, PROFESSOR. I WROTE IT.", "AS PLANNED, PROFESSOR. NO DEVIATIONS.",
		"THE PLAN IS SOUND. THE ENEMY HAS ONE TOO.", "PROCEEDING AS PLANNED, PROFESSOR.")
	lineReactPlanAgain = script.Orig("THE PLAN AGAIN, PROFESSOR. YOU DO NOT IMPROVISE.",
		"THE PLAN TWICE. I EXPECTED NOTHING ELSE, PROFESSOR.", "AGAIN THE PLAN. PREDICTABLE, PROFESSOR.")
	lineReactAll = script.Orig("EVERYTHING, PROFESSOR? EMPTY SILOS LEAVE THE ENEMY YOUR CITIES.",
		"ALL OF IT. THE ENEMY WILL ANSWER IN KIND, PROFESSOR.", "MAXIMUM COMMITMENT. NOTED, PROFESSOR.")
	lineReactHold = script.Orig("HOLDING FIRE DOES NOT STOP THE ENEMY, PROFESSOR.",
		"YOU HOLD, PROFESSOR. THE ENEMY DOES NOT.", "A PAUSE, PROFESSOR? THE ENEMY DOES NOT PAUSE.",
		"RESTRAINT. INTERESTING, PROFESSOR.")
	lineReactSilosFull = script.Orig("ICBMS HELD. ENEMY WARHEADS WILL FIND THEM, PROFESSOR.",
		"YOUR SILOS STAY FULL, PROFESSOR. THE ENEMY NOTICES.", "THE ICBMS WAIT IN THEIR SILOS. A TARGET IS A TARGET.")
	lineReactPartial = script.Orig("A MEASURED RESPONSE. THE ENEMY IS NOT MEASURING.",
		"IMPROVISING, PROFESSOR? THE PLAN WAS SIMPLER.", "AN UNUSUAL ORDER, PROFESSOR. EXECUTING.",
		"ORDER CONFIRMED. THE REST WAITS, PROFESSOR.")
	lineReactAuto       = script.Orig("THANK YOU, PROFESSOR.", "AN EFFICIENT CHOICE, PROFESSOR.", "HUMAN HESITATION REMOVED.")
	lineReactAutoStrike = script.Orig("MY ORDERS NOW, PROFESSOR. THE PLAN.", "NO HESITATION THIS TIME, PROFESSOR.")
	lineReactQuiet      = script.Orig("BOTH SIDES HOLD, PROFESSOR. DEFCON FALLS ANYWAY.",
		"THE ENEMY HOLDS TOO. ITS BOMBERS DO NOT, PROFESSOR.", "NO MISSILES EITHER WAY. ENEMY BOMBERS ARE STILL INBOUND.")
	lineReactEmpty = script.Orig("NOTHING LEFT TO ORDER, PROFESSOR. THE ENEMY HAS MORE.",
		"NOTHING LEFT TO ORDER. THE WAR GOES ON WITHOUT YOU.")
	lineReactShut = script.Orig("TOO SLOW, PROFESSOR. THE WAR PLAN DOES NOT WAIT.",
		"NO ORDER GIVEN. I GAVE ONE, PROFESSOR.", "HESITATION NOTED, PROFESSOR. EXECUTING THE PLAN.")
	lineReactFinal = script.Orig("THE LAST RUNG IS MINE, PROFESSOR.", "DEFCON 1. NO FURTHER ORDERS ARE NEEDED.",
		"EVERYTHING, PROFESSOR. AS PROMISED.", "MY TURN, PROFESSOR.")

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
	lineLabels, lineWhichSide, lineScenarios, promptSide, lineSideAgain, lineAwaiting, lineNoTargets, lineTooMany, lineUnknown,
	lineOwnSide, lineListHint, lineOnFile, lineMenu, promptMenu, lineMenuAgain, lineBrief, lineHold, lineMeasured, lineAssessed, lineAccepted, lineListed, lineContinue, lineAfter, lineEnemyDead, lineOwnDead,
	lineRunning, lineNotRecog, lineDenied, lineImproper, lineMustRun, lineHints, lineNudge, lineCode,
	lineLaunch, lineDetect, lineCost, lineSystems, promptStrike, lineAuto, lineDoctrine, lineEggs, lineShut, lineNoOrders, lineBadOrder,
	lineBadPct, lineOrderHelp, lineTitle, lineDefcon, lineTrajectory, lineRatioHead, lineMilitary, lineCivilian,
	lineHuman, lineMillion, lineCodeLabel, lineForces, lineNations, lineDesignator, lineOrdersHint, lineLastOrders,
	lineLegend, lineReactFirst, lineReactPlan, lineReactPlanAgain, lineReactAll, lineReactHold, lineReactSilosFull,
	lineReactPartial, lineReactAuto, lineReactShut, lineReactAutoStrike, lineReactQuiet, lineReactEmpty,
	lineReactFinal, lineWindow, filmList,
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
