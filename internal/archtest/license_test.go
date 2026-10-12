// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package archtest

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The lines every source file carries near its top: a copyright statement and an SPDX licence
// identifier (OpenSSF Best Practices, copyright_per_file and license_per_file).
const (
	copyrightLine = "SPDX-FileCopyrightText: " // then the year and the holder, as in LICENSE
	licenseLine   = "SPDX-License-Identifier: MIT"
	headerWindow  = 512 // bytes from the top of a file, after any shebang line
)

// sourcePatterns are the files git tracks that need the header. Vendored files keep the
// headers their projects publish.
var sourcePatterns = []string{
	"*.go", "*.sh", "*.ps1", "*.psm1", "*.iss", "*.js", "*.css", "site/layouts/*.html",
}

func TestSourceFilesHaveLicenseHeaders(t *testing.T) {
	root := moduleRoot(t)
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	cmd := exec.Command("git", append([]string{"ls-files", "--"}, sourcePatterns...)...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Skipf("not a git checkout: %v", err)
	}
	files := strings.Fields(string(out))
	if len(files) < 100 {
		t.Fatalf("git lists %d source files, want the whole repository", len(files))
	}
	for _, f := range files {
		if strings.Contains(f, "/vendor/") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f)))
		if err != nil {
			t.Fatal(err)
		}
		if bytes.HasPrefix(data, []byte("#!")) {
			if i := bytes.IndexByte(data, '\n'); i >= 0 {
				data = data[i+1:]
			}
		}
		head := string(data[:min(len(data), headerWindow)])
		for _, line := range []string{copyrightLine, licenseLine} {
			if !strings.Contains(head, line) {
				t.Errorf("%s: no %q near the top", f, line)
			}
		}
	}
}
