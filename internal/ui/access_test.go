package ui

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/catalog"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/theme"
)

// The accessibility pass (docs/PLAN.md §4.5) as checks that sweep every game, so a new game
// is covered as soon as it is in the catalog. Each game is played a little way through the
// real UI from a playbook of inputs (a game with no playbook is checked on its first screen),
// and every screen it reaches must hold up: real characters only, prompts that end in a
// question or a colon, the cursor at the end of the input line, key mode saying what the keys
// do, no attribute that no theme shows, and the same text without colour as with it, in every
// theme. Played again with pacing, no game may flash: no more than three changes over a large
// part of its view in any second.

// playbook is what to type into a game, in turn: lines at a prompt, and in key mode the
// names of keys (keyNames). With cycle, the list starts again when it runs out, up to steps
// inputs in all.
type playbook struct {
	inputs []string
	cycle  bool
	steps  int
}

// keyNames are the keys a playbook can press in key mode.
var keyNames = map[string]tea.KeyPressMsg{
	"up": {Code: tea.KeyUp}, "down": {Code: tea.KeyDown}, "left": {Code: tea.KeyLeft}, "right": {Code: tea.KeyRight},
}

// playbooks reach each game's main screens. Most inputs are plausible rather than right: a
// refusal is a screen too, and cycling the list makes progress through any of them.
var playbooks = map[string]playbook{
	"falkens-maze": {inputs: []string{"right", "right", "down", "right", "down", "left", "down", "right", "up"}, cycle: true, steps: 40},
	"black-jack":   {inputs: []string{"5", "h", "s", "d"}, cycle: true, steps: 24},
	"gin-rummy":    {inputs: []string{"s", "1", "d", "2", "y"}, cycle: true, steps: 30},
	"hearts":       {inputs: append([]string{"1 2 3"}, numbers(13)...), cycle: true, steps: 50},
	"bridge":       {inputs: allCards(), cycle: true, steps: 80},
	"checkers": {inputs: []string{
		"c3-d4", "b6-a5", "b2-c3", "a3-b4", "e3-f4", "d2-e3", "g3-h4", "f2-g3", "c1-d2", "e1-f2", "h2-g3", "c3xe5", "d4xb6",
	}, cycle: true, steps: 40},
	"chess":                        {inputs: []string{"e2e4", "d2d4", "g1f3", "b1c3", "f1c4", "c1f4", "d1d2", "e1g1", "a2a3", "h2h3", "resign"}},
	"poker":                        {inputs: []string{"c", "b", "1 2", "c", "f", "y"}, cycle: true, steps: 30},
	"fighter-combat":               {inputs: []string{"press", "climb", "fire", "extend", "break", "fire"}, cycle: true, steps: 30},
	"guerrilla-engagement":         {inputs: []string{"hold", "attack 4", "hide", "status", "end"}, cycle: true, steps: 30},
	"desert-warfare":               {inputs: []string{"move 2", "attack 5", "hold", "status", "end"}, cycle: true, steps: 30},
	"air-to-ground-actions":        {inputs: []string{"target 1 strike 2 sead 2 escort 2", "go"}, cycle: true, steps: 16},
	"theaterwide-tactical-warfare": {inputs: []string{"attack 4", "escalate", "hold", "end"}, cycle: true, steps: 30},
	"theaterwide-biotoxic-and-chemical-warfare": {inputs: []string{"release 4", "decon", "hold", "end"}, cycle: true, steps: 30},
	// The film's path: the big board to DEFCON 1, the kill ratios, the climax, tic-tac-toe
	// with zero players, and the ending.
	"global-thermonuclear-war": {inputs: []string{
		"2", "Las Vegas", "Seattle", "", "", "", "", "", "List Games", "Chess", "Tic-Tac-Toe", "0", "Hello.",
	}},
	"tic-tac-toe": {inputs: []string{"1", "5", "1", "3", "7", "9", "2", "8", "4", "6"}},
}

func numbers(n int) []string {
	var out []string
	for i := range n {
		out = append(out, strconv.Itoa(i+1))
	}
	return out
}

