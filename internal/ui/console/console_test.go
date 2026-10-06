package console

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/GhostofGoes/WOPR/internal/proto"
)

func TestSanitizeInput(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"Joshua":                       "Joshua",
		"a\tb":                         "a b",
		"line one\nline two\r\n":       "line one line two ",
		"\x1b]52;c;ZXZpbA==\x07Joshua": "Joshua",
		"\x1b[31mred\x1b[0m":           "red",
		"bell\a del\x7f c1\u0085":      "bell del c1",
		"bad\xffutf8":                  "bad�utf8",
	}
	for in, want := range cases {
		if got := SanitizeInput(in); got != want {
			t.Errorf("SanitizeInput(%q) = %q, want %q", in, got, want)
		}
	}
	long := strings.Repeat("👨‍👩‍👧", 300)
	if n := len(Graphemes(SanitizeInput(long))); n != MaxInput {
		t.Errorf("cap: %d graphemes, want %d", n, MaxInput)
	}
}

func TestSanitizeText(t *testing.T) {
	t.Parallel()
	got := SanitizeText("A\tB\r\nC\x1b[2JD\x00")
	if len(got) != 2 || got[0] != "A       B" || got[1] != "CD" {
		t.Errorf("SanitizeText = %q", got)
	}
}

func TestWrap(t *testing.T) {
	t.Parallel()
	m := ansi.WcWidth
	got := Wrap("EXCELLENT. IT'S BEEN A LONG TIME. CAN YOU EXPLAIN THE REMOVAL", 30, m)
	for _, r := range got {
		if m.StringWidth(r) > 30 {
			t.Errorf("row too wide: %q", r)
		}
	}
	if strings.Join(got, " ") != "EXCELLENT. IT'S BEEN A LONG TIME. CAN YOU EXPLAIN THE REMOVAL" {
		t.Errorf("words lost: %q", got)
	}
	if got := Wrap(strings.Repeat("X", 25), 10, m); len(got) != 3 || got[2] != "XXXXX" {
		t.Errorf("long word: %q", got)
	}
	if got := Wrap("", 10, m); len(got) != 1 || got[0] != "" {
		t.Errorf("empty line: %q", got)
	}
}

func text(rows []Row) string {
	var b strings.Builder
	for _, r := range rows {
		b.WriteString(r.Text)
		b.WriteByte('\n')
	}
	return b.String()
}

func TestScrollbackPagesAndExactHeight(t *testing.T) {
	t.Parallel()
	var sb Scrollback
	sb.Append("LOGON: 000001", 0)
	sb.Append("IDENTIFICATION NOT RECOGNIZED BY SYSTEM", 0)
	rows := sb.Render(80, 5, ansi.WcWidth, Row{Text: "LOGON: "})
	if len(rows) != 5 || rows[0].Text != "LOGON: 000001" || rows[2].Text != "LOGON: " || rows[4].Text != "" {
		t.Fatalf("fresh page should start at the top:\n%s", text(rows))
	}
	sb.PageBreak()
	sb.Append("GREETINGS PROFESSOR FALKEN.", 0)
	rows = sb.Render(80, 3, ansi.WcWidth)
	if rows[0].Text != "GREETINGS PROFESSOR FALKEN." {
		t.Fatalf("page break must restart at the top:\n%s", text(rows))
	}
	for i := range 10 {
		sb.Append(strings.Repeat("X", i+1), 0)
	}
	rows = sb.Render(80, 3, ansi.WcWidth)
	if rows[2].Text != "XXXXXXXXXX" {
		t.Fatalf("a long page must follow the newest line:\n%s", text(rows))
	}
	sb.Scroll(100)
	rows = sb.Render(80, 3, ansi.WcWidth)
	if rows[0].Text != "LOGON: 000001" {
		t.Fatalf("scrolling up must reach earlier pages:\n%s", text(rows))
	}
}

