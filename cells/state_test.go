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
