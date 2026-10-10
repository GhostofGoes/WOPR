package airtoground

import (
	"fmt"
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

var info = games.Info{Name: "AIR-TO-GROUND ACTIONS", Slug: "air-to-ground-actions", Layout: proto.LayoutConsole}

// order is a sensible campaign: blind the SAMs, then the soft targets, the airfield, the
// bunker last.
var order = []int{radar, fuelDepot, bridge, airfield, yard, bunker}

// planner picks the next standing target in order and a package sized to its SAMs and
// WOPR's interceptors.
func planner(g *Game) string {
	for _, t := range order {
		if g.targets[t].destroyed() {
			continue
		}
		sead := min(g.targets[t].sams+1, 3)
		escort := 0
		if !g.targets[airfield].destroyed() {
			escort = min(g.fighters/2+1, 3)
		}
		strike := max(g.aircraft-sead-escort, 1)
		strike = min(strike, 5)
		for strike+sead+escort > g.aircraft && escort > 0 {
			escort--
		}
		for strike+sead+escort > g.aircraft && sead > 0 {
			sead--
		}
		return fmt.Sprintf("target %d strike %d sead %d escort %d go", t+1, strike, sead, escort)
	}
	return "end"
}

func TestTranscript(t *testing.T) {
	t.Parallel()
	g := New().(*Game)
	s := testkit.Game(t, g, info, "", 42)
	s.Type("help").Type("status").Type("bunker").Type("go")
	for range 20 {
		if asking, _ := s.Asking(); !asking {
			break
		}
		s.Type(planner(g))
	}
	if _, over := s.Result(); !over {
		t.Fatalf("no end:\n%s", s.Transcript())
	}
	if wide := s.Wide(80); len(wide) > 0 {
		t.Errorf("lines wider than 80 columns: %q", wide)
	}
	golden.AssertString(t, "transcript", s.Transcript())
}

// Over seeded campaigns: every campaign ends within its sorties, nothing goes negative,
// the books balance, and a sensible planner both wins and loses.
func TestSeededCampaigns(t *testing.T) {
	t.Parallel()
	results := map[proto.Outcome]int{}
	for seed := range uint64(100) {
		g := New().(*Game)
		g.Start(proto.Env{Seed: seed, Instant: true})
		var outs []proto.Output
		for step := 0; !g.over; step++ {
			if step > sorties {
				t.Fatalf("seed %d: the campaign did not end", seed)
			}
			outs = g.Handle(proto.LineEvent{Text: planner(g)})
			if g.aircraft < 0 || g.fighters < 0 {
				t.Fatalf("seed %d: aircraft %d, fighters %d", seed, g.aircraft, g.fighters)
			}
			for i := range g.targets {
				if g.targets[i].sams < 0 || g.targets[i].damage > g.targets[i].toughness {
					t.Fatalf("seed %d: target %d is %+v", seed, i, g.targets[i])
				}
			}
		}
		if lost := g.losses[0][categories[0].Text]; lost != squadron-g.aircraft {
			t.Errorf("seed %d: %d aircraft lost on the books, %d gone", seed, lost, squadron-g.aircraft)
		}
		for _, o := range outs {
			if d, ok := o.(proto.Done); ok {
				results[d.Result.Outcome]++
			}
		}
	}
	t.Logf("win %d, loss %d, draw %d", results[proto.Win], results[proto.Loss], results[proto.Draw])
	if results[proto.Win] == 0 || results[proto.Loss] == 0 {
		t.Errorf("a sensible planner should both win and lose sometimes: %v", results)
	}
}

func said(outs []proto.Output) string {
	var b strings.Builder
	for _, o := range outs {
		if s, ok := o.(proto.Say); ok {
			b.WriteString(strings.Join(s.Lines, "\n"))
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func TestPackages(t *testing.T) {
	t.Parallel()
	g := New().(*Game)
	g.Start(proto.Env{Seed: 1, Instant: true})
	cases := []struct{ in, want string }{
		{"go", "WHICH TARGET"},
		{"target", "WHICH TARGET"},
		{"target 9", "WHICH TARGET"},
		{"loop the loop", "TARGET <NUMBER OR NAME>"},
		{"hit the command bunker", "PACKAGE: TARGET COMMAND BUNKER. STRIKE 4, SEAD 2, ESCORT 2."},
		{"fuel strike 3 sead 0 escort 1", "PACKAGE: TARGET FUEL DEPOT. STRIKE 3, SEAD 0, ESCORT 1."},
		{"marsh", "TARGET MARSHALLING YARD"},
		{"strike 0 go", "AT LEAST ONE STRIKE"},
		{"strike 9 sead 2 escort 2 go", "THAT IS 13 AIRCRAFT; YOU HAVE 12."},
		{"strike 9223372036854775807 sead 1 escort 0 go", "THAT IS 1000 AIRCRAFT; YOU HAVE 12."},
		{"strike 9223372036854775807 escort 9223372036854775807 sead 3 go", "THAT IS 2001 AIRCRAFT; YOU HAVE 12."},
		{"status", "SORTIE 1 OF 6"},
	}
	for _, c := range cases {
		if out := said(g.Handle(proto.LineEvent{Text: c.in})); !strings.Contains(out, c.want) {
			t.Errorf("%q: got %q, want %q", c.in, out, c.want)
		}
	}
	g.targets[radar].damage = g.targets[radar].toughness
	if out := said(g.Handle(proto.LineEvent{Text: "radar strike 2 go"})); !strings.Contains(out, "ALREADY DESTROYED") {
		t.Errorf("a destroyed target: %q", out)
	}
	if g.sortie != 1 {
		t.Fatalf("a refused package flew: sortie %d", g.sortie)
	}
	outs := g.Handle(proto.LineEvent{Text: "end"})
	if !g.over || !strings.Contains(said(outs), "CALL OFF") {
		t.Errorf("end calls it off: %q", said(outs))
	}
	if d, ok := outs[len(outs)-1].(proto.Done); !ok || d.Result.Outcome != proto.Loss {
		t.Errorf("calling it off with 2 points loses: %+v", outs[len(outs)-1])
	}
}

// WOPR learns: the mobile battery goes where the last raid went, and a strong escort
// keeps the interceptors on the ground (usually).
func TestWOPRLearns(t *testing.T) {
	t.Parallel()
	waited, held := 0, 0
	for seed := range uint64(40) {
		g := New().(*Game)
		g.Start(proto.Env{Seed: seed, Instant: true})
		g.Handle(proto.LineEvent{Text: "target yard strike 1 sead 0 escort 6 go"})
		if g.targets[yard].destroyed() {
			continue
		}
		if g.mobile == yard {
			waited++
		}
		if g.scramble() == 0 {
			held++
		}
	}
	if waited < 20 || held < 15 {
		t.Errorf("over 40 campaigns the battery waited at the last target %d times and the interceptors stayed down %d times", waited, held)
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
