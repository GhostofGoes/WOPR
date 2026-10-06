package wopr

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/prompt"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// Timings for the film's pacing; Env.Instant removes them (the host drops Wait).
const (
	dialPause     = 1200 * time.Millisecond
	redialPause   = 600 * time.Millisecond
	backdoorPause = 1500 * time.Millisecond
	brainBudget   = 20 * time.Second
	historyLimit  = 20
	hintEvery     = 3
	offerAnyGame  = "*"
	gtwSlug       = "global-thermonuclear-war"
	chessSlug     = "chess"
	flagChessAsk  = "asked-chess-instead"
)

// Session is the conversation state. Only the persona's Start and Handle change it.
type Session struct {
	phase     Phase
	failures  int // failed LOGON attempts
	greetStep int
	said      map[string]bool
	flags     map[string]bool
	offer     string // slug WOPR just offered ("*" = any game); valid for one turn
	menuArmed bool   // LIST GAMES was shown and no other input followed
	turn      uint64
	seed      uint64
	last      *proto.Result
	history   []Exchange
	pending   string // input awaiting a brain reply
	playing   string // slug of the game launched last
}

// Options configure the persona.
type Options struct {
	Play string // start this game directly, skipping LOGON and the greeting (--play)
}

// Persona is WOPR.
type Persona struct {
	reg   *games.Registry
	brain Brain
	opts  Options
	s     Session
}

// New returns the persona.
func New(reg *games.Registry, brain Brain, opts Options) *Persona {
	if brain == nil {
		brain = NewScripted()
	}
	return &Persona{reg: reg, brain: brain, opts: opts, s: Session{said: map[string]bool{}, flags: map[string]bool{}}}
}

// Phase reports the conversation phase (for the ui and tests).
func (p *Persona) Phase() Phase { return p.s.phase }

// Start implements proto.Program.
func (p *Persona) Start(env proto.Env) []proto.Output {
	p.s.seed = env.Seed
	if p.opts.Play != "" {
		p.s.phase = PhaseShell
		p.s.greetStep = 3
		return p.launch(p.opts.Play)
	}
	return p.dial(dialPause)
}

// View implements proto.Program; the persona is console text only.
func (p *Persona) View(*proto.Canvas) {}

// Handle implements proto.Program.
func (p *Persona) Handle(ev proto.Event) []proto.Output {
	switch ev := ev.(type) {
	case proto.LineEvent:
		switch p.s.phase {
		case PhaseLogon:
			return p.logon(ev.Text)
		case PhaseGreeting:
			return p.greeting(ev.Text)
		default:
			return p.shell(ev.Text)
		}
	case proto.ThinkDone:
		return p.brainReply(ev)
	case proto.GameOver:
		return p.gameOver(ev.Result)
	}
	return nil
}

// --- output helpers ---

func say(ls ...Ls) proto.Output {
	var lines []string
	for _, l := range ls {
		lines = append(lines, l.Texts()...)
	}
	return proto.Say{Lines: lines, Pace: proto.PaceSpeech}
}

func sayText(lines ...string) proto.Output { return proto.Say{Lines: lines, Pace: proto.PaceSpeech} }

var blank = proto.Say{Lines: []string{""}, Pace: proto.PaceInstant}

// respond is WOPR's turn: its lines, a blank line, and the input prompt.
func respond(outs ...proto.Output) []proto.Output {
	return append(outs, blank, proto.Prompt{})
}

// --- dialing and LOGON (a closed table; docs/PLAN.md §4.2) ---

func (p *Persona) dial(pause time.Duration) []proto.Output {
	p.s.phase = PhaseLogon
	return []proto.Output{
		say(lineDialing),
		proto.Wait{D: pause},
		say(lineConnected), blank,
		proto.Prompt{Text: lineLogon.Text},
	}
}

func (p *Persona) logon(input string) []proto.Output {
	again := []proto.Output{blank, proto.Prompt{Text: lineLogon.Text}}
	switch prompt.Normalize(input) {
	case "":
		return []proto.Output{proto.Prompt{Text: lineLogon.Text}}
	case "HELP LOGON":
		return append([]proto.Output{say(lineHelpNA)}, again...)
	case "HELP GAMES":
		return append([]proto.Output{say(lineHelpGames)}, again...)
	case "LIST GAMES":
		return append([]proto.Output{p.listGames()}, again...)
	case "JOSHUA":
		return p.backdoor()
	}
	p.s.failures++
	outs := []proto.Output{say(lineNotRecog), proto.Wait{D: redialPause}, proto.Clear{}}
	if p.s.failures%hintEvery == 0 {
		outs = append(outs, say(lineLogonHint), blank)
	}
	return append(outs, p.dial(redialPause)...)
}

