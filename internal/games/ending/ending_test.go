package ending

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GhostofGoes/WOPR/internal/assets"
	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/games/tictactoe"
	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

var info = games.Info{Name: "ENDING", Slug: Slug, Layout: proto.LayoutFull, Internal: true}

// The film's last exchange, as the player sees it after zero players.
func TestFinalDialogue(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, New(), info, "", 1)
	if asking, _ := g.Asking(); !asking || !g.Contains("GREETINGS PROFESSOR FALKEN.") {
		t.Fatalf("the ending greets and waits:\n%s", g.Transcript())
	}
	g.Type("Hello.")
	res, over := g.Result()
	if !over || !res.NoVerdict || res.Outcome != proto.NoWinner {
		t.Fatalf("result %+v", res)
	}
	golden.AssertString(t, "final_dialogue", g.Transcript())
}

func TestMovieModeTypesItsOwnHello(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, New(), info, MovieMode, 1)
	if res, over := g.Result(); !over || !res.NoVerdict || !g.Contains("NOT TO PLAY.") {
		t.Fatalf("movie mode runs to the end unaided: %+v\n%s", res, g.Transcript())
	}
	if !strings.Contains(g.Transcript(), "GREETINGS PROFESSOR FALKEN.\n\nHello.\n\nA STRANGE GAME.") {
		t.Errorf("the film's answer is typed as the player's would be:\n%s", g.Transcript())
	}
	for _, mode := range []string{"movie:2", MovieMode + ":9"} {
		e := New().(*Game)
		e.Start(proto.Env{Seed: 1, Mode: mode})
		if want := int(mode[len(mode)-1] - '0'); e.cracked() != want || !e.movie {
			t.Errorf("%q: the code starts at %d, want %d", mode, e.cracked(), want)
		}
	}
}

// With pacing on, the show runs self-play, then the montage, then the greeting; every
// self-play game is a draw, and the whole show is bounded.
func TestTheShow(t *testing.T) {
	t.Parallel()
	g := New().(*Game)
	g.Start(proto.Env{Seed: 3, Width: 80, Height: 19})
	var shots []string
	snap := func(label string) {
		c := proto.NewCanvas(80, 19)
		g.View(c)
		shots = append(shots, "==== "+label+" ====\n"+c.String())
	}
	elapsed := time.Duration(0)
	sawMontage, sawBoards := false, false
	for g.phase != greeting {
		g.Handle(proto.TickEvent{Dt: tick})
		elapsed += tick
		for _, b := range g.boards {
			if w := b.Winner(); w != 0 {
				t.Fatalf("self-play must never be won, but %c won:\n%s", w, strings.Join(b.Text(5), "\n"))
			}
		}
		if g.phase == selfPlay && g.round == 3 && marks(g.boards[0]) >= 5 && !sawBoards {
			sawBoards = true
			snap("self-play, round 4")
		}
		if g.phase == montage && g.shown >= 25 && !sawMontage {
			sawMontage = true
			snap("montage")
		}
		if elapsed > 2*time.Minute {
			t.Fatalf("the show never ends (phase %d)", g.phase)
		}
	}
	if !sawBoards || !sawMontage || g.shown != len(assets.Scenarios) || g.cracked() != len("CPE1704TKS") {
		t.Errorf("montage %v, shown %d, cracked %d", sawMontage, g.shown, g.cracked())
	}
	t.Logf("the show lasts %v", elapsed)
	if elapsed > 40*time.Second {
		t.Errorf("the show runs %v; keep it under 40 s", elapsed)
	}
	golden.AssertString(t, "show", strings.Join(shots, "\n"))
}

// The self-play frame fits the Full layout at 80x24, with the front panel (19 rows) and
// without (20): all seven boards whole, the last moves marked, the launch code below the big
// board, nothing drawn outside 80 columns, and the frame centred, a row or two clear above and
// below.
func TestSelfPlayFits(t *testing.T) {
	t.Parallel()
	for _, h := range []int{19, 20} {
		g := New().(*Game)
		g.Start(proto.Env{Seed: 3, Width: 80, Height: h})
		for g.round < 2 || marks(g.boards[0]) < 9 {
			g.Handle(proto.TickEvent{Dt: tick})
		}
		const w = 100
		c := proto.NewCanvas(w, h)
		g.View(c)
		corners, code, first, last := 0, -1, -1, -1
		for y, row := range strings.Split(strings.TrimSuffix(c.String(), "\n"), "\n") {
			if strings.TrimSpace(row) != "" {
				if first < 0 {
					first = y
				}
				last = y
			}
			for x, r := range row {
				if r != ' ' && (x < (w-80)/2 || x >= (w+80)/2) {
					t.Fatalf("height %d: drawn at column %d, outside 80:\n%s", h, x, c.String())
				}
			}
			corners += strings.Count(row, "+")
			if strings.Contains(row, lineCodeLabel[0].Text) {
				code = y
			}
		}
		underlined := 0
		for y := range c.H {
			for x := range c.W {
				if c.At(x, y).A&proto.AttrUnderline != 0 {
					underlined++
				}
			}
		}
		if underlined == 0 { // the last move on each board, as tictactoe draws it
			t.Errorf("height %d: no board marks its last move:\n%s", h, c.String())
		}
		if corners != 4*boards || code < tictactoe.BigRows || code >= h {
			t.Errorf("height %d: %d grid crossings (want %d), code on row %d:\n%s", h, corners, 4*boards, code, c.String())
		}
		if above, below := first, h-1-last; above < 1 || below < above || below > above+1 {
			t.Errorf("height %d: %d rows clear above the frame, %d below:\n%s", h, above, below, c.String())
		}
	}
}

