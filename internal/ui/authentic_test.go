// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package ui

import (
	"strings"
	"testing"
)

// --authentic reaches GTW: the film's two sides, a silent targets prompt, then the war, the
// kill ratios and the climax one after the other on the clock, with no prompt between them,
// even under --instant. The climax hands off to tic-tac-toe as in the film.
func TestAuthenticScreens(t *testing.T) {
	t.Parallel()
	opts := instant()
	opts.Play, opts.Panel, opts.Authentic = "global-thermonuclear-war", PanelOff, true
	d := newDriver(t, opts, 80, 24).settle()
	if s := d.screen(); !strings.Contains(s, "2.    SOVIET UNION") || strings.Contains(s, "SCENARIOS") {
		t.Fatalf("the side choice is not the film's:\n%s", s)
	}
	inspectScreen(t, "the film's side choice", d)
	d.line("2").line("Las Vegas")
	if s := d.screen(); strings.Contains(s, "TARGET 1 OF 8.") {
		t.Fatalf("the targets prompt names the target back:\n%s", s)
	}
	d.typ("").send(enter)
	d.until("the kill ratios", shows("POPULATION DEATHS"))
	if s := d.screen(); strings.Contains(s, "PRESS ENTER") || strings.Contains(s, "AGAINST") {
		t.Errorf("the kill ratios carry this project's lines:\n%s", s)
	}
	inspectScreen(t, "the film's kill ratios", d)
	d.until("the climax", shows("** GAME ROUTINE RUNNING **")).settle()
	inspectScreen(t, "the film's climax", d)
	d.line("TIC-TAC-TOE")
	if s := d.screen(); !strings.Contains(s, "PLEASE LIST NUMBER") {
		t.Fatalf("tic-tac-toe does not follow the climax:\n%s", s)
	}
}
