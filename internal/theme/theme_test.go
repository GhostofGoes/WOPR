package theme

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"

	"github.com/GhostofGoes/WOPR/internal/proto"
)

func TestEveryThemeDefinesEveryStyle(t *testing.T) {
	t.Parallel()
	for _, name := range Names() {
		th, ok := Get(name)
		if !ok {
			t.Fatalf("theme %q missing", name)
		}
		for _, s := range proto.Styles() {
			spec, ok := th.Styles[s]
			if !ok {
				t.Errorf("%s: style %c undefined", name, s.Letter())
				continue
			}
			if spec.FG.ANSI < 0 || spec.FG.ANSI > 15 || !strings.HasPrefix(spec.FG.Hex, "#") {
				t.Errorf("%s: style %c has a bad colour %+v", name, s.Letter(), spec.FG)
			}
		}
	}
	if !ValidName(Default) || ValidName("pink") {
		t.Error("ValidName")
	}
}

// In 16-colour mode, styles whose difference carries meaning must still differ in
// colour index or attributes (U-7).
func TestMeaningfulPairsStayDistinctIn16Colours(t *testing.T) {
	t.Parallel()
	for _, name := range Names() {
		th, _ := Get(name)
		for _, p := range Distinct() {
			a, b := th.Styles[p[0]], th.Styles[p[1]]
			if a.FG.ANSI == b.FG.ANSI && a.Attr == b.Attr && a.BG == nil && b.BG == nil {
				t.Errorf("%s: styles %c and %c look identical in 16 colours", name, p[0].Letter(), p[1].Letter())
			}
		}
	}
}

// Without colour (NO_COLOR, TERM=xterm-mono: the ASCII profile) a style is only its
// attributes, and Dim and Land are faint. The pairs whose difference carries meaning must
// still look different there, and a style that paints its own background (norad's DEFCON 1,
// white on red) must keep a visible block: Reverse.
func TestMeaningSurvivesWithoutColour(t *testing.T) {
	t.Parallel()
	look := func(th *Theme, s proto.Style) string {
		return th.Lip(s, 0, colorprofile.ASCII).Render("x")
	}
	for _, name := range Names() {
		th, _ := Get(name)
		for _, p := range Distinct() {
			if a, b := look(th, p[0]), look(th, p[1]); a == b {
				t.Errorf("%s: styles %c and %c both render %q without colour", name, p[0].Letter(), p[1].Letter(), a)
			}
		}
		for _, s := range proto.Styles() {
			if spec := th.Styles[s]; spec.BG != nil && (spec.Attr|spec.NoColAttr)&proto.AttrReverse == 0 {
				t.Errorf("%s: style %c paints a background, which vanishes without colour unless it is reversed", name, s.Letter())
			}
		}
	}
}

// Every style in every theme renders under each colour profile with only what that profile
// allows: 24-bit colour, the 16 ANSI colours, or no colour at all (attributes only).
func TestEveryStyleUnderEveryProfile(t *testing.T) {
	t.Parallel()
	for _, name := range Names() {
		th, _ := Get(name)
		for _, s := range proto.Styles() {
			for _, p := range []colorprofile.Profile{colorprofile.TrueColor, colorprofile.ANSI, colorprofile.ASCII} {
				params := sgrParams(th.Lip(s, proto.AttrBold, p).Render("x"))
				truecolour, colour := false, false
				for i, v := range params {
					switch n, _ := strconv.Atoi(v); {
					case (n == 38 || n == 48) && i+1 < len(params) && params[i+1] == "2":
						truecolour = true
						colour = true
					case n >= 30 && n <= 49, n >= 90 && n <= 107:
						colour = true
					}
				}
				if want := p != colorprofile.ASCII; colour != want {
					t.Errorf("%s %c under %v: colour %v, want %v (%v)", name, s.Letter(), p, colour, want, params)
				}
				if want := p == colorprofile.TrueColor; truecolour != want {
					t.Errorf("%s %c under %v: 24-bit colour %v, want %v (%v)", name, s.Letter(), p, truecolour, want, params)
				}
				if !slices.Contains(params, "1") {
					t.Errorf("%s %c under %v: the cell's own Bold was dropped (%v)", name, s.Letter(), p, params)
				}
			}
		}
	}
}

