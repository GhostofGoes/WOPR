package host

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/GhostofGoes/WOPR/internal/proto"
)

// fake is a scriptable program that records the events it receives.
type fake struct {
	name   string
	start  []proto.Output
	on     func(proto.Event) []proto.Output
	events []proto.Event
	env    proto.Env
}

func (f *fake) Start(e proto.Env) []proto.Output { f.env = e; return f.start }
func (f *fake) Handle(e proto.Event) []proto.Output {
	f.events = append(f.events, e)
	if f.on != nil {
		return f.on(e)
	}
	return nil
}
func (f *fake) View(*proto.Canvas) {}

func last(events []proto.Event) proto.Event {
	if len(events) == 0 {
		return nil
	}
	return events[len(events)-1]
}

func has[T Effect](effects []Effect) bool {
	for _, e := range effects {
		if _, ok := e.(T); ok {
			return true
		}
	}
	return false
}

func newRunner(games map[string]*fake, det bool) *Runner {
	return New(Config{
		Seed: 7, Deterministic: det,
		Resolve: func(l proto.Launch) (proto.Program, Placement, error) {
			g, ok := games[l.Slug]
			if !ok {
				return nil, Placement{}, errors.New("unknown")
			}
			return g, Placement{Layout: proto.LayoutPanel, PanelRows: 10}, nil
		},
		Area: func(p Placement) (int, int) {
			if p.Layout == proto.LayoutPanel {
				return 80, p.PanelRows
			}
			return 80, 24
		},
	})
}

func TestLineRoutingAndLaunch(t *testing.T) {
	t.Parallel()
	chess := &fake{name: "chess", start: []proto.Output{proto.Say{Lines: []string{"CHESS"}}, proto.Prompt{Text: "YOUR MOVE: "}}}
	root := &fake{start: []proto.Output{proto.Prompt{}}}
	root.on = func(e proto.Event) []proto.Output {
		switch e := e.(type) {
		case proto.LineEvent:
			if e.Text == "7" {
				return []proto.Output{proto.Launch{Slug: "chess", Mode: "x"}}
			}
		case proto.GameOver:
			return []proto.Output{proto.Say{Lines: []string{"VERDICT"}}, proto.Prompt{}}
		}
		return nil
	}
	r := newRunner(map[string]*fake{"chess": chess}, false)
	if eff := r.Start(root, Placement{}); !has[AskLine](eff) {
		t.Fatalf("root prompt not asked: %v", eff)
	}
	eff := r.Line("7")
	if r.Depth() != 2 || chess.env.Mode != "x" || chess.env.Height != 10 {
		t.Fatalf("launch: depth %d env %+v", r.Depth(), chess.env)
	}
	if !has[Relayout](eff) || !has[Print](eff) || !has[AskLine](eff) {
		t.Fatalf("launch effects: %#v", eff)
	}
	r.Line("e2e4")
	if !reflect.DeepEqual(last(chess.events), proto.LineEvent{Text: "e2e4"}) {
		t.Fatalf("line not routed to the game: %v", chess.events)
	}
	if eff := r.Line("again"); len(eff) != 0 || len(chess.events) != 1 {
		t.Fatal("a line without a pending Prompt must be ignored")
	}
}

func TestDoneNextHandOffAndGameOver(t *testing.T) {
	t.Parallel()
	ending := &fake{}
	ending.on = func(proto.Event) []proto.Output {
		return []proto.Output{proto.Done{Result: proto.Result{Outcome: proto.NoWinner, NoVerdict: true}}}
	}
	ttt := &fake{start: []proto.Output{proto.Prompt{}}}
	ttt.on = func(proto.Event) []proto.Output {
		return []proto.Output{proto.Done{Result: proto.Result{Outcome: proto.NoWinner, Next: &proto.Launch{Slug: "ending"}}}}
	}
	root := &fake{start: []proto.Output{proto.Launch{Slug: "ttt"}}}
	r := newRunner(map[string]*fake{"ttt": ttt, "ending": ending}, false)
	r.Start(root, Placement{})
	ending.start = []proto.Output{proto.Prompt{}}
	r.Line("0") // ttt hands off to ending
	if len(root.events) != 0 || r.Depth() != 2 {
		t.Fatalf("hand-off must not report to the root: %v depth %d", root.events, r.Depth())
	}
	r.Line("Hello.")
	got, ok := last(root.events).(proto.GameOver)
	if !ok || !got.Result.NoVerdict || r.Depth() != 1 {
		t.Fatalf("final GameOver: %v depth %d", root.events, r.Depth())
	}
}

