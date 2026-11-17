package main

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/net/html"
	"neuroplast.io/hotty-surf/surftest"
)

// names is the table's rows, top to bottom, by their menus' labels.
func names(r *surftest.Rig) []string {
	var out []string
	for _, n := range r.Find("span:text-start(Actions for )") {
		out = append(out, strings.TrimPrefix(text(n), "Actions for "))
	}
	return out
}

func text(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		b.WriteString(text(c))
	}
	return strings.TrimSpace(b.String())
}

func want(t *testing.T, r *surftest.Rig, rows ...string) {
	t.Helper()
	if got := names(r); !slices.Equal(got, rows) {
		t.Fatalf("rows %q, want %q", got, rows)
	}
}

// menu opens the row's menu, found by its entry's name in the folder shown,
// and returns the id its items are named by.
func menu(t *testing.T, a *App, r *surftest.Rig, name string) string {
	t.Helper()
	for _, e := range a.folder().Children {
		if e.Name == name {
			id := "menu-" + strconv.Itoa(e.ID)
			r.Click("#" + id + "-trigger")
			return id
		}
	}
	t.Fatalf("no %s", name)
	return ""
}

func TestExplore(t *testing.T) {
	a := NewApp()
	r := surftest.Start(t, newSurface("explorer", Page(a)), surftest.Size(100, 30))
	root := []string{"cmd", "docs", "internal", "themes", "go.mod", "go.sum", "LICENSE", "Makefile", "README.md"}
	want(t, r, root...)

	// Into a folder, and back up by the path at the top.
	r.Click("button:text(docs)")
	want(t, r, "images", "guide.md", "install.md", "themes.md")
	r.One("span[aria-current=page]:text(docs)")
	r.Click("#up-0")
	want(t, r, root...)

	// Sorted by size (a folder's by its entries), then the other way;
	// folders stay first.
	r.Click("#sort-size")
	want(t, r, "cmd", "themes", "internal", "docs", "go.mod", "Makefile", "README.md", "go.sum", "LICENSE")
	r.Click("#sort-size")
	want(t, r, "docs", "internal", "themes", "cmd", "LICENSE", "go.sum", "README.md", "Makefile", "go.mod")
	r.Click("#sort-date")
	want(t, r, "themes", "cmd", "internal", "docs", "LICENSE", "Makefile", "go.mod", "go.sum", "README.md")

	// Filtered by typing.
	r.Type("#filter", "go")
	want(t, r, "go.mod", "go.sum")
	r.Key("backspace")
	r.Key("backspace")
	want(t, r, "themes", "cmd", "internal", "docs", "LICENSE", "Makefile", "go.mod", "go.sum", "README.md")

	// Rename, from a row's menu.
	m := menu(t, a, r, "Makefile")
	r.Click("#" + m + "-items-1")
	r.One("h2:text(Rename Makefile)")
	if v := r.Value("#new-name"); v != "Makefile" {
		t.Fatalf("the new name starts as %q", v)
	}
	r.Key("ctrl+a")
	r.Type("#new-name", "go.mod")
	r.Key("enter")
	r.One("p:text(go.mod is already here.)")
	r.Key("ctrl+a")
	r.Type("#new-name", "Justfile")
	r.Key("enter")
	want(t, r, "themes", "cmd", "internal", "docs", "LICENSE", "Justfile", "go.mod", "go.sum", "README.md")
	if len(r.Find("dialog")) != 0 {
		t.Fatal("the dialog stays open")
	}

	// Delete, from a row's menu, once the user says so.
	m = menu(t, a, r, "go.sum")
	r.Click("#" + m + "-items-3")
	r.One("h2:text(Delete go.sum?)")
	r.Click("#delete-cancel")
	want(t, r, "themes", "cmd", "internal", "docs", "LICENSE", "Justfile", "go.mod", "go.sum", "README.md")
	m = menu(t, a, r, "go.sum")
	r.Click("#" + m + "-items-3")
	r.Click("#delete-confirm")
	want(t, r, "themes", "cmd", "internal", "docs", "LICENSE", "Justfile", "go.mod", "README.md")

	// Copy path shows the path from the project's root, to copy.
	r.Click("button:text(docs)")
	m = menu(t, a, r, "guide.md")
	r.Click("#" + m + "-items-2")
	if v := r.Value("#path"); v != "/docs/guide.md" {
		t.Fatalf("path %q", v)
	}
	r.Click("#copy-done")
}

func TestLint(t *testing.T) { surftest.Lint(t, ".") }
