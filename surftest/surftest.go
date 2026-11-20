// Package surftest drives a surface in a test, on hotty-go's test host,
// which applies the document and every delta as the SPEC says. After every
// act (a click, a key, typing) the rig checks that the host's document is the
// surface's own view, so a wrong delta fails the test where it was sent; the
// test then reads the host's document for what the user would see.
//
//	r := surftest.Start(t, surf.New("tasks", Page(a), CSS))
//	r.Click("button:text(Add task)")
//	r.Type("input[name=title]", "Write the docs")
//	r.Key("enter") // the field's form submits (surf.Submit)
//	if len(r.Find("li:text(Write the docs)")) != 1 { … }
//
// A selector is tag#id[attr=value]:text(…), every part optional: [attr]
// alone for presence, :text(t) for an element whose whole text is t
// (spaces normalised) and :text-start(t) for one whose text starts with t.
// An element's ancestors that hold nothing else have the same whole text,
// so :text alone matches them too: name the tag (h2:text(Saved)).
//
// A field's value and a check box's state are the host's, not attributes:
// a value or checked in the markup is only where the field starts. Read
// them with Rig.Value and Rig.Checked.
//
// What would go wrong silently in an app fails the test: two surf.On for
// one id in a render, a ui component given a variant, size or side it
// doesn't take.
package surftest

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"unicode"

	"golang.org/x/net/html"
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottytest"
	"neuroplast.io/hotty-surf"

	"neuroplast.io/hotty-surf/internal/input"
	"neuroplast.io/hotty-surf/internal/rt"
	"neuroplast.io/hotty-surf/lint"
)

// Start starts s on a test host and checks its first frame. A reply from the
// host that is an error fails the test.
func Start(t testing.TB, s *surf.Surface, opts ...Option) *Rig {
	t.Helper()
	c := config{cols: 80, rows: 24}
	for _, o := range opts {
		if o.set != nil {
			o.set(&c)
		}
	}
	h := hottytest.New(t, hottytest.Size(c.cols, c.rows))
	r := &Rig{t: t, host: h, s: s, faulted: map[string]bool{}}
	// Every kept compiled instance is rendered too, and checked.
	rt.Peek.Check(s, true)
	s.Attach(h, hottytest.DefaultCaps())
	doc, err := s.Document()
	if err != nil {
		t.Fatal(err)
	}
	// Placed as Surface.Attach asks: Press and Hover on.
	r.send(hotty.Doc(s.Name(), doc))
	r.send(hotty.Place(s.Name(), hotty.Placement{Cols: c.cols, Rows: c.rows, Press: hotty.On, Hover: hotty.On}))
	r.settle()
	return r
}

// An Option sets up the rig.
type Option struct {
	set func(*config)
}

type config struct{ cols, rows int }

// Size is the host's size in cells: 80×24 when not given.
func Size(cols, rows int) Option {
	return Option{set: func(c *config) { c.cols, c.rows = cols, rows }}
}

// A Rig is a surface on a test host.
type Rig struct {
	t     testing.TB
	host  *hottytest.Host
	s     *surf.Surface
	in    input.Reader
	taken bool // whether the surface took a key in the last drain
	// faulted are the faults the test has failed on, each said once.
	faulted map[string]bool
}

func (r *Rig) send(cmd string) {
	r.t.Helper()
	if _, err := r.host.Write([]byte(cmd)); err != nil {
		r.t.Fatal(err)
	}
}

// settle runs what is pending, hands the surface what the host sent until
// neither has more, and checks the host's document and the renders'
// faults.
func (r *Rig) settle() {
	r.t.Helper()
	for i := 0; ; i++ {
		if i == 100 {
			r.t.Fatal("surftest: the surface and the host are still talking after 100 rounds")
		}
		busy := false
		select {
		case <-r.s.Pending():
			if err := r.s.Update(); err != nil {
				r.t.Fatal(err)
			}
			busy = true
		default:
		}
		if r.host.Buffered() > 0 {
			r.drain()
			busy = true
		}
		if !busy {
			break
		}
	}
	r.consistent()
	for _, f := range rt.Peek.Faults(r.s) {
		// A fault repeats at every render; the test says it once.
		if !r.faulted[f] {
			r.faulted[f] = true
			r.t.Error(f)
		}
	}
}

// drain hands the surface what the host sent.
func (r *Rig) drain() {
	r.t.Helper()
	for r.host.Buffered() > 0 {
		buf := make([]byte, r.host.Buffered())
		n, _ := r.host.Read(buf)
		err := r.in.Feed(buf[:n], input.Handlers{
			Event: r.s.Event,
			Reply: func(rep hotty.Reply) error {
				if err := hotty.Err(rep); err != nil {
					r.t.Errorf("surftest: the host replied: %v", err)
				}
				return nil
			},
			Key: func(k string) error {
				took, err := r.s.Key(k)
				r.taken = r.taken || took
				return err
			},
		})
		if err != nil {
			r.t.Fatal(err)
		}
	}
}

