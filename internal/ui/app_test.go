package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/catalog"
	"github.com/GhostofGoes/WOPR/internal/games/gamestest"
	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

func instant() Options {
	return Options{Instant: true, Seed: 1, SeedSet: true, Registry: catalog.Registry()}
}

func TestLogonJoshua(t *testing.T) {
	t.Parallel()
	var s snapshots
	d := newDriver(t, instant(), 80, 24).settle()
	s.add("dial and LOGON", d)
	d.line("Joshua")
	s.add("greeting", d)
	d.line("Hello.")
	s.add("small talk", d)
	if !strings.Contains(d.screen(), "HOW ARE YOU FEELING TODAY?") {
		t.Errorf("no reply to Hello:\n%s", d.screen())
	}
	golden.AssertString(t, "logon_joshua", s.String())
}

func TestLogoffExitsCleanly(t *testing.T) {
	t.Parallel()
	var s snapshots
	d := newDriver(t, instant(), 80, 24).settle().line("Joshua").line("LOGOFF")
	s.add("after LOGOFF", d)
	if d.ended != "quit" {
		t.Errorf("LOGOFF ended with %q, want quit", d.ended)
	}
	golden.AssertString(t, "logoff_exit0", s.String())
}

func TestCtrlCAndCtrlD(t *testing.T) {
	t.Parallel()
	if d := newDriver(t, instant(), 80, 24).settle().key(ctrlC); d.ended != "interrupt" {
		t.Errorf("Ctrl+C ended with %q, want interrupt", d.ended)
	}
	if d := newDriver(t, instant(), 80, 24).settle().typ("x").key(ctrlD); d.ended != "" {
		t.Errorf("Ctrl+D on a non-empty line must not quit (ended %q)", d.ended)
	}
	if d := newDriver(t, instant(), 80, 24).settle().key(ctrlD); d.ended != "quit" {
		t.Errorf("Ctrl+D on an empty line ended with %q, want quit", d.ended)
	}
}

// Shrinking below 80x24 freezes the session behind the TOO SMALL card; growing it resumes
// exactly where it stopped.
func TestTooSmallPausesAndResumes(t *testing.T) {
	t.Parallel()
	var s snapshots
	opts := instant()
	opts.Instant = false
	d := newDriver(t, opts, 80, 24).steps(12)
	before := d.screen()
	s.add("dialling, part revealed", d)

	d.resize(60, 20).settle()
	s.add("too small", d)
	d.typ("JOSHUA")
	if len(d.pending) != 0 {
		t.Errorf("the clock must stop while too small; %d ticks armed", len(d.pending))
	}

	d.resize(80, 24)
	if got := d.screen(); got != before {
		t.Errorf("resume changed the screen:\n%s\nwant:\n%s", got, before)
	}
	d.settle()
	s.add("resumed and settled", d)
	if !strings.Contains(d.screen(), "LOGON:") || strings.Contains(d.screen(), "JOSHUA") {
		t.Errorf("keys typed while too small must be dropped:\n%s", d.screen())
	}
	golden.AssertString(t, "too_small_pause_resume", s.String())
}

// The first key during a reveal skips it; Enter on an empty line is only a skip.
func TestKeySkipsReveal(t *testing.T) {
	t.Parallel()
	opts := instant()
	opts.Instant = false
	d := newDriver(t, opts, 80, 24).steps(3)
	if !d.m.tw.Revealing() {
		t.Fatal("expected text mid-reveal")
	}
	d.send(enter)
	if d.m.tw.Revealing() || d.m.ed.Value() != "" {
		t.Errorf("Enter must skip the reveal and type nothing; editor %q", d.m.ed.Value())
	}
	d.settle().send(enter).settle()
	if strings.Count(d.screen(), "LOGON:") != 2 {
		t.Errorf("Enter at the prompt submits an empty logon:\n%s", d.screen())
	}
}

