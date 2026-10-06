package sim

import (
	"fmt"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/script"
)

// Simple is a plain strategy for scripted players in tests and the movie: attack the
// weakest enemy in range, else advance toward WOPR's rear, else hold.
func Simple(s *State, u *Unit) string {
	best, bestCol := -1, -1
	for at := range s.Regions {
		if !s.CanAttack(u, at) {
			continue
		}
		if col := Column(s.AttackOf(u), s.DefenceOf(strongest(s, at, Enemy(u.Side))), s.Regions[at].Terrain); col > bestCol {
			best, bestCol = at, col
		}
	}
	if best >= 0 {
		return fmt.Sprintf("ATTACK %d", best+1)
	}
	if to := s.Toward(u, len(s.Regions)-1); to >= 0 {
		return fmt.Sprintf("MOVE %d", to+1)
	}
	return "HOLD"
}

// OpeningRows is how many console rows the opening takes after the intro: the art, the
// first map and the first prompt. Scenario tests keep it within 23 rows (a 24-row console
// less the front panel), so the picture and the map are on the screen together.
func (g *Game) OpeningRows() int { return len(g.sc.Art) + len(g.mapLines()) + 1 }

// MapLines is the map as it prints now, for tests that look at it.
func (g *Game) MapLines() []string { return g.mapLines() }

// ArtProblems lists what is wrong with a scenario's pictures: the title (at most ArtRows
// rows) and any overlay blocks. Every line must be printable ASCII in capitals, at most 80
// columns, with no trailing spaces, and every block must be in lines, the scenario's
// script text, where the provenance test checks it.
func ArtProblems(lines []script.Ls, title script.Ls, more ...script.Ls) []string {
	var bad []string
	if len(title) == 0 || len(title) > ArtRows {
		bad = append(bad, fmt.Sprintf("the title art is %d rows; it must be 1 to %d", len(title), ArtRows))
	}
	for _, art := range append([]script.Ls{title}, more...) {
		if len(art) == 0 {
			continue
		}
		listed := false
		for _, block := range lines {
			listed = listed || len(block) > 0 && &block[0] == &art[0]
		}
		if !listed {
			bad = append(bad, fmt.Sprintf("%q... is not in the script text", art[0].Text))
		}
		for _, l := range art {
			if strings.ContainsFunc(l.Text, func(r rune) bool { return r < 0x20 || r > 0x7e }) {
				bad = append(bad, fmt.Sprintf("%q is not printable ASCII", l.Text))
			}
			if len(l.Text) > 80 || l.Text != strings.ToUpper(l.Text) || l.Text != strings.TrimRight(l.Text, " ") {
				bad = append(bad, fmt.Sprintf("%q is wider than 80 columns, not in capitals or ends in spaces", l.Text))
			}
		}
	}
	return bad
}

// IllegalAIOrders lists WOPR's current orders that its own checks would refuse.
func (g *Game) IllegalAIOrders() []string {
	var bad []string
	for _, o := range g.AIOrders() {
		if o.Verb.Check != nil {
			if why := o.Verb.Check(g.s, o.Unit, o.Target); why != "" {
				bad = append(bad, fmt.Sprintf("%s %s %d: %s", o.Unit.Name(), o.Verb.Name, o.Target+1, why))
			}
		}
		if o.Unit.Side != WOPR || !o.Unit.Alive() {
			bad = append(bad, o.Unit.Name()+": not WOPR's, or dead")
		}
	}
	return bad
}