// sgrParams lists the parameters of every SGR sequence in s; a sub-parameter (4:3) counts
// as its main one.
func sgrParams(s string) []string {
	var out []string
	for {
		i := strings.Index(s, "\x1b[")
		if i < 0 {
			return out
		}
		s = s[i+2:]
		j := strings.IndexByte(s, 'm')
		if j < 0 {
			return out
		}
		for p := range strings.SplitSeq(s[:j], ";") {
			main, _, _ := strings.Cut(p, ":")
			out = append(out, main)
		}
		s = s[j+1:]
	}
}

func TestLipByProfile(t *testing.T) {
	t.Parallel()
	th, _ := Get("norad")
	tc := th.Lip(proto.StyleIncoming, 0, colorprofile.TrueColor).Render("*")
	if !strings.Contains(tc, "38;2;255;59;48") {
		t.Errorf("truecolor incoming = %q", tc)
	}
	ansi16 := th.Lip(proto.StyleIncoming, 0, colorprofile.ANSI).Render("*")
	if !strings.Contains(ansi16, "91") { // bright red, index 9
		t.Errorf("16-colour incoming = %q", ansi16)
	}
	mono := th.Lip(proto.StyleDefcon1, 0, colorprofile.ASCII).Render("1")
	if strings.Contains(mono, "38;") || !strings.Contains(mono, "7") {
		t.Errorf("no-colour DEFCON 1 should be reverse without colour, got %q", mono)
	}
}

// Default 16-colour palettes the ANSI profile is likely to meet: xterm's, and VGA's (the
// Linux console, TERM=linux).
var (
	xtermPalette = [16]string{
		"#000000", "#cd0000", "#00cd00", "#cdcd00", "#0000ee", "#cd00cd", "#00cdcd", "#e5e5e5",
		"#7f7f7f", "#ff0000", "#00ff00", "#ffff00", "#5c5cff", "#ff00ff", "#00ffff", "#ffffff",
	}
	vgaPalette = [16]string{
		"#000000", "#aa0000", "#00aa00", "#aa5500", "#0000aa", "#aa00aa", "#00aaaa", "#aaaaaa",
		"#555555", "#ff5555", "#55ff55", "#ffff55", "#5555ff", "#ff55ff", "#55ffff", "#ffffff",
	}
)

// Every style must stay readable on its background. Truecolor meets WCAG AA (4.5:1), or
// 3:1 for the deliberately faint Dim and Land. The 16-colour palettes only promise 3:1,
// and VGA's bright black (imsai Dim, 2.8:1) is the one accepted exception.
func TestTextContrast(t *testing.T) {
	t.Parallel()
	for _, name := range Names() {
		th, _ := Get(name)
		for _, s := range proto.Styles() {
			spec := th.Styles[s]
			bg := th.bg(spec)
			faint := s == proto.StyleDim || s == proto.StyleLand
			want := 4.5
			if faint {
				want = 3
			}
			if got := contrast(spec.FG.Hex, bg.Hex); got < want {
				t.Errorf("%s %c: truecolor contrast %.2f, want %.1f", name, s.Letter(), got, want)
			}
			if got := contrast(xtermPalette[spec.FG.ANSI], xtermPalette[bg.ANSI]); got < 3 {
				t.Errorf("%s %c: xterm 16-colour contrast %.2f, want 3", name, s.Letter(), got)
			}
			vgaWant := 3.0
			if faint && spec.FG.ANSI == 8 {
				vgaWant = 2.8
			}
			if got := contrast(vgaPalette[spec.FG.ANSI], vgaPalette[bg.ANSI]); got < vgaWant {
				t.Errorf("%s %c: VGA 16-colour contrast %.2f, want %.1f", name, s.Letter(), got, vgaWant)
			}
		}
	}
}

// contrast is the WCAG 2 contrast ratio of two #rrggbb colours.
func contrast(a, b string) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func luminance(hex string) float64 {
	var rgb [3]float64
	for i := range rgb {
		v, err := strconv.ParseUint(hex[1+2*i:3+2*i], 16, 8)
		if err != nil {
			panic(hex)
		}
		c := float64(v) / 255
		if c <= 0.03928 {
			rgb[i] = c / 12.92
		} else {
			rgb[i] = math.Pow((c+0.055)/1.055, 2.4)
		}
	}
	return 0.2126*rgb[0] + 0.7152*rgb[1] + 0.0722*rgb[2]
}
