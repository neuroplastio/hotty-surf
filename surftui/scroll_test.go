package surftui_test

import (
	"fmt"
	"strings"
	"testing"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf/surftui"
)

func scrollDoc(n int) string {
	var b strings.Builder
	b.WriteString(`<div id="root"><div id="list" style="overflow-y:auto;max-height:3rem">`)
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, `<div id="item-%d">item %d</div>`, i, i)
	}
	b.WriteString(`</div><p>after list</p></div>`)
	return b.String()
}

func TestOverflowWheelAndReveal(t *testing.T) {
	c := surftui.New(30, 8, nil)
	if _, err := c.Write([]byte(hotty.Doc("test", scrollDoc(8)))); err != nil {
		t.Fatal(err)
	}
	text := func() string { return strings.Join(c.Lines(), "\n") }
	if got := text(); !strings.Contains(got, "item 1") || strings.Contains(got, "item 4") || !strings.Contains(got, "after list") {
		t.Fatalf("initial bounded list:\n%s", got)
	}
	if !c.Wheel(1, 1, 2) {
		t.Fatal("wheel did not scroll the list")
	}
	if got := text(); strings.Contains(got, "item 1") || !strings.Contains(got, "item 3") || !strings.Contains(got, "item 5") || !strings.Contains(got, "after list") {
		t.Fatalf("wheel did not keep the list bounded:\n%s", got)
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(scrollDoc(8), `<div id="root">`), `</div>`)
	if _, err := c.Write([]byte(hotty.Delta("test", []hotty.Op{hotty.OpInner("root", inner)}))); err != nil {
		t.Fatal(err)
	}
	if got := text(); strings.Contains(got, "item 1") || !strings.Contains(got, "item 3") {
		t.Fatalf("inner morph lost the list scroll:\n%s", got)
	}
	if _, err := c.Write([]byte(hotty.Reveal("test", "item-8", hotty.RevealNearest))); err != nil {
		t.Fatal(err)
	}
	if got := text(); !strings.Contains(got, "item 8") || strings.Contains(got, "item 5") || !strings.Contains(got, "after list") {
		t.Fatalf("reveal did not scroll the list:\n%s", got)
	}
	if _, err := c.Write([]byte(hotty.Reveal("test", "item-4", hotty.RevealStart))); err != nil {
		t.Fatal(err)
	}
	if got := text(); !strings.Contains(got, "item 4") || !strings.Contains(got, "item 6") || strings.Contains(got, "item 3") {
		t.Fatalf("start alignment:\n%s", got)
	}
	if _, err := c.Write([]byte(hotty.Reveal("test", "item-5", hotty.RevealEnd))); err != nil {
		t.Fatal(err)
	}
	if got := text(); !strings.Contains(got, "item 3") || !strings.Contains(got, "item 5") || strings.Contains(got, "item 6") {
		t.Fatalf("end alignment:\n%s", got)
	}
	if _, err := c.Write([]byte(hotty.Doc("test", scrollDoc(2)))); err != nil {
		t.Fatal(err)
	}
	if got := text(); !strings.Contains(got, "item 1") || !strings.Contains(got, "item 2") {
		t.Fatalf("new document retained old scroll:\n%s", got)
	}
}

func TestNestedOverflowReveal(t *testing.T) {
	c := surftui.New(30, 8, nil)
	doc := `<div id="root"><div id="outer" style="overflow-y:auto;max-height:3rem"><p>before</p><div id="inner" style="overflow-y:auto;max-height:2rem"><p id="a">alpha</p><p>beta</p><p id="z">zeta</p></div><p>after inner</p></div><p>after outer</p></div>`
	if _, err := c.Write([]byte(hotty.Doc("test", doc))); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Write([]byte(hotty.Reveal("test", "z", hotty.RevealNearest))); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(c.Lines(), "\n")
	if !strings.Contains(got, "zeta") || strings.Contains(got, "alpha") || !strings.Contains(got, "after outer") {
		t.Fatalf("nested reveal:\n%s", got)
	}
}
