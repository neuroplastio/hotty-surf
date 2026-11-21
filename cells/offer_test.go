package cells

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unsafe"

	"github.com/a-h/templ"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/rt"
	"neuroplast.io/hotty-surf/internal/stories"
	"neuroplast.io/hotty-surf/ui"
)

// checked is the Screen's host for a surface's offers, with the hand-off's
// oracle inside each offer, while the surface's tree is as it was
// rendered: the Screen's copy (H) is the surface's tree (S) node for node,
// sharing no node, no string but an atom's and no attribute list's room
// with it; and the document's parse (W) is S as well, or else the
// difference is noted, where the Screen shows S and a host that parses
// shows W.
type checked struct {
	t        testing.TB
	c        *Screen
	offers   int
	diverged []string
	// frags are the fragments of deltas offered, and copies the nodes the
	// Screen copied for them, the last delta's: taken says which of these
	// the Screen's tree holds.
	frags  int
	copies []*html.Node
}

// check makes c's offers go through a checked, until the test ends.
func check(t testing.TB, c *Screen) *checked {
	h := &checked{t: t, c: c}
	old := rt.HostOf
	rt.HostOf = func(w io.Writer) rt.Host {
		if sh, ok := hostOf(w).(screenHost); ok && sh.c == c {
			return h
		}
		return old(w)
	}
	t.Cleanup(func() { rt.HostOf = old })
	return h
}

func (h *checked) OfferDocument(name, doc string, nodes []*html.Node) {
	h.t.Helper()
	h.offers++
	screenHost{h.c}.OfferDocument(name, doc, nodes)
	if d := sameTrees(childrenOf(h.c.offer.body), nodes, true); d != "" {
		h.t.Errorf("the Screen's copy isn't the surface's tree: %s", d)
	}
	for _, n := range childrenOf(h.c.offer.body) {
		if n.Parent != h.c.offer.body {
			h.t.Errorf("a copied node's parent isn't the offer's body")
		}
	}
	body := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	w, err := html.ParseFragment(strings.NewReader(doc), body)
	if err != nil {
		h.t.Fatal(err)
	}
	if d := sameTrees(w, nodes, false); d != "" {
		h.diverged = append(h.diverged, d)
	}
}

func (h *checked) OfferDelta(name string, frags []rt.Fragment) {
	h.t.Helper()
	screenHost{h.c}.OfferDelta(name, frags)
	h.copies = h.copies[:0]
	for i, f := range frags {
		h.frags++
		got := h.c.frags.frags[i]
		if got.markup != f.Markup {
			h.t.Errorf("fragment %d's markup isn't the surface's", i)
		}
		if d := sameTrees(got.nodes, f.Nodes, true); d != "" {
			h.t.Errorf("the Screen's copy of fragment %d isn't the surface's nodes: %s", i, d)
		}
		for _, n := range got.nodes {
			if n.Parent != nil || n.PrevSibling != nil || n.NextSibling != nil {
				h.t.Errorf("fragment %d's copy is linked to another node", i)
			}
		}
		if cap(got.nodes) != len(got.nodes) {
			h.t.Errorf("fragment %d's nodes have room for the next fragment's", i)
		}
		h.copies = append(h.copies, got.nodes...)
		if len(f.Nodes) == 0 {
			// An inner that empties its target.
			if f.Markup != "" {
				h.t.Errorf("fragment %d is %q, of no nodes", i, f.Markup)
			}
			continue
		}
		// Parsed where the surface's nodes stand: under their parent.
		ctx := f.Nodes[0].Parent
		if ctx == nil || ctx.Type != html.ElementNode {
			ctx = &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
		}
		w, err := html.ParseFragment(strings.NewReader(f.Markup), ctx)
		if err != nil {
			h.t.Fatal(err)
		}
		if d := sameTrees(w, f.Nodes, false); d != "" {
			h.diverged = append(h.diverged, "a fragment: "+d)
		}
	}
}

// taken is how many of the last delta's copied nodes the Screen's tree
// holds, which it then forgets.
func (h *checked) taken() int {
	in := map[*html.Node]bool{}
	if h.c.body != nil {
		walk(h.c.body, func(n *html.Node) { in[n] = true })
	}
	n := 0
	for _, k := range h.copies {
		if in[k] {
			n++
		}
	}
	h.copies = h.copies[:0]
	return n
}

