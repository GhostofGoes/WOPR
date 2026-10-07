// Package theme maps the semantic styles of proto.Canvas and the console to colours.
//
// Every style is defined three ways: a truecolor hex value, an explicit ANSI 16-colour
// index (automatic downsampling collapses meaningful pairs, e.g. two reds into one), and
// attributes for terminals without colour (NO_COLOR, TERM=xterm-mono). Meaning never
// depends on colour alone: games also vary glyphs and the Reverse attribute.
package theme

import (
	"slices"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"github.com/GhostofGoes/WOPR/internal/proto"
)

// Color is one colour in its truecolor and 16-colour forms.
type Color struct {
	Hex  string // "#RRGGBB"
	ANSI int    // 0..15
}

// Spec is how one semantic style looks.
type Spec struct {
	FG        Color
	BG        *Color     // nil: the theme background
	Attr      proto.Attr // always applied
	NoColAttr proto.Attr // applied only when the terminal shows no colour
}

// Theme is a named palette.
type Theme struct {
	Name       string
	Background Color
	Styles     map[proto.Style]Spec
}

// Default is the film-accurate white-phosphor theme.
const Default = "imsai"

var themes = map[string]*Theme{
	"imsai": mono("imsai", Color{"#DCE6F0", 7}, Color{"#FFFFFF", 15}, Color{"#7A8694", 8}, Color{"#9EC5FF", 14}, Color{"#000000", 0}),
	"green": mono("green", Color{"#33FF33", 10}, Color{"#B6FFB6", 15}, Color{"#1A8C1A", 2}, Color{"#B6FFB6", 15}, Color{"#000000", 0}),
	"amber": mono("amber", Color{"#FFB000", 11}, Color{"#FFD27A", 15}, Color{"#8A5E00", 3}, Color{"#FFD27A", 15}, Color{"#000000", 0}),
	"norad": norad(),
}

// Names lists the themes in a stable order, default first.
func Names() []string {
	return []string{"imsai", "green", "amber", "norad"}
}

// Get returns the named theme.
func Get(name string) (*Theme, bool) {
	t, ok := themes[name]
	return t, ok
}

// mono builds a single-phosphor theme: meaning is carried by intensity, attributes and
// glyphs, as on the film's monochrome terminal.
func mono(name string, text, bright, dim, accent, bg Color) *Theme {
	t := &Theme{Name: name, Background: bg, Styles: map[proto.Style]Spec{}}
	plain := Spec{FG: text}
	strong := Spec{FG: bright, Attr: proto.AttrBold}
	faint := Spec{FG: dim}
	set := func(s Spec, styles ...proto.Style) {
		for _, st := range styles {
			t.Styles[st] = s
		}
	}
	set(plain, proto.StyleText, proto.StyleOutgoing, proto.StyleSuitBlack, proto.StyleDefcon5, proto.StyleDefcon4)
	set(strong, proto.StyleBright, proto.StyleLabel, proto.StyleIncoming, proto.StyleAlert, proto.StyleSuitRed,
		proto.StyleDefcon3)
	t.Styles[proto.StyleDefcon2] = Spec{FG: bright, Attr: proto.AttrBold | proto.AttrUnderline}
	set(faint, proto.StyleDim, proto.StyleLand)
	t.Styles[proto.StyleAccent] = Spec{FG: accent, Attr: proto.AttrBold}
	t.Styles[proto.StyleTarget] = Spec{FG: bright, Attr: proto.AttrBold | proto.AttrUnderline}
	t.Styles[proto.StyleDefcon1] = Spec{FG: bright, Attr: proto.AttrBold | proto.AttrReverse}
	t.Styles[proto.StyleSelected] = Spec{FG: bright, Attr: proto.AttrReverse}
	return t
}

// norad is the war-room big-board theme.
func norad() *Theme {
	var (
		text   = Color{"#5AC8FA", 14}
		bright = Color{"#FFFFFF", 15}
		dim    = Color{"#4A86D6", 6} // index 4 (blue) is under 2.3:1 on black in xterm and VGA palettes
		yellow = Color{"#FFD60A", 11}
		red    = Color{"#FF3B30", 9}
		blue   = Color{"#3A7BFF", 12}
		green  = Color{"#34C759", 10}
		redBG  = Color{"#C0001A", 1}
	)
	return &Theme{
		Name:       "norad",
		Background: Color{"#02060F", 0},
		Styles: map[proto.Style]Spec{
			proto.StyleText:      {FG: text},
			proto.StyleBright:    {FG: bright, Attr: proto.AttrBold},
			proto.StyleDim:       {FG: dim},
			proto.StyleAccent:    {FG: yellow, Attr: proto.AttrBold},
			proto.StyleLand:      {FG: dim},
			proto.StyleLabel:     {FG: bright},
			proto.StyleIncoming:  {FG: red, NoColAttr: proto.AttrBold},
			proto.StyleOutgoing:  {FG: yellow},
			proto.StyleTarget:    {FG: red, Attr: proto.AttrUnderline},
			proto.StyleDefcon5:   {FG: blue},
			proto.StyleDefcon4:   {FG: green},
			proto.StyleDefcon3:   {FG: yellow},
			proto.StyleDefcon2:   {FG: red, Attr: proto.AttrBold},
			proto.StyleDefcon1:   {FG: bright, BG: &redBG, Attr: proto.AttrBold, NoColAttr: proto.AttrReverse},
			proto.StyleAlert:     {FG: red, Attr: proto.AttrBold},
			proto.StyleSuitRed:   {FG: red},
			proto.StyleSuitBlack: {FG: bright},
			proto.StyleSelected:  {FG: bright, Attr: proto.AttrReverse},
		},
	}
}

// Lip returns the Lip Gloss style for a cell, for the terminal's colour profile. Blink is
// not rendered here: the ui toggles blinking cells on its clock.
func (t *Theme) Lip(s proto.Style, a proto.Attr, p colorprofile.Profile) lipgloss.Style {
	spec, ok := t.Styles[s]
	if !ok {
		spec = t.Styles[proto.StyleText]
	}
	attr := spec.Attr | a
	st := lipgloss.NewStyle()
	switch p {
	case colorprofile.TrueColor, colorprofile.ANSI256:
		st = st.Foreground(lipgloss.Color(spec.FG.Hex)).Background(lipgloss.Color(t.bg(spec).Hex))
	case colorprofile.ANSI:
		st = st.Foreground(ansi.BasicColor(spec.FG.ANSI)).Background(ansi.BasicColor(t.bg(spec).ANSI))
	default: // ASCII and NoTTY: no colour at all
		attr |= spec.NoColAttr
		if s == proto.StyleDim || s == proto.StyleLand {
			st = st.Faint(true)
		}
	}
	return st.Bold(attr&proto.AttrBold != 0).
		Reverse(attr&proto.AttrReverse != 0).
		Underline(attr&proto.AttrUnderline != 0)
}

func (t *Theme) bg(s Spec) Color {
	if s.BG != nil {
		return *s.BG
	}
	return t.Background
}

// ValidName reports whether name is a theme.
func ValidName(name string) bool { return slices.Contains(Names(), name) }