func TestEscArmConfirmAndExpire(t *testing.T) {
	t.Parallel()
	game := &fake{start: []proto.Output{proto.Think{Budget: time.Second}}}
	root := &fake{start: []proto.Output{proto.Launch{Slug: "g"}}}
	r := newRunner(map[string]*fake{"g": game}, false)
	r.Start(root, Placement{})
	if eff := r.Esc(); !has[Notice](eff) || r.Depth() != 2 {
		t.Fatal("first Esc must only arm")
	}
	r.Advance(EscWindow + time.Millisecond)
	if eff := r.Esc(); r.Depth() != 2 || !has[Notice](eff) {
		t.Fatal("after the window expires, Esc must arm again, not abort")
	}
	eff := r.Esc()
	if r.Depth() != 1 || !has[CancelThink](eff) {
		t.Fatalf("second Esc must abort and cancel the think: %#v", eff)
	}
	if over, ok := last(root.events).(proto.GameOver); !ok || over.Result.Outcome != proto.Aborted {
		t.Fatalf("root must hear Aborted: %v", root.events)
	}
}

// A NoAbort program (the film's ending) ignores Esc: no notice, nothing popped.
func TestEscCannotEndANoAbortProgram(t *testing.T) {
	t.Parallel()
	// The program changes its layout: NoAbort must survive that.
	game := &fake{start: []proto.Output{proto.SetLayout{Layout: proto.LayoutConsole}, proto.Prompt{}}}
	r := New(Config{
		Resolve: func(proto.Launch) (proto.Program, Placement, error) {
			return game, Placement{Layout: proto.LayoutFull, NoAbort: true}, nil
		},
		Area: func(Placement) (int, int) { return 80, 20 },
	})
	r.Start(&fake{start: []proto.Output{proto.Launch{Slug: "ending"}}}, Placement{})
	for range 3 {
		if eff := r.Esc(); len(eff) != 0 {
			t.Fatalf("Esc in a NoAbort program: %v", eff)
		}
	}
	if r.Depth() != 2 {
		t.Fatal("the program must still be running")
	}
}

func TestEscCancelsBrainAtRoot(t *testing.T) {
	t.Parallel()
	root := &fake{start: []proto.Output{proto.Think{Budget: 20 * time.Second}}}
	r := newRunner(nil, false)
	eff := r.Start(root, Placement{})
	gen := eff[0].(StartThink).Gen
	eff = r.Esc()
	if !has[CancelThink](eff) {
		t.Fatal("Esc must cancel the pending reply")
	}
	if done, ok := last(root.events).(proto.ThinkDone); !ok || !errors.Is(done.Err, proto.ErrCanceled) {
		t.Fatalf("root must hear ErrCanceled: %v", root.events)
	}
	if eff := r.ThinkResult(gen, "late", nil); eff != nil || len(root.events) != 1 {
		t.Fatal("a late result for a cancelled think must be dropped")
	}
}

func TestThinkDeadlines(t *testing.T) {
	t.Parallel()
	limited := proto.Think{Limit: proto.Limit{MaxDepth: 3}, Budget: time.Second}
	open := proto.Think{Budget: 2 * time.Second}
	if d := newRunner(nil, true).deadline(limited); d != SafetyCap {
		t.Errorf("deterministic + Limit: deadline %v, want the safety cap", d)
	}
	if d := newRunner(nil, false).deadline(limited); d != time.Second {
		t.Errorf("interactive: deadline %v, want Budget", d)
	}
	if d := newRunner(nil, true).deadline(open); d != 2*time.Second {
		t.Errorf("no Limit: deadline %v, want Budget", d)
	}
}