func childrenOf(n *html.Node) []*html.Node {
	var ns []*html.Node
	for k := n.FirstChild; k != nil; k = k.NextSibling {
		ns = append(ns, k)
	}
	return ns
}

// sameTrees says where got isn't want, "" where it is. With copied, got is
// a copy of want, which may share nothing with it but atoms' names.
func sameTrees(got, want []*html.Node, copied bool) string {
	if len(got) != len(want) {
		return fmt.Sprintf("%d nodes, want %d", len(got), len(want))
	}
	for i := range got {
		if d := sameNode(got[i], want[i], copied, strconv.Itoa(i)); d != "" {
			return d
		}
	}
	return ""
}

func sameNode(g, w *html.Node, copied bool, at string) string {
	at += "/" + w.Data
	switch {
	case g.Type != w.Type, g.Data != w.Data, g.DataAtom != w.DataAtom, g.Namespace != w.Namespace:
		return fmt.Sprintf("%s: %v %q %v %q, want %v %q %v %q", at, g.Type, g.Data, g.DataAtom, g.Namespace, w.Type, w.Data, w.DataAtom, w.Namespace)
	case len(g.Attr) != len(w.Attr):
		return fmt.Sprintf("%s: attributes %v, want %v", at, g.Attr, w.Attr)
	}
	ga, wa := g.Attr, w.Attr
	if !copied {
		// A parse puts a formatting element's attributes (an a, a b) in
		// order of their names, which a tree's DOM doesn't keep.
		ga, wa = sortedAttrs(ga), sortedAttrs(wa)
	}
	for i, a := range ga {
		if a != wa[i] {
			return fmt.Sprintf("%s: attributes %v, want %v", at, g.Attr, w.Attr)
		}
	}
	if copied {
		switch {
		case g == w:
			return at + ": the node itself, not a copy"
		case cap(g.Attr) != len(g.Attr):
			return fmt.Sprintf("%s: an attribute list with room for %d more", at, cap(g.Attr)-len(g.Attr))
		case shared(g.Data, w.Data):
			return at + ": the surface's own string " + strconv.Quote(g.Data)
		}
		for i, a := range g.Attr {
			if shared(a.Key, w.Attr[i].Key) || shared(a.Val, w.Attr[i].Val) || shared(a.Namespace, w.Attr[i].Namespace) {
				return fmt.Sprintf("%s: the surface's own string in %v", at, a)
			}
		}
	}
	gk, wk := g.FirstChild, w.FirstChild
	for i := 0; gk != nil || wk != nil; i++ {
		if gk == nil || wk == nil {
			return at + ": not as many children"
		}
		if gk.Parent != g {
			return at + ": a child whose parent is another node"
		}
		if d := sameNode(gk, wk, copied, at+"."+strconv.Itoa(i)); d != "" {
			return d
		}
		gk, wk = gk.NextSibling, wk.NextSibling
	}
	return ""
}

func sortedAttrs(as []html.Attribute) []html.Attribute {
	return slices.SortedFunc(slices.Values(as), func(a, b html.Attribute) int {
		return cmp.Or(strings.Compare(a.Namespace, b.Namespace), strings.Compare(a.Key, b.Key), strings.Compare(a.Val, b.Val))
	})
}

// shared says whether a copy's string g is the original's w itself, which
// it may be only as an atom's name, a static string.
func shared(g, w string) bool {
	if g == "" || unsafe.StringData(g) != unsafe.StringData(w) {
		return false
	}
	a := atom.Lookup([]byte(g))
	return a == 0 || unsafe.StringData(a.String()) != unsafe.StringData(g)
}

// onScreen is a surface on a Screen as run's host has it: a refused delta
// sends the document again. With offer, the writer is a wrapper the offer
// sees through; without, it is one it doesn't, and the Screen parses.
type onScreen struct {
	s     *surf.Surface
	c     *Screen
	stale bool
}

func (o *onScreen) Write(p []byte) (int, error) {
	if _, err := o.c.Write(p); err != nil && bytes.HasPrefix(p, []byte("\x1b]7279;a=delta")) {
		o.stale = true
	}
	return len(p), nil
}

type offering struct{ *onScreen }

func (o offering) Wrapped(rt.Seal) io.Writer { return o.c }

