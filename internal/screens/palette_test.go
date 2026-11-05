package screens

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-surf/internal/screens/demodata"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// The palette's parts, by id.
var (
	palQuery  = id("box", 1, "pick", "query", 1)
	palStatus = id("box", 1, "pick", "status")
	palFoot   = id("box", 1, "pick", "foot")
	palEmpty  = id("box", 1, "pick", "empty")
	palHead   = id("box", 1, "pick", "head")
)

func palRow(value string) string  { return id("box", 1, "pick", "rows", value) }
func palAct(action string) string { return id("box", 1, "pick", "acts", action) }

// openPalette is the palette in a pane, 64 cells by 44 rows, on the mock.
func openPalette(t *testing.T, m mode) (*app, *Palette) {
	t.Helper()
	src := demodata.Mock()
	p := NewPalette(src)
	return start(t, m.palette(p), src, false, 64, 44), p
}

// closed is whether p has closed, read on the run's goroutine.
func (a *app) closed(p *Palette) (closed bool) {
	a.Call(func() { closed = p.Closed })
	return closed
}

// rowsShown are the values of the rows the host shows, in order.
func rowsShown(a *app) []string {
	var out []string
	html := a.html()
	prefix := `id="` + id("box", 1, "pick", "rows") + ".k"
	for {
		i := strings.Index(html, prefix)
		if i < 0 {
			return out
		}
		html = html[i+len(prefix):]
		end := strings.IndexByte(html, '"')
		if !strings.Contains(html[:end], ".") {
			out = append(out, html[:end])
		}
		html = html[end:]
	}
}

// At open: the query has the keyboard, the first rows that fit are
// listed with row 0 selected, there is no status, and the footer is row
// 0's: its id, Actions, then Run and stay and Run (Copy command gives
// way).
func TestPaletteOpens(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		a, _ := openPalette(t, m)
		if f := a.Surface().Focused(); f != palQuery {
			t.Errorf("focus on %q, want the query", f)
		}
		// 16 rows fit in 44 lines at scale 80 (place).
		rows := rowsShown(a)
		if len(rows) != 16 {
			t.Errorf("%d rows shown, want 16 in 44 rows", len(rows))
		}
		if len(rows) > 0 && rows[0] != "pane_2enew" {
			t.Errorf("first row %s", rows[0])
		}
		if !a.Has(palRow("pane.new"), "is-sel") {
			t.Error("row 0 is not selected")
		}
		if strings.Contains(a.html(), palStatus) {
			t.Error("a status at open")
		}
		foot := a.text(palFoot)
		for _, w := range []string{"pane.new", "Actions", "\U000F0634k", "Run and stay", "\U000F0634\U000F0311", "Run", "\U000F0311"} {
			if !strings.Contains(foot, w) {
				t.Errorf("footer %q has no %q", foot, w)
			}
		}
		if strings.Contains(foot, "Copy command") {
			t.Errorf("footer %q: Copy command does not fit", foot)
		}
		if got := a.text(palRow("pane.split")); !strings.Contains(got, "Pane: Split") || !strings.Contains(got, "p-1") {
			t.Errorf("pane.split's row %q", got)
		}
		// The row's label is one line, its keys beside it.
		if got := a.text(palRow("pane.new") + ".0"); got != "Pane: New" {
			t.Errorf("pane.new's label %q", got)
		}
		if got := a.text(palRow("pane.new") + ".1"); got != "\U000F0634bpn" {
			t.Errorf("pane.new's keys %q, want ^b p n", got)
		}
		if acts := a.src.Acts(); len(acts) != 0 {
			t.Errorf("acts at open: %q", acts)
		}
	})
}

