package conformance

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-go"
	"github.com/neuroplastio/hotty-surf/pkg/cells"
	"github.com/neuroplastio/hotty-surf/surflive"
	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surfui"
	"golang.org/x/tools/txtar"
)

// The fixtures' tooltips (board.surf's heading, a title=, and its
// select, a bound :title; note-row.surf's item, a <title> of a row with
// a dot, bold text and a muted line, reading its note, and each tag's
// chip in a :each, a :title reading its row) are found alike in both
// modes: by an element's id, the innermost one's around it, and by a
// cell; none where no element around has one, or the id is gone. A
// change shows in the tooltip asked again after it. The document has no
// HTML title, and the surface asks for hover.
func TestTips(t *testing.T) {
	src, err := fs.ReadFile(Cases, "cases/board.txt")
	if err != nil {
		t.Fatal(err)
	}
	state, _ := File(txtar.Parse(src), "state.json")
	want := strings.Join([]string{
		"show: hover on",
		"r.khead.0: r.khead.0 \"The board's notes\"",
		"r.khead.kfilter: r.khead.kfilter \"Shows\"",
		"r.klist.kn1: r.klist.kn1 \"● n1 / Pane p-3 exited\"",
		"r.klist.kn1.0.2: r.klist.kn1 \"● n1 / Pane p-3 exited\"",
		"r.klist.kn1.0.kp-3: r.klist.kn1.0.kp-3 \"Tag p-3 of n1\"",
		"r.klist: none",
		"r.kstatus: none",
		"r.gone: none",
		"cell of r.klist.kn3.0.2: r.klist.kn3 \"● n3 / Theme switched\"",
		"cell of r.kstatus: none",
		"after a pick: r.khead.kfilter \"Shows unread\"",
	}, "\n")
	for _, m := range modes {
		t.Run(m.name, func(t *testing.T) {
			b := &Board{}
			if err := json.Unmarshal(state, b); err != nil {
				t.Fatal(err)
			}
			var heard []string
			s := surfrun.NewSurface(surflive.New(hear(m.view(b), &heard)), nil)
			r := cells.New(s.Cells())
			got := []string{}
			f := s.Show()
			got = append(got, "show: hover "+map[hotty.Flag]string{hotty.On: "on", hotty.Off: "off"}[f.Hover])
			r.Show()
			r.Draw(CellsCols, CellsRows)
			doc := surfui.HTML(s.Tree())
			if strings.Contains(doc, " title=") {
				t.Errorf("the document has an HTML title: %s", doc)
			}
			for _, id := range []string{"r.khead.0", "r.khead.kfilter", "r.klist.kn1", "r.klist.kn1.0.kp-3"} {
				if !strings.Contains(doc, ` id="`+id+`"`) {
					t.Errorf("no element %s in the document", id)
				}
			}
			tip := func(what string, at surfrun.TipAt) {
				tp, ok := s.Tip(at)
				got = append(got, what+": "+tipText(t, tp, ok))
			}
			for _, id := range []string{"r.khead.0", "r.khead.kfilter", "r.klist.kn1", "r.klist.kn1.0.2", "r.klist.kn1.0.kp-3", "r.klist", "r.kstatus", "r.gone"} {
				tip(id, surfrun.TipAt{ID: id})
			}
			for _, id := range []string{"r.klist.kn3.0.2", "r.kstatus"} {
				x, y, ok := r.Where(id)
				if !ok {
					t.Fatalf("%s is not on the screen", id)
				}
				tip("cell of "+id, surfrun.TipAt{ID: r.At(x, y)})
			}
			s.Pick("r.khead", "0") // its menu: unread only
			s.Update()
			tip("after a pick", surfrun.TipAt{ID: "r.khead.kfilter"})
			if g := strings.Join(got, "\n"); g != want {
				t.Errorf("got:\n%s\nwant:\n%s", g, want)
			}
			for _, err := range heard { // the lab's are its own (lab.txt)
				if !strings.HasPrefix(err, "lab-") {
					t.Errorf("render error: %s", err)
				}
			}
		})
	}
}

// tipText is a Tip as TestTips writes it: "none", or its id and its
// text, its lines joined by " / ". Its cells hold the same text, in a
// card with a line round it, a row more than it takes on a host; its
// document has surf's card and no HTML title.
func tipText(t *testing.T, tip surfrun.Tip, ok bool) string {
	t.Helper()
	if !ok {
		return "none"
	}
	var lines []string
	for _, l := range strings.Split(tip.Cells.Text(), "\n") {
		l = strings.Trim(l, " │╭╮╰╯─")
		if l != "" {
			lines = append(lines, l)
		}
	}
	if tip.Cells.Rows != tip.Rows+1 || tip.Cells.Cols != tip.Cols || tip.Cols > surfui.TipCols {
		t.Errorf("%s: %d×%d on a host, %d×%d in the cells", tip.ID, tip.Cols, tip.Rows, tip.Cells.Cols, tip.Cells.Rows)
	}
	if doc := tip.HTML(); !strings.Contains(doc, `class="u-tip u-small"`) || strings.Contains(doc, " title=") {
		t.Errorf("%s: document %s", tip.ID, doc)
	}
	return fmt.Sprintf("%s %q", tip.ID, strings.Join(lines, " / "))
}