func newOnScreen(t testing.TB, name string, view templ.Component, css string, cols, rows int, offer bool) *onScreen {
	o := &onScreen{s: surf.New(name, view, css)}
	o.c = New(cols, rows, func(e hotty.Event) {
		if err := o.s.Event(e); err != nil {
			t.Error(err)
		}
	})
	var w io.Writer = o
	if offer {
		w = offering{o}
	}
	o.s.Attach(w, hotty.Caps{})
	return o
}

// checked attaches o's surface again, through a checked: a surface asks
// for its host when it is attached.
func (o *onScreen) checked(t testing.TB) *checked {
	h := check(t, o.c)
	o.s.Attach(offering{o}, hotty.Caps{})
	return h
}

func (o *onScreen) document(t testing.TB) {
	t.Helper()
	doc, err := o.s.Document()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := o.c.Write([]byte(hotty.Doc(o.s.Name(), doc))); err != nil {
		t.Fatal(err)
	}
	o.stale = false
}

// TestOfferIsTheTree holds every story of the kit, with the kit's sheet
// and with none (a plx block's), to the oracle (checked), and the Screen
// that takes the offer to one that parses, through random steps of keys,
// the pointer, resizes and documents sent again.
func TestOfferIsTheTree(t *testing.T) {
	steps := 40
	if testing.Short() {
		steps = 10
	}
	frags, taken := 0, 0
	defer func() {
		t.Logf("deltas offered %d fragments; the Screen's trees took %d of their nodes", frags, taken)
		if frags == 0 || taken == 0 {
			t.Error("no delta's fragment was offered and taken")
		}
	}()
	for si, st := range stories.All() {
		for k, css := range []string{"", ui.Stylesheet()} {
			name := fmt.Sprintf("%s/%d", st.ID, k)
			r := rand.New(rand.NewPCG(uint64(si), uint64(k)))
			cols, rows := 30+r.IntN(60), 6+r.IntN(20)
			got := newOnScreen(t, "v", st.View(), css, cols, rows, true)
			h := got.checked(t)
			want := newOnScreen(t, "v", st.View(), css, cols, rows, false)
			got.document(t)
			want.document(t)
			if h.offers != 1 || got.c.offer != nil {
				t.Fatalf("%s: %d offers, %v left", name, h.offers, got.c.offer)
			}
			for i := range steps {
				act := ""
				for _, o := range []*onScreen{got, want} {
					rr := rand.New(rand.NewPCG(uint64(si), uint64(1000*k+i)))
					act = walkStep(t, rr, o, cols, rows)
				}
				if len(h.diverged) > 0 {
					t.Logf("%s: the wire parses as another tree, where the Screen shows the surface's: %s", name, h.diverged[0])
					break
				}
				taken += h.taken()
				what := fmt.Sprintf("%s step %d (%s)", name, i, act)
				if !sameCells(t, what, got.c.Cells(), want.c.Cells()) {
					break
				}
				gx, gy, gok := got.c.Cursor()
				wx, wy, wok := want.c.Cursor()
				if gx != wx || gy != wy || gok != wok {
					t.Errorf("%s: the caret at %d,%d %v, want %d,%d %v", what, gx, gy, gok, wx, wy, wok)
					break
				}
			}
			frags += h.frags
			rt.HostOf = hostOf
		}
	}
}

// changing is a view of the markup cur holds, which a test changes
// between renders.
func changing(cur *string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, *cur)
		return err
	})
}

