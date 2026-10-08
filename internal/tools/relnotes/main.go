// Command relnotes turns the change notes that changie keeps in .changes/ into a release's notes,
// and checks that the notes are in order. AGENTS.md ("Change notes" and "Releasing") is the guide.
//
//	go run ./internal/tools/relnotes -version 0.3.0 -out DIR  # the notes for a release
//	go run ./internal/tools/relnotes -snapshot -out DIR       # the same for a snapshot build
//	go run ./internal/tools/relnotes -check                   # the notes parse; CHANGELOG.md is current
//	go run ./internal/tools/relnotes -since origin/main       # this branch has a note if it needs one
//	go run ./internal/tools/relnotes -catch-up                # commit notes a release batched on the fly
//
// For a version it writes three files into DIR, which must not be a tracked part of the git tree
// (GoReleaser refuses a dirty tree). The workflows use build/notes, which .gitignore covers:
//
//   - notes.md, the GitHub Release body: that version's notes and a footer;
//   - CHANGELOG.md, the whole changelog, that version included;
//   - changelog.yml, every version in nFPM's chglog format, for the .deb and .rpm changelogs,
//     each under its package version (X.Y.Z-1) and dated by its tagged commit.
//
// A release pull request batches the notes into .changes/vX.Y.Z.md. When it did not, relnotes
// batches them itself in a temporary copy, dated with HEAD's commit date (-date overrides it), and
// warns; the git tree is never changed. Either way the same commit gives the same files. A
// release is refused when another version is batched but not tagged (the tag names the wrong
// version) or when it has no notes at all.
//
// A snapshot is named as GoReleaser names it (see snapshotVersion). On a release pull request,
// whose version is batched but not yet tagged, the snapshot's notes are that version's, followed
// by any added since, so the packages' newest changelog entry is always their own version.
//
// Only tags that release.yml publishes (vX.Y.Z, no prerelease) count as releases.
//
// Exit status: 0 when all is well, 1 when a check fails, 2 when relnotes could not run.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// defaultPackager signs each entry of the package changelogs. It is the packages' maintainer, as
// .goreleaser.yaml's nfpms name it (a test checks): Debian wants "name <email address>", and
// GitHub's no-reply address for the owner's account stands in, so no one's mailbox is published.
const defaultPackager = "GhostofGoes <6599820+GhostofGoes@users.noreply.github.com>"

// errCheck marks a failed check (exit 1), as against a failure to run (exit 2).
var errCheck = errors.New("check failed")

func main() {
	version := flag.String("version", "", "write the notes for this version (1.2.3 or v1.2.3) into -out")
	snapshot := flag.Bool("snapshot", false, "write the notes for this commit's snapshot version into -out")
	out := flag.String("out", "", "directory for notes.md, CHANGELOG.md and changelog.yml")
	date := flag.String("date", "", "date (YYYY-MM-DD) for a version batched here; default: HEAD's commit date")
	packager := flag.String("packager", defaultPackager, "packager named in changelog.yml")
	check := flag.Bool("check", false, "check the notes and that CHANGELOG.md is changie merge's output")
	since := flag.String("since", "", "fail if the commits since this base change user-visible code without a note")
	catchUp := flag.Bool("catch-up", false, "write .changes/vX.Y.Z.md for each release tag that has none")
	flag.Parse()

	modes := 0
	for _, on := range []bool{*version != "", *snapshot, *check, *since != "", *catchUp} {
		if on {
			modes++
		}
	}
	if modes != 1 || flag.NArg() > 0 || (*out == "") != (*version == "" && !*snapshot) {
		fmt.Fprintln(os.Stderr, "relnotes: use one of -version or -snapshot (with -out), -check, -since or -catch-up")
		flag.Usage()
		os.Exit(2)
	}

	r := &runner{root: ".", log: os.Stdout}
	err := func() error {
		defer r.cleanup()
		switch {
		case *check:
			return r.check()
		case *since != "":
			return r.since(*since)
		case *catchUp:
			return r.catchUp()
		default:
			return r.release(*version, *snapshot, *date, *packager, *out)
		}
	}()
	switch {
	case errors.Is(err, errCheck):
		os.Exit(1)
	case err != nil:
		fmt.Fprintln(os.Stderr, "relnotes:", err)
		os.Exit(2)
	}
}

