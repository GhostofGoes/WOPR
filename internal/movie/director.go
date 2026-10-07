package movie

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/games/ending"
	"github.com/GhostofGoes/WOPR/internal/games/gtw"
	"github.com/GhostofGoes/WOPR/internal/movie/scenes"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// Options configure a Director.
type Options struct {
	// Scene plays from this scene (a name Resolve accepts) to the end of the list, then quits
	// (RF-5). Empty opens the scene menu instead.
	Scene string
	// Single plays only Scene, then quits (tests).
	Single bool
	// Scenes replaces the film's scenes (tests); nil plays scenes.All().
	Scenes []scenes.Scene
}

type phase uint8

const (
	inMenu    phase = iota // the scene menu waits for a line
	playing                // a step is out; the next one follows on Drained
	animating              // a Board step's war plays out on ticks
	running                // a Run step's program is on top
	ended
)

// Director plays the film's scenes as the root program (docs/PLAN.md §7). It emits one step at
// a time and waits for the host's Drained before the next, so a pause or a skip to another
// scene drops only what is on screen. It captures keys: Space pauses and resumes, Right or n
// skips to the next scene, Left or p goes back one, and Esc opens the menu. While a program it
// launched runs (the climax), keys do nothing but Ctrl+C.
type Director struct {
	opts   Options
	scenes []scenes.Scene
	env    proto.Env
	phase  phase
	scene  int
	step   int
	paused bool
	menued bool       // the menu has been shown: the end of the list returns to it
	rng    *rand.Rand // the scene's typing jitter
	film   *gtw.Film  // the scene's war, from its first Board step
	board  bool       // the board is on screen (the Full layout)
	target gtw.FilmStage
}

// New returns a director.
func New(opts Options) *Director {
	d := &Director{opts: opts, scenes: opts.Scenes}
	if d.scenes == nil {
		d.scenes = scenes.All()
	}
	return d
}

// Start implements proto.Program. Env.Seed is ignored: movie mode pins Seed.
func (d *Director) Start(env proto.Env) []proto.Output {
	d.env = env
	if d.opts.Scene != "" {
		if i, err := resolve(infos(d.scenes), d.opts.Scene); err == nil {
			return d.begin(i)
		}
	}
	return d.menu()
}

// View implements proto.Program: the big board, when a scene shows it.
func (d *Director) View(c *proto.Canvas) {
	if d.board && d.film != nil {
		d.film.View(c)
	}
}

// Handle implements proto.Program.
func (d *Director) Handle(ev proto.Event) []proto.Output {
	switch ev := ev.(type) {
	case proto.Drained:
		if d.phase != playing {
			return nil
		}
		d.step++
		return d.emit()
	case proto.TickEvent:
		return d.tick(ev)
	case proto.GameOver:
		if d.phase != running {
			return nil
		}
		d.phase = playing
		return []proto.Output{proto.Drain{}} // the program's last words, then the next step
	case proto.KeyEvent:
		return d.key(ev)
	case proto.LineEvent:
		return d.choose(ev.Text)
	case proto.ResizeEvent:
		d.env.Width, d.env.Height = ev.Width, ev.Height
	}
	return nil
}

// begin starts scene i on a clean page, with the board put away.
func (d *Director) begin(i int) []proto.Output {
	d.phase, d.scene, d.step = playing, i, 0
	d.rng = proto.NewRand(Seed, proto.DomainMovie|uint64(i+1))
	d.film, d.target, d.board = nil, 0, false
	outs := []proto.Output{
		proto.AwaitKeys{Capture: true},
		proto.Animate{},
		proto.SetLayout{Layout: proto.LayoutConsole},
		proto.Clear{},
	}
	return append(outs, d.emit()...)
}

