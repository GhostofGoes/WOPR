package host

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/GhostofGoes/WOPR/internal/proto"
)

// The movie-mode hooks (docs/PLAN.md §7, AR-7): key capture, Hold, Drain and Skip.

// A root that opts in captures every key, Esc included; a launched game cannot, and while
// a capturing root's game runs, the root is shielded.
func TestCaptureIsForTheRootOnly(t *testing.T) {
	t.Parallel()
	game := &fake{start: []proto.Output{proto.AwaitKeys{Capture: true}}}
	root := &fake{start: []proto.Output{proto.AwaitKeys{Hint: "KEYS", Capture: true}}}
	root.on = func(e proto.Event) []proto.Output {
		if k, ok := e.(proto.KeyEvent); ok && k.Key == proto.KeyRune && k.Rune == 'g' {
			return []proto.Output{proto.Launch{Slug: "g"}}
		}
		return nil
	}
	r := newRunner(map[string]*fake{"g": game}, false)
	r.Start(root, Placement{})
	if !r.Captures() || r.Shielded() {
		t.Fatalf("the root asked to capture: captures %v shielded %v", r.Captures(), r.Shielded())
	}
	r.Key(proto.KeyEsc, 0)
	if got, ok := last(root.events).(proto.KeyEvent); !ok || got.Key != proto.KeyEsc {
		t.Fatalf("a capturing root receives Esc as a key: %v", root.events)
	}
	if eff := r.Esc(); len(eff) != 0 {
		t.Errorf("the host's Esc machine has nothing to do at the root: %v", eff)
	}
	r.Key(proto.KeyRune, 'g')
	if r.Depth() != 2 || r.Captures() || !r.Shielded() {
		t.Fatalf("a launched game cannot capture, and shields its root: depth %d captures %v shielded %v",
			r.Depth(), r.Captures(), r.Shielded())
	}
	r.Key(proto.KeyEsc, 0)
	if len(game.events) != 0 {
		t.Errorf("Esc is never a key for a launched game: %v", game.events)
	}
	if eff := r.Esc(); !has[Notice](eff) {
		t.Error("in a launched game, Esc is the host's again")
	}
}

// A key dropped while shielded brings up a notice saying why, which clears after the Esc
// window or when the game ends.
func TestRefusedKeyNotice(t *testing.T) {
	t.Parallel()
	game := &fake{start: []proto.Output{proto.Prompt{}}}
	game.on = func(e proto.Event) []proto.Output {
		if _, ok := e.(proto.LineEvent); ok {
			return []proto.Output{proto.Done{}}
		}
		return nil
	}
	root := &fake{start: []proto.Output{proto.AwaitKeys{Capture: true}}}
	r := newRunner(map[string]*fake{"g": game}, false)
	r.Start(root, Placement{})
	if eff := r.Refused(); eff != nil {
		t.Fatalf("nothing is refused unshielded: %#v", eff)
	}
	root.on = func(proto.Event) []proto.Output { return []proto.Output{proto.Launch{Slug: "g"}} }
	r.Key(proto.KeyRune, 'g')
	if eff := r.Refused(); len(eff) != 1 || eff[0] != (Notice{Text: ShieldedNotice}) || !r.NeedsTicks() {
		t.Fatalf("a refused key: %#v", eff)
	}
	if eff := r.Advance(EscWindow - time.Millisecond); has[Notice](eff) {
		t.Fatal("the notice stays for the Esc window")
	}
	if eff := r.Advance(time.Millisecond); !has[Notice](eff) || r.NeedsTicks() {
		t.Fatalf("then it clears: %#v", eff)
	}
	r.Refused()
	if eff := r.Line("done"); !has[Notice](eff) || r.NeedsTicks() {
		t.Fatalf("the notice ends with the game: %#v", eff)
	}
}

// A Prompt ends the capture.
func TestPromptEndsCapture(t *testing.T) {
	t.Parallel()
	root := &fake{start: []proto.Output{proto.AwaitKeys{Capture: true}}}
	root.on = func(e proto.Event) []proto.Output {
		if k, ok := e.(proto.KeyEvent); ok && k.Rune == 'm' {
			return []proto.Output{proto.Prompt{Text: "SCENE: "}}
		}
		return nil
	}
	r := newRunner(nil, false)
	r.Start(root, Placement{})
	r.Key(proto.KeyRune, 'm')
	if r.Captures() || r.KeyMode() {
		t.Fatal("a Prompt switches to line mode and ends the capture")
	}
	r.Key(proto.KeyEsc, 0)
	if len(root.events) != 1 {
		t.Errorf("Esc is not a key once the capture ends: %v", root.events)
	}
}