func TestAnimateCadenceCoalesces(t *testing.T) {
	t.Parallel()
	root := &fake{start: []proto.Output{proto.Animate{Every: 500 * time.Millisecond}}}
	r := newRunner(nil, false)
	r.Start(root, Placement{})
	r.Advance(200 * time.Millisecond)
	r.Advance(200 * time.Millisecond)
	if len(root.events) != 0 {
		t.Fatal("no tick before the cadence")
	}
	r.Advance(200 * time.Millisecond)
	if tick, ok := last(root.events).(proto.TickEvent); !ok || tick.Dt != 600*time.Millisecond {
		t.Fatalf("want one coalesced 600ms tick, got %v", root.events)
	}
	if !r.NeedsTicks() {
		t.Error("an animating program needs ticks")
	}
}

func TestHostCommandsAndLayout(t *testing.T) {
	t.Parallel()
	game := &fake{start: []proto.Output{proto.Prompt{}}}
	root := &fake{start: []proto.Output{proto.Launch{Slug: "g"}}}
	r := newRunner(map[string]*fake{"g": game}, false)
	r.Start(root, Placement{})
	if eff := r.Line("  log   off "); !has[Exit](eff) {
		t.Fatal("LOGOFF must exit from inside a game")
	}
	if eff := r.Line("anything"); eff != nil {
		t.Fatal("nothing runs after Exit")
	}
	for _, in := range []string{"Logoff.", "quit!", "Log off.", "EXIT"} {
		r := newRunner(nil, false)
		r.Start(&fake{start: []proto.Output{proto.Prompt{}}}, Placement{})
		if eff := r.Line(in); !has[Exit](eff) {
			t.Errorf("%q must exit: punctuation does not hide a host command", in)
		}
	}
	r = newRunner(nil, false)
	r.Start(&fake{start: []proto.Output{proto.Prompt{}}}, Placement{})
	if eff := r.Line("quite"); has[Exit](eff) {
		t.Error("QUITE is not QUIT")
	}

	g2 := &fake{start: []proto.Output{proto.SetLayout{Layout: proto.LayoutFull}}}
	r = newRunner(map[string]*fake{"g": g2}, false)
	eff := r.Start(&fake{start: []proto.Output{proto.Launch{Slug: "g"}}}, Placement{})
	if !has[Relayout](eff) {
		t.Fatal("SetLayout must relayout")
	}
	if rs, ok := last(g2.events).(proto.ResizeEvent); !ok || rs.Height != 24 {
		t.Fatalf("SetLayout must send the new area: %v", g2.events)
	}
}

func TestInstantDropsWaits(t *testing.T) {
	t.Parallel()
	root := &fake{start: []proto.Output{proto.Wait{D: time.Second}, proto.Clear{}}}
	r := New(Config{Instant: true})
	eff := r.Start(root, Placement{})
	if has[Pause](eff) || !has[PageBreak](eff) {
		t.Fatalf("instant: %#v", eff)
	}
}

// Each launch gets its own seed, derived from the session seed, the slug and the play
// index, so games and replays do not share a random sequence but a session reproduces.
func TestLaunchSeeds(t *testing.T) {
	t.Parallel()
	seeds := func() []uint64 {
		chess, checkers := &fake{start: []proto.Output{proto.Prompt{}}}, &fake{start: []proto.Output{proto.Prompt{}}}
		quit := func(proto.Event) []proto.Output {
			return []proto.Output{proto.Done{Result: proto.Result{Outcome: proto.Draw}}}
		}
		chess.on, checkers.on = quit, quit
		root := &fake{start: []proto.Output{proto.Prompt{}}}
		root.on = func(e proto.Event) []proto.Output {
			if l, ok := e.(proto.LineEvent); ok {
				return []proto.Output{proto.Launch{Slug: l.Text}}
			}
			return []proto.Output{proto.Prompt{}}
		}
		r := newRunner(map[string]*fake{"chess": chess, "checkers": checkers}, true)
		r.Start(root, Placement{})
		var got []uint64
		for _, slug := range []string{"chess", "checkers", "chess"} {
			r.Line(slug)
			if slug == "chess" {
				got = append(got, chess.env.Seed)
			} else {
				got = append(got, checkers.env.Seed)
			}
			r.Line("over")
		}
		return append(got, root.env.Seed)
	}
	a, b := seeds(), seeds()
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("seeds differ between identical sessions: %v vs %v", a, b)
	}
	if a[3] != 7 {
		t.Errorf("the root keeps the session seed, got %d", a[3])
	}
	if a[0] == a[1] || a[0] == a[2] || a[1] == a[2] || a[0] == 7 {
		t.Errorf("launch seeds must differ per game and per play: %v", a)
	}
}

