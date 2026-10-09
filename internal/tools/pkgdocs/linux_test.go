package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
)

// repoRoot is the repository, from this package's directory.
var repoRoot = filepath.Join("..", "..", "..")

func readRepo(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestFill(t *testing.T) {
	t.Parallel()
	got, err := fill([]byte("a=@ONE@/x\n  @MANY@\nend\n"), map[string][]string{"ONE": {"1"}, "MANY": {"<p>", "", "  </p>"}})
	if want := "a=1/x\n  <p>\n\n    </p>\nend\n"; err != nil || string(got) != want {
		t.Errorf("fill = %q, %v; want %q", got, err, want)
	}
	for _, c := range []struct {
		why, tmpl string
		values    map[string][]string
	}{
		{"a placeholder with no value", "@ONE@ @TWO@\n", map[string][]string{"ONE": {"1"}}},
		{"a line placeholder with no value", "  @TWO@\n", map[string][]string{}},
		{"a value no placeholder uses", "@ONE@\n", map[string][]string{"ONE": {"1"}, "TWO": {"2"}}},
		{"several lines inside a line", "x @ONE@\n", map[string][]string{"ONE": {"1", "2"}}},
	} {
		if _, err := fill([]byte(c.tmpl), c.values); err == nil {
			t.Errorf("%s: no error", c.why)
		}
	}
}

func TestDescriptionXML(t *testing.T) {
	t.Parallel()
	got, err := descriptionXML("Summary\nOne & two\n<three>.\n\n\nFour.\n")
	if want := []string{"<p>One &amp; two &lt;three&gt;.</p>", "<p>Four.</p>"}; err != nil || !slices.Equal(got, want) {
		t.Errorf("descriptionXML = %q, %v; want %q", got, err, want)
	}
	if _, err := descriptionXML("Only a summary\n"); err == nil {
		t.Error("a description with only a summary must fail")
	}
}

func TestReleasesXML(t *testing.T) {
	t.Parallel()
	const changelog = `[
  {"semver": "1.3.0~snapshot.abc1234-1", "date": "2026-12-01T23:30:00-05:00", "changes": [{"note": "A <new> game & more."}]},
  {"semver": "1.2.0-1", "date": "2026-11-01T10:00:00Z", "changes": [{"note": "One."}, {"note": "Two."}]},
  {"semver": "1.1.0-1", "date": "2026-10-01T10:00:00Z", "changes": []},
  {"semver": "1.0.0-1", "date": "2026-09-01T10:00:00Z", "changes": [{"note": "Too old to describe."}]}
]`
	got, err := releasesXML([]byte(changelog), "1.3.0-snapshot.abc1234")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		// The date is the UTC one, as in the package changelogs.
		`<release version="1.3.0-snapshot.abc1234" date="2026-12-02" type="snapshot">`,
		`  <description>`,
		`    <ul>`,
		`      <li>A &lt;new&gt; game &amp; more.</li>`,
		`    </ul>`,
		`  </description>`,
		`</release>`,
		`<release version="1.2.0" date="2026-11-01">`,
		`  <description>`,
		`    <ul>`,
		`      <li>One.</li>`,
		`      <li>Two.</li>`,
		`    </ul>`,
		`  </description>`,
		`  <url>https://github.com/GhostofGoes/WOPR/releases/tag/v1.2.0</url>`,
		`</release>`,
		`<release version="1.1.0" date="2026-10-01">`,
		`  <url>https://github.com/GhostofGoes/WOPR/releases/tag/v1.1.0</url>`,
		`</release>`,
		`<release version="1.0.0" date="2026-09-01">`,
		`  <url>https://github.com/GhostofGoes/WOPR/releases/tag/v1.0.0</url>`,
		`</release>`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("releasesXML =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	for _, c := range []struct{ why, changelog, version string }{
		{"notes for another version", changelog, "1.3.0"},
		{"no releases", `[]`, "1.0.0"},
		{"not JSON", `- semver: 1.0.0-1`, "1.0.0"},
		{"no package revision", `[{"semver": "1.0.0", "date": "2026-09-01T10:00:00Z"}]`, "1.0.0"},
		{"a version that is not one", `[{"semver": "one-1", "date": "2026-09-01T10:00:00Z"}]`, "one"},
		{"no date", `[{"semver": "1.0.0-1"}]`, "1.0.0"},
	} {
		if _, err := releasesXML([]byte(c.changelog), c.version); err == nil {
			t.Errorf("%s: no error", c.why)
		}
	}
}

// desktopGroup is one group of a desktop entry file: its name and its keys.
type desktopGroup struct {
	name string
	keys map[string]string
}

// desktopKey is a key, with an optional locale, as the Desktop Entry Specification allows them.
var desktopKey = regexp.MustCompile(`^[A-Za-z0-9-]+(\[[A-Za-z_@.]+\])?$`)

// parseDesktop parses a desktop entry file and checks the rules of the Desktop Entry Specification
// that desktop-file-validate checks and that Go can: line syntax, plain ASCII, no tabs or trailing
// space, a [Desktop Entry] group first, no group or key twice.
func parseDesktop(t *testing.T, name, data string) []desktopGroup {
	t.Helper()
	var groups []desktopGroup
	for i, line := range strings.Split(strings.TrimSuffix(data, "\n"), "\n") {
		where := func(format string, args ...any) {
			t.Helper()
			t.Errorf("%s:%d: "+format, append([]any{name, i + 1}, args...)...)
		}
		for _, r := range line {
			if r < ' ' || r > '~' {
				where("%q: keep it printable ASCII, with no tabs", r)
			}
		}
		switch {
		case line != strings.TrimRight(line, " "):
			where("trailing space")
		case line == "" || strings.HasPrefix(line, "#"):
		case strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]"):
			g := line[1 : len(line)-1]
			if slices.ContainsFunc(groups, func(o desktopGroup) bool { return o.name == g }) {
				where("group [%s] again", g)
			}
			groups = append(groups, desktopGroup{name: g, keys: map[string]string{}})
		default:
			k, v, ok := strings.Cut(line, "=")
			switch {
			case !ok || !desktopKey.MatchString(k):
				where("%q is not a group, a comment or a key=value line (no spaces around =)", line)
			case len(groups) == 0:
				where("a key before the first group")
			case groups[len(groups)-1].keys[k] != "":
				where("key %s again", k)
			default:
				groups[len(groups)-1].keys[k] = v
			}
		}
	}
	if len(groups) == 0 || groups[0].name != "Desktop Entry" {
		t.Fatalf("%s: the first group must be [Desktop Entry]", name)
	}
	return groups
}