// A narrow palette keeps what it shows legible. At 40 cells, it is as
// at open, but Run and stay gives way. At 30, Actions goes too, so that
// the note keeps its start. At 20, too narrow for a label beside its
// caps, the rows show no keys, and the footer is Enter's alone. A row's
// label is one line at every width, cut once at its end.
func TestPaletteNarrow(t *testing.T) {
	for _, c := range []struct {
		cols         int
		keys, note   bool
		foot, absent []string
	}{
		{40, true, true, []string{"pane.new", "Actions", "Run", "\U000F0311"}, []string{"Run and stay", "Copy command"}},
		{26, true, true, []string{"pane.new", "Run", "\U000F0311"}, []string{"Actions", "Run and stay"}},
		{18, false, false, []string{"Run", "\U000F0311"}, []string{"pane.new", "Actions", "Run and stay"}},
	} {
		t.Run(strconv.Itoa(c.cols), func(t *testing.T) {
			inModes(t, func(t *testing.T, m mode) {
				src := demodata.Mock()
				a := start(t, m.palette(NewPalette(src)), src, false, c.cols, 44)
				row := palRow("pane.split")
				if got := a.text(row + ".0"); !a.Has(row+".0", "u-line") || !strings.HasPrefix(got, "Pane: Split") || !strings.HasSuffix(got, "p-1") {
					t.Errorf("%d cells: pane.split's label %q is not one line", c.cols, got)
				}
				if _, keyed := a.Surface().Element(row + ".1"); keyed != c.keys {
					t.Errorf("%d cells: keys shown %v, want %v", c.cols, keyed, c.keys)
				}
				foot := a.text(palFoot)
				for _, w := range c.foot {
					if !strings.Contains(foot, w) {
						t.Errorf("%d cells: footer %q has no %q", c.cols, foot, w)
					}
				}
				for _, w := range c.absent {
					if strings.Contains(foot, w) {
						t.Errorf("%d cells: footer %q has %q", c.cols, foot, w)
					}
				}
			})
		})
	}
}

// Rows with icons show their keys from iconCells more than rows without:
// at 26 cells (a palette 25 of its own) the agents' rows, which have
// icons, show their labels alone, the commands' their keys too; at 30
// both show keys.
func TestPaletteKeyedIcons(t *testing.T) {
	for _, c := range []struct {
		cols         int
		query        string
		icons, keyed bool
	}{
		{26, "", false, true},
		{26, "agent", true, false},
		{30, "agent", true, true},
	} {
		p := NewPalette(demodata.Mock())
		p.Resize(c.cols, 44)
		p.SetQuery(c.query)
		if p.Icons() != c.icons || p.Keyed() != c.keyed {
			t.Errorf("%d cells, query %q: icons %v keyed %v, want %v %v", c.cols, c.query, p.Icons(), p.Keyed(), c.icons, c.keyed)
		}
	}
}

// The rows that fit at each height (place): the room in drawn lines,
// rounded down at the scale, then the chrome's five lines off that room
// over the scale, whether or not there is a status.
func TestPalettePlace(t *testing.T) {
	for _, c := range []struct{ lines, scale, fit int }{
		{24, 100, 9}, {30, 80, 11}, {38, 80, 15}, {44, 80, 16}, {45, 80, 17},
		{50, 80, 20}, {55, 80, 21}, {80, 80, 30},
	} {
		if _, _, fit := place(80, c.lines, c.scale); fit != c.fit {
			t.Errorf("%d lines at %d%%: %d rows fit, want %d", c.lines, c.scale, fit, c.fit)
		}
	}
}

// In Dense, Pad's step is a row, so the query and the status take a row
// more each than in Compact, and the box grows with them.
func TestPaletteBoxDense(t *testing.T) {
	p := NewPalette(demodata.Mock())
	p.Resize(80, 50)
	p.SetQuery("split")
	compact := p.drawnLines()
	p.SetDensity(surfui.Dense)
	if got := p.drawnLines(); got != compact+1 {
		t.Errorf("dense: %d lines, want compact's %d and the query's row", got, compact)
	}
	p.Note = "Copied pane.split"
	p.SetDensity(surfui.Compact)
	compact = p.drawnLines()
	p.SetDensity(surfui.Dense)
	if got := p.drawnLines(); got != compact+2 {
		t.Errorf("dense with a status: %d lines, want compact's %d and a row each for the query and the status", got, compact)
	}
	if got, want := p.BoxHeight(), (p.drawnLines()*p.Scale()+99)/100+1; got != want {
		t.Errorf("box: %d cells, want %d", got, want)
	}
}

