package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/games"
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
		b.WriteString("\n* not playable yet; WOPR will tell you so.\n")
	}
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

// RenderHelp writes the usage text for --help.
func RenderHelp(w io.Writer) error {
	var b strings.Builder
	b.WriteString("wopr: the WOPR computer from WarGames (1983), in your terminal.\n\n")
	b.WriteString("Usage:\n  wopr [flags] [game]\n  wopr --movie [scene]\n\nFlags:\n")
	for _, f := range flagPairs {
		names := "-" + f.short + ", --" + f.long
		if f.arg != "" {
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

Examples:
  wopr                 dial in and log on
  wopr chess -i        play chess with no typewriter pacing
  wopr --games         list the games
`)
	_, err := io.WriteString(w, b.String())
	return err
}