// consistent fails the test unless the host's document is the surface's
// own tree: what its deltas built is what it meant to build.
func (r *Rig) consistent() {
	r.t.Helper()
	tree := rt.Peek.Tree(r.s)
	if tree == nil {
		r.t.Fatal("surftest: the surface has no document")
	}
	var b strings.Builder
	if err := html.Render(&b, tree); err != nil {
		r.t.Fatal(err)
	}
	got, want := canon(r.t, r.HTML()), canon(r.t, b.String())
	if got != want {
		i := 0
		for i < len(got) && i < len(want) && got[i] == want[i] {
			i++
		}
		r.t.Fatalf("surftest: the host's document is not the surface's, from byte %d:\nhost:    …%s\nsurface: …%s", i,
			got[max(0, i-80):min(len(got), i+200)], want[max(0, i-80):min(len(want), i+200)])
	}
}

// Click clicks the one element matching sel. It fails the test if no
// element or more than one matches.
func (r *Rig) Click(sel string) {
	r.t.Helper()
	n := r.One(sel)
	id := attr(n, "id")
	// A click on what reports none (text, a field) still presses, focuses
	// or blurs, as the user's would.
	if err := r.host.Click(r.s.Name(), id); err != nil && !errors.Is(err, hottytest.ErrNoReport) {
		r.t.Fatalf("surftest: click %s: %v", sel, err)
	}
	// A click on a check box or a radio button toggles it, as space on it
	// does: the test host's click focuses it only.
	if t := strings.ToLower(attr(n, "type")); n.Data == "input" && (t == "checkbox" || t == "radio") && !hasAttr(n, "disabled") {
		r.host.Key("space")
	}
	r.settle()
}

// DoubleClick double clicks the element sel as a host reports it: a
// click with count 1, then one with count 2, each with its press and
// release.
func (r *Rig) DoubleClick(sel string) {
	r.t.Helper()
	n := r.One(sel)
	if err := r.host.DoubleClick(r.s.Name(), attr(n, "id")); err != nil && !errors.Is(err, hottytest.ErrNoReport) {
		r.t.Fatalf("surftest: double click %s: %v", sel, err)
	}
	r.settle()
}

// Drag drags the one element matching from onto the one matching to, as a
// user's pointer does (SPEC §9.1): a press on from, a move onto to, and the
// release there. from's surf.Drag hears Start, a move Over to, and End;
// to's surf.Drop hears the End. The test host lays nothing out, so the
// pointer's cells are 0 and no step is said: DragStep says one. It fails
// the test if either selector matches no element or more than one, or if
// from takes no drag.
func (r *Rig) Drag(from, to string) {
	r.t.Helper()
	r.drag(from, to, -1, -1)
}

// DragStep is Drag released at step tx, ty of to (TX, TY in the
// DragEvent), for a to with surf.Drop and data-steps: in a row split into
// halves (data-steps="0 1"), ty 0 is before it and 1 after it. A step
// along an axis to does not count is not said.
func (r *Rig) DragStep(from, to string, tx, ty int) {
	r.t.Helper()
	r.drag(from, to, tx, ty)
}

func (r *Rig) drag(from, to string, tx, ty int) {
	r.t.Helper()
	src, dst := attr(r.One(from), "id"), attr(r.One(to), "id")
	if err := r.host.DragStart(r.s.Name(), src, 0, 0); err != nil {
		r.t.Fatalf("surftest: drag %s: %v", from, err)
	}
	r.settle()
	var err error
	if tx < 0 {
		err = r.host.DragMove(dst, 0, 0)
	} else {
		err = r.host.DragOver(dst, 0, 0, tx, ty)
	}
	if err != nil {
		r.t.Fatalf("surftest: drag %s onto %s: %v", from, to, err)
	}
	r.settle()
	if tx < 0 {
		err = r.host.DragEnd(dst, 0, 0)
	} else {
		err = r.host.DragEndOver(dst, 0, 0, tx, ty)
	}
	if err != nil && !errors.Is(err, hottytest.ErrNoReport) {
		r.t.Fatalf("surftest: drop %s on %s: %v", from, to, err)
	}
	r.settle()
}

// Secondary makes a secondary press (a right click) on the one element
// matching sel, as a user's does (SPEC §9): the surf.Secondary handler of
// the nearest element from it outward that has one hears it, at cell 0, 0
// (the test host lays nothing out). It fails the test if no element or
// more than one matches.
func (r *Rig) Secondary(sel string) {
	r.t.Helper()
	id := attr(r.One(sel), "id")
	if err := r.host.SecondaryPress(r.s.Name(), id, 0, 0); err != nil && !errors.Is(err, hottytest.ErrNoReport) {
		r.t.Fatalf("surftest: secondary press on %s: %v", sel, err)
	}
	r.settle()
}

// Type gives the keyboard to the one field matching sel and types text into
// it, a key at a time.
func (r *Rig) Type(sel, text string) {
	r.t.Helper()
	id := attr(r.One(sel), "id")
	// A click on a field only focuses it: no click is reported.
	if err := r.host.Click(r.s.Name(), id); err != nil && !errors.Is(err, hottytest.ErrNoReport) {
		r.t.Fatalf("surftest: type into %s: %v", sel, err)
	}
	r.settle()
	for _, c := range text {
		k := string(c)
		switch lo := unicode.ToLower(c); {
		case c == ' ':
			k = "space"
		case lo != c:
			// hotty names a capital by its key and shift (SPEC §10.4):
			// "A" alone is the key a.
			k = "shift+" + string(lo)
		}
		r.Key(k)
	}
}

