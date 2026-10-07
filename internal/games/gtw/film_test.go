package gtw_test

import (
	"strings"
	"testing"
	"time"

	"github.com/GhostofGoes/WOPR/internal/games/gtw"
	"github.com/GhostofGoes/WOPR/internal/games/testkit"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// The film's war, as movie mode shows it, is the game's own: the same strip lines and prompt
// as a player who types the film's choices and then presses Enter at each strike.
func TestFilmPlaysTheGame(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, gtw.New(), info, "", 1983)
	g.Type(gtw.FilmSide)
	for _, target := range gtw.FilmTargets {
		g.Type(target)
	}
	before := len(strings.Split(g.Transcript(), "\n")) - 1 // the transcript ends with a newline
	g.Type("").Type("")                                    // the first strike, then the second on the war plan
	played := strings.Split(g.Transcript(), "\n")[before:]

	f := gtw.NewFilm(7, false)
	strike1 := f.Run(gtw.FilmStrike1)
	prompt := f.Prompt()
	strike2 := f.Run(gtw.FilmStrike2)
	want := []string{"", "", "[CLEAR]"} // the echo of the Enter that ends the target list, then the board
	want = append(want, strike1...)
	want = append(want, prompt, "")
	want = append(want, strike2...)
	if got := strings.Join(played[:len(want)], "\n"); got != strings.Join(want, "\n") {
		t.Errorf("the film's lines:\n%s\nthe game's:\n%s", strings.Join(want, "\n"), got)
	}
	if prompt != "STRIKE 2 OF 3 [50 50 100]: " || f.Stage() != gtw.FilmStrike2 {
		t.Errorf("prompt %q, stage %d", prompt, f.Stage())
	}
}

// Frame by frame, the film prints what Run prints, and reaches each stage in the game's time.
func TestFilmAdvancesByFrames(t *testing.T) {
	t.Parallel()
	all := gtw.NewFilm(1, false).Run(gtw.FilmStrike1)
	f := gtw.NewFilm(1, false)
	var got []string
	frames := 0
	for done := false; !done; frames++ {
		var lines []string
		lines, done = f.Advance(gtw.FilmFrame, gtw.FilmStrike1)
		got = append(got, lines...)
		if frames > 100 {
			t.Fatal("the first strike never lands")
		}
	}
	if strings.Join(got, "\n") != strings.Join(all, "\n") {
		t.Errorf("frame by frame:\n%s\nat once:\n%s", strings.Join(got, "\n"), strings.Join(all, "\n"))
	}
	if frames < 40 || frames > 50 {
		t.Errorf("the first strike took %d frames, want the game's 44", frames)
	}
}

// Skipping ahead prints nothing; the climax board then cracks the code as time passes, and
// never all of it: the ending cracks the rest.
func TestFilmClimax(t *testing.T) {
	t.Parallel()
	f := gtw.NewFilm(1, false)
	f.Skip(gtw.FilmRatios)
	c := proto.NewCanvas(80, 19)
	f.View(c)
	if f.Stage() != gtw.FilmRatios || !strings.Contains(c.String(), "PROJECTED KILL RATIOS") {
		t.Fatalf("stage %d:\n%s", f.Stage(), c.String())
	}
	if lines := f.Run(gtw.FilmClimax); len(lines) != 1 || lines[0] != "** GAME ROUTINE RUNNING **" {
		t.Errorf("the climax opens with %q", lines)
	}
	if f.Cracked() != 0 {
		t.Fatal("nothing is cracked yet")
	}
	for range 400 {
		f.Advance(250*time.Millisecond, gtw.FilmClimax)
	}
	c = proto.NewCanvas(80, 19)
	f.View(c)
	if f.Cracked() != 7 || !strings.Contains(c.String(), "CPE 1704 ___") {
		t.Errorf("cracked %d:\n%s", f.Cracked(), c.String())
	}
}

// The side-choice picture is the one the game prints, every row tagged.
func TestSideChoiceIsTheGames(t *testing.T) {
	t.Parallel()
	g := testkit.Game(t, gtw.New(), info, "", 1)
	pic := gtw.SideChoice()
	if !strings.HasPrefix(g.Transcript(), strings.Join(pic.Texts(), "\n")+"\n\nWHICH SIDE") {
		t.Errorf("the picture differs from the game's:\n%s", g.Transcript())
	}
	for _, l := range pic {
		if l.Prov == "" || len(l.Text) > 80 {
			t.Errorf("row %q: provenance %q, width %d", l.Text, l.Prov, len(l.Text))
		}
	}
}
