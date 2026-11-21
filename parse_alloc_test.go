package surf

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"unsafe"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// bytesPerRun is what f allocates a run, on average over runs, after one
// run to warm what it keeps.
func bytesPerRun(runs int, f func()) uint64 {
	f()
	var a, b runtime.MemStats
	runtime.ReadMemStats(&a)
	for range runs {
		f()
	}
	runtime.ReadMemStats(&b)
	return (b.TotalAlloc - a.TotalAlloc) / uint64(runs)
}

// needs is the room a tree's slabs need: its nodes under root, and its
// elements' attributes with the one more each that number may put first.
func needs(root *html.Node) (nodes, attrs int) {
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			nodes++
			if c.Type == html.ElementNode {
				attrs += len(c.Attr) + 1
			}
			walk(c)
		}
	}
	walk(root)
	return nodes, attrs
}

// A parse takes slabs about its render's size, whatever it is: its first
// sized to the markup, and one that runs out (a text after a void
// element, boolean attributes, which the estimate doesn't count) twice
// the last, never a whole slab for a small render.
func TestParseSlabsFit(t *testing.T) {
	nodeSize, attrSize := int(unsafe.Sizeof(html.Node{})), int(unsafe.Sizeof(html.Attribute{}))
	var pill strings.Builder
	pill.WriteString(`<div class="flex justify-end"><span class="grow"></span><div class="flex shrink-0 bg-card">`)
	for _, in := range []string{"AB", "CD"} {
		fmt.Fprintf(&pill, `<span class="avatar" data-size="sm"><span>%s</span></span>`, in)
	}
	pill.WriteString(`<span class="truncate text-muted-foreground"> typing…</span><button type="button" class="btn-xs-ghost" data-on="click">Take over</button></div></div>`)
	var big strings.Builder
	for i := range 400 {
		fmt.Fprintf(&big, `<p class="row"><span>%d</span></p>`, i)
	}
	for _, c := range []struct{ name, src string }{
		{"pill", pill.String()},
		{"texts after void elements", `<div><br>a<br>b<br>c<br>d<br>e<br>f<br>g<br>h<br>i<br>j</div>`},
		{"inputs with labels", strings.Repeat(`<label><input type="checkbox"> Remember me</label>`, 6)},
		{"boolean attributes", strings.Repeat(`<input disabled readonly required autofocus multiple>`, 8)},
		{"one text", `just text`},
		{"larger than a slab", big.String()},
	} {
		t.Run(c.name, func(t *testing.T) {
			var p parser
			src := []byte(c.src)
			var root *html.Node
			got := bytesPerRun(50, func() {
				root = newRoot()
				if !p.parse(root, src) {
					t.Fatal("the fast parser gave it up")
				}
			})
			nodes, attrs := needs(root)
			need := nodes*nodeSize + attrs*attrSize
			// The render's string, its first slabs, and at most one of
			// each that grew from them: under three times what the tree
			// needs, beside a floor of a grown slab of each.
			limit := uint64(len(src) + 3*need + minSlab*(nodeSize+attrSize) + 256)
			if nodes > nodeSlab {
				// Whole slabs, as many as it takes, and one more of each.
				limit = uint64(len(src) + (nodes/nodeSlab+2)*nodeSlab*nodeSize + (attrs/attrSlab+2)*attrSlab*attrSize)
			}
			t.Logf("%d nodes, %d attributes' room: %d B (%d needed, %d allowed)", nodes, attrs, got, need, limit)
			if got > limit {
				t.Errorf("a parse takes %d B, more than %d", got, limit)
			}
		})
	}
}

