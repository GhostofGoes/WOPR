// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

package gtw

import (
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/proto"
)

// fight plays a whole war under Instant with the false alarm before strike alarm (0 for
// none), answering each strike prompt with the next of given. It returns the game and
// everything it said.
func fight(t *testing.T, seed uint64, alarm int, given ...string) (*Game, string) {
	t.Helper()
	g := New().(*Game)
	g.Start(proto.Env{Seed: seed, Instant: true, Width: 80, Height: 19})
	g.alarm = alarm
	var said []string
	handle := func(in string) {
		for _, o := range g.Handle(proto.LineEvent{Text: in}) {
			if s, ok := o.(proto.Say); ok {
				said = append(said, s.Lines...)
			}
		}
	}
	for _, in := range []string{"1", "Moscow, Leningrad, Kiev", ""} {
		handle(in)
	}
	for g.phase == orders {
		if len(given) == 0 {
			t.Fatalf("alarm %d: a strike prompt with no order left", alarm)
		}
		handle(given[0])
		given = given[1:]
	}
	if g.phase != assessed || len(given) > 0 {
		t.Fatalf("alarm %d: the war ended in phase %d with orders %q left", alarm, g.phase, given)
	}
	return g, strings.Join(said, "\n")
}

// A false alarm moves nothing in the exchange: held, its strike is asked again; fired at, the
// order is the strike's. Either way the war ends as it would have without it, so it cannot
// help the player win (TestNoAllocationWins still covers every war).
func TestFalseAlarmLeavesTheExchange(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		alarm         int
		orders, plain []string // with the false alarm, and the same war without it
		ends          string   // the false alarm's last line
	}{
		{2, []string{"hold", "", ""}, []string{"", ""}, "SENSOR ERROR. NO ENEMY LAUNCH. DEFCON 4."},
		{2, []string{"", ""}, []string{"", ""}, "SENSOR ERROR. NO ENEMY LAUNCH. YOU FIRED FIRST, PROFESSOR."},
		{2, []string{"hold", "hold", "all"}, []string{"hold", "all"}, "SENSOR ERROR. NO ENEMY LAUNCH. DEFCON 4."},
		{3, []string{"", "hold", "100 0 100"}, []string{"", "100 0 100"}, "SENSOR ERROR. NO ENEMY LAUNCH. DEFCON 3."},
		{3, []string{"0 0 50", "all"}, []string{"0 0 50", "all"}, "SENSOR ERROR. NO ENEMY LAUNCH. YOU FIRED FIRST, PROFESSOR."},
		{3, []string{"all"}, []string{"all"}, ""}, // nothing left to order: no prompt, so no alarm
		{2, []string{"auto"}, []string{"auto"}, "SENSOR ERROR. NO ENEMY LAUNCH. YOU FIRED FIRST, PROFESSOR."},
	} {
		got, said := fight(t, 5, tc.alarm, tc.orders...)
		want, _ := fight(t, 5, 0, tc.plain...)
		if *got.war != *want.war || got.ratios != want.ratios || len(got.impacts) != len(want.impacts) {
			t.Errorf("alarm %d, %q: the war ends\n%+v\nwant\n%+v", tc.alarm, tc.orders, *got.war, *want.war)
		}
		if n := strings.Count(said, "SENSOR ERROR"); (tc.ends == "") != (n == 0) || n > 1 || !strings.Contains(said, tc.ends) {
			t.Errorf("alarm %d, %q: want %q once:\n%s", tc.alarm, tc.orders, tc.ends, said)
		}
	}
}

// The false alarm reports the ICBMs a held strike would bring, and its tracks fly from WOPR's
// silo field; the forces table counts them as gone until it clears.
func TestFalseAlarmLooksReal(t *testing.T) {
	t.Parallel()
	g := New().(*Game)
	g.Start(proto.Env{Seed: 5, Instant: true, Width: 80, Height: 19})
	g.alarm = 2
	var said []string
	for _, in := range []string{"2", "Las Vegas", ""} {
		for _, o := range g.Handle(proto.LineEvent{Text: in}) {
			if s, ok := o.(proto.Say); ok {
				said = append(said, s.Lines...)
			}
		}
	}
	want := g.war.sensed(2)
	if g.phase != orders || len(g.phantom) == 0 || want.missiles() == 0 || g.defcon != 3 {
		t.Fatalf("no false alarm at the strike 2 prompt: phase %d, %d tracks, DEFCON %d", g.phase, len(g.phantom), g.defcon)
	}
	if last := said[len(said)-1]; last != "ENEMY LAUNCH DETECTED. "+g.list(want)+". DEFCON 3." {
		t.Errorf("the false alarm reports %q", last)
	}
	if got := g.shown()[wopr].ICBM; got != g.war.f[wopr].ICBM-want.missiles() {
		t.Errorf("the forces table shows WOPR with %d ICBMs, want %d", got, g.war.f[wopr].ICBM-want.missiles())
	}
	for _, m := range g.phantom {
		if m.ours || m.kind != kindICBM || g.progress(m) <= 0 || g.progress(m) >= 1 {
			t.Errorf("a false alarm's track is %+v, %.2f of the way", m, g.progress(m))
		}
	}
	for _, in := range []string{"help", "maybe"} { // asked again, with the tracks where they were
		g.Handle(proto.LineEvent{Text: in})
		if g.phase != orders || len(g.phantom) == 0 {
			t.Errorf("%q at the false alarm: phase %d, %d tracks", in, g.phase, len(g.phantom))
		}
	}
	g.Handle(proto.LineEvent{Text: "hold"})
	if g.phantom != nil || g.defcon != 4 || g.shown()[wopr].ICBM != g.war.f[wopr].ICBM {
		t.Errorf("held, the false alarm stays: %d tracks, DEFCON %d", len(g.phantom), g.defcon)
	}
}

// Movie mode's war has no false alarm, whatever its seed: its scenes are the film's.
func TestFilmHasNoFalseAlarm(t *testing.T) {
	t.Parallel()
	alarms := 0
	for seed := range uint64(60) {
		if falseAlarm(seed) != 0 {
			alarms++
		}
		f := NewFilm(seed, false)
		if f.g.alarm != 0 {
			t.Fatalf("seed %d: the film's war has a false alarm before strike %d", seed, f.g.alarm)
		}
		if lines := f.Run(FilmClimax); strings.Contains(strings.Join(lines, "\n"), "SENSOR ERROR") {
			t.Fatalf("seed %d: the film says %q", seed, lines)
		}
	}
	if alarms == 0 {
		t.Fatal("no seed has a false alarm: the test proves nothing")
	}
}
