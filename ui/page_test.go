package ui_test

import (
	"context"
	"html"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/a-h/templ"
	x "golang.org/x/net/html"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/ui"
)

// page is a <main> holding parts, for a test's view.
func page(parts ...templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		io.WriteString(w, `<main>`)
		for _, p := range parts {
			if err := p.Render(ctx, w); err != nil {
				return err
			}
		}
		_, err := io.WriteString(w, `</main>`)
		return err
	})
}

// button is a plain button with the id, its click fn: something to click
// outside a component, or that opens one.
func button(id, label string, fn func()) templ.Component {
	return element("button", `type="button"`, id, html.EscapeString(label), surf.Click(fn))
}

// field is a text field with the id, holding value, its input fn.
func field(id, value string, fn func(string)) templ.Component {
	return element("input", `class="input" value="`+html.EscapeString(value)+`"`, id, "", surf.Input(fn))
}

// element is a tag with the id and handlers hs, more attributes, and
// inner markup (none for an input).
func element(tag, more, id, inner string, hs ...surf.Handler) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		a := surf.On(ctx, id, hs...)
		var b strings.Builder
		b.WriteString("<" + tag + " " + more)
		keys := make([]string, 0, len(a))
		for k := range a {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			b.WriteString(" " + k + `="` + html.EscapeString(a[k].(string)) + `"`)
		}
		b.WriteString(">")
		if tag != "input" {
			b.WriteString(inner + "</" + tag + ">")
		}
		_, err := io.WriteString(w, b.String())
		return err
	})
}

// seq is parts one after the other.
func seq(parts ...templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		for _, p := range parts {
			if err := p.Render(ctx, w); err != nil {
				return err
			}
		}
		return nil
	})
}

// text is a paragraph: something to click that takes no focus.
func text(id, s string) templ.Component {
	return templ.Raw(`<p id="` + id + `">` + html.EscapeString(s) + `</p>`)
}

// with is c holding children.
func with(c templ.Component, children templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		return c.Render(templ.WithChildren(ctx, children), w)
	})
}

func start(t *testing.T, view templ.Component) *surftest.Rig {
	t.Helper()
	return surftest.Start(t, surf.New("t", view, ui.Stylesheet()))
}

func attr(n *x.Node, k string) string {
	for _, a := range n.Attr {
		if a.Key == k {
			return a.Val
		}
	}
	return ""
}

func hasAttr(n *x.Node, k string) bool {
	for _, a := range n.Attr {
		if a.Key == k {
			return true
		}
	}
	return false
}

// want fails the test unless attribute k of the one element sel is v.
func want(t *testing.T, r *surftest.Rig, sel, k, v string) {
	t.Helper()
	if got := attr(r.One(sel), k); got != v {
		t.Errorf("%s's %s is %q, not %q", sel, k, got, v)
	}
}

// active is the id of the element with class active in the host's
// document under sel, "" for none.
func active(r *surftest.Rig, sel string) string {
	var out []string
	var walk func(*x.Node)
	walk = func(n *x.Node) {
		if n.Type == x.ElementNode && slices.Contains(strings.Fields(attr(n, "class")), "active") {
			out = append(out, attr(n, "id"))
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(r.One(sel))
	return strings.Join(out, " ")
}
