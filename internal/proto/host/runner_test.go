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
	if r.Depth() != 2 || chess.env.Mode != "x" || chess.env.Seed != 7 || chess.env.Height != 10 {
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
