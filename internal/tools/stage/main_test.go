package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCheckArchive(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write := func(name string, files ...string) string {
		p := filepath.Join(dir, name)
		f, err := os.Create(p)
		if err != nil {
			t.Fatal(err)
		}
		zw := zip.NewWriter(f)
		for _, n := range files {
			w, err := zw.Create(n)
			if err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(n))
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
		return p
	}
	good := write("good.zip", "wopr.exe", "LICENSE", "NOTICE.md", "README.md", "THIRD_PARTY_NOTICES.txt")
	if err := checkArchive(good); err != nil {
		t.Errorf("complete archive rejected: %v", err)
	}
	bad := write("bad.zip", "wopr.exe", "LICENSE", "README.md")
	if err := checkArchive(bad); err == nil || !strings.Contains(err.Error(), "NOTICE.md") {
		t.Errorf("archive without notices accepted: %v", err)
	}
}

func TestAssemble(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(root, name)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	sum := func(body string) string {
		h := sha256.Sum256([]byte(body))
		return hex.EncodeToString(h[:])
	}
	var lines strings.Builder
	for _, name := range requiredInArchives {
		write(name, name)
		lines.WriteString(sum(name) + "  " + name + "\n")
	}
	bin := write("wopr", "binary")
	arc := write("wopr_1.0.0_linux_amd64.tar.gz", "archive")
	lines.WriteString(sum("binary") + "  wopr_1.0.0_linux_amd64\n")
	lines.WriteString(sum("archive") + "  wopr_1.0.0_linux_amd64.tar.gz\n")
	sums := write("checksums.txt", lines.String())
	bare := artifact{Name: "wopr_1.0.0_linux_amd64", Path: bin, Type: "Binary"}
	bare.Extra.Format = "binary"
	arts := []artifact{
		{Name: "wopr", Path: bin, Type: "Binary"},
		bare,
		{Name: "wopr_1.0.0_linux_amd64.tar.gz", Path: arc, Type: "Archive"},
		{Name: "checksums.txt", Path: sums, Type: "Checksum"},
	}

	dir := filepath.Join(root, "release")
	if err := assemble(arts, dir, root); err != nil {
		t.Fatalf("good release rejected: %v", err)
	}
	fi, err := os.Stat(filepath.Join(dir, "wopr_1.0.0_linux_amd64"))
	switch {
	case err != nil:
		t.Errorf("bare binary missing: %v", err)
	case runtime.GOOS != "windows" && fi.Mode().Perm()&0o100 == 0: // Windows has no executable bit
		t.Errorf("bare binary not executable: %v", fi.Mode())
	}
	if err := assemble(arts, dir, root); err != nil {
		t.Errorf("a second run must replace the first: %v", err)
	}

	// A file that changed after GoReleaser summed it must not be published.
	write("NOTICE.md", "edited")
	if err := assemble(arts, dir, root); err == nil || !strings.Contains(err.Error(), "NOTICE.md") {
		t.Errorf("edited file accepted: %v", err)
	}
	write("NOTICE.md", "NOTICE.md")

	if err := assemble(arts[1:], dir, root); err == nil || !strings.Contains(err.Error(), "bare binaries") {
		t.Errorf("a bare binary without a build accepted: %v", err)
	}
	if err := assemble(append(arts[:1:1], arts[2:]...), dir, root); err == nil {
		t.Error("a release without its bare binary accepted")
	}

	// A directory that holds more than release files is never emptied.
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := assemble(arts, dir, root); err == nil || !strings.Contains(err.Error(), "refusing") {
		t.Errorf("non-flat directory emptied: %v", err)
	}
}
