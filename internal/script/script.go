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

// Provenance values. A third-party tag has the form "third-party:<repo>@<commit>:<path>".
const (
	Film          Prov = "film"          // confirmed against the film (M5 viewing pass)
	Reconstructed Prov = "reconstructed" // from transcripts or subtitles; not yet confirmed
	ABS0          Prov = "third-party:abs0/wargames@010ed92:wargames.sh"
	Original      Prov = "original" // written for this project
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
// in capitals, as WOPR's screen is.
func Validate(blocks []Ls, notice string) []error {
	var errs []error
	for _, block := range blocks {
		for _, l := range block {
			switch {
			case l.Prov == Film, l.Prov == Reconstructed, l.Prov == Original:
			case strings.HasPrefix(string(l.Prov), "third-party:"):
				repo, _, _ := strings.Cut(strings.TrimPrefix(string(l.Prov), "third-party:"), "@")
				if repo == "" || !strings.Contains(notice, repo) {
					errs = append(errs, fmt.Errorf("%q is credited to %q, which NOTICE.md does not mention", l.Text, repo))
				}
			default:
				errs = append(errs, fmt.Errorf("%q has no valid provenance (%q)", l.Text, l.Prov))
			}
			if l.Text != strings.ToUpper(l.Text) {
				errs = append(errs, fmt.Errorf("screen text is in capitals: %q", l.Text))
			}
		}
	}
	return errs
}