func allCards() []string {
	var out []string
	for _, s := range []string{"S", "H", "D", "C"} {
		for _, r := range []string{"A", "K", "Q", "J", "10", "9", "8", "7", "6", "5", "4", "3", "2"} {
			out = append(out, r+s)
		}
	}
	return out
}

// sweepCase is one thing the sweep plays: a game by slug, the persona, or movie mode.
type sweepCase struct {
	name string
	opts func(Options) Options
	book playbook
	film bool // it plays itself: check its screens as the clock runs, not after each input
}

func sweepCases() []sweepCase {
	cases := []sweepCase{
		{"persona", func(o Options) Options { return o }, playbook{inputs: []string{
			"000001", "Joshua", "Hello.", "I'm fine. How are you?", "help", "list games", "logoff",
		}}, false},
		{"movie menu", func(o Options) Options { o.Movie = true; return o }, playbook{inputs: []string{"nowhere", "c", "q"}}, false},
		// The whole film, from the first scene to the end of the list.
		{"movie", func(o Options) Options { o.Movie, o.Scene = true, "first-contact"; return o }, playbook{}, true},
	}
	for _, e := range catalog.Registry().All() {
		if e.Info.Status != games.Playable {
			continue
		}
		slug := e.Info.Slug
		cases = append(cases, sweepCase{slug, func(o Options) Options { o.Play = slug; return o }, playbooks[slug], false})
	}
	return cases
}

// play runs c's playbook on d, calling see after each input (and once before the first). It
// stops when the inputs run out, the session ends, or the game is over (the persona alone is
// left, except for the persona's own case).
func (c sweepCase) play(d *driver, see func(label string)) {
	see("start")
	book, game := c.book, c.opts(Options{}).Play != ""
	steps := book.steps
	if steps == 0 || !book.cycle {
		steps = len(book.inputs)
	}
	for i := 0; i < steps && d.ended == ""; i++ {
		if game && d.m.runner.Depth() < 2 {
			return // the game is over
		}
		in := book.inputs[i%len(book.inputs)]
		switch k, isKey := keyNames[in]; {
		case d.m.keyMode && isKey:
			d.send(k)
		case d.m.keyMode:
			continue
		default:
			d.press(in).send(enter)
		}
		see(fmt.Sprintf("after %q (input %d)", in, i+1))
	}
}

// TestEveryScreenIsAccessible plays every game, the persona, the movie menu and the whole film
// with no pacing, and checks each screen they reach (the film's every half second), in every
// theme, with and without colour.
func TestEveryScreenIsAccessible(t *testing.T) {
	t.Parallel()
	for _, c := range sweepCases() {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			opts := c.opts(Options{Instant: true, Seed: 1, SeedSet: true, Registry: catalog.Registry(), Panel: PanelOff})
			d := newDriver(t, opts, 80, 24)
			screens := 0
			c.play(d, func(label string) {
				for i := 0; c.film && d.ended == "" && i < maxSteps; i++ {
					if i%filmEvery == 0 {
						inspectScreen(t, fmt.Sprintf("%s, tick %d", c.name, i), d)
						screens++
					}
					d.step()
				}
				d.settle()
				if d.ended == "" {
					inspectScreen(t, label, d)
					screens++
				}
			})
			if screens == 0 {
				t.Error("no screen was checked")
			}
		})
	}
}

// filmEvery is how often, in ticks, the sweep checks the screen of a case that plays itself:
// about every half second.
const filmEvery = 15

