package games

import (
	"strings"
	"testing"

	"github.com/GhostofGoes/WOPR/internal/proto"
)

func TestNewRegistryValidates(t *testing.T) {
	t.Parallel()
	ok := Entry{Info: Info{Number: 1, Listed: true, Name: "CHESS", Slug: "chess"}}
	bad := []struct {
		name    string
		entries []Entry
		want    string
	}{
		{"duplicate slug", []Entry{ok, {Info: Info{Number: 2, Listed: true, Name: "CHESS TWO", Slug: "chess"}}}, "collides"},
		{"alias collides", []Entry{ok, {Info: Info{Number: 2, Listed: true, Name: "POKER", Slug: "poker", Aliases: []string{"Chess"}}}}, "collides"},
		{"gap in numbers", []Entry{ok, {Info: Info{Number: 3, Listed: true, Name: "POKER", Slug: "poker"}}}, "1..2"},
		{"unlisted with number", []Entry{ok, {Info: Info{Number: 2, Name: "POKER", Slug: "poker"}}}, "Number 0"},
		{"listed without number", []Entry{{Info: Info{Listed: true, Name: "POKER", Slug: "poker"}}}, "Number from 1"},
		{"playable without constructor", []Entry{{Info: Info{Number: 1, Listed: true, Name: "POKER", Slug: "poker", Status: Playable}}}, "constructor"},
		{"lower-case name", []Entry{{Info: Info{Number: 1, Listed: true, Name: "Poker", Slug: "poker"}}}, "upper case"},
		{"bad slug", []Entry{{Info: Info{Number: 1, Listed: true, Name: "POKER", Slug: "Poker Game"}}}, "invalid slug"},
		{"numeric alias", []Entry{{Info: Info{Number: 1, Listed: true, Name: "BLACK JACK", Slug: "black-jack", Aliases: []string{"21"}}}}, "is a number"},
		{"panel without rows", []Entry{{Info: Info{Number: 1, Listed: true, Name: "POKER", Slug: "poker", Layout: proto.LayoutPanel}}}, "PanelRows"},
	}
	for _, tt := range bad {
		_, err := NewRegistry(tt.entries...)
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: got %v, want an error containing %q", tt.name, err, tt.want)
		}
	}
	if _, err := NewRegistry(ok); err != nil {
		t.Fatalf("valid registry rejected: %v", err)
	}
}

// An internal program (the ending) is found only by Get: players can never choose it.
func TestInternalEntries(t *testing.T) {
	t.Parallel()
	r, err := NewRegistry(
		Entry{Info: Info{Number: 1, Listed: true, Name: "CHESS", Slug: "chess", Status: Playable}, New: func() Game { return nil }},
		Entry{Info: Info{Name: "ENDING", Slug: "ending", Internal: true, Status: Playable}, New: func() Game { return nil }},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := r.Get("ending"); !ok {
		t.Error("Get must find an internal entry")
	}
	if _, err := r.Resolve("ending"); err == nil {
		t.Error("Resolve must not find an internal entry")
	}
	if _, err := r.Resolve("end"); err == nil {
		t.Error("nor by prefix")
	}
	if _, ok := r.Exact("ENDING"); ok {
		t.Error("Exact must not find an internal entry")
	}
	if len(r.All()) != 1 || len(r.Listed()) != 1 {
		t.Error("All and Listed skip internal entries")
	}
	if _, err := NewRegistry(Entry{Info: Info{Number: 1, Listed: true, Name: "X", Slug: "x", Internal: true}}); err == nil {
		t.Error("an internal entry cannot be listed")
	}
}
