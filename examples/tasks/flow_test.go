package main

import (
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
	a *App
}

func start(t *testing.T) *rig {
	t.Helper()
	tasks, err := upstream()
	if err != nil {
		t.Fatal(err)
	}
	a := NewApp(tasks)
	return &rig{surftest.Start(t, surf.New(surface, Page(a), CSS), surftest.Size(160, 50)), a}
}

func (r *rig) has(sel string) bool { return len(r.Find(sel)) > 0 }

func attr(n *html.Node, k string) string {
	for _, a := range n.Attr {
		if a.Key == k {
			return a.Val
		}
	}
	return ""
}

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

// rows are the body's rows, each its cells' texts.
func (r *rig) rows() [][]string {
	var out [][]string
	for _, tr := range r.Find("tr[data-slot=table-row]") {
		if tr.Parent == nil || tr.Parent.Data != "tbody" {
			continue
		}
		var cells []string
		for c := tr.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				cells = append(cells, norm(text(c)))
			}
		}
		out = append(out, cells)
	}
	return out
}

// TestFlow drives what the TSX port's flow test drives: the filter, the
// facets, sorting, paging, selection, the row's menu and its sub-menu,
// the View menu, the page-size select, and the keys in the lists.
func TestFlow(t *testing.T) {
	r := start(t)
	if n := len(r.rows()); n != 25 {
		t.Fatalf("first page: %d rows, want 25", n)
	}
	if !r.has("div:text(0 of 100 row(s) selected.)") || !r.has("div:text(Page 1 of 4)") {
		t.Fatal("the footer is wrong")
	}
	// The text filter.
	r.Type("input[placeholder=Filter tasks...]", "driver")
	for _, row := range r.rows() {
		if !strings.Contains(strings.ToLower(row[2]), "driver") {
			t.Fatalf("a row without driver: %v", row)
		}
	}
	if !r.has("button:text(Reset)") {
		t.Fatal("no Reset")
	}
	r.Click("button:text(Reset)")
	if len(r.rows()) != 25 || r.One("input[placeholder=Filter tasks...]") == nil {
		t.Fatal("Reset didn't")
	}
	if got := r.a.filter; got != "" {
		t.Fatalf("filter %q after Reset", got)
	}
	// A facet: the popover, its counts, a pick, the chip, Escape.
	r.Click("button[data-slot=popover-trigger]:text-start(Status)")
	if !r.has("[data-slot=popover-content]") {
		t.Fatal("the popover didn't open")
	}
	r.Click("div[role=menuitem]:text-start(In Progress)")
	for _, row := range r.rows() {
		if row[3] != "In Progress" {
			t.Fatalf("a row not In Progress: %v", row)
		}
	}
	if !r.has("span[data-slot=badge]:text(In Progress)") {
		t.Fatal("no chip")
	}
	// The keys in the list: the first item (Backlog) is active from the
	// start, as cmdk's; down once, and Enter picks Todo.
	if a := r.Find("[data-slot=command-item]"); len(a) == 0 || !strings.Contains(attr(a[0], "class"), "active") {
		t.Fatal("the first item isn't active")
	}
	r.Key("arrow_down")
	if !r.has("div[data-selected]") && len(r.Find("div[role=menuitem]")) == 0 {
		t.Fatal("no list")
	}
	active := r.Find("[data-slot=command-item]")
	n := 0
	for _, a := range active {
		if strings.Contains(attr(a, "class"), "active") {
			n++
			if !strings.HasPrefix(norm(text(a)), "Todo") {
				t.Fatalf("active: %q, want Todo", norm(text(a)))
			}
		}
	}
	if n != 1 {
		t.Fatalf("%d active items", n)
	}
	r.Key("enter")
	if !r.has("div:text(2 selected)") && !r.has("span[data-slot=badge]:text(Todo)") {
		t.Fatal("Enter didn't pick Todo")
	}
	r.Key("escape")
	if r.has("[data-slot=popover-content]") {
		t.Fatal("Escape didn't close")
	}
	r.Click("button:text(Reset)")
	// Sort by Title, both ways.
	r.Click("button:text-start(Title)")
	r.Click("div[role=menuitem]:text-start(Asc)")
	rows := r.rows()
	title := func(c string) string {
		for _, l := range labels {
			c = strings.TrimPrefix(c, l.Label)
		}
		return strings.ToLower(c)
	}
	if !(title(rows[0][2]) <= title(rows[1][2])) {
		t.Fatalf("not ascending: %q, %q", rows[0][2], rows[1][2])
	}
	r.Click("button:text-start(Title)")
	r.Click("div[role=menuitem]:text-start(Desc)")
	if !r.has("button[data-slot=dropdown-menu-trigger]:text-start(Title)") {
		t.Fatal("header gone")
	}
	// Hide a column from the View menu, and show it again.
	r.Click("button:text-start(View)")
	r.Click("div[role=menuitemcheckbox]:text(priority)")
	if r.has("button:text-start(Priority)") && len(r.Find("th")) != 5 {
		t.Fatalf("priority not hidden: %d heads", len(r.Find("th")))
	}
	r.Click("button:text-start(View)")
	r.Click("div[role=menuitemcheckbox]:text(priority)")
	// Selection.
	r.Click("input#r0-sel")
	r.Click("input#r1-sel")
	if !r.has("div:text(2 of 100 row(s) selected.)") {
		t.Fatal("selection not counted")
	}
	r.Click("input#select-all")
	if !r.has("div:text(25 of 100 row(s) selected.)") {
		t.Fatal("select all didn't")
	}
	r.Click("input#select-all")
	// Paging and the page size.
	r.Click("button:text-start(Go to next page)")
	if !r.has("div:text(Page 2 of 4)") {
		t.Fatal("next page")
	}
	r.Click("button[role=combobox]")
	r.Click("div[data-slot=select-item]:text(10)")
	// TanStack keeps the top row (the 26th) in view: page 3 of 10.
	if !r.has("div:text(Page 3 of 10)") || len(r.rows()) != 10 {
		t.Fatal("page size")
	}
	// A row's menu, its Labels sub-menu, Escape twice.
	r.Click("button#r0-menu")
	if !r.has("div[role=menuitem]:text(Edit)") {
		t.Fatal("row menu")
	}
	r.Click("div[role=menuitem]:text-start(Labels)")
	if !r.has("div[role=menuitemradio]:text(Bug)") {
		t.Fatal("labels sub-menu")
	}
	r.Key("escape")
	r.Key("escape")
	if r.has("div[role=menuitem]:text(Edit)") {
		t.Fatal("row menu still open")
	}
}

// TestLint: the app keeps the kit's rules.
func TestLint(t *testing.T) { surftest.Lint(t, ".") }
