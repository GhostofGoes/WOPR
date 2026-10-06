package ui

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/golden"
)

// Air-to-Ground Actions keeps what the player plans with on the 80x24 screen, with and
// without the front panel. The start screen shows the whole title card, from the lead
// jet's nose to the name, above the PACKAGE: prompt. After each sortie of a short
// campaign, two of whose sorties destroy a target, the sortie's report from its first
// line is still on screen at the next prompt, and after a destroying sortie the map
// shows the hit. The screens with the panel are the golden.
func TestAirToGroundScreens(t *testing.T) {
	t.Parallel()
	const (
		nose = "                                        /\\"
		name = "---===[ A I R - T O - G R O U N D   A C T I O N S ]===---"
	)
	var s snapshots
	for _, panel := range []Panel{PanelOff, PanelOn} {
		opts := instant()
		opts.Play, opts.Seed, opts.Panel = "air-to-ground-actions", 42, panel
		d := newDriver(t, opts, 80, 24).settle()
		rows := strings.Split(d.screen(), "\n")
		if !slices.Contains(rows, nose) || !strings.Contains(d.screen(), name) || !atPrompt(d) {
			t.Errorf("panel %d: the start screen lacks the title card or the prompt:\n%s", panel, d.screen())
		}
		if panel == PanelOn {
			s.add("start", d)
		}
		hits := 0
		for n, plan := range []string{"bunker go", "target 5 strike 4 sead 3 escort 3 go", "target 2 strike 3 sead 2 go"} {
			d.line(plan)
			screen := d.screen()
			sortie := fmt.Sprintf("SORTIE %d: THE ", n+1)
			if !strings.Contains(screen, sortie) || !atPrompt(d) {
				t.Errorf("panel %d: after %q the report's first line or the prompt is off screen:\n%s", panel, plan, screen)
			}
			if strings.Contains(screen, "DIRECT HIT ON ") {
				hits++
				if !strings.Contains(screen, " IS DESTROYED. ") {
					t.Errorf("panel %d: after %q the map shows a hit the report does not:\n%s", panel, plan, screen)
				}
			}
			if panel == PanelOn {
				s.add(plan, d)
			}
		}
		if hits < 2 {
			t.Errorf("panel %d: %d sorties destroyed a target; the campaign needs two to test", panel, hits)
		}
	}
	golden.AssertString(t, "airtoground_screens", s.String())
}

// atPrompt reports whether the cursor is on the PACKAGE: prompt.
func atPrompt(d *driver) bool {
	v := d.m.View()
	rows := strings.Split(d.screen(), "\n")
	return v.Cursor != nil && v.Cursor.Y < len(rows) && strings.HasPrefix(rows[v.Cursor.Y], "PACKAGE:")
}
