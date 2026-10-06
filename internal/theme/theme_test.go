package theme

import (
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