func TestTypewriterSequencing(t *testing.T) {
	t.Parallel()
	var sb Scrollback
	var tw Typewriter
	tw.Say([]string{"GREETINGS PROFESSOR FALKEN."}, 0, proto.PaceSpeech) // 27 graphemes at 30/s
	tw.Pause(time.Second)
	tw.Page()
	tw.Prompt("")
	ev := tw.Advance(500*time.Millisecond, &sb)
	if len(ev) != 0 || sb.Lines()[0].Visible() != "GREETINGS PROFE" { // 15 of 27
		t.Fatalf("after 0.5s: visible %q, events %v", sb.Lines()[0].Visible(), ev)
	}
	if !tw.Revealing() {
		t.Error("should be revealing")
	}
	tw.Advance(500*time.Millisecond, &sb) // line done, pause starts
	if sb.Lines()[0].Visible() != "GREETINGS PROFESSOR FALKEN." || !tw.Busy() {
		t.Fatal("line should be complete and the pause pending")
	}
	ev = tw.Advance(2*time.Second, &sb)
	if len(ev) != 1 || !ev[0].Prompt || tw.Busy() {
		t.Fatalf("pause, page and prompt should all complete: %v busy=%v", ev, tw.Busy())
	}
}

func TestTypewriterFlushAndInstant(t *testing.T) {
	t.Parallel()
	var sb Scrollback
	var tw Typewriter
	tw.Say([]string{"A", "", "B"}, 0, proto.PaceSpeech)
	tw.Pause(time.Hour)
	tw.Prompt("LOGON: ")
	ev := tw.Flush(&sb)
	if len(ev) != 1 || ev[0].PromptText != "LOGON: " || len(sb.Lines()) != 3 || tw.Busy() {
		t.Fatalf("flush: events %v lines %d busy %v", ev, len(sb.Lines()), tw.Busy())
	}
	tw.SetInstant(true)
	tw.Say([]string{"C"}, 0, proto.PaceSpeech)
	tw.Advance(0, &sb)
	if sb.Lines()[3].Visible() != "C" {
		t.Error("instant mode must reveal at once")
	}
}

func TestEditor(t *testing.T) {
	t.Parallel()
	var e Editor
	e.Insert("Joshua")
	e.Backspace()
	e.Insert("a")
	if got := e.Submit(); got != "Joshua" || !e.Empty() {
		t.Fatalf("submit = %q", got)
	}
	e.Insert("help games")
	e.Submit()
	e.Insert("draft")
	e.HistoryPrev()
	if e.Value() != "help games" {
		t.Fatalf("prev = %q", e.Value())
	}
	e.HistoryPrev()
	if e.Value() != "Joshua" {
		t.Fatalf("prev2 = %q", e.Value())
	}
	e.HistoryNext()
	e.HistoryNext()
	if e.Value() != "draft" {
		t.Fatalf("back to draft = %q", e.Value())
	}
	e.Clear()
	e.Insert("👨‍👩‍👧x")
	e.Backspace()
	e.Backspace()
	if !e.Empty() {
		t.Errorf("backspace must remove whole graphemes, left %q", e.Value())
	}
}

func FuzzSanitizeInput(f *testing.F) {
	for _, s := range []string{"Joshua", "\x1b]8;;http://x\x1b\\link", "\t\n\r", "\xff\xfe", "👨‍👩‍👧"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		out := SanitizeInput(s)
		for _, r := range out {
			if r < 0x20 || r == 0x7f || (r >= 0x80 && r < 0xa0) {
				t.Fatalf("control %U survived in %q", r, out)
			}
		}
		if len(Graphemes(out)) > MaxInput {
			t.Fatal("cap exceeded")
		}
		for _, row := range Wrap(out, 20, ansi.WcWidth) {
			if ansi.WcWidth.StringWidth(row) > 20 {
				t.Fatalf("row wider than 20: %q", row)
			}
		}
	})
}