func (p *Persona) backdoor() []proto.Output {
	p.s.phase = PhaseGreeting
	p.s.greetStep = 0
	return []proto.Output{
		proto.Clear{},
		proto.Say{Lines: lineHeader.Texts(), Pace: proto.PaceTable},
		proto.Wait{D: backdoorPause},
		proto.Clear{},
		proto.Say{Lines: lineBurst.Texts(), Pace: proto.PaceTable},
		proto.Wait{D: backdoorPause},
		proto.Clear{},
		say(lineGreetings), blank,
		proto.Prompt{},
	}
}

// --- the greeting scene ---

func (p *Persona) greeting(input string) []proto.Output {
	if prompt.Normalize(input) == "" { // an empty Enter is not the player's turn
		return []proto.Output{proto.Prompt{}}
	}
	// A command or an explicit request ends the scene early.
	if isCommand(input) {
		p.endGreeting()
		return p.shell(input)
	}
	if _, ok := gameIntent(input, p.reg); ok {
		p.endGreeting()
		return p.shell(input)
	}
	p.s.greetStep++
	switch p.s.greetStep {
	case 1:
		return respond(say(lineHowFeeling))
	case 2:
		return respond(say(lineExcellent))
	default:
		p.endGreeting()
		p.s.offer = offerAnyGame
		return respond(say(lineYesTheyDo))
	}
}

func (p *Persona) endGreeting() {
	p.s.phase = PhaseShell
	p.s.greetStep = 3
}

// --- the shell ---

func isCommand(input string) bool {
	switch prompt.Normalize(input) {
	case "HELP", "HELP GAMES", "LIST GAMES", "HELP LOGON":
		return true
	}
	return false
}

func (p *Persona) shell(input string) []proto.Output {
	norm := prompt.Normalize(input)
	if norm == "" { // an empty Enter keeps a pending offer and a numbered list
		return []proto.Output{proto.Prompt{}}
	}
	offer := p.s.offer
	p.s.offer = ""
	armed := p.s.menuArmed
	p.s.menuArmed = false

	switch norm {
	case "HELP", "HELP LOGON":
		return respond(say(lineHelpShell))
	case "HELP GAMES":
		return respond(say(lineHelpGames))
	case "LIST GAMES":
		p.s.menuArmed = true
		return respond(p.listGames())
	}

	if armed {
		if n, ok := prompt.MenuChoice(input, len(p.reg.Listed())); ok {
			return p.request(p.reg.Listed()[n-1])
		}
	}
	if e, ok := gameIntent(input, p.reg); ok {
		return p.request(e)
	}
	if outs, ok := p.unknownPlay(input); ok {
		return outs
	}
	if offer != "" {
		switch prompt.YesNo(input) {
		case prompt.Yes:
			if offer == offerAnyGame {
				p.s.menuArmed = true
				return respond(say(lineWhichGame), p.listGames())
			}
			if e, ok := p.reg.Get(offer); ok {
				return p.request(e)
			}
		case prompt.No:
			return respond(say(lineDeclined))
		case prompt.Unclear:
		}
	}
	return p.askBrain(input)
}

// request handles an explicit request for a game, including the film's GTW-vs-chess scene.
func (p *Persona) request(e games.Entry) []proto.Output {
	if e.Info.Slug == gtwSlug && !p.s.flags[flagChessAsk] {
		p.s.flags[flagChessAsk] = true
		p.s.offer = chessSlug
		return respond(say(linePreferChess))
	}
	if e.Info.Slug == gtwSlug {
		return append([]proto.Output{say(lineFine), blank}, p.launch(e.Info.Slug)...)
	}
	return p.launch(e.Info.Slug)
}

// unknownPlay answers the PLAY command when its name matches no game, or several. Other
// phrasings with an unknown object ("HOW ABOUT THAT?") are conversation, for the brain.
func (p *Persona) unknownPlay(input string) ([]proto.Output, bool) {
	cs := prompt.Clauses(input)
	if len(cs) != 1 || len(cs[0].Words) < 2 || cs[0].Words[0] != "PLAY" {
		return nil, false
	}
	object := requestObject(cs[0].Words[1:])
	if object == "" {
		return nil, false
	}
	_, err := p.reg.Resolve(object)
	var amb *games.AmbiguousError
	switch {
	case errors.As(err, &amb):
		names := make([]string, len(amb.Candidates))
		for i, e := range amb.Candidates {
			names[i] = e.Info.Name
		}
		return respond(say(lineAmbiguous), proto.Say{Lines: names, Pace: proto.PaceTable}), true
	case errors.Is(err, games.ErrNotFound):
		return respond(say(lineNoSuchGame)), true
	}
	return nil, false
}

