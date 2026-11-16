package main

import (
	"slices"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
)

// rig is the app on surftest's rig: after every act, the host's document
// must be the surface's own tree, so the deltas are right.
type rig struct {
	*surftest.Rig
	t *testing.T
	a *App
}

func start(t *testing.T) *rig {
	t.Helper()
	a := NewApp(Mock())
	return &rig{surftest.Start(t, surf.New(surface, Page(a), CSS), surftest.Size(40, 40)), t, a}
}

// Walk calls f on every element of the host's document, in order.
func (r *rig) Walk(f func(*html.Node)) {
	for _, n := range r.Find("") {
		f(n)
	}
}

func attr(n *html.Node, k string) string {
	for _, a := range n.Attr {
		if a.Key == k {
			return a.Val
		}
	}
	return ""
}

func has(n *html.Node, class string) bool { return slices.Contains(strings.Fields(attr(n, "class")), class) }

// row is a session's button.
func (r *rig) row(id string) *html.Node {
	r.t.Helper()
	var out *html.Node
	r.Walk(func(n *html.Node) {
		if n.Data == "button" && attr(n, "id") == "row-"+id {
			out = n
		}
	})
	if out == nil {
		r.t.Fatalf("no row for %s", id)
	}
	return out
}

// selected is the ids of the rows that say they are selected.
func (r *rig) selected() []string {
	var out []string
	r.Walk(func(n *html.Node) {
		if n.Data == "button" && attr(n, "role") == "option" && attr(n, "aria-selected") == "true" {
			out = append(out, strings.TrimPrefix(attr(n, "id"), "row-"))
		}
	})
	return out
}

func (r *rig) click(id string) { r.t.Helper(); r.Click("#row-" + id) }

func TestNav(t *testing.T) {
	r := start(t)
	// The data: five rows in a listbox, the selection the data's, not the
	// code's.
	var opts int
	var list *html.Node
	r.Walk(func(n *html.Node) {
		if attr(n, "role") == "option" {
			opts++
		}
		if attr(n, "role") == "listbox" {
			list = n
		}
	})
	if opts != 5 || list == nil {
		t.Fatalf("%d options, listbox %v; want 5 in a listbox", opts, list != nil)
	}
	if got := r.selected(); !slices.Equal(got, []string{"s3"}) {
		t.Fatalf("selected %v, want [s3]", got)
	}
	if !has(r.row("s3"), "bg-accent") || !has(r.row("s1"), "bg-sidebar-accent") || has(r.row("s2"), "bg-sidebar-accent") {
		t.Fatalf("classes: s3 %q s1 %q s2 %q", attr(r.row("s3"), "class"), attr(r.row("s1"), "class"), attr(r.row("s2"), "class"))
	}
	// A layout's thumbnail says what it shows, and busy is the layout's.
	for id, want := range map[string]string{"s1": "2 panes, 2 working", "s2": "1 pane", "s3": "4 panes, 2 working", "s5": "2 panes"} {
		if got := r.thumb(id); got != want {
			t.Errorf("%s: thumbnail %q, want %q", id, got, want)
		}
	}
	// A click selects, and picks: the row is lighter, its bar grows, and the
	// one that was selected gives both up.
	r.click("s4")
	if got := r.selected(); !slices.Equal(got, []string{"s4"}) {
		t.Fatalf("after a click: selected %v, want [s4]", got)
	}
	if !has(r.row("s4"), "bg-accent") || has(r.row("s3"), "bg-accent") {
		t.Fatalf("after a click: s4 %q, s3 %q", attr(r.row("s4"), "class"), attr(r.row("s3"), "class"))
	}
	if !r.bar("s4", "h-11") || !r.bar("s3", "h-0") {
		t.Fatal("after a click: the bar is not on s4 alone")
	}
	if r.a.active != "s4" {
		t.Fatalf("active %q after a click, want s4", r.a.active)
	}
	// The arrows move the cursor and stop at the ends; they don't pick.
	r.Key("arrow_down")
	r.Key("arrow_down")
	if got := r.selected(); !slices.Equal(got, []string{"s5"}) {
		t.Fatalf("after two Down: selected %v, want [s5] (the last row)", got)
	}
	r.Key("arrow_up")
	r.Key("arrow_up")
	r.Key("arrow_up")
	r.Key("arrow_up")
	r.Key("arrow_up")
	if got := r.selected(); !slices.Equal(got, []string{"s1"}) {
		t.Fatalf("after Up to the top: selected %v, want [s1]", got)
	}
	if r.a.active != "s4" {
		t.Fatalf("the arrows picked: active %q, want s4", r.a.active)
	}
	// Enter picks the row the cursor is on: the host clicks the focused button.
	r.Key("arrow_down")
	r.Key("enter")
	if r.a.active != "s2" || !slices.Equal(r.selected(), []string{"s2"}) {
		t.Fatalf("after Enter: active %q, selected %v; want s2", r.a.active, r.selected())
	}
	// A key that isn't the list's goes on.
	if r.Key("x") {
		t.Fatal("the list used x")
	}
}

// thumb is the text alternative of a row's layout thumbnail.
func (r *rig) thumb(id string) string {
	var out string
	var in func(*html.Node)
	in = func(n *html.Node) {
		if n.Type == html.ElementNode && attr(n, "role") == "img" {
			out = attr(n, "aria-label")
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			in(c)
		}
	}
	in(r.row(id))
	return out
}

// bar says whether a row's accent bar has the class.
func (r *rig) bar(id, class string) bool {
	for c := r.row(id).FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "span" && has(c, "bg-sidebar-primary") {
			return has(c, class)
		}
	}
	return false
}

// TestLint: the app keeps the kit's rules.
func TestLint(t *testing.T) { surftest.Lint(t, ".") }
