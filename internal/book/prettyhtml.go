package book

import (
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// prettyHTML formats HTML to read, as a browser's Elements panel or
// prettier shows it:
//
//   - an element on a line of its own, its children a level (two spaces) in;
//   - an element holding only text and inline elements (a span, a b, a
//     kbd) on one line, if it fits htmlWidth: inline formatting stays as
//     it reads;
//   - an open tag that doesn't fit with one attribute a line, and its `>`
//     on a line of its own;
//   - text on its own line, its white space collapsed, wrapped at words;
//   - an svg on one line, with its id and class only (a lucide icon's class
//     names it) and `…` for the rest: an icon's paths and presentation
//     attributes say nothing.
func prettyHTML(src string) string {
	kids, err := html.ParseFragment(strings.NewReader(src), bodyCtx)
	if err != nil {
		return src
	}
	var b strings.Builder
	for _, k := range kids {
		block(&b, k, 0)
	}
	return b.String()
}

const htmlWidth = 80

var bodyCtx = &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}

var void = map[string]bool{"area": true, "base": true, "br": true, "col": true, "embed": true, "hr": true, "img": true,
	"input": true, "link": true, "meta": true, "source": true, "track": true, "wbr": true}

// inline are the elements that flow in text: an element holding only
// these and text may stay on one line.
var inline = map[string]bool{"a": true, "abbr": true, "b": true, "bdi": true, "bdo": true, "br": true, "cite": true,
	"code": true, "data": true, "dfn": true, "em": true, "i": true, "kbd": true, "mark": true, "q": true, "s": true,
	"samp": true, "small": true, "span": true, "strong": true, "sub": true, "sup": true, "time": true, "u": true,
	"var": true, "wbr": true, "svg": true, "img": true}

// block writes n at depth, on lines of its own.
func block(b *strings.Builder, n *html.Node, depth int) {
	pad := strings.Repeat("  ", depth)
	switch n.Type {
	case html.TextNode:
		if t := collapse(n.Data); strings.TrimSpace(t) != "" {
			wrapText(b, strings.TrimSpace(t), pad)
		}
		return
	case html.ElementNode:
	default:
		return
	}
	if line, ok := flat(n, true); ok && (len(pad)+len(line) <= htmlWidth || n.Data == "svg") {
		b.WriteString(pad + line + "\n")
		return
	}
	openTag(b, n, pad)
	if void[n.Data] {
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		block(b, c, depth+1)
	}
	b.WriteString(pad + "</" + n.Data + ">\n")
}

// flat is n on one line, and whether it may be: its content is text and
// inline elements only (top: n itself may be any element).
func flat(n *html.Node, top bool) (string, bool) {
	switch n.Type {
	case html.TextNode:
		return html.EscapeString(collapse(n.Data)), true
	case html.ElementNode:
	default:
		return "", true
	}
	if !top && !inline[n.Data] {
		return "", false
	}
	if n.Data == "svg" {
		var keep []html.Attribute
		for _, a := range n.Attr {
			if a.Key == "id" || a.Key == "class" {
				keep = append(keep, a)
			}
		}
		return "<svg" + attrs(keep, " ") + " …></svg>", true
	}
	open := "<" + n.Data + attrs(n.Attr, " ") + ">"
	if void[n.Data] {
		return open, true
	}
	var in strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		s, ok := flat(c, false)
		if !ok {
			return "", false
		}
		in.WriteString(s)
	}
	t := in.String()
	if top {
		t = strings.TrimSpace(t)
	}
	return open + t + "</" + n.Data + ">", true
}

// openTag writes n's open tag: on one line if it fits, else one attribute
// a line.
func openTag(b *strings.Builder, n *html.Node, pad string) {
	one := "<" + n.Data + attrs(n.Attr, " ") + ">"
	if len(pad)+len(one) <= htmlWidth || len(n.Attr) < 2 {
		b.WriteString(pad + one + "\n")
		return
	}
	b.WriteString(pad + "<" + n.Data + attrs(n.Attr, "\n"+pad+"  ") + "\n" + pad + ">\n")
}

func attrs(as []html.Attribute, sep string) string {
	var b strings.Builder
	for _, a := range as {
		b.WriteString(sep + a.Key)
		if a.Val != "" {
			b.WriteString(`="` + html.EscapeString(a.Val) + `"`)
		}
	}
	return b.String()
}

// collapse turns each run of white space into one space, as HTML shows it.
func collapse(s string) string {
	var b strings.Builder
	sp := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f' {
			if !sp {
				b.WriteByte(' ')
			}
			sp = true
			continue
		}
		sp = false
		b.WriteRune(r)
	}
	return b.String()
}

// wrapText writes text at pad, wrapped at words to htmlWidth.
func wrapText(b *strings.Builder, t, pad string) {
	line := pad
	for _, w := range strings.Fields(html.EscapeString(t)) {
		if line != pad && len(line)+1+len(w) > htmlWidth {
			b.WriteString(line + "\n")
			line = pad
		}
		if line != pad {
			line += " "
		}
		line += w
	}
	b.WriteString(line + "\n")
}