// Typeahead: while WOPR thinks, keys go to the editor and Enter is held until the prompt
// returns; the thinking indicator shows meanwhile.
func TestTypeaheadWhileThinking(t *testing.T) {
	t.Parallel()
	var s snapshots
	d := newDriver(t, Options{Instant: true, Play: "stub", Registry: gamestest.Registry()}, 80, 24).settle()
	d.slowThinks = true
	d.press("THINK").send(enter)
	d.steps(40) // over a second of the indicator
	d.press("WIN").send(enter)
	s.add("thinking, with typeahead", d)
	if !d.m.held || d.m.ed.Value() != "WIN" || !strings.Contains(d.screen(), "PROCESSING") {
		t.Fatalf("Enter while thinking must be held; held %v, editor %q:\n%s", d.m.held, d.m.ed.Value(), d.screen())
	}
	d.finishThinks()
	s.add("thought, then the held line", d)
	if d.m.held || !strings.Contains(d.screen(), "WINNER: PROFESSOR FALKEN") {
		t.Errorf("the held line was not submitted:\n%s", d.screen())
	}
	golden.AssertString(t, "typeahead_thinking", s.String())
}

// Esc in a game asks for confirmation, then ends the game. The stub game uses the panel
// layout, so its board sits above the console.
func TestEscConfirmInGame(t *testing.T) {
	t.Parallel()
	var s snapshots
	d := newDriver(t, Options{Instant: true, Play: "stub", Registry: gamestest.Registry()}, 80, 24).settle()
	s.add("game started", d)
	d.send(esc) // no settle: the confirmation lasts host.EscWindow
	s.add("first Esc", d)
	if !strings.Contains(d.screen(), "PRESS ESC AGAIN") {
		t.Error("no confirmation notice")
	}
	d.send(esc).settle()
	s.add("second Esc", d)
	if !strings.Contains(d.screen(), "GAME TERMINATED BEFORE COMPLETION.") {
		t.Error("the game was not aborted")
	}
	golden.AssertString(t, "esc_confirm", s.String())
}

// The confirmation expires: after the window, one Esc only asks again.
func TestEscWindowExpires(t *testing.T) {
	t.Parallel()
	d := newDriver(t, Options{Instant: true, Play: "stub", Registry: gamestest.Registry()}, 80, 24).settle()
	d.key(esc) // settle runs the clock past the window
	if strings.Contains(d.screen(), "PRESS ESC AGAIN") {
		t.Error("the notice outlived its window")
	}
	d.send(esc)
	if !strings.Contains(d.screen(), "PRESS ESC AGAIN") || d.m.runner.Depth() != 2 {
		t.Errorf("an Esc after the window must ask again, not abort:\n%s", d.screen())
	}
}

// In key mode, keys go to the game instead of the line editor.
func TestKeyMode(t *testing.T) {
	t.Parallel()
	d := newDriver(t, Options{Instant: true, Play: "stub", Registry: gamestest.Registry()}, 80, 24).settle()
	d.line("KEYS").key(up)
	if !strings.Contains(d.screen(), "KEY up") {
		t.Errorf("the arrow did not reach the game:\n%s", d.screen())
	}
}

func TestFrontPanelAndCentring(t *testing.T) {
	t.Parallel()
	var s snapshots
	opts := instant()
	opts.Panel = PanelOn
	d := newDriver(t, opts, 100, 30).settle().line("Joshua")
	s.add("panel, 100 columns", d)
	golden.AssertString(t, "front_panel", s.String())

	norad := instant()
	norad.Theme = "norad"
	if d := newDriver(t, norad, 80, 24).settle(); !strings.Contains(d.screen(), "W.O.P.R.") {
		t.Error("norad shows the front panel by default")
	}
	norad.Panel = PanelOff
	if d := newDriver(t, norad, 80, 24).settle(); strings.Contains(d.screen(), "W.O.P.R.") {
		t.Error("WOPR_PANEL=0 hides it, even in norad")
	}
}

func TestScrollback(t *testing.T) {
	t.Parallel()
	d := newDriver(t, instant(), 80, 24).settle().line("Joshua")
	for range 6 {
		d.line("HELP")
	}
	bottom := d.screen()
	d.key(pgUp)
	if d.screen() == bottom {
		t.Fatal("PgUp did not scroll")
	}
	d.key(pgDn)
	if d.screen() != bottom {
		t.Errorf("PgDn did not return to the bottom:\n%s", d.screen())
	}
}

