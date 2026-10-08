// Command pkgdocs writes the documents that the release archives and the .deb and .rpm packages
// carry. The packages carry them compressed, as `gzip -9n` would: maximum compression, and no file
// name or time stamp in the header, so the same input gives the same bytes on every host and the
// packages stay reproducible. Debian Policy asks for manual pages (§12.1), longer documents
// (§12.3) and release notes (§12.7) to be compressed that way; RPM packages carry their manual
// pages compressed too.
//
//	go run ./internal/tools/pkgdocs -version 0.3.0 -notes build/notes -out build/pkg
//
// GoReleaser runs it as a before hook (.goreleaser.yaml), with the version it is building, after
// internal/tools/relnotes has written the release notes into -notes. It writes into -out, which
// must be outside the tracked tree (build/ is in .gitignore):
//
//	wopr.6         docs/man/wopr.6, the manual page, naming -version (the Linux and macOS archives)
//	wopr.6.gz      the same page, compressed (both packages)
//	NEWS.gz        CHANGELOG.md from -notes: the release notes for players (the .deb)
//	README.md.gz   README.md (the .deb)
//	NOTICE.md.gz   NOTICE.md (the .deb)
//	changelog.yml  changelog.yml from -notes, which nfpms.changelog reads (both packages)
//
// The manual page's header names the version of the newest .changes/vX.Y.Z.md, which a release
// pull request batches. pkgdocs puts the version being built there instead, so every archive and
// package names its own version, even a snapshot's or a release's whose notes were not batched.
// changelog.yml is copied so that the packages' changelogs come from -notes, wherever it is:
// nfpms.changelog takes no template, so it names build/pkg/changelog.yml.
package main

import (
	"bytes"
	"compress/gzip"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// unixOS is the gzip header's operating system byte for Unix, which `gzip -n` writes.
const unixOS = 3

// manPage is the manual page in the tree, as internal/tools/manpage writes it.
const manPage = "docs/man/wopr.6"

// doc is one document: where it comes from, the name it gets in -out, and whether it is gzipped.
type doc struct {
	src, name string
	gz        bool
}

// docs lists what to write. A src that starts with "notes:" is read from the -notes directory;
// the manual page gets the version being built.
var docs = []doc{
	{manPage, "wopr.6", false},
	{manPage, "wopr.6.gz", true},
	{"notes:CHANGELOG.md", "NEWS.gz", true},
	{"README.md", "README.md.gz", true},
	{"NOTICE.md", "NOTICE.md.gz", true},
	{"notes:changelog.yml", "changelog.yml", false},
}

func main() {
	version := flag.String("version", "", "the version being built, as GoReleaser's {{ .Version }} gives it (1.2.3 or 1.2.4-snapshot.abc1234)")
	notes := flag.String("notes", "", "directory relnotes wrote the release notes into (CHANGELOG.md, changelog.yml)")
	out := flag.String("out", "", "directory to write the documents into")
	flag.Parse()
	if *version == "" || *notes == "" || *out == "" || flag.NArg() > 0 {
		fmt.Fprintln(os.Stderr, "pkgdocs: use -version VERSION -notes DIR -out DIR")
		os.Exit(2)
	}
	if err := run(".", *notes, *out, *version); err != nil {
		fmt.Fprintln(os.Stderr, "pkgdocs:", err)
		os.Exit(1)
	}
}

// run writes every document from root (the repository) and notes into out.
func run(root, notes, out, version string) error {
	if !versionRE.MatchString(version) {
		return fmt.Errorf("-version %q is not a version like 1.2.3 or 1.2.4-snapshot.abc1234", version)
	}
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
		if d.src == manPage {
			if data, err = stampVersion(data, version); err != nil {
				return fmt.Errorf("%s: %w", src, err)
			}
		}
		if d.gz {
			if data, err = compress(data); err != nil {
				return fmt.Errorf("%s: %w", src, err)
			}
		}
		if err := os.WriteFile(filepath.Join(out, d.name), data, 0o644); err != nil {
			return err
		}
		fmt.Printf("pkgdocs: %s -> %s\n", src, filepath.Join(out, d.name))
	}
	return nil
}

// versionRE is a version as GoReleaser's {{ .Version }} gives it: no "v".
var versionRE = regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`)

// titleLine is the manual page's title line, .TH WOPR 6 <date> "<source>" "<manual>", up to the
// source, which names the program and its version.
var titleLine = regexp.MustCompile(`(?m)^(\.TH \S+ \S+ \S+ )"[^"]*"`)

// stampVersion sets the source in the manual page's title line to "wopr <version>", escaped as
// internal/tools/manpage escapes text. The rest of the page is left as it is.
func stampVersion(page []byte, version string) ([]byte, error) {
	loc := titleLine.FindSubmatchIndex(page)
	if loc == nil {
		return nil, errors.New(`no title line like .TH WOPR 6 2006-01-02 "wopr 1.2.3"`)
	}
	source := `"wopr ` + strings.ReplaceAll(strings.ReplaceAll(version, `\`, `\e`), "-", `\-`) + `"`
	return slices.Concat(page[:loc[3]], []byte(source), page[loc[1]:]), nil
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
