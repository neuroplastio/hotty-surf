package cells

import (
	"fmt"
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
// around it match hover:, the others don't; with a button down it follows
// the pointer; off the Screen and on losing focus there is none.
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
	// A press on #b, the pointer moved to #a with it down: the hover
	// follows the pointer, as a host's does.
	col, row := cellIn(t, c, "b")
	c.Press(col, row, 0, 0)
	if got := cellAt(t, c, "b"); got.Bg != "secondary" {
		t.Fatalf("#b, pressed: %+v", got)
	}
	if got := over(t, c, "a"); got.Bg != "accent" {
		t.Fatalf("over #a with a button down: %+v", got)
	}
	if got := cellAt(t, c, "b"); got.Bg != "muted" {
		t.Fatalf("#b, left with a button down: %+v", got)
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

// hoveredIDs are the ids of the hovered chain, innermost first, "-" for
// an element with none.
func hoveredIDs(c *Screen) string {
	var s []string
	for _, n := range c.hovered {
		s = append(s, cmpOr(attr(n, "id"), "-"))
	}
	return strings.Join(s, " ")
}

func cmpOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// holdFresh holds the Screen to a fresh pass, the chain to what is under the
// pointer among it.
func holdFresh(t *testing.T, c *Screen, at string) {
	t.Helper()
	if err := CheckFresh(c); err != nil {
		t.Fatalf("%s: %v", at, err)
	}
}

// What is under a still pointer moves, with a wheel, a delta, a resize, a
// key that scrolls: the chain is what is under it after, as a host hits
// the pointer again after each pass.
func TestHoverFollowsWhatMoves(t *testing.T) {
	var d strings.Builder
	d.WriteString(`<div id="root">`)
	for i := range 30 {
		fmt.Fprintf(&d, `<p id="r%d" class="hover:bg-accent">row %d</p>`, i, i)
	}
	d.WriteString(`</div>`)
	c := hoverScreen(t, d.String())
	c.Move(3, 0, 0)
	c.Wheel(3, 0, 1)
	if got := c.Cells()[0][3]; got.Bg != "accent" || hoveredIDs(c) != "r1 root -" {
		t.Fatalf("wheeled: %s, %+v", hoveredIDs(c), got)
	}
	holdFresh(t, c, "wheeled")
	c.Write([]byte(hotty.Delta("v", []hotty.Op{hotty.OpRemove("r1")})))
	if got := c.Cells()[0][3]; got.Bg != "accent" || hoveredIDs(c) != "r2 root -" {
		t.Fatalf("the hovered row taken out: %s, %+v", hoveredIDs(c), got)
	}
	holdFresh(t, c, "the hovered row taken out")

	c = hoverScreen(t, `<div id="root"><p id="a" class="hover:bg-accent">alpha</p><p id="z">zz</p></div>`)
	over(t, c, "a")
	c.Write([]byte(hotty.Delta("v", []hotty.Op{hotty.OpBefore("a", `<p id="n" class="hover:bg-secondary">new</p>`)})))
	if got := c.Cells()[0][2]; got.Bg != "secondary" || cellAt(t, c, "a").Bg != "" || hoveredIDs(c) != "n root -" {
		t.Fatalf("a row put in above: %s, %+v", hoveredIDs(c), got)
	}
	holdFresh(t, c, "a row put in above")
	c.Write([]byte(hotty.Delta("v", []hotty.Op{hotty.OpReplace("n", `<p id="n" class="hover:bg-accent">newer</p>`)})))
	if got := c.Cells()[0][2]; got.Bg != "accent" || c.hovered[0] != c.byID("n") {
		t.Fatalf("the hovered replaced: %s, %+v", hoveredIDs(c), got)
	}
	holdFresh(t, c, "the hovered replaced")

	c = hoverScreen(t, `<div id="root" class="flex"><span id="a" class="grow hover:bg-accent">aaaa</span><span id="b" class="grow hover:bg-secondary">bbbb</span></div>`)
	c.Move(28, 0, 0)
	c.Resize(50, 12)
	if got := c.Cells()[0][28]; got.Bg != "secondary" || hoveredIDs(c) != "b root -" {
		t.Fatalf("resized: %s, %+v", hoveredIDs(c), got)
	}
	holdFresh(t, c, "resized")
	c.Resize(20, 12)
	c.Cells()
	if hoveredIDs(c) != "" {
		t.Fatalf("resized narrower than the pointer's column: %s", hoveredIDs(c))
	}
}

// CheckFresh holds the chain to what is under the pointer.
func TestCheckFreshHoldsTheChain(t *testing.T) {
	c := hoverScreen(t, `<div id="root"><p id="a" class="hover:bg-accent">alpha</p><p id="b">beta</p></div>`)
	moveOver(t, c, "a")
	c.setHover(c.byID("b"))
	if err := CheckFresh(c); err == nil || !strings.Contains(err.Error(), "under the pointer") {
		t.Fatalf("a chain not under the pointer: %v", err)
	}
}

// A press, the focus lost and given back with the button still down: the
// next move lights what it is over.
func TestHoverAfterAPressAndTheFocus(t *testing.T) {
	c := hoverScreen(t, `<div id="root"><p id="a" class="hover:bg-accent">alpha</p><p id="b" class="hover:bg-secondary">beta</p></div>`)
	c.Focus(true)
	col, row := cellIn(t, c, "a")
	c.Press(col, row, 0, 0)
	c.Focus(false)
	if hoveredIDs(c) != "" || c.tip != nil {
		t.Fatalf("the focus lost: %s", hoveredIDs(c))
	}
	c.Focus(true)
	c.Cells()
	if hoveredIDs(c) != "" {
		t.Fatalf("the focus back, the pointer not moved: %s", hoveredIDs(c))
	}
	if got := over(t, c, "b"); got.Bg != "secondary" {
		t.Fatalf("over #b: %+v, %s", got, hoveredIDs(c))
	}
}

// A document again, of the same shape or another: the chain and the
// tooltip are what is under the pointer in it, as on a fresh Screen with
// the pointer there.
func TestHoverNewDocument(t *testing.T) {
	doc := `<div id="root"><p id="a" class="hover:bg-accent" data-tooltip="tip">alpha</p><p id="b">beta</p><p id="c">gamma</p></div>`
	c := hoverScreen(t, doc)
	col, row := moveOver(t, c, "a")
	if c.tip == nil || c.Cells()[row][col].Bg != "accent" {
		t.Fatalf("over #a: tip %v, %+v", c.tip != nil, c.Cells()[row][col])
	}
	want := c.Lines()
	c.Write([]byte(hotty.Doc("v", doc)))
	holdFresh(t, c, "the same document")
	if got := c.Lines(); strings.Join(got, "\n") != strings.Join(want, "\n") || c.tip == nil {
		t.Fatalf("the same document: tip %v\n%s\nwant\n%s", c.tip != nil, strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	f := hoverScreen(t, doc)
	f.Move(col, row, 0)
	if got := strings.Join(f.Lines(), "\n"); got != strings.Join(want, "\n") {
		t.Fatalf("a fresh Screen, the pointer there:\n%s", got)
	}
	c.Write([]byte(hotty.Doc("v", `<div id="root"><p id="x" class="hover:bg-secondary">xx</p><p>more</p></div>`)))
	holdFresh(t, c, "another document")
	if c.Cells()[0][2].Bg != "secondary" || c.tip != nil {
		t.Fatalf("another document: %s, %+v", hoveredIDs(c), c.Cells()[0][2])
	}
}

// The tooltip is the chain's: it follows the pointer with a button down,
// goes with the focus, and with its data-tooltip.
func TestHoverTooltipIsTheChains(t *testing.T) {
	c := hoverScreen(t, `<div id="root"><p>top</p><p>second</p><p id="a" data-tooltip="a tip">alpha</p><p id="b" data-tooltip="b tip">beta</p></div>`)
	col, row := moveOver(t, c, "a")
	c.Press(col, row, 0, 0)
	moveOver(t, c, "b")
	if c.tip != c.byID("b") {
		t.Fatalf("moved to #b with a button down: tip %v", idOf2(c.tip))
	}
	c.Write([]byte(hotty.Delta("v", []hotty.Op{hotty.OpAttr("b", "data-tooltip", "")})))
	c.Cells()
	if c.tip != nil {
		t.Fatalf("#b's tooltip taken away: tip %v", idOf2(c.tip))
	}
	holdFresh(t, c, "#b's tooltip taken away")
	c.Focus(true)
	c.Write([]byte(hotty.Delta("v", []hotty.Op{hotty.OpAttr("b", "data-tooltip", "again")})))
	c.Cells()
	if c.tip != c.byID("b") {
		t.Fatalf("#b's tooltip back: tip %v", idOf2(c.tip))
	}
	c.Focus(false)
	if c.tip != nil {
		t.Fatalf("the focus lost: tip %v", idOf2(c.tip))
	}
}

// A pointer-events:none rule of the kit's sheet, mirrored (kitPointer):
// with the pointer over #h, the chain starts at want.
type pointerCase struct {
	selector, doc, want string
}

var pointerCases = []pointerCase{
	{`.btn:disabled`, `<button id="h" class="btn" disabled>Go</button>`, ""},
	{`.accordion>details[aria-disabled=true]>summary,.accordion>details>summary[aria-disabled=true]`,
		`<div class="accordion"><details aria-disabled="true"><summary id="h">One</summary>body</details></div>`, ""},
	{`.combobox>input[role=combobox]:disabled`, `<div class="combobox"><input id="h" role="combobox" disabled value="abc"></div>`, ""},
	{`:is(.combobox .input-group input[role=combobox],.combobox [data-popover] input[role=combobox]):disabled`,
		`<div class="combobox"><div class="input-group"><input id="h" role="combobox" disabled value="abc"></div></div>`, ""},
	{`.combobox [data-clear]:disabled`, `<div class="combobox"><button id="h" data-clear disabled>Clear</button></div>`, ""},
	{`.combobox [role=option]:disabled,.combobox [role=option][aria-disabled=true]`,
		`<div class="combobox"><div data-popover aria-hidden="false"><div role="listbox"><div id="h" role="option" aria-disabled="true">One</div></div></div></div>`, ""},
	{`.combobox-chips>input[role=combobox]:disabled`, `<div class="combobox-chips"><input id="h" role="combobox" disabled value="abc"></div>`, ""},
	{`.command-dialog>.command>:is(button,form[method=dialog]>button):disabled,.command-dialog>.command>:is(button,form[method=dialog]>button) svg`,
		`<div class="command-dialog"><div class="command"><button id="h" disabled>Close</button></div></div>`, ""},
	{`.command [role=menuitem]:disabled,.command [role=menuitem][aria-disabled=true]`,
		`<div class="command"><div role="menu"><div id="h" role="menuitem" aria-disabled="true">Copy</div></div></div>`, ""},
	{`.command [role=menuitem][data-disabled=true]`,
		`<div class="command"><div role="menu"><div id="h" role="menuitem" data-disabled="true">Copy</div></div></div>`, ""},
	{`.dialog>*>:is(button,form[method=dialog]>button):disabled,.dialog>*>:is(button,form[method=dialog]>button) svg`,
		`<div class="dialog"><div><form method="dialog"><button id="h" disabled>Close</button></form></div></div>`, ""},
	{`.dropdown-menu :is([role=menuitem],[role=menuitemcheckbox],[role=menuitemradio]):disabled,.dropdown-menu :is([role=menuitem],[role=menuitemcheckbox],[role=menuitemradio])[aria-disabled=true]`,
		`<div class="dropdown-menu"><div data-popover aria-hidden="false"><div role="menu"><div id="h" role="menuitem" aria-disabled="true">Copy</div></div></div></div>`, ""},
	{`.dropdown-menu [data-indicator]`, `<div class="dropdown-menu"><span id="h" data-indicator>Mark</span></div>`, ""},
	{`.kbd`, `<span id="h" class="kbd">Ctrl</span>`, ""},
	{`:is(.field>label,.field>section>label,.label):is(:where(.peer):disabled~*),:is(.field>label,.field>section>label,.label):has(>:disabled),:is(.field>label,.field>section>label,.label):has(+:disabled)`,
		`<div class="field"><label id="h">Name</label><input disabled value="x"></div>`, ""},
	{`:is(.field>input[type=range],.input[type=range]):disabled`, `<input id="h" class="input" type="range" disabled>`, ""},
	{`.select:not(select)>button:disabled`, `<div class="select"><button id="h" disabled>Pick</button></div>`, ""},
	{`.select:not(select) [role=option]:disabled,.select:not(select) [role=option][aria-disabled=true]`,
		`<div class="select"><div data-popover aria-hidden="false"><div role="listbox"><div id="h" role="option" aria-disabled="true">One</div></div></div></div>`, ""},
	{`:is(.sidebar>nav>section ul li>a,.sidebar>nav>section ul li>button,.sidebar>nav>section ul li>details>summary):disabled,:is(.sidebar>nav>section ul li>a,.sidebar>nav>section ul li>button,.sidebar>nav>section ul li>details>summary)[aria-disabled=true]`,
		`<div class="sidebar"><nav><section><ul><li><a id="h" href="#x" aria-disabled="true">Home</a></li></ul></section></nav></div>`, ""},
	{`.tabs>[role=tablist]>[role=tab]:disabled,.tabs>[role=tablist]>[role=tab][aria-disabled=true]`,
		`<div class="tabs"><div role="tablist"><button role="tab" aria-selected="true">One</button><button id="h" role="tab" disabled>Two</button></div></div>`, ""},
	{`.toaster`, `<div id="h" class="toaster">gap</div>`, ""},
	{`.chart-tooltip`, `<div id="h" class="chart-tooltip">3 items</div>`, ""},
	{`.combobox>svg,.combobox-trigger-icon`, `<div class="combobox"><span id="h" class="combobox-trigger-icon">v</span></div>`, ""},
}

// The pointer-events:none rules of the kit's sheet the chain doesn't
// read, and why.
var pointerNotRead = map[string]string{
	`.accordion>details>summary>svg:last-child`: "an icon is its control's glyph (iconLabel), not an element drawn",
	`.badge>svg`: "an icon",
	`.btn svg`:   "an icon",
	`.button-group>:is(span,label,output) svg`: "an icon",
	`.combobox [role=option] svg`:              "an icon",
	`.command [role=menuitem] svg`:             "an icon",
	`.alert-dialog>*>header>figure svg`:        "an icon",
	`.dropdown-menu :is([role=menuitem],[role=menuitemcheckbox],[role=menuitemradio]) svg`: "an icon",
	`.empty figure svg`:                     "an icon",
	`.input-group>svg`:                      "an icon",
	`.item>figure svg`:                      "an icon",
	`.select:not(select)>button svg`:        "an icon",
	`.select:not(select) [role=option] svg`: "an icon",
	`.tabs>[role=tablist]>[role=tab] svg`:   "an icon",
	`:is(.field>input[type=checkbox][role=switch],.input[type=checkbox][role=switch]):before`: "a pseudo-element",
	`[data-tooltip]:before`: "a pseudo-element: the tooltip's cells are what is under it",
}

// fieldDisabledRule is the kit's rule for a disabled field's input, too
// long to list as its selector: it starts so.
const fieldDisabledRule = `:is(.field>input[type=text]:not(:is(.command>header input,.dialog>*>header input))`

// pointerRules are the selectors of the rules in sheet that set
// pointer-events:none.
func pointerRules(sheet string) []string {
	var out []string
	for i := 0; ; {
		j := strings.Index(sheet[i:], "pointer-events:none")
		if j < 0 {
			return out
		}
		j += i
		a := strings.LastIndexByte(sheet[:j], '}')
		b := strings.IndexByte(sheet[a+1:], '{')
		out = append(out, sheet[a+1:a+1+b])
		i = j + 1
	}
}

// An element the kit's sheet, a modal or its own style takes the pointer
// from isn't hovered: the chain starts at the nearest one around it that
// takes it, as a host's hit test passes over it. Every pointer-events:none
// rule of the sheet is mirrored or named as not read.
func TestHoverTakesNoPointer(t *testing.T) {
	b, err := os.ReadFile("../ui/kit.css")
	if err != nil {
		t.Fatal(err)
	}
	listed := map[string]bool{}
	for _, pc := range pointerCases {
		listed[pc.selector] = true
	}
	for _, sel := range pointerRules(string(b)) {
		if !listed[sel] && pointerNotRead[sel] == "" && !strings.HasPrefix(sel, fieldDisabledRule) {
			t.Errorf("pointer-events:none on %s: neither mirrored (pointerCases) nor named (pointerNotRead)", sel)
		}
	}
	cases := append(pointerCases,
		pointerCase{fieldDisabledRule, `<div class="field"><input id="h" type="text" disabled value="abc"></div>`, ""},
		pointerCase{"a modal's inert", `<div data-hotty-modal-inert style="pointer-events:none!important"><p id="h" class="hover:bg-accent">behind</p></div>`, "p"},
		pointerCase{"a class", `<div class="pointer-events-none"><p id="h">through</p></div>`, "p"},
		pointerCase{"auto inside none", `<div style="pointer-events:none"><p id="h" class="pointer-events-auto">back</p></div>`, "h"},
		pointerCase{"a toast in the toaster", `<div class="toaster"><div id="h" class="toast"><div class="toast-content">Saved</div></div></div>`, "h"},
	)
	for _, pc := range cases {
		c := hoverScreen(t, `<div id="root"><div id="p">`+pc.doc+`</div><p id="none">none</p></div>`)
		col, row := cellIn(t, c, "h")
		c.Move(col, row, 0)
		holdFresh(t, c, pc.selector)
		h := c.byID("h")
		want := h.Parent
		if pc.want != "" {
			want = c.byID(pc.want)
		}
		if len(c.hovered) == 0 || !within(c.hovered[0], want) || pc.want != "h" && c.hovered[0] != want {
			t.Errorf("%s: over #h, the chain is %s, want it from #%s", pc.selector, hoveredIDs(c), cmpOr(pc.want, "h's parent"))
		}
	}
	// Lit as a host lights them: a disabled button keeps its look.
	c := hoverScreen(t, `<div id="root"><button id="h" class="btn" disabled>Go</button><p id="none">x</p></div>`)
	if got, was := over(t, c, "h"), cellAt(t, c, "h"); got != was || got.Bg != "primary" {
		t.Errorf("a disabled button hovered: %+v, not hovered %+v", got, was)
	}
	c = hoverScreen(t, `<div id="root"><div data-hotty-modal-inert style="pointer-events:none!important"><p id="h" class="hover:bg-accent">behind</p></div></div>`)
	if got := over(t, c, "h"); got.Bg != "" {
		t.Errorf("behind a modal: %+v", got)
	}
}