// Every layout at exactly 80x24, with and without the front panel: the geometry of
// docs/PLAN.md §4.3 (H' is the height minus the panel row).
func TestLayoutsAt80x24(t *testing.T) {
	t.Parallel()
	var s snapshots
	for _, panel := range []Panel{PanelOff, PanelOn} {
		name := map[Panel]string{PanelOff: "no panel", PanelOn: "panel"}[panel]
		console := instant()
		console.Panel = panel
		s.add("console, "+name, newDriver(t, console, 80, 24).settle().line("Joshua"))

		game := Options{Instant: true, Play: "stub", Registry: gamestest.Registry(), Panel: panel}
		d := newDriver(t, game, 80, 24).settle()
		s.add("panel layout, "+name, d)
		d.line("FULL")
		s.add("full layout, "+name, d)
		if lines := strings.Count(d.screen(), "\n"); lines != 25 { // 24 rows plus the cursor line
			t.Errorf("%s: %d rows, want 24", name, lines-1)
		}
	}
	golden.AssertString(t, "layouts_80x24", s.String())
}

// The real GTW board in the norad theme, front panel shown: the Appendix C screens.
func TestGTWScreens(t *testing.T) {
	t.Parallel()
	var s snapshots
	opts := instant()
	opts.Theme, opts.Play = "norad", "global-thermonuclear-war" // cli resolves aliases to slugs
	d := newDriver(t, opts, 80, 24).settle()
	s.add("side choice", d)
	d.line("2").line("Las Vegas").line("Seattle").line("")
	s.add("big board: strike 2 orders (DEFCON 4)", d)
	d.line("help")
	s.add("big board: help at strike 2", d)
	d.line("")
	s.add("big board: strike 3 orders (DEFCON 3)", d)
	d.line("")
	s.add("big board: DEFCON 1, assessed", d)
	d.line("")
	s.add("kill ratios", d)
	d.line("")
	s.add("climax", d)
	d.line("List Games")
	s.add("climax: LIST GAMES in the console", d)
	d.line("Chess")
	s.add("climax: the board again", d)
	golden.AssertString(t, "gtw_screens", s.String())
}

// Every playable game starts and fits 80x24, with and without the front panel: 24 rows,
// none wider than 80 columns.
func TestEveryGameFitsAt80x24(t *testing.T) {
	t.Parallel()
	for _, e := range catalog.Registry().All() {
		if e.Info.Status != games.Playable {
			continue
		}
		for _, panel := range []Panel{PanelOff, PanelOn} {
			opts := instant()
			opts.Play, opts.Panel = e.Info.Slug, panel
			d := newDriver(t, opts, 80, 24).settle()
			screen := d.screen()
			rows := strings.Split(strings.TrimSuffix(screen, "\n"), "\n")
			if len(rows) != 25 { // 24 rows plus the cursor line
				t.Errorf("%s (panel %d): %d rows", e.Info.Slug, panel, len(rows)-1)
			}
			for i, r := range rows[:len(rows)-1] {
				if w := ansi.StringWidth(r); w > 80 {
					t.Errorf("%s (panel %d): row %d is %d wide: %q", e.Info.Slug, panel, i, w, r)
				}
			}
		}
	}
}

// The card games (and the maze, in key mode with its hint) as the player first sees them:
// the Hearts mockup the plan asks for, kept current by this golden.
func TestCardGameScreens(t *testing.T) {
	t.Parallel()
	var s snapshots
	for _, slug := range []string{"hearts", "gin-rummy", "bridge", "falkens-maze"} {
		opts := instant()
		opts.Play = slug
		d := newDriver(t, opts, 80, 24).settle()
		if slug == "hearts" {
			d.line("1 2 3")
		}
		s.add(slug, d)
	}
	golden.AssertString(t, "card_screens", s.String())
}

// Esc's notice above the input line, then PgUp: at some history lengths the view clamps
// to an offset of one row, the notice is the bottom row and the input line is off screen.
// View used to index past its rows there and panic.
func TestEscThenPageUpDoesNotPanic(t *testing.T) {
	t.Parallel()
	for k := range 24 {
		opts := instant()
		opts.Play = "chess"
		d := newDriver(t, opts, 80, 24).settle()
		for range k {
			d.m.sb.Append("FILLER", 0)
		}
		d.send(esc).send(pgUp) // no settling: the Esc window stays open
		_ = d.screen()
		d.send(pgUp).send(pgDn)
		_ = d.screen()
	}
}

