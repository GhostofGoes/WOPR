package wopr

import (
	"strings"

	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/prompt"
)

// Phrases that ask for a game when they start a clause (docs/PLAN.md §4.6).
var (
	intentVerbs = []string{
		"I WANT TO PLAY", "I'D LIKE TO PLAY", "ID LIKE TO PLAY", "CAN WE PLAY", "SHALL WE PLAY",
		"LETS PLAY", "HOW ABOUT", "WHAT ABOUT", "PLAY",
	}
	intentFillers = map[string]bool{
		"LATER": true, "LOVE": true, "TO": true, "OK": true, "OKAY": true, "YES": true, "SURE": true,
		"WELL": true, "FINE": true, "THEN": true, "SO": true, "NOW": true,
	}
	gameArticles = []string{"A GAME OF", "A ROUND OF", "A NICE GAME OF", "A GOOD GAME OF", "SOME", "A", "THE"}
	// Words that may follow the game ("LET'S PLAY CHESS NOW"). Anything else after the game
	// means the line is not a plain request, so it goes to the brain.
	intentTrailers = []string{"NOW", "INSTEAD", "PLEASE", "THEN", "AGAIN", "TODAY", "WITH ME", "WITH YOU"}
)

// gameIntent finds an explicit request for a game in input. It returns the first clause
// that, after optional fillers, starts with a request phrase followed by a game name, and
// has no negation before the phrase. Input that is exactly a game's name also counts.
func gameIntent(input string, reg *games.Registry) (games.Entry, bool) {
	if e, ok := reg.Exact(input); ok {
		return e, true
	}
	for _, c := range prompt.Clauses(input) {
		words := c.Words
		start := 0
		for start < len(words) && intentFillers[words[start]] {
			start++
		}
		rest := words[start:]
		for _, verb := range intentVerbs {
			if !prompt.HasPrefix(rest, verb) {
				continue
			}
			if prompt.NegatedBefore(words, start+len(strings.Fields(verb))) {
				break
			}
			object := requestObject(rest[len(strings.Fields(verb)):])
			if e, ok := reg.Exact(object); ok {
				return e, true
			}
			if _, isNum := prompt.Number(object); !isNum && (len(object) < 3 || prompt.Negators[object]) {
				break // "HOW ABOUT NO?" asks for nothing, least of all NOUGHTS AND CROSSES
			}
			if e, err := reg.Resolve(object); err == nil {
				return e, true // a number or a unique prefix after an explicit verb
			}
			break
		}
	}
	return games.Entry{}, false
}

// requestObject is what a request asks for: the words after the verb without a leading
// article or a trailing NOW, PLEASE and the like. A bare article asks for nothing.
func requestObject(words []string) string {
	object := strings.Join(words, " ")
	for _, a := range gameArticles {
		object = strings.TrimPrefix(object, a+" ")
	}
	object = trimTrailers(object)
	for _, a := range gameArticles {
		if object == a {
			return ""
		}
	}
	return object
}

// trimTrailers removes intentTrailers from the end of a request's object.
func trimTrailers(object string) string {
	for {
		before := object
		for _, t := range intentTrailers {
			object = strings.TrimSuffix(object, " "+t)
		}
		if object == before {
			return object
		}
	}
}
