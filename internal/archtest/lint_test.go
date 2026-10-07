package archtest

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestLintRulesFire proves the custom golangci-lint rules still catch what they exist
// for: `config verify` accepts rules that never match anything. It lints a fixture
// under testdata/, which ./... skips. CI sets WOPR_LINT_SELFTEST=1; locally the first
// run builds golangci-lint from tools/lint/go.mod, which takes a minute.
func TestLintRulesFire(t *testing.T) {
	if os.Getenv("WOPR_LINT_SELFTEST") == "" {
		t.Skip("set WOPR_LINT_SELFTEST=1 to run")
	}
	root := moduleRoot(t)
	trackSources(t, root)
	cmd := exec.Command("go", "tool", "-modfile=tools/lint/go.mod", "golangci-lint", "run",
		"--output.json.path=stdout", "--output.text.path=stderr", "--issues-exit-code=0",
		"./internal/archtest/testdata/lintfixture/")
	cmd.Dir = root
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("golangci-lint: %v\n%s", err, stderr.String())
	}
	line, _, _ := bytes.Cut(out, []byte("\n"))
	var report struct {
		Issues []struct {
			FromLinter string
			Text       string
		}
	}
	if err := json.Unmarshal(line, &report); err != nil {
		t.Fatalf("parse report: %v\n%s", err, out)
	}
	want := map[string]string{ // linter + text fragment that must appear
		"depguard":            "charm.land/bubbletea/v2",
		"forbidigo:tea.Tick":  "tea.Tick",
		"forbidigo:rand.IntN": "rand.IntN",
		"forbidigo:sleep":     "time.Sleep",
		"forbidigo:randv1":    "rand.Intn",
	}
	for name, fragment := range want {
		linter, _, _ := strings.Cut(name, ":")
		found := false
		for _, is := range report.Issues {
			if is.FromLinter == linter && strings.Contains(is.Text, fragment) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("rule %s did not fire on the fixture; the lint config no longer enforces it", name)
		}
	}
}

// Tools that run both as a prek hook and as a Go tool are pinned twice; the pins must agree.
func TestToolPinsAgree(t *testing.T) {
	root := moduleRoot(t)
	prek, err := os.ReadFile(filepath.Join(root, "prek.toml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range []struct{ hookRepo, modfile, module string }{
		{"https://github.com/golangci/golangci-lint", "tools/lint/go.mod", "github.com/golangci/golangci-lint/v2"},
		{"https://github.com/gitleaks/gitleaks", "tools/go.mod", "github.com/zricethezav/gitleaks/v8"},
	} {
		re := regexp.MustCompile(`repo = "` + regexp.QuoteMeta(tool.hookRepo) + `"\s*\nrev = "[0-9a-f]{40}"\s*# frozen: (v[0-9.]+)`)
		m := re.FindSubmatch(prek)
		if m == nil {
			t.Errorf("prek.toml: no %s rev with a '# frozen: vX.Y.Z' comment", tool.hookRepo)
			continue
		}
		if got := requiredVersion(t, filepath.Join(root, tool.modfile), tool.module); got != string(m[1]) {
			t.Errorf("prek.toml pins %s %s but %s pins %s", tool.module, m[1], tool.modfile, got)
		}
	}
}

// requiredVersion returns the version of module required by the go.mod at path.
func requiredVersion(t *testing.T, path, module string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		fields := strings.Fields(strings.TrimPrefix(strings.TrimSpace(sc.Text()), "require "))
		if len(fields) >= 2 && fields[0] == module {
			return fields[1]
		}
	}
	t.Fatalf("%s does not require %s", path, module)
	return ""
}