// A line longer than the console wraps onto further rows instead of running off the edge,
// and the cursor follows its end.
func TestLongInputWraps(t *testing.T) {
	t.Parallel()
	d := newDriver(t, instant(), 80, 24).settle().line("Joshua")
	d.typ(strings.Repeat("ABCDEFGHI ", 9) + "XYZ_END")
	screen := d.screen()
	if !strings.Contains(screen, "XYZ_END") {
		t.Fatalf("the end of the line is not on screen:\n%s", screen)
	}
	for i, r := range strings.Split(screen, "\n") {
		if ansi.StringWidth(r) > 80 {
			t.Errorf("row %d is %d wide", i, ansi.StringWidth(r))
		}
	}
	if !strings.Contains(screen, "cursor: col 17, row 3") { // after "ABCDEFGHI XYZ_END" on the second row
		t.Errorf("cursor:\n%s", screen)
	}
}

// Any key but Esc closes the Esc window: Esc, typing, Esc does not end the game.
func TestTypingDisarmsEsc(t *testing.T) {
	t.Parallel()
	opts := instant()
	opts.Play = "chess"
	d := newDriver(t, opts, 80, 24).settle()
	d.send(esc).send(tea.KeyPressMsg{Code: 'e', Text: "e"})
	if strings.Contains(d.screen(), "PRESS ESC AGAIN") {
		t.Fatal("typing must clear the Esc notice")
	}
	d.send(esc)
	if d.m.runner.Depth() != 2 {
		t.Fatal("Esc, a key, Esc must not end the game")
	}
}

// A Think that finishes while the TOO SMALL card is up is held until the size is valid.
func TestThinkHeldWhileTooSmall(t *testing.T) {
	t.Parallel()
	opts := Options{Instant: true, Play: "stub", Registry: gamestest.Registry()}
	d := newDriver(t, opts, 80, 24).settle()
	d.slowThinks = true
	d.press("THINK").send(enter).steps(5)
	d.resize(60, 20).steps(5)
	d.finishThinks() // the result arrives behind the card
	if strings.Contains(strings.Join(transcript(d), "\n"), "THOUGHT.") {
		t.Fatal("the result must wait behind the TOO SMALL card")
	}
	d.resize(80, 24).settle()
	if !strings.Contains(strings.Join(transcript(d), "\n"), "THOUGHT.") {
		t.Fatalf("the result arrives once the size is valid:\n%s", d.screen())
	}
}

// When a game ends, its prompt and a line typed ahead for it do not carry over to WOPR.
func TestPopDropsTheGamesTypeahead(t *testing.T) {
	t.Parallel()
	opts := instant()
	opts.Play = "chess"
	d := newDriver(t, opts, 80, 24).settle()
	d.slowThinks = true
	d.press("e2e4").send(enter).steps(3)
	d.press("d2d4").send(enter) // held while WOPR thinks
	if !d.m.held {
		t.Fatal("the second move should be held")
	}
	d.send(esc).send(esc).settle()
	if d.m.runner.Depth() != 1 {
		t.Fatal("Esc twice ends the game")
	}
	if strings.Contains(strings.Join(transcript(d), "\n"), "d2d4") || d.m.ed.Value() != "" {
		t.Fatalf("the held move must not reach WOPR:\n%s", d.screen())
	}
	if d.m.prompt == "YOUR MOVE: " {
		t.Fatalf("the game's prompt must not linger:\n%s", d.screen())
	}
}

// Once the terminal confirms grapheme widths (mode 2027), wrapping measures that way too.
func TestModeReportSwitchesWidths(t *testing.T) {
	t.Parallel()
	d := newDriver(t, instant(), 80, 24).settle()
	d.send(tea.ModeReportMsg{Mode: ansi.ModeUnicodeCore, Value: ansi.ModeSet})
	if d.m.method != ansi.GraphemeWidth {
		t.Fatal("mode 2027 must switch to grapheme widths")
	}
}

func transcript(d *driver) []string {
	var out []string
	for _, l := range d.m.sb.Lines() {
		out = append(out, l.Visible())
	}
	return out
}

// A game that ends on a move keeps its board up while its last words type out; the
// layout changes when WOPR's verdict starts.
func TestFinalBoardStaysUntilItsLastWords(t *testing.T) {
	t.Parallel()
	opts := Options{Seed: 13, SeedSet: true, Registry: catalog.Registry(), Play: "chess"}
	d := newDriver(t, opts, 80, 24).settle()
	d.line("f2f3")
	d.press("g2g4").send(enter)
	sawMate := false
	for range 400 {
		if !d.step() {
			break
		}
		screen := d.screen()
		if strings.Contains(screen, "WOPR: D8H4") && !strings.Contains(screen, "CHECKMATE.") {
			sawMate = true // the mating move is out, the game's last line still typing
			if !strings.Contains(screen, "a b c d e f g h") {
				t.Fatalf("the board vanished before the mating move was shown:\n%s", screen)
			}
		}
	}
	if !sawMate {
		t.Fatalf("expected fool's mate:\n%s", d.screen())
	}
	d.settle()
	if d.m.last != nil || d.m.place.Layout != proto.LayoutConsole || !strings.Contains(d.screen(), "WINNER: WOPR") {
		t.Fatalf("after the verdict the console layout returns:\n%s", d.screen())
	}
}