// Typing lists the rows for the query, its matches in the accent and
// semibold, and selects row 0; Enter on a command runs it and closes.
func TestPaletteRuns(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		a, p := openPalette(t, m)
		a.keys("arrow_down", "arrow_down")
		a.Wait("row 2 selected", func() bool { return a.Has(palRow("pane.reopen"), "is-sel") })
		a.fill(palQuery, "zoom")
		a.Wait("pane.zoom first and selected", func() bool { return a.Has(palRow("pane.zoom"), "is-sel") })
		if got := a.html(); !strings.Contains(got, `class="u-text u-tone-accent u-semi">zoom</span>`) {
			t.Errorf("zoom is not marked: %s", got)
		}
		a.keys("enter")
		a.acted("run pane.zoom")
		a.until("the palette closes", func() bool { return p.Closed })
	})
}

// A click selects a row, and a click on the selected row runs it; a
// double click on another runs it at once.
func TestPaletteClicks(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		a, _ := openPalette(t, m)
		a.click(palRow("pane.reopen"))
		a.Wait("pane.reopen selected", func() bool { return a.Has(palRow("pane.reopen"), "is-sel") })
		if acts := a.src.Acts(); len(acts) != 0 {
			t.Fatalf("a click ran %q", acts)
		}
		a.click(palRow("pane.reopen"))
		a.acted("run pane.reopen")

		b, _ := openPalette(t, m)
		b.double(palRow("tab.new"))
		b.acted("run tab.new")
	})
}

// Run… stages a command's value: the line becomes its id and a space (an
// a=field from the program), the status says the argument, and its
// values are listed; the last value runs it.
func TestPaletteStages(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		a, p := openPalette(t, m)
		a.fill(palQuery, "split")
		a.Wait("pane.split first", func() bool { return a.Has(palRow("pane.split"), "is-sel") })
		if got := a.text(palFoot); !strings.Contains(got, "Run…") || strings.Contains(got, "Run and stay") {
			t.Errorf("a command that takes a value: footer %q", got)
		}
		a.keys("enter")
		a.Wait("the line staged", func() bool { return a.value(palQuery) == "pane.split " })
		if got := a.text(palStatus); got != "pane.split · direction" {
			t.Errorf("status %q", got)
		}
		if got := rowsShown(a); !slices.Equal(got, []string{"right", "down", "left", "up"}) {
			t.Errorf("values %q", got)
		}
		a.keys("arrow_down", "enter")
		a.Wait("the direction staged", func() bool { return a.value(palQuery) == "pane.split down " })
		a.Wait("the free-form value's prompt", func() bool { return a.text(palEmpty) == "Type command" })
		if got := a.text(palStatus); got != "pane.split · command" {
			t.Errorf("status %q", got)
		}
		a.keys("enter")
		a.acted("run pane.split down")
		a.until("the palette closes", func() bool { return p.Closed })
	})
}

// Complete (ctrl+space) writes the row's id on the line, and a space
// where it takes a value.
func TestPaletteCompletes(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		a, _ := openPalette(t, m)
		a.fill(palQuery, "theme")
		a.Wait("theme.use selected", func() bool { return a.Has(palRow("theme.use"), "is-sel") })
		a.keys("ctrl+space")
		a.Wait("the line completed", func() bool { return a.value(palQuery) == "theme.use " })
		a.Wait("the themes listed", func() bool { return a.Has(palRow("ink"), "is-sel") })
		if got := a.text(palRow("ink")); !strings.Contains(got, "in effect") {
			t.Errorf("the theme in force %q", got)
		}
		if got := rowsShown(a); !slices.Equal(got, []string{"ink", "paper", "dusk", "harbor", "moss", "ember"}) {
			t.Errorf("themes %q", got)
		}
	})
}

