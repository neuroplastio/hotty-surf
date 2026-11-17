package cells_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/cells"
)

// logW is the surface's writer: it logs each command's a= (and a delta's
// ops), and passes it to the Screen.
type logW struct {
	c   *cells.Screen
	log []string
}

func (w *logW) Write(p []byte) (int, error) {
	var sc hotty.Scanner
	var dec hotty.Decoder
	for _, sg := range append(sc.Feed(p), sc.Flush()...) {
		if sg.Kind != hotty.SegmentOSC {
			continue
		}
		m, r := dec.Feed(string(sg.Data))
		if r != hotty.Complete {
			continue
		}
		a := hotty.Get(m.Control, "a")
		if a == "delta" {
			ops, _ := hotty.DeltaOf(m)
			for _, o := range ops {
				a += " " + o.Op + ":" + o.Target
			}
		}
		w.log = append(w.log, a)
	}
	return w.c.Write(p)
}

// logged starts a surface on a Screen through a logW.
func logged(t *testing.T, cols, rows int, v func(ctx context.Context, w io.Writer, on on)) (*rig, *logW) {
	s := surf.New("t", view(v), "")
	c := cells.New(cols, rows, func(e hotty.Event) {
		if err := s.Event(e); err != nil {
			t.Errorf("event %s %s: %v", e.Kind, e.Target, err)
		}
	})
	lw := &logW{c: c}
	s.Attach(lw, hotty.Caps{})
	doc, err := s.Document()
	if err != nil {
		t.Fatal(err)
	}
	lw.Write([]byte(hotty.Doc("t", doc)))
	return &rig{t: t, s: s, c: c}, lw
}

// The surface sends inner where text beside a field changes: the field
// keeps the keyboard and its value (SPEC §6.1 morphs).
func TestInnerKeepsTheFocusedField(t *testing.T) {
	val := ""
	r, lw := logged(t, 40, 2, func(ctx context.Context, w io.Writer, on on) {
		fmt.Fprintf(w, `<div>typed %d <input%s value="%s"/></div>`, len(val),
			on("f", surf.Input(func(s string) { val = s })), val)
	})
	r.key("tab", "a")
	if !strings.Contains(strings.Join(lw.log, ","), "inner:") {
		t.Fatalf("no inner was sent: %v", lw.log)
	}
	taken := r.c.Key("b")
	_, _, caret := r.c.Cursor()
	if !taken || !caret || val != "ab" {
		t.Fatalf("the field lost the keyboard: b taken %v, caret %v, val %q", taken, caret, val)
	}
}

// After an inner remade a field, its rev is still the user's: the
// program's a=field is not stale.
func TestInnerKeepsTheRev(t *testing.T) {
	val := ""
	r, lw := logged(t, 40, 2, func(ctx context.Context, w io.Writer, on on) {
		state := "empty"
		if val != "" {
			state = "set"
		}
		fmt.Fprintf(w, `<div><div>%s <input%s value="%s"/></div><button%s>Fill</button></div>`, state,
			on("f", surf.Input(func(s string) { val = s })), val, on("fill", surf.Click(func() { val = "filled" })))
	})
	r.key("tab", "a")
	col, row := r.at("Fill")
	r.c.Click(col, row)
	if !strings.Contains(r.text(), "filled") {
		t.Fatalf("the program's value isn't shown: %q; the surface wrote %v", r.text(), lw.log)
	}
}

// A program sets a select through its options' selected (the surface's
// attr and unattr): the Screen shows the option (SPEC §6.2).
func TestProgramSetsASelect(t *testing.T) {
	size := "m"
	r, lw := logged(t, 40, 2, func(ctx context.Context, w io.Writer, on on) {
		fmt.Fprintf(w, `<div><select%s>`, on("size", surf.Input(func(v string) { size = v })))
		for _, o := range []string{"s", "m", "l"} {
			sel := ""
			if o == size {
				sel = " selected"
			}
			fmt.Fprintf(w, `<option value="%s"%s>Size %s</option>`, o, sel, strings.ToUpper(o))
		}
		fmt.Fprintf(w, `</select><button%s>Small</button></div>`, on("small", surf.Click(func() { size = "s" })))
	})
	col, row := r.at("Small")
	r.c.Click(col, row)
	if !strings.Contains(r.text(), "[Size S ▾]") {
		t.Fatalf("the select doesn't show the program's value: %q; the surface wrote %v", r.text(), lw.log)
	}
}

