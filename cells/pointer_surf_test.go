package cells_test

import (
	"context"
	"fmt"
	"io"
	"testing"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf"
)

// The surface's own handlers hear the pointer in cells as on a host:
// a reorder by Drag and Drop, a Secondary, a double Click.
func TestPointerHandlersInCells(t *testing.T) {
	var heard []string
	v := view(func(ctx context.Context, w io.Writer, on func(string, ...surf.Handler) string) {
		for _, id := range []string{"a", "b", "c"} {
			fmt.Fprintf(w, `<div%s data-steps="0 1">row %s</div>`, on(id,
				surf.Drag(func(d surf.DragEvent) {
					if d.End {
						heard = append(heard, fmt.Sprintf("end %s over %s ty %d", d.ID, d.Over, d.TY))
					}
				}),
				surf.Drop(func(d surf.DragEvent) { heard = append(heard, "drop "+d.ID+" on "+d.Over) }),
				surf.Secondary(func(at hotty.Cell, a hotty.Area) {
					heard = append(heard, fmt.Sprintf("menu %s at %d,%d", id, at.Col, at.Row))
				}),
				surf.Click(func() { heard = append(heard, "click "+id) }),
			), id)
		}
	})
	r := start(t, 30, 6, v)
	ac, ar := r.at("row a")
	_, cr := r.at("row c")
	r.c.Press(ac, ar, 0, 0)
	r.settle()
	r.c.Move(ac, cr, 0)
	r.settle()
	r.c.Release(ac, cr, 0)
	r.settle()
	r.c.Press(ac+1, cr, 0, 2)
	r.settle()
	want := []string{"end a over c ty 1", "drop a on c", fmt.Sprintf("menu c at %d,%d", ac+1, cr)}
	if fmt.Sprint(heard) != fmt.Sprint(want) {
		t.Errorf("heard %q, want %q", heard, want)
	}
	if id, _ := r.c.At(ac, cr); id != "c" {
		t.Errorf("At on row c: %q", id)
	}
}