func (p *Persona) launch(slug string) []proto.Output {
	e, ok := p.reg.Get(slug)
	switch {
	case !ok || e.Info.Internal: // the ending is reached by a hand-off, never by name
		return respond(say(lineNoSuchGame))
	case e.Info.Status != games.Playable:
		return respond(say(lineNotAvail, lineNotYet))
	}
	p.s.playing = slug
	return []proto.Output{proto.Launch{Slug: slug}}
}

// listGames is the film's LIST GAMES screen: the names, with a blank line before the last.
func (p *Persona) listGames() proto.Output {
	listed := p.reg.Listed()
	lines := make([]string, 0, len(listed)+1)
	for i, e := range listed {
		if i == len(listed)-1 && i > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, e.Info.Name)
	}
	return proto.Say{Lines: lines, Pace: proto.PaceTable}
}

// --- the brain ---

func (p *Persona) askBrain(input string) []proto.Output {
	p.s.turn++
	p.s.pending = input
	snap := p.s.snapshot()
	brain := p.brain
	return []proto.Output{proto.Think{
		Fn:     func(ctx context.Context) (any, error) { return brain.Reply(ctx, snap, input) },
		Budget: brainBudget,
	}}
}

func (p *Persona) brainReply(done proto.ThinkDone) []proto.Output {
	input := p.s.pending
	p.s.pending = ""
	if done.Err != nil {
		if errors.Is(done.Err, proto.ErrCanceled) {
			return respond(say(lineCancelled))
		}
		return respond(say(lineRestate))
	}
	reply, ok := done.Value.(Reply)
	if !ok {
		return respond(say(lineRestate))
	}
	var outs []proto.Output
	if len(reply.Lines) > 0 {
		outs = append(outs, sayText(reply.Lines...))
	}
	for _, eff := range reply.Effects {
		switch eff := eff.(type) {
		case MarkSaid:
			p.s.said[eff.ID] = true
		case SetFlag:
			p.s.flags[eff.Key] = true
		case StartGame:
			if e, ok := p.reg.Get(eff.Slug); ok {
				p.record(input, reply.Lines)
				return append(outs, p.request(e)...)
			}
		}
	}
	p.record(input, reply.Lines)
	if len(reply.Lines) > 0 && reply.Lines[len(reply.Lines)-1] == lineShallWe[0].Text {
		p.s.offer = offerAnyGame
	}
	return respond(outs...)
}

func (p *Persona) record(in string, out []string) {
	p.s.history = append(p.s.history, Exchange{In: in, Out: out})
	if len(p.s.history) > historyLimit {
		p.s.history = p.s.history[len(p.s.history)-historyLimit:]
	}
}

// --- after a game ---

func (p *Persona) gameOver(res proto.Result) []proto.Output {
	r := res
	p.s.last = &r
	if res.NoVerdict {
		p.s.phase = PhaseShell
		p.s.offer = chessSlug
		return []proto.Output{blank, proto.Prompt{}}
	}
	var outs []proto.Output
	switch res.Outcome {
	case proto.Aborted:
		if p.s.playing == gtwSlug { // Esc twice walked away from the war (docs/PLAN.md §6.2)
			outs = append(outs, say(lineWarAbandoned))
			break
		}
		outs = append(outs, say(lineAborted))
	case proto.Win:
		outs = append(outs, say(lineWinUser))
	case proto.Loss:
		outs = append(outs, say(lineWinWOPR))
	case proto.Draw, proto.NoWinner:
		outs = append(outs, say(lineWinNone))
	}
	if len(res.Lines) > 0 {
		outs = append(outs, proto.Say{Lines: res.Lines, Pace: proto.PaceTable})
	}
	p.s.offer = offerAnyGame
	outs = append(outs, blank, say(lineAnother))
	return respond(outs...)
}

// String is for debugging.
func (p *Persona) String() string {
	return fmt.Sprintf("persona{phase=%d failures=%d offer=%q}", p.s.phase, p.s.failures, p.s.offer)
}