// emit puts the current step on screen, then asks to hear when it is out.
func (d *Director) emit() []proto.Output {
	steps := d.scenes[d.scene].Steps
	if d.step >= len(steps) {
		return d.next()
	}
	switch s := steps[d.step].(type) {
	case scenes.Type:
		prompt := ""
		if len(s.Prompt) > 0 {
			prompt = s.Prompt[0].Text
		}
		return append(proto.Typed(prompt, s.Text.Text, d.rng), proto.Drain{})
	case scenes.Say:
		return []proto.Output{proto.Say{Lines: s.Lines.Texts(), Pace: s.Pace}, proto.Drain{}}
	case scenes.Clear:
		return []proto.Output{proto.Clear{}, proto.Drain{}}
	case scenes.Wait:
		return []proto.Output{proto.Wait{D: s.D}, proto.Drain{}}
	case scenes.Board:
		return d.showBoard(s)
	case scenes.Run:
		return d.run(s)
	}
	panic(fmt.Sprintf("movie: unknown step %T", steps[d.step]))
}

// next follows the end of a scene: the next scene, or the end of the list.
func (d *Director) next() []proto.Output {
	if d.opts.Single || d.scene+1 >= len(d.scenes) {
		return d.end()
	}
	return d.begin(d.scene + 1)
}

// end follows the last scene: back to the menu if the viewer came from it, otherwise exit 0.
func (d *Director) end() []proto.Output {
	if d.menued && !d.opts.Single {
		return d.menu()
	}
	d.phase = ended
	return []proto.Output{proto.Quit{}}
}

// showBoard plays a Board step.
func (d *Director) showBoard(s scenes.Board) []proto.Output {
	if s.At == 0 {
		return append(d.hideBoard(), proto.Drain{})
	}
	var outs []proto.Output
	if d.film == nil {
		d.film = gtw.NewFilm(proto.NewRand(Seed, proto.DomainMovie|1<<32|uint64(d.scene)).Uint64(), d.env.ReduceMotion)
	}
	if !d.board {
		d.board = true
		outs = append(outs, proto.SetLayout{Layout: proto.LayoutFull})
	}
	switch {
	case !s.Play:
		d.film.Skip(s.At)
	case d.env.Instant:
		outs = append(outs, strip(d.film.Run(s.At))...)
	default:
		d.phase, d.target = animating, s.At
		return append(outs, proto.Redraw{}, proto.Animate{Every: gtw.FilmFrame})
	}
	return append(outs, proto.Redraw{}, d.animate(), proto.Drain{})
}

// tick moves the board on: a playing Board step's war, or the climax's code search.
func (d *Director) tick(ev proto.TickEvent) []proto.Output {
	if d.film == nil || !d.board {
		return nil
	}
	to := d.target
	if to == 0 {
		to = d.film.Stage()
	}
	lines, done := d.film.Advance(ev.Dt, to)
	outs := append(strip(lines), proto.Redraw{})
	if d.phase == animating && done {
		d.phase, d.target = playing, 0
		outs = append(outs, d.animate(), proto.Drain{})
	}
	return outs
}

// animate keeps the clock running while the board moves by itself: at the climax, WOPR
// searches for the launch code.
func (d *Director) animate() proto.Output {
	if d.board && d.film.Stage() == gtw.FilmClimax && !d.env.Instant {
		return proto.Animate{Every: gtw.FilmFrame}
	}
	return proto.Animate{}
}

func (d *Director) hideBoard() []proto.Output {
	if !d.board {
		return nil
	}
	d.board = false
	return []proto.Output{proto.Animate{}, proto.SetLayout{Layout: proto.LayoutConsole}}
}

// strip prints the board's strip lines at table pace, as the game does.
func strip(lines []string) []proto.Output {
	if len(lines) == 0 {
		return nil
	}
	return []proto.Output{proto.Say{Lines: lines, Pace: proto.PaceTable}}
}

// run launches a Run step's program. The board makes way for it; an ending launched from the
// climax board carries on cracking the launch code from where the board left it.
func (d *Director) run(s scenes.Run) []proto.Output {
	mode := s.Mode
	if mode == ending.MovieMode && d.film != nil && d.film.Stage() == gtw.FilmClimax {
		mode += ":" + strconv.Itoa(d.film.Cracked())
	}
	outs := d.hideBoard()
	d.phase = running
	return append(outs, proto.Launch{Slug: s.Slug, Mode: mode})
}

