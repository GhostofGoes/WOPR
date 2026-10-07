package proto

import (
	"strings"
	"testing"
)

func TestCanvasPutClipsAndRenders(t *testing.T) {
	t.Parallel()
	c := NewCanvas(10, 2)
	end := c.Put(7, 0, "DEFCON", StyleDefcon3, AttrReverse)
	if end != 13 {
		t.Errorf("Put returned %d, want 13", end)
	}
	c.Put(-2, 1, "*SEATTLE", StyleIncoming, 0)
	if got, want := c.String(), "       DEF\nEATTLE\n"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
	if got, want := c.StyleMap(), ".......333\niiiiii\n"; got != want {
		t.Errorf("StyleMap() = %q, want %q", got, want)
	}
}

func TestStyleLettersCoverEveryStyle(t *testing.T) {
	t.Parallel()
	if len(styleLetters) != int(numStyles) {
		t.Fatalf("styleLetters has %d letters for %d styles", len(styleLetters), numStyles)
	}
	if got := Styles(); len(got) != int(numStyles) || got[0] != StyleText || got[len(got)-1] != numStyles-1 {
		t.Fatalf("Styles() = %v, want every style in order", got)
	}
	seen := map[byte]bool{}
	for s := range numStyles {
		l := s.Letter()
		if seen[l] || l == '.' {
			t.Errorf("style %d letter %q is not unique", s, l)
		}
		seen[l] = true
	}
}

func TestNewRandIsStable(t *testing.T) {
	t.Parallel()
	a, b := NewRand(1, DomainBrain|7), NewRand(1, DomainBrain|7)
	for range 5 {
		if a.Uint64() != b.Uint64() {
			t.Fatal("same seed and stream must give the same sequence")
		}
	}
	// Go guarantees seeded math/rand/v2 sequences across releases; pin one value so a
	// change in our derivation is caught.
	if got, want := NewRand(1, 2).Uint64(), uint64(0xc4f5a58656eef510); got != want {
		t.Errorf("NewRand(1, 2).Uint64() = %#x, want %#x", got, want)
	}
	if GameStream("chess", 0) == GameStream("checkers", 0) || GameStream("chess", 0) == GameStream("chess", 1) {
		t.Error("game streams must differ by slug and play index")
	}
	if GameStream("chess", 0)&(0xff<<56) != DomainGame {
		t.Error("game stream must sit in the game domain")
	}
}

func TestPaceSpeeds(t *testing.T) {
	t.Parallel()
	if PaceInstant.CharsPerSecond() != 0 || PaceSpeech.CharsPerSecond() <= PaceTyping.CharsPerSecond() {
		t.Error("unexpected pace speeds")
	}
	if !strings.Contains(KeyBackspace.String(), "back") {
		t.Error("key names")
	}
}
