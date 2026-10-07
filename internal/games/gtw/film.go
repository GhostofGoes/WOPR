package gtw

import (
	"time"

	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// FilmSide is the film's choice at the side prompt, the Soviet Union (docs/PLAN.md §6.2).
// Movie mode types it and FilmTargets, and Film plays them.
const FilmSide = "2"

// FilmTargets are the film's targets, typed one to a line.
var FilmTargets = []string{"Las Vegas", "Seattle"}

// FilmFrame is the board's animation cadence, for a program that steps a Film on its ticks.
const FilmFrame = frame

// FilmStage is a point in the film's war that movie mode's Board steps show (docs/PLAN.md
// §7). The stages come in order.
type FilmStage uint8

// Stages.
const (
	FilmStrike1 FilmStage = iota + 1 // the first strike has landed (DEFCON 4); strike 2's prompt is next
	FilmStrike2                      // strike 2, on WOPR's war plan, has landed (DEFCON 3)
	FilmRatios                       // the war is over and the projected kill ratios are up
	FilmClimax                       // DEFCON 1: WOPR searches for the launch code
)

// Film is the film's scenario as movie mode shows it: the real game, given the film's side and
// targets and then WOPR's war plan at every strike, played a frame at a time. The board, the
// strip lines and the kill ratios are therefore the game's own. Film is not a program: the
// movie's director draws it and asks it for frames.
type Film struct {
	g       *Game
	stage   FilmStage // the furthest stage reached
	pending []string  // strip lines printed before the first frame (the first strike's launch)
	prompt  string    // the last prompt the game asked
	acc     time.Duration
	codeAcc time.Duration
}

// NewFilm sets up the film's war with seed: side and targets chosen, the board open and the
// first strike leaving. reduceMotion stops the DEFCON 1 blink, as --reduce-motion does.
func NewFilm(seed uint64, reduceMotion bool) *Film {
	g := New().(*Game)
	g.Start(proto.Env{Seed: seed, Width: 80, Height: 19, ReduceMotion: reduceMotion}) // paced: Film steps frames itself
	g.Handle(proto.LineEvent{Text: FilmSide})
	for _, t := range FilmTargets {
		g.Handle(proto.LineEvent{Text: t})
	}
	f := &Film{g: g}
	f.pending = f.take(g.Handle(proto.LineEvent{Text: ""}))
	return f
}

// SideChoice is the side-choice picture GTW prints first: the two nations' outlines (original
// art), each named beneath (film text).
func SideChoice() script.Ls {
	rows := sideChoice()
	out := script.Orig(rows[:len(rows)-1]...)
	return append(out, script.Recon(rows[len(rows)-1])...)
}

// Stage is the furthest stage the film has reached; 0 before the first strike lands.
func (f *Film) Stage() FilmStage { return f.stage }

// Prompt is the last prompt the game asked: at FilmStrike1, the second strike's.
func (f *Film) Prompt() string { return f.prompt }

// Cracked is how many characters of the launch code the climax board shows.
func (f *Film) Cracked() int { return f.g.cracked }

// View draws the board, or the kill ratios once they are up.
func (f *Film) View(c *proto.Canvas) { f.g.View(c) }

// Advance moves the film on by dt toward stage, a frame per FilmFrame, and returns the strip
// lines printed on the way and whether the film is at stage. At the climax, time moves WOPR's
// search for the launch code on.
func (f *Film) Advance(dt time.Duration, to FilmStage) ([]string, bool) {
	lines := f.flush()
	if f.stage >= to {
		f.search(dt)
		return lines, true
	}
	f.acc += dt
	for f.acc >= FilmFrame && f.stage < to {
		f.acc -= FilmFrame
		lines = append(lines, f.step()...)
	}
	return lines, f.stage >= to
}

// Run plays the film to stage at once and returns every strip line on the way (movie mode
// under --instant).
func (f *Film) Run(to FilmStage) []string {
	lines := f.flush()
	for f.stage < to {
		lines = append(lines, f.step()...)
	}
	return lines
}

// Skip goes to stage at once, printing nothing: a scene that joins the war later.
func (f *Film) Skip(to FilmStage) { f.Run(to) }

func (f *Film) flush() []string {
	lines := f.pending
	f.pending = nil
	return lines
}

// step takes the war one step toward the next stage: a frame in flight, Enter at a strike
// prompt (WOPR's war plan), Enter for the kill ratios, Enter for the climax.
func (f *Film) step() []string {
	var outs []proto.Output
	switch f.g.phase {
	case flight:
		outs = f.g.run(1)
	case orders:
		outs = f.g.onOrder("")
	case assessed:
		outs = f.g.showRatios()
	case ratios:
		outs = f.g.startClimax()
	case chooseSide, listTargets, climax:
	}
	lines := f.take(outs)
	switch {
	case f.g.phase == climax:
		f.stage = FilmClimax
	case f.g.phase == ratios:
		f.stage = FilmRatios
	case f.g.phase == orders && f.g.stage >= 2:
		f.stage = FilmStrike2
	case f.g.phase == orders:
		f.stage = FilmStrike1
	}
	return lines
}

// search moves the climax's code search on by dt, at the game's own pace.
func (f *Film) search(dt time.Duration) {
	if f.g.phase != climax {
		return
	}
	f.codeAcc += dt
	for f.codeAcc >= codeFrame {
		f.codeAcc -= codeFrame
		f.g.climaxT++
	}
	f.g.cracked = min(f.g.climaxT/codeEvery, len(lineCode[0].Text)-codeHeldBack)
}

// take keeps the lines a step printed and the prompt it asked; the rest is the game's own
// pacing, which the film replaces.
func (f *Film) take(outs []proto.Output) []string {
	var lines []string
	for _, o := range outs {
		switch o := o.(type) {
		case proto.Say:
			lines = append(lines, o.Lines...)
		case proto.Prompt:
			f.prompt = o.Text
		}
	}
	return lines
}
