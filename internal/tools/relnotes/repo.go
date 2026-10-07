package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	configFile    = ".changie.yaml"
	changesDir    = ".changes"
	unreleasedDir = ".changes/unreleased"
	changelogFile = "CHANGELOG.md"
)

// git runs git in dir and returns its standard output. Signatures stay out of log output whatever
// the user's configuration says.
func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-c", "log.showSignature=false"}, args...)...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}

// tag is a release tag reachable from HEAD.
type tag struct {
	name    string // "v1.2.3"
	version semver
}

// releaseTags lists the vX.Y.Z tags that HEAD contains, oldest first. Tags that are not versions
// are skipped.
func releaseTags(root string) ([]tag, error) {
	out, err := git(root, "tag", "--merged", "HEAD", "--list", "v*")
	if err != nil {
		return nil, err
	}
	var tags []tag
	for _, name := range strings.Fields(out) {
		if v, err := parseSemver(name); err == nil && strings.HasPrefix(name, "v") {
			tags = append(tags, tag{name: name, version: v})
		}
	}
	slices.SortFunc(tags, func(a, b tag) int { return a.version.compare(b.version) })
	return tags, nil
}

// commitDate is the UTC date (YYYY-MM-DD) of a commit, or of the commit a tag points at.
func commitDate(root, rev string) (string, error) {
	t, err := commitTime(root, rev)
	if err != nil {
		return "", err
	}
	return t.Format(time.DateOnly), nil
}

// commitTime is the time, in UTC, of a commit, or of the commit a tag points at.
func commitTime(root, rev string) (time.Time, error) {
	out, err := git(root, "log", "-1", "--format=%ct", rev+"^{commit}", "--")
	if err != nil {
		return time.Time{}, err
	}
	secs, err := strconv.ParseInt(strings.TrimSpace(out), 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("commit time of %s: %w", rev, err)
	}
	return time.Unix(secs, 0).UTC(), nil
}

// releaseTimes maps every release tag that HEAD contains to its commit's time, and head, the
// version being built, to HEAD's. The package changelogs date each version by it: a version file
// holds only a day, and a Debian changelog's newest entry must be dated after the one below it,
// which two releases on the same day would not be.
func releaseTimes(root string, head semver) (map[semver]time.Time, error) {
	tags, err := releaseTags(root)
	if err != nil {
		return nil, err
	}
	times := make(map[semver]time.Time, len(tags)+1)
	for _, t := range tags {
		if times[t.version], err = commitTime(root, t.name); err != nil {
			return nil, err
		}
	}
	if times[head], err = commitTime(root, "HEAD"); err != nil {
		return nil, err
	}
	return times, nil
}

// snapshotVersion is the version GoReleaser gives a snapshot build with .goreleaser.yaml's
// version_template, "{{ incpatch .Version }}-snapshot.{{ .ShortCommit }}": the latest tag
// (v0.0.0 if there is none) with its patch number raised, and HEAD's short hash. ref is that hash.
func snapshotVersion(root string) (v semver, ref string, err error) {
	latest := "v0.0.0"
	if out, err := git(root, "describe", "--tags", "--abbrev=0"); err == nil {
		latest = strings.TrimSpace(out)
	}
	v, err = parseSemver(latest)
	if err != nil {
		return semver{}, "", fmt.Errorf("latest tag: %w", err)
	}
	out, err := git(root, "rev-parse", "--short", "HEAD")
	if err != nil {
		return semver{}, "", err
	}
	ref = strings.TrimSpace(out)
	return semver{major: v.major, minor: v.minor, patch: v.patch + 1, pre: "snapshot." + ref}, ref, nil
}

// fragmentsAt returns the change notes waiting in .changes/unreleased at a tag: file name to content.
func fragmentsAt(root, rev string) (map[string][]byte, error) {
	out, err := git(root, "ls-tree", "-z", "--name-only", rev, "--", unreleasedDir+"/")
	if err != nil {
		return nil, err
	}
	frags := map[string][]byte{}
	for _, p := range strings.Split(out, "\x00") {
		if !strings.HasSuffix(p, ".yaml") {
			continue
		}
		data, err := git(root, "cat-file", "blob", rev+":"+p)
		if err != nil {
			return nil, err
		}
		frags[filepath.Base(p)] = []byte(data)
	}
	return frags, nil
}

