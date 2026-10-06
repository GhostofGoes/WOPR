package archtest

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const module = "github.com/GhostofGoes/WOPR"

// rule says what a package (or, with a trailing "/...", a subtree) may import. Entries in
// allow are module-relative ("internal/proto"), external module prefixes ("charm.land/"),
// or "." for the module root. The standard library is allowed except where std bans it.
type rule struct {
	pkg   string
	allow []string
}

// testOnly may be imported by any package's tests in addition to its own rule.
var testOnly = []string{
	"internal/golden", "internal/games/catalog", "internal/games/gamestest", "internal/games/testkit",
}

// rules is the import DAG. Order matters: the first matching rule applies, so specific
// packages come before the subtrees that contain them.
var rules = []rule{
	{".", nil},
	{"cmd/wopr", []string{".", "internal/cli", "internal/ui", "internal/version", "internal/games/catalog", "internal/llm"}},
	{"internal/version", nil},
	{"internal/cli", []string{"internal/games", "internal/theme", "internal/movie", "internal/version"}},
	{"internal/proto", nil},
	{"internal/proto/host", []string{"internal/proto"}},
	{"internal/prompt", nil},
	{"internal/theme", []string{"internal/proto", "charm.land/lipgloss/v2", "github.com/charmbracelet/colorprofile", "github.com/charmbracelet/x/ansi"}},
	{"internal/ui/...", []string{
		"internal/ui/...", "internal/proto", "internal/proto/host", "internal/prompt", "internal/wopr", "internal/games",
		"internal/theme", "internal/assets", "internal/movie",
		"charm.land/", "github.com/charmbracelet/x/ansi", "github.com/charmbracelet/x/term", "github.com/charmbracelet/colorprofile",
		"github.com/rivo/uniseg",
	}},
	{"internal/wopr", []string{"internal/proto", "internal/prompt", "internal/games"}},
	{"internal/games", []string{"internal/proto", "internal/prompt"}},
	{"internal/games/catalog", []string{"internal/games", "internal/games/...", "internal/proto"}},
	{"internal/games/gamestest", []string{"internal/games", "internal/proto"}},
	{"internal/games/testkit", []string{"internal/games", "internal/proto", "internal/proto/host", "internal/golden"}},
	{"internal/games/ending", []string{"internal/proto", "internal/prompt", "internal/assets"}},
	{"internal/games/ai", []string{"internal/proto", "internal/games"}},
	{"internal/games/cards", []string{"internal/proto", "internal/games", "internal/prompt"}},
	{"internal/games/board", []string{"internal/proto", "internal/games"}},
	{"internal/games/...", []string{
		"internal/proto", "internal/prompt", "internal/games", "internal/games/ai", "internal/games/cards",
		"internal/games/board", "internal/games/ending", "internal/assets", "internal/sim", "github.com/corentings/chess/v2",
	}},
	{"internal/sim", []string{"internal/proto"}},
	{"internal/assets", nil},
	{"internal/movie/...", []string{"internal/movie/...", "internal/proto", "internal/prompt", "internal/assets", "internal/games/gtw", "internal/games/ending"}},
	{"internal/golden", nil},
	{"internal/archtest", nil},
	{"internal/e2e", []string{"github.com/charmbracelet/x/xpty"}},
	{"internal/tools/...", nil},
	{"internal/llm", []string{"internal/proto", "internal/wopr"}},
}

// stdBans fences standard-library packages to the places that need them.
var stdBans = map[string][]string{ // std package -> packages allowed to import it
	"net/http": {"internal/llm"},
	"os/exec":  {"internal/tools/...", "internal/archtest", "internal/e2e"},
	"unsafe":   {},
}

type listed struct {
	ImportPath   string
	Standard     bool
	Imports      []string
	TestImports  []string
	XTestImports []string
}

func TestImportDAG(t *testing.T) {
	root := moduleRoot(t)
	trackSources(t, root)
	cmd := exec.Command("go", "list", "-json=ImportPath,Standard,Imports,TestImports,XTestImports", "./...")
	cmd.Dir = root
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v\n%s", err, stderr.String())
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var p listed
		if err := dec.Decode(&p); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		rel := relative(p.ImportPath)
		r, ok := match(rel)
		if !ok {
			t.Errorf("%s matches no rule; add it to the DAG in internal/archtest and docs/PLAN.md", rel)
			continue
		}
		check := func(imports []string, test bool) {
			for _, imp := range imports {
				if why := allowed(rel, r, imp, test); why != "" {
					t.Errorf("%s imports %s: %s", rel, imp, why)
				}
			}
		}
		check(p.Imports, false)
		check(p.TestImports, true)
		check(p.XTestImports, true)
	}
}

