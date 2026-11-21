package surf

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"unsafe"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// icon is an <svg> no other test parses, so that it is new to svgs.
func icon(name string) string {
	return fmt.Sprintf(`<svg viewBox="0 0 24 24" class="size-4" data-icon="%s"><path d="M12,2A10,10 0 0,0 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2Z"></path><use xlink:href="#%s"></use></svg>`, name, name)
}

// pillWith is a pill holding svg, as a new block's render.
func pillWith(svg string) []byte {
	return []byte(`<span class="flex items-center gap-1">` + svg + `<b>build</b><span class="text-muted-foreground">running</span></span>`)
}

// TestSvgsAreShared: an <svg> one surface's render parsed is copied, not
// parsed, in every other surface's: a new block's parser finds the icons
// the blocks before it parsed.
func TestSvgsAreShared(t *testing.T) {
	src := pillWith(icon(t.Name()))
	bytes := func() uint64 {
		var p parser // a new block's
		var a, b runtime.MemStats
		runtime.ReadMemStats(&a)
		if !p.parse(newRoot(), src) {
			t.Fatal("the fast parser gave the pill up")
		}
		runtime.ReadMemStats(&b)
		return b.TotalAlloc - a.TotalAlloc
	}
	first, next := bytes(), bytes()
	t.Logf("a new block's pill with a new icon: %d B; with an icon parsed before: %d B", first, next)
	if next*2 > first {
		t.Errorf("a parse of an icon parsed before takes %d B, where its first took %d B", next, first)
	}
	var p parser
	if n := testing.AllocsPerRun(20, func() { p.parse(newRoot(), src) }); n > 6 {
		t.Errorf("a parse of the pill with a known icon takes %.0f allocations: more than its root, its string, its slabs and the lookup's", n)
	}
}

// TestSvgCopiesAreTheirOwn: a render changes its tree (an id put first, a
// class taken out), and what it changes is its copy's alone: neither the
// tree kept for the markup nor another render's copy.
func TestSvgCopiesAreTheirOwn(t *testing.T) {
	svg := icon(t.Name())
	src := pillWith(svg)
	var p, q parser
	a, b := newRoot(), newRoot()
	if !p.parse(a, src) || !q.parse(b, src) {
		t.Fatal("the fast parser gave the pill up")
	}
	sa, sb := a.FirstChild.FirstChild, b.FirstChild.FirstChild
	if sa.DataAtom != atom.Svg || sb.DataAtom != atom.Svg {
		t.Fatal("no <svg> where the pill has it")
	}
	sa.Attr[0].Val = "changed"
	sa.Attr = append(sa.Attr, html.Attribute{Key: "id", Val: "s1"})
	sa.Attr = append(sa.Attr, html.Attribute{Key: "data-x", Val: "past the room"})
	sa.FirstChild.Attr[0].Val = "M0"
	sa.FirstChild.Data = "rect"
	if len(sb.Attr) != 3 || sb.Attr[0].Val == "changed" || sb.FirstChild.Attr[0].Val == "M0" || sb.FirstChild.Data != "path" {
		t.Error("a change to one render's <svg> reached another's")
	}
	kept := svgs.get(svg)
	if kept == nil || kept.Attr[0].Val == "changed" || kept.FirstChild.Attr[0].Val == "M0" {
		t.Error("a change to a render's <svg> reached the tree kept for its markup")
	}
	// The copy's attribute lists have room for the id only: an append
	// past it moves the list, and touches no other node's.
	use := sb.FirstChild.NextSibling
	if use == nil || use.Attr[0].Namespace != "xlink" {
		t.Fatal("the copy lost <use>'s xlink:href")
	}
	if cap(use.Attr) != len(use.Attr)+1 {
		t.Errorf("<use>'s attributes have room for %d more, want 1 (the id)", cap(use.Attr)-len(use.Attr))
	}
}

