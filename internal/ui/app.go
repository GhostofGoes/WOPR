// Package ui is the Bubble Tea application: the only package that imports Bubble Tea.
// It turns the protocol in internal/proto into a terminal user interface.
//
// M0 ships a minimal screen that proves the stack end to end: alternate screen, theme
// background, native cursor and exit handling. The console, host and persona arrive in M1.
package ui

import (
	"errors"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/term"

	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/theme"
)

// Options configure a session.
type Options struct {
	Theme        string
	Instant      bool
	Seed         uint64
	SeedSet      bool
	ReduceMotion bool
	Play         string // game slug to start directly
	Movie        bool
	Scene        string
	NoColor      bool // NO_COLOR set to any non-empty value (no-color.org)
}

// Outcome is how a session ended, for the caller's exit code.
type Outcome int

// Outcomes.
const (
	Finished    Outcome = iota // LOGOFF or SIGTERM
	Interrupted                // Ctrl+C or SIGINT
	Panicked                   // the terminal is restored and the stack is on stderr
	NoTerminal                 // no terminal could be opened
	Failed                     // any other error
)

// Run runs the TUI until the user leaves.
func Run(opts Options) (Outcome, error) {
	th, ok := theme.Get(opts.Theme)
	if !ok {
		th, _ = theme.Get(theme.Default)
	}
	var teaOpts []tea.ProgramOption
	if opts.NoColor { // colorprofile alone parses NO_COLOR with ParseBool and ignores NO_COLOR=yes
		teaOpts = append(teaOpts, tea.WithColorProfile(colorprofile.Ascii))
	}
	_, err := tea.NewProgram(newModel(th), teaOpts...).Run()
	return classify(err), err
}

// classify maps Bubble Tea's error. ErrInterrupted comes first: both it and
// ErrProgramPanic wrap ErrProgramKilled.
func classify(err error) Outcome {
	switch {
	case err == nil:
		return Finished
	case errors.Is(err, tea.ErrInterrupted):
		return Interrupted
	case errors.Is(err, tea.ErrProgramPanic):
		return Panicked
	case strings.Contains(err.Error(), "TTY"):
		return NoTerminal
	default:
		return Failed
	}
}

// TerminalProblem explains why the TUI cannot start here, or returns "". A redirected
// stdin is fine: Bubble Tea opens the controlling terminal itself.
func TerminalProblem(getenv func(string) string) string {
	switch {
	case getenv("TERM") == "dumb":
		return "TERM=dumb: this terminal cannot run the full-screen interface (try wopr --games)"
	case !term.IsTerminal(os.Stdout.Fd()):
		return "standard output is not a terminal"
	}
	return ""
}

type model struct {
	th      *theme.Theme
	profile colorprofile.Profile
	w, h    int
}

func newModel(th *theme.Theme) model {
	return model{th: th, profile: colorprofile.TrueColor}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case tea.ColorProfileMsg:
		m.profile = msg.Profile
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Interrupt
		case "ctrl+z":
			return m, tea.Suspend
		}
	}
	return m, nil
}

const prompt = "LOGON: "

func (m model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	if m.w > 0 && m.h > 0 {
		v.Cursor = tea.NewCursor(len(prompt), 0)
	}
	return v
}

// render paints the theme background into every cell, so the screen looks the same
// whatever the terminal's own background is.
func (m model) render() string {
	if m.w <= 0 || m.h <= 0 {
		return ""
	}
	text := m.th.Lip(proto.StyleText, 0, m.profile)
	lines := make([]string, m.h)
	for y := range lines {
		s := ""
		if y == 0 {
			s = prompt
		}
		lines[y] = text.Render(s + strings.Repeat(" ", max(m.w-len(s), 0)))
	}
	return strings.Join(lines, "\n")
}
