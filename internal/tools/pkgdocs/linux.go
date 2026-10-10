package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

// The Linux packages' desktop integration (docs/PLAN.md §8): a menu entry for each package, and the
// AppStream metadata that software centres show. Both come from templates in packaging/linux/; the
// icons (packaging/icons/hicolor) need nothing filled in, so .goreleaser.yaml installs them as they
// are.

// appID names wopr's menu entry, icon and AppStream component on every platform (docs/PLAN.md §8).
// Software centres and desktops know the program by it, so it never changes.
const appID = "io.github.ghostofgoes.wopr"

// The templates, in the tree.
const (
	desktopTemplate  = "packaging/linux/" + appID + ".desktop.in"
	metainfoTemplate = "packaging/linux/" + appID + ".metainfo.xml.in"
	descriptionFile  = "packaging/description.txt"
)

// bindirs maps each package format to where it puts the program: nfpms' bindir in .goreleaser.yaml
// (a test checks). The menu entry names the program by its full path, so that the desktop finds it
// whatever its PATH.
var bindirs = map[string]string{"deb": "/usr/games", "rpm": "/usr/bin"}

// describedReleases is how many of the newest releases the metadata describes, with their notes;
// older ones get only their version and date. Software centres show the newest release's notes as
// "What's New" and older ones on request, and the file stays small.
const describedReleases = 3

// releasePage is a release's page on GitHub, without the tag.
const releasePage = "https://github.com/GhostofGoes/WOPR/releases/tag/"

// writeLinux writes into out the menu entry for each package, deb/<appID>.desktop and
// rpm/<appID>.desktop, and the AppStream metadata both install, <appID>.metainfo.xml.
func writeLinux(root, notes, out, version string) error {
	tmpl, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(desktopTemplate)))
	if err != nil {
		return err
	}
	for _, format := range slices.Sorted(maps.Keys(bindirs)) {
		entry, err := fill(tmpl, map[string][]string{"BINDIR": {bindirs[format]}})
		if err != nil {
			return fmt.Errorf("%s: %w", desktopTemplate, err)
		}
		if err := write(filepath.Join(out, format, appID+".desktop"), desktopTemplate, entry); err != nil {
			return err
		}
	}

	desc, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(descriptionFile)))
	if err != nil {
		return err
	}
	description, err := descriptionXML(string(desc))
	if err != nil {
		return fmt.Errorf("%s: %w", descriptionFile, err)
	}
	changelog := filepath.Join(notes, "changelog.yml")
	data, err := os.ReadFile(changelog)
	if err != nil {
		return err
	}
	releases, err := releasesXML(data, version)
	if err != nil {
		return fmt.Errorf("%s: %w", changelog, err)
	}
	tmpl, err = os.ReadFile(filepath.Join(root, filepath.FromSlash(metainfoTemplate)))
	if err != nil {
		return err
	}
	metainfo, err := fill(tmpl, map[string][]string{"DESCRIPTION": description, "RELEASES": releases})
	if err != nil {
		return fmt.Errorf("%s: %w", metainfoTemplate, err)
	}
	return write(filepath.Join(out, appID+".metainfo.xml"), metainfoTemplate, metainfo)
}

// write writes data to path, making its directory, and says so.
func write(path, from string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	fmt.Printf("pkgdocs: %s -> %s\n", from, path)
	return nil
}

// placeholder is a value to fill into a template: @NAME@.
var placeholder = regexp.MustCompile(`@([A-Z]+)@`)

// fill replaces each @NAME@ in a template with values[NAME]. A placeholder alone on its line stands
// for any number of lines, each indented as the placeholder is; anywhere else its value must be one
// line. Every placeholder needs a value and every value must be used, so that the templates and
// pkgdocs cannot drift apart.
func fill(tmpl []byte, values map[string][]string) ([]byte, error) {
	used := map[string]bool{}
	var b strings.Builder
	var err error
	for line := range strings.Lines(string(tmpl)) {
		if m := placeholder.FindStringSubmatch(line); m != nil && strings.TrimSpace(line) == m[0] {
			v, ok := values[m[1]]
			if !ok {
				return nil, fmt.Errorf("no value for %s", m[0])
			}
			used[m[1]] = true
			indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
			for _, l := range v {
				if l != "" {
					b.WriteString(indent + l)
				}
				b.WriteString("\n")
			}
			continue
		}
		b.WriteString(placeholder.ReplaceAllStringFunc(line, func(p string) string {
			name := strings.Trim(p, "@")
			v, ok := values[name]
			if !ok || len(v) != 1 {
				err = errors.Join(err, fmt.Errorf("%s needs a one-line value", p))
				return p
			}
			used[name] = true
			return v[0]
		}))
	}
	if err != nil {
		return nil, err
	}
	for _, name := range slices.Sorted(maps.Keys(values)) {
		if !used[name] {
			return nil, fmt.Errorf("no @%s@ to fill", name)
		}
	}
	return []byte(b.String()), nil
}