// TestSvgsForgetTheUnused: svgs keeps an <svg> used in every generation,
// forgets one used in none, and keeps none longer than an icon's markup.
func TestSvgsForgetTheUnused(t *testing.T) {
	hot, cold := icon(t.Name()+"-hot"), icon(t.Name()+"-cold")
	var p parser
	parse := func(svg string) {
		if !p.parse(newRoot(), pillWith(svg)) {
			t.Fatal("the fast parser gave the pill up")
		}
	}
	parse(hot)
	parse(cold)
	for i := range 3 * svgCache {
		parse(icon(fmt.Sprintf("%s-%d", t.Name(), i)))
		if i%(svgCache/4) == 0 {
			parse(hot)
		}
	}
	if svgs.get(hot) == nil {
		t.Error("an <svg> used in every generation was forgotten")
	}
	if svgs.get(cold) != nil {
		t.Error("an <svg> unused for two generations is still kept")
	}
	svgs.mu.Lock()
	n := len(svgs.new) + len(svgs.old)
	svgs.mu.Unlock()
	if n > 2*svgCache {
		t.Errorf("svgs keeps %d trees, more than two generations of %d", n, svgCache)
	}
	// Markup of the most an icon's length, each generation up to its
	// bytes: no more than two generations' of them.
	for i := range 3 * svgBytes / svgMax {
		parse(fmt.Sprintf(`<svg data-i="%d"><path d="%s"></path></svg>`, i, strings.Repeat("M1,1L2,2", svgMax/8-8)))
	}
	svgs.mu.Lock()
	bytes := 0
	for _, gen := range []map[string]*html.Node{svgs.new, svgs.old} {
		for k := range gen {
			bytes += len(k)
		}
	}
	svgs.mu.Unlock()
	if bytes > 2*svgBytes {
		t.Errorf("svgs keeps the trees of %d B of markup, more than two generations of %d", bytes, svgBytes)
	}
	// A small chart, 6 KB, is longer than any icon: its trees, about
	// eight times its markup, are not kept.
	chart := `<svg class="chart">` + strings.Repeat(`<rect x="1" y="2" width="3" height="4"></rect>`, 6<<10/46) + `</svg>`
	parse(chart)
	if svgs.get(chart) != nil {
		t.Errorf("an <svg> of %d B, a chart's, is kept", len(chart))
	}
}

// TestSvgKeysAreTheirOwn: a kept markup is a copy of its own, not a
// slice of the render it was found in, which it would keep: after a
// first parse, and after it moves from the older generation to the
// newer.
func TestSvgKeysAreTheirOwn(t *testing.T) {
	x := icon(t.Name())
	// pinned says whether x's key in gen lies in p's render.
	pinned := func(p *parser, gen map[string]*html.Node) bool {
		lo := uintptr(unsafe.Pointer(unsafe.StringData(p.src)))
		for k := range gen {
			if k == x {
				a := uintptr(unsafe.Pointer(unsafe.StringData(k)))
				return a >= lo && a < lo+uintptr(len(p.src))
			}
		}
		t.Fatal("the icon isn't kept where it should be")
		return false
	}
	var p parser
	if !p.parse(newRoot(), pillWith(x)) {
		t.Fatal("the fast parser gave the pill up")
	}
	svgs.mu.Lock()
	first := pinned(&p, svgs.new)
	svgs.mu.Unlock()
	if first {
		t.Error("a first parse's key is a slice of its render")
	}
	// Into the older generation, then found there by a render of its
	// own, which moves it to the newer.
	for i := 0; ; i++ {
		svgs.mu.Lock()
		_, old := svgs.old[x]
		svgs.mu.Unlock()
		if old {
			break
		}
		if !p.parse(newRoot(), pillWith(icon(fmt.Sprintf("%s-%d", t.Name(), i)))) {
			t.Fatal("the fast parser gave the pill up")
		}
	}
	var q parser
	if !q.parse(newRoot(), []byte(strings.Repeat(`<span class="pad">padding</span>`, 300)+string(pillWith(x)))) {
		t.Fatal("the fast parser gave the render up")
	}
	svgs.mu.Lock()
	promoted := pinned(&q, svgs.new)
	svgs.mu.Unlock()
	if promoted {
		t.Errorf("a key moved to the newer generation is a slice of the %d B render it was found in", len(q.src))
	}
}

// TestSvgsAcrossGoroutines: surfaces render on their own goroutines, each
// with its own parser, over the same icons (go test -race).
func TestSvgsAcrossGoroutines(t *testing.T) {
	var wg sync.WaitGroup
	for g := range 8 {
		wg.Go(func() {
			var p parser
			for i := range 200 {
				src := pillWith(icon(fmt.Sprintf("%s-%d", t.Name(), (i+g)%40)))
				root := newRoot()
				if !p.parse(root, src) {
					t.Error("the fast parser gave the pill up")
					return
				}
				root.FirstChild.FirstChild.Attr[0].Val = "mine"
			}
		})
	}
	wg.Wait()
}
