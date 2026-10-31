//go:build surfstack && tinygo

package surflua

import (
	"fmt"
	"io/fs"
	"strings"
	"testing"

	cellite "github.com/neuroplastio/cel-lite"
	surf "github.com/neuroplastio/hotty-surf"
	"github.com/neuroplastio/hotty-surf/internal/stackprobe"
	"github.com/neuroplastio/hotty-surf/surfui"
	lua "github.com/yuin/gopher-lua"
)

const painted = 256 << 10

// stackBudget is conformance's: the most a render may need at
// surf.MaxDepth levels, with what its expressions need at its leaves,
// under TinyGo's 64 KB stacks.
const stackBudget = 32 << 10

type use struct{ probe, paint int }

//go:noinline
func measure(f func()) use {
	base := stackprobe.Here()
	stackprobe.Paint(painted)
	stackprobe.Reset()
	f()
	return use{stackprobe.Used(base), stackprobe.Deepest(base, painted)}
}

// nested is a list holding a list, n deep, as a template's literal
// [[[1]]] is: not a Lua table, so ToLua builds one of each.
type nested int

func (l nested) Len() int { return 1 }
func (l nested) At(int) cellite.Value {
	if l <= 1 {
		return cellite.IntValue(1)
	}
	return cellite.ListValue(l - 1)
}

// TestStackUse measures the stack surflua's own deep paths need under
// TinyGo (`make stack`): == on two Lua tables as deep as MaxNesting lets
// a template read them, which cel-lite compares by what they hold, reading
// each level through value; and ToLua of a list a template builds, as
// deep as cel-lite lets a literal nest (16). It reports our frames (the
// probes, in value and ToLua) and every frame (the paint): cel-lite's
// equality, ours, and for view:html gopher-lua's Go frames. Lua's own
// call stack is on the heap. Past MaxNesting, the == is refused
// (doc.go). The == is drawn at a render's first levels, and at its last,
// surf.MaxDepth: there, it fails where the render needs more than
// stackBudget.
func TestStackUse(t *testing.T) {
	deep := "<surface>" + strings.Repeat("<stack>", surf.MaxDepth-2) + `<text key="a">{{ a == b }}</text>` +
		strings.Repeat("</stack>", surf.MaxDepth-2) + "</surface>"
	dir := write(t, map[string]string{"x.surf": `<surface><text key="a">{{ a == b }}</text></surface>`, "deep.surf": deep})
	sub, err := fs.Sub(files[t], "t")
	if err != nil {
		t.Fatal(err)
	}
	L := state(t)
	at := map[int][2]use{}
	for _, n := range []int{1, MaxNesting, MaxNesting + 8} {
		eval(t, L, fmt.Sprintf(`
local function nest(n)
  local t = { name = "x" }
  for i = 2, n do t = { name = "x", next = t } end
  return t
end
s = { a = nest(%d), b = nest(%d) }
view = require("surf").load(%q):view("x", s)`, n, n, dir))
		inst, err := New(L, L.GetGlobal("s").(*lua.LTable), "x")
		if err != nil {
			t.Fatal(err)
		}
		set, err := surf.Load(sub, []*surf.Shape{inst.Shape()})
		if err != nil {
			t.Fatal(err)
		}
		var text string
		var errs []error
		direct := measure(func() {
			var root *surfui.Node
			root, errs = set.Render(inst)
			text = surfui.TextOf(root)
		})
		switch {
		case n > MaxNesting && len(errs) == 1 && strings.Contains(errs[0].Error(), "comparing structs"):
			text = "refused" // past MaxNesting, two such tables are structs, which cel-lite doesn't compare
		case len(errs) > 0:
			t.Fatal(errs)
		}
		html := measure(func() { eval(t, L, `return view:html()`) })
		at[n] = [2]use{direct, html}
		t.Logf("== of two tables %2d deep (%s): Render %5d / %5d, view:html %5d / %5d bytes (probes / paint)",
			n, text, direct.probe, direct.paint, html.probe, html.paint)
		if n != MaxNesting {
			continue
		}
		dinst, err := New(L, L.GetGlobal("s").(*lua.LTable), "deep")
		if err != nil {
			t.Fatal(err)
		}
		dset, err := surf.Load(sub, []*surf.Shape{dinst.Shape()})
		if err != nil {
			t.Fatal(err)
		}
		at := measure(func() {
			root, errs := dset.Render(dinst)
			if len(errs) > 0 {
				t.Fatal(errs)
			}
			text = surfui.TextOf(root)
		})
		t.Logf("== of two tables %2d deep (%s), its text at level %d: Render %5d / %5d bytes (of %d)",
			n, text, surf.MaxDepth, at.probe, at.paint, stackBudget)
		if at.paint > stackBudget {
			t.Errorf("== of two tables %d deep, at level %d: %d bytes, past the budget, %d", n, surf.MaxDepth, at.paint, stackBudget)
		}
	}
	lv := MaxNesting - 1
	t.Logf("== a table deeper: our frames %d bytes, every frame %d (Render), %d (view:html)",
		(at[MaxNesting][0].probe-at[1][0].probe)/lv, (at[MaxNesting][0].paint-at[1][0].paint)/lv, (at[MaxNesting][1].paint-at[1][1].paint)/lv)

	tl := map[int]use{}
	for _, n := range []int{1, 16} {
		tl[n] = measure(func() { ToLua(L, cellite.ListValue(nested(n))) })
	}
	t.Logf("ToLua of a list 16 deep: %d / %d bytes, %d / %d a level", tl[16].probe, tl[16].paint,
		(tl[16].probe-tl[1].probe)/15, (tl[16].paint-tl[1].paint)/15)
}