// TestDeltaOffersAreTheTree changes a surface's markup at random, which
// makes deltas of every op surf's differ writes (inner, replace, before,
// append), and holds each delta's offer to the oracle (checked), and the
// Screen that takes the offers to one that parses, cell for cell.
func TestDeltaOffersAreTheTree(t *testing.T) {
	runs, steps := 60, 30
	if testing.Short() {
		runs = 15
	}
	frags, taken := 0, 0
	for run := range runs {
		r := rand.New(rand.NewPCG(uint64(run), 9))
		doc := func() string { return strings.Replace(randomDoc(r), `id="root"`, `id="top"`, 1) }
		cur := doc()
		cols, rows := 30+r.IntN(50), 6+r.IntN(20)
		got := newOnScreen(t, "v", changing(&cur), "", cols, rows, true)
		h := got.checked(t)
		want := newOnScreen(t, "v", changing(&cur), "", cols, rows, false)
		got.document(t)
		want.document(t)
		for i := range steps {
			var act string
			switch k := r.IntN(6); {
			case k == 0:
				cur = doc()
				act = "another document's markup"
			case k == 1:
				// Some of it: the same document with a part dropped.
				if a, b, ok := strings.Cut(cur, `<div id="sc"`); ok {
					cur = a + `<p id="cut">cut</p><div id="sc"` + b
				}
				act = "a part added"
			default:
				ns := parseDoc(cur)
				mutate(r, ns, false)
				cur = renderDoc(ns)
				act = "texts and attributes"
			}
			for _, o := range []*onScreen{got, want} {
				if err := o.s.Update(); err != nil {
					t.Fatal(err)
				}
				if o.stale {
					o.document(t)
				}
			}
			taken += h.taken()
			if len(h.diverged) > 0 {
				t.Logf("run %d: the wire parses as another tree, where the Screen shows the surface's: %s", run, h.diverged[0])
				break
			}
			if !sameCells(t, fmt.Sprintf("run %d step %d (%s)", run, i, act), got.c.Cells(), want.c.Cells()) {
				break
			}
		}
		frags += h.frags
		rt.HostOf = hostOf
	}
	t.Logf("%d fragments offered; the Screen's trees took %d of their nodes", frags, taken)
	if taken == 0 {
		t.Error("no fragment was taken")
	}
}

