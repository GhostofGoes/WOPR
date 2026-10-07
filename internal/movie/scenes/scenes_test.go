package scenes

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

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
	for _, s := range All() {
		for _, err := range script.Validate(s.Lines(), string(notice)) {
			t.Errorf("%s: %v", s.Slug, err)
		}
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
			case Clear, Wait:
			}
		}
	}
}
