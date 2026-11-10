package surf

import (
	"fmt"
	"strconv"
	"strings"

	"neuroplast.io/hotty-surf/pkg/markup"
	"neuroplast.io/hotty-surf/surfui"
)

// The table's modifiers (docs/surf.md, "Table"): a <col>'s, which both
// modes call. They apply to a surfui.Col, and do nothing to another node
// but where they are the generic modifier's (`width`, `fill`).

var (
	colAligns = []string{"start", "end", "center"}
	colSorts  = []string{"none", "ascending", "descending"}
)

// ModColWidth is a col's width="8", width="auto" or width="fill".
func ModColWidth(n *surfui.Node, v string) error {
	switch v {
	case "auto":
		n.Auto()
	case "fill":
		n.FillCol()
	default:
		w, err := strconv.Atoi(v)
		if err != nil || w < 0 {
			return fmt.Errorf("width is a number of cells, auto or fill, not %q", v)
		}
		n.Width(w)
	}
	return nil
}

// ModColFill is a col's bare fill: it takes the room left, as width="fill".
func ModColFill(n *surfui.Node, v bool) {
	if v {
		n.FillCol()
	}
}

// ModColMax is a col's max="40" (cells) or max="40%" (of the table).
func ModColMax(n *surfui.Node, v string) error {
	num, pct := strings.CutSuffix(v, "%")
	w, err := strconv.Atoi(num)
	if err != nil || w < 0 || pct && w > 100 {
		return fmt.Errorf("max is a number of cells or a percentage, 40 or 40%%, not %q", v)
	}
	if pct {
		n.ColMax(0, w)
	} else {
		n.ColMax(w, 0)
	}
	return nil
}

// ModColAlign is a col's align="start", "end" or "center".
func ModColAlign(n *surfui.Node, v string) error {
	if !contains(colAligns, v) {
		return fmt.Errorf("align is %s, not %q", orList(colAligns), v)
	}
	n.Align(v)
	return nil
}

// ModColSort is a col's sort="none", "ascending" or "descending".
func ModColSort(n *surfui.Node, v string) error {
	if !contains(colSorts, v) {
		return fmt.Errorf("sort is %s, not %q", orList(colSorts), v)
	}
	n.Sort(v)
	return nil
}

// ModColShow is a col's show, which `:if` on a col is: false drops the
// column and its cells.
func ModColShow(n *surfui.Node, v bool) { n.Show(v) }

// tableAliases is a <table> whose <tr>, <th> and <td> (with <thead>,
// <tbody> and <tfoot> around them) are the shape a Markdown table
// converts to one for one: a row of <th> is the columns (<col>, its
// align="left" or "right" start or end), every other <tr> is an <item>
// whose <td> and <th> are <line>s, and a table with no <th> row gets
// <col>s with no heading. The static form has no selection: nothing in
// it says one. A table with none of them is returned as it is.
func tableAliases(n *markup.Node) *markup.Node {
	var kids []*markup.Node
	var flat func(ks []*markup.Node)
	flat = func(ks []*markup.Node) {
		for _, k := range ks {
			if k.Kind == markup.Element && (k.Tag == "thead" || k.Tag == "tbody" || k.Tag == "tfoot") {
				flat(k.Kids)
				continue
			}
			kids = append(kids, k)
		}
	}
	flat(n.Kids)
	rows, head, widest := false, false, 0
	for _, k := range kids {
		if k.Kind != markup.Element || k.Tag != "tr" {
			continue
		}
		rows = true
		cells, ths := 0, 0
		for _, c := range k.Kids {
			if c.Kind == markup.Element && (c.Tag == "td" || c.Tag == "th") {
				cells++
				if c.Tag == "th" {
					ths++
				}
			}
		}
		head = head || cells > 0 && ths == cells
		widest = max(widest, cells)
	}
	if !rows {
		return n
	}
	out := *n
	out.Kids = nil
	if !head {
		for range widest {
			out.Kids = append(out.Kids, &markup.Node{Kind: markup.Element, Pos: n.Pos, Tag: "col"})
		}
	}
	for _, k := range kids {
		if k.Kind != markup.Element || k.Tag != "tr" {
			out.Kids = append(out.Kids, k)
			continue
		}
		onlyTh, cells := true, 0
		for _, c := range k.Kids {
			if c.Kind == markup.Element && (c.Tag == "td" || c.Tag == "th") {
				cells++
				onlyTh = onlyTh && c.Tag == "th"
			}
		}
		if cells > 0 && onlyTh {
			for _, c := range k.Kids {
				if c.Kind != markup.Element {
					continue
				}
				col := *c
				col.Tag = "col"
				col.Attrs = nil
				for _, a := range c.Attrs {
					if a.Name == "align" {
						switch a.Value {
						case "left":
							a.Value = "start"
						case "right":
							a.Value = "end"
						}
					}
					col.Attrs = append(col.Attrs, a)
				}
				out.Kids = append(out.Kids, &col)
			}
			continue
		}
		item := *k
		item.Tag = "item"
		item.Kids = nil
		for _, c := range k.Kids {
			if c.Kind != markup.Element || c.Tag != "td" && c.Tag != "th" {
				item.Kids = append(item.Kids, c)
				continue
			}
			cell := *c
			cell.Tag = "line"
			cell.Attrs = nil
			for _, a := range c.Attrs {
				if a.Name != "align" && a.Name != "colspan" && a.Name != "rowspan" {
					cell.Attrs = append(cell.Attrs, a)
				}
			}
			if c.Tag == "th" {
				cell.Attrs = append(cell.Attrs, markup.Attr{Name: "semibold", Bare: true, Pos: c.Pos, ValuePos: c.Pos})
			}
			item.Kids = append(item.Kids, &cell)
		}
		out.Kids = append(out.Kids, &item)
	}
	return &out
}

// tablesRead reads the table aliases (tableAliases) of every <table> in
// the tree under root, at load, before anything is compiled. It walks
// with a stack of its own: TinyGo's stacks are small, and a template is
// as deep as MaxDepth.
func tablesRead(root *markup.Node) *markup.Node {
	if root.Kind == markup.Element && root.Tag == "table" {
		root = tableAliases(root)
	}
	todo := []*markup.Node{root}
	for len(todo) > 0 {
		n := todo[len(todo)-1]
		todo = todo[:len(todo)-1]
		for i, k := range n.Kids {
			if k.Kind == markup.Element && k.Tag == "table" {
				k = tableAliases(k)
				n.Kids[i] = k
			}
			todo = append(todo, k)
		}
	}
	return root
}