// Run and stay runs and keeps the palette, which lists the command first
// now, used; Copy command copies its line and says so; Remove from used
// puts it back among the rest.
func TestPaletteActionKeys(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		a, p := openPalette(t, m)
		a.keys("arrow_down") // pane.run takes a value: Run… stages
		a.keys("arrow_up")
		a.Host.Type("\x1b[13;5u") // ctrl+enter, mod+enter on Linux
		a.acted("run pane.new")
		a.Wait("pane.new used, first", func() bool { return len(rowsShown(a)) > 0 && rowsShown(a)[0] == "pane_2enew" })
		a.until("pane.new used", func() bool { return len(p.Rows) > 0 && p.Rows[0].Used })
		a.Host.Type("\x1b[13;3u") // alt+enter
		a.acted("run pane.new", "copy pane.new")
		a.Wait("the status", func() bool { return a.text(palStatus) == "Copied pane.new" })
		a.Host.Type("\x1b[3;2~") // shift+delete
		a.acted("run pane.new", "copy pane.new", "forget pane.new")
		a.until("pane.new no longer used", func() bool { return len(p.Rows) > 0 && !p.Rows[0].Used })
		if a.closed(p) {
			t.Error("the palette closed")
		}
	})
}

// ctrl+k opens the actions panel in the rows' place: its heading, an
// item per action with its key, Enter's always enter, and the footer's
// Back and Run. Arrows move in it, Enter runs its action, and Escape
// closes the panel only.
func TestPaletteActionsPanel(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		a, p := openPalette(t, m)
		a.fill(palQuery, "split")
		a.Wait("pane.split first", func() bool { return a.Has(palRow("pane.split"), "is-sel") })
		a.keys("ctrl+k")
		a.Wait("the panel", func() bool { return a.text(palHead) == "Actions: Split" })
		if got := a.text(palAct("run")); got != "Run…\U000F0311" {
			t.Errorf("Run… %q", got)
		}
		if got := a.text(palAct("copy")); got != "Copy commandalt+\U000F0311" {
			t.Errorf("Copy command %q", got)
		}
		if foot := a.text(palFoot); !strings.Contains(foot, "pane.split") || !strings.Contains(foot, "Backesc") || !strings.Contains(foot, "Run\U000F0311") {
			t.Errorf("footer %q", foot)
		}
		if !a.Has(palAct("run"), "is-sel") {
			t.Error("the first action is not selected")
		}
		a.keys("arrow_down")
		a.Wait("Copy command selected", func() bool { return a.Has(palAct("copy"), "is-sel") })
		a.keys("escape")
		a.Wait("the panel closed", func() bool { return a.Has(palRow("pane.split"), "is-sel") })
		if a.closed(p) {
			t.Fatal("escape closed the palette")
		}
		a.keys("ctrl+k", "arrow_down", "enter")
		a.acted("copy pane.split")
		a.Wait("the status", func() bool { return a.text(palStatus) == "Copied pane.split" })
		a.keys("escape")
		a.until("the palette closes", func() bool { return p.Closed })
	})
}

