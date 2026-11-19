package cells

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"testing"

	"golang.org/x/net/html"
	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/internal/stories"
)

// A delta of attributes and texts forgets what its ops reach, and no
// more: a Screen given it draws as one that forgets all it keeps.
func TestDeltaForgetsWhatItReaches(t *testing.T) {
	r := rand.New(rand.NewPCG(7, 8))
	for _, st := range stories.All() {
		t.Run(st.ID, func(t *testing.T) {
			body := parseBody(t, storyMarkup(t, st))
			i := 0
			walk(body, func(n *html.Node) {
				if n.Type == html.ElementNode && n != body {
					set(n, "id", "e"+strconv.Itoa(i))
					i++
				}
			})
			m0 := render(t, body)
			cols, rows := 50, 20
			c, all := New(cols, rows, nil), New(cols, rows, nil)
			for _, x := range []*Screen{c, all} {
				if _, err := x.Write([]byte(hotty.Doc("s", m0))); err != nil {
					t.Fatal(err)
				}
				x.Cells()
			}
			last := parseBody(t, m0)
			for round := range 4 {
				b := parseBody(t, render(t, last))
				mutate(r, b, true)
				m1 := render(t, b)
				next := parseBody(t, m1)
				if !sameSlots(last, next) {
					continue
				}
				ops := deltaOps(last, next)
				if len(ops) == 0 {
					continue
				}
				d := []byte(hotty.Delta("s", ops))
				for _, x := range []*Screen{c, all} {
					if _, err := x.Write(d); err != nil {
						t.Fatal(err)
					}
				}
				all.kept, all.lay = nil, nil
				sameCells(t, fmt.Sprintf("round %d", round), c.Cells(), all.Cells())
				last = next
			}
		})
	}
}

// deltaOps are the attr, unattr and text ops that make a's tree b's, of
// the same shape, for elements that hold only text.
func deltaOps(a, b *html.Node) []hotty.Op {
	var ops []hotty.Op
	var walk2 func(x, y *html.Node)
	walk2 = func(x, y *html.Node) {
		if x.Type == html.ElementNode && x.Parent != nil {
			id := attr(x, "id")
			for _, k := range y.Attr {
				if v, ok := get(x, k.Key); !ok || v != k.Val {
					ops = append(ops, hotty.OpAttr(id, k.Key, k.Val))
				}
			}
			for _, k := range x.Attr {
				if !slices.ContainsFunc(y.Attr, func(a html.Attribute) bool { return a.Key == k.Key }) {
					ops = append(ops, hotty.OpUnattr(id, k.Key))
				}
			}
			only := func(n *html.Node) bool {
				return n.FirstChild == nil || n.FirstChild.Type == html.TextNode && n.FirstChild.NextSibling == nil
			}
			if only(x) && only(y) && textOf(x) != textOf(y) && textKept(x) && !(x.Namespace == "" && voids[x.DataAtom]) {
				ops = append(ops, hotty.OpText(id, textOf(y)))
			}
		}
		kx, ky := x.FirstChild, y.FirstChild
		for {
			for kx != nil && kx.Type != html.ElementNode {
				kx = kx.NextSibling
			}
			for ky != nil && ky.Type != html.ElementNode {
				ky = ky.NextSibling
			}
			if kx == nil || ky == nil {
				return
			}
			walk2(kx, ky)
			kx, ky = kx.NextSibling, ky.NextSibling
		}
	}
	walk2(a, b)
	return ops
}
