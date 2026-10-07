// Package gamestest provides stub programs for tests of the host, persona and ui. It is
// imported only by tests (internal/archtest enforces this).
package gamestest

import (
	"context"
	"strings"
	"time"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// StubInfo is a playable stub's registry entry.
func StubInfo() games.Info {
	return games.Info{
		Name: "STUB GAME", Slug: "stub", Aliases: []string{"stub game"},
		Layout: proto.LayoutPanel, PanelRows: 4, Status: games.Playable, Blurb: "A test stub.",
	}
}

// Registry returns a registry of the given entries plus a playable stub.
func Registry(entries ...games.Entry) *games.Registry {
	all := append([]games.Entry{}, entries...)
	all = append(all, games.Entry{Info: StubInfo(), New: func() games.Game { return &Stub{} }})
	r, err := games.NewRegistry(all...)
	if err != nil {
		panic(err)
	}
	return r
}

// Stub is a game that exercises every output. Commands: WIN, LOSE, DRAW, THINK, KEYS,
// FULL, ANIMATE, and the movie-mode hooks: CAPTURE (key mode capturing every key, Esc
// included; honoured only when the stub is the root), HOLD and RELEASE (Hold), DRAIN (a
// Drained event once its line is out), SKIP (reveal at once) and TYPE (a typed line, Say.Open).
type Stub struct {
	moves int
	ticks int
}

// Start implements proto.Program.
func (s *Stub) Start(proto.Env) []proto.Output {
	return []proto.Output{proto.Say{Lines: []string{"STUB READY."}, Pace: proto.PaceSpeech}, proto.Prompt{Text: "MOVE: "}}
}

// Handle implements proto.Program.
func (s *Stub) Handle(ev proto.Event) []proto.Output {
	switch ev := ev.(type) {
	case proto.LineEvent:
		s.moves++
		switch strings.ToUpper(ev.Text) {
		case "WIN":
			return []proto.Output{proto.Done{Result: proto.Result{Outcome: proto.Win, Lines: []string{"MOVES: 1"}}}}
		case "LOSE":
			return []proto.Output{proto.Done{Result: proto.Result{Outcome: proto.Loss}}}
		case "DRAW":
			return []proto.Output{proto.Done{Result: proto.Result{Outcome: proto.Draw}}}
		case "THINK":
			n := s.moves // captured by value
			return []proto.Output{proto.Think{
				Fn:     func(context.Context) (any, error) { return n * 2, nil },
				Limit:  proto.Limit{MaxDepth: 1},
				Budget: time.Second,
			}}
		case "KEYS":
			return []proto.Output{proto.AwaitKeys{Hint: "ARROWS"}}
		case "FULL":
			return []proto.Output{proto.SetLayout{Layout: proto.LayoutFull}, proto.Prompt{Text: "MOVE: "}}
		case "ANIMATE":
			return []proto.Output{proto.Animate{Every: 100 * time.Millisecond}}
		case "CAPTURE":
			return []proto.Output{proto.AwaitKeys{Hint: "CAPTURED", Capture: true}}
		case "HOLD":
			return []proto.Output{proto.Hold{On: true}, proto.Say{Lines: []string{"HELD."}}, proto.Prompt{Text: "MOVE: "}}
		case "RELEASE":
			return []proto.Output{proto.Hold{}, proto.Say{Lines: []string{"RELEASED."}}, proto.Prompt{Text: "MOVE: "}}
		case "DRAIN":
			return []proto.Output{proto.Say{Lines: []string{"DRAINING."}}, proto.Drain{}}
		case "SKIP":
			return []proto.Output{proto.Say{Lines: []string{"SKIPPED AT ONCE."}}, proto.Skip{}, proto.Prompt{Text: "MOVE: "}}
		case "TYPE":
			return append(proto.Typed("TYPED: ", "Hello.", nil), proto.Prompt{Text: "MOVE: "})
		}
		return []proto.Output{proto.Say{Lines: []string{"** IMPROPER REQUEST **"}}, proto.Prompt{Text: "MOVE: "}}
	case proto.ThinkDone:
		return []proto.Output{proto.Say{Lines: []string{"THOUGHT."}}, proto.Prompt{Text: "MOVE: "}}
	case proto.KeyEvent:
		return []proto.Output{proto.Say{Lines: []string{"KEY " + ev.Key.String()}}, proto.Prompt{Text: "MOVE: "}}
	case proto.Drained:
		return []proto.Output{proto.Say{Lines: []string{"DRAINED."}}, proto.Prompt{Text: "MOVE: "}}
	case proto.TickEvent:
		s.ticks++
		if s.ticks >= 3 {
			return []proto.Output{proto.Animate{}, proto.Say{Lines: []string{"ANIMATED."}}, proto.Prompt{Text: "MOVE: "}}
		}
	}
	return nil
}

// View implements proto.Program.
func (s *Stub) View(c *proto.Canvas) {
	c.Put(0, 0, "STUB BOARD", proto.StyleBright, 0)
}