// inspectScreen checks the screen d shows now (docs/PLAN.md §4.5).
func inspectScreen(t *testing.T, label string, d *driver) {
	t.Helper()
	m := d.m
	v := m.View()
	screen := ansi.Strip(v.Content)
	rows := strings.Split(screen, "\n")
	fail := func(format string, args ...any) {
		t.Helper()
		t.Errorf("%s: %s\n%s", label, fmt.Sprintf(format, args...), d.screen())
	}

	// Real characters: printable ASCII, so a screen reader or a copy reads what is drawn.
	for _, r := range screen {
		if r != '\n' && (r < ' ' || r > '~') {
			fail("the screen holds %q, which is not printable ASCII", r)
			break
		}
	}

	captured := m.runner.Captures() || m.runner.Shielded() // movie mode, which documents its keys itself
	switch {
	case m.keyMode && !captured && m.keyHint == "":
		fail("key mode with no hint: nothing says what the keys do")
	case m.keyMode && !captured && !strings.Contains(screen, m.keyHint):
		fail("the key hint %q is not on screen", m.keyHint)
	case m.asking && !m.keyMode:
		// A prompt asks a question or ends in a colon. The film's open prompt (WOPR's
		// conversation, GTW's target list) has no text: the line above it asks.
		if p := strings.TrimRight(m.prompt, " "); p != "" && !strings.HasSuffix(p, ":") && !strings.HasSuffix(p, "?") {
			fail("the prompt %q ends in neither a colon nor a question mark", m.prompt)
		}
		// The cursor sits at the end of the input line, where a screen magnifier and a
		// screen reader's caret look for it.
		input := m.prompt + m.ed.Value()
		g := m.geometry(m.place)
		switch {
		case v.Cursor == nil:
			fail("the line is asked for, but the cursor is hidden")
		case len(input) < g.width && (v.Cursor.Y >= len(rows) || !strings.HasPrefix(rows[v.Cursor.Y][g.margin:], input) || v.Cursor.X != g.margin+len(input)):
			fail("the cursor is at col %d, row %d, not at the end of %q", v.Cursor.X, v.Cursor.Y, input)
		}
	}

	if c := currentView(m); c != nil {
		inspectCanvas(t, label, c)
	}

	// Without colour (NO_COLOR, a monochrome terminal) nothing is lost: in every theme the
	// screen has the same text, and no colour at all.
	th, profile := m.th, m.profile
	defer func() { m.th, m.profile = th, profile }()
	for _, name := range theme.Names() {
		m.th, _ = theme.Get(name)
		m.profile = colorprofile.TrueColor
		coloured := ansi.Strip(m.View().Content)
		m.profile = colorprofile.Ascii
		plain := m.View().Content
		if ansi.Strip(plain) != coloured {
			fail("theme %s: the text without colour differs from the text with it", name)
		}
		if p := colourParam(plain); p != "" {
			fail("theme %s: SGR %s is a colour, under NO_COLOR", name, p)
		}
	}
}

// currentView draws the running program's view as the UI would, or returns nil in the
// console layout.
func currentView(m *model) *proto.Canvas {
	g := m.geometry(m.place)
	prog, _ := m.runner.Top()
	if m.last != nil {
		prog = m.last
	}
	if g.viewRows == 0 || prog == nil {
		return nil
	}
	c := proto.NewCanvas(g.width, g.viewRows)
	prog.View(c)
	return c
}

// inspectCanvas checks a program's view: pure ASCII art and words, and no attribute that no
// theme shows. An attribute the style already has in every theme (bold on bright) changes
// nothing: as decoration it is dead, and as meaning (the trick's winning card, the last move)
// it is lost.
func inspectCanvas(t *testing.T, label string, c *proto.Canvas) {
	t.Helper()
	for y := range c.H {
		for x := range c.W {
			cell := c.At(x, y)
			if cell.R < ' ' || cell.R > '~' {
				t.Errorf("%s: the view has %q at %d,%d, which is not printable ASCII", label, cell.R, x, y)
				return
			}
			for a, name := range map[proto.Attr]string{proto.AttrBold: "bold", proto.AttrReverse: "reverse", proto.AttrUnderline: "underline"} {
				if cell.A&a != 0 && !attrShows(cell.S, a) {
					t.Errorf("%s: %q at %d,%d is %s, which its style (%c) already is in every theme:\n%s",
						label, cell.R, x, y, name, cell.S.Letter(), c.String())
					return
				}
			}
		}
	}
}

