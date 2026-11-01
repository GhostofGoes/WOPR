// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/gtw"
	"github.com/GhostofGoes/WOPR/internal/movie"
)

// RenderGames writes the plain-text game list for --games: the film's LIST GAMES order
// with a blank line before the last entry, as on screen, then the unlisted games.
func RenderGames(w io.Writer, reg *games.Registry) error {
	var b strings.Builder
	listed := reg.Listed()
	anyPlanned := false
	row := func(num string, in games.Info) {
		mark := " "
		if in.Status == games.Planned {
			mark = "*"
			anyPlanned = true
		}
		fmt.Fprintf(&b, "%3s %s %-42s %s\n", num, mark, in.Name, Handle(in))
	}
	for i, e := range listed {
		if i == len(listed)-1 && i > 0 {
			b.WriteString("\n")
		}
		row(fmt.Sprintf("%d.", e.Info.Number), e.Info)
	}
	first := true
	for _, e := range reg.All() {
		if e.Info.Listed {
			continue
		}
		if first {
			b.WriteString("\nALSO AVAILABLE:\n")
			first = false
		}
		row("", e.Info)
	}
	if anyPlanned {
		b.WriteString("\n* not playable yet. WOPR will tell you so.\n")
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// RenderScenes writes the plain-text scene list for --scenes: number, slug and what happens,
// in the film's order.
func RenderScenes(w io.Writer) error {
	_, err := io.WriteString(w, sceneList()+"\nPlay from one with wopr -m <number or name>. Add -o to play just that one.\n"+
		"wopr -m alone opens a menu.\n")
	return err
}

func sceneList() string {
	var b strings.Builder
	for _, s := range movie.Scenes() {
		fmt.Fprintf(&b, "%2d. %-16s %s\n", s.Number, s.Slug, s.Blurb)
	}
	return b.String()
}

// RenderScenarios writes the plain-text list of GTW's scenarios for --scenario alone: the
// closing montage's strategies, numbered as GTW's scenario menu numbers them.
func RenderScenarios(w io.Writer) error {
	var b strings.Builder
	for i, name := range gtw.Scenarios() {
		fmt.Fprintf(&b, "%3d. %s\n", i+1, name)
	}
	b.WriteString("\nWatch one play itself with wopr --scenario <number or name>.\n" +
		"In GTW, 3 at the side choice opens the same list.\n")
	_, err := io.WriteString(w, b.String())
	return err
}

// Handle is the shortest name a game answers to: its slug or its shortest alias.
func Handle(in games.Info) string {
	h := in.Slug
	for _, a := range in.Aliases {
		if len(a) < len(h) && !strings.Contains(a, " ") {
			h = a
		}
	}
	return h
}

// DocsURL is the documentation site: installing, every option, and how to play each game.
const DocsURL = "https://ghostofgoes.github.io/WOPR/"

// RenderHelp writes the usage text for --help.
func RenderHelp(w io.Writer) error {
	var b strings.Builder
	b.WriteString("wopr: the WOPR computer from WarGames (1983), in your terminal.\n\n")
	b.WriteString("Usage:\n  wopr [flags] [game]\n  wopr --movie [--only] [scene]\n  wopr --scenario [scenario]\n\nFlags:\n")
	for _, f := range flagPairs {
		names := "-" + f.short + ", --" + f.long
		switch {
		case strings.HasPrefix(f.arg, "="):
			names += "[" + f.arg + "]"
		case f.arg != "":
			names += " <" + f.arg + ">"
		}
		help := f.help
		if f.env != "" {
			help += " (env " + f.env + ")"
		}
		fmt.Fprintf(&b, "  %-28s %s\n", names, help)
	}
	b.WriteString(`
At the LOGON: prompt, the film's backdoor still works: Joshua.
Type LOGOFF to leave. Ctrl+C quits at any time.

With -r nothing blinks, the front panel's lights stand still, and the ending
keeps a steady pace. The typing and the animations still play. -i skips them.

Movie mode plays the film's scenes from the one named to the end, then exits.
Without a scene, it opens a menu of scenes. With -o, each scene plays alone.
Then wopr exits, or it returns to the menu if you came from it. Space pauses.
n or Right skips to the next scene, p or Left goes back one, and Esc opens
the menu. The climax's games play to the end: those keys do nothing there.
With -i the text appears at once, but the scenes still pause.

With -c alone, wopr lists the scenarios of the film's closing montage. With a
number or a name, GTW plays that scenario by itself, from a short briefing to
the kill ratios. Enter then opens the scenario menu.

With -a, GTW shows only the film's own text. You choose a side and name the
targets. The war and the kill ratios then run by themselves, up to the climax.

Examples:
  wopr                 dial in and log on
  wopr chess -i        play chess with no typewriter pacing
  wopr gtw -w=20       play GTW with 20 seconds to give each strike order
  wopr gtw -n          play GTW war after war, with no launch code at the end
  wopr gtw -a          play GTW as the film shows it: a side and targets only
  wopr --games         list the games
  wopr -m 2            replay the film from scene 2 (wopr --scenes lists them)
  wopr -m joshua -o    replay just the joshua scene
  wopr -c 2            watch GTW play USSR FIRST STRIKE

Documentation, with how to play each game: ` + DocsURL + "\n")
	_, err := io.WriteString(w, b.String())
	return err
}