// consoleText is the console part of the screen: the rows below the running program's
// View, without the front panel.
func consoleText(d *driver) string {
	g := d.m.geometry(d.m.place)
	rows := strings.Split(d.screen(), "\n")
	return strings.Join(rows[g.viewRows:g.viewRows+g.consoleRows], "\n")
}

// The owner's report: after Falken's Maze, GIN RUMMY's board sat on top of the maze's
// last lines and the WHICH GAME? menu, and so did CHECKERS after gin. A game with its own
// screen now starts on a new page, so its console strip shows only its own text; earlier
// pages stay in the scrollback. A game ending keeps its page, so its last lines stay above
// WOPR's verdict and SHALL WE PLAY ANOTHER GAME?.
func TestPanelGameStartsOnANewPage(t *testing.T) {
	t.Parallel()
	before := []string{
		"FIND THE EXIT", "RETREAT ACCEPTED", "WINNER: WOPR", "SHALL WE PLAY ANOTHER GAME?",
		"WHICH GAME?", "BLACK JACK", "GLOBAL THERMONUCLEAR WAR", "yes",
	}
	clean := func(d *driver, label string, old []string, want string) {
		t.Helper()
		text := consoleText(d)
		for _, s := range old {
			if strings.Contains(text, s) {
				t.Errorf("%s: %q from before the launch is still under the board:\n%s", label, s, d.screen())
			}
		}
		if want != "" && !strings.Contains(text, want) {
			t.Errorf("%s: no %q in the console:\n%s", label, want, d.screen())
		}
	}

	opts := instant()
	opts.Play = "falkens-maze"
	d := newDriver(t, opts, 80, 24).settle()
	d.key(keyRune('q')) // give up: the maze ends and WOPR's console returns
	if s := d.screen(); !strings.Contains(s, "RETREAT ACCEPTED") || !strings.Contains(s, "SHALL WE PLAY ANOTHER GAME?") {
		t.Fatalf("the maze's last line and WOPR's question stay on the page:\n%s", s)
	}
	d.line("yes").line("gin rummy")
	if _, place := d.m.runner.Top(); place.Layout != proto.LayoutPanel {
		t.Fatalf("gin rummy is not running:\n%s", d.screen())
	}
	clean(d, "gin after the maze", append(before, "gin rummy"), "STOCK OR DISCARD?")

	d.send(esc).send(esc).settle() // end gin; its page stays with the verdict
	if s := d.screen(); !strings.Contains(s, "GIN RUMMY TO 100") || !strings.Contains(s, "SHALL WE PLAY ANOTHER GAME?") {
		t.Fatalf("gin's text and WOPR's question stay on the page:\n%s", s)
	}
	d.line("yes").line("checkers")
	clean(d, "checkers after gin", append(before, "GIN RUMMY TO 100", "STOCK OR DISCARD?", "GAME TERMINATED", "checkers"), "YOUR MOVE:")

	d.key(pgUp) // the earlier pages are still there
	if !strings.Contains(consoleText(d), "checkers") {
		t.Errorf("PgUp must reach the page before the launch:\n%s", d.screen())
	}

	// With pacing on, the very first frame of the new board is already clean, before the
	// clock ticks.
	paced := instant()
	paced.Instant, paced.Play = false, "falkens-maze"
	d = newDriver(t, paced, 80, 24).settle()
	d.key(keyRune('q')).line("yes")
	d.typ("gin rummy").send(enter)
	if _, place := d.m.runner.Top(); place.Layout != proto.LayoutPanel {
		t.Fatalf("gin rummy is not running:\n%s", d.screen())
	}
	clean(d, "gin's first frame", append(before, "gin rummy"), "")
	d.settle()
	clean(d, "gin, paced", append(before, "gin rummy"), "STOCK OR DISCARD?")
}
