package surftui_test

import (
	"fmt"
	"strings"
	"testing"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf/surftui"
)

// freshShot is what a test compares: Lines, and every cell with its roles.
func freshShot(c *surftui.Screen) string {
	var b strings.Builder
	for _, l := range c.Lines() {
		fmt.Fprintf(&b, "%q\n", l)
	}
	for _, r := range c.Cells() {
		for _, k := range r {
			fmt.Fprintf(&b, "%s|%s|%s|%d ", k.Text, k.Fg, k.Bg, k.Attr)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// A colour-only class change is told by splitting classes at spaces, and
// every rule that reads a class must split it the same way: a class with a
// tab, a vertical tab or a no-break space in it isn't colour-only (the verifier's).
func TestColourOnlyClassesWithOtherSpaces(t *testing.T) {
	cases := []struct {
		name, before, after string
		ops                 []hotty.Op
	}{
		{"tab-font-normal",
			`<b><span id="a" class="x` + "\t" + `font-normal">word</span> bold</b>`,
			`<b><span id="a" class="x font-normal">word</span> bold</b>`,
			[]hotty.Op{hotty.OpAttr("a", "class", "x font-normal")}},
		{"nbsp-hidden",
			`<div><span id="a" class="x">word</span> more</div>`,
			`<div><span id="a" class="x bg-primary/5` + "\u00a0" + `hidden">word</span> more</div>`,
			[]hotty.Op{hotty.OpAttr("a", "class", "x bg-primary/5\u00a0hidden")}},
		{"nbsp-paren-hidden",
			`<div><span id="a" class="x">word</span> more</div>`,
			`<div><span id="a" class="x text-(--pill)` + "\u00a0" + `hidden">word</span> more</div>`,
			[]hotty.Op{hotty.OpAttr("a", "class", "x text-(--pill)\u00a0hidden")}},
		{"tab-input-full",
			`<div class="flex"><input id="a" class="input` + "\t" + `x" value="v"><span>after</span></div>`,
			`<div class="flex"><input id="a" class="input x" value="v"><span>after</span></div>`,
			[]hotty.Op{hotty.OpAttr("a", "class", "input x")}},
		{"vt-hidden",
			`<div><span id="a" class="x">word</span> more</div>`,
			`<div><span id="a" class="x bg-primary/5` + "\v" + `hidden">word</span> more</div>`,
			[]hotty.Op{hotty.OpAttr("a", "class", "x bg-primary/5\vhidden")}},
	}
	for _, tc := range cases {
		c := surftui.New(40, 3, func(hotty.Event) {})
		if _, err := c.Write([]byte(hotty.Doc("v", tc.before))); err != nil {
			t.Fatal(err)
		}
		c.Cells()
		if _, err := c.Write([]byte(hotty.Delta("v", tc.ops))); err != nil {
			t.Fatal(err)
		}
		dirty := c.Dirty()
		got := freshShot(c)
		f := surftui.New(40, 3, func(hotty.Event) {})
		f.Write([]byte(hotty.Doc("v", tc.after)))
		want := freshShot(f)
		if got != want {
			t.Errorf("%s: delta's Screen differs from a fresh one (dirty %v)\n got:\n%s\nwant:\n%s", tc.name, dirty, got, want)
		}
	}
}

// Ops later in one delta find the elements the earlier ones made, moved,
// renamed or removed (the verifier's).
func TestDeltaOpsSeeEarlierOps(t *testing.T) {
	cases := []struct {
		name, before, after string
		ops                 []hotty.Op
	}{
		{"replace-then-attr", `<div id="r"><p id="a">old</p></div>`, `<div id="r"><p id="a" hidden>new</p> tail</div>`,
			[]hotty.Op{hotty.OpReplace("a", `<p id="a">new</p> tail`), hotty.OpAttr("a", "hidden", "")}},
		{"append-then-text", `<div id="r">x</div>`, `<div id="r">x<span id="n">set</span></div>`,
			[]hotty.Op{hotty.OpAppend("r", `<span id="n">new</span>`), hotty.OpText("n", "set")}},
		{"reid-then-attr", `<p id="a">word</p>`, `<p id="b" hidden>word</p>`,
			[]hotty.Op{hotty.OpAttr("a", "id", "b"), hotty.OpAttr("b", "hidden", "")}},
		{"remove-then-text", `<div id="r"><p id="a">one</p><p id="b">two</p></div>`, ``,
			[]hotty.Op{hotty.OpRemove("a"), hotty.OpText("a", "ghost")}},
	}
	for _, tc := range cases {
		c := surftui.New(30, 3, func(hotty.Event) {})
		c.Write([]byte(hotty.Doc("v", tc.before)))
		c.Cells()
		_, err := c.Write([]byte(hotty.Delta("v", tc.ops)))
		got := freshShot(c)
		t.Logf("%s: err %v\n%s", tc.name, err, got[:firstLines(got)])
		if tc.after != "" {
			f := surftui.New(30, 3, func(hotty.Event) {})
			f.Write([]byte(hotty.Doc("v", tc.after)))
			if want := freshShot(f); got != want || err != nil {
				t.Errorf("%s: differs from a fresh Screen of the intended tree (err %v)", tc.name, err)
			}
		} else if err == nil {
			t.Errorf("%s: an op on a removed element succeeded", tc.name)
		}
	}
}

// firstLines is where the third line ends: what a log of a shot shows.
func firstLines(s string) int {
	n := 0
	for i := range s {
		if s[i] == '\n' {
			n++
			if n == 3 {
				return i
			}
		}
	}
	return len(s)
}
