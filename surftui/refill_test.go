package surftui

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"neuroplast.io/hotty-go"
)

// A same-shape a=doc refills the kept tree (load). It must leave the
// Screen as a load leaves it: drawn the same, its focus, fields, scroll,
// pointer and pending reveal as a new tree's, so that the path it takes
// can't be told from the other.

// sameShapeCase is a history: a first document, what the user does with
// it, a second document of the same shape (after pre, in the same
// Write), and what the user does after.
type sameShapeCase struct {
	name          string
	cols, rows    int
	a, b          string
	pre           string
	before, after func(c *Screen)
	// pending leaves what before did for the next layout, the second
	// document's: no layout between them.
	pending bool
}

// twin plays a case on a new Screen. load says whether its second
// document is loaded as a new tree: its kept memos dropped first, which a
// load needs, else it is refilled. It returns the Screen, what it sent
// after the second document, and whether the second document kept the
// tree (refilled it).
func (tc sameShapeCase) twin(t *testing.T, load bool) (*Screen, []string, bool) {
	t.Helper()
	var evs []string
	c := New(tc.cols, tc.rows, func(e hotty.Event) { evs = append(evs, hotty.EncodeEvent(e)) })
	if _, err := c.Write([]byte(hotty.Doc("s", tc.a))); err != nil {
		t.Fatal(err)
	}
	c.Lines()
	c.Cells()
	if tc.before != nil {
		tc.before(c)
	}
	if !tc.pending {
		c.Lines() // a layout, which a refill needs
	}
	shown := fmt.Sprint(c.shown.cells)
	if load {
		c.kept = nil
	}
	body := c.body
	if _, err := c.Write([]byte(tc.pre + hotty.Doc("s", tc.b))); err != nil {
		t.Fatal(err)
	}
	kept := c.body == body
	// Dirty may say so of a document that draws the same (a load's
	// cells are new elements'), never not of one that draws otherwise.
	if !c.Dirty() && fmt.Sprint(c.Cells()) != shown {
		t.Errorf("load %v: not dirty, and the a=doc changed what it draws", load)
	}
	evs = nil
	if tc.after != nil {
		tc.after(c)
	}
	return c, evs, kept
}

// drawn is what a Screen shows now: its lines, its cells and its caret.
func drawn(c *Screen) string {
	col, row, ok := c.Cursor()
	return fmt.Sprintf("cursor %d,%d %v\n%s\n%v", col, row, ok, strings.Join(c.Lines(), "\n"), c.Cells())
}

func checkSameShape(t *testing.T, tc sameShapeCase) {
	t.Helper()
	r, rEvs, refilled := tc.twin(t, false)
	l, lEvs, kept := tc.twin(t, true)
	if !refilled {
		t.Fatal("the second document was loaded, not refilled: the case doesn't test the refill")
	}
	if kept {
		t.Fatal("the load twin kept its tree")
	}
	if !slices.Equal(rEvs, lEvs) {
		t.Errorf("events after the a=doc:\nrefilled %q\nloaded   %q", rEvs, lEvs)
	}
	if got, want := drawn(r), drawn(l); got != want {
		t.Errorf("refilled draws:\n%s\nloaded draws:\n%s", got, want)
	}
}