// attrShows reports whether attribute a changes how a cell of style s looks in at least one
// theme.
func attrShows(s proto.Style, a proto.Attr) bool {
	for _, name := range theme.Names() {
		th, _ := theme.Get(name)
		if th.Styles[s].Attr&a == 0 {
			return true
		}
	}
	return false
}

// colourParam returns the first SGR parameter in s that sets a colour, or "".
func colourParam(s string) string {
	for {
		i := strings.Index(s, "\x1b[")
		if i < 0 {
			return ""
		}
		s = s[i+2:]
		j := strings.IndexFunc(s, func(r rune) bool { return r >= '@' && r <= '~' })
		if j < 0 {
			return ""
		}
		if s[j] == 'm' {
			for p := range strings.SplitSeq(s[:j], ";") {
				main, _, _ := strings.Cut(p, ":")
				if n, err := strconv.Atoi(main); err == nil && (n >= 30 && n <= 49 || n >= 90 && n <= 107) {
					return p
				}
			}
		}
		s = s[j+1:]
	}
}

// flashWindow and flashMost are the flash rule: no more than three large changes in any one
// second (WCAG 2.3.1's three flashes, counted strictly: each change, not each pair). A change
// is large when it covers a tenth of the view or more.
const (
	flashWindow = time.Second
	flashMost   = 3
	largeShare  = 10
)

// TestNothingFlashes plays every game with pacing, as a player sees it, and watches the
// program's own view on every tick of the clock: what the program animates (GTW's flights and
// its DEFCON 1 blink, the ending's self-play and montage, movie mode's board and game clock)
// must stay under the flash rule. A change that follows a key press is the player's own doing
// and is not counted, nor is the console's scrolling text. Movie mode is watched from its
// first scene to the film's last words.
func TestNothingFlashes(t *testing.T) {
	t.Parallel()
	for _, c := range sweepCases() {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			opts := c.opts(Options{Seed: 1, SeedSet: true, Registry: catalog.Registry(), Panel: PanelOff})
			d := newDriver(t, opts, 80, 24)
			var w flashWatch
			watch := func(string) { w.run(t, d) }
			c.play(d, watch)
			w.run(t, d)
			if w.ticks == 0 {
				t.Error("the clock never ran")
			}
		})
	}
}

// flashWatch counts the large changes the clock's ticks make to a program's view.
type flashWatch struct {
	ticks   int
	elapsed time.Duration
	large   []time.Duration // when each large change happened
}

// lingerFor is how long the watch goes on once a game waits for the player: what still
// moves then (GTW's DEFCON 1 blink, its search for the launch code) would move forever.
const lingerFor = 2 * time.Second

// run runs the clock, comparing the view before and after each tick, until nothing is armed
// or the game has waited lingerFor for the player.
func (w *flashWatch) run(t *testing.T, d *driver) {
	t.Helper()
	var waited time.Duration
	for i := 0; ; i++ {
		m := d.m
		captured := m.runner.Captures() || m.runner.Shielded()
		if !captured && (m.asking || m.keyMode) && !m.tw.Busy() {
			if waited += tickInterval; waited > lingerFor {
				return
			}
		}
		before := visibleView(m)
		if i > maxSteps || !d.step() {
			return
		}
		w.ticks++
		w.elapsed += tickInterval
		after := visibleView(d.m)
		if before == nil || after == nil || before.W != after.W || before.H != after.H {
			continue // a layout change: a new screen, not a flash
		}
		changed := 0
		for y := range after.H {
			for x := range after.W {
				if before.At(x, y) != after.At(x, y) {
					changed++
				}
			}
		}
		if changed*largeShare < after.W*after.H {
			continue
		}
		w.large = append(w.large, w.elapsed)
		if n := len(w.large); n > flashMost && w.large[n-1]-w.large[n-1-flashMost] < flashWindow {
			t.Fatalf("%d large changes to the view within %v (the last changed %d cells):\n%s",
				flashMost+1, w.large[n-1]-w.large[n-1-flashMost], changed, after.String())
		}
	}
}