// list splits a desktop entry list value, which ends with a semicolon.
func list(t *testing.T, key, v string) []string {
	t.Helper()
	if !strings.HasSuffix(v, ";") {
		t.Errorf("%s=%s must end with ;", key, v)
	}
	items := strings.Split(strings.TrimSuffix(v, ";"), ";")
	if slices.Contains(items, "") || len(slices.Compact(slices.Sorted(slices.Values(items)))) != len(items) {
		t.Errorf("%s=%s has an empty or repeated item", key, v)
	}
	return items
}

// The registered categories (Desktop Menu Specification, appendix A): the main ones, and the
// additional ones that go with Game. A menu entry needs exactly one main category.
var (
	mainCategories = []string{
		"AudioVideo", "Audio", "Video", "Development", "Education", "Game", "Graphics", "Network", "Office",
		"Science", "Settings", "System", "Utility",
	}
	gameCategories = []string{
		"ActionGame", "AdventureGame", "ArcadeGame", "BoardGame", "BlocksGame", "CardGame", "KidsGame",
		"LogicGame", "RolePlaying", "Shooter", "Simulation", "SportsGame", "StrategyGame",
	}
)

// shellSpecial are the characters an Exec argument must quote (Desktop Entry Specification,
// "The Exec key"); wopr's arguments need none.
const shellSpecial = " \t\n\"'\\><~|&;$*?#()`"

