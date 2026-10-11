package scenes

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/GhostofGoes/WOPR/internal/golden"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// The six scenes of docs/PLAN.md §7, in the film's order.
func TestTheFilmsScenes(t *testing.T) {
	t.Parallel()
	var slugs []string
	for _, s := range All() {
		slugs = append(slugs, s.Slug)
	}
	if got := strings.Join(slugs, " "); got != "first-contact joshua first-strike call-back norad-terminal climax" {
		t.Errorf("scenes %s", got)
	}
}

// Every step carries provenance that NOTICE.md backs, WOPR's text is in capitals and only the
// user's typed lines are not, and the text fits 80 columns and the menu.
func TestEveryStepHasProvenance(t *testing.T) {
	t.Parallel()
	notice, err := os.ReadFile(filepath.Join("..", "..", "..", "NOTICE.md"))
	if err != nil {
		t.Fatal(err)
	}
	slug := regexp.MustCompile(`^[a-z]+(-[a-z]+)*$`)
	seen := map[string]bool{}
	var filmLines []script.Ls
	for _, s := range All() {
		for _, err := range script.Validate(s.Lines(), string(notice)) {
			t.Errorf("%s: %v", s.Slug, err)
		}
		filmLines = append(filmLines, s.Lines()...)
		if !slug.MatchString(s.Slug) || seen[s.Slug] {
			t.Errorf("slug %q must be unique, lower case and hyphenated", s.Slug)
		}
		seen[s.Slug] = true
		if s.Title != strings.ToUpper(s.Title) || len(s.Title) > 16 || s.Blurb != strings.ToUpper(s.Blurb) || len(s.Blurb) > 56 {
			t.Errorf("%s: the menu shows %q and %q in capitals, within 16 and 56 columns", s.Slug, s.Title, s.Blurb)
		}
		if len(s.Steps) == 0 {
			t.Errorf("%s has no steps", s.Slug)
		}
		for i, st := range s.Steps {
			switch st := st.(type) {
			case Type:
				if !st.Text.User || len(st.Prompt) > 1 {
					t.Errorf("%s step %d: a typed line is the user's, after at most one prompt", s.Slug, i)
				}
				if line := strings.Join(st.Prompt.Texts(), "") + st.Text.Text; len(line) > 80 {
					t.Errorf("%s step %d: %q is wider than 80 columns", s.Slug, i, line)
				}
			case Say:
				for _, l := range st.Lines {
					if len(l.Text) > 80 || l.User {
						t.Errorf("%s step %d: %q is WOPR's and fits 80 columns", s.Slug, i, l.Text)
					}
				}
			case Board, Run:
				if s.Interactive {
					t.Errorf("%s: an interactive scene is console text the persona prints", s.Slug)
				}
			case Clock:
				checkClock(t, st)
				if s.Interactive {
					t.Errorf("%s: an interactive scene is console text the persona prints", s.Slug)
				}
			case Clear, Wait:
			}
		}
	}
	// The film's lines, frozen: a change here is a change to the film's text (AGENTS.md).
	golden.AssertString(t, "film_lines", strings.Join(script.FilmTexts(filmLines), "\n")+"\n")
}

// Every page is left up for a pause before a new page, a launched program or the end of the
// scene replaces it. Under --instant nothing else holds a page: text and typing appear at once.
func TestEveryPageIsHeld(t *testing.T) {
	t.Parallel()
	held := func(st Step) bool {
		switch st.(type) {
		case Wait, Clock:
			return true
		}
		return false
	}
	for _, s := range All() {
		for i, st := range s.Steps {
			switch st.(type) {
			case Clear, Run:
				if i > 0 && !held(s.Steps[i-1]) {
					t.Errorf("%s step %d: %T follows %T, not a pause", s.Slug, i, st, s.Steps[i-1])
				}
			}
		}
		if last := s.Steps[len(s.Steps)-1]; !held(last) {
			t.Errorf("%s ends on %T, not a pause", s.Slug, last)
		}
	}
}

// checkClock: a game clock has two seconds readings, the film's, which stay within their
// minute for as long as the clock runs; its lines are the film's and fit 80 columns.
func checkClock(t *testing.T, c Clock) {
	t.Helper()
	runs := int(c.D / time.Second)
	if r := c.Readings(); len(r) != 2 || r[0]+runs > 59 || r[1]-runs < 0 {
		t.Errorf("a clock of readings %v for %v passes a minute", r, c.D)
	}
	for _, l := range c.Lines {
		if l.Prov != script.Film || len(l.Text) > 80 {
			t.Errorf("clock line %q: the film's, within 80 columns", l.Text)
		}
	}
}

// The clock's readings tick from the film's, the first up and the second down, a second at a
// time.
func TestClockReading(t *testing.T) {
	t.Parallel()
	c := Clock{Lines: script.Recon("UP 00 SEC", "", "DOWN 59 SEC"), D: 4 * time.Second}
	if r := c.Readings(); len(r) != 2 || r[0] != 0 || r[1] != 59 {
		t.Errorf("readings %v", r)
	}
	for _, tc := range []struct {
		t    time.Duration
		want string
	}{
		{0, "UP 00 SEC||DOWN 59 SEC"},
		{999 * time.Millisecond, "UP 00 SEC||DOWN 59 SEC"},
		{time.Second, "UP 01 SEC||DOWN 58 SEC"},
		{75 * time.Second, "UP 59 SEC||DOWN 00 SEC"},
	} {
		if got := strings.Join(c.Reading(tc.t), "|"); got != tc.want {
			t.Errorf("after %v: %q, want %q", tc.t, got, tc.want)
		}
	}
}
