package theme

import (
	"math"
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
		for s := proto.StyleText; s <= proto.StyleSelected; s++ {
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
	pairs := [][2]proto.Style{
		{proto.StyleText, proto.StyleAccent},
		{proto.StyleText, proto.StyleBright},
		{proto.StyleText, proto.StyleDim},
		{proto.StyleIncoming, proto.StyleOutgoing},
		{proto.StyleDefcon1, proto.StyleDefcon2},
		{proto.StyleDefcon2, proto.StyleDefcon3},
	}
	for _, name := range Names() {
		th, _ := Get(name)
		for _, p := range pairs {
			a, b := th.Styles[p[0]], th.Styles[p[1]]
			if a.FG.ANSI == b.FG.ANSI && a.Attr == b.Attr && a.BG == nil && b.BG == nil {
				t.Errorf("%s: styles %c and %c look identical in 16 colours", name, p[0].Letter(), p[1].Letter())
			}
		}
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
		for s := proto.StyleText; s <= proto.StyleSelected; s++ {
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
