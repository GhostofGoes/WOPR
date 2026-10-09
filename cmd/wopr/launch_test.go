package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

const appExe = "/Applications/WOPR.app/Contents/MacOS/wopr"

func TestRelaunch(t *testing.T) {
	t.Parallel()
	opened := func(exe string) []string { return []string{"/usr/bin/open", "-b", "com.apple.Terminal", exe} }
	for _, tc := range []struct {
		name   string
		l      launch
		status []int // open's exit status, run by run (0 after the last)
		err    error // or open could not run
		want   []string
		runs   int // how many times it runs open, when it is not once
		code   int
		stderr string
	}{
		{name: "Finder", l: launch{ppid: 1, exe: appExe}, want: opened(appExe)},
		{name: "older macOS's process serial number", l: launch{ppid: 1, exe: appExe, args: []string{"-psn_0_1234567"}}, want: opened(appExe)},
		{name: "renamed and moved", l: launch{ppid: 1, exe: "/Users/d/Games/War Games.app/Contents/MacOS/wopr"}, want: opened("/Users/d/Games/War Games.app/Contents/MacOS/wopr")},
		{
			name: "translocated",
			l:    launch{ppid: 1, exe: "/private/var/folders/x1/T/AppTranslocation/0A1B/d/WOPR.app/Contents/MacOS/wopr"},
			want: opened("/private/var/folders/x1/T/AppTranslocation/0A1B/d/WOPR.app/Contents/MacOS/wopr"),
		},
		{
			name: "open fails, then works", l: launch{ppid: 1, exe: appExe}, status: []int{1, 0}, want: opened(appExe), runs: 2,
			stderr: "wopr: /usr/bin/open -b com.apple.Terminal " + appExe + " exited with status 1; trying again\n",
		},
		{
			name: "open fails twice", l: launch{ppid: 1, exe: appExe}, status: []int{1, 1}, want: opened(appExe), runs: 2, code: exitError,
			stderr: "wopr: /usr/bin/open -b com.apple.Terminal " + appExe + " exited with status 1; trying again\n" +
				"wopr: cannot open Terminal: /usr/bin/open -b com.apple.Terminal " + appExe + " exited with status 1\n",
		},
		{name: "open is missing", l: launch{ppid: 1, exe: appExe}, err: errors.New("no such file"), want: opened(appExe), code: exitError, stderr: "wopr: cannot open Terminal: no such file\n"},

		// Everything else starts as before.
		{name: "Terminal's run", l: launch{ppid: 4321, exe: appExe, stdinTTY: true, stdoutTTY: true}},
		{name: "stdin is a terminal", l: launch{ppid: 1, exe: appExe, stdinTTY: true}},
		{name: "stdout is a terminal", l: launch{ppid: 1, exe: appExe, stdoutTTY: true}},
		{name: "an option", l: launch{ppid: 1, exe: appExe, args: []string{"--version"}}},
		{name: "a game", l: launch{ppid: 1, exe: appExe, args: []string{"--play", "chess"}}},
		{name: "two arguments", l: launch{ppid: 1, exe: appExe, args: []string{"-psn_0_1234567", "--version"}}},
		{name: "not a process serial number", l: launch{ppid: 1, exe: appExe, args: []string{"-psn"}}},
		{name: "an empty argument", l: launch{ppid: 1, exe: appExe, args: []string{""}}},
		{name: "on the PATH", l: launch{ppid: 1, exe: "/usr/local/bin/wopr"}},
		{name: "Homebrew", l: launch{ppid: 1, exe: "/opt/homebrew/Cellar/wopr/1.2.3/bin/wopr"}},
		{name: "in Resources", l: launch{ppid: 1, exe: "/Applications/WOPR.app/Contents/Resources/wopr"}},
		{name: "not a bundle", l: launch{ppid: 1, exe: "/Applications/WOPR/Contents/MacOS/wopr"}},
		{name: "a bundle with no name", l: launch{ppid: 1, exe: "/Applications/.app/Contents/MacOS/wopr"}},
		{name: "relative", l: launch{ppid: 1, exe: "WOPR.app/Contents/MacOS/wopr"}},
		{name: "the MacOS folder itself", l: launch{ppid: 1, exe: "/Applications/WOPR.app/Contents/MacOS"}},
		{name: "no path", l: launch{ppid: 1}},
		{name: "a shell or a test", l: launch{ppid: 4321, exe: appExe}},
		{name: "a shell, with -psn_", l: launch{ppid: 4321, exe: appExe, args: []string{"-psn_0_1234567"}}},
	} {
		var ran []string
		runs := 0
		run := func(name string, args ...string) (int, error) {
			cmd := append([]string{name}, args...)
			if ran != nil && !slices.Equal(ran, cmd) {
				t.Errorf("%s: ran %q, then %q", tc.name, ran, cmd)
			}
			ran = cmd
			runs++
			if runs <= len(tc.status) {
				return tc.status[runs-1], tc.err
			}
			return 0, tc.err
		}
		var stderr strings.Builder
		code, relaunched := relaunch(tc.l, run, &stderr)
		if !slices.Equal(ran, tc.want) {
			t.Errorf("%s: ran %q, want %q", tc.name, ran, tc.want)
		}
		if want := max(tc.runs, min(len(tc.want), 1)); runs != want {
			t.Errorf("%s: ran open %d times, want %d", tc.name, runs, want)
		}
		if relaunched != (tc.want != nil) || code != tc.code {
			t.Errorf("%s: relaunch = %d, %v; want %d, %v", tc.name, code, relaunched, tc.code, tc.want != nil)
		}
		if stderr.String() != tc.stderr {
			t.Errorf("%s: stderr %q, want %q", tc.name, stderr.String(), tc.stderr)
		}
	}
}

// LaunchServices gives an app /dev/null; pipes and files are not terminals either. A terminal
// itself needs a pty, which the e2e tests have.
func TestIsTerminal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the relaunch is for macOS; Windows has no /dev/null")
	}
	t.Parallel()
	null, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = null.Close() }()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	defer func() { _ = w.Close() }()
	file, err := os.Create(filepath.Join(t.TempDir(), "out.txt"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	for name, f := range map[string]*os.File{"/dev/null": null, "a pipe's read end": r, "a pipe's write end": w, "a file": file} {
		if isTerminal(f) {
			t.Errorf("isTerminal(%s) = true", name)
		}
	}
}

// go test runs this binary with arguments and from no app bundle, so it starts as usual.
func TestRelaunchInTerminalLeavesOtherRunsAlone(t *testing.T) {
	t.Parallel()
	if code, relaunched := relaunchInTerminal(); relaunched {
		t.Errorf("relaunchInTerminal = %d, true", code)
	}
}
