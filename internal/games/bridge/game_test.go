package bridge

import (
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/cards"
	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

var info = games.Info{Name: "BRIDGE", Slug: "bridge", Layout: proto.LayoutPanel, PanelRows: 10}

// A deterministic transcript of one deal: the player tries one card that does not follow
// suit, then plays the defence heuristic's choice for declarer and dummy.
func TestTranscript(t *testing.T) {
	t.Parallel()
	g := New().(*Game)
	s := testkit.Game(t, g, info, "", 42)
	tried := false
	for range 60 {
		if asking, _ := s.Asking(); !asking {
			break
		}
		seat := g.trick.Next()
		h := g.hands[seat]
		if !tried && len(g.trick.Cards) > 0 {
			for _, c := range h {
				if !cards.Follows(h, g.trick, c) {
					tried = true
					s.Type(c.String())
					break
				}
			}
		}
		s.Type(defend(h, g.trick, g.contract.Strain.Trump(), seat).String())
	}
	res, over := s.Result()
	if !over || strings.Count(s.Transcript(), "TAKES IT.") != 13 || !s.Contains("MUST FOLLOW SUIT") {
		t.Fatalf("transcript:\n%s", s.Transcript())
	}
	if len(res.Lines) != 1 || !strings.Contains(res.Lines[0], "SCORE") {
		t.Fatalf("the result carries the score: %v", res.Lines)
	}
	golden.AssertString(t, "transcript", s.Transcript())
}

func TestView(t *testing.T) {
	t.Parallel()
	g := New().(*Game)
	g.Start(proto.Env{Seed: 3, Width: 80, Height: info.PanelRows, Instant: true, Deterministic: true})
	seat := g.trick.Next()
	g.Handle(proto.LineEvent{Text: defend(g.hands[seat], g.trick, g.contract.Strain.Trump(), seat).String()})
	c := proto.NewCanvas(80, info.PanelRows)
	g.View(c)
	golden.AssertString(t, "view", c.String()+"\n"+c.StyleMap())
}

func TestPlayRefusals(t *testing.T) {
	t.Parallel()
	g := New().(*Game)
	g.Start(proto.Env{Seed: 5, Instant: true, Deterministic: true})
	text := func(outs []proto.Output) string {
		var b strings.Builder
		for _, o := range outs {
			if s, ok := o.(proto.Say); ok {
				b.WriteString(strings.Join(s.Lines, "\n"))
			}
		}
		return b.String()
	}
	seat := g.trick.Next()
	other := g.hands[(seat+1)%cards.Seats][0]
	if out := text(g.Handle(proto.LineEvent{Text: other.String()})); !strings.Contains(out, "DOES NOT HOLD") {
		t.Errorf("another seat's card: %q", out)
	}
	if out := text(g.Handle(proto.LineEvent{Text: "banana"})); !strings.Contains(out, "NAME A CARD") {
		t.Errorf("not a card: %q", out)
	}
	if out := text(g.Handle(proto.LineEvent{Text: "leave"})); !strings.Contains(out, "FINISH THE HAND") {
		t.Errorf("leave: %q", out)
	}
}
