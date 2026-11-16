// Package surftest drives a surface in a test, on hotty-go's test host,
// which applies the document and every delta as the SPEC says. After every
// act (a click, a key, typing) the rig checks that the host's document is the
// surface's own view, so a wrong delta fails the test where it was sent; the
// test then reads the host's document for what the user would see.
//
//	r := surftest.Start(t, surf.New("tasks", Page(a), CSS))
//	r.Click("button:text(Add task)")
//	r.Type("input[name=title]", "Write the docs")
//	r.Key("enter")
//	if len(r.Find("li:text(Write the docs)")) != 1 { … }
//
// A selector is tag#id[attr=value]:text(…), every part optional: [attr]
// alone for presence, :text(t) for an element whose whole text is t
// (spaces normalised) and :text-start(t) for one whose text starts with t.
//
// What would go wrong silently in an app fails the test: two surf.On for
// one id in a render, a ui component given a variant, size or side it
// doesn't take.
package surftest

import (
	"errors"
	"strings"
	"testing"

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
	r := &Rig{t: t, host: h, s: s}
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
		r.t.Error(f)
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
	id := attr(r.One(sel), "id")
	// A click on what reports none (text, a field) still presses, focuses
	// or blurs, as the user's would.
	if err := r.host.Click(r.s.Name(), id); err != nil && !errors.Is(err, hottytest.ErrNoReport) {
		r.t.Fatalf("surftest: click %s: %v", sel, err)
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
		if c == ' ' {
			k = "space"
		}
		r.Key(k)
	}
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
// ("enter", "arrow_down", "ctrl+s"), and says whether the surface took it.
// The test host's mod is ctrl: Key("ctrl+s") reaches a surf.Key handler as
// "mod+s".
func (r *Rig) Key(key string) bool {
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
		r.t.Fatalf("surftest: %q matches %d elements, not 1", sel, len(ns))
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
