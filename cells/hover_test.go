package cells

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"neuroplast.io/hotty-go"
)

// hoverScreen is a Screen of doc, 60 by 12, drawn.
func hoverScreen(t *testing.T, doc string) *Screen {
	t.Helper()
	c := New(60, 12, func(hotty.Event) {})
	if _, err := c.Write([]byte(hotty.Doc("v", doc))); err != nil {
		t.Fatal(err)
	}
	c.Cells()
	return c
}

// cellIn is a cell of the element with the id, inside its box, from the
// Screen's top left: a cell of its label where it is a control.
func cellIn(t *testing.T, c *Screen, id string) (col, row int) {
	t.Helper()
	n := c.byID(id)
	r, ok := c.layout().boxes[n]
	if n == nil || !ok || r.w == 0 || r.h == 0 {
		t.Fatalf("#%s isn't drawn", id)
	}
	return r.x + min(2, r.w-1), r.y - c.scroll
}

// moveOver moves the pointer over the element with the id, and holds the
// Screen to a fresh pass.
func moveOver(t *testing.T, c *Screen, id string) (col, row int) {
	t.Helper()
	col, row = cellIn(t, c, id)
	c.Move(col, row, 0)
	if err := CheckFresh(c); err != nil {
		t.Fatalf("over #%s: %v", id, err)
	}
	return col, row
}

// over moves the pointer over the element with the id, and says what
// Cells draws there.
func over(t *testing.T, c *Screen, id string) Cell {
	t.Helper()
	col, row := moveOver(t, c, id)
	return c.Cells()[row][col]
}

// cellAt is what Cells draws at the element with the id.
func cellAt(t *testing.T, c *Screen, id string) Cell {
	t.Helper()
	col, row := cellIn(t, c, id)
	return c.Cells()[row][col]
}

// The hovered chain: the element under the pointer and the elements
// around it match hover:, the others don't; with a button down it stays
// where the press was; off the Screen and on losing focus there is none.
func TestHoverChain(t *testing.T) {
	c := hoverScreen(t, `<div id="root"><div id="out" class="hover:bg-accent"><p id="a" class="hover:text-primary">alpha</p></div>`+
		`<p id="b" class="bg-muted hover:bg-secondary">beta</p><p id="plain">plain</p></div>`)
	if got := cellAt(t, c, "a"); got.Fg != "" || got.Bg != "" {
		t.Fatalf("not hovered: %+v", got)
	}
	if got := over(t, c, "a"); got.Fg != "primary" || got.Bg != "accent" {
		t.Fatalf("over #a, its own and its parent's hover: %+v", got)
	}
	if got := over(t, c, "b"); got.Bg != "secondary" {
		t.Fatalf("over #b, hover: over the class before it: %+v", got)
	}
	if got := cellAt(t, c, "a"); got.Fg != "" || got.Bg != "" {
		t.Fatalf("#a left: %+v", got)
	}
	// A press on #b, the pointer moved to #a with it down: #b stays lit.
	col, row := cellIn(t, c, "b")
	c.Press(col, row, 0, 0)
	if got := over(t, c, "a"); got.Bg != "" {
		t.Fatalf("over #a with a button down: %+v", got)
	}
	if got := cellAt(t, c, "b"); got.Bg != "secondary" {
		t.Fatalf("#b, pressed: %+v", got)
	}
	col, row = cellIn(t, c, "a")
	c.Release(col, row, 0)
	if got := cellAt(t, c, "a"); got.Bg != "accent" {
		t.Fatalf("released over #a: %+v", got)
	}
	c.Move(-1, 0, 0)
	if got := cellAt(t, c, "a"); got.Bg != "" || len(c.hovered) != 0 {
		t.Fatalf("the pointer off the Screen: %+v, %d hovered", got, len(c.hovered))
	}
	over(t, c, "a")
	c.Focus(true)
	c.Focus(false)
	if got := cellAt(t, c, "a"); got.Bg != "" || len(c.hovered) != 0 {
		t.Fatalf("the Screen without focus: %+v", got)
	}
}

