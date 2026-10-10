package main

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// page builds a manual page in man(7) macros. Text goes through escape and, in body text,
// one sentence to a line, as groff and mandoc want ("new sentence, new line").
//
// In the rendered page, words and sentences are one space apart. groff stretches the spaces
// in a line to reach the right margin (leftAlign stops that), and groff and mandoc both put
// two spaces after a sentence that ends an input line (oneSpace stops that).
//
// No name is split across lines: every word that names something, an option, a variable, a
// path, a URL, a game or a command, starts with \%, which stops groff hyphenating it (see
// literal). Other words are hyphenated as the reader's groff is set to. mandoc never
// hyphenates.
type page struct {
	b strings.Builder
	// names are the lower-case words the page uses as names: the program's, its themes',
	// games' and scenes'. Words that do not look like plain words are names already.
	names map[string]bool
}

func (p *page) String() string { return p.b.String() }

// line writes one raw input line.
func (p *page) line(s string) {
	p.b.WriteString(s)
	p.b.WriteByte('\n')
}

// leftAlign keeps the right margin ragged, so that groff does not widen the spaces in a line
// to fill it, and goes right after .TH. Groff 1.22 keeps the .ad l request to the end; groff
// 1.23 sets the adjustment again at .TH and at each paragraph's tag, and groff 1.24 at each
// paragraph, from the AD string, which .ds sets. mandoc does not stretch lines, and neither
// request changes what it shows.
func (p *page) leftAlign() {
	p.line(".ad l")
	p.line(".ds AD l")
}

// fontMacros set their arguments in alternating fonts as text, so their words can be
// hyphenated as body text can.
var fontMacros = []string{"B", "I", "BR", "RB", "IR", "RI", "BI", "IB"}

