// Package lintfixture deliberately breaks every custom lint rule, so the self-test in
// internal/archtest can prove each rule still fires. Go wildcards (./...) skip testdata,
// so the normal lint run never sees this package.
package lintfixture

import (
	"math/rand/v2"
	"time"

	tea "charm.land/bubbletea/v2" // depguard: only internal/ui may import Bubble Tea
)

// Tick bypasses the single clock (forbidigo: tea.Tick).
func Tick() tea.Cmd { return tea.Tick(time.Second, func(time.Time) tea.Msg { return nil }) }

// Roll uses an unseedable top-level function (forbidigo: math/rand/v2).
func Roll() int { return rand.IntN(6) }

// Nap sleeps instead of using the clock (forbidigo: time.Sleep).
func Nap() { time.Sleep(time.Millisecond) }