func TestDeltaOfferEdges(t *testing.T) {
	const doc = `<div id="root"><div id="a"><p>old</p></div><div id="b"><p>old</p></div></div>`
	p := func(text string) *html.Node {
		n := &html.Node{Type: html.ElementNode, Data: "p", DataAtom: atom.P}
		n.AppendChild(&html.Node{Type: html.TextNode, Data: text})
		return n
	}
	frag := func(text string) rt.Fragment {
		return rt.Fragment{Markup: "<p>" + text + "</p>", Nodes: []*html.Node{p(text)}}
	}
	// screen is a Screen showing doc, offered frags; copies are the
	// copies it made, by fragment.
	screen := func(t *testing.T, frags ...rt.Fragment) (*Screen, []*html.Node) {
		c := New(30, 4, nil)
		if _, err := c.Write([]byte(hotty.Doc("v", doc))); err != nil {
			t.Fatal(err)
		}
		screenHost{c}.OfferDelta("v", frags)
		var copies []*html.Node
		for _, f := range c.frags.frags {
			copies = append(copies, f.nodes[0])
		}
		return c, copies
	}
	write := func(t *testing.T, c *Screen, cmd string) {
		t.Helper()
		if _, err := c.Write([]byte(cmd)); err != nil {
			t.Fatal(err)
		}
	}
	child := func(c *Screen, id string) *html.Node { return c.byID(id).FirstChild }
	t.Run("taken in order, past other ops", func(t *testing.T) {
		c, copies := screen(t, frag("x"), frag("y"))
		write(t, c, hotty.Delta("v", []hotty.Op{hotty.OpInner("a", "<p>x</p>"), hotty.OpAttr("b", "class", "flex"), hotty.OpInner("b", "<p>y</p>")}))
		if child(c, "a") != copies[0] || child(c, "b") != copies[1] {
			t.Error("the delta didn't take its fragments' copies")
		}
		if c.frags != nil || c.taking != nil {
			t.Error("the offer outlived its delta")
		}
	})
	t.Run("the same delta twice", func(t *testing.T) {
		c, copies := screen(t, frag("x"))
		cmd := hotty.Delta("v", []hotty.Op{hotty.OpInner("a", "<p>x</p>")})
		write(t, c, cmd)
		write(t, c, cmd)
		if child(c, "a") == copies[0] {
			t.Error("the second delta took the first's copy")
		}
	})
	t.Run("other bytes", func(t *testing.T) {
		c, copies := screen(t, frag("x"), frag("y"))
		// From the first op of other bytes, the rest is parsed, even an
		// op of a fragment's bytes.
		write(t, c, hotty.Delta("v", []hotty.Op{hotty.OpInner("a", "<p>z</p>"), hotty.OpInner("b", "<p>x</p>")}))
		if child(c, "a") == copies[0] || child(c, "b") == copies[0] || !strings.Contains(c.Lines()[0], "z") {
			t.Error("a delta of other bytes took the offer, or the rest of it after")
		}
	})
	t.Run("another surface's delta", func(t *testing.T) {
		c, _ := screen(t, frag("x"))
		write(t, c, hotty.Delta("w", []hotty.Op{hotty.OpInner("a", "<p>x</p>")}))
		if c.frags != nil {
			t.Error("another surface's delta left the offer")
		}
	})
	t.Run("another surface's offer", func(t *testing.T) {
		c, copies := screen(t)
		screenHost{c}.OfferDelta("w", []rt.Fragment{frag("x")})
		copies = append(copies, c.frags.frags[0].nodes[0])
		write(t, c, hotty.Delta("v", []hotty.Op{hotty.OpInner("a", "<p>x</p>")}))
		if child(c, "a") == copies[0] {
			t.Error("a delta took another surface's offer")
		}
	})
	t.Run("a document between", func(t *testing.T) {
		c, copies := screen(t, frag("x"))
		write(t, c, hotty.Doc("v", doc))
		write(t, c, hotty.Delta("v", []hotty.Op{hotty.OpInner("a", "<p>x</p>")}))
		if child(c, "a") == copies[0] {
			t.Error("the offer outlived a document")
		}
	})
	t.Run("del", func(t *testing.T) {
		for _, del := range []string{hotty.Del("v"), hotty.DelAll()} {
			c, _ := screen(t, frag("x"))
			write(t, c, del)
			if c.frags != nil {
				t.Errorf("%q kept the offer", del)
			}
		}
	})
	t.Run("compressed", func(t *testing.T) {
		long := strings.Repeat("a long text, of some length ", 200)
		c, copies := screen(t, frag(long))
		cmd := hotty.Delta("v", []hotty.Op{hotty.OpInner("a", "<p>"+long+"</p>")})
		if !strings.Contains(cmd, "o=z") {
			t.Fatal("the delta isn't compressed")
		}
		write(t, c, cmd)
		if child(c, "a") != copies[0] {
			t.Error("a compressed delta didn't take its offer")
		}
	})
	t.Run("lent", func(t *testing.T) {
		f := frag("x")
		c, _ := screen(t, f)
		f.Nodes[0].FirstChild.Data = "changed"
		f.Nodes[0].Attr = append(f.Nodes[0].Attr, html.Attribute{Key: "class", Val: "hidden"})
		write(t, c, hotty.Delta("v", []hotty.Op{hotty.OpInner("a", "<p>x</p>")}))
		if l := c.Lines()[0]; !strings.HasPrefix(l, "x ") {
			t.Errorf("the Screen shows %q, from the surface's nodes as they were after", l)
		}
	})
	t.Run("one copy for many", func(t *testing.T) {
		// Every fragment's nodes in one array: a copy each, sharing it.
		c, copies := screen(t, frag("x"), frag("y"), frag("z"))
		size := unsafe.Sizeof(html.Node{})
		for i := 1; i < len(copies); i++ {
			// A <p> and its text a fragment, one after the other.
			if uintptr(unsafe.Pointer(copies[i]))-uintptr(unsafe.Pointer(copies[i-1])) != 2*size {
				t.Fatal("the fragments' copies aren't in one array")
			}
		}
		if cap(c.frags.frags[0].nodes) != 1 {
			t.Error("a fragment's nodes have room for the next's")
		}
	})
}

// parseDoc and renderDoc are markup as a body's children and back.
func parseDoc(markup string) *html.Node {
	body := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	ns, _ := html.ParseFragment(strings.NewReader(markup), body)
	for _, n := range ns {
		body.AppendChild(n)
	}
	return body
}

func renderDoc(body *html.Node) string {
	var b strings.Builder
	for k := body.FirstChild; k != nil; k = k.NextSibling {
		html.Render(&b, k)
	}
	return b.String()
}

