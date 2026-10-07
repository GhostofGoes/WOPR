// Command pkgdocs writes the documents that the .deb and .rpm packages carry compressed, as
// `gzip -9n` would: maximum compression, and no file name or time stamp in the header, so the
// same input gives the same bytes on every host and the packages stay reproducible. Debian
// Policy asks for manual pages (§12.1), longer documents (§12.3) and release notes (§12.7) to be
// compressed that way; RPM packages carry their manual pages compressed too.
//
//	go run ./internal/tools/pkgdocs -notes build/notes -out build/pkg
//
// GoReleaser runs it as a before hook (.goreleaser.yaml), after internal/tools/relnotes has
// written the release notes into -notes. It writes into -out, which must be outside the tracked
// tree (build/ is in .gitignore):
//
//	wopr.6.gz      docs/man/wopr.6, the manual page (both packages)
//	NEWS.gz        CHANGELOG.md from -notes: the release notes for players (the .deb)
//	README.md.gz   README.md (the .deb)
//	NOTICE.md.gz   NOTICE.md (the .deb)
package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// unixOS is the gzip header's operating system byte for Unix, which `gzip -n` writes.
const unixOS = 3

// doc is one compressed document: where it comes from and the name it gets in -out.
type doc struct {
	src, name string
}

// docs lists what to compress. A src that starts with "notes:" is read from the -notes directory.
var docs = []doc{
	{"docs/man/wopr.6", "wopr.6.gz"},
	{"notes:CHANGELOG.md", "NEWS.gz"},
	{"README.md", "README.md.gz"},
	{"NOTICE.md", "NOTICE.md.gz"},
}

func main() {
	notes := flag.String("notes", "", "directory relnotes wrote the release notes into (CHANGELOG.md)")
	out := flag.String("out", "", "directory to write the compressed documents into")
	flag.Parse()
	if *notes == "" || *out == "" || flag.NArg() > 0 {
		fmt.Fprintln(os.Stderr, "pkgdocs: use -notes DIR -out DIR")
		os.Exit(2)
	}
	if err := run(".", *notes, *out); err != nil {
		fmt.Fprintln(os.Stderr, "pkgdocs:", err)
		os.Exit(1)
	}
}

// run compresses every document from root (the repository) and notes into out.
func run(root, notes, out string) error {
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	for _, d := range docs {
		src := filepath.Join(root, filepath.FromSlash(d.src))
		if rest, ok := strings.CutPrefix(d.src, "notes:"); ok {
			src = filepath.Join(notes, rest)
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		gz, err := compress(data)
		if err != nil {
			return fmt.Errorf("%s: %w", src, err)
		}
		if err := os.WriteFile(filepath.Join(out, d.name), gz, 0o644); err != nil {
			return err
		}
		fmt.Printf("pkgdocs: %s -> %s\n", src, filepath.Join(out, d.name))
	}
	return nil
}

// compress gzips data as `gzip -9n` does: best compression, which sets the header's
// maximum-compression flag, and a header with no name and a zero time.
func compress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	w, err := gzip.NewWriterLevel(&b, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	w.OS = unixOS
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
