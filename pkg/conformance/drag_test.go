package conformance

import (
	"slices"
	"testing"

	"github.com/neuroplastio/hotty-go"
	"github.com/neuroplastio/hotty-surf/pkg/cells"
	"github.com/neuroplastio/hotty-surf/surflive"
	"github.com/neuroplastio/hotty-surf/surfrun"
)

// A drag in the cells is a press, the pointer's moves and the release, which
// the renderer turns into the host's dragstart, drag and dragend: the
// handlers it runs are the host's, in both modes. Here a row is dragged
// over a second row's top half, then its bottom half, and let go there;
// then a press and release on a row, which is its click.
func TestDragCellsAsHost(t *testing.T) {
	deck := func() *Deck {
		return &Deck{Rows: []*Card{{ID: "a", Name: "Alpha"}, {ID: "b", Name: "Beta"}, {ID: "c", Name: "Gamma"}}}
	}
	want := []string{"dragstart a", "move a after b", "dragend a", "dragstart c", "dragend c", "open c"}
	for _, m := range modes {
		// The host: its events, as hotty-go reads them.
		d := deck()
		s := surfrun.NewSurface(surflive.New(m.deck(d)), nil)
		s.Show()
		for _, e := range []hotty.Event{
			{Kind: hotty.EventDragStart, Target: "r.krows.ka", Drag: &hotty.Drag{}},
			{Kind: hotty.EventDrag, Target: "r.krows.kb", Drag: &hotty.Drag{TY: 0, HasTY: true}},
			{Kind: hotty.EventDrag, Target: "r.krows.kb", Drag: &hotty.Drag{TY: 1, HasTY: true}},
			{Kind: hotty.EventDragEnd, Target: "r.krows.kb", Drag: &hotty.Drag{TY: 1, HasTY: true}},
			{Kind: hotty.EventDragStart, Target: "r.krows.kc", Drag: &hotty.Drag{}},
			{Kind: hotty.EventDragEnd, Target: "r.krows.kc", Drag: &hotty.Drag{TY: 0, HasTY: true}},
			{Kind: hotty.EventClick, Target: "r.krows.kc", Count: 1},
		} {
			s.Event(e)
			s.Update()
		}
		if !slices.Equal(d.Log, want) {
			t.Errorf("%s, the host: %q, want %q", m.name, d.Log, want)
		}

		// The cells: a press, moves and a release at cells.
		d = deck()
		s = surfrun.NewSurface(surflive.New(m.deck(d)), nil)
		r := cells.New(s.Cells())
		r.Show()
		r.Draw(CellsCols, CellsRows)
		at := func(id string, ty int) (int, int) {
			x, y, ok := dragCell(r, id, ty)
			if !ok {
				t.Fatalf("%s, %s at %d is not on the screen:\n%s", m.name, id, ty, r.Screen().Text())
			}
			return x, y
		}
		x, y := at("r.krows.ka", -1)
		r.Press(x, y, 0)
		r.Draw(CellsCols, CellsRows) // as a host draws after each input
		x, y = at("r.krows.kb", 0)
		r.Move(x, y, 0)
		r.Draw(CellsCols, CellsRows)
		x, y = at("r.krows.kb", 1)
		r.Move(x, y, 0)
		r.Draw(CellsCols, CellsRows)
		r.Release(x, y, 0)
		r.Draw(CellsCols, CellsRows)
		x, y = at("r.krows.kc", -1)
		r.Press(x, y, 0)
		r.Draw(CellsCols, CellsRows)
		r.Release(x, y, 0)
		if !slices.Equal(d.Log, want) {
			t.Errorf("%s, the cells: %q, want %q", m.name, d.Log, want)
		}
	}
}