// A hand-off to a program that cannot be built must still end the game for the program
// below; otherwise it would wait for a GameOver that never comes.
func TestHandOffToMissingProgram(t *testing.T) {
	t.Parallel()
	ttt := &fake{start: []proto.Output{proto.Prompt{}}}
	ttt.on = func(proto.Event) []proto.Output {
		return []proto.Output{proto.Done{Result: proto.Result{Outcome: proto.NoWinner, Next: &proto.Launch{Slug: "ending"}}}}
	}
	root := &fake{start: []proto.Output{proto.Launch{Slug: "ttt"}}}
	r := newRunner(map[string]*fake{"ttt": ttt}, false)
	r.Start(root, Placement{})
	eff := r.Line("0")
	got, ok := last(root.events).(proto.GameOver)
	if !ok || got.Result.Next != nil || got.Result.Outcome != proto.NoWinner || r.Depth() != 1 {
		t.Fatalf("root events %v, depth %d", root.events, r.Depth())
	}
	if !has[Print](eff) || !has[Relayout](eff) {
		t.Errorf("want the not-available line and a relayout: %#v", eff)
	}
}

// A program with its own screen (Panel or Full) starts on a new console page, right after
// its relayout and before its first output, whether the persona launched it or a game
// handed off to it. A console game continues the page, and a game ending keeps its page,
// so its last lines stay above the verdict.
func TestLaunchWithAScreenStartsANewPage(t *testing.T) {
	t.Parallel()
	place := map[string]Placement{
		"maze":   {Layout: proto.LayoutPanel, PanelRows: 12},
		"cards":  {Layout: proto.LayoutConsole},
		"ttt":    {Layout: proto.LayoutPanel, PanelRows: 12},
		"ending": {Layout: proto.LayoutFull, NoAbort: true},
	}
	progs := map[string]*fake{}
	for slug := range place {
		progs[slug] = &fake{start: []proto.Output{proto.Say{Lines: []string{slug}}, proto.Prompt{}}}
	}
	over := func(proto.Event) []proto.Output {
		return []proto.Output{proto.Done{Result: proto.Result{Outcome: proto.Win}}}
	}
	progs["maze"].on, progs["cards"].on = over, over
	progs["ttt"].on = func(proto.Event) []proto.Output {
		return []proto.Output{proto.Done{Result: proto.Result{Outcome: proto.NoWinner, Next: &proto.Launch{Slug: "ending"}}}}
	}
	root := &fake{start: []proto.Output{proto.Prompt{}}}
	root.on = func(e proto.Event) []proto.Output {
		if l, ok := e.(proto.LineEvent); ok {
			return []proto.Output{proto.Launch{Slug: l.Text}}
		}
		return []proto.Output{proto.Say{Lines: []string{"VERDICT"}}, proto.Prompt{}}
	}
	r := New(Config{Resolve: func(l proto.Launch) (proto.Program, Placement, error) {
		return progs[l.Slug], place[l.Slug], nil
	}})
	r.Start(root, Placement{})
	opened := func(slug string) []Effect {
		return []Effect{
			Relayout{Placement: place[slug], NewProgram: true},
			PageBreak{},
			Print{Lines: []string{slug}},
			AskLine{},
		}
	}

	if eff := r.Line("maze"); !reflect.DeepEqual(eff, opened("maze")) {
		t.Errorf("a panel launch: %#v", eff)
	}
	if eff := r.Line("exit found"); has[PageBreak](eff) || !has[Print](eff) {
		t.Errorf("a game ending keeps its page for the verdict: %#v", eff)
	}
	if eff := r.Line("cards"); has[PageBreak](eff) {
		t.Errorf("a console game continues the page: %#v", eff)
	}
	r.Line("stand")
	if eff := r.Line("ttt"); !reflect.DeepEqual(eff, opened("ttt")) {
		t.Errorf("a panel launch after a console game: %#v", eff)
	}
	if eff := r.Line("0"); !reflect.DeepEqual(eff, opened("ending")) {
		t.Errorf("a hand-off to a full-screen program: %#v", eff)
	}
}
