package hearts

import (
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/cards"
	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

var info = games.Info{Name: "HEARTS", Slug: "hearts", Layout: proto.LayoutPanel, PanelRows: 10}

// A deterministic transcript of one deal: the player passes their first three cards, tries
// an illegal card once, then plays the first legal card each trick, and leaves.
func TestTranscript(t *testing.T) {
	t.Parallel()
	g := New().(*Game)
	s := testkit.Game(t, g, info, "", 42)
	s.Type("1 2 3")
	tried := false
	for range 100 {
		asking, p := s.Asking()
		if !asking {
			break
		}
		if strings.HasPrefix(p, "NEXT") {
			s.Type("leave")
			continue
		}
		h := g.hands[you]
		if !tried {
			for _, c := range h {
				if Check(h, g.trick, c, g.tricks == 0, g.broken) != OK {
					tried = true
					s.Type(c.String())
					break
				}
			}
		}
		s.Type(Legal(h, g.trick, g.tricks == 0, g.broken)[0].String())
	}
	if _, over := s.Result(); !over || !s.Contains("SCORES:") || strings.Count(s.Transcript(), "TAKES IT.")+strings.Count(s.Transcript(), "TAKE IT.") != 13 {
		t.Fatalf("transcript:\n%s", s.Transcript())
	}
	golden.AssertString(t, "transcript", s.Transcript())
}

// Whole games through the program, the player's seat choosing at random among legal
// cards: 52 cards in play each deal, 26 points a deal, and a game that ends at 100.
func TestWholeGames(t *testing.T) {
	t.Parallel()
	for seed := range uint64(20) {
		g := New().(*Game)
		outs := g.Start(proto.Env{Seed: seed, Instant: true, Deterministic: true})
		r := proto.NewRand(seed, 5)
		for step := 0; ; step++ {
			if step > 2000 {
				t.Fatalf("seed %d: the game never ended", seed)
			}
			if done(outs) {
				break
			}
			var input string
			switch g.phase {
			case passing:
				input = "1 2 3"
			case playing:
				n := len(g.trick.Cards)
				for _, h := range g.hands {
					n += len(h)
				}
				if n != 52-4*g.tricks {
					t.Fatalf("seed %d: %d cards in play after %d tricks", seed, n, g.tricks)
				}
				legal := Legal(g.hands[you], g.trick, g.tricks == 0, g.broken)
				input = legal[r.IntN(len(legal))].String()
			default:
				if sum := g.taken[0] + g.taken[1] + g.taken[2] + g.taken[3]; sum != 26 {
					t.Fatalf("seed %d: %d points in a deal", seed, sum)
				}
				input = ""
			}
			outs = g.Handle(proto.LineEvent{Text: input})
		}
		if max(g.score[0], g.score[1], g.score[2], g.score[3]) < GameOver {
			t.Fatalf("seed %d: ended at %v", seed, g.score)
		}
	}
}

func done(outs []proto.Output) bool {
	for _, o := range outs {
		if _, ok := o.(proto.Done); ok {
			return true
		}
	}
	return false
}

func TestRefusalsInCharacter(t *testing.T) {
	t.Parallel()
	g := New().(*Game)
	g.Start(proto.Env{Seed: 9, Instant: true, Deterministic: true})
	text := func(outs []proto.Output) string {
		var b strings.Builder
		for _, o := range outs {
			if s, ok := o.(proto.Say); ok {
				b.WriteString(strings.Join(s.Lines, "\n"))
			}
		}
		return b.String()
	}
	if out := text(g.Handle(proto.LineEvent{Text: "1 2"})); !strings.Contains(out, "NAME THREE CARDS") {
		t.Errorf("two cards: %q", out)
	}
	if out := text(g.Handle(proto.LineEvent{Text: "1 2 99"})); !strings.Contains(out, "NAME THREE CARDS") {
		t.Errorf("bad position: %q", out)
	}
	g.Handle(proto.LineEvent{Text: "1 2 3"})
	missing := cards.Deck()
	for _, c := range g.hands[you] {
		missing = cards.Remove(missing, c)
	}
	if out := text(g.Handle(proto.LineEvent{Text: missing[0].String()})); !strings.Contains(out, "YOU DO NOT HOLD") {
		t.Errorf("not held: %q", out)
	}
	if out := text(g.Handle(proto.LineEvent{Text: "leave"})); !strings.Contains(out, "FINISH THE HAND") {
		t.Errorf("leave mid-hand: %q", out)
	}
}