// runner holds what every mode needs: the repository, where to report, and a changie binary,
// built on first use.
type runner struct {
	root    string
	log     io.Writer
	changie changie // set by tests; built by changieBin otherwise
	tmp     []string
}

func (r *runner) cleanup() {
	for _, d := range r.tmp {
		_ = os.RemoveAll(d)
	}
	r.tmp = nil
}

func (r *runner) tempDir() (string, error) {
	d, err := os.MkdirTemp("", "relnotes-")
	if err == nil {
		r.tmp = append(r.tmp, d)
	}
	return d, err
}

func (r *runner) changieBin() (changie, error) {
	if r.changie != "" {
		return r.changie, nil
	}
	dir, err := r.tempDir()
	if err != nil {
		return "", err
	}
	c, err := buildChangie(r.root, dir)
	if err != nil {
		return "", err
	}
	r.changie = c
	return c, nil
}

func (r *runner) stage(frags map[string][]byte, omit map[string]bool) (string, error) {
	dir, err := stage(r.root, frags, omit)
	if dir != "" {
		r.tmp = append(r.tmp, dir)
	}
	return dir, err
}

// logf reports progress. A failed write to the log is not worth failing a release for.
func (r *runner) logf(format string, args ...any) {
	_, _ = fmt.Fprintf(r.log, format, args...)
}

// annotate reports a warning or an error, as a GitHub annotation inside GitHub Actions.
func (r *runner) annotate(level, msg string) {
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		r.logf("::%s title=Release notes::%s\n", level, msg)
		return
	}
	r.logf("relnotes: %s: %s\n", level, msg)
}

// release writes the notes for a version (or this commit's snapshot) into out.
func (r *runner) release(version string, snapshot bool, date, packager, out string) error {
	var v semver
	var ref string
	var err error
	if snapshot {
		v, ref, err = snapshotVersion(r.root)
	} else {
		v, err = parseSemver(version)
		ref = "v" + v.String()
	}
	if err != nil {
		return err
	}
	if date != "" {
		if _, err := time.Parse(time.DateOnly, date); err != nil {
			return fmt.Errorf("-date %q is not YYYY-MM-DD", date)
		}
	}
	kinds, err := configKinds(filepath.Join(r.root, configFile))
	if err != nil {
		return err
	}

	// Each earlier release must have its own file, or its notes would be counted again here. The
	// commit a release was tagged at is that release, not an earlier one, so a snapshot of it is
	// not refused.
	behind, err := unbatchedTags(r.root, &v, snapshot)
	if err != nil {
		return err
	}
	if len(behind) > 0 {
		r.annotate("error", fmt.Sprintf("%s went out without .changes/%s.md, so its notes would repeat in v%s; run `go run ./internal/tools/relnotes -catch-up` in a pull request first",
			behind[0].name, behind[0].name, v))
		return errCheck
	}

	files, err := versionFiles(r.root)
	if err != nil {
		return err
	}
	_, batched := files[v.String()]
	waiting, err := unreleasedNotes(r.root)
	if err != nil {
		return err
	}
	var omit map[string]bool
	var ahead []section
	if snapshot {
		// A release pull request batches its notes before its merge commit is tagged. Its snapshot
		// is that release: the batched notes go under the snapshot's version, ahead of any added
		// since, so the packages' newest changelog entry is always their own version.
		if ahead, omit, err = pendingSections(r.root, files, kinds); err != nil {
			return err
		}
	} else if err := r.checkRelease(v, files, batched, waiting); err != nil {
		return err
	}

	dir, err := r.stage(nil, omit)
	if err != nil {
		return err
	}
	c, err := r.changieBin()
	if err != nil {
		return err
	}
	if !batched {
		if date == "" {
			if date, err = commitDate(r.root, "HEAD"); err != nil {
				return err
			}
		}
		if _, err := c.run(dir, date, "batch", "v"+v.String()); err != nil {
			return err
		}
		if v.pre == "" {
			r.annotate("warning", fmt.Sprintf("v%s was not batched in its release pull request, so its notes were made from .changes/unreleased here; the next pull request must run `go run ./internal/tools/relnotes -catch-up`", v))
		}
	}
	if len(ahead) > 0 {
		if err := prependNotes(filepath.Join(dir, changesDir, "v"+v.String()+".md"), ahead, kinds); err != nil {
			return err
		}
	}
	if _, err := c.run(dir, "", "merge"); err != nil {
		return err
	}

	sections, err := readSections(filepath.Join(dir, changesDir), kinds)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(sections, func(s section) bool { return s.version == v })
	if i < 0 {
		return fmt.Errorf("changie wrote no notes for v%s", v)
	}
	if batched && !snapshot {
		if err := r.warnBatched(sections[i], waiting); err != nil {
			return err
		}
	}
	prev := ""
	if i+1 < len(sections) {
		prev = sections[i+1].version.String()
	}
	times, err := releaseTimes(r.root, v)
	if err != nil {
		return err
	}
	yml, err := chglog(sections, packager, times)
	if err != nil {
		return err
	}
	changelog, err := os.ReadFile(filepath.Join(dir, changelogFile))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	for name, data := range map[string][]byte{
		"notes.md":      []byte(releaseNotes(sections[i], prev, ref)),
		changelogFile:   changelog,
		"changelog.yml": yml,
	} {
		if err := os.WriteFile(filepath.Join(out, name), data, 0o644); err != nil {
			return err
		}
	}
	r.logf("relnotes: wrote the notes for v%s to %s\n", v, out)
	return nil
}