// The menu entry pkgdocs writes for each package follows the Desktop Entry Specification 1.5 and
// starts wopr from where that package puts it.
func TestDesktopEntries(t *testing.T) {
	t.Parallel()
	tmpl := []byte(readRepo(t, desktopTemplate))
	for format, bindir := range bindirs {
		data, err := fill(tmpl, map[string][]string{"BINDIR": {bindir}})
		if err != nil {
			t.Fatal(err)
		}
		name := format + "/" + appID + ".desktop"
		groups := parseDesktop(t, name, string(data))
		entry := groups[0].keys
		for k, want := range map[string]string{
			"Type": "Application", "Version": "1.5", "Name": "WOPR", "Icon": appID, "Terminal": "true",
			"StartupNotify": "true",
		} {
			if entry[k] != want {
				t.Errorf("%s: %s=%s, want %s", name, k, entry[k], want)
			}
		}
		for _, k := range []string{"GenericName", "Comment"} {
			if entry[k] == "" || entry[k] == entry["Name"] {
				t.Errorf("%s: %s must be set, and say more than Name", name, k)
			}
		}
		if entry["Comment"] == entry["GenericName"] {
			t.Errorf("%s: Comment must say more than GenericName", name)
		}
		var main []string
		for _, c := range list(t, "Categories", entry["Categories"]) {
			switch {
			case slices.Contains(mainCategories, c):
				main = append(main, c)
			case !slices.Contains(gameCategories, c):
				t.Errorf("%s: category %s is not a registered one for a game", name, c)
			}
		}
		if !slices.Equal(main, []string{"Game"}) {
			t.Errorf("%s: main categories %v, want only Game", name, main)
		}
		list(t, "Keywords", entry["Keywords"])

		// Every command, the entry's and each action's, starts wopr by its path in the package.
		actions := list(t, "Actions", entry["Actions"])
		execs := []string{entry["Exec"]}
		var actionGroups []string
		for _, g := range groups[1:] {
			id, ok := strings.CutPrefix(g.name, "Desktop Action ")
			if !ok {
				t.Errorf("%s: unexpected group [%s]", name, g.name)
				continue
			}
			actionGroups = append(actionGroups, id)
			if g.keys["Name"] == "" {
				t.Errorf("%s: action %s has no Name", name, id)
			}
			execs = append(execs, g.keys["Exec"])
		}
		if !slices.Equal(actions, actionGroups) {
			t.Errorf("%s: Actions=%s, but the action groups are %v", name, entry["Actions"], actionGroups)
		}
		for _, e := range execs {
			args := strings.Split(e, " ")
			if args[0] != bindir+"/wopr" {
				t.Errorf("%s: Exec=%s, want %s/wopr and its arguments", name, e, bindir)
			}
			for _, a := range args[1:] {
				if a == "" || strings.ContainsAny(a, shellSpecial) || strings.Contains(a, "%") {
					t.Errorf("%s: Exec=%s has an argument that needs quoting", name, e)
				}
			}
		}
	}
}

// metainfo is the part of an AppStream metainfo file that the tests check.
type metainfo struct {
	XMLName         xml.Name `xml:"component"`
	Type            string   `xml:"type,attr"`
	ID              string   `xml:"id"`
	MetadataLicense string   `xml:"metadata_license"`
	ProjectLicense  string   `xml:"project_license"`
	Name            string   `xml:"name"`
	Summary         string   `xml:"summary"`
	Description     struct {
		Paras []string `xml:"p"`
	} `xml:"description"`
	Developer struct {
		ID   string `xml:"id,attr"`
		Name string `xml:"name"`
	} `xml:"developer"`
	Launchable struct {
		Type  string `xml:"type,attr"`
		Value string `xml:",chardata"`
	} `xml:"launchable"`
	Binaries []string `xml:"provides>binary"`
	URLs     []struct {
		Type  string `xml:"type,attr"`
		Value string `xml:",chardata"`
	} `xml:"url"`
	Screenshots []struct {
		Type    string `xml:"type,attr"`
		Caption string `xml:"caption"`
		Image   string `xml:"image"`
	} `xml:"screenshots>screenshot"`
	Rating struct {
		Type       string `xml:"type,attr"`
		Attributes []struct {
			ID    string `xml:"id,attr"`
			Value string `xml:",chardata"`
		} `xml:"content_attribute"`
	} `xml:"content_rating"`
	Controls   []string `xml:"supports>control"`
	Colors     []string `xml:"branding>color"`
	Categories []string `xml:"categories>category"`
	Keywords   []string `xml:"keywords>keyword"`
	Releases   []struct {
		Version string `xml:"version,attr"`
		Date    string `xml:"date,attr"`
		Type    string `xml:"type,attr"`
		URL     string `xml:"url"`
	} `xml:"releases>release"`
}