// Hold freezes Animate (and so the Esc window) until released, reports only changes, and
// comes only from a root that captures keys. It never outlives its program's turn at the top:
// launching another program ends it.
func TestHoldFreezesTime(t *testing.T) {
	t.Parallel()
	game := &fake{start: []proto.Output{proto.Hold{On: true}, proto.Prompt{}}}
	root := &fake{start: []proto.Output{
		proto.AwaitKeys{Capture: true}, proto.Animate{Every: 100 * time.Millisecond}, proto.Hold{On: true},
	}}
	root.on = func(e proto.Event) []proto.Output {
		k, ok := e.(proto.KeyEvent)
		if !ok {
			return nil
		}
		switch k.Rune {
		case 'r':
			return []proto.Output{proto.Hold{}}
		case 'h':
			return []proto.Output{proto.Hold{On: true}, proto.Hold{On: true}}
		case 'g':
			return []proto.Output{proto.Launch{Slug: "g"}}
		}
		return nil
	}
	r := newRunner(map[string]*fake{"g": game}, false)
	if eff := r.Start(root, Placement{}); !has[Hold](eff) || !r.Held() {
		t.Fatalf("Hold must be reported: %#v", eff)
	}
	if r.NeedsTicks() {
		t.Error("nothing needs ticks while held")
	}
	r.Advance(time.Second)
	if len(root.events) != 0 {
		t.Fatalf("no tick while held: %v", root.events)
	}
	if eff := r.Key(proto.KeyRune, 'r'); !has[Hold](eff) || r.Held() {
		t.Fatalf("release: %#v", eff)
	}
	r.Advance(150 * time.Millisecond)
	if _, ok := last(root.events).(proto.TickEvent); !ok {
		t.Fatalf("ticks resume: %v", root.events)
	}
	holds := 0
	for _, e := range r.Key(proto.KeyRune, 'h') {
		if _, ok := e.(Hold); ok {
			holds++
		}
	}
	if holds != 1 || !r.Held() {
		t.Errorf("only a change of hold is reported, got %d", holds)
	}
	eff := r.Key(proto.KeyRune, 'g')
	if !has[Hold](eff) || r.Held() || r.Depth() != 2 {
		t.Fatalf("launching ends the hold, and the launched game cannot hold: %#v", eff)
	}
	if eff[0] != (Hold{}) {
		t.Errorf("the hold ends before the launch: %#v", eff)
	}
}

// Hold from anything but a capturing root is ignored: nobody could be sure to release it.
func TestHoldIsForACapturingRoot(t *testing.T) {
	t.Parallel()
	for name, root := range map[string]*fake{
		"a root in line mode":       {start: []proto.Output{proto.Hold{On: true}, proto.Prompt{}}},
		"a root in plain key mode":  {start: []proto.Output{proto.AwaitKeys{}, proto.Hold{On: true}}},
		"a game a root launched":    {start: []proto.Output{proto.AwaitKeys{Capture: true}, proto.Launch{Slug: "g"}}},
		"a capture ended by Prompt": {start: []proto.Output{proto.AwaitKeys{Capture: true}, proto.Prompt{}, proto.Hold{On: true}}},
	} {
		game := &fake{start: []proto.Output{proto.AwaitKeys{Capture: true}, proto.Hold{On: true}, proto.Prompt{}}}
		r := newRunner(map[string]*fake{"g": game}, false)
		if eff := r.Start(root, Placement{}); has[Hold](eff) || r.Held() {
			t.Errorf("%s: %#v", name, eff)
		}
	}
}

// Drained answers only the running program's latest Drain.
func TestDrainAnswersTheLatestMarker(t *testing.T) {
	t.Parallel()
	root := &fake{start: []proto.Output{proto.Say{Lines: []string{"A"}}, proto.Drain{}}}
	root.on = func(e proto.Event) []proto.Output {
		if _, ok := e.(proto.Drained); ok {
			return []proto.Output{proto.Say{Lines: []string{"B"}}, proto.Drain{}, proto.Drain{}}
		}
		return nil
	}
	r := newRunner(nil, false)
	first := drainID(t, r.Start(root, Placement{}))
	eff := r.Drained(first)
	if _, ok := last(root.events).(proto.Drained); !ok || len(eff) != 3 {
		t.Fatalf("the marker is answered: events %v, effects %#v", root.events, eff)
	}
	older, newer := eff[1].(Drain).ID, eff[2].(Drain).ID
	if r.Drained(older) != nil || r.Drained(first) != nil || len(root.events) != 1 {
		t.Fatal("a replaced or answered marker is dropped")
	}
	r.Drained(newer)
	if len(root.events) != 2 {
		t.Fatal("the latest marker is answered")
	}

	// A program that has launched another does not hear its marker.
	game := &fake{start: []proto.Output{proto.Prompt{}}}
	launcher := &fake{start: []proto.Output{proto.Drain{}, proto.Launch{Slug: "g"}}}
	r = newRunner(map[string]*fake{"g": game}, false)
	if id := drainID(t, r.Start(launcher, Placement{})); r.Drained(id) != nil || len(launcher.events) != 0 {
		t.Error("a marker reached while another program runs is dropped")
	}
}