// checkRelease refuses a release whose notes would be wrong: when another version is batched but
// not tagged (the tag names the wrong version, and the notes would go out under it), and when
// there is nothing to batch for an unbatched release.
func (r *runner) checkRelease(v semver, files map[string]string, batched bool, waiting int) error {
	tags, err := releaseTags(r.root)
	if err != nil {
		return err
	}
	tagged := map[string]bool{}
	for _, t := range tags {
		tagged[t.version.String()] = true
	}
	for _, name := range slices.Sorted(maps.Keys(files)) {
		if name != v.String() && !tagged[name] {
			r.annotate("error", fmt.Sprintf("v%s is batched in .changes/v%s.md but not tagged, so its notes would go out under v%s; tag the commit that batched them as v%s instead",
				name, name, v, name))
			return errCheck
		}
	}
	if !batched && waiting == 0 {
		r.annotate("error", fmt.Sprintf("v%s has no notes: there is no .changes/v%s.md, and .changes/unreleased is empty", v, v))
		return errCheck
	}
	return nil
}

// warnBatched warns when a batched release's notes may not match what it ships: notes merged after
// the release pull request, whose changes ship now but are listed under the next release, and a
// version header dated another day than the tagged commit, which dates the package changelogs.
func (r *runner) warnBatched(s section, waiting int) error {
	if waiting > 0 {
		r.annotate("warning", fmt.Sprintf("v%s was batched, but .changes/unreleased still holds notes at this commit (%d): their changes ship in v%s and are listed under the next release. Tag the release pull request's merge commit, not a later one",
			s.version, waiting, s.version))
	}
	day, err := commitDate(r.root, "HEAD")
	if err != nil {
		return err
	}
	if day != s.date {
		r.annotate("warning", fmt.Sprintf(".changes/v%s.md is dated %s, but the tagged commit is from %s (UTC), which dates the .deb and .rpm changelogs. To make them agree, set the header's date to the day the release pull request merges, then run changie merge and the manpage tool again",
			s.version, s.date, day))
	}
	return nil
}

// pendingSections reads the batched versions that are not released yet (pendingVersions), oldest
// first, and names their files for stage to leave out.
func pendingSections(root string, files map[string]string, kinds []string) ([]section, map[string]bool, error) {
	pending, err := pendingVersions(root)
	if err != nil {
		return nil, nil, err
	}
	var out []section
	omit := map[string]bool{}
	for _, p := range pending {
		path := files[p.String()]
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, err
		}
		s, err := parseSection(string(data), kinds)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", filepath.ToSlash(path), err)
		}
		out = append(out, s)
		omit[filepath.Base(path)] = true
	}
	return out, omit, nil
}

