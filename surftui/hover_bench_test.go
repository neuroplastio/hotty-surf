package surftui

import (
	"fmt"
	"strings"
	"testing"

	"neuroplast.io/hotty-go"
)

// BenchmarkHoverMove is the pointer moved from pill to pill along a strip
// of twenty, each lit by its hover: classes (colours only), and the
// Screen drawn into a program's grid: what a hover costs on the Screen
// alone, where BenchmarkCompiled's hover scene renders the strip again
// with the hovered pill's class.
func BenchmarkHoverMove(b *testing.B) {
	var d strings.Builder
	d.WriteString(`<div id="root" class="flex bg-(--strip-ground)">`)
	for i := range 20 {
		fmt.Fprintf(&d, `<span id="pill-%d" class="flex shrink-0 text-(--pill) hover:text-(--pill-hot) hover:bg-(--pill-hover)">`+
			`<span class="whitespace-pre"> </span><span class="text-(--ok)">●</span><span class="whitespace-pre"> </span>`+
			`<span class="truncate font-bold">tab %d</span><span class="whitespace-pre"> </span><span class="text-(--pill)">×</span></span>`, i, i)
	}
	d.WriteString(`</div>`)
	c := New(240, 1, func(hotty.Event) {})
	if _, err := c.Write([]byte(hotty.Doc("z", d.String()))); err != nil {
		b.Fatal(err)
	}
	g := make([][]Cell, 3)
	for y := range g {
		g[y] = make([]Cell, 244)
	}
	c.Draw(g, 0, 0)
	var at [20]int
	for i := range 20 {
		r := c.layout().boxes[c.byID(fmt.Sprintf("pill-%d", i))]
		at[i] = r.x + 2
	}
	b.ResetTimer()
	for i := range b.N {
		c.Move(at[i%20], 0, 0)
		c.Draw(g, 0, 0)
	}
}