func allowed(from string, r rule, imp string, test bool) string {
	if imp == "C" {
		return "cgo is not allowed (CGO_ENABLED=0 builds)"
	}
	if !strings.Contains(strings.SplitN(imp, "/", 2)[0], ".") { // standard library
		if who, banned := stdBans[imp]; banned && !matchesAny(from, who) {
			return "this standard-library package is fenced to " + strings.Join(who, ", ")
		}
		return ""
	}
	if imp == module || strings.HasPrefix(imp, module+"/") {
		target := relative(imp)
		if target == from || strings.HasPrefix(target, from+"/") && strings.HasSuffix(r.pkg, "/...") {
			return ""
		}
		if test && matchesAny(target, testOnly) {
			return ""
		}
		if matchesAny(target, r.allow) {
			return ""
		}
		return "not in this package's allowed imports"
	}
	for _, a := range r.allow {
		if strings.HasSuffix(a, "/") && strings.HasPrefix(imp, a) || imp == a || strings.HasPrefix(imp, a+"/") {
			return ""
		}
	}
	return "external module not allowed here"
}

func match(rel string) (rule, bool) {
	for _, r := range rules {
		if matches(rel, r.pkg) {
			return r, true
		}
	}
	return rule{}, false
}

func matchesAny(rel string, patterns []string) bool {
	for _, p := range patterns {
		if matches(rel, p) {
			return true
		}
	}
	return false
}

func matches(rel, pattern string) bool {
	if base, ok := strings.CutSuffix(pattern, "/..."); ok {
		return rel == base || strings.HasPrefix(rel, base+"/")
	}
	return rel == pattern
}

func relative(importPath string) string {
	if importPath == module {
		return "."
	}
	return strings.TrimPrefix(importPath, module+"/")
}

// trackSources stats every Go source and go.mod file. The go command records files a test
// opens or stats, so a change to any of them invalidates the cached result; without
// this, the result of the go list subprocess would be cached across source changes.
func trackSources(t *testing.T, root string) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path != root && (strings.HasPrefix(d.Name(), ".") || d.Name() == "testdata" || d.Name() == "dist") {
			return filepath.SkipDir
		}
		if !d.IsDir() && (strings.HasSuffix(path, ".go") || d.Name() == "go.mod") {
			_, err = os.Stat(path)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func TestRulesMatchThemselves(t *testing.T) {
	t.Parallel()
	for _, in := range []string{"internal/ui/console", "internal/games/chess", "internal/games/catalog", "cmd/wopr"} {
		if _, ok := match(in); !ok {
			t.Errorf("%s matches no rule", in)
		}
	}
	if r, _ := match("internal/games/catalog"); r.pkg != "internal/games/catalog" {
		t.Errorf("catalog matched %q; specific rules must come before subtrees", r.pkg)
	}
	if why := allowed("internal/games/chess", rule{pkg: "internal/games/..."}, "charm.land/bubbletea/v2", false); why == "" {
		t.Error("a game must not import Bubble Tea")
	}
	if why := allowed("internal/wopr", rule{pkg: "internal/wopr", allow: []string{"internal/proto"}}, "net/http", false); why == "" {
		t.Error("net/http must be fenced to internal/llm")
	}
}

// CI builds with the toolchain pinned in go.mod. actions/setup-go silently falls back to
// the go directive when GOTOOLCHAIN=local is set before it runs, so check (D-6).
func TestToolchainMatchesGoMod(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		t.Skip("only enforced in CI; locally a newer Go is fine")
	}
	data, err := os.ReadFile(filepath.Join(moduleRoot(t), "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		if v, ok := strings.CutPrefix(strings.TrimSpace(sc.Text()), "toolchain "); ok {
			if runtime.Version() != v {
				t.Fatalf("running %s, but go.mod pins %s; check the setup-go step", runtime.Version(), v)
			}
			return
		}
	}
	t.Fatal("go.mod has no toolchain line")
}
