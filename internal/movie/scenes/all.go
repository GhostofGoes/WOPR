package scenes

// All returns the film's scenes in order (docs/PLAN.md §7).
func All() []Scene { return film }

// film is the scene list, in the film's order.
var film = []Scene{firstContact, joshua, firstStrike, callBack, noradTerminal, climax}
