package surfui

import (
	"strings"
	"sync"
	"testing"
)

var m1 = Metrics{CellW: 10, CellH: 21.25, Scale: 1.6}

func surf(d Density, kids ...*Node) *Node {
	return Surface(Stack(kids...)).Density(d)
}

// Rows follows surf.css: a token changed in the sheet changes the
// estimate, with no number copied into the code.
func TestRowsFollowsTheSheet(t *testing.T) {
	root := func() *Node { return surf(Compact, List(Item(Text("one")), Item(Text("two")))) }
	was := Height(root(), 40, m1)
	old := CSS
	defer func() { CSS = old; sheetOnce = sync.Once{} }()
	// Compact's iy is 4px, and the rem block that follows says it again.
	CSS = strings.ReplaceAll(strings.ReplaceAll(CSS, "--u-iy: 4px;", "--u-iy: 10px;"), "--u-iy: 0.23529412rem;", "--u-iy: 10px;")
	if CSS == old {
		t.Fatal("surf.css has no --u-iy: 4px to change")
	}
	sheetOnce = sync.Once{}
	if got := Height(root(), 40, m1); got < was+23.99 || got > was+24.01 {
		t.Errorf("an item's padding 4px to 10px: height %v, was %v; want %v (two items, 12px more each)", got, was, was+24)
	}
}

// The units of each density, as surf.css gives them.
func TestRowsUnits(t *testing.T) {
	for _, c := range []struct {
		name string
		d    Density
		root *Node
		want float64
	}{
		{"a dense text is a row", Dense, Text("hi"), 21.25},
		{"a compact text is a row", Compact, Text("hi"), 21.25},
		{"a compact item has 4px above and below", Compact, List(Item(Text("hi"))), 21.25 + 8},
		{"two compact items have the gap between", Compact, List(Item(Text("a")), Item(Text("b"))), 2*(21.25+8) + 2},
		{"a compact button is 28px", Compact, Button("Send"), 28},
		{"a dense button is a row", Dense, Button("Send"), 21.25},
		{"a compact rule has 4px round it", Compact, Rule(), 9},
		{"a roomy text is 1.6 of 17px", Roomy, Text("hi"), 27.2},
		{"a field of 3 rows is 3 rows and 8px", Compact, Field().Rows(3, 8), 3*21.25 + 8},
		{"a field of 3..8 holding 5 lines is 5", Compact, Field().Rows(3, 8).Value("1\n2\n3\n4\n5"), 5*21.25 + 8},
		{"a field holding 12 lines stops at 8", Compact, Field().Rows(3, 8).Value(strings.Repeat("x\n", 11) + "x"), 8*21.25 + 8},
		{"a stack's gap is half a row in Compact", Compact, Stack(Text("a"), Text("b")).Gap(2), 3 * 21.25},
		{"a height is rows", Compact, Stack().Height(4), 4 * 21.25},
	} {
		root := c.root
		if root.tag != "div" || !hasClassOf(root, "u-surface") {
			root = Surface(c.root)
		}
		root = root.Density(c.d)
		if got := Height(root, 40, m1); got < c.want-0.01 || got > c.want+0.01 {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}

func TestRowsRounds(t *testing.T) {
	r := surf(Dense, Text("a"), Text("b"), Text("c"))
	if got := Rows(r, 20, m1); got != 3 {
		t.Errorf("three dense lines: %d rows, want 3", got)
	}
	if got := Rows(surf(Compact), 20, m1); got != 1 {
		t.Errorf("an empty surface: %d rows, want the least, 1", got)
	}
	long := surf(Dense, Text(strings.Repeat("word ", 20)).Wrap())
	if got := Rows(long, 20, m1); got != 5 {
		t.Errorf("100 characters wrapped in 20 cells: %d rows, want 5", got)
	}
}

// A field's rows from the host's measure: the rows that make the
// document fit, and its text's own where there is none.
func TestFieldRows(t *testing.T) {
	root := Surface(Stack(Text("Reply"), Field().Key("t").Rows(3, 8).Value("one\ntwo"), Button("Send")).Gap(1)).Density(Compact)
	Resolve(root, "r")
	base := Rows(root, 40, m1)
	id := areaID(root)
	if got := FieldRows(root, 40, m1, 0)[id]; got != 3 {
		t.Errorf("two lines in a field of 3..8: %d rows with no measure, want 3", got)
	}
	for _, c := range []struct{ fit, want int }{{base, 3}, {base + 4, 7}, {base + 9, 8}, {base - 2, 3}} {
		if got := FieldRows(root, 40, m1, c.fit)[id]; got != c.want {
			t.Errorf("fit %d (base %d): %d rows, want %d", c.fit, base, got, c.want)
		}
	}
	text := strings.Repeat("a long line of prose ", 30)
	grown := Surface(Stack(Field().Key("t").Rows(3, 8).Value(text))).Density(Compact)
	Resolve(grown, "r")
	if got := FieldRows(grown, 30, m1, 0)[areaID(grown)]; got < 4 {
		t.Errorf("630 characters in 30 cells: %d rows, want more than 3", got)
	}
}

func areaID(n *Node) string {
	if n.tag == "textarea" {
		return n.id
	}
	for _, k := range n.kids {
		if id := areaID(k); id != "" {
			return id
		}
	}
	return ""
}
