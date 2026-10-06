package sim

import "fmt"

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
