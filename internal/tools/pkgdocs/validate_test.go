package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// requireValidators, set to 1, makes a missing validator fail TestValidators instead of skipping
// it. CI's lint job sets it after installing them.
const requireValidators = "WOPR_DESKTOP_VALIDATORS"

// releaseChangelog is changelog.yml as relnotes writes it for a release, 0.4.0, with more older
// releases than the metadata describes, so that one of them has only its page.
const releaseChangelog = `[
  {"semver": "0.4.0-1", "date": "2026-10-08T21:06:44Z", "changes": [{"note": "A new Mac app, in a .dmg."}]},
  {"semver": "0.3.0-1", "date": "2026-10-07T18:12:09Z", "changes": [{"note": "A new game."}]},
  {"semver": "0.2.0-1", "date": "2026-10-07T15:32:04Z", "changes": [{"note": "Fixed a crash."}]},
  {"semver": "0.1.0-1", "date": "2026-10-07T09:00:00Z", "changes": [{"note": "The first release."}]}
]
`

// The validators that distributions and software centres use, each with the files pkgdocs writes
// that it checks, and a way to break one that it must report.
var validators = []struct {
	name   string
	args   []string // before the file
	quiet  bool     // any output fails: desktop-file-validate exits 0 on warnings and hints
	files  []string // in pkgdocs's output directory
	broken func(string) string
}{{
	name:   "desktop-file-validate",
	quiet:  true,
	files:  []string{"deb/" + appID + ".desktop", "rpm/" + appID + ".desktop"},
	broken: func(s string) string { return strings.Replace(s, "\nType=Application\n", "\nType=App\n", 1) },
}, {
	name:   "appstreamcli",
	args:   []string{"validate", "--no-net", "--pedantic"},
	files:  []string{appID + ".metainfo.xml"},
	broken: notAnApp,
}, {
	name:   "appstream-util", // validate-relax is the check Fedora's packaging guidelines ask for
	args:   []string{"validate-relax", "--nonet"},
	files:  []string{appID + ".metainfo.xml"},
	broken: notAnApp,
}}

// notAnApp breaks AppStream metadata: "game" is no component type.
func notAnApp(s string) string {
	return strings.Replace(s, `type="desktop-application"`, `type="game"`, 1)
}

// The menu entries and the metadata pkgdocs writes pass the validators that distributions use, which
// check what the Go tests do not (docs/PLAN.md §8): both forms of the newest release, a snapshot's and
// a release's, and older releases with and without their notes. The validators are Linux tools, not
// Go modules: each subtest skips without its tool, unless WOPR_DESKTOP_VALIDATORS=1. Each must also
// report a file broken on purpose, so that a validator that checks nothing cannot pass.
func TestValidators(t *testing.T) {
	t.Parallel()
	out := map[string]string{}
	for shape, c := range map[string]struct{ version, changelog string }{
		"snapshot": {"0.3.0-snapshot.abc1234", testChangelog},
		"release":  {"0.4.0", releaseChangelog},
	} {
		notes := t.TempDir()
		if err := os.WriteFile(filepath.Join(notes, "changelog.yml"), []byte(c.changelog), 0o644); err != nil {
			t.Fatal(err)
		}
		out[shape] = t.TempDir()
		if err := writeLinux(repoRoot, notes, out[shape], c.version); err != nil {
			t.Fatal(err)
		}
	}
	for _, v := range validators {
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			if _, err := exec.LookPath(v.name); err != nil {
				if os.Getenv(requireValidators) == "1" {
					t.Fatalf("%s=1, but %s is not on the PATH", requireValidators, v.name)
				}
				t.Skipf("%s is not on the PATH (on Debian and Ubuntu: sudo apt install desktop-file-utils appstream appstream-util)", v.name)
			}
			ver, _ := exec.Command(v.name, "--version").CombinedOutput()
			version := strings.Join(strings.Fields(string(ver)), " ") // appstream-util's has a tab
			run := func(file string) (string, error) {
				report, err := exec.Command(v.name, append(v.args, file)...).CombinedOutput()
				if err == nil && v.quiet && len(report) > 0 {
					err = errors.New("it printed a report")
				}
				return string(report), err
			}
			for shape, dir := range out {
				for _, f := range v.files {
					if report, err := run(filepath.Join(dir, filepath.FromSlash(f))); err != nil {
						t.Errorf("%s (%s) on the %s's %s: %v\n%s", v.name, version, shape, f, err, report)
					}
				}
			}
			f := filepath.Join(out["release"], filepath.FromSlash(v.files[0]))
			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			broken := v.broken(string(data))
			if broken == string(data) {
				t.Fatalf("breaking %s changed nothing", f)
			}
			bad := filepath.Join(t.TempDir(), filepath.Base(f))
			if err := os.WriteFile(bad, []byte(broken), 0o644); err != nil {
				t.Fatal(err)
			}
			if report, err := run(bad); err == nil {
				t.Errorf("%s passed a broken %s:\n%s", v.name, filepath.Base(f), report)
			}
		})
	}
}
