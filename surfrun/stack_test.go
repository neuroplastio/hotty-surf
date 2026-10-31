//go:build surfstack && tinygo

package surfrun

import (
	"testing"

	"github.com/neuroplastio/hotty-surf/internal/stackprobe"
	"github.com/neuroplastio/hotty-surf/pkg/markup"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// depth is how many elements deep the measured tree goes: as deep as a
// render goes (surf.MaxDepth, which is markup's; run can't import surf).
const depth = markup.MaxDepth

// stackBudget is conformance's: the most a walk of a render's tree may
// need, on its own, under TinyGo's 64 KB stacks.
const stackBudget = 32 << 10

// TestStackUse measures the stack autofocused needs, the walk that finds
// the keys' home after each render, under TinyGo (`make stack`): for a
// tree of stacks as deep as a render goes, the deepest asking for the
// keyboard. It fails where that is past stackBudget. The renders and the
// surfui walks are conformance's TestStackUse.
func TestStackUse(t *testing.T) {
	const painted = 256 << 10
	type use struct{ probe, paint int }
	at := map[int]use{}
	for _, n := range []int{4, depth} {
		root := surfui.Text("x").Autofocus()
		for range n - 1 {
			root = surfui.Stack(root)
		}
		surfui.Resolve(root, "r")
		base := stackprobe.Here()
		stackprobe.Paint(painted)
		stackprobe.Reset()
		if autofocused(root) == "" {
			t.Fatal("no home")
		}
		at[n] = use{stackprobe.Used(base), stackprobe.Deepest(base, painted)}
	}
	t.Logf("autofocused: %d bytes a level by the probes, %d by the paint; %d at %d elements (of %d)",
		(at[depth].probe-at[4].probe)/(depth-4), (at[depth].paint-at[4].paint)/(depth-4), at[depth].paint, depth, stackBudget)
	if at[depth].paint > stackBudget {
		t.Errorf("autofocused: %d bytes at %d elements, past the budget, %d", at[depth].paint, depth, stackBudget)
	}
}