// A changed attribute changes the styles of other elements than its own:
// its children's (kit: a field's label grows, a breadcrumb's list is a
// row), its parent's (an input group's addon, a popover's side), its
// ancestors' (a command group whose items are all filtered out); and a
// scheme's colours, details' open, plain text. The verifier's cases.
func TestSameShapeDocDrawsAsALoad(t *testing.T) {
	for _, tc := range []sameShapeCase{
		{name: "a field's orientation",
			a: `<div class="field"><label>Name</label><input class="input" value="x"></div>`,
			b: `<div class="field" data-orientation="horizontal"><label>Name</label><input class="input" value="x"></div>`},
		{name: "an input group's addon to block-end",
			a: `<div class="input-group"><input class="input"><span data-align="inline-end">kg</span></div>`,
			b: `<div class="input-group"><input class="input"><span data-align="block-end">kg</span></div>`},
		{name: "a command group's last item filtered out",
			a: `<div class="command"><div role="menu"><div role="group"><b>Files</b><div role="menuitem">Open</div></div></div></div>`,
			b: `<div class="command"><div role="menu"><div role="group"><b>Files</b><div role="menuitem" aria-hidden="true">Open</div></div></div></div>`},
		{name: "a breadcrumb's class on its nav",
			a: `<nav><ol><li>Home</li><li>Docs</li></ol></nav>`,
			b: `<nav class="breadcrumb"><ol><li>Home</li><li>Docs</li></ol></nav>`},
		{name: "a popover's side",
			a: `<div class="popover"><button>T</button><div data-popover data-side="bottom">Pop</div></div>`,
			b: `<div class="popover"><button>T</button><div data-popover data-side="inline-end">Pop</div></div>`},
		{name: "text only", a: `<p>one</p>`, b: `<p>two</p>`},
		{name: "details open",
			a: `<details><summary>S</summary><p>body</p></details>`,
			b: `<details open><summary>S</summary><p>body</p></details>`},
		{name: "a dark scheme",
			a: `<div><p class="text-primary">x</p></div>`,
			b: `<div class="dark"><p class="btn" data-variant="outline">x</p></div>`},
		{name: "a colour class",
			a: `<p><span class="text-primary">word</span> and <b>more</b></p>`,
			b: `<p><span class="text-destructive">word</span> and <b class="font-normal">more</b></p>`},
	} {
		tc.cols, tc.rows = 40, 6
		t.Run(tc.name, func(t *testing.T) { checkSameShape(t, tc) })
	}
}

// What the user did with the first document is gone with it: the focus,
// a field's value, revision and selection, the page's scroll and a box's,
// a details the user opened.
func TestSameShapeDocResetsState(t *testing.T) {
	field := `<input id="f" class="input" value="ab" data-on="input change"><button id="b">ok</button>`
	long := strings.Repeat("<p>line</p>", 20) + "<p>end</p>"
	for _, tc := range []sameShapeCase{
		{name: "a field typed in, its selection",
			a: field, b: field,
			before: func(c *Screen) {
				c.Click(1, 0)
				c.Paste("xyz")
				c.Key("left")
				c.Key("shift+left")
			},
			after: func(c *Screen) {
				c.Click(1, 0)
				c.Paste("Q")
				c.Key("enter")
			}},
		{name: "the focus",
			a: field, b: field,
			before: func(c *Screen) { c.Key("tab"); c.Key("tab") }},
		{name: "the page's scroll",
			a: long, b: long,
			before: func(c *Screen) { c.Wheel(0, 0, 6) }},
		{name: "a box's scroll",
			a: scrollDocOf(8), b: scrollDocOf(8),
			before: func(c *Screen) { c.Wheel(1, 1, 2) }},
		{name: "a details the user opened",
			a:      `<details><summary id="s">S</summary><p>body</p></details>`,
			b:      `<details><summary id="s">S</summary><p>body</p></details>`,
			before: func(c *Screen) { c.Click(1, 0) }},
	} {
		tc.cols, tc.rows = 30, 5
		t.Run(tc.name, func(t *testing.T) { checkSameShape(t, tc) })
	}
}

