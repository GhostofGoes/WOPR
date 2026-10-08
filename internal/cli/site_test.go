package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/movie"
)

// The docs site's Usage page lists every option and environment variable, and its Movie
// scenes page every scene, as the help and --scenes do. These tests catch one added here and
// left out there.

func TestSiteUsageListsEveryOption(t *testing.T) {
	t.Parallel()
	page := sitePage(t, "usage/_index.md")
	for _, f := range flagPairs {
		for _, want := range []string{"`--" + f.long, "`-" + f.short + "`"} {
			if !strings.Contains(page, want) {
				t.Errorf("site/content/usage/_index.md does not list %s", strings.Trim(want, "`"))
			}
		}
		if f.env != "" && !strings.Contains(page, "`"+f.env+"`") {
			t.Errorf("site/content/usage/_index.md does not list %s", f.env)
		}
	}
	// Read by cmd/wopr and the ui rather than by Parse.
	for _, env := range []string{"WOPR_DEBUG", "WOPR_PANEL", "NO_COLOR"} {
		if !strings.Contains(page, "`"+env+"`") {
			t.Errorf("site/content/usage/_index.md does not list %s", env)
		}
	}
}

func TestSiteMovieListsEveryScene(t *testing.T) {
	t.Parallel()
	page := sitePage(t, "movie.md")
	for _, s := range movie.Scenes() {
		if !strings.Contains(page, "(`"+s.Slug+"`)") {
			t.Errorf("site/content/movie.md does not describe the scene %s", s.Slug)
		}
	}
}

func sitePage(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "site", "content", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