// A hover that changes only colours paints again and lays nothing out;
// one that changes a look lays out again. Dirty says where it changed
// what is drawn.
func TestHoverRepaints(t *testing.T) {
	c := hoverScreen(t, `<div id="root"><p id="col" class="hover:bg-accent">colour</p><p id="look" class="hover:underline">look</p><p id="none">none</p></div>`)
	c.Dirty()
	l := c.layout()
	moveOver(t, c, "none")
	if c.Dirty() {
		t.Fatal("Dirty over an element with no hover")
	}
	moveOver(t, c, "col")
	if c.lay != l {
		t.Fatal("a colour's hover laid the Screen out again")
	}
	if !c.Dirty() {
		t.Fatal("Dirty not over an element whose hover is a colour")
	}
	c.Cells()
	moveOver(t, c, "look")
	if !c.Dirty() {
		t.Fatal("Dirty not over an element whose hover is a look")
	}
	if got := cellAt(t, c, "look"); got.Attr&Underline == 0 {
		t.Fatalf("over a hover:underline: %+v", got)
	}
	moveOver(t, c, "none")
	if !c.Dirty() {
		t.Fatal("Dirty not where a hover's look went")
	}
	if got := cellAt(t, c, "look"); got.Attr&Underline != 0 {
		t.Fatalf("left a hover:underline: %+v", got)
	}
}

// dark: and hover: together, in either order, under the dark scheme only.
func TestHoverDark(t *testing.T) {
	doc := func(scheme string) string {
		return `<div id="root" class="` + scheme + `"><p id="p" class="dark:hover:bg-muted hover:dark:text-primary hover:bg-accent">p</p></div>`
	}
	if got := over(t, hoverScreen(t, doc("dark")), "p"); got.Bg != "muted" || got.Fg != "primary" {
		t.Fatalf("dark: %+v", got)
	}
	if got := over(t, hoverScreen(t, doc("light")), "p"); got.Bg != "accent" || got.Fg != "" {
		t.Fatalf("light: %+v", got)
	}
	// dark: is after hover: in the sheet, at the same specificity: a
	// dark: class is over a hover: class under .dark, and over a plain
	// one before or after it in the attribute.
	c := hoverScreen(t, `<div id="root" class="dark"><p id="p" class="dark:bg-muted hover:bg-accent dark:text-primary text-destructive">p</p><p id="q">q</p></div>`)
	if got := cellAt(t, c, "p"); got.Bg != "muted" || got.Fg != "primary" {
		t.Fatalf("dark, not hovered: %+v", got)
	}
	if got := over(t, c, "p"); got.Bg != "muted" || got.Fg != "primary" {
		t.Fatalf("dark, hovered: %+v", got)
	}
	c = hoverScreen(t, `<div id="root" class="light"><p id="p" class="dark:bg-muted hover:bg-accent">p</p></div>`)
	if got := over(t, c, "p"); got.Bg != "accent" {
		t.Fatalf("light, hovered: %+v", got)
	}
}

// hoverCase is a kit component lit under the pointer: its markup (with
// the element #h), under the dark scheme or not, and the selector of its
// :hover rule as ui's stylesheet has it, compiled.
type hoverCase struct {
	name, markup string
	dark         bool
	selector     string
}

