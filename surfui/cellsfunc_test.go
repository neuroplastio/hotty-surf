package surfui_test

import (
	"strings"
	"testing"

	"neuroplast.io/hotty-surf/pkg/cellblock"
	"neuroplast.io/hotty-surf/surfui"
)

// A Graphic is its SVG on a host, as it is, and a block in the cells; a
// node with Cells is the host's own on a host.
func TestGraphic(t *testing.T) {
	g := surfui.Graphic(`<svg viewBox="0 0 4 4"></svg>`, func(w int) cellblock.Block {
		return cellblock.Block{Rows: [][]cellblock.Run{{{Col: 0, Width: w, Text: strings.Repeat("#", w)}}}}
	})
	if html := html(g); !strings.Contains(html, `<svg viewBox="0 0 4 4"></svg>`) || !strings.Contains(html, "u-graphic") {
		t.Errorf("a host's graphic: %s", html)
	}
	f := surfui.BlockOf(g)
	if f == nil {
		t.Fatal("no cells func")
	}
	if b := f(3, cellblock.Style{}); b.Height() != 1 || b.Width() != 3 {
		t.Errorf("block %+v", b)
	}

	n := surfui.El("div", surfui.Text("kid")).Cells(func(int, cellblock.Style) cellblock.Block { return cellblock.Block{} })
	if h := html(n); h != `<div id="r"><span id="r.0" class="u-text">kid</span></div>` {
		t.Errorf("Cells changed the host's tree: %s", h)
	}
	if surfui.BlockOf(surfui.El("div")) != nil {
		t.Error("a block on a node with none")
	}
}

func html(n *surfui.Node) string {
	surfui.Resolve(n, "r")
	return surfui.HTML(n)
}
