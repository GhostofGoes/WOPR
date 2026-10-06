package chess

import (
	"strings"
	"testing"

	cg "github.com/corentings/chess/v2"

	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// Every line of art and panel text is printable ASCII in capitals, at most 80 columns,
// tagged original and listed in Lines (where the provenance test checks it); the title card
// is at most 12 rows, the panel's height.
func TestArt(t *testing.T) {
	t.Parallel()
	if len(artTitle) > info.PanelRows {
		t.Errorf("the title card is %d rows; the panel is %d", len(artTitle), info.PanelRows)
	}
	for _, block := range []script.Ls{artTitle, panelWOPR, panelYou, panelSides, panelTaken, panelLast, panelMove, panelCheck} {
		listed := false
		for _, l := range Lines {
			listed = listed || &l[0] == &block[0]
		}
		if !listed {
			t.Errorf("%q is not in Lines", block[0].Text)
		}
		for _, l := range block {
			if strings.ContainsFunc(l.Text, func(r rune) bool { return r < 0x20 || r > 0x7e }) {
				t.Errorf("%q is not printable ASCII", l.Text)
			}
			if len(l.Text) > 80 || l.Text != strings.ToUpper(l.Text) || l.Text != strings.TrimRight(l.Text, " ") {
				t.Errorf("%q: wider than 80 columns, lower case, or trailing spaces", l.Text)
			}
			if l.Prov != script.Original {
				t.Errorf("%q is tagged %q, not original", l.Text, l.Prov)
			}
		}
	}
}

// The panel stays inside its 80 columns and PanelRows rows, centred on a wider canvas, even
// with everything showing: fifteen pieces taken, a check, WOPR's last move.
func TestPanelFits(t *testing.T) {
	t.Parallel()
	mate := cg.NewGame() // fool's mate: White in check, beside its side's name
	for _, m := range []string{"f3", "e5", "g4", "Qh4"} {
		if err := mate.PushNotationMove(m, cg.AlgebraicNotation{}, nil); err != nil {
			t.Fatal(err)
		}
	}
	games := map[string]*cg.Game{"fool's mate": mate}
	for _, fen := range []string{
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		"4k3/8/8/8/8/8/8/R3K3 b - - 0 40",             // Black has lost fifteen pieces
		"rnbqkbnr/pppppppp/8/8/8/8/8/4K3 w kq - 0 50", // White has lost fifteen
	} {
		opt, err := cg.FEN(fen)
		if err != nil {
			t.Fatal(err)
		}
		games[fen] = cg.NewGame(opt)
	}
	for fen, game := range games {
		g := &Game{g: game, from: cg.E7, last: cg.E5, lastText: "E7E5"}
		const w, h = 100, 20
		c := proto.NewCanvas(w, h)
		g.View(c)
		for y := range h {
			for x := range w {
				if c.At(x, y).R != ' ' && (y >= info.PanelRows || x < (w-80)/2 || x >= (w+80)/2) {
					t.Fatalf("%s: the panel draws at %d,%d, outside 80x%d:\n%s", fen, x, y, info.PanelRows, c.String())
				}
			}
		}
		if lines := g.result(proto.Draw).Lines; len(lines) > 12 {
			t.Errorf("%s: the final position is %d lines", fen, len(lines))
		}
		if fen == "fool's mate" && !strings.Contains(c.String(), "WHITE  CHECK") {
			t.Errorf("the side in check is marked:\n%s", c.String())
		}
	}
}

func TestTaken(t *testing.T) {
	t.Parallel()
	for fen, want := range map[string][2]string{
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1": {"", ""},
		"r1bqkb1r/pppp1ppp/8/8/8/8/PPP2PPP/RNBQKB1R w KQkq - 0 6":  {"NPP", "NNP"},
		"QQ2k3/8/8/8/8/8/PPPPPP2/4K3 w - - 0 40":                   {"RRBBNNP", "QRRBBNNPPPPPPPP"}, // a pawn made a queen
	} {
		opt, err := cg.FEN(fen)
		if err != nil {
			t.Fatal(err)
		}
		b := cg.NewGame(opt).Position().Board()
		white, black := strings.Join(taken(b, cg.White), ""), strings.ToUpper(strings.Join(taken(b, cg.Black), ""))
		if white != want[0] || black != want[1] {
			t.Errorf("%s: White lost %q, Black %q; want %q", fen, white, black, want)
		}
	}
}
