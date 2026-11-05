package surfrun_test

import (
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-go"

	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// handles is a surface of a drag handle and an element that is none.
type handles struct{}

func (handles) Render() *surfui.Node {
	return surfui.Surface(
		surfui.Row(surfui.Text("a")).Key("a").Drag(),
		surfui.Row(surfui.Text("b")).Key("b"),
	)
}

func (handles) Do(act surfui.Action, ev surfrun.Event) {}

// rootClass is the root's class after ops, whether they say it: the
// last attr op on "r"'s class, an unattr being none.
func rootClass(ops []hotty.Op) (class string, said bool) {
	for _, op := range ops {
		if op.Target != "r" || op.Key != "class" {
			continue
		}
		switch op.Op {
		case "attr":
			class, said = *op.Value, true
		case "unattr":
			class, said = "", true
		}
	}
	return class, said
}

// A drag's mark on the root (surfui.DraggingClass) is in the frame after
// dragstart, and gone in the frame after dragend or Escape's cancel, which
// the host's dragend after it leaves be; a dragstart on what is no handle
// marks nothing.
func TestDraggingMarksRoot(t *testing.T) {
	step := func(s *surfrun.Surface, e hotty.Event) []hotty.Op {
		s.Event(e)
		return s.Update().Ops
	}
	start := func(id string) hotty.Event {
		return hotty.Event{Kind: hotty.EventDragStart, Target: id, Drag: &hotty.Drag{}}
	}
	end := hotty.Event{Kind: hotty.EventDragEnd, Target: "r.ka", Drag: &hotty.Drag{}}
	marked := func(ops []hotty.Op) bool {
		c, _ := rootClass(ops)
		return strings.Contains(" "+c+" ", " "+surfui.DraggingClass+" ")
	}
	show := func() *surfrun.Surface {
		s := surfrun.NewSurface(handles{}, nil)
		s.Show()
		s.Update()
		return s
	}

	s := show()
	if ops := step(s, start("r.ka")); !marked(ops) || !s.Dragging() {
		t.Errorf("after dragstart on a handle the root is not marked: %v", ops)
	}
	if ops := step(s, end); marked(ops) {
		t.Errorf("after dragend the root is still marked: %v", ops)
	} else if _, said := rootClass(ops); !said {
		t.Errorf("after dragend the ops do not touch the root's class: %v", ops)
	}

	s = show()
	step(s, start("r.ka"))
	s.Key("escape")
	ops := s.Update().Ops
	if marked(ops) {
		t.Errorf("after Escape the root is still marked: %v", ops)
	} else if _, said := rootClass(ops); !said {
		t.Errorf("after Escape the ops do not touch the root's class: %v", ops)
	}
	if ops := step(s, end); len(ops) != 0 {
		t.Errorf("the host's dragend after Escape changed %v", ops)
	}

	s = show()
	if ops := step(s, start("r.kb")); len(ops) != 0 || s.Dragging() {
		t.Errorf("a dragstart on no handle changed %v", ops)
	}
}