// walkStep takes one random step on o, and says what it was.
func walkStep(t testing.TB, r *rand.Rand, o *onScreen, cols, rows int) string {
	var act string
	switch k := r.IntN(10); {
	case k < 3:
		x, y := r.IntN(cols), r.IntN(rows)
		o.c.Click(x, y)
		act = fmt.Sprintf("click %d,%d", x, y)
	case k < 5:
		key := []string{"tab", "shift+tab", "enter", "space", "arrow_down", "arrow_up", "a", "escape"}[r.IntN(8)]
		o.c.Key(key)
		act = "key " + key
	case k < 7:
		x, y := r.IntN(cols), r.IntN(rows)
		o.c.Move(x, y, 0)
		act = fmt.Sprintf("move %d,%d", x, y)
	case k == 7:
		o.c.Wheel(r.IntN(cols), r.IntN(rows), r.IntN(5)-2)
		act = "wheel"
	default:
		act = "document"
		o.document(t)
	}
	if o.stale {
		act += ", refused, document"
		o.document(t)
	}
	return act
}

// counter is a view whose document is another at each render.
func counter() templ.Component {
	n := 0
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		n++
		_, err := fmt.Fprintf(w, `<p id="n">render %d</p>`, n)
		return err
	})
}

func TestOfferEdges(t *testing.T) {
	view := templ.Raw(`<div id="a" class="flex gap-1"><b>hi</b><span>there</span></div>`)
	t.Run("taken", func(t *testing.T) {
		o := newOnScreen(t, "v", view, "", 20, 2, true)
		doc, _ := o.s.Document()
		offered := o.c.offer.body
		o.c.Write([]byte(hotty.Doc("v", doc)))
		if o.c.body != offered || o.c.offer != nil {
			t.Errorf("the Screen didn't take the offer, or kept it")
		}
	})
	t.Run("the same document twice", func(t *testing.T) {
		o := newOnScreen(t, "v", view, "", 20, 2, true)
		want := newOnScreen(t, "v", view, "", 20, 2, false)
		doc, _ := o.s.Document()
		want.document(t)
		for range 2 {
			o.c.Write([]byte(hotty.Doc("v", doc)))
			if o.c.offer != nil {
				t.Error("an offer outlived its document")
			}
			sameCells(t, "twice", o.c.Cells(), want.c.Cells())
		}
	})
	t.Run("two documents before a write", func(t *testing.T) {
		o := newOnScreen(t, "v", counter(), "", 20, 1, true)
		d1, _ := o.s.Document()
		d2, _ := o.s.Document()
		offered := o.c.offer.body
		o.c.Write([]byte(hotty.Doc("v", d1)))
		if o.c.body == offered || o.c.offer != nil || !strings.Contains(o.c.Lines()[0], "render 1") {
			t.Errorf("the first document shows %q, from the second's offer or with it kept", o.c.Lines()[0])
		}
		o.c.Write([]byte(hotty.Doc("v", d2)))
		if !strings.Contains(o.c.Lines()[0], "render 2") {
			t.Errorf("the second document shows %q", o.c.Lines()[0])
		}
	})
	t.Run("two surfaces on a Screen", func(t *testing.T) {
		o := newOnScreen(t, "a", view, "", 20, 2, true)
		b := surf.New("b", view, "")
		b.Attach(offering{o}, hotty.Caps{})
		da, _ := o.s.Document()
		b.Document()
		offered := o.c.offer.body
		o.c.Write([]byte(hotty.Doc("a", da)))
		if o.c.body == offered || o.c.name != "a" || o.c.offer != nil {
			t.Error("a's document took b's offer, or kept it")
		}
	})
	t.Run("a tee", func(t *testing.T) {
		o := newOnScreen(t, "v", view, "", 20, 2, false)
		var b bytes.Buffer
		o.s.Attach(io.MultiWriter(offering{o}, &b), hotty.Caps{})
		o.s.Document()
		if o.c.offer != nil {
			t.Error("a tee's Screen was offered the tree")
		}
	})
	t.Run("compressed", func(t *testing.T) {
		big := templ.Raw(`<div id="a">` + strings.Repeat(`<p class="text-sm">a row of text, at some length</p>`, 200) + `</div>`)
		o := newOnScreen(t, "v", big, "", 40, 4, true)
		doc, _ := o.s.Document()
		offered := o.c.offer.body
		cmd := hotty.Doc("v", doc)
		if !strings.Contains(cmd, "o=z") {
			t.Fatal("the document isn't compressed")
		}
		o.c.Write([]byte(cmd))
		if o.c.body != offered {
			t.Error("a compressed document didn't take its offer")
		}
	})
	t.Run("other bytes", func(t *testing.T) {
		o := newOnScreen(t, "v", view, "", 20, 2, true)
		doc, _ := o.s.Document()
		offered := o.c.offer.body
		o.c.Write([]byte(hotty.Doc("v", strings.Replace(doc, "hi", "ho", 1))))
		if o.c.body == offered || !strings.Contains(o.c.Lines()[0], "ho") {
			t.Errorf("a document of other bytes shows %q, from the offer", o.c.Lines()[0])
		}
	})
	t.Run("del", func(t *testing.T) {
		for _, del := range []string{hotty.Del("v"), hotty.DelAll()} {
			o := newOnScreen(t, "v", view, "", 20, 2, true)
			o.s.Document()
			o.c.Write([]byte(del))
			if o.c.offer != nil {
				t.Errorf("%q kept the offer", del)
			}
		}
	})
	t.Run("lent", func(t *testing.T) {
		// What the surface does to its tree after the offer doesn't
		// reach the Screen's copy.
		c := New(20, 2, nil)
		n := &html.Node{Type: html.ElementNode, Data: "p", DataAtom: atom.P, Attr: []html.Attribute{{Key: "id", Val: "p"}}}
		n.AppendChild(&html.Node{Type: html.TextNode, Data: "one"})
		doc := `<p id="p">one</p>`
		screenHost{c}.OfferDocument("v", doc, []*html.Node{n})
		n.Attr[0].Val = "q"
		n.FirstChild.Data = "two"
		n.AppendChild(&html.Node{Type: html.TextNode, Data: "three"})
		c.Write([]byte(hotty.Doc("v", doc)))
		if l := c.Lines()[0]; !strings.HasPrefix(l, "one ") || strings.Contains(l, "three") || c.byID("p") == nil {
			t.Errorf("the Screen shows %q, from the surface's tree as it was after", l)
		}
	})
	t.Run("a namespaced attribute", func(t *testing.T) {
		// An icon's <use xlink:href>: its attribute's namespace is copied.
		o := newOnScreen(t, "v", templ.Raw(`<svg><a xlink:href="#x"><use xlink:href="#y"></use></a></svg>`), "", 20, 2, true)
		o.checked(t)
		o.s.Document()
		n := 0
		walk(o.c.offer.body, func(k *html.Node) {
			for _, a := range k.Attr {
				if a.Key == "href" && a.Namespace == "xlink" {
					n++
				}
			}
		})
		if n != 2 {
			t.Errorf("the copy holds %d xlink:href, want 2", n)
		}
	})
	t.Run("detached", func(t *testing.T) {
		o := newOnScreen(t, "v", view, "", 20, 2, true)
		o.s.Attach(nil, hotty.Caps{})
		o.s.Document()
		if o.c.offer != nil {
			t.Error("a detached surface offered its tree")
		}
	})
}