// Value is the value of the one field matching sel as the user sees it:
// what they typed, or what the app set. The host's document doesn't show
// it, as a field's value attribute is only where it starts. It fails the
// test if sel matches no element, more than one, or one that isn't a
// field. A select's is its picked option's value.
func (r *Rig) Value(sel string) string {
	r.t.Helper()
	n := r.One(sel)
	if t := strings.ToLower(attr(n, "type")); n.Data == "input" && (t == "checkbox" || t == "radio") {
		r.t.Fatalf("surftest: %s is a check box or a radio button: Checked reads it", sel)
	}
	v, ok := r.host.Surface(r.s.Name()).Value(attr(n, "id"))
	if !ok || n.Data != "input" && n.Data != "textarea" && n.Data != "select" {
		r.t.Fatalf("surftest: %s is no field", sel)
	}
	return v
}

// Checked says whether the one check box or radio button matching sel is
// checked, as the user sees it. The host's document doesn't show it, as
// its checked attribute is only where it starts. It fails the test if sel
// matches no element, more than one, or one that isn't a check box or a
// radio button.
func (r *Rig) Checked(sel string) bool {
	r.t.Helper()
	n := r.One(sel)
	if t := strings.ToLower(attr(n, "type")); n.Data != "input" || t != "checkbox" && t != "radio" {
		r.t.Fatalf("surftest: %s is no check box or radio button", sel)
	}
	v, _ := r.host.Surface(r.s.Name()).Value(attr(n, "id"))
	return v == "true"
}

// Update runs what the surface's Do has queued (a load, a toast's timer),
// renders, and checks the host's document, as after any act.
func (r *Rig) Update() {
	r.t.Helper()
	if err := r.s.Update(); err != nil {
		r.t.Fatal(err)
	}
	r.settle()
}

// Key presses a key where the keyboard is, named as hotty names keys
// ("enter", "arrow_down", "ctrl+s"), and says whether the surface took it:
// the host's element used it (a text field keeps the keys it edits with,
// Enter among them, which commits the field and submits its form) or a
// surf.Key handler did. A key the surface took never reaches the program;
// true doesn't say a handler ran, which the test reads in the document.
// The test host's mod is ctrl: Key("ctrl+s") reaches a surf.Key handler as
// "mod+s".
func (r *Rig) Key(key string) (taken bool) {
	r.t.Helper()
	r.taken = false
	used := r.host.Key(key)
	r.settle()
	return used || r.taken
}

// Find is the elements of the host's document that match sel, in order.
func (r *Rig) Find(sel string) []*html.Node {
	r.t.Helper()
	m, err := parseSelector(sel)
	if err != nil {
		r.t.Fatal(err)
	}
	doc, err := html.Parse(strings.NewReader(r.HTML()))
	if err != nil {
		r.t.Fatal(err)
	}
	var out []*html.Node
	walk(doc, func(n *html.Node) {
		if m.match(n) {
			out = append(out, n)
		}
	})
	return out
}

// One is the one element matching sel. It fails the test if no element or
// more than one matches.
func (r *Rig) One(sel string) *html.Node {
	r.t.Helper()
	ns := r.Find(sel)
	if len(ns) != 1 {
		hint := ""
		if len(ns) == 0 && (strings.Contains(sel, "[checked") || strings.Contains(sel, "[value")) {
			hint = " (a field's value and a check box's state are the host's, not attributes: Rig.Value and Rig.Checked read them)"
		}
		for _, n := range ns {
			hint += "\n  " + opening(n)
		}
		r.t.Fatalf("surftest: %q matches %d elements, not 1%s", sel, len(ns), hint)
	}
	return ns[0]
}

// HTML is the host's document, without its stylesheet.
func (r *Rig) HTML() string {
	doc := r.host.Surface(r.s.Name()).HTML()
	if i := strings.Index(doc, "</style>"); i >= 0 {
		return doc[i+len("</style>"):]
	}
	return doc
}

// Lint fails the test for each rule of the kit's lint that the app's sources
// under dir break (colours outside the theme, px sizes, the root font,
// hiding the page below a breakpoint), as `hotty-surf lint` and
// `hotty-surf run` report them.
func Lint(t testing.TB, dir string) {
	t.Helper()
	fs, err := lint.Dir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		t.Errorf("%s:%d:%d: %s: %s", f.File, f.Line, f.Col, f.Rule, f.Text)
	}
}

// opening is n's start tag, as a failure names a match.
func opening(n *html.Node) string {
	var b strings.Builder
	b.WriteString("<" + n.Data)
	for _, a := range n.Attr {
		fmt.Fprintf(&b, " %s=%q", a.Key, a.Val)
	}
	b.WriteString(">")
	return b.String()
}
