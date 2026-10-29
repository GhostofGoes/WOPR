// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// requireSandbox is set where the kernel must have every feature the sandbox uses (CI), so
// that a missing one fails instead of passing quietly.
var requireSandbox = os.Getenv("WOPR_SANDBOX_REQUIRE") != ""

var statusField = regexp.MustCompile(`(?m)^(NoNewPrivs|Seccomp|Seccomp_filters|CapEff|CapPrm|CapInh|CapAmb):\s+(\S+)$`)

// checkThreads reads /proc/<pid>/task/*/status and checks the sandbox on every thread
// that wopr has: the runtime's own, and any it started after the lockdown.
func checkThreads(t *testing.T, pid int, when string) int {
	t.Helper()
	dir := filepath.Join("/proc", strconv.Itoa(pid), "task")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(dir, e.Name(), "status"))
		if err != nil {
			continue // the thread ended
		}
		n++
		st := map[string]string{}
		for _, m := range statusField.FindAllStringSubmatch(string(data), -1) {
			st[m[1]] = m[2]
		}
		// snapd puts its own filters on a snap before wopr starts, and wopr's goes on top of them.
		filters, _ := strconv.Atoi(st["Seccomp_filters"])
		if st["NoNewPrivs"] != "1" || st["Seccomp"] != "2" || filters != 1 && !snapped() || filters < 2 && snapped() {
			t.Errorf("%s, thread %s: NoNewPrivs %s, Seccomp %s, Seccomp_filters %s; want 1, 2 and 1 (2 or more in a snap)",
				when, e.Name(), st["NoNewPrivs"], st["Seccomp"], st["Seccomp_filters"])
		}
		for _, c := range []string{"CapEff", "CapPrm", "CapInh", "CapAmb"} {
			if strings.Trim(st[c], "0") != "" {
				t.Errorf("%s, thread %s: %s %s, want none", when, e.Name(), c, st[c])
			}
		}
	}
	return n
}

// The release build (no cgo) sandboxes every thread, those started after the lockdown too,
// and a game still plays: the chess engine thinks on other goroutines, the window resizes,
// and the debug log keeps its lines. Landlock does not show in /proc. The sandbox package's
// tests check what it refuses, with the same Restrict this binary calls.
func TestLinuxSandbox(t *testing.T) {
	var env []string
	cache := ""
	if !snapped() { // the snap cannot write the log where the test reads it
		env, cache = debugEnv(t)
	}
	s := startEnv(t, 80, 24, env, "chess", "--instant", "--seed", "1")
	s.waitFor("YOUR MOVE:", 10*time.Second)
	before := checkThreads(t, s.cmd.Process.Pid, "at the start")
	s.send("e2e4\r")
	s.waitFor("WOPR: ", 30*time.Second) // the engine has answered
	s.resize(100, 30)
	s.waitFor("YOUR MOVE", 10*time.Second)
	after := checkThreads(t, s.cmd.Process.Pid, "after a move")
	t.Logf("%d threads at the start, %d after the engine thought", before, after)
	s.send("\x03")
	if code := s.wait(10 * time.Second); code != 130 {
		t.Errorf("exit %d after Ctrl+C, want 130", code)
	}
	if cache == "" {
		return
	}
	log := debugLog(t, cache)
	sandbox := regexp.MustCompile(`(?m) sandbox: no_new_privs, (capabilities dropped, )?landlock ABI [1-9][0-9]*, seccomp$`)
	if !sandbox.MatchString(log) {
		if requireSandbox || !strings.Contains(log, "no landlock") {
			t.Errorf("the log must say that every layer went in:\n%s", log)
		}
	}
	inOrder(t, log, ` sandbox: `, ` think [0-9]+: `, ` session ended: outcome 1, `)
}

// Inside tmux, the colour detection runs `tmux info` (charmbracelet/colorprofile). That must
// happen before the lockdown, which refuses to run a program: a fake tmux that says it has
// true colour must raise the profile, and must run only then. A suspend and a resume do not
// run it again.
func TestTmuxColours(t *testing.T) {
	if snapped() {
		t.Skip("the snap cannot run a program from the test's /tmp")
	}
	bin := t.TempDir()
	runs := filepath.Join(bin, "runs")
	script := "#!/bin/sh\necho run >> '" + runs + "'\necho 'Tc: (flag) true'\n"
	if err := os.WriteFile(filepath.Join(bin, "tmux"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	env, cache := debugEnv(t)
	env = append(env, "TMUX=/tmp/tmux-e2e/default,1,0", "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	s := startEnv(t, 80, 24, env, "--instant")
	s.waitFor("LOGON:", 10*time.Second)
	count := func() int {
		data, _ := os.ReadFile(runs)
		return strings.Count(string(data), "run\n")
	}
	started := count()
	suspend(t, s)
	s.send("Joshua\r")
	s.waitFor("GREETINGS PROFESSOR FALKEN.", 10*time.Second)
	s.send("\x03")
	if code := s.wait(10 * time.Second); code != 130 {
		t.Errorf("exit %d after Ctrl+C, want 130", code)
	}
	if started == 0 || count() != started {
		t.Errorf("tmux ran %d times by the logon screen and %d times in all; want at least once, and only then", started, count())
	}
	inOrder(t, debugLog(t, cache), ` colour profile TrueColor, `)
}