// TestOfferSparesTheParse: a new block's document, offered, costs the
// Screen a copy where it cost a parse.
func TestOfferSparesTheParse(t *testing.T) {
	view := templ.Raw(`<div id="pill" class="flex items-center gap-1 px-1">` +
		`<span class="text-(--pill)">●</span><b class="font-bold">build</b>` +
		`<span class="text-muted-foreground">running, 3 of 7</span>` +
		`<span class="badge" data-variant="secondary">12s</span></div>`)
	cost := func(offer bool) (allocs float64, bytes uint64) {
		block := func() {
			newOnScreen(t, "v", view, "", 40, 1, offer).document(t)
		}
		allocs = testing.AllocsPerRun(50, block)
		var a, b runtime.MemStats
		runtime.ReadMemStats(&a)
		for range 50 {
			block()
		}
		runtime.ReadMemStats(&b)
		return allocs, (b.TotalAlloc - a.TotalAlloc) / 50
	}
	pa, pb := cost(false)
	oa, ob := cost(true)
	t.Logf("a new block: %.0f allocations, %d B parsed; %.0f, %d B offered", pa, pb, oa, ob)
	// The parse is two thirds of the allocations and some 6 KB; the
	// copy takes a few.
	if oa > pa/2 || ob+4096 > pb {
		t.Errorf("an offered document costs %.0f allocations and %d B, where a parse costs %.0f and %d B", oa, ob, pa, pb)
	}
}