// key is the controls table (docs/PLAN.md §7). Ctrl+C is the host's.
func (d *Director) key(ev proto.KeyEvent) []proto.Output {
	if d.phase != playing && d.phase != animating {
		return nil
	}
	r := ev.Rune
	if ev.Key != proto.KeyRune {
		r = 0
	}
	switch {
	case r == ' ':
		if d.paused {
			return d.unpause()
		}
		d.paused = true
		return []proto.Output{proto.Hold{On: true}, proto.AwaitKeys{Hint: linePaused[0].Text, Capture: true}}
	case ev.Key == proto.KeyRight || r == 'n' || r == 'N':
		if d.opts.Single || d.scene+1 >= len(d.scenes) {
			return d.cut(d.end())
		}
		return d.cut(d.begin(d.scene + 1))
	case ev.Key == proto.KeyLeft || r == 'p' || r == 'P':
		return d.cut(d.begin(max(d.scene-1, 0)))
	case ev.Key == proto.KeyEsc:
		return d.cut(d.menu())
	}
	return nil
}

// unpause releases a pause; the notice goes with it.
func (d *Director) unpause() []proto.Output {
	if !d.paused {
		return nil
	}
	d.paused = false
	return []proto.Output{proto.Hold{}, proto.AwaitKeys{Capture: true}}
}

// cut drops what is left of the scene on screen before then: the pause ends, the board stops,
// and what is queued is revealed at once.
func (d *Director) cut(then []proto.Output) []proto.Output {
	outs := append(d.unpause(), proto.Skip{}, proto.Animate{})
	return append(outs, then...)
}

// menu is the scene menu: the list, the keys, and a prompt for a number or a name.
func (d *Director) menu() []proto.Output {
	d.phase, d.menued = inMenu, true
	d.film, d.target = nil, 0
	outs := d.hideBoard()
	outs = append(outs, proto.Animate{}, proto.SetLayout{Layout: proto.LayoutConsole}, proto.Clear{})
	lines := lineMenuTitle.Texts()
	for _, s := range infos(d.scenes) {
		lines = append(lines, fmt.Sprintf("  %d.  %-16s %s", s.Number, s.Title, s.Blurb))
	}
	lines = append(lines, lineMenuKeys.Texts()...)
	return append(outs, proto.Say{Lines: lines, Pace: proto.PaceTable}, proto.Prompt{Text: promptScene[0].Text})
}

// choose reads the menu's line: a scene to play from, or Q to leave. The host itself ends the
// session on LOGOFF, EXIT or QUIT.
func (d *Director) choose(text string) []proto.Output {
	if d.phase != inMenu {
		return nil
	}
	again := []proto.Output{proto.Say{Lines: []string{""}, Pace: proto.PaceInstant}, proto.Prompt{Text: promptScene[0].Text}}
	switch strings.ToUpper(strings.TrimSpace(text)) {
	case "":
		return again[1:]
	case "Q":
		d.phase = ended
		return []proto.Output{proto.Quit{}}
	}
	list := infos(d.scenes)
	i, err := resolve(list, text)
	var amb *AmbiguousError
	switch {
	case err == nil:
		return d.begin(i)
	case errors.As(err, &amb):
		names := make([]string, len(amb.Candidates))
		for j, c := range amb.Candidates {
			names[j] = fmt.Sprintf("  %d.  %s", c.Number, c.Title)
		}
		return append([]proto.Output{say(lineAmbiguous.Texts()...), proto.Say{Lines: names, Pace: proto.PaceTable}}, again...)
	}
	return append([]proto.Output{say(lineNoScene.Texts()...)}, again...)
}

func say(lines ...string) proto.Output { return proto.Say{Lines: lines, Pace: proto.PaceSpeech} }