// A Drain marker reached while the program has a Think pending is answered after its
// ThinkDone, so the two arrive in the same order however fast the typewriter was: testkit runs
// a Think inline, --instant reveals before it ends, pacing may reveal after.
func TestDrainedFollowsThinkDone(t *testing.T) {
	t.Parallel()
	think := proto.Think{Fn: func(context.Context) (any, error) { return nil, nil }}
	// prog's first ThinkDone also outputs onThought.
	prog := func(onThought []proto.Output) *fake {
		f := &fake{start: []proto.Output{proto.Say{Lines: []string{"A"}}, proto.Drain{}, think}}
		f.on = func(e proto.Event) []proto.Output {
			switch e.(type) {
			case proto.Drained:
				return []proto.Output{proto.Say{Lines: []string{"DRAINED"}}}
			case proto.ThinkDone:
				outs := append([]proto.Output{proto.Say{Lines: []string{"THOUGHT"}}}, onThought...)
				onThought = nil
				return outs
			}
			return nil
		}
		return f
	}
	said := func(eff []Effect) []string {
		var out []string
		for _, e := range eff {
			if p, ok := e.(Print); ok {
				out = append(out, p.Lines...)
			}
		}
		return out
	}
	gen := func(eff []Effect) uint64 {
		for _, e := range eff {
			if st, ok := e.(StartThink); ok {
				return st.Gen
			}
		}
		t.Fatalf("no Think in %#v", eff)
		return 0
	}

	// The marker is reached first (instant), then the Think ends.
	r := newRunner(nil, false)
	eff := r.Start(prog(nil), Placement{})
	if r.Drained(drainID(t, eff)) != nil {
		t.Fatal("Drained must wait for the pending Think")
	}
	if got := said(r.ThinkResult(gen(eff), nil, nil)); strings.Join(got, "|") != "THOUGHT|DRAINED" {
		t.Errorf("after the Think: %q", got)
	}

	// The Think ends first (paced), then the marker is reached.
	r = newRunner(nil, false)
	eff = r.Start(prog(nil), Placement{})
	if got := said(r.ThinkResult(gen(eff), nil, nil)); strings.Join(got, "|") != "THOUGHT" {
		t.Errorf("the Think's answer: %q", got)
	}
	if got := said(r.Drained(drainID(t, eff))); strings.Join(got, "|") != "DRAINED" {
		t.Errorf("then the marker: %q", got)
	}

	// A newer Drain sent with ThinkDone replaces the reached one; another Think defers it again.
	r = newRunner(nil, false)
	eff = r.Start(prog([]proto.Output{proto.Drain{}}), Placement{})
	r.Drained(drainID(t, eff))
	if got := said(r.ThinkResult(gen(eff), nil, nil)); strings.Join(got, "|") != "THOUGHT" {
		t.Errorf("a replaced marker is dropped: %q", got)
	}
	r = newRunner(nil, false)
	eff = r.Start(prog([]proto.Output{think}), Placement{})
	r.Drained(drainID(t, eff))
	next := r.ThinkResult(gen(eff), nil, nil)
	if got := said(next); strings.Join(got, "|") != "THOUGHT" {
		t.Errorf("another Think defers the answer: %q", got)
	}
	if got := said(r.ThinkResult(gen(next), nil, nil)); strings.Join(got, "|") != "THOUGHT|DRAINED" {
		t.Errorf("after the second Think: %q", got)
	}

	// Esc cancelling a root's Think also delivers the marker after its ThinkDone.
	r = newRunner(nil, false)
	eff = r.Start(prog(nil), Placement{})
	r.Drained(drainID(t, eff))
	if got := said(r.Esc()); strings.Join(got, "|") != "THOUGHT|DRAINED" {
		t.Errorf("after a cancelled Think: %q", got)
	}
}

func drainID(t *testing.T, eff []Effect) uint64 {
	t.Helper()
	for _, e := range eff {
		if d, ok := e.(Drain); ok {
			return d.ID
		}
	}
	t.Fatalf("no Drain in %#v", eff)
	return 0
}

// Say.Open and Skip pass through to the caller.
func TestOpenLinesAndSkip(t *testing.T) {
	t.Parallel()
	root := &fake{start: []proto.Output{proto.Say{Lines: []string{"LOGON: "}, Open: true}, proto.Skip{}}}
	eff := New(Config{}).Start(root, Placement{})
	if p, ok := eff[0].(Print); !ok || !p.Open {
		t.Errorf("an open Say: %#v", eff)
	}
	if !has[Skip](eff) {
		t.Errorf("Skip: %#v", eff)
	}
}
