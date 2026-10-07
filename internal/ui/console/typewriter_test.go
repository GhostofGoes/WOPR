package console

import (
	"testing"
	"time"

	"github.com/GhostofGoes/WOPR/internal/proto"
)

// An open line is continued by the next line queued, keeping what is already shown; a page
// break, or a line added from outside (the console's echo of a submitted line), closes it.
func TestTypewriterOpenLines(t *testing.T) {
	t.Parallel()
	var sb Scrollback
	var tw Typewriter
	tw.SayOpen([]string{"LOGON: "}, 0, proto.PaceInstant)
	tw.Pause(time.Second)
	tw.SayOpen([]string{"Jo"}, 0, proto.PaceTyping)
	tw.Say([]string{"shua"}, 0, proto.PaceTyping)
	tw.Advance(500*time.Millisecond, &sb)
	if len(sb.Lines()) != 1 || sb.Lines()[0].Visible() != "LOGON: " {
		t.Fatalf("the prompt shows during the pause: %q", sb.Lines()[0].Visible())
	}
	tw.Advance(630*time.Millisecond, &sb) // the rest of the pause, then one keystroke at 8/s
	if got := sb.Lines()[0].Visible(); got != "LOGON: J" {
		t.Fatalf("one keystroke in: %q", got)
	}
	tw.Flush(&sb)
	if len(sb.Lines()) != 1 || sb.Lines()[0].Text != "LOGON: Joshua" {
		t.Fatalf("the typed line is one row: %+v", sb.Lines())
	}
	tw.Say([]string{"NEXT"}, 0, proto.PaceInstant) // the closed line is not continued
	tw.SayOpen([]string{"A"}, 0, proto.PaceInstant)
	tw.Page()
	tw.Say([]string{"B"}, 0, proto.PaceInstant)
	tw.SayOpen([]string{"C"}, 0, proto.PaceInstant)
	tw.Flush(&sb)
	sb.Append("echo", 0)
	tw.Say([]string{"D"}, 0, proto.PaceInstant)
	tw.Flush(&sb)
	var got []string
	for _, l := range sb.Lines() {
		got = append(got, l.Text)
	}
	if want := []string{"LOGON: Joshua", "NEXT", "A", "B", "C", "echo", "D"}; !equal(got, want) {
		t.Errorf("lines %q, want %q", got, want)
	}
}

// A page break made on the scrollback directly (the UI does so when the typewriter is idle and
// a game with its own screen launches) closes an open line too: the next line starts its own
// row on the new page instead of continuing the old page's last line.
func TestOpenLineEndsAtADirectPageBreak(t *testing.T) {
	t.Parallel()
	var sb Scrollback
	var tw Typewriter
	tw.SayOpen([]string{"OPEN LINE "}, 0, proto.PaceInstant)
	tw.Flush(&sb)
	if tw.Busy() {
		t.Fatal("the typewriter is idle")
	}
	sb.PageBreak()
	tw.Say([]string{"STUB READY."}, 0, proto.PaceInstant)
	tw.Flush(&sb)
	var got []string
	for _, l := range sb.Lines() {
		got = append(got, l.Text)
	}
	if want := []string{"OPEN LINE ", "STUB READY."}; !equal(got, want) {
		t.Fatalf("lines %q, want %q", got, want)
	}
	if rows := sb.Render(80, 3, 0); rows[0].Text != "STUB READY." {
		t.Errorf("the new page starts with the new line: %+v", rows)
	}
}

// A drain marker is reported once everything before it is out, and does not count as output
// being revealed.
func TestTypewriterDrain(t *testing.T) {
	t.Parallel()
	var sb Scrollback
	var tw Typewriter
	tw.Say([]string{"ABC"}, 0, proto.PaceSpeech)
	tw.Pause(time.Second)
	tw.Drain(7)
	if ev := tw.Advance(200*time.Millisecond, &sb); len(ev) != 0 {
		t.Fatalf("too early: %v", ev)
	}
	ev := tw.Advance(time.Second, &sb)
	if len(ev) != 1 || ev[0].Drain != 7 || tw.Busy() || tw.Revealing() {
		t.Fatalf("after the line and the pause: %v busy %v", ev, tw.Busy())
	}
	tw.Drain(8)
	if tw.Revealing() {
		t.Error("a pending marker is not output to skip")
	}
	if ev := tw.Flush(&sb); len(ev) != 1 || ev[0].Drain != 8 {
		t.Errorf("a flush reaches the marker: %v", ev)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
