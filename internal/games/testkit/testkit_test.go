package testkit

import (
	"testing"

	"github.com/GhostofGoes/WOPR/internal/games/gamestest"
	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/proto/host"
)

// A typed line (Say.Open) is one transcript line, as the console shows it. (The stub has a
// panel, so it starts on a new page.)
func TestTypedLinesJoin(t *testing.T) {
	t.Parallel()
	g := Game(t, &gamestest.Stub{}, gamestest.StubInfo(), "", 1)
	g.Type("TYPE")
	if want := "[CLEAR]\nSTUB READY.\nMOVE: TYPE\n\nTYPED: Hello.\n\n"; g.Transcript() != want {
		t.Errorf("transcript %q, want %q", g.Transcript(), want)
	}
}

// drainer prints A, asks for Drained, prints B; Drained prints C. The console reveals A and B
// before it reaches the marker, so C comes last.
type drainer struct{}

func (drainer) Start(proto.Env) []proto.Output {
	return []proto.Output{proto.Say{Lines: []string{"A"}}, proto.Drain{}, proto.Say{Lines: []string{"B"}}}
}

func (drainer) Handle(ev proto.Event) []proto.Output {
	if _, ok := ev.(proto.Drained); ok {
		return []proto.Output{proto.Say{Lines: []string{"C"}}, proto.Hold{On: true}}
	}
	return nil
}

func (drainer) View(*proto.Canvas) {}

func TestDrainedAfterItsBatch(t *testing.T) {
	t.Parallel()
	s := Start(t, drainer{}, host.Placement{}, host.Config{})
	if s.Transcript() != "A\nB\nC\n" || !s.Held() {
		t.Errorf("transcript %q, held %v", s.Transcript(), s.Held())
	}
}
