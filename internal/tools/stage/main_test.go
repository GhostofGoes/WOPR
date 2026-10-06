package main

import (
	"archive/zip"
	"os"
	"path/filepath"
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
