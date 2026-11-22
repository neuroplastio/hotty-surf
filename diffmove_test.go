package surf_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/surftui"
	"neuroplast.io/hotty-surf/surftest"
)

// moveView renders two lists; the rows named in left are in the first.
func moveView(left map[string]bool, inner bool) templ.Component {
	return view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		for _, list := range []string{"left", "right"} {
			fmt.Fprintf(w, `<ul id="%s"><li id="%s-head">%s</li>`, list, list, strings.ToUpper(list[:1]))
			if inner && list == "left" {
				io.WriteString(w, "<li id=\"left-note\">") // a text beside it: the list's children go whole
				io.WriteString(w, "</li>")
				io.WriteString(w, "~")
			}
			for _, r := range []string{"a", "x", "b"} {
				if left[r] == (list == "left") {
					fmt.Fprintf(w, `<li id="row-%s"><span id="row-%s-label">%s</span></li>`, r, r, r)
				}
			}
			io.WriteString(w, "</ul>")
		}
	})
}

// rows are the screen's non-blank lines, trimmed.
func rows(c *surftui.Screen) string {
	var out []string
	for _, l := range c.Lines() {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return strings.Join(out, " ")
}

// A keyed row that moves to another parent within one render is never in
// the document twice: the delta takes it out of its old parent before it
// puts it in the new one, so a host that targets it by id finds one.
// Here x moves from the second list into the first, which the diff visits
// first; and again where the first list's children go whole (inner).
func TestMoveBetweenParents(t *testing.T) {
	for _, inner := range []bool{false, true} {
		t.Run(fmt.Sprint("inner=", inner), func(t *testing.T) {
			left := map[string]bool{"a": true}
			s := surf.New("t", moveView(left, inner), "")
			var cellsErr error
			c := surftui.New(40, 8, func(hotty.Event) {})
			s.Attach(writerFunc(func(p []byte) (int, error) {
				n, err := c.Write(p)
				if err != nil && cellsErr == nil {
					cellsErr = err
				}
				return n, nil
			}), hotty.Caps{})
			doc, err := s.Document()
			if err != nil {
				t.Fatal(err)
			}
			if _, err := c.Write([]byte(hotty.Doc("t", doc))); err != nil {
				t.Fatal(err)
			}
			left["x"] = true
			if err := s.Update(); err != nil {
				t.Fatal(err)
			}
			if cellsErr != nil {
				t.Fatalf("cells: %v", cellsErr)
			}
			if got := rows(c); got != want(inner, "L a x R b") {
				t.Fatalf("cells after x moved left: %q", got)
			}
			// And back, the other way round.
			delete(left, "x")
			if err := s.Update(); err != nil {
				t.Fatal(err)
			}
			if got := rows(c); got != want(inner, "L a R x b") || cellsErr != nil {
				t.Fatalf("cells after x moved back: %q, %v", got, cellsErr)
			}
		})
	}
}

// The same on the test host, whose document surftest checks against the
// surface's after each update.
func TestMoveBetweenParentsHost(t *testing.T) {
	left := map[string]bool{"a": true}
	s := surf.New("t", moveView(left, false), "")
	r := surftest.Start(t, s)
	under := func() string {
		x := r.One("#row-x")
		for _, a := range x.Parent.Attr {
			if a.Key == "id" {
				return a.Val
			}
		}
		return ""
	}
	left["x"] = true
	r.Update()
	if p := under(); p != "left" {
		t.Fatalf("after x moved left, it is under %q", p)
	}
	delete(left, "x")
	r.Update()
	if p := under(); p != "right" {
		t.Fatalf("after x moved back, it is under %q", p)
	}
}

// want is rows' expectation, with the inner case's note after L.
func want(inner bool, rs string) string {
	if inner {
		return strings.Replace(rs, "L ", "L ~ ", 1)
	}
	return rs
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }

// x moves to the other list, and y, which was inside x, moves out of it
// into the first list: y goes with x when x is taken out, so the delta
// takes out x alone and inserts each where it now is.
func TestMoveNested(t *testing.T) {
	split := false
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		y := `<li id="y">y</li>`
		if !split {
			fmt.Fprintf(w, `<ul id="left"><li id="lh">L</li><li id="x"><ul id="xs">%s</ul></li></ul><ul id="right"><li id="rh">R</li></ul>`, y)
			return
		}
		fmt.Fprintf(w, `<ul id="left"><li id="lh">L</li>%s</ul><ul id="right"><li id="rh">R</li><li id="x"><ul id="xs"></ul></li></ul>`, y)
	})
	s := surf.New("t", v, "")
	var cellsErr error
	c := surftui.New(40, 8, func(hotty.Event) {})
	s.Attach(writerFunc(func(p []byte) (int, error) {
		n, err := c.Write(p)
		if err != nil && cellsErr == nil {
			cellsErr = err
		}
		return n, nil
	}), hotty.Caps{})
	doc, err := s.Document()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Write([]byte(hotty.Doc("t", doc))); err != nil {
		t.Fatal(err)
	}
	split = true
	if err := s.Update(); err != nil {
		t.Fatal(err)
	}
	if got := rows(c); got != "L y R" || cellsErr != nil {
		t.Fatalf("cells: %q, %v", got, cellsErr)
	}
	r := surftest.Start(t, surf.New("h", v, ""))
	split = false
	r.Update()
	split = true
	r.Update()
	r.One("#y")
	r.One("#x")
}
