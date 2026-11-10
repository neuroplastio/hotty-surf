package surf

import (
	"strings"
	"testing"
	"testing/fstest"

	cellite "neuroplast.io/cel-lite"
	"neuroplast.io/hotty-surf/surfui"
)

func tableObj(tree bool, rows ...string) dynObj {
	return dynObj{&Shape{Name: "threads", Component: true, Fields: []Field{
		{Name: "tree", Type: cellite.Dyn}, {Name: "rows", Type: cellite.Dyn},
	}, Methods: []Method{{Name: "noop", Args: []cellite.Type{cellite.String}}}}, []cellite.Value{cellite.BoolValue(tree), cellite.ListValue(strList(rows))}}
}

func tableTree(t *testing.T, src string, obj dynObj) *surfui.Node {
	t.Helper()
	set, err := Load(fstest.MapFS{"threads.surf": {Data: []byte(src)}}, []*Shape{obj.shape})
	if err != nil {
		t.Fatal(err)
	}
	root, errs := set.Render(obj)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	return root
}

const threadsSrc = `<surface density="dense">
  <table key="t">
    <col width="2"/>
    <col :if="tree" width="auto" max="40%">document</col>
    <col fill sort="descending">thread</col>
    <col align="end" muted>age</col>
    <item :each="r in rows" :key="r">
      <glyph>●</glyph><text>doc-{{ r }}.md</text><text>{{ r }}</text><text>3h</text>
    </item>
    <empty><text muted>No threads yet.</text></empty>
  </table>
</surface>`

// A table's columns are declared once: its cells are as wide as their
// column's widest, a fill column grows, a col's :if drops its cells too,
// and an empty table shows its empty.
func TestTable(t *testing.T) {
	root := tableTree(t, threadsSrc, tableObj(true, "a", "bbbbbb"))
	h := html(root)
	for _, want := range []string{
		`class="u-stack u-table"`, `class="u-row u-thead u-gap2"`, `aria-sort="descending"`,
		`style="--u-w:2"`, `style="--u-w:13;--u-maxp:40"`, // doc-bbbbbb.md is 13 cells
		`class="u-row u-cell u-grow"`, // the fill column's heading
	} {
		if !strings.Contains(h, want) {
			t.Errorf("no %s in\n%s", want, h)
		}
	}
	// Without the tree column its cells go: four cells a row became three.
	h = html(tableTree(t, threadsSrc, tableObj(false, "a")))
	if strings.Contains(h, "doc-a.md") || strings.Contains(h, "document") {
		t.Errorf("a dropped column's cells stay:\n%s", h)
	}
	if !strings.Contains(h, ">a</span>") {
		t.Errorf("the thread cell is gone:\n%s", h)
	}
	// No rows: the empty, with its header.
	h = html(tableTree(t, threadsSrc, tableObj(true)))
	if !strings.Contains(h, "No threads yet.") || strings.Contains(h, "u-table-body") {
		t.Errorf("empty:\n%s", h)
	}
}

// A Markdown table converts one to one: a row of <th> is the columns,
// the rest are rows, and no selection is drawn.
func TestTableAliases(t *testing.T) {
	root := tableTree(t, `<surface><table>
	  <tr><th align="left">Flag</th><th align="right">Default</th></tr>
	  <tr><td>--theme</td><td><text>graphite</text></td></tr>
	</table></surface>`, tableObj(false))
	h := html(root)
	for _, want := range []string{">Flag<", ">Default<", "--theme", "graphite", `role="columnheader"`, "u-table-ruled"} {
		if !strings.Contains(h, want) {
			t.Errorf("no %s in\n%s", want, h)
		}
	}
	if strings.Contains(h, "is-sel") {
		t.Errorf("a static table has a selection:\n%s", h)
	}
	// With no <th>, the columns have no heading and no header is drawn.
	h = html(tableTree(t, `<surface><table><tr><td>a</td><td>b</td></tr></table></surface>`, tableObj(false)))
	if strings.Contains(h, "u-thead") || !strings.Contains(h, ">b<") {
		t.Errorf("headless:\n%s", h)
	}
}

func TestTableRefusals(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{`<surface><table><col width="wide"/></table></surface>`, `width is a number of cells, auto or fill, not "wide"`},
		{`<surface><table><col max="200%"/></table></surface>`, `max is a number of cells or a percentage`},
		{`<surface><table><col align="left"/></table></surface>`, `align is start, end or center, not "left"`},
		{`<surface><table><col sort="up"/></table></surface>`, `sort is none, ascending or descending, not "up"`},
		{`<surface><text align="end">x</text></surface>`, `<text> takes no align`},
		{`<surface><tr/></surface>`, `<tr> is no widget`},
	} {
		_, err := Load(fstest.MapFS{"threads.surf": {Data: []byte(c.src)}}, []*Shape{tableObj(false).shape})
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: got %v, want %s", c.src, err, c.want)
		}
	}
}

// $range in a table's foot says the rows it shows, and is a load error
// anywhere else; the same tree resolved again says it again.
func TestTableRange(t *testing.T) {
	src := `<surface density="dense"><table height="7" scroll>
	  <col>name</col>
	  <item :each="r in rows" :key="r"><text>{{ r }}</text></item>
	  <text key="range">{{ $range }}</text>
	</table></surface>`
	h := html(tableTree(t, src, tableObj(false, "a", "b", "c", "d", "e", "f")))
	if !strings.Contains(h, ">1–2 of 6<") { // 7 rows high: header, rule and foot take 3; hairlines, so 2 rows
		t.Errorf("range:\n%s", h)
	}
	h = html(tableTree(t, `<surface><table><col>n</col><item><text>x</text></item><text>{{ $range }}</text></table></surface>`, tableObj(false)))
	if !strings.Contains(h, ">1 of 1<") {
		t.Errorf("all shown:\n%s", h)
	}
	_, err := Load(fstest.MapFS{"threads.surf": {Data: []byte(`<surface><text>{{ $range }}</text></surface>`)}}, []*Shape{tableObj(false).shape})
	if err == nil || !strings.Contains(err.Error(), "$range is the rows a <table> shows") {
		t.Errorf("outside a table: %v", err)
	}
}

// @activate on an item, and a table's foot, in a template.
func TestTableActivate(t *testing.T) {
	root := tableTree(t, `<surface><table>
	  <col>name</col>
	  <item selected key="a" @activate="noop($value)"><text>a</text></item>
	</table></surface>`, tableObj(false))
	h := html(root)
	body := surfui.Kids(surfui.Kids(root)[0])[2]
	keys := surfui.Bindings(body)
	if !strings.Contains(h, `data-on="click"`) || len(keys) != 1 || keys[0].Key != "enter" {
		t.Errorf("activate: %+v\n%s", keys, h)
	}
	if a, ok := keys[0].Act.(surfui.Activation); !ok || a.Value != "a" {
		t.Errorf("Enter's action: %+v", keys[0].Act)
	}
}