// What the list says with no rows; a refusal is the status, in the error
// tone, and the palette stays.
func TestPaletteNoMatchAndRefused(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		a, p := openPalette(t, m)
		a.fill(palQuery, "qqqq")
		a.Wait("no match", func() bool { return a.text(palEmpty) == "No command matches." })
		if strings.Contains(a.text(palFoot), "Actions") {
			t.Error("a footer with no row")
		}
		a.fill(palQuery, "@qqqq")
		a.Wait("nothing after @", func() bool { return a.text(palEmpty) == "Nothing matches." })

		a.fill(palQuery, "palette scale")
		a.Wait("the setting's row", func() bool { return a.Has(palRow("palette.style.scale"), "is-sel") })
		if got := a.text(palRow("palette.style.scale")); !strings.Contains(got, "Setting: Palette scale") ||
			!strings.Contains(got, "palette.style.scale · 80") {
			t.Errorf("the setting's row %q", got)
		}
		a.keys("enter")
		a.Wait("its stage", func() bool { return a.value(palQuery) == "config.set palette.style.scale " })
		a.fill(palQuery, "config.set palette.style.scale 30")
		a.Wait("the typed value", func() bool { return a.Has(palRow("30"), "is-sel") })
		a.keys("enter")
		a.acted("run config.set palette.style.scale 30")
		a.Wait("the refusal", func() bool {
			return a.text(palStatus) == "✗ palette.style.scale wants a number in 60..200, got 30" && a.Has(palStatus+".0", "u-tone-error")
		})
		if a.closed(p) {
			t.Error("a refusal closed the palette")
		}
	})
}

// Typing selects row 0; deleting characters keeps the selection.
func TestPaletteSelectionOnTyping(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		a, p := openPalette(t, m)
		a.fill(palQuery, "pa")
		a.keys("arrow_down", "arrow_down")
		a.until("row 2", func() bool { return p.Sel == 2 })
		a.fill(palQuery, "p")
		a.until("kept on a deletion", func() bool { return p.Query == "p" && p.Sel == 2 })
		a.fill(palQuery, "pa")
		a.until("row 0 on typing", func() bool { return p.Query == "pa" && p.Sel == 0 })
		// The arrows go round the ends.
		a.keys("arrow_up")
		a.until("the last row", func() bool { return p.Sel == len(p.Rows)-1 })
	})
}

// One change sends a few ops (F10): the selection's move is two class
// changes, not the list again.
func TestPaletteMoveIsAFewOps(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		a, _ := openPalette(t, m)
		n := len(a.Ops())
		a.keys("arrow_down")
		a.Wait("row 1 selected", func() bool { return a.Has(palRow("pane.run"), "is-sel") })
		ops := a.opsSince(n)
		if len(ops) == 0 || len(ops) > 8 {
			t.Errorf("%d ops for a move: %+v", len(ops), ops)
		}
		for _, op := range ops {
			if op.Op == "morph" || op.Op == "replace" {
				t.Errorf("a move sent %s %s", op.Op, op.Target)
			}
		}
	})
}

// The cells UI (F7) draws the same state: the box, the query, the rows
// with their keys, and takes typing and the palette's keys; Tab
// completes a command that takes a value with a space, which lists its
// values.
func TestPaletteCells(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		a, p := openPalette(t, m)
		a.cells()
		a.screenHas("❯")
		a.screenHas("Pane: New")
		a.screenHas("╭")
		for _, ch := range "zoom" {
			a.keys(string(ch))
		}
		a.until("the query typed in the cells", func() bool { return p.Query == "zoom" })
		a.screenHas("Pane: Toggle zoom")
		a.keys("backspace", "backspace", "backspace", "backspace", "t", "h", "e", "m", "e")
		a.until("theme", func() bool { return p.Query == "theme" })
		a.keys("tab")
		a.until("tab completes in the cells", func() bool { return p.Query == "theme.use " })
		a.screenHas("in effect")
		a.keys("enter")
		a.acted("run theme.use ink")
		a.until("the palette closes", func() bool { return p.Closed })
	})
}

// Back on the host after the cells, the host shows what the cells left.
func TestPaletteCellsAndBack(t *testing.T) {
	inModes(t, func(t *testing.T, m mode) {
		a, p := openPalette(t, m)
		a.cells()
		a.keys("z", "o", "o", "m")
		a.until("typed", func() bool { return p.Query == "zoom" })
		a.Call(a.Run.Recover)
		a.Wait("the host's rows", func() bool { return a.Has(palRow("pane.zoom"), "is-sel") })
	})
}
