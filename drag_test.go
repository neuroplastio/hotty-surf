package surf_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"

	"golang.org/x/net/html"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftest"
)

// A list that reorders by drag: each row is dragged and dropped on, split
// in halves (data-steps="0 1"), and the row lands before or after the one
// it is released over.
func TestDragReorders(t *testing.T) {
	rows := []string{"a", "b", "c"}
	var heard []string
	move := func(d surf.DragEvent) {
		heard = append(heard, fmt.Sprintf("start=%v end=%v id=%s over=%s ty=%d/%v", d.Start, d.End, d.ID, d.Over, d.TY, d.HasTY))
		if !d.End || d.Over == "" || d.Over == d.ID {
			return
		}
		from := slices.Index(rows, strings.TrimPrefix(d.ID, "row-"))
		item := rows[from]
		rows = slices.Delete(rows, from, from+1)
		to := slices.Index(rows, strings.TrimPrefix(d.Over, "row-"))
		if d.HasTY && d.TY == 1 {
			to++
		}
		rows = slices.Insert(rows, to, item)
	}
	dropped := ""
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		fmt.Fprint(w, `<ul>`)
		for _, r := range rows {
			fmt.Fprintf(w, `<li data-steps="0 1"%s>%s</li>`, on("row-"+r, surf.Drag(move), surf.Drop(func(d surf.DragEvent) { dropped = d.ID + " on " + d.Over })), r)
		}
		fmt.Fprint(w, `</ul>`)
	})
	r := surftest.Start(t, surf.New("t", v, ""))
	if on := attrOf(r.One("#row-a"), "data-on"); on != "drag drop" {
		t.Fatalf("data-on %q, want drag drop", on)
	}
	r.DragStep("#row-a", "#row-c", 0, 1)
	if got := strings.Join(rows, ""); got != "bca" {
		t.Fatalf("rows %s after dragging a after c; heard %q", got, heard)
	}
	want := []string{
		"start=true end=false id=row-a over=row-a ty=0/false",
		"start=false end=false id=row-a over=row-c ty=1/true",
		"start=false end=true id=row-a over=row-c ty=1/true",
	}
	if !slices.Equal(heard, want) {
		t.Fatalf("heard\n%q\nwant\n%q", heard, want)
	}
	if dropped != "row-a on row-c" {
		t.Fatalf("Drop heard %q", dropped)
	}
	if strings.Contains(r.HTML(), "dragging") {
		t.Fatalf("dragging stays after the drag: %s", r.HTML())
	}
	heard = nil
	r.Drag("#row-b", "#row-a")
	if got := strings.Join(rows, ""); got != "cba" {
		t.Fatalf("rows %s after dragging b onto a (no step: before it); heard %q", got, heard)
	}
}

func attrOf(n *html.Node, k string) string {
	for _, a := range n.Attr {
		if a.Key == k {
			return a.Val
		}
	}
	return ""
}

// The dragged element has the class dragging from the drag's first move
// to its release, and a press with no move leaves it off.
func TestDraggingClass(t *testing.T) {
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		fmt.Fprintf(w, `<div class="row"%s>a</div><div%s>b</div>`, on("a", surf.Drag(func(surf.DragEvent) {})), on("b", surf.Drop(func(surf.DragEvent) {})))
	})
	s := surf.New("t", v, "")
	var out bytes.Buffer
	s.Attach(&out, hotty.Caps{})
	doc := func() string {
		d, err := s.Document()
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	ev := func(kind, target string) {
		if err := s.Event(hotty.Event{Surface: "t", Kind: kind, Target: target, Drag: &hotty.Drag{}}); err != nil {
			t.Fatal(err)
		}
	}
	doc()
	ev(hotty.EventDragStart, "a")
	if strings.Contains(doc(), "dragging") {
		t.Fatal("dragging at the press, before a move")
	}
	ev(hotty.EventDrag, "b")
	if d := doc(); !strings.Contains(d, `class="row dragging"`) {
		t.Fatalf("no dragging while moving: %s", d)
	}
	ev(hotty.EventDragEnd, "b")
	if strings.Contains(doc(), "dragging") {
		t.Fatal("dragging after the release")
	}
}

// A secondary press runs the Handler of the element it is on, with the
// pointer's cell and the element's area, and is no click.
func TestSecondary(t *testing.T) {
	clicks, menus := 0, 0
	var at hotty.Cell
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		fmt.Fprintf(w, `<button%s>row</button>`, on("row", surf.Click(func() { clicks++ }),
			surf.Secondary(func(c hotty.Cell, _ hotty.Area) { menus++; at = c })))
	})
	r := surftest.Start(t, surf.New("t", v, ""))
	if on := attrOf(r.One("#row"), "data-on"); on != "click secondary" {
		t.Fatalf("data-on %q", on)
	}
	r.Secondary("#row")
	if menus != 1 || clicks != 0 || at != (hotty.Cell{}) {
		t.Fatalf("menus %d clicks %d at %v", menus, clicks, at)
	}
}

// A right click on another row while a menu is open moves the menu: the
// open menu's Outside closes it first, then the row's Secondary opens it.
func TestSecondaryAfterOutside(t *testing.T) {
	open, row := false, ""
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		for _, r := range []string{"a", "b"} {
			fmt.Fprintf(w, `<button%s>%s</button>`, on("row-"+r, surf.Secondary(func(hotty.Cell, hotty.Area) { open, row = true, r })), r)
		}
		if open {
			fmt.Fprintf(w, `<div%s>menu</div>`, on("menu", surf.Outside(func() { open = false })))
		}
	})
	r := surftest.Start(t, surf.New("t", v, ""))
	r.Secondary("#row-a")
	r.Secondary("#row-b")
	if !open || row != "b" {
		t.Fatalf("open %v at row %q, want the menu open at b", open, row)
	}
}