var hoverCases = []hoverCase{
	{"primary button", `<button id="h" class="btn">Go</button>`, false, `:is(.btn:not([data-variant]),.btn[data-variant=primary])`},
	{"outline button", `<button id="h" class="btn" data-variant="outline">Go</button>`, false, `.btn[data-variant=outline]`},
	{"outline button, dark", `<button id="h" class="btn" data-variant="outline">Go</button>`, true, `.btn[data-variant=outline]:is(.dark *)`},
	{"ghost button", `<button id="h" class="btn" data-variant="ghost">Go</button>`, false, `.btn[data-variant=ghost]`},
	{"ghost button, dark", `<button id="h" class="btn" data-variant="ghost">Go</button>`, true, `.btn[data-variant=ghost]:is(.dark *)`},
	{"destructive button", `<button id="h" class="btn" data-variant="destructive">Go</button>`, false, `.btn[data-variant=destructive]`},
	{"destructive button, dark", `<button id="h" class="btn" data-variant="destructive">Go</button>`, true, `.btn[data-variant=destructive]:is(.dark *)`},
	{"secondary link badge", `<a id="h" class="badge" data-variant="secondary" href="#x">New</a>`, false, `.badge[data-variant=secondary]:is(a)`},
	{"outline link badge", `<a id="h" class="badge" data-variant="outline" href="#x">New</a>`, false, `.badge[data-variant=outline]:is(a)`},
	{"destructive link badge", `<a id="h" class="badge" data-variant="destructive" href="#x">New</a>`, false, `.badge[data-variant=destructive]:is(a)`},
	{"ghost badge", `<span id="h" class="badge" data-variant="ghost">New</span>`, false, `.badge[data-variant=ghost]`},
	{"tab", `<div class="tabs"><div role="tablist"><button role="tab" aria-selected="true">One</button><button id="h" role="tab">Two</button></div></div>`, false, `.tabs>[role=tablist]>[role=tab]`},
	{"table row", `<table class="table"><tbody><tr id="h"><td>alpha</td></tr></tbody></table>`, false, `.table tr`},
	{"breadcrumb link", `<nav class="breadcrumb"><ol><li><a id="h" href="#x">Home</a></li></ol></nav>`, false, `.breadcrumb a`},
	{"link item", `<a id="h" class="item" href="#x">An item</a>`, false, `a.item`},
	{"command item", `<div class="command"><div role="menu"><div id="h" role="menuitem">Open</div></div></div>`, false, `.command:not([data-command-initialized]) [role=menuitem]:not([aria-disabled=true]):not(:disabled)`},
	{"select option", `<div class="select"><div data-popover aria-hidden="false"><div role="listbox"><div id="h" role="option">One</div></div></div></div>`, false, `.select:not(select):not([data-select-initialized]) [role=option]:not([aria-disabled=true]):not(:disabled)`},
	{"sidebar link", `<aside class="sidebar"><nav><ul><li><a id="h" href="#x">Home</a></li></ul></nav></aside>`, false, `:is(.sidebar>nav li>:is(a,button),.sidebar>nav li>details>summary)`},
	{"select, dark", `<select id="h" class="select"><option>One</option></select>`, true, `:is(.field>select,select.select):is(.dark *)`},
	{"primary link badge", `<a id="h" class="badge" href="#x">New</a>`, false, `:is(.badge:not([data-variant]),.badge[data-variant=primary]):is(a)`},
	{"dropdown menu item", `<div class="dropdown-menu"><div data-popover aria-hidden="false"><div role="menu"><div id="h" role="menuitem">Copy</div></div></div></div>`, false,
		`.dropdown-menu:not([data-dropdown-menu-initialized]) :is([role=menuitem],[role=menuitemcheckbox],[role=menuitemradio]):not([aria-disabled=true]):not(:disabled)`},
	{"combobox option", `<div class="combobox"><div data-popover aria-hidden="false"><div role="listbox"><div id="h" role="option">One</div></div></div></div>`, false,
		`.combobox:not([data-combobox-initialized]) [role=option]:not([aria-disabled=true]):not(:disabled)`},
	{"clickable toast", `<div class="toaster"><div class="toast"><div id="h" class="toast-content" data-on="click">Saved</div></div></div>`, false, `.toaster .toast .toast-content[data-on~=click]`},
}

// The kit's components lit under the pointer, as a host lights them: the
// colours their :hover rules give them in ui's stylesheet (kit.css,
// compiled from Basecoat's, and toast.css), which a host applies, are the
// roles Cells gives them while hovered.
func TestHoverAsTheHost(t *testing.T) {
	var sheet string
	for _, f := range []string{"../ui/kit.css", "../ui/toast.css"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		sheet += string(b)
	}
	for _, tc := range hoverCases {
		t.Run(tc.name, func(t *testing.T) {
			fg, bg, ok := hoverRule(sheet, tc.selector)
			if !ok {
				t.Fatalf("no :hover rule for %s in ui's stylesheet", tc.selector)
			}
			scheme := "light"
			if tc.dark {
				scheme = "dark"
			}
			c := hoverScreen(t, `<div id="root" class="`+scheme+`">`+tc.markup+`</div>`)
			got := over(t, c, "h")
			if fg != "" && got.Fg != fg || bg != "" && got.Bg != bg {
				t.Fatalf("hovered: %q on %q, the host's %q on %q", got.Fg, got.Bg, fg, bg)
			}
		})
	}
}