// macro writes a request such as .SH or .TP, with its arguments escaped and quoted where
// they hold a space. The last argument ends the line, so it goes through oneSpace. A font
// macro's names are marked as body text's are.
func (p *page) macro(name string, args ...string) {
	if slices.Contains(fontMacros, name) {
		args = p.fontArgs(len(name) == 2, args)
	} else {
		args = slices.Clone(args)
		for i, a := range args {
			args[i] = escape(a)
		}
	}
	var b strings.Builder
	b.WriteString("." + name)
	for i, a := range args {
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

// fontArgs escapes a font macro's arguments and marks their names, as words does. The
// arguments of an alternating macro, such as .BR, run together with no space between them,
// so one word can start in one argument and end in the next: \% goes only at the start of a
// whole word, since inside a word it marks a place where groff may hyphenate it.
func (p *page) fontArgs(alternating bool, args []string) []string {
	parts := make([][]string, len(args)) // each argument's words, or the parts of words in it
	type at struct{ arg, i int }
	var word []at // the parts of the word being read
	end := func() {
		var raw strings.Builder
		for _, w := range word {
			raw.WriteString(parts[w.arg][w.i])
		}
		name := p.literal(raw.String(), false)
		for k, w := range word {
			parts[w.arg][w.i] = escape(parts[w.arg][w.i])
			if k == 0 && name {
				parts[w.arg][w.i] = `\%` + parts[w.arg][w.i]
			}
		}
		word = nil
	}
	for a, arg := range args {
		if !alternating {
			end() // .B and .I put a space between their arguments
		}
		for i, w := range strings.Split(arg, " ") {
			if i > 0 {
				end()
			}
			parts[a] = append(parts[a], w)
			word = append(word, at{a, i})
		}
	}
	end()
	out := make([]string, len(args))
	for a := range parts {
		out[a] = strings.Join(parts[a], " ")
	}
	return out
}

// lineWidth is the longest input line, as mandoc's lint asks.
const lineWidth = 80

// text writes body text written here, each sentence on lines of its own no longer than
// lineWidth, counting the \& that oneSpace adds. It is marked up: `wopr -m` is in bold, for
// what is typed as it stands, and <name> in italics, for what stands for something else.
func (p *page) text(s string) { p.write(s, true) }

// plain writes body text from elsewhere, the games' pages, as text does but with no markup.
func (p *page) plain(s string) { p.write(s, false) }

func (p *page) write(s string, markup bool) {
	for _, sentence := range sentences(s) {
		words := p.words(sentence, markup)
		lines := wrap(words, lineWidth)
		if slices.ContainsFunc(lines, func(l string) bool { return len(oneSpace(l)) > lineWidth }) {
			lines = wrap(words, lineWidth-len(`\&`))
		}
		for _, l := range lines {
			p.line(textLine(oneSpace(l)))
		}
	}
}

// phrase is a few words of marked-up text, such as a tag line, as one input line.
func (p *page) phrase(s string) string { return strings.Join(p.words(s, true), " ") }

// wrap joins words into lines of at most width bytes; a longer word keeps a line to itself.
func wrap(words []string, width int) []string {
	var out []string
	line := ""
	for _, w := range words {
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

// bullets writes each item, from a game's page, as an indented paragraph led by a bullet.
func (p *page) bullets(items []string) {
	for _, it := range items {
		p.line(`.IP \(bu 2`)
		p.plain(it)
	}
}

// span is a run of text in one font: 'R' (roman), 'B' (bold) or 'I' (italic).
type span struct {
	font byte
	text string
}

// words splits s at spaces into words in roff, escaped and with their names marked. With
// markup, `...` is bold and <...> italic, and either may hold several words.
func (p *page) words(s string, markup bool) []string {
	var out []string
	var word []span
	font := byte('R')
	add := func(c byte) {
		if n := len(word); n > 0 && word[n-1].font == font {
			word[n-1].text += string(c)
		} else {
			word = append(word, span{font, string(c)})
		}
	}
	for i := range len(s) {
		c := s[i]
		switch {
		case c == ' ':
			if len(word) > 0 {
				out = append(out, p.word(word))
				word = nil
			}
		case markup && c == '`' && font == 'R':
			font = 'B'
		case markup && c == '`' && font == 'B':
			font = 'R'
		case markup && c == '<' && font == 'R':
			font = 'I'
		case markup && c == '>' && font == 'I':
			font = 'R'
		default:
			add(c)
		}
	}
	if font != 'R' {
		panic(fmt.Sprintf("manpage: unclosed markup in %q", s)) // a mistake in this program's text
	}
	if len(word) > 0 {
		out = append(out, p.word(word))
	}
	return out
}

// word is one word in roff: each span escaped and in its font, and the whole led by \% when
// it is a name (see literal).
func (p *page) word(spans []span) string {
	var b, raw strings.Builder
	styled := false
	for _, s := range spans {
		raw.WriteString(s.text)
		if s.font == 'R' {
			b.WriteString(escape(s.text))
			continue
		}
		styled = true
		b.WriteString(`\f` + string(s.font) + escape(s.text) + `\fR`)
	}
	if p.literal(raw.String(), styled) {
		return `\%` + b.String()
	}
	return b.String()
}

// plainWord is a word of prose: letters, in lower case after an optional capital, and
// perhaps an apostrophe's ending, as in film's.
var plainWord = regexp.MustCompile(`^[A-Z]?[a-z]*(?:'[a-z]+)?$`)

// literal reports whether a word is a name that must never be split, so that what the reader
// types from the page still works: a word in bold or italics, anything that is not a plain
// word (an option, a variable, a path, a URL, a word in capitals), or a word the page uses as
// a name (wopr, a theme, a game, a scene). Its letters are what can be hyphenated, so a word
// with none is left alone.
//
// \% at the start of a word stops groff hyphenating it, in every version; mandoc reads it
// too. Turning hyphenation off for the whole page would not last: groff 1.24 turns it on
// again at each paragraph, from the HY register, which groff 1.23 reads at .TH and groff 1.22
// once, at start-up. It would also take the reader's choice away for plain words.
func (p *page) literal(w string, styled bool) bool {
	core := strings.Trim(w, `"'()[]{},.;:!?`)
	if !strings.ContainsFunc(core, func(r rune) bool { return 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' }) {
		return false
	}
	return styled || !plainWord.MatchString(core) || p.names[strings.ToLower(strings.TrimSuffix(core, "'s"))]
}

// roffText makes text safe as roff. A backslash is the escape character. A hyphen becomes
// \-, so that options and commands copied from the page still work. A tilde, a caret and a
// grave accent become the characters that print as themselves: groff 1.23 and later print
// the plain ones as a small tilde, a modifier circumflex and an opening quote, which a shell
// does not read as ~, ^ and ` (Debian's man.local maps them back; groff's own does not).
var roffText = strings.NewReplacer(`\`, `\e`, "-", `\-`, "~", `\(ti`, "^", `\(ha`, "`", `\(ga`)

func escape(s string) string { return roffText.Replace(s) }

// sentenceTail is what may follow a sentence's last mark on its line.
var sentenceTail = regexp.MustCompile(`(?:\\f[RBIP]|["')\]])+$`)

// oneSpace ends an input line that ends a sentence with \&, a character of no width, so that
// groff and mandoc put one space after the sentence, not two. A sentence ends in a full stop,
// a question mark or an exclamation mark, which closing quotes and brackets may follow, and a
// change of font, which groff looks past (mandoc does not, and \& does no harm there).
func oneSpace(s string) string {
	end := sentenceTail.ReplaceAllString(s, "")
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
// followed by a space and a capital letter, a digit or the program's name, perhaps marked
// up.
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
	s = strings.TrimLeft(s, "`<")
	if s == "" {
		return false
	}
	c := s[0]
	return c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.HasPrefix(s, "wopr ") || strings.HasPrefix(s, "wopr`")
}