// A fill that rides the room a whole parse left, and runs out of it (its
// estimate fits, what it takes doesn't: a text after a void element,
// boolean attributes), grows from its own size, not the document's: for
// every room the document leaves near the fill's estimate, the fill takes
// a few KB, not a whole slab.
func TestFillSlabsFitTheRoomLeft(t *testing.T) {
	nodeSize, attrSize := int(unsafe.Sizeof(html.Node{})), int(unsafe.Sizeof(html.Attribute{}))
	inst := []byte(`<span>a<br>b<br>c<br>d<input type="checkbox" checked disabled required></span>`)
	var p parser
	fill := func() *html.Node {
		holder := &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div}
		if !p.parseIn(holder, nil, inst) {
			t.Fatal("the fast parser gave the fill up")
		}
		return holder
	}
	nodes, attrs := needs(fill())
	limit := uint64(len(inst) + 3*(nodes*nodeSize+attrs*attrSize) + minSlab*(nodeSize+attrSize) + 256)
	worst := uint64(0)
	for k := range 12 {
		for j := range 12 {
			// 300 nodes whose estimate is exact, then a node's room for
			// each <b></b> and an attribute's for each '=' in a text.
			doc := strings.Repeat(`<p class="a" id="b">x</p>`, 150) + strings.Repeat(`<b></b>`, k) + strings.Repeat(`<i>=</i>`, j)
			if !p.parse(newRoot(), []byte(doc)) {
				t.Fatal("the fast parser gave the document up")
			}
			roomN, roomA := cap(p.nodes)-len(p.nodes), cap(p.attrs)-len(p.attrs)
			var a, b runtime.MemStats
			runtime.ReadMemStats(&a)
			fill()
			runtime.ReadMemStats(&b)
			got := b.TotalAlloc - a.TotalAlloc
			worst = max(worst, got)
			if got > limit {
				t.Errorf("with %d nodes' and %d attributes' room left, a fill takes %d B, more than %d", roomN, roomA, got, limit)
			}
		}
	}
	t.Logf("a fill of %d nodes, %d attributes' room after a document: at most %d B (%d allowed)", nodes, attrs, worst, limit)
}

// A fill after a whole parse takes slabs sized to its own markup, the
// first as the later ones: not a whole slab where the parse left no room.
func TestFillSlabsFit(t *testing.T) {
	nodeSize, attrSize := int(unsafe.Sizeof(html.Node{})), int(unsafe.Sizeof(html.Attribute{}))
	var doc strings.Builder
	for i := range 120 {
		fmt.Fprintf(&doc, `<span class="pill" data-i="%d">tab %d</span>`, i, i)
	}
	inst := []byte(`<span class="flex shrink-0 text-(--pill)" data-on="click hover"><span class="whitespace-pre"> </span><span class="text-(--ok)">●</span><span class="truncate font-bold">tab 7</span><span class="text-(--pill)">×</span></span>`)
	var p parser
	fill := func() *html.Node {
		holder := &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div}
		if !p.parseIn(holder, nil, inst) {
			t.Fatal("the fast parser gave the fill up")
		}
		return holder
	}
	nodes, attrs := needs(fill())
	need := nodes*nodeSize + attrs*attrSize
	limit := uint64(len(inst) + 3*need + minSlab*(nodeSize+attrSize) + 256)
	for _, first := range []bool{true, false} {
		var got uint64
		if first {
			// One fill right after a whole parse, each time.
			src := []byte(doc.String())
			var total uint64
			for range 20 {
				if !p.parse(newRoot(), src) {
					t.Fatal("the fast parser gave the document up")
				}
				var a, b runtime.MemStats
				runtime.ReadMemStats(&a)
				fill()
				runtime.ReadMemStats(&b)
				total += b.TotalAlloc - a.TotalAlloc
			}
			got = total / 20
		} else {
			got = bytesPerRun(50, func() { fill() })
		}
		t.Logf("first %v: %d B a fill of %d nodes, %d attributes' room (%d needed, %d allowed)", first, got, nodes, attrs, need, limit)
		if !first {
			// Fills in a row share slabs, as they shared what a whole
			// slab left: 64 of them take a few of each, not one each.
			nodeSlabs, attrSlabs := map[*html.Node]bool{}, map[*html.Attribute]bool{}
			for range 64 {
				fill()
				nodeSlabs[unsafe.SliceData(p.nodes)] = true
				attrSlabs[unsafe.SliceData(p.attrs)] = true
			}
			if len(nodeSlabs) > 16 || len(attrSlabs) > 16 {
				t.Errorf("64 fills in a row take %d slabs of nodes and %d of attributes", len(nodeSlabs), len(attrSlabs))
			}
		}
	}
}
