// Package script holds screen text with its provenance (docs/PLAN.md §2.1): every line WOPR
// or a game prints carries a tag saying where it comes from. The persona, the games, the
// assets and the movie scenes all use it, and each package's tests run Validate over its
// text.
package script

import (
	"fmt"
	"strings"
)

// Prov records where a line of script text comes from.
type Prov string

// Provenance values. A third-party tag names its source: "third-party:<repo>@<commit>:<path>"
// for text from a repository, or "third-party:<https URL>" for art from a web page.
const (
	Film          Prov = "film"          // confirmed against the film (M5 viewing pass)
	Reconstructed Prov = "reconstructed" // from transcripts or subtitles; not yet confirmed
	ABS0          Prov = "third-party:abs0/wargames@010ed92:wargames.sh"
	// MatthewThomasMap is the big board's world map: "Map (C) 1998 Matthew Thomas. Freely
	// usable if this line is included."
	MatthewThomasMap Prov = "third-party:https://asciiart.website/art/3719"
	Original         Prov = "original" // written for this project
)

// L is one line of script text with its provenance.
type L struct {
	Text string
	Prov Prov
}

// Ls is a block of lines.
type Ls []L

// Texts returns the text of each line.
func (ls Ls) Texts() []string {
	out := make([]string, len(ls))
	for i, l := range ls {
		out[i] = l.Text
	}
	return out
}

// Recon tags lines as reconstructed from transcripts.
func Recon(texts ...string) Ls { return Tag(Reconstructed, texts...) }

// Orig tags lines as written for this project.
func Orig(texts ...string) Ls { return Tag(Original, texts...) }

// Tag tags lines with p.
func Tag(p Prov, texts ...string) Ls {
	out := make(Ls, len(texts))
	for i, t := range texts {
		out[i] = L{Text: t, Prov: p}
	}
	return out
}

// Validate checks every line: its tag must be one of the provenance values, a third-party
// tag must name a source that notice (the text of NOTICE.md) credits, and the text must be
// in capitals, as WOPR's screen is. Third-party art from a web page is the exception to the
// capitals: it is reproduced exactly as its author drew it, signature included.
func Validate(blocks []Ls, notice string) []error {
	var errs []error
	for _, block := range blocks {
		for _, l := range block {
			switch {
			case l.Prov == Film, l.Prov == Reconstructed, l.Prov == Original:
			case strings.HasPrefix(string(l.Prov), "third-party:"):
				if err := checkThirdParty(l, notice); err != nil {
					errs = append(errs, err)
				}
			default:
				errs = append(errs, fmt.Errorf("%q has no valid provenance (%q)", l.Text, l.Prov))
			}
			if l.Text != strings.ToUpper(l.Text) && !isWebArt(l.Prov) {
				errs = append(errs, fmt.Errorf("screen text is in capitals: %q", l.Text))
			}
		}
	}
	return errs
}

// isWebArt reports a third-party tag that names a web page: art kept as its author drew it.
func isWebArt(p Prov) bool { return strings.HasPrefix(string(p), "third-party:https://") }

// checkThirdParty checks a third-party tag. A web page's tag needs NOTICE.md to credit that
// exact URL. A repository's needs every part present, "<repo>@<commit>:<path>", and
// NOTICE.md crediting the repository by name (as `owner/repo` or its GitHub URL), not merely
// containing the text somewhere.
func checkThirdParty(l L, notice string) error {
	rest := strings.TrimPrefix(string(l.Prov), "third-party:")
	if isWebArt(l.Prov) {
		if !strings.Contains(notice, "<"+rest+">") && !strings.Contains(notice, "("+rest+")") {
			return fmt.Errorf("%q is credited to %q, which NOTICE.md does not link", l.Text, rest)
		}
		return nil
	}
	repo, at, ok1 := strings.Cut(rest, "@")
	commit, path, ok2 := strings.Cut(at, ":")
	owner, name, ok3 := strings.Cut(repo, "/")
	if !ok1 || !ok2 || !ok3 || owner == "" || name == "" || commit == "" || path == "" {
		return fmt.Errorf("%q has a malformed third-party tag %q (want third-party:<owner/repo>@<commit>:<path>)", l.Text, l.Prov)
	}
	if !strings.Contains(notice, "`"+repo+"`") && !strings.Contains(notice, "github.com/"+repo) {
		return fmt.Errorf("%q is credited to %q, which NOTICE.md does not mention", l.Text, repo)
	}
	return nil
}
