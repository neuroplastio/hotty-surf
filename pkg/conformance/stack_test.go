//go:build surfstack && tinygo

package conformance

import (
	"fmt"
	"io/fs"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	cellite "github.com/neuroplastio/cel-lite"
	"github.com/neuroplastio/hotty-go"

	surf "github.com/neuroplastio/hotty-surf"
	"github.com/neuroplastio/hotty-surf/internal/stackprobe"
	"github.com/neuroplastio/hotty-surf/pkg/cells"
	"github.com/neuroplastio/hotty-surf/surflive"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// depth is how deep the measured trees go, in a render's levels, as deep
// as a render goes (surf.MaxDepth): each widget, each text in a
// container and each use of a component or a partial is one, the root's
// widget the first; a :if, a :each, a slot add none. A screen of plx's
// went 10 deep at most when R-23 measured them.
const depth = surf.MaxDepth

// stackBudget is the most stack a render, a Load, or a walk of a
// render's tree may need at depth: a render and a Load with cel-lite's
// most at their leaves, a walk on its own. It is half a goroutine's
// 64 KB under TinyGo, which leaves the rest to the program, hotty-go, and
// a pause's unwind data. TestStackUse fails where a path needs more.
const stackBudget = 32 << 10

// painted is how much of the stack below a measure is painted: well
// within make stack's 1 MB stacks.
const painted = 256 << 10

// deepCols and deepRows are the screen the deep trees are drawn on in
// cells: room for every branch, the deepest's indent and rows too, which
// the goldens' 64×32 cut off.
const deepCols, deepRows = 256, 256

// use is how far below its caller a call took the stack: to the deepest
// probe (our recursion's frames), and to the deepest word written (every
// frame, the leaves' too: cel-lite's, the allocator's, the collector's,
// and the compiled mode's generated functions, which have no probe).
type use struct{ probe, paint int }

//go:noinline
func measure(f func()) use {
	base := stackprobe.Here()
	stackprobe.Paint(painted)
	stackprobe.Reset()
	f()
	return use{stackprobe.Used(base), stackprobe.Deepest(base, painted)}
}

// steps are the levels each way of holding the next branch adds
// (lab-tree.surf): a use and its stack through a :each or a :if; a use, its
// box, the fill's use and its stack through a slot; a use, its eight
// stacks, the use in them and its stack through the widgets.
var steps = map[string]int{"each": 2, "if": 2, "slot": 4, "widgets": 11}

// grow is a tree of len(vias)+1 branches, branch i holding the next by
// vias[i], the deepest named name; and how deep a render of a Board of it
// goes: the surface, the tree's use, its stack and its text are 1 to 4.
func grow(name string, vias ...string) (*Branch, int) {
	b := &Branch{ID: "leaf", Name: name}
	levels := 4
	for i := len(vias) - 1; i >= 0; i-- {
		id := "b" + strconv.Itoa(i)
		nb := &Branch{ID: id, Name: id}
		if vias[i] == "each" {
			nb.Kids = []*Branch{b}
		} else {
			nb.Via, nb.Next = vias[i], b
		}
		b = nb
		levels += steps[vias[i]]
	}
	return b, levels
}

// deepest is the longest run of via that stays within depth.
func deepest(via string) []string {
	vias := make([]string, (depth-4)/steps[via])
	for i := range vias {
		vias[i] = via
	}
	return vias
}

func quiet(c surflive.Component, errs *[]error) surflive.Component {
	on := func(err error) { *errs = append(*errs, err) }
	switch v := c.(type) {
	case *surf.View:
		v.OnError = on
	case *surf.CompiledView:
		v.OnError = on
	}
	return c
}

// uiDepth is how many elements deep n's tree is.
func uiDepth(n *surfui.Node) int {
	if surfui.Tag(n) == "" {
		return 0
	}
	d := 0
	for _, k := range surfui.Kids(n) {
		d = max(d, uiDepth(k))
	}
	return d + 1
}

// row is one path's measure: at a shallow tree and a deep one, and the
// bytes per level between them.
type row struct {
	path, via     string
	leaf          int // what cel-lite may add at its leaves (celLiteUse)
	low, high     use
	lowLv, highLv int // the levels, or for a walk the tree's elements
}

func (r *row) per() (probe, paint int) {
	n := r.highLv - r.lowLv
	return (r.high.probe - r.low.probe) / n, (r.high.paint - r.low.paint) / n
}

// at is what r's path needs at depth levels, by the paint: what it took
// at the deepest tree, and where that is shallower (a walk's tree has
// fewer elements than levels; eight widgets' steps don't divide depth),
// that plus a level's bytes for each level more.
func (r *row) at(levels int) int {
	if r.highLv >= levels {
		return r.high.paint
	}
	_, per := r.per()
	return r.high.paint + max(per, 0)*(levels-r.highLv)
}

// fits is the deepest r's path goes within stackBudget, by the paint,
// with cel-lite's most at its leaves.
func (r *row) fits() string {
	_, per := r.per()
	room := stackBudget - r.low.paint - r.leaf
	if per <= 0 {
		return "any"
	}
	return strconv.Itoa(r.lowLv + room/per)
}

func dash(n int) string {
	if n == 0 {
		return "-"
	}
	return strconv.Itoa(n)
}

// TestStackUse measures the stack a render needs in each mode, and each
// walk of its tree, under TinyGo, whose goroutine stacks are fixed (`make
// stack`): for a tree that uses itself as deep as depth, through each way
// lab-tree.surf holds the next branch; and a Load of a file nested as
// deep, plainly, through :if and through :each, and of templates that
// use each other as deep. It logs what each path needs a level and at
// depth, and how deep it could go within stackBudget; it fails where a
// path at depth, with cel-lite's most at its leaves, needs more than
// stackBudget, or where a render or a walk goes wrong.
func TestStackUse(t *testing.T) {
	load, eval := celLiteUse(t)
	var rows []*row
	for _, via := range []string{"each", "if", "slot", "widgets"} {
		trees := map[bool][2]*surfui.Node{}
		for _, high := range []bool{false, true} {
			vias := []string{via}
			if high {
				vias = deepest(via)
			}
			for _, m := range modes {
				var errs []error
				b := &Board{}
				v := quiet(m.view(b), &errs)
				var tree *surfui.Node
				var lv int
				b.Trees = make([]*Branch, 1)
				b.Trees[0], lv = grow("deep", vias...)
				u := measure(func() { tree = v.Render(nil) })
				for _, err := range errs { // not the empty Board's own, as in TestCases
					if s := err.Error(); strings.HasPrefix(s, "lab-tree.surf") || strings.HasPrefix(s, "lab-box.surf") || strings.HasPrefix(s, "lab-nest.surf") {
						t.Fatalf("%s, %s, %d levels: %v", via, m.name, lv, err)
					}
				}
				if !strings.Contains(surfui.TextOf(tree), "deep") {
					t.Fatalf("%s, %s, %d levels: the deepest branch is not drawn", via, m.name, lv)
				}
				r := rowOf(&rows, "render, "+m.name, via)
				r.leaf = eval
				if high {
					r.high, r.highLv = u, lv
				} else {
					r.low, r.lowLv = u, lv
				}
				if m.name == "interpreted" {
					b.Trees[0], _ = grow("deeper", vias...)
					trees[high] = [2]*surfui.Node{tree, v.Render(nil)}
				}
			}
		}
		// The walks, of the interpreter's trees: the compiled mode's are
		// the same (TestCases).
		for _, high := range []bool{false, true} {
			old, next := trees[high][0], trees[high][1]
			surfui.Resolve(next, "r")
			var doc string
			var ops []hotty.Op
			walks := []struct {
				name string
				f    func()
			}{
				{"Resolve", func() { surfui.Resolve(old, "r") }},
				{"HTML", func() { doc = surfui.Document(old) }},
				{"Diff", func() {
					if ops = surfui.Diff(old, next); len(ops) != 1 {
						t.Errorf("%s: the diff is %d ops, not one text", via, len(ops))
					}
				}},
				{"Clone", func() { surfui.Clone(old) }},
				{"TextOf", func() { surfui.TextOf(old) }},
				// hotty-go's alone, given the HTML and the ops.
				{"hotty.Doc", func() { hotty.Doc("board", doc) }},
				{"hotty.Delta", func() { hotty.Delta("board", ops) }},
				// The tree in cells, where there is no host.
				{"cells.Draw", func() {
					r := cells.New(still{old})
					r.Show()
					if !strings.Contains(r.Draw(deepCols, deepRows).Text(), "deep") {
						t.Errorf("%s: the cells don't draw the deepest branch", via)
					}
				}},
			}
			for _, w := range walks {
				u := measure(w.f)
				r := rowOf(&rows, w.name, via)
				if high {
					r.high, r.highLv = u, uiDepth(old)
				} else {
					r.low, r.lowLv = u, uiDepth(old)
				}
			}
		}
	}

	// A file nested as deep, parsed and compiled: its stacks plain, or
	// each with a :if or a :each; and templates that use each other as
	// deep, each a stack round the next one's use.
	// The suite's templates, the components' and their partials', with
	// board.surf given.
	files := func(board string) fstest.MapFS {
		fsys := fstest.MapFS{}
		names, _ := fs.Glob(surfFiles, "*.surf")
		for _, n := range names {
			src, _ := fs.ReadFile(surfFiles, n)
			fsys[n] = &fstest.MapFile{Data: src}
		}
		fsys["board.surf"] = &fstest.MapFile{Data: []byte(board)}
		return fsys
	}
	nested := func(open string) func(n int) fstest.MapFS {
		return func(n int) fstest.MapFS {
			return files("<surface>" + rep(open, n-2) + "<text>x</text>" + rep("</stack>", n-2) + "</surface>")
		}
	}
	uses := func(n int) fstest.MapFS { // the surface, then a use and its stack for each template
		k := (n - 2) / 2
		fsys := files("<surface><chain-1/></surface>")
		for i := 1; i <= k; i++ {
			next := "<text>x</text>"
			if i < k {
				next = "<chain-" + strconv.Itoa(i+1) + "/>"
			}
			fsys["chain-"+strconv.Itoa(i)+".surf"] = &fstest.MapFile{Data: []byte("<stack>" + next + "</stack>")}
		}
		return fsys
	}
	for _, l := range []struct {
		via   string
		files func(n int) fstest.MapFS
	}{
		{"a file", nested("<stack>")},
		{"a file, :if", nested(`<stack :if="count >= 0">`)},
		{"a file, :each", nested(`<stack :each="x in log">`)},
		{"uses", uses},
	} {
		for _, high := range []bool{false, true} {
			n := 4
			if high {
				n = depth
			}
			u := measure(func() {
				if _, err := surf.Load(l.files(n), surfShapes); err != nil {
					t.Fatal(err)
				}
			})
			r := rowOf(&rows, "Load", l.via)
			r.leaf = load
			if high {
				r.high, r.highLv = u, n
			} else {
				r.low, r.lowLv = u, n
			}
		}
	}

	t.Logf("bytes of stack under TinyGo's wasm, by the probes (our frames: - where none runs) and by the paint (every frame);")
	t.Logf("at %d levels (surf.MaxDepth), or a walk at as many elements, with cel-lite's most at a render's leaves (%d) and a Load's (%d);", depth, eval, load)
	t.Logf("and the deepest each could go within %d bytes, with the same:", stackBudget)
	t.Logf("%-20s %-14s %8s %8s %16s %8s %8s %8s %8s", "path", "through", "a level", "", "deepest tree", "", "at depth", "+ leaves", "in budget")
	for _, r := range rows {
		probe, paint := r.per()
		need := r.at(depth) + r.leaf
		over := ""
		if need > stackBudget {
			over = "  OVER"
			t.Errorf("%s, through %s: %d bytes at %d levels, with %d at its leaves: past the budget, %d", r.path, r.via, r.at(depth), depth, r.leaf, stackBudget)
		}
		t.Logf("%-20s %-14s %8s %8d %8s %3s %8d %8d %8d %8s%s", r.path, r.via, dash(probe), paint,
			dash(r.high.probe), fmt.Sprintf("(%d)", r.highLv), r.high.paint, r.at(depth), need, r.fits(), over)
	}
}

// still is a tree as a program that does nothing, for the renderer.
type still struct{ n *surfui.Node }

func (s still) Show() cells.Frame                       { return cells.Frame{} }
func (s still) Update() cells.Frame                     { return cells.Frame{} }
func (s still) Tree() *surfui.Node                      { return s.n }
func (still) Event(hotty.Event) (took, redraw bool)     { return false, false }
func (still) ListEvent(hotty.Event) (took, redraw bool) { return false, false }
func (still) Key(string) (took, redraw bool)            { return false, false }
func (still) Keys() ([]cells.KeyUse, cells.Chars)       { return nil, cells.CharsBound }

func rowOf(rows *[]*row, path, via string) *row {
	for _, r := range *rows {
		if r.path == path && r.via == via {
			return r
		}
	}
	r := &row{path: path, via: via}
	*rows = append(*rows, r)
	return r
}

// celFamilies are the shapes that make an expression deep, as cel-lite's
// limit_test.go builds them, each n levels deep.
var celFamilies = []struct {
	name  string
	build func(n int) string
}{
	{"parens", func(n int) string { return rep("(", n) + "x" + rep(")", n) }},
	{"lists", func(n int) string { return rep("[", n) + "x" + rep("]", n) }},
	{"maps", func(n int) string { return rep("{'a': ", n) + "x" + rep("}", n) }},
	{"calls", func(n int) string { return rep("dyn(", n) + "x" + rep(")", n) }},
	{"indexes", func(n int) string { return rep("m[", n) + "'a'" + rep("]", n) }},
	{"ternaries", func(n int) string { return rep("x ? x : ", n) + "x" }},
	{"negations", func(n int) string { return rep("-(", n) + "1" + rep(")", n) }},
	{"nots", func(n int) string { return rep("!(", n) + "x" + rep(")", n) }},
	{"ands", func(n int) string { return rep("x && (", n) + "x" + rep(")", n) }},
	{"ins", func(n int) string { return rep("x in [", n) + "x" + rep("]", n) }},
	{"selects", func(n int) string { return "m" + rep(".a", n) }},
	{"has", func(n int) string { return "has(m" + rep(".a", n) + ")" }},
	{"index chain", func(n int) string { return "m" + rep("['a']", n) }},
	{"sums", func(n int) string { return "1" + rep(" + 1", n) }},
	{"equalities", func(n int) string { return "x" + rep(" == x", n) }},
	{"climbs", func(n int) string { return rep("x || x && m == m + m * dyn(", n) + "x" + rep(")", n) }},
}

func rep(s string, n int) string { return strings.Repeat(s, n) }

// celLiteUse measures, by the paint, the stack cel-lite needs for each of
// its families at its limits (the deepest Parse takes): the most of its
// parse, check and plan, which a Load's compile runs at its leaves, and
// of its eval, which a render runs at its.
func celLiteUse(t *testing.T) (load, eval int) {
	env, err := cellite.NewEnv(cellite.Decls{Vars: []cellite.Var{{Name: "x", Type: cellite.Bool}, {Name: "m", Type: cellite.Dyn}}})
	if err != nil {
		t.Fatal(err)
	}
	act := env.Activation(map[string]any{"x": true, "m": map[string]any{"a": "a"}})
	t.Logf("cel-lite at its limits, by the paint (every frame):")
	for _, f := range celFamilies {
		n := 0
		for ; n <= 2*cellite.MaxDepth; n++ {
			if _, err := cellite.Parse(f.build(n + 1)); err != nil {
				break
			}
		}
		src := f.build(n)
		var x, y *cellite.Expr
		var p *cellite.Program
		var errs [3]error
		parse := measure(func() { x, errs[0] = cellite.Parse(src) })
		check := measure(func() { y, errs[1] = env.Check(x) })
		plan := measure(func() { p, errs[2] = env.Program(y) })
		for _, err := range errs {
			if err != nil {
				t.Fatalf("%s at %d levels: %v", f.name, n, err)
			}
		}
		run := measure(func() { p.Eval(act) })
		t.Logf("%-12s %3d levels: parse %6d  check %6d  plan %6d  eval %6d", f.name, n, parse.paint, check.paint, plan.paint, run.paint)
		load = max(load, parse.paint, check.paint, plan.paint)
		eval = max(eval, run.paint)
	}
	t.Logf("most: %d bytes at a Load's leaves, %d at a render's", load, eval)
	return load, eval
}
