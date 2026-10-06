package fightercombat

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

var info = games.Info{Name: "FIGHTER COMBAT", Slug: "fighter-combat", Layout: proto.LayoutConsole}

var names = []string{"PRESS", "EXTEND", "CLIMB", "BREAK", "FIRE"}

// pilot flies sensibly: shoot when there is a shot, break when defensive, climb when slow,
// otherwise press.
func pilot(g *Game) string {
	switch {
	case g.cannotFire(0) < 0:
		return "FIRE"
	case g.pos < 0:
		return "BREAK"
	case g.energy[0] <= 1:
		return "CLIMB"
	}
	return "PRESS"
}

func TestTranscript(t *testing.T) {
	t.Parallel()
	g := New().(*Game)
	s := testkit.Game(t, g, info, "", 42)
	for range 40 {
		if asking, _ := s.Asking(); !asking {
			break
		}
		s.Type(pilot(g))
	}
	if _, over := s.Result(); !over {
		t.Fatalf("no end:\n%s", s.Transcript())
	}
	golden.AssertString(t, "transcript", s.Transcript())
}

// Over seeded fights with a sensible and a random pilot: every fight ends within the fuel,
// missiles never go below zero, WOPR never tries a shot it does not have, and both
// pilots win some.
func TestSeededFights(t *testing.T) {
	t.Parallel()
	for _, random := range []bool{false, true} {
		results := map[proto.Outcome]int{}
		for seed := range uint64(60) {
			g := New().(*Game)
			g.Start(proto.Env{Seed: seed, Instant: true})
			r := proto.NewRand(seed, 3)
			var outs []proto.Output
			for step := 0; !g.over; step++ {
				// Refused orders do not use a turn, so bound the steps loosely and the turns tightly.
				if step > 20*turns || g.turn > turns {
					t.Fatalf("seed %d: the fight did not end (turn %d)", seed, g.turn)
				}
				if m := g.choose(); m == fire && g.cannotFire(1) >= 0 {
					t.Fatalf("seed %d: WOPR would fire without a shot", seed)
				}
				in := pilot(g)
				if random {
					in = names[r.IntN(len(names))]
				}
				outs = g.Handle(proto.LineEvent{Text: in})
				if g.missiles[0] < 0 || g.missiles[1] < 0 {
					t.Fatalf("seed %d: missiles %v", seed, g.missiles)
				}
			}
			for _, o := range outs {
				if d, ok := o.(proto.Done); ok {
					results[d.Result.Outcome]++
				}
			}
		}
		t.Logf("random %v: win %d, loss %d, draw %d, none %d", random, results[proto.Win], results[proto.Loss], results[proto.Draw], results[proto.NoWinner])
		if !random && (results[proto.Win] == 0 || results[proto.Loss] == 0) {
			t.Errorf("a sensible pilot should both win and lose sometimes: %v", results)
		}
	}
}

func TestRefusals(t *testing.T) {
	t.Parallel()
	g := New().(*Game)
	g.Start(proto.Env{Seed: 1, Instant: true})
	g.pos = -1
	said := func(outs []proto.Output) string {
		var b strings.Builder
		for _, o := range outs {
			if s, ok := o.(proto.Say); ok {
				b.WriteString(strings.Join(s.Lines, "\n"))
			}
		}
		return b.String()
	}
	if out := said(g.Handle(proto.LineEvent{Text: "fire"})); !strings.Contains(out, "NO SHOT") {
		t.Errorf("defensive at long range: %q", out)
	}
	g.dist = 2
	if out := said(g.Handle(proto.LineEvent{Text: "disengage"})); !strings.Contains(out, "ONLY AT LONG RANGE") {
		t.Errorf("disengage at medium: %q", out)
	}
	if out := said(g.Handle(proto.LineEvent{Text: "loop"})); !strings.Contains(out, "PRESS, EXTEND") {
		t.Errorf("unknown: %q", out)
	}
	g.dist = rangeLong
	outs := g.Handle(proto.LineEvent{Text: "d"})
	if !g.over || !strings.Contains(said(outs), "DISENGAGE") {
		t.Errorf("disengage at long range ends it: %q", said(outs))
	}
}

func TestEscAndProvenance(t *testing.T) {
	t.Parallel()
	s := testkit.Game(t, New(), info, "", 1)
	s.Esc().Esc()
	if res, over := s.Result(); !over || res.Outcome != proto.Aborted {
		t.Fatalf("Esc twice aborts: %+v", res)
	}
	notice, err := os.ReadFile(filepath.Join("..", "..", "..", "NOTICE.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, err := range script.Validate(Lines, string(notice)) {
		t.Error(err)
	}
	for _, block := range Lines {
		for _, l := range block {
			if len(l.Text) > 80 {
				t.Errorf("%q is wider than 80 columns", l.Text)
			}
		}
	}
}
