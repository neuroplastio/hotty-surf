package cells_test

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/cells"
)

// view is a view written by hand: f writes its HTML, with on spreading
// surf.On's attributes.
func view(f func(ctx context.Context, w io.Writer, on func(id string, hs ...surf.Handler) string)) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		f(ctx, w, func(id string, hs ...surf.Handler) string {
			var b strings.Builder
			for _, kv := range surf.On(ctx, id, hs...).Items() {
				fmt.Fprintf(&b, " %s=%q", kv.Key, fmt.Sprint(kv.Value))
			}
			return b.String()
		})
		return nil
	})
}

type on = func(string, ...surf.Handler) string

// rig is a surface on a Screen, driven as a program drives one.
type rig struct {
	t      *testing.T
	s      *surf.Surface
	c      *cells.Screen
	events []string
}

func start(t *testing.T, cols, rows int, v templ.Component) *rig {
	t.Helper()
	r := &rig{t: t, s: surf.New("t", v, "")}
	r.c = cells.New(cols, rows, func(e hotty.Event) {
		r.events = append(r.events, e.Kind+" "+e.Target)
		if err := r.s.Event(e); err != nil {
			t.Errorf("event %s %s: %v", e.Kind, e.Target, err)
		}
	})
	r.s.Attach(r.c, hotty.Caps{})
	doc, err := r.s.Document()
	if err != nil {
		t.Fatal(err)
	}
	r.write(hotty.Doc(r.s.Name(), doc))
	r.write(hotty.Place(r.s.Name(), hotty.Placement{Cols: cols, Rows: rows, Press: hotty.On, Hover: hotty.On}))
	r.settle()
	return r
}

func (r *rig) write(cmd string) {
	r.t.Helper()
	if _, err := r.c.Write([]byte(cmd)); err != nil {
		r.t.Fatal(err)
	}
}

func (r *rig) settle() {
	r.t.Helper()
	select {
	case <-r.s.Pending():
		if err := r.s.Update(); err != nil {
			r.t.Fatal(err)
		}
	default:
	}
}

func (r *rig) key(keys ...string) {
	r.t.Helper()
	for _, k := range keys {
		if !r.c.Key(k) {
			r.t.Fatalf("the Screen didn't take %s", k)
		}
	}
}

var sgrRE = regexp.MustCompile("\x1b\\[[0-9;]*m")