// visibleView is the program's view as the screen shows it at this moment: a blinking cell
// is blank in the blink's off phase, as the renderer draws it.
func visibleView(m *model) *proto.Canvas {
	c := currentView(m)
	if c == nil {
		return nil
	}
	off := !m.opts.ReduceMotion && int(m.phase/blinkPeriod)%2 == 1
	for y := range c.H {
		for x := range c.W {
			if cell := c.At(x, y); cell.A&proto.AttrBlink != 0 && off {
				cell.R = ' '
				c.Set(x, y, cell)
			}
		}
	}
	return c
}

// TestReduceMotion checks --reduce-motion (docs/PLAN.md §4.5) on the screen, tick by tick: the
// cursor never blinks, a Blink cell always shows, the front panel's lights stand still, and
// PROCESSING keeps three dots. It watches movie mode's climax, which has every kind of motion
// a program makes (the DEFCON 1 blink on the board, its search for the launch code,
// tic-tac-toe's self-play and the montage), and chess thinking about a slow move. Without the
// flag each of them moves, so the check is not vacuous.
func TestReduceMotion(t *testing.T) {
	t.Parallel()
	for _, reduce := range []bool{true, false} {
		t.Run(fmt.Sprintf("reduce motion %v", reduce), func(t *testing.T) {
			t.Parallel()
			var m motion
			movie := Options{Movie: true, Scene: "climax", Registry: catalog.Registry(), Panel: PanelOn, ReduceMotion: reduce}
			d := newDriver(t, movie, 80, 24)
			for i := 0; d.ended == "" && i < maxSteps; i++ {
				if i%4 == 0 { // every 132 ms: the fastest motion (the lights) steps every 250
					m.see(d)
				}
				d.step()
			}
			chess := Options{Seed: 1, SeedSet: true, Registry: catalog.Registry(), Play: "chess", Panel: PanelOn, ReduceMotion: reduce}
			d = newDriver(t, chess, 80, 24).settle()
			d.slowThinks = true
			d.press("e2e4").send(enter)
			for range 60 { // two seconds of WOPR thinking
				m.see(d)
				d.step()
			}
			d.finishThinks()
			if !m.thought {
				t.Fatal("WOPR never thought where the test could see it")
			}
			moved := m.moved()
			if reduce && len(moved) > 0 {
				t.Errorf("with --reduce-motion, these moved: %v", moved)
			}
			if !reduce && len(moved) != 4 {
				t.Errorf("without --reduce-motion, only these moved: %v; the check would miss the rest", moved)
			}
		})
	}
}

// motion records what has moved on screen.
type motion struct {
	cursorBlink, blinkHidden, dots, thought bool
	panels                                  map[string]bool // the front panel rows seen while WOPR thinks
}

// see looks at the screen d shows now.
func (m *motion) see(d *driver) {
	v := d.m.View()
	rows := strings.Split(ansi.Strip(v.Content), "\n")
	if v.Cursor != nil && v.Cursor.Blink {
		m.cursorBlink = true
	}
	if m.panels == nil {
		m.panels = map[string]bool{}
	}
	if d.m.thinking() { // the lights move only while WOPR thinks
		m.thought = true
		m.panels[rows[len(rows)-1]] = true
		for _, r := range rows {
			if strings.HasPrefix(r, "PROCESSING") && strings.TrimSpace(r) != "PROCESSING ..." {
				m.dots = true
			}
		}
	}
	c := currentView(d.m)
	if c == nil {
		return
	}
	g := d.m.geometry(d.m.place)
	for y := range c.H {
		for x := range c.W {
			if cell := c.At(x, y); cell.A&proto.AttrBlink != 0 && rows[y][g.margin+x] != byte(cell.R) {
				m.blinkHidden = true
			}
		}
	}
}

// moved names what has moved.
func (m *motion) moved() []string {
	var out []string
	for name, moved := range map[string]bool{
		"the cursor's blink": m.cursorBlink, "a Blink cell": m.blinkHidden, "PROCESSING's dots": m.dots,
		"the front panel's lights": len(m.panels) > 1,
	} {
		if moved {
			out = append(out, name)
		}
	}
	slices.Sort(out)
	return out
}
