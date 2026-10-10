package main

import (
	"slices"
	"strings"
)

// page builds a manual page in man(7) macros. Text goes through escape and, in body text,
// one sentence to a line, as groff and mandoc want ("new sentence, new line").
//
// In the rendered page, words and sentences are one space apart. groff stretches the spaces
// in a line to reach the right margin (leftAlign stops that), and groff and mandoc both put
// two spaces after a sentence that ends an input line (oneSpace stops that).
type page struct{ b strings.Builder }

func (p *page) String() string { return p.b.String() }

// line writes one raw input line.
func (p *page) line(s string) {
	p.b.WriteString(s)
	p.b.WriteByte('\n')
}

// leftAlign keeps the right margin ragged, so that groff does not widen the spaces in a line
// to fill it, and goes right after .TH. Groff 1.22 keeps the .ad l request to the end; groff
// 1.23 sets the adjustment again at .TH and at each paragraph's tag, from the AD string,
// which .ds sets. mandoc does not stretch lines, and neither request changes what it shows.
func (p *page) leftAlign() {
	p.line(".ad l")
	p.line(".ds AD l")
}

// macro writes a request such as .SH or .TP, with its arguments escaped and quoted where
// they hold a space. The last argument ends the line, so it goes through oneSpace.
func (p *page) macro(name string, args ...string) {
	var b strings.Builder
	b.WriteString("." + name)
	for i, a := range args {
		a = escape(a)
		if i == len(args)-1 {
			a = oneSpace(a)
		}
		if a == "" || strings.ContainsAny(a, " \t") {
			a = `"` + strings.ReplaceAll(a, `"`, `""`) + `"`
		}
		b.WriteString(" " + a)
	}
	p.line(b.String())
}

// lineWidth is the longest input line, as mandoc's lint asks.
const lineWidth = 80

// text writes body text, each sentence on lines of its own no longer than lineWidth,
// counting the \& that oneSpace adds.
func (p *page) text(s string) {
	for _, sentence := range sentences(s) {
		lines := wrap(escape(sentence), lineWidth)
		if slices.ContainsFunc(lines, func(l string) bool { return len(oneSpace(l)) > lineWidth }) {
			lines = wrap(escape(sentence), lineWidth-len(`\&`))
		}
		for _, l := range lines {
			p.line(textLine(oneSpace(l)))
		}
	}
}

// wrap breaks s at spaces into lines of at most width bytes; a longer word keeps a line
// to itself.
func wrap(s string, width int) []string {
	var out []string
	line := ""
	for _, w := range strings.Fields(s) {
		switch {
		case line == "":
			line = w
		case len(line)+1+len(w) <= width:
			line += " " + w
		default:
			out = append(out, line)
			line = w
		}
	}
	if line != "" {
		out = append(out, line)
	}
	return out
}

// bullets writes each item as an indented paragraph led by a bullet.
func (p *page) bullets(items []string) {
	for _, it := range items {
		p.line(`.IP \(bu 2`)
		p.text(it)
	}
}

// escape makes s safe as roff text: a backslash is the escape character, and a hyphen
// becomes \- so that options and commands copied from the page still work.
func escape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\e`)
	return strings.ReplaceAll(s, "-", `\-`)
}

// oneSpace ends an input line that ends a sentence with \&, a character of no width, so that
// groff and mandoc put one space after the sentence, not two. A sentence ends in a full stop,
// a question mark or an exclamation mark, which closing quotes and brackets may follow.
func oneSpace(s string) string {
	end := strings.TrimRight(s, `"')]`)
	if end != "" && strings.IndexByte(".?!", end[len(end)-1]) >= 0 {
		return s + `\&`
	}
	return s
}

// textLine keeps a text line from being read as a request: a line that starts with a
// period or an apostrophe is a control line in roff.
func textLine(s string) string {
	if strings.HasPrefix(s, ".") || strings.HasPrefix(s, "'") {
		return `\&` + s
	}
	return s
}

// sentences splits s after each sentence: a full stop, question mark or exclamation mark
// followed by a space and a capital letter, a digit or the program's name.
func sentences(s string) []string {
	var out []string
	start := 0
	for i := 0; i+2 < len(s); i++ {
		if strings.IndexByte(".?!", s[i]) >= 0 && s[i+1] == ' ' && startsSentence(s[i+2:]) {
			out = append(out, strings.TrimSpace(s[start:i+1]))
			start = i + 2
		}
	}
	if rest := strings.TrimSpace(s[start:]); rest != "" {
		out = append(out, rest)
	}
	return out
}

func startsSentence(s string) bool {
	c := s[0]
	return c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.HasPrefix(s, "wopr ")
}
