package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// testChangie is the pinned changie, built once for every test that runs it.
var testChangie changie

func TestMain(m *testing.M) {
	os.Exit(func() int {
		dir, err := os.MkdirTemp("", "relnotes-test-")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		defer func() { _ = os.RemoveAll(dir) }()
		root, err := findModuleRoot()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		if testChangie, err = buildChangie(root, dir); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		return m.Run()
	}())
}

func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if data, err := os.ReadFile(filepath.Join(dir, "go.mod")); err == nil &&
			strings.HasPrefix(string(data), "module github.com/GhostofGoes/WOPR\n") {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no wopr go.mod above %s", dir)
		}
		dir = parent
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	root, err := findModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// testRepo is a scratch git repository with this repository's changie configuration.
type testRepo struct {
	t   *testing.T
	dir string
	log bytes.Buffer
}

// newRepo isolates git from the user's configuration (signing, hooks, line endings) and starts a
// repository on main. The tests that use it set environment variables, so they do not run in
// parallel.
func newRepo(t *testing.T) *testRepo {
	t.Helper()
	global := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(global, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, who := range []string{"AUTHOR", "COMMITTER"} {
		t.Setenv("GIT_"+who+"_NAME", "relnotes test")
		t.Setenv("GIT_"+who+"_EMAIL", "relnotes-test")
	}
	t.Setenv("GITHUB_ACTIONS", "")
	r := &testRepo{t: t, dir: t.TempDir()}
	r.git("-c", "init.defaultBranch=main", "init", "-q")
	root := moduleRoot(t)
	for _, p := range []string{configFile, changesDir + "/header.tpl.md"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
		if err != nil {
			t.Fatal(err)
		}
		r.write(p, string(data))
	}
	r.write(unreleasedDir+"/.gitkeep", "")
	return r
}

func (r *testRepo) git(args ...string) string {
	r.t.Helper()
	return r.gitEnv(nil, args...)
}

func (r *testRepo) gitEnv(env []string, args ...string) string {
	r.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.dir
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func (r *testRepo) path(p string) string { return filepath.Join(r.dir, filepath.FromSlash(p)) }

func (r *testRepo) write(p, content string) {
	r.t.Helper()
	if err := os.MkdirAll(filepath.Dir(r.path(p)), 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(r.path(p), []byte(content), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

func (r *testRepo) read(p string) string {
	r.t.Helper()
	data, err := os.ReadFile(r.path(p))
	if err != nil {
		r.t.Fatal(err)
	}
	return string(data)
}

func (r *testRepo) exists(p string) bool {
	_, err := os.Stat(r.path(p))
	return err == nil
}

// note adds a change note as changie new writes one.
func (r *testRepo) note(name, kind, body, time string) {
	r.write(unreleasedDir+"/"+name+".yaml", fmt.Sprintf("kind: %s\nbody: %s\ntime: %s\n", kind, body, time))
}

// commit commits everything at a fixed time (RFC 3339).
func (r *testRepo) commit(when, msg string) {
	r.t.Helper()
	r.git("add", "-A")
	r.gitEnv([]string{"GIT_AUTHOR_DATE=" + when, "GIT_COMMITTER_DATE=" + when}, "commit", "-q", "--allow-empty", "-m", msg)
}

// releasePR does what a release pull request does: changie batch, then changie merge.
func (r *testRepo) releasePR(version, date string) {
	r.t.Helper()
	if _, err := testChangie.run(r.dir, date, "batch", version); err != nil {
		r.t.Fatal(err)
	}
	if _, err := testChangie.run(r.dir, "", "merge"); err != nil {
		r.t.Fatal(err)
	}
}

func (r *testRepo) runner() *runner {
	run := &runner{root: r.dir, log: &r.log, changie: testChangie}
	r.t.Cleanup(run.cleanup)
	return run
}

func (r *testRepo) clean() {
	r.t.Helper()
	if st := r.git("status", "--porcelain"); st != "" {
		r.t.Errorf("relnotes changed the git tree:\n%s", st)
	}
}

// release runs relnotes for a version and returns the three files it wrote.
func (r *testRepo) release(version string, snapshot bool, date string) map[string]string {
	r.t.Helper()
	out := filepath.Join(r.t.TempDir(), "notes")
	if err := r.runner().release(version, snapshot, date, defaultPackager, out); err != nil {
		r.t.Fatalf("release: %v\n%s", err, r.log.String())
	}
	files := map[string]string{}
	for _, name := range []string{"notes.md", changelogFile, "changelog.yml"} {
		data, err := os.ReadFile(filepath.Join(out, name))
		if err != nil {
			r.t.Fatal(err)
		}
		files[name] = string(data)
	}
	return files
}

func entries(t *testing.T, yml string) []chglogEntry {
	t.Helper()
	var e []chglogEntry
	if err := json.Unmarshal([]byte(yml), &e); err != nil {
		t.Fatalf("changelog.yml: %v\n%s", err, yml)
	}
	return e
}

// v010 makes the v0.1.0 release the way a release pull request does.
func v010(t *testing.T) *testRepo {
	t.Helper()
	r := newRepo(t)
	r.note("Added-first", "Added", "The first release.", "2026-10-01T10:00:00Z")
	r.commit("2026-10-01T10:00:00Z", "start")
	r.releasePR("v0.1.0", "2026-10-01")
	r.commit("2026-10-01T11:00:00Z", "v0.1.0")
	r.git("tag", "v0.1.0")
	return r
}

// A release whose pull request batched its notes uses that file as it is.
func TestReleaseFromBatchedNotes(t *testing.T) {
	r := v010(t)
	r.note("Fixed-chess", "Fixed", "Fixed an issue with the Chess game.", "2026-10-03T10:00:00Z")
	r.note("Security-go", "Security", "'Updated Go: it fixes a problem `go` had.'", "2026-10-03T11:00:00Z")
	r.commit("2026-10-03T12:00:00Z", "fixes")
	r.releasePR("0.2.0", "2026-10-05") // without the v, as a person might type it
	r.commit("2026-10-05T12:00:00Z", "v0.2.0")
	r.git("tag", "v0.2.0")

	files := r.release("v0.2.0", false, "")
	if strings.Contains(r.log.String(), "warning") {
		t.Errorf("a batched release must not warn:\n%s", r.log.String())
	}
	if !r.exists(changesDir + "/v0.2.0.md") {
		t.Fatal("changie batch 0.2.0 must write .changes/v0.2.0.md")
	}
	wantNotes := "### Fixed\n\n- Fixed an issue with the Chess game.\n\n### Security\n\n- Updated Go: it fixes a problem `go` had.\n\n---\n\n"
	if !strings.HasPrefix(files["notes.md"], wantNotes) ||
		!strings.HasSuffix(files["notes.md"], "/compare/v0.1.0...v0.2.0\n") {
		t.Errorf("notes.md =\n%s", files["notes.md"])
	}
	if files[changelogFile] != r.read(changelogFile) {
		t.Errorf("CHANGELOG.md differs from the merged one:\n%s", files[changelogFile])
	}
	e := entries(t, files["changelog.yml"])
	if len(e) != 2 || e[0].Semver != "0.2.0-1" || e[0].Date != "2026-10-05T12:00:00Z" || e[0].Deb.Urgency != "high" ||
		e[0].Changes[1].Note != "Updated Go: it fixes a problem go had." ||
		e[1].Semver != "0.1.0-1" || e[1].Date != "2026-10-01T11:00:00Z" || e[1].Deb.Urgency != "medium" {
		t.Errorf("changelog.yml =\n%s", files["changelog.yml"])
	}
	r.clean()
}

// A tag pushed without a batched file still gets its notes, dated by the tagged commit in UTC,
// the same every time, and the git tree is left alone.
func TestReleaseBatchesOnTheFly(t *testing.T) {
	r := v010(t)
	r.note("Added-seed", "Added", "'`WOPR_SEED` sets the seed.'", "2026-10-09T10:00:00Z")
	r.commit("2026-10-09T23:30:00-05:00", "seed") // 2026-10-10 in UTC
	r.git("tag", "v0.2.0")

	files := r.release("0.2.0", false, "")
	if !strings.Contains(r.log.String(), "warning") || !strings.Contains(r.log.String(), "-catch-up") {
		t.Errorf("an unbatched release must warn and say what to do:\n%s", r.log.String())
	}
	if !strings.HasPrefix(files["notes.md"], "### Added\n\n- `WOPR_SEED` sets the seed.\n") {
		t.Errorf("notes.md =\n%s", files["notes.md"])
	}
	if !strings.Contains(files[changelogFile], "## v0.2.0 - 2026-10-10\n\n### Added\n\n- `WOPR_SEED` sets the seed.\n\n## v0.1.0 - 2026-10-01\n") {
		t.Errorf("CHANGELOG.md =\n%s", files[changelogFile])
	}
	if e := entries(t, files["changelog.yml"]); len(e) != 2 || e[0].Date != "2026-10-10T04:30:00Z" {
		t.Errorf("changelog.yml =\n%s", files["changelog.yml"])
	}
	again := r.release("0.2.0", false, "")
	for name, data := range files {
		if again[name] != data {
			t.Errorf("%s differs between two runs", name)
		}
	}
	if dated := r.release("0.2.0", false, "2026-12-25"); !strings.Contains(dated[changelogFile], "## v0.2.0 - 2026-12-25\n") {
		t.Errorf("-date must set the date:\n%s", dated[changelogFile])
	}
	if !r.exists(unreleasedDir + "/Added-seed.yaml") {
		t.Error("the note must stay in .changes/unreleased")
	}
	r.clean()
}

// A snapshot is named as GoReleaser names it and is batched without a warning.
func TestSnapshot(t *testing.T) {
	r := v010(t)
	r.note("Fixed-maze", "Fixed", "Fixed a crash in some cases.", "2026-10-02T10:00:00Z")
	r.commit("2026-10-02T10:00:00Z", "maze")
	short := strings.TrimSpace(r.git("rev-parse", "--short", "HEAD"))

	files := r.release("", true, "")
	if strings.Contains(r.log.String(), "warning") {
		t.Errorf("a snapshot must not warn:\n%s", r.log.String())
	}
	if e := entries(t, files["changelog.yml"]); e[0].Semver != "0.1.1~snapshot."+short+"-1" || e[0].Date != "2026-10-02T10:00:00Z" ||
		e[0].Changes[0].Note != "Fixed a crash in some cases." {
		t.Errorf("changelog.yml =\n%s", files["changelog.yml"])
	}
	if !strings.HasSuffix(files["notes.md"], "/compare/v0.1.0..."+short+"\n") {
		t.Errorf("a snapshot compares with its commit:\n%s", files["notes.md"])
	}
	if !strings.Contains(files[changelogFile], "## v0.1.1-snapshot."+short+" - 2026-10-02\n") {
		t.Errorf("CHANGELOG.md =\n%s", files[changelogFile])
	}
	r.clean()
}

// A release after one that went out unbatched is refused until -catch-up commits the earlier
// notes; then each version gets only its own notes.
func TestCatchUp(t *testing.T) {
	r := v010(t)
	r.note("Added-seed", "Added", "Seeds.", "2026-10-09T10:00:00Z")
	r.commit("2026-10-09T12:00:00Z", "seed")
	r.git("tag", "v0.2.0") // released on the fly
	r.note("Fixed-chess", "Fixed", "Fixed an issue with the Chess game.", "2026-10-11T10:00:00Z")
	r.commit("2026-10-11T12:00:00Z", "chess")
	r.git("tag", "v0.3.0")

	err := r.runner().release("0.3.0", false, "", defaultPackager, t.TempDir())
	if !errors.Is(err, errCheck) || !strings.Contains(r.log.String(), "v0.2.0") {
		t.Fatalf("release = %v, want a refusal naming v0.2.0:\n%s", err, r.log.String())
	}
	if err := r.runner().check(); !errors.Is(err, errCheck) || !strings.Contains(r.log.String(), "-catch-up") {
		t.Errorf("check = %v, want it to ask for -catch-up:\n%s", err, r.log.String())
	}

	r.log.Reset()
	if err := r.runner().catchUp(); err != nil {
		t.Fatalf("catchUp: %v\n%s", err, r.log.String())
	}
	if got, want := r.read(changesDir+"/v0.2.0.md"), "## v0.2.0 - 2026-10-09\n\n### Added\n\n- Seeds.\n"; got != want {
		t.Errorf("v0.2.0.md =\n%s\nwant\n%s", got, want)
	}
	if !r.exists(changesDir+"/v0.3.0.md") || r.exists(unreleasedDir+"/Added-seed.yaml") || r.exists(unreleasedDir+"/Fixed-chess.yaml") {
		t.Error("catch-up must batch v0.2.0 and v0.3.0 and remove their notes")
	}
	if got := r.read(changesDir + "/v0.3.0.md"); strings.Contains(got, "Seeds.") || !strings.Contains(got, "Chess") {
		t.Errorf("v0.3.0 must hold only its own note:\n%s", got)
	}
	if err := r.runner().check(); err != nil {
		t.Errorf("check after catch-up: %v\n%s", err, r.log.String())
	}
	r.commit("2026-10-12T12:00:00Z", "catch up")
	r.note("Fixed-maze", "Fixed", "Fixed a crash in some cases.", "2026-10-12T10:00:00Z")
	r.commit("2026-10-13T12:00:00Z", "maze")
	r.git("tag", "v0.4.0")
	files := r.release("0.4.0", false, "")
	if strings.Contains(files["notes.md"], "Chess") || strings.Contains(files["notes.md"], "Seeds") {
		t.Errorf("v0.4.0 repeats earlier notes:\n%s", files["notes.md"])
	}
}

func TestCheck(t *testing.T) {
	r := v010(t)
	if err := r.runner().check(); err != nil {
		t.Fatalf("check: %v\n%s", err, r.log.String())
	}
	for _, tc := range []struct {
		name, want string
		breakIt    func()
	}{
		{"edited changelog", "CHANGELOG.md is not what changie merge writes", func() {
			r.write(changelogFile, r.read(changelogFile)+"\n- An extra line.\n")
		}},
		{"long note", "more than 76", func() {
			r.note("Added-long", "Added", strings.Repeat("word ", 20), "2026-10-02T10:00:00Z")
		}},
		{"unknown kind", "the unreleased notes", func() {
			r.note("Feature-x", "Feature", "Something new.", "2026-10-02T10:00:00Z")
		}},
		{"bad version file", "v0.0.9.md", func() {
			r.write(changesDir+"/v0.0.9.md", "## v0.0.9 - 2026-09-01\n\nSome words.\n")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r.git("reset", "-q", "--hard") // back to the committed, valid tree
			r.git("clean", "-fdq")
			tc.breakIt()
			r.log.Reset()
			if err := r.runner().check(); !errors.Is(err, errCheck) || !strings.Contains(r.log.String(), tc.want) {
				t.Errorf("check = %v, want a problem with %q:\n%s", err, tc.want, r.log.String())
			}
		})
	}
}

func TestSince(t *testing.T) {
	r := newRepo(t)
	r.write("internal/game/game.go", "package game\n")
	r.commit("2026-10-01T10:00:00Z", "start")

	for _, tc := range []struct {
		name   string
		change func()
		msg    string
		ok     bool
	}{
		{"tests only", func() { r.write("internal/game/game_test.go", "package game\n") }, "test", true},
		{"tools only", func() { r.write("internal/tools/x/main.go", "package main\n") }, "tool", true},
		{"docs only", func() { r.write("README.md", "# wopr\n") }, "docs", true},
		{"code", func() { r.write("internal/game/game.go", "package game // changed\n") }, "code", false},
		{"code and a note", func() {
			r.write("cmd/wopr/main.go", "package main\n")
			r.note("Fixed-x", "Fixed", "Fixed a crash in some cases.", "2026-10-02T10:00:00Z")
		}, "code", true},
		{"code and a release", func() {
			r.write("cmd/wopr/main.go", "package main\n")
			r.write(changesDir+"/v0.1.0.md", "## v0.1.0 - 2026-10-01\n")
		}, "code", true},
		{
			"code, Changelog: none", func() { r.write("internal/game/game.go", "package game // refactor\n") },
			"Refactor\n\nNothing a player sees.\n\nChangelog: none", true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r.git("checkout", "-q", "-B", "work", "main")
			tc.change()
			r.commit("2026-10-02T10:00:00Z", tc.msg)
			r.log.Reset()
			err := r.runner().since("main")
			if tc.ok && err != nil {
				t.Errorf("since = %v\n%s", err, r.log.String())
			}
			if !tc.ok && (!errors.Is(err, errCheck) || !strings.Contains(r.log.String(), "internal/game/game.go")) {
				t.Errorf("since = %v, want a failure naming the file:\n%s", err, r.log.String())
			}
		})
	}
}