// escapeXML escapes text for an XML element. Quotes need no escaping there.
var escapeXML = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace

// descriptionXML turns description.txt, after its first line (the summary), into AppStream
// paragraphs: a <p> for each paragraph, its lines joined into one.
func descriptionXML(desc string) ([]string, error) {
	_, body, _ := strings.Cut(strings.ReplaceAll(desc, "\r\n", "\n"), "\n")
	var paras []string
	for p := range strings.SplitSeq(body, "\n\n") {
		if p = strings.Join(strings.Fields(p), " "); p != "" {
			paras = append(paras, "<p>"+escapeXML(p)+"</p>")
		}
	}
	if len(paras) == 0 {
		return nil, errors.New("no description after the summary line")
	}
	return paras, nil
}

// release is one entry of changelog.yml, nFPM's chglog format, as relnotes writes it: JSON, which
// is YAML too, with the newest release first.
type release struct {
	Semver  string    `json:"semver"`
	Date    time.Time `json:"date"`
	Changes []struct {
		Note string `json:"note"`
	} `json:"changes"`
}

// releasesXML writes AppStream <release> elements from changelog.yml, newest first, as AppStream
// wants them. The newest must be the version being built, as relnotes makes it. Each has its
// version, as `wopr --version` prints it, and the date the packages' changelogs give it (its tagged
// commit's, in UTC). The newest describedReleases list their notes. A snapshot is marked as one, so
// software centres do not offer it as a release, and only releases link to their page.
func releasesXML(data []byte, version string) ([]string, error) {
	var rels []release
	if err := json.Unmarshal(data, &rels); err != nil {
		return nil, err
	}
	if len(rels) == 0 {
		return nil, errors.New("no releases")
	}
	var lines []string
	for i, r := range rels {
		v, err := upstreamVersion(r.Semver)
		if err != nil {
			return nil, err
		}
		if i == 0 && v != version {
			return nil, fmt.Errorf("the newest release is %s, but the version being built is %s", v, version)
		}
		if r.Date.IsZero() {
			return nil, fmt.Errorf("release %s has no date", v)
		}
		attrs := fmt.Sprintf(`version="%s" date="%s"`, v, r.Date.UTC().Format(time.DateOnly))
		snapshot := strings.Contains(v, "-")
		if snapshot {
			attrs += ` type="snapshot"`
		}
		var body []string
		if i < describedReleases && len(r.Changes) > 0 {
			body = append(body, "<description>", "  <ul>")
			for _, c := range r.Changes {
				body = append(body, "    <li>"+escapeXML(c.Note)+"</li>")
			}
			body = append(body, "  </ul>", "</description>")
		}
		if !snapshot {
			body = append(body, "<url>"+releasePage+"v"+v+"</url>")
		}
		if len(body) == 0 {
			lines = append(lines, "<release "+attrs+"/>")
			continue
		}
		lines = append(lines, "<release "+attrs+">")
		for _, l := range body {
			lines = append(lines, "  "+l)
		}
		lines = append(lines, "</release>")
	}
	return lines, nil
}

// upstreamVersion turns a package version from changelog.yml back into wopr's own: 1.2.3-1 is
// 1.2.3, and a snapshot's 1.2.4~snapshot.abc1234-1 is 1.2.4-snapshot.abc1234.
func upstreamVersion(semver string) (string, error) {
	i := strings.LastIndex(semver, "-")
	if i < 0 {
		return "", fmt.Errorf("version %q has no package revision (-1)", semver)
	}
	v := strings.ReplaceAll(semver[:i], "~", "-")
	if !versionRE.MatchString(v) {
		return "", fmt.Errorf("version %q is not a package version like 1.2.3-1", semver)
	}
	return v, nil
}