// hasFile reports whether a file exists at a revision.
func hasFile(root, rev, path string) bool {
	_, err := git(root, "cat-file", "-e", rev+":"+path)
	return err == nil
}

// changie runs a changie binary built from tools/release/go.mod.
type changie string

// buildChangie builds the pinned changie into dir. go tool cannot run it outside the module, and
// relnotes runs it in temporary copies of .changes.
func buildChangie(root, dir string) (changie, error) {
	bin := filepath.Join(dir, "changie")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command("go", "build", "-modfile=tools/release/go.mod", "-o", bin, "github.com/miniscruff/changie")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("building changie: %w\n%s", err, out)
	}
	return changie(bin), nil
}

// run runs changie in dir. Inherited CHANGIE_ variables are dropped, so that nothing but date (when
// not empty) reaches the templates: CHANGIE_CONFIG_PATH, for one, would read another config.
func (c changie) run(dir, date string, args ...string) (string, error) {
	cmd := exec.Command(string(c), args...)
	cmd.Dir = dir
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(strings.ToUpper(kv), "CHANGIE_") {
			cmd.Env = append(cmd.Env, kv)
		}
	}
	if date != "" {
		cmd.Env = append(cmd.Env, "CHANGIE_DATE="+date)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("changie %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}

// stage copies .changie.yaml and .changes into a new temporary directory, where changie can batch
// and merge without touching the git tree. With frags not nil, the unreleased notes are frags
// instead of the tree's.
func stage(root string, frags map[string][]byte) (string, error) {
	dir, err := os.MkdirTemp("", "relnotes-")
	if err != nil {
		return "", err
	}
	if err := copyFile(filepath.Join(root, configFile), filepath.Join(dir, configFile)); err != nil {
		return dir, err
	}
	src := filepath.Join(root, changesDir)
	err = filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		dst := filepath.Join(dir, changesDir, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		if frags != nil && filepath.Dir(rel) == "unreleased" && strings.HasSuffix(rel, ".yaml") {
			return nil
		}
		return copyFile(p, dst)
	})
	if err != nil {
		return dir, err
	}
	unreleased := filepath.Join(dir, filepath.FromSlash(unreleasedDir))
	if err := os.MkdirAll(unreleased, 0o755); err != nil { // changie batch fails without it
		return dir, err
	}
	for name, data := range frags {
		if err := os.WriteFile(filepath.Join(unreleased, name), data, 0o644); err != nil {
			return dir, err
		}
	}
	return dir, nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

// versionFiles maps each version with a .changes/vX.Y.Z.md file to that file.
func versionFiles(root string) (map[string]string, error) {
	paths, err := filepath.Glob(filepath.Join(root, changesDir, "v*.md"))
	if err != nil {
		return nil, err
	}
	files := map[string]string{}
	for _, p := range paths {
		v, err := parseSemver(strings.TrimSuffix(filepath.Base(p), ".md"))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.ToSlash(p), err)
		}
		files[v.String()] = p
	}
	return files, nil
}

// unbatchedTags lists the tags that HEAD contains, below limit (all of them if limit is nil),
// that have no .changes/vX.Y.Z.md: releases whose notes were batched on the fly and never
// committed.
func unbatchedTags(root string, limit *semver) ([]tag, error) {
	tags, err := releaseTags(root)
	if err != nil {
		return nil, err
	}
	files, err := versionFiles(root)
	if err != nil {
		return nil, err
	}
	var out []tag
	for _, t := range tags {
		if limit != nil && t.version.compare(*limit) >= 0 {
			continue
		}
		if _, ok := files[t.version.String()]; !ok {
			out = append(out, t)
		}
	}
	return out, nil
}
