// Package testkit drives proto programs through the real host runner, with no terminal,
// and records a transcript: what WOPR printed and what the user typed, as the console
// would show it. Every game's definition of done includes a testkit transcript test.
package testkit

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/proto/host"
)

// Session is a program running under the host, for tests.
type Session struct {
	t      testing.TB
	r      *host.Runner
	lines  []string
	prompt string
	asking bool
	keys   bool
	exited bool
}

// Start runs root under a runner configured by cfg. Instant and Deterministic are forced on:
// transcripts must not depend on time or machine speed.
func Start(t testing.TB, root proto.Program, place host.Placement, cfg host.Config) *Session {
	t.Helper()
	cfg.Instant, cfg.Deterministic = true, true
	if cfg.Area == nil {
		cfg.Area = func(p host.Placement) (int, int) {
			switch p.Layout {
			case proto.LayoutPanel:
				return 80, p.PanelRows
			case proto.LayoutFull:
				return 80, 20
			default:
				return 80, 23
			}
		}
	}
	s := &Session{t: t, r: host.New(cfg)}
	s.apply(s.r.Start(root, place))
	return s
}

// Type submits a line, echoing it the way the console does (prompt + text, then a blank).
func (s *Session) Type(line string) *Session {
	s.t.Helper()
	if s.exited {
		s.t.Fatalf("Type(%q) after the session ended", line)
	}
	if !s.asking {
		s.t.Fatalf("Type(%q) but nothing is asking for a line; transcript:\n%s", line, s.Transcript())
	}
	s.lines = append(s.lines, s.prompt+line, "")
	s.asking = false
	s.apply(s.r.Line(line))
	return s
}

// Key sends a key in key mode.
func (s *Session) Key(k proto.Key, ch rune) *Session {
	s.apply(s.r.Key(k, ch))
	return s
}

// Esc presses Esc.
func (s *Session) Esc() *Session {
	s.apply(s.r.Esc())
	return s
}

// Advance moves the clock forward.
func (s *Session) Advance(d time.Duration) *Session {
	s.apply(s.r.Advance(d))
	return s
}

// Asking reports whether a line is expected, and with which prompt.
func (s *Session) Asking() (bool, string) { return s.asking, s.prompt }

// Exited reports whether the session ended.
func (s *Session) Exited() bool { return s.exited }

// Runner exposes the runner, for assertions about the stack.
func (s *Session) Runner() *host.Runner { return s.r }

// Transcript is everything printed so far. A page break is shown as "[CLEAR]".
func (s *Session) Transcript() string { return strings.Join(s.lines, "\n") + "\n" }

// Wide lists the transcript's lines wider than cols columns.
func (s *Session) Wide(cols int) []string {
	var out []string
	for _, l := range s.lines {
		if utf8.RuneCountInString(l) > cols {
			out = append(out, l)
		}
	}
	return out
}

// Contains reports whether the transcript contains text.
func (s *Session) Contains(text string) bool { return strings.Contains(s.Transcript(), text) }

func (s *Session) apply(effects []host.Effect) {
	for len(effects) > 0 {
		e := effects[0]
		effects = effects[1:]
		switch e := e.(type) {
		case host.Print:
			s.lines = append(s.lines, e.Lines...)
		case host.PageBreak:
			s.lines = append(s.lines, "[CLEAR]")
		case host.AskLine:
			s.asking, s.prompt, s.keys = true, e.Prompt, false
		case host.AskKeys:
			s.asking, s.keys = false, true
		case host.Notice:
			if e.Text != "" {
				s.lines = append(s.lines, "[NOTICE] "+e.Text)
			}
		case host.Exit:
			s.exited = true
		case host.StartThink:
			effects = append(s.think(e), effects...)
		case host.Pause, host.CancelThink, host.Relayout, host.Redraw:
		}
	}
}

// think runs Fn on a goroutine while drawing the running program's View, so the race
// detector sees any state Fn shares with the program.
func (s *Session) think(st host.StartThink) []host.Effect {
	ctx := context.Background()
	cancel := func() {}
	if st.Deadline > 0 {
		ctx, cancel = context.WithTimeout(ctx, st.Deadline)
	}
	defer cancel()
	var (
		wg    sync.WaitGroup
		value any
		err   error
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		value, err = st.Fn(ctx)
	}()
	if prog, place := s.r.Top(); prog != nil && place.Layout != proto.LayoutConsole {
		w, h := 80, max(place.PanelRows, 1)
		prog.View(proto.NewCanvas(w, h))
	}
	wg.Wait()
	if ctx.Err() == context.DeadlineExceeded {
		s.t.Errorf("a deterministic Think ran past its deadline (%v): it ignored its Limit", st.Deadline)
	}
	return s.r.ThinkResult(st.Gen, value, err)
}

// GameSession is a single game running under a stand-in persona (see Game).
type GameSession struct {
	*Session
	root     *standIn
	launches []string
}

// Game runs game under a stand-in persona that launches it in mode and records how it
// ends. The resolver builds only that game, once; any other launch (a hand-off such as
// the ending) is recorded and reported as not available, so the stand-in still gets a
// GameOver.
func Game(t testing.TB, game proto.Program, info games.Info, mode string, seed uint64) *GameSession {
	t.Helper()
	gs := &GameSession{root: &standIn{slug: info.Slug, mode: mode}}
	built := false
	cfg := host.Config{
		Seed: seed,
		Resolve: func(l proto.Launch) (proto.Program, host.Placement, error) {
			gs.launches = append(gs.launches, l.Slug)
			if l.Slug != info.Slug || built {
				return nil, host.Placement{}, games.ErrNotFound
			}
			built = true
			return game, host.Placement{Layout: info.Layout, PanelRows: info.PanelRows, NoAbort: info.Internal}, nil
		},
	}
	gs.Session = Start(t, gs.root, host.Placement{}, cfg)
	return gs
}

// Result returns the game's result once it has ended.
func (g *GameSession) Result() (proto.Result, bool) {
	if g.root.result == nil {
		return proto.Result{}, false
	}
	return *g.root.result, true
}

// Launches lists every slug the game session asked the host to launch, in order: the game
// itself first, then any hand-off.
func (g *GameSession) Launches() []string { return g.launches }

// standIn is the root program for Game: it launches one game and keeps its result.
type standIn struct {
	slug, mode string
	result     *proto.Result
}

func (s *standIn) Start(proto.Env) []proto.Output {
	return []proto.Output{proto.Launch{Slug: s.slug, Mode: s.mode}}
}

func (s *standIn) Handle(ev proto.Event) []proto.Output {
	if over, ok := ev.(proto.GameOver); ok {
		r := over.Result
		s.result = &r
	}
	return nil
}

func (s *standIn) View(*proto.Canvas) {}
