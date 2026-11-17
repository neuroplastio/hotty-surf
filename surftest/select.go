package surftest

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// selector is a parsed tag#id[attr=value]:text(…).
type selector struct {
	tag, id string
	attrs   [][2]string // name, value; a value of "\x00" is presence only
	text    string
	how     string // "", "text" or "text-start"
}

func parseSelector(sel string) (*selector, error) {
	m := &selector{}
	rest := sel
	i := 0
	for i < len(rest) && (rest[i] >= 'a' && rest[i] <= 'z' || rest[i] >= '0' && rest[i] <= '9') {
		i++
	}
	m.tag, rest = rest[:i], rest[i:]
	if strings.HasPrefix(rest, "#") {
		j := strings.IndexAny(rest, "[:")
		if j < 0 {
			j = len(rest)
		}
		m.id, rest = rest[1:j], rest[j:]
	}
	for strings.HasPrefix(rest, "[") {
		j := strings.IndexByte(rest, ']')
		if j < 0 {
			return nil, fmt.Errorf("surftest: selector %q: a [ without its ]", sel)
		}
		in := rest[1:j]
		rest = rest[j+1:]
		if k := strings.IndexByte(in, '='); k >= 0 {
			v := in[k+1:]
			if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
				v = v[1 : len(v)-1]
			}
			m.attrs = append(m.attrs, [2]string{in[:k], v})
		} else {
			m.attrs = append(m.attrs, [2]string{in, "\x00"})
		}
	}
	for _, how := range []string{"text-start", "text"} {
		if strings.HasPrefix(rest, ":"+how+"(") && strings.HasSuffix(rest, ")") {
			m.how, m.text = how, rest[len(how)+2:len(rest)-1]
			rest = ""
			break
		}
	}
	if rest != "" {
		return nil, fmt.Errorf("surftest: selector %q: can't read %q (tag#id[attr=value]:text(…))", sel, rest)
	}
	return m, nil
}

func (m *selector) match(n *html.Node) bool {
	if m.tag != "" && n.Data != m.tag {
		return false
	}
	if m.id != "" && attr(n, "id") != m.id {
		return false
	}
	for _, a := range m.attrs {
		v, ok := get(n, a[0])
		if !ok || a[1] != "\x00" && v != a[1] {
			return false
		}
	}
	switch m.how {
	case "text":
		return norm(text(n)) == m.text
	case "text-start":
		return strings.HasPrefix(norm(text(n)), m.text)
	}
	return true
}

func walk(n *html.Node, f func(*html.Node)) {
	if n.Type == html.ElementNode {
		f(n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, f)
	}
}

func get(n *html.Node, k string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == k {
			return a.Val, true
		}
	}
	return "", false
}

func attr(n *html.Node, k string) string { v, _ := get(n, k); return v }

func hasAttr(n *html.Node, k string) bool { _, ok := get(n, k); return ok }

func text(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		b.WriteString(text(c))
	}
	return b.String()
}

func norm(s string) string { return strings.Join(strings.Fields(s), " ") }

// canon is a document as text with each element's attributes sorted, and
// without a live field's value and checked state, or a select's chosen
// option: they are the host's, set by a=field, not attributes.
func canon(t testing.TB, doc string) string {
	t.Helper()
	n, err := html.Parse(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	var visit func(*html.Node)
	visit = func(x *html.Node) {
		if x.Type == html.ElementNode {
			var keep []html.Attribute
			for _, a := range x.Attr {
				if x.Data == "input" && (a.Key == "value" || a.Key == "checked") || x.Data == "option" && a.Key == "selected" {
					continue
				}
				keep = append(keep, a)
			}
			slices.SortFunc(keep, func(a, b html.Attribute) int { return strings.Compare(a.Key, b.Key) })
			x.Attr = keep
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(n)
	var b strings.Builder
	html.Render(&b, n)
	return b.String()
}