// TestDeltaOfferSparesTheParse: a step whose delta carries markup (rows
// that come and go), offered, costs the Screen a copy of the rows where
// it cost a parse.
func TestDeltaOfferSparesTheParse(t *testing.T) {
	var rows [2]string
	for k := range rows {
		var b strings.Builder
		b.WriteString(`<div id="list" class="flex flex-col">`)
		for i := range 8 + 4*k {
			fmt.Fprintf(&b, `<div id="row%d" class="flex gap-1"><span class="text-(--pill)">●</span><b>row %d</b><span class="text-muted-foreground">%d of %d</span></div>`, i, i, i, 12)
		}
		b.WriteString(`</div>`)
		rows[k] = b.String()
	}
	cost := func(offer bool) (allocs float64, bytes uint64) {
		cur := rows[0]
		o := newOnScreen(t, "v", changing(&cur), "", 50, 14, offer)
		o.document(t)
		i := 0
		step := func() {
			i++
			cur = rows[i%2]
			if err := o.s.Update(); err != nil || o.stale {
				t.Fatal("the step was refused", err)
			}
		}
		allocs = testing.AllocsPerRun(50, step)
		var a, b runtime.MemStats
		runtime.ReadMemStats(&a)
		for range 50 {
			step()
		}
		runtime.ReadMemStats(&b)
		return allocs, (b.TotalAlloc - a.TotalAlloc) / 50
	}
	pa, pb := cost(false)
	oa, ob := cost(true)
	t.Logf("a step of rows that come and go: %.0f allocations, %d B parsed; %.0f, %d B offered", pa, pb, oa, ob)
	if oa > pa*3/4 || ob > pb {
		t.Errorf("an offered delta costs %.0f allocations and %d B, where a parse costs %.0f and %d B", oa, ob, pa, pb)
	}
}

// FuzzHandoff: on any markup a surface renders, and any it renders next,
// the Screen's copies of its offers, the document's and the delta's, are
// its tree (checked), and where the wire parses as that tree the Screen
// draws as one that parsed it. Seeded with FuzzParse's seeds and corpus
// and the kit's stories, each followed by the next.
func FuzzHandoff(f *testing.F) {
	seeds := []string{`<div class="a">b<span>c</span></div>`, `<ul><li>a</li></ul>`, `<p>x &amp; y</p>`, `<svg><path d="M0"></path></svg>`, `<table><tbody><tr><td>1</td></tr></tbody></table>`,
		`<svg><use xlink:href="#i"></use></svg>`}
	files, _ := filepath.Glob("../testdata/fuzz/FuzzParse/*")
	for _, p := range files {
		b, err := os.ReadFile(p)
		if err != nil {
			f.Fatal(err)
		}
		if _, v, ok := strings.Cut(string(b), "\nstring("); ok {
			if s, err := strconv.Unquote(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(v), ")"))); err == nil {
				seeds = append(seeds, s)
			}
		}
	}
	for _, st := range stories.All() {
		seeds = append(seeds, rootOf(storyMarkup(f, st)))
	}
	for i, s := range seeds {
		f.Add(s, seeds[(i+1)%len(seeds)])
	}
	f.Fuzz(func(t *testing.T, first, next string) {
		cur := first
		got := newOnScreen(t, "v", changing(&cur), "", 40, 8, true)
		h := got.checked(t)
		want := newOnScreen(t, "v", changing(&cur), "", 40, 8, false)
		if _, err := got.s.Document(); err != nil {
			return
		}
		got.document(t)
		want.document(t)
		if h.offers != 2 {
			t.Fatalf("%d offers", h.offers)
		}
		cur = next
		for _, o := range []*onScreen{got, want} {
			if err := o.s.Update(); err != nil {
				return
			}
			if o.stale {
				o.document(t)
			}
		}
		if len(h.diverged) == 0 {
			sameCells(t, strconv.Quote(first)+" then "+strconv.Quote(next), got.c.Cells(), want.c.Cells())
		}
	})
}

// rootOf is what a story's document holds in its #root, its markup.
func rootOf(doc string) string {
	_, in, ok := strings.Cut(doc, `<div id="root">`)
	if !ok {
		return doc
	}
	return strings.TrimSuffix(in, "</div>")
}
