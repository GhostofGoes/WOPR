// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package ui

import (
	"strings"
	"testing"
	"time"
)

// GTW's launch window (--launch-window) on the 80x24 screen, paced: the strike prompt shows
// the seconds left at the board's foot and counts them down. When the window shuts, the key
// hint takes the prompt's place while WOPR's strikes fly, and the assessment's prompt follows,
// without the order that was still being typed. Every one of these screens passes the
// accessibility checks.
func TestLaunchWindowScreens(t *testing.T) {
	t.Parallel()
	opts := instant()
	opts.Instant = false
	opts.Play, opts.Panel, opts.LaunchWindow = "global-thermonuclear-war", PanelOff, 5*time.Second
	d := newDriver(t, opts, 80, 24).settle()
	d.line("2").line("Las Vegas").send(enter)
	until := func(what string) {
		t.Helper()
		for i := 0; !strings.Contains(d.screen(), what); i++ {
			if i > maxSteps || !d.step() {
				t.Fatalf("never shows %q:\n%s", what, d.screen())
			}
		}
	}
	until("STRIKE 2 OF 3 [50 50 100]:")
	if !strings.Contains(d.screen(), " WINDOW 5 SECONDS\n") || !strings.Contains(d.screen(), "STRIKE 2 OF 3 [50 50 100]:") {
		t.Fatalf("the strike prompt shows no launch window:\n%s", d.screen())
	}
	inspectScreen(t, "the window open", d)
	d.press("50 5") // an order still being typed when the window shuts
	until(" WINDOW 1 SECOND\n")
	inspectScreen(t, "the last second", d)
	until("NO ORDERS ACCEPTED, PROFESSOR.")
	if !d.m.keyMode || strings.Contains(d.screen(), "STRIKE 2 OF 3") {
		t.Errorf("the strike prompt stays after the window shut:\n%s", d.screen())
	}
	inspectScreen(t, "the window shut", d)
	until("PRESS ENTER FOR PROJECTED KILL RATIOS.") // the DEFCON 1 blink keeps the clock running
	d.steps(30)
	if d.m.keyMode || !d.m.asking {
		t.Errorf("the war does not reach the assessment:\n%s", d.screen())
	}
	if !d.m.ed.Empty() || strings.Contains(d.screen(), "50 5") {
		t.Errorf("the order typed for the closed window stays in the input line:\n%s", d.screen())
	}
}