// A click in an unfocused input whose value is wider than it puts the
// caret in the cell clicked.
func TestClickInALongInput(t *testing.T) {
	got := "abcdefghijklmnopqrstuvwxyz"
	r := start(t, 40, 2, view(func(ctx context.Context, w io.Writer, on on) {
		fmt.Fprintf(w, `<div><input size="10"%s value="%s"/></div>`,
			on("f", surf.Input(func(s string) { got = s })), got)
	}))
	col, row := r.at("c") // the field's third cell, drawn unfocused
	r.c.Click(col, row)
	r.key("x")
	if got != "abxcdefghijklmnopqrstuvwxyz" {
		t.Fatalf("the caret wasn't where the click was: %q", got)
	}
	if !strings.Contains(r.text(), "abxcdefghi") {
		t.Fatalf("the field scrolled away: %q", r.text())
	}
}

// A command split across Writes, as a pipe's reads split one, is applied
// once it is whole.
func TestSplitWrite(t *testing.T) {
	c := cells.New(20, 2, nil)
	cmd := hotty.Doc("t", `<p>hello</p>`)
	half := len(cmd) / 2
	if _, err := c.Write([]byte(cmd[:half])); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Write([]byte(cmd[half:])); err != nil {
		t.Fatal(err)
	}
	if l := sgrRE.ReplaceAllString(c.Lines()[0], ""); !strings.Contains(l, "hello") {
		t.Fatalf("a split command was lost: %q", l)
	}
}

// The program's a=blur while the Screen hasn't the keyboard holds:
// Focus(true) gives it to no element.
func TestBlurWhileInactive(t *testing.T) {
	var evs []string
	c := cells.New(20, 2, func(e hotty.Event) { evs = append(evs, e.Kind+" "+e.Target) })
	c.Write([]byte(hotty.Doc("t", `<input id="q">`)))
	c.Key("tab")
	c.Focus(false)
	c.Write([]byte(hotty.Blur("t")))
	evs = nil
	c.Focus(true)
	if _, _, ok := c.Cursor(); ok || len(evs) != 0 {
		t.Fatalf("the program blurred the field, and Focus(true) focused it again: %v", evs)
	}
}

// A readonly field moves its caret, and takes the keys that would edit it
// without editing.
func TestReadonly(t *testing.T) {
	var evs []string
	c := cells.New(20, 2, func(e hotty.Event) { evs = append(evs, e.Kind+" "+e.Value) })
	c.Write([]byte(hotty.Doc("t", `<input id="q" readonly value="fixed" data-on="input">`)))
	c.Key("tab")
	for _, k := range []string{"x", "backspace", "home", "delete"} {
		if !c.Key(k) {
			t.Errorf("%s went to the program", k)
		}
	}
	if !c.Paste("y") {
		t.Error("a paste went to the program")
	}
	if col, _, _ := c.Cursor(); col != 0 {
		t.Errorf("home didn't move the caret: col %d", col)
	}
	if len(evs) != 1 || evs[0] != "focus " {
		t.Fatalf("a readonly field was edited: %v", evs)
	}
	if l := sgrRE.ReplaceAllString(c.Lines()[0], ""); !strings.HasPrefix(l, "fixed ") {
		t.Fatalf("drawn %q", l)
	}
}

// A field disabled, or hidden, while it has the keyboard loses it.
func TestDisabledWhileFocused(t *testing.T) {
	for _, op := range []hotty.Op{hotty.OpAttr("q", "disabled", ""), hotty.OpAttr("q", "class", "hidden")} {
		var evs []string
		c := cells.New(20, 2, func(e hotty.Event) { evs = append(evs, e.Kind+" "+e.Value) })
		c.Write([]byte(hotty.Doc("t", `<input id="q" data-on="input">`)))
		c.Key("tab")
		c.Write([]byte(hotty.Delta("t", []hotty.Op{op})))
		if c.Key("x") || c.Paste("y") {
			t.Errorf("%s %s: the field took typing: %v", op.Op, op.Key, evs)
		}
		if _, _, ok := c.Cursor(); ok {
			t.Errorf("%s %s: the field still has a caret", op.Op, op.Key)
		}
	}
}

// A character on a select picks the next option it starts (SPEC §10.2).
func TestSelectCharacter(t *testing.T) {
	var evs []string
	c := cells.New(30, 2, func(e hotty.Event) { evs = append(evs, e.Kind+" "+e.Value) })
	c.Write([]byte(hotty.Doc("t", `<select id="s"><option value="a">Apple</option><option value="b">Banana</option>`+
		`<option value="c" disabled>Berry</option><option value="d">blue</option></select>`)))
	c.Key("tab")
	for _, k := range []string{"b", "shift+b", "b"} {
		if !c.Key(k) {
			t.Fatalf("%s went to the program", k)
		}
	}
	if got := strings.Join(evs, ","); got != "focus ,change b,change d,change b" {
		t.Fatalf("events %v", got)
	}
	if c.Key("z") != true || c.Key("ctrl+b") {
		t.Fatal("z (no option) isn't the select's, or ctrl+b is")
	}
}