// hoverDecl is a colour, background-color or background declaration, and what it
// names: var(--color-x) or var(--x), or a color-mix of one with
// transparent at a percent (x/50).
var hoverDecl = regexp.MustCompile(`(?:^|;)(color|background-color|background):(?:var\(--(?:color-)?([a-z0-9-]+)\)|color-mix\(in oklab,var\(--(?:color-)?([a-z0-9-]+)\)(\d+)%,transparent\))`)

// hoverRule is what the :hover rules of the selector in sheet give a
// host's element, as roles: the last declaration of each property, as a
// cascade has it (the color-mix one where the engine has color-mix).
func hoverRule(sheet, selector string) (fg, bg Role, ok bool) {
	key := selector + ":hover"
	for i := 0; ; {
		j := strings.Index(sheet[i:], key)
		if j < 0 {
			return fg, bg, ok
		}
		i += j + len(key)
		if sheet[i] != '{' && sheet[i] != ',' {
			continue // a longer selector: one of its descendants' rules
		}
		i += strings.IndexByte(sheet[i:], '{') + 1
		end := strings.IndexByte(sheet[i:], '}')
		for _, m := range hoverDecl.FindAllStringSubmatch(sheet[i:i+end], -1) {
			r := Role(m[2])
			if m[3] != "" {
				r = Role(m[3] + "/" + m[4])
			}
			if m[1] == "color" {
				fg = r
			} else {
				bg = r
			}
			ok = true
		}
	}
}

// The kit's :hover look: a link button underlined under the pointer, laid
// out again.
func TestHoverKitLook(t *testing.T) {
	c := hoverScreen(t, `<div id="root"><button id="lk" class="btn" data-variant="link">More</button><p id="none">none</p></div>`)
	if got := cellAt(t, c, "lk"); got.Attr&Underline != 0 {
		t.Fatalf("not hovered: %+v", got)
	}
	if got := over(t, c, "lk"); got.Attr&Underline == 0 {
		t.Fatalf("hovered: %+v", got)
	}
	over(t, c, "none")
	if got := cellAt(t, c, "lk"); got.Attr&Underline != 0 {
		t.Fatalf("left: %+v", got)
	}
}

// A hover's text colour inside muted text takes the element out of it,
// as a colour of its own does (lookOf): a colour that changes a look.
func TestHoverUnmutes(t *testing.T) {
	c := hoverScreen(t, `<div id="root" class="text-muted-foreground"><p id="m" class="hover:text-primary">lit</p><p id="none">none</p></div>`)
	if got := cellAt(t, c, "m"); got.Fg != "muted-foreground" {
		t.Fatalf("not hovered: %+v", got)
	}
	if got := over(t, c, "m"); got.Fg != "primary" {
		t.Fatalf("hovered: %+v", got)
	}
	over(t, c, "none")
	if got := cellAt(t, c, "m"); got.Fg != "muted-foreground" {
		t.Fatalf("left: %+v", got)
	}
}

// What a delta takes out of the document leaves the hovered chain; the
// elements around it stay in it.
func TestHoverPruned(t *testing.T) {
	c := hoverScreen(t, `<div id="root"><div id="out" class="hover:bg-accent"><p id="a">alpha</p></div></div>`)
	over(t, c, "a")
	a := c.byID("a")
	if _, err := c.Write([]byte(hotty.Delta("v", []hotty.Op{hotty.OpRemove("a")}))); err != nil {
		t.Fatal(err)
	}
	if c.hoverOn(a) || !c.hoverOn(c.byID("out")) {
		t.Fatalf("after #a went: %d hovered, #a among them %v", len(c.hovered), c.hoverOn(a))
	}
	if err := CheckFresh(c); err != nil {
		t.Fatal(err)
	}
}