func marks(b tictactoe.Board) int {
	n := 0
	for _, m := range b {
		if m != 0 {
			n++
		}
	}
	return n
}

func TestEveryLineHasProvenance(t *testing.T) {
	t.Parallel()
	notice, err := os.ReadFile(filepath.Join("..", "..", "..", "NOTICE.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, err := range script.Validate(Lines, string(notice)) {
		t.Error(err)
	}
	if Slug != tictactoe.EndingSlug {
		t.Error("tic-tac-toe hands off to this slug")
	}
}

// The montage moves the whole table, so it redraws at most 3 times a second, at any tick
// size; with --reduce-motion it does not speed up.
func TestMontageFlashCap(t *testing.T) {
	t.Parallel()
	for _, rm := range []bool{false, true} {
		for _, dt := range []time.Duration{tick, 66 * time.Millisecond, 100 * time.Millisecond} {
			g := New().(*Game)
			g.Start(proto.Env{Seed: 3, Width: 80, Height: 19, ReduceMotion: rm})
			for g.phase == selfPlay {
				g.Handle(proto.TickEvent{Dt: dt})
			}
			var changes []time.Duration // when the montage table changed
			last, now := -1, time.Duration(0)
			counts := map[int]bool{}
			for g.phase == montage {
				g.Handle(proto.TickEvent{Dt: dt})
				now += dt
				if g.shown != last {
					changes = append(changes, now)
					if last >= 0 {
						counts[g.shown-last] = true
					}
					last = g.shown
				}
			}
			for i := 3; i < len(changes); i++ {
				if changes[i]-changes[i-3] < time.Second {
					t.Fatalf("reduce motion %v, dt %v: four montage frames within %v", rm, dt, changes[i]-changes[i-3])
				}
			}
			if rm && len(counts) > 2 { // a steady step, and perhaps a short last frame
				t.Errorf("reduce motion must not accelerate: steps %v", counts)
			}
		}
	}
}

// Self-play speeds up round by round, but not with --reduce-motion, which plays every move at
// one steady pace and so takes about as long in all.
func TestSelfPlayAcceleration(t *testing.T) {
	t.Parallel()
	took := map[bool]time.Duration{}
	for _, rm := range []bool{false, true} {
		g := New().(*Game)
		g.Start(proto.Env{Seed: 3, Width: 80, Height: 19, ReduceMotion: rm})
		steps := map[time.Duration]bool{}
		for g.phase == selfPlay {
			steps[g.step] = true
			g.Handle(proto.TickEvent{Dt: tick})
			took[rm] += tick
		}
		if rm && (len(steps) != 1 || !steps[steadyStep]) {
			t.Errorf("reduce motion: steps %v, want only %v", steps, steadyStep)
		}
		if !rm && len(steps) < rounds {
			t.Errorf("self-play must speed up each round: steps %v", steps)
		}
	}
	if d := took[true] - took[false]; d < -2*time.Second || d > 2*time.Second {
		t.Errorf("self-play takes %v with reduce motion and %v without; keep them close", took[true], took[false])
	}
}

// An empty Enter at GREETINGS PROFESSOR FALKEN. (one pressed to hurry the show along) is
// not the answer.
func TestEmptyLineDoesNotAnswerTheGreeting(t *testing.T) {
	t.Parallel()
	g := New().(*Game)
	g.Start(proto.Env{Seed: 1, Instant: true})
	outs := g.Handle(proto.LineEvent{Text: ""})
	if g.phase != greeting || len(outs) != 1 {
		t.Fatalf("an empty line re-prompts: %v", outs)
	}
	g.Handle(proto.LineEvent{Text: "Hello."})
	if g.phase != done {
		t.Fatal("Hello. answers")
	}
}

// The launch code carries on from what GTW cracked and is complete by the last round.
func TestLaunchCodeProgress(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		mode  string
		first int
	}{{"", 7}, {"code:2", 2}, {"code:7", 7}, {"code:0", 0}, {"code:99", 10}} {
		g := New().(*Game)
		g.Start(proto.Env{Seed: 1, Mode: tc.mode})
		if got := g.cracked(); got != tc.first {
			t.Errorf("%q: starts at %d, want %d", tc.mode, got, tc.first)
		}
		prev := g.cracked()
		for g.phase == selfPlay {
			g.Handle(proto.TickEvent{Dt: tick})
			if g.phase == selfPlay {
				if g.cracked() < prev {
					t.Fatalf("%q: the code went backwards", tc.mode)
				}
				prev = g.cracked()
			}
		}
		if prev != len("CPE1704TKS") {
			t.Errorf("%q: the last self-play frame shows %d characters, want all 10", tc.mode, prev)
		}
	}
}