// prependNotes rewrites the version file at path, as changie batch wrote it, with the notes of
// ahead before its own.
func prependNotes(path string, ahead []section, kinds []string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	s, err := parseSection(string(data), kinds)
	if err != nil {
		return fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	all := slices.Concat(ahead, []section{s})
	return os.WriteFile(path, []byte(combine(s, all, kinds).markdown()), 0o644)
}

// check reports every problem it finds with the notes, then fails if there was one:
//
//   - each .changes/vX.Y.Z.md parses (see parseSection) and is named for its version;
//   - changie can batch the unreleased notes, and each comes out as one short line;
//   - CHANGELOG.md is exactly what changie merge writes;
//   - every release tag that HEAD contains has its .changes/vX.Y.Z.md (only where tags are fetched),
//     except one at HEAD itself: that commit is the release, which release.yml batches.
func (r *runner) check() error {
	kinds, err := configKinds(filepath.Join(r.root, configFile))
	if err != nil {
		return err
	}
	var problems []string
	if _, err := readSections(filepath.Join(r.root, changesDir), kinds); err != nil {
		problems = append(problems, err.Error())
	}

	c, err := r.changieBin()
	if err != nil {
		return err
	}
	// The placeholder version only names the dry run's header.
	batch, err := c.run(r.root, "", "batch", "v0.0.0-unreleased", "--dry-run")
	if err != nil {
		problems = append(problems, "the unreleased notes: "+err.Error())
	} else if _, err := parseSection(batch, kinds); err != nil {
		problems = append(problems, "the unreleased notes: "+err.Error())
	}

	merged, err := c.run(r.root, "", "merge", "--dry-run")
	if err != nil {
		problems = append(problems, err.Error())
	} else {
		cur, err := os.ReadFile(filepath.Join(r.root, changelogFile))
		if err != nil || strings.ReplaceAll(string(cur), "\r\n", "\n") != merged {
			problems = append(problems, changelogFile+" is not what changie merge writes; run: go tool -modfile=tools/release/go.mod changie merge")
		}
	}

	behind, err := unbatchedTags(r.root, nil, true)
	if err != nil {
		return err
	}
	for _, t := range behind {
		problems = append(problems, fmt.Sprintf("%s was released without .changes/%s.md; run: go run ./internal/tools/relnotes -catch-up", t.name, t.name))
	}

	for _, p := range problems {
		r.annotate("error", p)
	}
	if len(problems) > 0 {
		return errCheck
	}
	return nil
}

// testOnly are the packages under internal/ that only tests and tools use.
var testOnly = []string{
	"internal/tools/", "internal/archtest/", "internal/e2e/", "internal/golden/",
	"internal/games/testkit/", "internal/games/gamestest/",
}

// shipped are the files outside cmd/ and internal/ that decide what players get: how the archives
// and the Linux packages are built and what they hold, and the manual page they carry.
var shipped = []string{".goreleaser.yaml", "packaging/", "docs/man/"}

// userVisible is the -since heuristic for "a player could notice this": a file under cmd/ or
// internal/ that is not a test, not test data, and not in a test-only package, or one of shipped.
// It errs towards asking for a note; "Changelog: none" answers a false alarm.
func userVisible(path string) bool {
	for _, p := range shipped {
		if path == p || (strings.HasSuffix(p, "/") && strings.HasPrefix(path, p)) {
			return true
		}
	}
	if !strings.HasPrefix(path, "cmd/") && !strings.HasPrefix(path, "internal/") {
		return false
	}
	if strings.HasSuffix(path, "_test.go") || strings.Contains("/"+path, "/testdata/") {
		return false
	}
	for _, dir := range testOnly {
		if strings.HasPrefix(path, dir) {
			return false
		}
	}
	return true
}

// isNote reports whether a path is a change note or a batched version.
func isNote(path string) bool {
	dir, name := filepath.ToSlash(filepath.Dir(path)), filepath.Base(path)
	return (dir == unreleasedDir && strings.HasSuffix(name, ".yaml")) ||
		(dir == changesDir && strings.HasPrefix(name, "v") && strings.HasSuffix(name, ".md"))
}

// since fails when the branch, compared with its merge base with base, changes a user-visible
// file but adds or edits no change note, unless one of its commits has a "Changelog: none" trailer.
func (r *runner) since(base string) error {
	mb, err := git(r.root, "merge-base", base, "HEAD")
	if err != nil {
		return err
	}
	mb = strings.TrimSpace(mb)
	diff, err := git(r.root, "diff", "-z", "--no-renames", "--name-status", mb, "HEAD")
	if err != nil {
		return err
	}
	var visible []string
	noted := false
	fields := strings.Split(strings.TrimSuffix(diff, "\x00"), "\x00")
	for i := 0; i+1 < len(fields); i += 2 {
		status, path := fields[i], fields[i+1]
		if userVisible(path) {
			visible = append(visible, path)
		}
		if status != "D" && isNote(path) {
			noted = true
		}
	}
	if len(visible) == 0 || noted {
		return nil
	}
	trailers, err := git(r.root, "log", "--format=%(trailers:key=Changelog,valueonly)", mb+"..HEAD")
	if err != nil {
		return err
	}
	for _, line := range strings.Split(trailers, "\n") {
		if strings.EqualFold(strings.TrimSpace(line), "none") {
			return nil
		}
	}
	shown := visible
	if len(shown) > 5 {
		shown = append(shown[:5:5], fmt.Sprintf("and %d more", len(visible)-5))
	}
	r.annotate("error", fmt.Sprintf("this branch changes what players get (%s) but adds no change note. Add one with `go tool -modfile=tools/release/go.mod changie new`, or, if players will not notice, add \"Changelog: none\" as a trailer in a commit message's last paragraph, next to any Co-Authored-By lines (AGENTS.md, Change notes)",
		strings.Join(shown, ", ")))
	return errCheck
}

// catchUp writes .changes/vX.Y.Z.md for each release tag that went out with its notes batched on
// the fly: the notes waiting in .changes/unreleased at that tag (and not at an earlier one), dated
// with the tag's commit. It removes those notes from .changes/unreleased, where they are unchanged,
// and merges CHANGELOG.md again.
func (r *runner) catchUp() error {
	tags, err := releaseTags(r.root)
	if err != nil {
		return err
	}
	files, err := versionFiles(r.root)
	if err != nil {
		return err
	}
	seen := map[string]bool{} // name and content of every note waiting at an earlier tag
	wrote := 0
	for _, t := range tags {
		frags, err := fragmentsAt(r.root, t.name)
		if err != nil {
			return err
		}
		fresh := map[string][]byte{}
		for name, data := range frags {
			if key := name + "\x00" + string(data); !seen[key] {
				fresh[name] = data
				seen[key] = true
			}
		}
		if _, ok := files[t.version.String()]; ok {
			continue
		}
		if !hasFile(r.root, t.name, configFile) {
			r.annotate("warning", fmt.Sprintf("%s predates changie; write .changes/%s.md by hand", t.name, t.name))
			continue
		}
		if err := r.batchTag(t, fresh); err != nil {
			return err
		}
		wrote++
	}
	if wrote == 0 {
		r.logf("relnotes: every release tag has its .changes/vX.Y.Z.md\n")
		return nil
	}
	c, err := r.changieBin()
	if err != nil {
		return err
	}
	if _, err := c.run(r.root, "", "merge"); err != nil {
		return err
	}
	r.logf("relnotes: merged %s; review and commit the changes\n", changelogFile)
	return nil
}

// batchTag batches a tag's notes in a temporary copy and brings back its version file, then removes
// the notes it used from the tree.
func (r *runner) batchTag(t tag, frags map[string][]byte) error {
	date, err := commitDate(r.root, t.name)
	if err != nil {
		return err
	}
	dir, err := r.stage(frags, nil)
	if err != nil {
		return err
	}
	c, err := r.changieBin()
	if err != nil {
		return err
	}
	if _, err := c.run(dir, date, "batch", t.name); err != nil {
		return err
	}
	name := t.name + ".md"
	if err := copyFile(filepath.Join(dir, changesDir, name), filepath.Join(r.root, changesDir, name)); err != nil {
		return err
	}
	r.logf("relnotes: wrote %s/%s from the %d notes waiting at %s\n", changesDir, name, len(frags), t.name)
	names := make([]string, 0, len(frags))
	for n := range frags {
		names = append(names, n)
	}
	slices.Sort(names)
	for _, n := range names {
		p := filepath.Join(r.root, filepath.FromSlash(unreleasedDir), n)
		cur, err := os.ReadFile(p)
		switch {
		case errors.Is(err, os.ErrNotExist):
		case err != nil:
			return err
		case string(cur) != string(frags[n]):
			r.annotate("warning", fmt.Sprintf("%s/%s changed after %s; left in place, so check that %s did not already ship it", unreleasedDir, n, t.name, t.name))
		default:
			if err := os.Remove(p); err != nil {
				return err
			}
		}
	}
	return nil
}