// wellFormed reads every token of an XML document, which fails on malformed XML.
func wellFormed(data []byte) error {
	d := xml.NewDecoder(bytes.NewReader(data))
	for {
		_, err := d.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// The AppStream metadata pkgdocs writes from the tree has what software centres need, agrees with
// the menu entry and the packages, and links only to what the docs site and GitHub serve.
// appstreamcli validate --pedantic and appstream-util validate-relax check the rest, where they are
// installed (TestValidators; docs/PLAN.md §8).
func TestMetainfo(t *testing.T) {
	t.Parallel()
	description, err := descriptionXML(readRepo(t, descriptionFile))
	if err != nil {
		t.Fatal(err)
	}
	releases, err := releasesXML([]byte(testChangelog), "0.3.0-snapshot.abc1234")
	if err != nil {
		t.Fatal(err)
	}
	data, err := fill([]byte(readRepo(t, metainfoTemplate)), map[string][]string{"DESCRIPTION": description, "RELEASES": releases})
	if err != nil {
		t.Fatal(err)
	}
	if err := wellFormed(data); err != nil {
		t.Fatalf("not well-formed XML: %v", err)
	}
	var m metainfo
	if err := xml.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}

	// A desktop-application, deliberately (see the template), launched by the menu entry.
	if m.Type != "desktop-application" || m.ID != appID || m.Launchable.Type != "desktop-id" || m.Launchable.Value != appID+".desktop" {
		t.Errorf("component %s %q launched by %s %q; want desktop-application %q launched by desktop-id %q",
			m.Type, m.ID, m.Launchable.Type, m.Launchable.Value, appID, appID+".desktop")
	}
	if !slices.Equal(m.Binaries, []string{"wopr"}) {
		t.Errorf("provides binaries %v, want wopr", m.Binaries)
	}
	if m.MetadataLicense != "CC0-1.0" {
		t.Errorf("metadata_license %s, want CC0-1.0, which AppStream catalogues can use", m.MetadataLicense)
	}
	cfg := readRepo(t, ".goreleaser.yaml")
	if lic := regexp.MustCompile(`(?m)^    license: (.+)$`).FindAllStringSubmatch(cfg, -1); len(lic) != 1 || lic[0][1] != m.ProjectLicense {
		t.Errorf("project_license %q must be the .rpm's License in .goreleaser.yaml, %q", m.ProjectLicense, lic)
	}
	if m.Developer.ID == "" || m.Developer.ID != strings.ToLower(m.Developer.ID) || m.Developer.Name == "" {
		t.Errorf("developer %q %q needs a lowercase ID and a name", m.Developer.ID, m.Developer.Name)
	}
	if len(m.Description.Paras) != len(description) {
		t.Errorf("the description has %d paragraphs, want description.txt's %d", len(m.Description.Paras), len(description))
	}

	// The same name, summary, categories and keywords as the menu entry.
	entry := parseDesktop(t, desktopTemplate, readRepo(t, desktopTemplate))[0].keys
	switch s := m.Summary; {
	case m.Name != entry["Name"] || s != entry["Comment"]:
		t.Errorf("name %q and summary %q must be the menu entry's Name %q and Comment %q", m.Name, s, entry["Name"], entry["Comment"])
	case len(s) > 90 || strings.HasSuffix(s, "."):
		t.Errorf("summary %q: at most 90 characters and no full stop (appstreamcli)", s)
	}
	if want := list(t, "Categories", entry["Categories"]); !slices.Equal(m.Categories, want) {
		t.Errorf("categories %v, want the menu entry's %v", m.Categories, want)
	}
	if want := list(t, "Keywords", entry["Keywords"]); !slices.Equal(m.Keywords, want) {
		t.Errorf("keywords %v, want the menu entry's %v", m.Keywords, want)
	}

	// Links: the docs site's pages and images, which must exist in site/, and GitHub.
	base := regexp.MustCompile(`(?m)^baseURL: (\S+)$`).FindStringSubmatch(readRepo(t, "site/hugo.yaml"))
	if base == nil {
		t.Fatal("site/hugo.yaml has no baseURL")
	}
	site := func(link, dir string) string {
		t.Helper()
		u, err := url.Parse(link)
		rest, ok := strings.CutPrefix(link, base[1])
		if err != nil || u.Scheme != "https" || (!ok && u.Host != "github.com") {
			t.Errorf("%s: want https, on the docs site (%s) or GitHub", link, base[1])
		}
		if !ok {
			return ""
		}
		return filepath.Join(repoRoot, "site", dir, filepath.FromSlash(rest))
	}
	types := map[string]bool{}
	for _, u := range m.URLs {
		if types[u.Type] {
			t.Errorf("url type %s twice", u.Type)
		}
		types[u.Type] = true
		page := site(u.Value, "content")
		if page == "" {
			continue
		}
		if !strings.HasSuffix(u.Value, "/") {
			t.Errorf("%s: a docs site page's address ends with /", u.Value)
		}
		_, errIndex := os.Stat(filepath.Join(page, "_index.md"))
		_, errPage := os.Stat(strings.TrimSuffix(page, string(filepath.Separator)) + ".md")
		if u.Value != base[1] && errIndex != nil && errPage != nil {
			t.Errorf("%s: no page in site/content for it", u.Value)
		}
	}
	for _, want := range []string{"homepage", "help", "bugtracker", "vcs-browser", "contribute"} {
		if !types[want] {
			t.Errorf("no url of type %s", want)
		}
	}
	defaults := 0
	for _, s := range m.Screenshots {
		if s.Type == "default" {
			defaults++
		}
		if s.Caption == "" || len(s.Caption) > 50 || strings.HasSuffix(s.Caption, ".") {
			t.Errorf("screenshot %s: caption %q needs 1 to 50 characters (appstream-util) and no full stop", s.Image, s.Caption)
		}
		if f := site(s.Image, "static"); f == "" || !strings.HasSuffix(f, ".png") {
			t.Errorf("screenshot %s: want a PNG from the docs site", s.Image)
		} else if _, err := os.Stat(f); err != nil {
			t.Errorf("screenshot %s: the docs site does not serve it: %v", s.Image, err)
		}
	}
	if defaults != 1 || len(m.Screenshots) < 3 {
		t.Errorf("%d screenshots, %d of them default; want at least 3, and 1 default", len(m.Screenshots), defaults)
	}

	// The ratings the template justifies, and nothing else.
	rating := map[string]string{}
	for _, a := range m.Rating.Attributes {
		rating[a.ID] = a.Value
	}
	if want := map[string]string{"violence-fantasy": "moderate", "money-gambling": "moderate"}; m.Rating.Type != "oars-1.1" || !maps.Equal(rating, want) {
		t.Errorf("content rating %s %v, want oars-1.1 %v", m.Rating.Type, rating, want)
	}
	if !slices.Equal(m.Controls, []string{"keyboard"}) {
		t.Errorf("supported controls %v, want keyboard", m.Controls)
	}
	for _, c := range m.Colors {
		if !regexp.MustCompile(`^#[0-9a-f]{6}$`).MatchString(c) {
			t.Errorf("branding colour %q is not like #1a2b3c", c)
		}
	}

	// The releases, newest first.
	if len(m.Releases) != 2 || m.Releases[0].Type != "snapshot" || m.Releases[1].URL == "" {
		t.Errorf("releases %+v, want the snapshot and 0.2.0 with its page", m.Releases)
	}
	for i, r := range m.Releases {
		d, err := time.Parse(time.DateOnly, r.Date)
		if err != nil {
			t.Errorf("release %s: date %q", r.Version, r.Date)
		}
		if i > 0 && d.After(must(time.Parse(time.DateOnly, m.Releases[i-1].Date))) {
			t.Errorf("release %s is newer than the one before it", r.Version)
		}
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// Each package's menu entry names the program where .goreleaser.yaml's nfpms put it.
func TestBindirs(t *testing.T) {
	t.Parallel()
	cfg := readRepo(t, ".goreleaser.yaml")
	for format, want := range bindirs {
		_, block, ok := strings.Cut(cfg, "\n  - id: "+format+"\n")
		block, _, _ = strings.Cut(block, "\n  - id: ")
		got := regexp.MustCompile(`(?m)^    bindir: (\S+)$`).FindStringSubmatch(block)
		if !ok || got == nil || got[1] != want {
			t.Errorf("the %s package's bindir in .goreleaser.yaml is %q; pkgdocs's menu entry names %s", format, got, want)
		}
	}
}
