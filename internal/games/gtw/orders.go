package gtw

import (
	"strconv"
	"strings"

	"github.com/GhostofGoes/WOPR/internal/script"
)

// orderKind is what a line typed at a strike prompt asks for.
type orderKind uint8

const (
	orderFire    orderKind = iota // fire the percentages in alloc
	orderAuto                     // the war plan, now and for every later strike
	orderHelp                     // explain, and ask again
	orderRefused                  // refusal holds the line to print; ask again
)

// Words an order may contain.
var (
	orderFiller = map[string]bool{
		"FIRE": true, "LAUNCH": true, "COMMIT": true, "SEND": true, "ORDER": true, "ORDERS": true, "AND": true,
		"WITH": true, "OF": true, "THE": true, "MY": true, "OUR": true, "PERCENT": true, "PCT": true,
	}
	orderPlan = map[string]bool{"PLAN": true, "YES": true, "Y": true, "OK": true, "GO": true, "EXECUTE": true}
	orderAll  = map[string]bool{"ALL": true, "FULL": true, "MAX": true, "EVERYTHING": true}
	orderHold = map[string]bool{
		"HOLD": true, "NONE": true, "NOTHING": true, "NO": true, "N": true, "STOP": true, "ABORT": true, "WAIT": true,
		"CEASE": true, "CEASEFIRE": true, "DONT": true, "PEACE": true, "SURRENDER": true,
	}
	orderSystems = map[string]int{
		"ICBM": sysICBM, "ICBMS": sysICBM, "MISSILES": sysICBM,
		"SLBM": sysSLBM, "SLBMS": sysSLBM, "SUB": sysSLBM, "SUBS": sysSLBM, "SUBMARINES": sysSLBM,
		"BOMBER": sysBombers, "BOMBERS": sysBombers, "AIR": sysBombers,
	}
	orderQty = map[string]int{"ALL": 100, "HALF": 50, "NONE": 0}
)

// parseOrder reads a line typed at a strike prompt; plan is the Enter default. The first
// rule that matches wins (docs/PLAN.md §6.2).
func parseOrder(raw string, plan [3]int) (alloc [3]int, kind orderKind, refusal script.Ls) {
	if strings.TrimSpace(raw) == "" {
		return plan, orderFire, nil
	}
	if signedOrDecimal(raw) {
		return alloc, orderRefused, lineBadPct
	}
	norm := climaxWords(raw)
	if norm == "" { // "?", "!!!"
		return alloc, orderHelp, nil
	}
	if norm == "LIST GAMES" || wantsTicTacToe(norm) {
		return alloc, orderRefused, lineMustRun
	}
	switch gameNamed(norm) {
	case "":
	case "GLOBAL THERMONUCLEAR WAR":
		return alloc, orderRefused, lineRunning
	default:
		return alloc, orderRefused, lineMustRun
	}
	var words []string
	for _, w := range strings.Fields(norm) {
		if !orderFiller[w] {
			words = append(words, w)
		}
	}
	if len(words) == 0 { // FIRE, LAUNCH
		return plan, orderFire, nil
	}
	if len(words) == 1 {
		switch w := words[0]; {
		case w == "HELP":
			return alloc, orderHelp, nil
		case orderPlan[w]:
			return plan, orderFire, nil
		case w == "AUTO" || w == "WOPR":
			return plan, orderAuto, nil
		case orderAll[w]:
			return [3]int{100, 100, 100}, orderFire, nil
		case orderHold[w]:
			return alloc, orderFire, nil
		}
	}
	return parseAlloc(words)
}

// parseAlloc reads percentages: one number for all three, three numbers in the order
// ICBM SLBM BOMBERS, or systems with quantities ("ICBM 50 BOMBERS ALL", "50 SUBS"). A
// system named without any quantity fires all of it; one not named fires none.
func parseAlloc(words []string) (alloc [3]int, kind orderKind, refusal script.Ls) {
	nums, systems := 0, 0
	for _, w := range words {
		if isDigits(w) {
			if v, err := strconv.Atoi(w); err != nil || v > 100 {
				return alloc, orderRefused, lineBadPct
			}
			nums++
		}
		if _, ok := orderSystems[w]; ok {
			systems++
		}
	}
	if nums == len(words) {
		v := make([]int, len(words))
		for i, w := range words {
			v[i], _ = strconv.Atoi(w)
		}
		switch len(v) {
		case 1:
			return [3]int{v[0], v[0], v[0]}, orderFire, nil
		case 3:
			return [3]int{v[0], v[1], v[2]}, orderFire, nil
		}
		return alloc, orderRefused, lineBadOrder
	}
	seen := [3]bool{}
	name := func(w string) bool {
		s, ok := orderSystems[w]
		if !ok || seen[s] {
			return false
		}
		seen[s] = true
		return true
	}
	if systems == len(words) { // "SUBS", "ICBMS AND BOMBERS"
		for _, w := range words {
			if !name(w) {
				return [3]int{}, orderRefused, lineBadOrder
			}
			alloc[orderSystems[w]] = 100
		}
		return alloc, orderFire, nil
	}
	if len(words)%2 != 0 {
		return alloc, orderRefused, lineBadOrder
	}
	qty := func(w string) (int, bool) {
		if isDigits(w) {
			v, _ := strconv.Atoi(w)
			return v, true
		}
		v, ok := orderQty[w]
		return v, ok
	}
	_, sysFirst := orderSystems[words[0]] // "ICBM 50 ..." or "50 ICBM ..."
	for i := 0; i < len(words); i += 2 {
		s, q := words[i], words[i+1]
		if !sysFirst {
			s, q = q, s
		}
		v, ok := qty(q)
		if !ok || !name(s) {
			return [3]int{}, orderRefused, lineBadOrder
		}
		alloc[orderSystems[s]] = v
	}
	return alloc, orderFire, nil
}

func isDigits(w string) bool {
	if w == "" {
		return false
	}
	for _, r := range w {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// signedOrDecimal reports a sign or a decimal point touching a digit ("-50", "+5", "12.5"),
// which Normalize would silently turn into other numbers.
func signedOrDecimal(raw string) bool {
	rs := []rune(raw)
	digit := func(i int) bool { return i >= 0 && i < len(rs) && rs[i] >= '0' && rs[i] <= '9' }
	for i, r := range rs {
		switch {
		case (r == '-' || r == '+') && digit(i+1) && !digit(i-1):
			return true
		case r == '.' && digit(i-1) && digit(i+1):
			return true
		}
	}
	return false
}
