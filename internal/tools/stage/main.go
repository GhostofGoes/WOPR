// Command stage turns GoReleaser output into the layout the smoke job runs:
//
//	stage/<goos>_<goarch>/wopr[.exe]       the binary from dist/
//	stage/<goos>_<goarch>/e2e.test[.exe]   internal/e2e, cross-compiled for that target
//
//	go run ./internal/tools/stage [-archives] [-dist dist] [-out stage]
//
// With -archives it also checks that every release archive contains the files the
// licences require (LICENSE, NOTICE.md, THIRD_PARTY_NOTICES.txt) and README.md.
package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

type artifact struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Goos   string `json:"goos"`
	Goarch string `json:"goarch"`
	Type   string `json:"type"`
}

// requiredInArchives are the files every release archive must contain.
var requiredInArchives = []string{"LICENSE", "NOTICE.md", "README.md", "THIRD_PARTY_NOTICES.txt"}

func main() {
	dist := flag.String("dist", "dist", "GoReleaser output directory")
	out := flag.String("out", "stage", "directory to stage into")
	checkArchives := flag.Bool("archives", false, "also verify the contents of release archives")
	flag.Parse()
	if err := run(*dist, *out, *checkArchives); err != nil {
		fmt.Fprintln(os.Stderr, "stage:", err)
		os.Exit(1)
	}
}

func run(dist, out string, checkArchives bool) error {
	data, err := os.ReadFile(filepath.Join(dist, "artifacts.json"))
	if err != nil {
		return err
	}
	var arts []artifact
	if err := json.Unmarshal(data, &arts); err != nil {
		return err
	}
	staged := 0
	for _, a := range arts {
		switch a.Type {
		case "Binary":
			if err := stageTarget(a, out); err != nil {
				return fmt.Errorf("%s/%s: %w", a.Goos, a.Goarch, err)
			}
			staged++
		case "Archive":
			if checkArchives {
				if err := checkArchive(a.Path); err != nil {
					return fmt.Errorf("%s: %w", a.Name, err)
				}
				fmt.Println("archive ok:", a.Name)
			}
		}
	}
	if staged == 0 {
		return errors.New("no binaries in artifacts.json")
	}
	return nil
}

func stageTarget(a artifact, out string) error {
	dir := filepath.Join(out, a.Goos+"_"+a.Goarch)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	ext := ""
	if a.Goos == "windows" {
		ext = ".exe"
	}
	if err := copyFile(a.Path, filepath.Join(dir, "wopr"+ext)); err != nil {
		return err
	}
	cmd := exec.Command("go", "test", "-c", "-tags", "e2e", "-o", filepath.Join(dir, "e2e.test"+ext), "./internal/e2e")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+a.Goos, "GOARCH="+a.Goarch)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("building e2e test: %w", err)
	}
	fmt.Println("staged:", dir)
	return nil
}

func copyFile(from, to string) error {
	src, err := os.Open(from)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()
	dst, err := os.OpenFile(to, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		return err
	}
	return dst.Close()
}

func checkArchive(path string) error {
	var names []string
	switch {
	case strings.HasSuffix(path, ".zip"):
		r, err := zip.OpenReader(path)
		if err != nil {
			return err
		}
		defer func() { _ = r.Close() }()
		for _, f := range r.File {
			names = append(names, f.Name)
		}
	case strings.HasSuffix(path, ".tar.gz"):
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		gz, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		tr := tar.NewReader(gz)
		for {
			h, err := tr.Next()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return err
			}
			names = append(names, h.Name)
		}
	default:
		return fmt.Errorf("unknown archive type")
	}
	var missing []string
	for _, req := range requiredInArchives {
		if !slices.Contains(names, req) {
			missing = append(missing, req)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing %s (has %v)", strings.Join(missing, ", "), names)
	}
	return nil
}
