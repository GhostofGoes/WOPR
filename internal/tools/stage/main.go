// Command stage turns GoReleaser output into the layout the smoke job runs:
//
//	stage/<goos>_<goarch>/wopr[.exe]       the binary from dist/
//	stage/<goos>_<goarch>/e2e.test[.exe]   internal/e2e, cross-compiled for that target
//
//	go run ./internal/tools/stage [-archives] [-assets dir] [-dist dist] [-out stage]
//
// With -archives it also checks that every release archive contains the files the
// licences require (LICENSE, NOTICE.md, THIRD_PARTY_NOTICES.txt) and README.md.
//
// With -assets it also collects every file a release publishes into one flat directory:
// the archives, the bare binaries under their release names, those four files and
// checksums.txt. Each one must match its line in checksums.txt, which the release attests.
package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
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
	Extra  struct {
		Format string `json:"Format"`
	} `json:"extra"`
}

// bare reports whether a is a binary-format archive: a built binary listed again under its
// release name. Its Path is the built binary's.
func (a artifact) bare() bool { return a.Type == "Binary" && a.Extra.Format == "binary" }

// requiredInArchives are the files every release archive must contain.
var requiredInArchives = []string{"LICENSE", "NOTICE.md", "README.md", "THIRD_PARTY_NOTICES.txt"}

func main() {
	dist := flag.String("dist", "dist", "GoReleaser output directory")
	out := flag.String("out", "stage", "directory to stage into")
	checkArchives := flag.Bool("archives", false, "also verify the contents of release archives")
	assets := flag.String("assets", "", "also collect every release file into this directory")
	flag.Parse()
	if err := run(*dist, *out, *checkArchives, *assets); err != nil {
		fmt.Fprintln(os.Stderr, "stage:", err)
		os.Exit(1)
	}
}

func run(dist, out string, checkArchives bool, assets string) error {
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
		switch {
		case a.bare(): // the same binary again; assemble copies it
		case a.Type == "Binary":
			if err := stageTarget(a, out); err != nil {
				return fmt.Errorf("%s/%s: %w", a.Goos, a.Goarch, err)
			}
			staged++
		case a.Type == "Archive" && checkArchives:
			if err := checkArchive(a.Path); err != nil {
				return fmt.Errorf("%s: %w", a.Name, err)
			}
			fmt.Println("archive ok:", a.Name)
		}
	}
	if staged == 0 {
		return errors.New("no binaries in artifacts.json")
	}
	if assets != "" {
		if err := assemble(arts, assets, "."); err != nil {
			return fmt.Errorf("assets: %w", err)
		}
	}
	return nil
}

// assemble copies every file a release publishes into dir, which must be new or hold only
// files from an earlier run: the archives, the bare binaries, checksums.txt, and the four
// documents from root. Then it checks them all against checksums.txt.
func assemble(arts []artifact, dir, root string) error {
	if err := emptyFlatDir(dir); err != nil {
		return err
	}
	built, bare := 0, 0
	for _, a := range arts {
		var err error
		switch {
		case a.bare():
			bare++
			err = copyFile(a.Path, filepath.Join(dir, a.Name), 0o755)
		case a.Type == "Binary":
			built++
		case a.Type == "Archive", a.Type == "Checksum":
			err = copyFile(a.Path, filepath.Join(dir, a.Name), 0o644)
		}
		if err != nil {
			return err
		}
	}
	if bare != built {
		return fmt.Errorf("%d bare binaries for %d builds; is the binaries archive in .goreleaser.yaml?", bare, built)
	}
	for _, name := range requiredInArchives {
		if err := copyFile(filepath.Join(root, name), filepath.Join(dir, name), 0o644); err != nil {
			return err
		}
	}
	return verifyChecksums(dir)
}

// emptyFlatDir creates dir, or empties it if it holds only regular files, so a mistyped
// path such as "." is refused rather than wiped.
func emptyFlatDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return os.MkdirAll(dir, 0o755)
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.Type().IsRegular() {
			return fmt.Errorf("%s holds %s, which is not a release file; refusing to empty it", dir, e.Name())
		}
	}
	for _, e := range entries {
		if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// verifyChecksums checks that dir holds exactly the files checksums.txt lists, with those
// SHA-256 sums.
func verifyChecksums(dir string) error {
	data, err := os.ReadFile(filepath.Join(dir, "checksums.txt"))
	if err != nil {
		return err
	}
	want := map[string]string{}
	for line := range strings.Lines(string(data)) {
		f := strings.Fields(line)
		if len(f) != 2 {
			return fmt.Errorf("checksums.txt: malformed line %q", line)
		}
		want[f[1]] = f[0]
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if name == "checksums.txt" {
			continue
		}
		sum, err := sha256File(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		switch w, ok := want[name]; {
		case !ok:
			return fmt.Errorf("%s is not in checksums.txt", name)
		case w != sum:
			return fmt.Errorf("%s: sha256 %s, but checksums.txt says %s", name, sum, w)
		}
		delete(want, name)
	}
	if len(want) > 0 {
		missing := slices.Sorted(maps.Keys(want))
		return fmt.Errorf("checksums.txt lists files that are missing: %s", strings.Join(missing, ", "))
	}
	fmt.Printf("assets ok: %d files in %s match checksums.txt\n", len(entries)-1, dir)
	return nil
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
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
	if err := copyFile(a.Path, filepath.Join(dir, "wopr"+ext), 0o755); err != nil {
		return err
	}
	// -trimpath, like GoReleaser's builds, so the packages they share come from the build cache.
	cmd := exec.Command("go", "test", "-c", "-trimpath", "-tags", "e2e", "-o", filepath.Join(dir, "e2e.test"+ext), "./internal/e2e")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+a.Goos, "GOARCH="+a.Goarch)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("building e2e test: %w", err)
	}
	fmt.Println("staged:", dir)
	return nil
}

func copyFile(from, to string, mode os.FileMode) error {
	src, err := os.Open(from)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()
	dst, err := os.OpenFile(to, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
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
