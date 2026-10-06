package sim

import (
	"strings"

	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// Scenario is a war game on the engine: data plus hooks.
type Scenario struct {
	Title string    // the map's heading
	Intro script.Ls // printed at the start
	// Art is the title picture, printed fast after the intro and before the first map: at
	// most ArtRows rows of printable ASCII capitals, each at most 80 columns. Nil for none.
	Art       script.Ls
	Regions   []Region
	Setup     func(s *State) // places units (AddUnit) and sets Vars and Control
	Verbs     []*Verb        // beyond MOVE, ATTACK and HOLD
	Events    []Event
	TurnLimit int
	// Victory decides the game after each turn; final is set on the last turn. It returns
	// the outcome for the player and the line that says why.
	Victory func(s *State, final bool) (proto.Outcome, string, bool)
	// AI gives WOPR's orders; nil uses DefaultAI.
	AI func(s *State) []Order
	// AttackMod adjusts a unit's attack (supply, weather); nil leaves it.
	AttackMod func(s *State, u *Unit, attack int) int
	// Upkeep runs after the event each turn (supply, contamination).
	Upkeep func(s *State)
	// Status is an extra line under the turn header (SUPPLY 4, ESCALATION 2); nil for none.
	Status func(s *State) string
	// RegionNote is shown after a region's name on the map (contamination); nil for none.
	RegionNote func(s *State, r int) string
	// Overlay is drawn over a region in the map's picture, above its ground (a gas cloud,
	// blowing sand), centred and cut to the region's width; "" for clear air. The row is
	// left out when no region has one. Nil for none.
	Overlay func(s *State, r int) string
}

// ArtRows is the most rows a scenario's Art may have, as for any block on the console.
// Scenario tests also keep the opening (art, first map, first prompt) on one screen; see
// Game.OpeningRows.
const ArtRows = 12

// Verb is an order a unit can be given.
type Verb struct {
	Name   string
	Help   string // one line, as HELP lists it
	Target bool   // takes a region
	Phase  int    // 0 movement, 1 combat, 2 everything else
	// Drastic verbs (ESCALATE, RELEASE) must be typed in full, never as a prefix.
	Drastic bool
	// Check says why u cannot do this (at target), or "" when it can.
	Check func(s *State, u *Unit, target int) string
	Apply func(s *State, u *Unit, target int)
}

// Event is an event card. Text is its headline (none when empty); Apply may Say more,
// such as which unit it touched.
type Event struct {
	Text  string
	Apply func(s *State)
}

// Order is one unit's order for the turn.
type Order struct {
	Unit   *Unit
	Verb   *Verb
	Target int
}

// AddUnit puts a unit of type t for side in region r; it is numbered within its side and
// type in the order units are added.
func (s *State) AddUnit(side int, t *UnitType, r int) *Unit {
	id := 1
	for _, u := range s.Units {
		if u.Side == side && u.Type == t {
			id++
		}
	}
	u := &Unit{ID: id, Side: side, Type: t, Region: r, Steps: 2}
	s.Units = append(s.Units, u)
	return u
}

// The built-in verbs.
var (
	Move = &Verb{
		Name: "MOVE", Help: engineText[TextVerbHelp][0].Text, Target: true, Phase: 0,
		Check: func(s *State, u *Unit, to int) string {
			if !s.CanMove(u, to) {
				return s.Scenario.text(TextNoMove)
			}
			return ""
		},
		Apply: func(_ *State, u *Unit, to int) { u.Region = to },
	}
	Attack = &Verb{
		Name: "ATTACK", Help: engineText[TextVerbHelp][1].Text, Target: true, Phase: 1,
		Check: func(s *State, u *Unit, at int) string {
			if !s.CanAttack(u, at) {
				return s.Scenario.text(TextNoAttack)
			}
			return ""
		},
		Apply: func(s *State, u *Unit, at int) { s.Attack(u, at) },
	}
	Hold = &Verb{Name: "HOLD", Help: engineText[TextVerbHelp][2].Text, Phase: 2}
)

// verbs is every verb in the scenario, built-ins first.
func (sc *Scenario) verbs() []*Verb { return append([]*Verb{Move, Attack, Hold}, sc.Verbs...) }

// TextKey names the engine's own screen text.
type TextKey int

// Engine text.
const (
	TextAttack TextKey = iota
	TextDestroyed
	TextWOPRs
	TextNoMove
	TextNoAttack
	TextTurn
	TextOrders
	TextHelp
	TextWhere
	TextUnknown
	TextMapHead
	TextYou
	TextWOPR
	TextHeld
	TextRatios
	TextRatioHead
	TextEnd
	TextTerrains
	TextVerbHelp
	TextLegend
	TextVoid
	TextLoses
	TextResults
	TextGround
	TextRoad
	TextSpan
	textKeys // the number of keys
)

// engineText is the engine's screen text, all original. A # is filled in when shown.
var engineText = map[TextKey]script.Ls{
	TextAttack:    script.Orig("# ATTACKS # AT #: #."),
	TextDestroyed: script.Orig("# IS DESTROYED."),
	TextWOPRs:     script.Orig("WOPR'S "),
	TextNoMove:    script.Orig("IT CANNOT GET THERE THIS TURN."),
	TextNoAttack:  script.Orig("NO ENEMY THERE IN RANGE."),
	TextTurn:      script.Orig("TURN # OF #"),
	TextOrders:    script.Orig("# IN #: "),
	TextHelp:      script.Orig("ORDERS: #.", "STATUS SHOWS THE MAP; END HOLDS THE REST OF THIS TURN'S UNITS."),
	TextWhere:     script.Orig("WHICH REGION? A NUMBER OR A NAME FROM THE MAP."),
	TextUnknown:   script.Orig("UNKNOWN ORDER. TYPE HELP."),
	TextMapHead:   script.Orig("REGION", "GROUND", "HELD", "YOUR UNITS", "WOPR'S UNITS"),
	TextYou:       script.Orig("YOU"),
	TextWOPR:      script.Orig("WOPR"),
	TextHeld:      script.Orig("-"),
	TextRatios:    script.Orig("KILL RATIOS"),
	TextRatioHead: script.Orig("STEPS LOST                 YOU      WOPR"),
	TextEnd:       script.Orig("THE GAME ENDS."),
	TextTerrains:  script.Orig("OPEN", "ROUGH", "CITY"),
	TextVerbHelp: script.Orig(
		"MOVE <REGION>: AS FAR AS THE UNIT'S MOVE, NOT THROUGH THE ENEMY.",
		"ATTACK <REGION>: AN ENEMY IN RANGE.",
		"HOLD: STAY PUT.",
	),
	TextLegend: script.Orig(
		"RESULTS: NE NO EFFECT; AE THE ATTACKER LOSES A STEP; EX BOTH DO;",
		"DR THE DEFENDER FALLS BACK (OR LOSES A STEP); DE THE DEFENDER LOSES A STEP.",
	),
	TextVoid:    script.Orig("# STANDS FAST: #"),
	TextLoses:   script.Orig("# LOSES A STEP."),
	TextResults: script.Orig("NE", "AE", "EX", "DR", "DE"),
	// The map's picture (diagram.go): two rows of ground for each terrain, in Terrain
	// order (open sand or fields, rough hills, a city's roofs); the road's paving, a
	// region's number on it, a unit of yours, one hidden, one of WOPR's, and more than fit;
	// the ends and line of a stretch one side holds.
	TextGround: script.Orig(
		" .  .  .", ". .  .  .",
		" /\\  /\\", "/  \\/  \\",
		" _ [] _", "|#|##|#|",
	),
	TextRoad: script.Orig("=", "(#)", ">", "~", "<", "+"),
	TextSpan: script.Orig("<", "-", ">"),
}

// EngineLines is the engine's text, for the provenance test.
func EngineLines() []script.Ls {
	out := make([]script.Ls, 0, len(engineText))
	for k := range textKeys {
		out = append(out, engineText[k])
	}
	return out
}

func (sc *Scenario) text(k TextKey, args ...string) string {
	t := engineText[k][0].Text
	for _, a := range args {
		t = strings.Replace(t, "#", a, 1)
	}
	return t
}

func terrainName(t Terrain) string { return engineText[TextTerrains][t].Text }
