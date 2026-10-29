// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package gtw

import "github.com/GhostofGoes/WOPR/internal/script"

// WOPR remarks on each stage as it flies (docs/PLAN.md §6.2): a line on the board's last row,
// in place of the legend, typed at speech pace and then gone, so the strip keeps its three
// report lines a strike. The film's board (movie mode) has no remarks: nobody gives orders.
const (
	remarkStream = 1  // the remarks' random stream; stream 0's draws stay fixed whatever the orders
	remarkChars  = 3  // characters typed a frame: 30 a second, WOPR's speech pace
	remarkFrames = 30 // frames a remark stays on the board before the legend returns
)

// reaction is what WOPR remarks on: the order a stage was started with, or the stage itself.
type reaction uint8

const (
	reactFirst      reaction = iota // the first strike, at the listed targets
	reactPlan                       // the war plan, at a strike prompt
	reactPlanAgain                  // the war plan, at the second strike prompt too
	reactAll                        // everything
	reactHold                       // nothing
	reactQuiet                      // nothing, in a strike where WOPR launches no missiles either
	reactSilosFull                  // an order that keeps every ICBM in its silo
	reactPartial                    // any other order
	reactAuto                       // AUTO
	reactAutoStrike                 // a strike WOPR orders under AUTO
	reactEmpty                      // a strike with nothing left to order
	reactFinal                      // WOPR's DEFCON 1 attack
)

// reactLines are each reaction's variants.
var reactLines = [...]script.Ls{
	reactFirst: lineReactFirst, reactPlan: lineReactPlan, reactPlanAgain: lineReactPlanAgain, reactAll: lineReactAll,
	reactHold: lineReactHold, reactQuiet: lineReactQuiet, reactSilosFull: lineReactSilosFull, reactPartial: lineReactPartial,
	reactAuto: lineReactAuto, reactAutoStrike: lineReactAutoStrike, reactEmpty: lineReactEmpty, reactFinal: lineReactFinal,
}

// reactionTo is WOPR's reaction to an order fired at a strike prompt. plan is the strike's
// war plan, planned whether an earlier prompt took the plan too, and icbms the ICBMs you hold.
// The plan wins over ALL: strike 3's plan is everything.
func reactionTo(alloc, plan [3]int, planned bool, icbms int) reaction {
	switch {
	case alloc == plan && planned:
		return reactPlanAgain
	case alloc == plan:
		return reactPlan
	case alloc == [3]int{}:
		return reactHold
	case alloc == [3]int{100, 100, 100}:
		return reactAll
	case alloc[sysICBM] == 0 && icbms > 0:
		return reactSilosFull
	}
	return reactPartial
}

// remark draws r's line for the stage just begun: a variant from the seed, never the one this
// reaction gave last, so a reaction that comes twice reads differently. A # is your first
// target.
func (g *Game) remark(r reaction) {
	g.remarked = ""
	if g.film {
		return
	}
	n, last := len(reactLines[r]), g.lastRemark[r]-1 // -1: none yet
	var i int
	if last < 0 {
		i = g.voice.IntN(n)
	} else if i = g.voice.IntN(n - 1); i >= last {
		i++
	}
	g.lastRemark[r] = i + 1
	g.remarked = fill(reactLines[r][i], g.targets[0].Name)
}

// remarkShown is as much of the stage's remark as the board shows at this frame: typed in,
// then held, then gone.
func (g *Game) remarkShown() string {
	if g.phase != flight || g.frame >= remarkFrames {
		return ""
	}
	return g.remarked[:min(len(g.remarked), remarkChars*(g.frame+1))]
}
