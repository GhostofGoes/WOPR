package host

import (
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
// never outlives the program that set it.
func TestHoldFreezesTime(t *testing.T) {
	t.Parallel()
	game := &fake{start: []proto.Output{proto.Animate{Every: 100 * time.Millisecond}, proto.Hold{On: true}, proto.Prompt{}}}
	game.on = func(e proto.Event) []proto.Output {
		if l, ok := e.(proto.LineEvent); ok {
			switch l.Text {
			case "release":
				return []proto.Output{proto.Hold{}, proto.Prompt{}}
			case "hold":
				return []proto.Output{proto.Hold{On: true}, proto.Hold{On: true}, proto.Prompt{}}
			case "end":
				return []proto.Output{proto.Done{}}
			}
		}
		return nil
	}
	root := &fake{start: []proto.Output{proto.Launch{Slug: "g"}}}
	r := newRunner(map[string]*fake{"g": game}, false)
	if eff := r.Start(root, Placement{}); !has[Hold](eff) || !r.Held() {
		t.Fatalf("Hold must be reported: %#v", eff)
	}
	if r.NeedsTicks() {
		t.Error("nothing needs ticks while held")
	}
	r.Advance(time.Second)
	if len(game.events) != 0 {
		t.Fatalf("no tick while held: %v", game.events)
	}
	if eff := r.Line("release"); !has[Hold](eff) || r.Held() {
		t.Fatalf("release: %#v", eff)
	}
	r.Advance(150 * time.Millisecond)
	if _, ok := last(game.events).(proto.TickEvent); !ok {
		t.Fatalf("ticks resume: %v", game.events)
	}
	holds := 0
	for _, e := range r.Line("hold") {
		if _, ok := e.(Hold); ok {
			holds++
		}
	}
	if holds != 1 {
		t.Errorf("only a change of hold is reported, got %d", holds)
	}
	if eff := r.Line("end"); !has[Hold](eff) || r.Held() {
		t.Fatalf("the hold ends with its program: %#v", eff)
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