// text is what the Screen shows, without its SGR, each line's trailing
// spaces cut.
func (r *rig) text() string {
	var out []string
	for _, l := range r.c.Lines() {
		out = append(out, strings.TrimRight(sgrRE.ReplaceAllString(l, ""), " "))
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n")
}

// at is the cell where s first shows.
func (r *rig) at(s string) (col, row int) {
	r.t.Helper()
	for y, l := range r.c.Lines() {
		l = sgrRE.ReplaceAllString(l, "")
		if i := strings.Index(l, s); i >= 0 {
			return utf8.RuneCountInString(l[:i]), y
		}
	}
	r.t.Fatalf("%q isn't shown:\n%s", s, r.text())
	return 0, 0
}

// lineOf is the line, with its SGR, where s shows.
func (r *rig) lineOf(s string) string {
	_, y := r.at(s)
	return r.c.Lines()[y]
}

func TestDraw(t *testing.T) {
	v := view(func(ctx context.Context, w io.Writer, on on) {
		fmt.Fprintf(w, `<main class="flex flex-col gap-4">`+
			`<h1 class="font-semibold">Settings</h1>`+
			`<p class="text-muted-foreground">Change what you like.</p>`+
			`<div class="flex gap-2"><label><input type="checkbox"%s checked/> Dark</label><label><input type="radio" name="d"%s/> Wide</label></div>`+
			`<input class="input" placeholder="Your name"%s/>`+
			`<div class="flex gap-2"><button%s>Save</button><button%s>Cancel</button></div>`+
			`<div hidden>secret</div><span class="sr-only">also secret</span><p style="display:none">gone</p>`+
			`</main>`,
			on("dark"), on("wide"), on("name"), on("save", surf.Click(func() {})), on("cancel", surf.Click(func() {})))
	})
	r := start(t, 40, 10, v)
	want := "Settings\n\n" +
		"Change what you like.\n\n" +
		"[x] Dark ( ) Wide\n\n" +
		"Your name\n\n" +
		"[ Save ] [ Cancel ]"
	if got := r.text(); got != want {
		t.Fatalf("drawn:\n%s\nwant:\n%s", got, want)
	}
	if l := r.lineOf("Settings"); !strings.Contains(l, "\x1b[0;1mSettings") {
		t.Errorf("the heading isn't bold: %q", l)
	}
	if l := r.lineOf("Change"); !strings.Contains(l, "\x1b[0;2mChange") {
		t.Errorf("muted text isn't dim: %q", l)
	}
	if l := r.lineOf("Your name"); !strings.Contains(l, "\x1b[0;2;4mYour name") {
		t.Errorf("the placeholder isn't dim in an underlined field: %q", l)
	}
	if n := len(r.c.Lines()); n != 10 {
		t.Errorf("%d lines", n)
	}
}

func TestLayout(t *testing.T) {
	v := view(func(ctx context.Context, w io.Writer, on on) {
		io.WriteString(w, `<div>`+
			`<div class="flex justify-between"><span>left</span><span>right</span></div>`+
			`<div class="flex gap-2"><span class="flex-1">grow</span><b>end</b></div>`+
			`<table><thead><tr><th>Name</th><th>Age</th></tr></thead>`+
			`<tbody><tr><td>Ann</td><td>31</td></tr><tr><td>Bartholomew</td><td>7</td></tr></tbody></table>`+
			`<ul class="list-disc"><li>one</li><li>two</li></ul>`+
			`<p>a long line of words that has to wrap</p>`+
			`</div>`)
	})
	r := start(t, 20, 14, v)
	want := "left           right\n" +
		"grow             end\n" +
		"Name         Age\n" +
		"────────────────\n" +
		"Ann          31\n" +
		"Bartholomew  7\n" +
		"• one\n" +
		"• two\n" +
		"a long line of words\n" +
		"that has to wrap"
	if got := r.text(); got != want {
		t.Fatalf("drawn:\n%s\nwant:\n%s", got, want)
	}
}

func TestTabMovesFocus(t *testing.T) {
	var log []string
	v := view(func(ctx context.Context, w io.Writer, on on) {
		fmt.Fprintf(w, `<div><button%s>One</button><a href="#x">plain</a><button%s>Two</button><button disabled>Off</button></div>`,
			on("one", surf.Click(func() {}), surf.Focus(func(in bool) { log = append(log, fmt.Sprint("one ", in)) })),
			on("two", surf.Click(func() {}), surf.Focus(func(in bool) { log = append(log, fmt.Sprint("two ", in)) })))
	})
	r := start(t, 40, 3, v)
	r.key("tab")
	if strings.Join(log, ",") != "one true" {
		t.Fatalf("after tab: %v", log)
	}
	if l := r.lineOf("One"); !strings.Contains(l, "\x1b[0;7m[ One ]") {
		t.Errorf("focus isn't reverse: %q", l)
	}
	r.key("tab", "tab")
	if strings.Join(log, ",") != "one true,one false,two true" {
		t.Fatalf("after tab tab tab: %v", log)
	}
	r.key("shift+tab", "shift+tab")
	if strings.Join(log, ",") != "one true,one false,two true,two false,one true" {
		t.Fatalf("after shift+tab: %v", log)
	}
	// Past the first: nothing is focused, and the next tab starts again.
	r.events = nil
	r.key("shift+tab")
	if strings.Join(r.events, ",") != "blur " {
		t.Fatalf("before the first: %v", r.events)
	}
	r.key("tab")
	if r.events[len(r.events)-1] != "focus one" {
		t.Fatalf("tab again: %v", r.events)
	}
}

func TestTypingReachesInput(t *testing.T) {
	name, committed := "", ""
	v := view(func(ctx context.Context, w io.Writer, on on) {
		fmt.Fprintf(w, `<div><input type="text"%s value="%s"/><p>Hello, %s!</p><button%s>go</button></div>`,
			on("name", surf.Input(func(v string) { name = v })), name, name, on("go", surf.Click(func() { committed = name })))
	})
	r := start(t, 40, 4, v)
	r.key("tab")
	r.key("a", "c", "arrow_left", "b")
	if name != "abc" {
		t.Fatalf("name = %q", name)
	}
	r.key("home", "x", "end", "shift+y", "backspace", "y", "home", "delete")
	if name != "abcy" {
		t.Fatalf("name = %q", name)
	}
	if !strings.Contains(r.text(), "Hello, abcy!") {
		t.Fatalf("the delta isn't drawn:\n%s", r.text())
	}
	if col, row, ok := r.c.Cursor(); !ok || col != 0 || row != 0 {
		t.Errorf("caret at %d,%d %v", col, row, ok)
	}
	if !r.c.Paste("zz ") || name != "zz abcy" {
		t.Fatalf("after a paste: %q", name)
	}
	if col, _, _ := r.c.Cursor(); col != 3 {
		t.Errorf("caret at col %d after the paste", col)
	}
	// Leaving the field commits it.
	r.events = nil
	r.key("tab")
	if strings.Join(r.events, ",") != "change name,focus go" {
		t.Fatalf("leaving the field: %v", r.events)
	}
	r.key("enter")
	if committed != "zz abcy" {
		t.Fatalf("committed %q", committed)
	}
}

func TestButtonByEnterAndClick(t *testing.T) {
	n := 0
	v := view(func(ctx context.Context, w io.Writer, on on) {
		fmt.Fprintf(w, `<div><p>Count</p><div class="flex gap-2"><button%s>Added %d</button><span%s>more</span></div></div>`,
			on("add", surf.Click(func() { n++ })), n, on("more", surf.Click(func() { n += 10 })))
	})
	r := start(t, 40, 4, v)
	r.key("tab", "enter", "space")
	if n != 2 {
		t.Fatalf("n = %d", n)
	}
	if !strings.Contains(r.text(), "[ Added 2 ]") {
		t.Fatalf("not redrawn:\n%s", r.text())
	}
	r.events = nil
	col, row := r.at("Added")
	if !r.c.Click(col, row) {
		t.Fatal("the click wasn't taken")
	}
	if n != 3 || !strings.Contains(r.text(), "[ Added 3 ]") {
		t.Fatalf("n = %d:\n%s", n, r.text())
	}
	if strings.Join(r.events, ",") != "press add,click add" {
		t.Fatalf("events: %v", r.events)
	}
	// An element with data-on~=click reports its clicks; it takes no focus,
	// so the button's is lost.
	r.events = nil
	col, row = r.at("more")
	r.c.Click(col+1, row)
	if n != 13 || strings.Join(r.events, ",") != "press more,blur ,click more" {
		t.Fatalf("n = %d, events %v", n, r.events)
	}
	// Outside the Screen is not the surface's.
	if r.c.Click(40, 0) || r.c.Click(0, 4) {
		t.Fatal("a click outside the Screen was taken")
	}
}

func TestCheckbox(t *testing.T) {
	ticked := false
	v := view(func(ctx context.Context, w io.Writer, o on) {
		checked := ""
		if ticked {
			checked = " checked"
		}
		fmt.Fprintf(w, `<div><label><input type="checkbox"%s%s/> Notify</label><p>%v</p></div>`,
			o("notify", surf.Check(func(c bool) { ticked = c })), checked, ticked)
	})
	r := start(t, 40, 3, v)
	col, row := r.at("Notify")
	r.c.Click(col, row) // on the label
	if !ticked || !strings.Contains(r.text(), "[x] Notify") || !strings.Contains(r.text(), "true") {
		t.Fatalf("ticked = %v:\n%s", ticked, r.text())
	}
	r.key("space") // the box has the keyboard
	if ticked || !strings.Contains(r.text(), "[ ] Notify") {
		t.Fatalf("ticked = %v:\n%s", ticked, r.text())
	}
}

func TestFieldSetBySurface(t *testing.T) {
	q := ""
	v := view(func(ctx context.Context, w io.Writer, on on) {
		fmt.Fprintf(w, `<div><input type="text"%s value="%s"/><button%s>Clear</button><button%s>Fill</button></div>`,
			on("q", surf.Input(func(v string) { q = v })), q,
			on("clear", surf.Click(func() { q = "" })), on("fill", surf.Click(func() { q = "filled" })))
	})
	r := start(t, 60, 2, v)
	r.key("tab", "a", "b")
	col, row := r.at("Clear")
	r.c.Click(col, row)
	if q != "" || strings.Contains(r.text(), "ab") {
		t.Fatalf("q = %q:\n%s", q, r.text())
	}
	col, row = r.at("Fill")
	r.c.Click(col, row)
	if !strings.Contains(r.text(), "filled") {
		t.Fatalf("the field isn't set:\n%s", r.text())
	}
	// The field's rev is the surface's: typing goes on from the value set.
	r.key("shift+tab", "shift+tab", "end", "!")
	if q != "filled!" {
		t.Fatalf("q = %q", q)
	}
}

func TestKeysTheScreenDoesntUse(t *testing.T) {
	var keys []string
	v := view(func(ctx context.Context, w io.Writer, on on) {
		fmt.Fprintf(w, `<div%s><button%s>b</button><input%s data-keys="enter=program arrow_up=program"/></div>`,
			on("page", surf.Key(func(k string) bool { keys = append(keys, k); return true })),
			on("b", surf.Click(func() {})), on("f", surf.Input(func(string) {})))
	})
	r := start(t, 40, 2, v)
	for _, k := range []string{"ctrl+s", "x", "enter", "arrow_down"} {
		if r.c.Key(k) {
			t.Errorf("%s taken with nothing focused", k)
		}
	}
	r.key("tab")
	for _, k := range []string{"x", "escape", "ctrl+s", "arrow_down"} {
		if r.c.Key(k) {
			t.Errorf("%s taken by a button", k)
		}
	}
	r.key("tab")
	for _, k := range []string{"enter", "arrow_up", "escape", "ctrl+q"} {
		if r.c.Key(k) {
			t.Errorf("%s taken by the field", k)
		}
	}
	// What the Screen doesn't take, the program offers the surface.
	if taken, err := r.s.Key("enter"); !taken || err != nil || keys[len(keys)-1] != "enter" {
		t.Fatalf("the surface's key: %v %v %v", taken, err, keys)
	}
}

func TestResizeAndWheel(t *testing.T) {
	v := view(func(ctx context.Context, w io.Writer, on on) {
		io.WriteString(w, `<div>`)
		for i := range 20 {
			fmt.Fprintf(w, `<p>line %d</p>`, i)
		}
		io.WriteString(w, `<p>the last words of all</p></div>`)
	})
	r := start(t, 30, 5, v)
	if !r.c.Wheel(0, 0, 3) || !strings.HasPrefix(r.text(), "line 3\n") {
		t.Fatalf("after the wheel:\n%s", r.text())
	}
	if !r.c.Wheel(0, 0, 100) || !strings.HasSuffix(r.text(), "the last words of all") {
		t.Fatalf("at the end:\n%s", r.text())
	}
	if r.c.Wheel(0, 0, 1) {
		t.Fatal("scrolled past the end")
	}
	r.c.Wheel(0, 0, -100)
	r.c.Resize(10, 4)
	lines := r.c.Lines()
	if len(lines) != 4 || lines[0] != "line 0    " {
		t.Fatalf("resized: %q", lines)
	}
	r.c.Wheel(0, 0, 100)
	if !strings.HasSuffix(r.text(), "line 19\nthe last\nwords of\nall") {
		t.Fatalf("wrapped at 10:\n%s", r.text())
	}
}

func TestFocusAndBlur(t *testing.T) {
	var log []string
	q := ""
	v := view(func(ctx context.Context, w io.Writer, on on) {
		surf.Autofocus(ctx, "q")
		fmt.Fprintf(w, `<div%s><input%s value="%s"/></div>`,
			on("box", surf.Focus(func(in bool) { log = append(log, fmt.Sprint("focus ", in)) }), surf.Outside(func() { log = append(log, "outside") })),
			on("q", surf.Input(func(v string) { q = v })), q)
	})
	r := start(t, 30, 2, v)
	// The view's autofocus reached the Screen as a=focus.
	r.key("h")
	if _, _, ok := r.c.Cursor(); !ok {
		t.Fatal("no caret in the autofocused field")
	}
	r.events = nil
	r.c.Focus(false)
	if strings.Join(r.events, ",") != "change q,blur " || !strings.Contains(strings.Join(log, ","), "focus false") {
		t.Fatalf("events %v, log %v", r.events, log)
	}
	if r.c.Key("x") {
		t.Fatal("a key taken without the keyboard")
	}
	r.events = nil
	r.c.Focus(true)
	if strings.Join(r.events, ",") != "focus q" {
		t.Fatalf("back: %v", r.events)
	}
	r.key("i")
	if !strings.Contains(r.text(), "hi") {
		t.Fatalf("typing after the focus came back:\n%s", r.text())
	}
}

func TestBadDeltaIsAnError(t *testing.T) {
	r := start(t, 20, 2, view(func(ctx context.Context, w io.Writer, on on) { io.WriteString(w, `<p>x</p>`) }))
	if _, err := r.c.Write([]byte(hotty.Delta("t", []hotty.Op{hotty.OpText("nope", "y")}))); err == nil {
		t.Fatal("a delta with no target was applied")
	}
}

func TestSelectAndTextarea(t *testing.T) {
	size, notes := "m", ""
	v := view(func(ctx context.Context, w io.Writer, on on) {
		fmt.Fprintf(w, `<div class="flex flex-col"><select%s>`, on("size", surf.Input(func(v string) { size = v })))
		for _, o := range []string{"s", "m", "l"} {
			sel := ""
			if o == size {
				sel = " selected"
			}
			fmt.Fprintf(w, `<option value="%s"%s>Size %s</option>`, o, sel, strings.ToUpper(o))
		}
		fmt.Fprintf(w, `</select><textarea rows="3"%s>%s</textarea></div>`, on("notes", surf.Input(func(v string) { notes = v })), notes)
	})
	r := start(t, 20, 4, v)
	if !strings.Contains(r.text(), "[Size M ▾]") {
		t.Fatalf("the select:\n%s", r.text())
	}
	r.key("tab", "arrow_down")
	if size != "l" || !strings.Contains(r.text(), "[Size L ▾]") {
		t.Fatalf("size %q:\n%s", size, r.text())
	}
	r.key("tab", "o", "n", "e", "enter", "t", "w", "o")
	if notes != "one\ntwo" || !strings.Contains(r.text(), "one\ntwo") {
		t.Fatalf("notes %q:\n%s", notes, r.text())
	}
	if col, row, ok := r.c.Cursor(); !ok || col != 3 || row != 2 {
		t.Fatalf("caret at %d,%d %v", col, row, ok)
	}
}

func TestRemovingTheFocusedElement(t *testing.T) {
	items := []string{"a", "b"}
	v := view(func(ctx context.Context, w io.Writer, on on) {
		io.WriteString(w, `<ul>`)
		for i, it := range items {
			fmt.Fprintf(w, `<li><button%s>drop %s</button></li>`, on("drop-"+it, surf.Click(func() {
				items = append(items[:i:i], items[i+1:]...)
			})), it)
		}
		io.WriteString(w, `</ul>`)
	})
	r := start(t, 20, 3, v)
	r.key("tab", "enter")
	if r.text() != "[ drop b ]" {
		t.Fatalf("after the drop:\n%s", r.text())
	}
	// The button that had the keyboard is gone: tab starts again.
	r.key("tab", "enter")
	if r.text() != "" || len(items) != 0 {
		t.Fatalf("items %v:\n%s", items, r.text())
	}
}

func TestStaleFieldChangesNothing(t *testing.T) {
	r := start(t, 20, 2, view(func(ctx context.Context, w io.Writer, on on) {
		fmt.Fprintf(w, `<input%s value="x"/>`, on("f"))
	}))
	v := "y"
	r.write(hotty.SetField("t", "f", hotty.FieldChange{Rev: 5, Value: &v}))
	if r.text() != "x" {
		t.Fatalf("a stale change was made:\n%s", r.text())
	}
	r.write(hotty.SetField("t", "f", hotty.FieldChange{Rev: 0, Value: &v}))
	if r.text() != "y" {
		t.Fatalf("the change wasn't made:\n%s", r.text())
	}
	on := true
	if _, err := r.c.Write([]byte(hotty.SetField("t", "f", hotty.FieldChange{Rev: 0, Checked: &on}))); err == nil {
		t.Fatal("a text field was checked")
	}
}