// Enter and space on an element that only reports clicks (data-on~=click)
// are the program's keys (SPEC §10.2).
func TestEnterOnAClickDiv(t *testing.T) {
	var evs []string
	c := cells.New(30, 2, func(e hotty.Event) { evs = append(evs, e.Kind+" "+e.Target) })
	c.Write([]byte(hotty.Doc("t", `<div id="row" tabindex="0" data-on="click">row</div>`)))
	c.Key("tab")
	evs = nil
	if c.Key("enter") || c.Key("space") || len(evs) != 0 {
		t.Fatalf("enter or space on a div was taken as a click: %v", evs)
	}
}

// a=del without s deletes every surface (SPEC §5.4).
func TestDelAll(t *testing.T) {
	c := cells.New(20, 1, nil)
	c.Write([]byte(hotty.Doc("t", `<p>still here</p>`)))
	c.Write([]byte(hotty.DelAll()))
	if l := sgrRE.ReplaceAllString(c.Lines()[0], ""); strings.Contains(l, "still") {
		t.Fatalf("a=del left the document: %q", l)
	}
}

// After Focus(false), a click on a field gives the Screen the keyboard
// back, as a click on a host's surface does (SPEC §10.1).
func TestClickAfterFocusFalse(t *testing.T) {
	var evs []string
	c := cells.New(20, 2, func(e hotty.Event) { evs = append(evs, e.Kind+" "+e.Target) })
	c.Write([]byte(hotty.Doc("t", `<input id="q">`)))
	c.Focus(false)
	c.Click(1, 0)
	if !c.Key("x") || strings.Join(evs, ",") != "blur ,press q,focus q" {
		t.Fatalf("events %v", evs)
	}
}

// A default the markup changes sets a control the user changed, unless it
// has the keyboard (SPEC §6.2). Sent at a bare Screen, as any program's
// attr and unattr reach a host: the surface sets a select with a=field.
func TestChangedDefaultSetsState(t *testing.T) {
	c := cells.New(40, 2, func(hotty.Event) {})
	c.Write([]byte(hotty.Doc("t", `<div><select id="s"><option id="o-a" value="a" selected>Apple</option>`+
		`<option id="o-b" value="b">Banana</option><option id="o-c" value="c">Cherry</option></select>`+
		`<button id="x">X</button></div>`)))
	c.Key("tab")
	c.Key("b")   // the user picks Banana
	c.Key("tab") // the keyboard leaves the select
	empty := ""
	c.Write([]byte(hotty.Delta("t", []hotty.Op{{Op: "unattr", Target: "o-a", Key: "selected"},
		{Op: "attr", Target: "o-c", Key: "selected", Value: &empty}})))
	if l := sgrRE.ReplaceAllString(c.Lines()[0], ""); !strings.Contains(l, "Cherry") {
		t.Errorf("a changed selected didn't set the select: %q", l)
	}
}

// A field an inner moves to another parent enters again, as the surface
// takes it (rev 0): the program's a=field then reaches it.
func TestInnerFieldMovedEntersAgain(t *testing.T) {
	n, moved, val := 0, false, ""
	r, lw := logged(t, 40, 4, func(ctx context.Context, w io.Writer, on on) {
		in := fmt.Sprintf(`<input%s value="%s"/>`, on("name", surf.Input(func(s string) { val = s })), val)
		a, b := in, ""
		if moved {
			a, b = "", in
		}
		fmt.Fprintf(w, `<div id="page">n %d <div id="pa">%s</div><div id="pb">%s</div></div><button%s>Move</button><button%s>Fill</button>`,
			n, a, b, on("mv", surf.Click(func() { moved = true; n++ })), on("fill", surf.Click(func() { val = "filled" })))
	})
	r.c.Click(0, 1)
	r.key("a", "b", "c") // the field's rev is past 0
	r.c.Click(r.at("Move"))
	r.c.Click(r.at("Fill"))
	if !strings.Contains(r.text(), "filled") {
		t.Errorf("the program's value didn't reach the moved field: %q; the surface wrote %v", r.text(), lw.log)
	}
}
