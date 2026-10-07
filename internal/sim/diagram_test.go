package sim

import (
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// pictureScenario has n regions of every terrain in turn, and the units and holdings
// setup gives it.
func pictureScenario(n int, setup func(s *State)) *Scenario {
	names := []string{"ALPHA", "BRAVO", "CHARLIE", "DELTA", "ECHO", "FOXTROT", "GOLF"}
	regions := make([]Region, n)
	for i := range regions {
		regions[i] = Region{Name: names[i], Terrain: []Terrain{City, Rough, Open}[i%3]}
	}
	return &Scenario{Title: "TEST", Intro: script.Orig("TEST."), Regions: regions, TurnLimit: 3, Setup: setup}
}

func picture(t *testing.T, sc *Scenario) []string {
	t.Helper()
	g := NewGame(sc)
	g.Start(proto.Env{Seed: 1, Instant: true})
	return g.diagram()
}

// The picture of a crowded strip: more units than fit (a +), a hidden cell of yours (~)
// sharing a region with WOPR's units, WOPR's hidden unit left out, the stretches each side
// holds with a gap nobody holds, an overlay over one region in place of its rooftops, and
// a region with ground of the scenario's own.
func TestDiagram(t *testing.T) {
	t.Parallel()
	sc := pictureScenario(7, func(s *State) {
		s.AddUnit(Player, testInf, 0)
		for range 5 {
			s.AddUnit(Player, testInf, 2)
		}
		s.AddUnit(Player, testInf, 3).Hidden = true
		s.AddUnit(WOPR, testInf, 3)
		s.AddUnit(WOPR, testInf, 3).Hidden = true
		for range 6 {
			s.AddUnit(WOPR, testInf, 5)
		}
		s.Control[1], s.Control[6] = Player, WOPR
	})
	sc.Overlay = func(_ *State, r int) string {
		if r == 3 {
			return "~~~~~~~~~~~~~~~~~~~~"
		}
		return ""
	}
	sc.Ground = func(_ *State, r int) (string, string) {
		if r == 5 {
			return "(@)(@)", " |  |"
		}
		return "", ""
	}
	got := picture(t, sc)
	want := []string{
		`   _ [] _     /\  /\     .  .  .  ~~~~~~~~~~~  /\  /\     (@)(@)     _ [] _`,
		`  |#|##|#|   /  \/  \   . .  .  .  |#|##|#|   /  \/  \     |  |     |#|##|#|`,
		` ===>(1)========(2)====+>>>(3)=======~(4)<=======(5)========(6)<<<+====(7)====`,
		` <------------- YOU ------------->                      <------- WOPR ------->`,
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("the picture:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// ArtProblems passes good art and reports each kind of bad line, a title too tall, and a
// block left out of the script text.
func TestArtProblems(t *testing.T) {
	t.Parallel()
	title, cloud := script.Orig(`  /\`, ` /  \`), script.Orig(``, `~ ~`)
	if bad := ArtProblems([]script.Ls{title, cloud}, title, cloud); len(bad) > 0 {
		t.Errorf("good art: %q", bad)
	}
	tall := make([]string, ArtRows+1)
	for i := range tall {
		tall[i] = "X"
	}
	for _, art := range []script.Ls{
		script.Orig("LOWER CASE v"), script.Orig("TRAILING "), script.Orig(strings.Repeat("X", 81)),
		script.Orig("A TAB\t"), script.Orig("£"), script.Orig(tall...),
	} {
		if bad := ArtProblems([]script.Ls{art}, art); len(bad) != 1 {
			t.Errorf("%q: %q", art.Texts(), bad)
		}
	}
	if bad := ArtProblems([]script.Ls{title}, title, cloud); len(bad) != 1 {
		t.Errorf("an unlisted block: %q", bad)
	}
}

// Every strip the engine allows fits 80 columns in four rows, overlay or not, numbers its
// regions in order on the road and marks the front where the sides' stretches meet.
func TestDiagramFits(t *testing.T) {
	t.Parallel()
	for n := 5; n <= 7; n++ {
		sc := pictureScenario(n, func(s *State) {
			for r := range n {
				side := Player
				if r >= n/2 {
					side = WOPR
				}
				for range 9 {
					s.AddUnit(side, testInf, r)
				}
			}
		})
		sc.Overlay = func(*State, int) string { return strings.Repeat(": ", 20) }
		got := picture(t, sc)
		if len(got) != 4 {
			t.Fatalf("%d regions: %d rows, want 4", n, len(got))
		}
		for _, l := range got {
			if len(l) > 80 {
				t.Errorf("%d regions: wider than 80: %q", n, l)
			}
		}
		road, held := got[2], got[3]
		at := 0
		for r := 1; r <= n; r++ {
			i := strings.Index(road[at:], "("+string(rune('0'+r))+")")
			if i < 0 {
				t.Fatalf("%d regions: (%d) missing or out of order on %q", n, r, road)
			}
			at += i
		}
		if !strings.Contains(held, "><") || !strings.Contains(held, " YOU ") || !strings.Contains(held, " WOPR ") {
			t.Errorf("%d regions: the stretches held: %q", n, held)
		}
		if strings.Count(road, "+") != n {
			t.Errorf("%d regions: nine units a region should overflow every cell: %q", n, road)
		}
	}
}
