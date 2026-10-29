// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package ui

import (
	"strings"
	"testing"
)

// --no-climax reaches GTW: after the kill ratios, Enter asks for another war over the table
// instead of going to the climax, under the after-action report, which is still on screen.
// Yes clears the board away for the side choice, and no, at the second war's end, returns to
// WOPR with no winner.
func TestNoClimaxScreens(t *testing.T) {
	t.Parallel()
	opts := instant()
	opts.Play, opts.Panel, opts.NoClimax = "global-thermonuclear-war", PanelOff, true
	d := newDriver(t, opts, 80, 24).settle()
	war := func(side, target string) {
		t.Helper()
		d.line(side).line(target).line("").line("").line("").line("").line("")
		if s := d.screen(); !strings.Contains(s, "ANOTHER WAR, PROFESSOR?") || !strings.Contains(s, "POPULATION DEATHS") ||
			!strings.Contains(s, "AGAINST HOLDING FIRE: ") || !strings.Contains(s, "PROFESSOR.\nANOTHER WAR") ||
			strings.Contains(s, "PRESS ENTER TO CONTINUE.") || strings.Contains(s, "LAUNCH CODE") {
			t.Fatalf("the kill ratios do not offer another war:\n%s", s)
		}
		inspectScreen(t, "another war?", d)
	}
	war("2", "Las Vegas")
	d.line("yes")
	if s := d.screen(); !strings.Contains(s, "PLEASE CHOOSE ONE:") || strings.Contains(s, "POPULATION DEATHS") {
		t.Fatalf("yes does not return to the side choice:\n%s", s)
	}
	war("1", "Moscow")
	d.line("no")
	if s := d.screen(); !strings.Contains(s, "WINNER: NONE") || !strings.Contains(s, "SHALL WE PLAY ANOTHER GAME?") {
		t.Fatalf("no does not end the game with no winner:\n%s", s)
	}
	inspectScreen(t, "no more wars", d)
}