// What the pointer held is the old document's: a press doesn't click on
// release, a click doesn't count on to a double, a tooltip goes until the
// pointer moves, and a reveal waiting for a layout doesn't scroll the
// new document.
func TestSameShapeDocDropsPointerState(t *testing.T) {
	at := time.Unix(1000, 0)
	now = func() time.Time { return at }
	defer func() { now = time.Now }()
	button := `<button id="b" data-on="click">go</button>`
	tip := `<p>x</p><p>x</p><button id="t" data-tooltip="Save the file">Save</button>`
	for _, tc := range []sameShapeCase{
		{name: "a press, then its release",
			a: button, b: button,
			before: func(c *Screen) { c.Press(1, 0, 0, 0) },
			after: func(c *Screen) {
				c.Move(2, 0, 0)
				c.Release(1, 0, 0)
			}},
		{name: "a click, then another",
			a: button, b: button,
			before: func(c *Screen) {
				at = time.Unix(1000, 0)
				c.Press(1, 0, 0, 0)
				c.Release(1, 0, 0)
			},
			after: func(c *Screen) {
				at = at.Add(200 * time.Millisecond)
				c.Press(1, 0, 0, 0)
				c.Release(1, 0, 0)
			}},
		{name: "a tooltip shown",
			a: tip, b: tip,
			before: func(c *Screen) { c.Move(2, 2, 0) }},
		{name: "a tooltip shown, the pointer moved after",
			a: tip, b: tip,
			before: func(c *Screen) { c.Move(2, 2, 0) },
			after:  func(c *Screen) { c.Lines(); c.Move(3, 2, 0) }},
		// A reveal op is a delta, which drops the kept tree: the reveal
		// a refill can meet is a focus's.
		{name: "the program's focus, its reveal pending",
			a: scrollDocOf(8), b: scrollDocOf(8),
			pre: hotty.Focus("s", "item-8")},
		{name: "a key's focus, its reveal pending",
			a: scrollDocOf(8), b: scrollDocOf(8), pending: true,
			before: func(c *Screen) {
				for range 8 {
					c.Key("tab")
				}
			}},
	} {
		tc.cols, tc.rows = 30, 6
		t.Run(tc.name, func(t *testing.T) { checkSameShape(t, tc) })
	}
}

// scrollDocOf is a list of n items, which take the focus, in a box three
// rows high.
func scrollDocOf(n int) string {
	var b strings.Builder
	b.WriteString(`<div id="list" style="overflow-y:auto;max-height:3rem">`)
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, `<div id="item-%d" tabindex="0">item %d</div>`, i, i)
	}
	b.WriteString(`</div><p>after list</p>`)
	return b.String()
}

// The memos a layout keeps until the tree changes hold for a delta that
// changes what another element's style reads, and for a details the user
// toggles.
func TestKeptMemosAcrossChanges(t *testing.T) {
	fresh := func(doc string) string {
		c := New(40, 6, nil)
		if _, err := c.Write([]byte(hotty.Doc("s", doc))); err != nil {
			t.Fatal(err)
		}
		return strings.Join(c.Lines(), "\n")
	}
	t.Run("a delta on a parent", func(t *testing.T) {
		c := New(40, 6, nil)
		c.Write([]byte(hotty.Doc("s", `<div id="f" class="field"><label>Name</label><input class="input"></div>`)))
		c.Lines()
		if _, err := c.Write([]byte(hotty.Delta("s", []hotty.Op{hotty.OpAttr("f", "data-orientation", "horizontal")}))); err != nil {
			t.Fatal(err)
		}
		want := fresh(`<div id="f" class="field" data-orientation="horizontal"><label>Name</label><input class="input"></div>`)
		if got := strings.Join(c.Lines(), "\n"); got != want {
			t.Errorf("drew:\n%s\nwant:\n%s", got, want)
		}
	})
	t.Run("a details opened", func(t *testing.T) {
		c := New(40, 6, nil)
		c.Write([]byte(hotty.Doc("s", `<details><summary id="s">S</summary><p>body</p></details>`)))
		c.Lines()
		c.Click(1, 0)
		// As text: the summary has the keyboard.
		want := noSGR(fresh(`<details open><summary id="s">S</summary><p>body</p></details>`))
		if got := noSGR(strings.Join(c.Lines(), "\n")); !strings.Contains(got, "body") || got != want {
			t.Errorf("drew:\n%s\nwant:\n%s", got, want)
		}
	})
}

// noSGR is s without its SGR.
func noSGR(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
