package gallery

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottytest"

	surf "neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/screens"
	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/surfui"
)

// Every story draws, with no error, in a gallery that draws with none,
// in both modes: each widget's own, stories.All's, and each stories
// file's, the scenes among them. Each screen with a cells UI draws by it too, and each story
// with states in each of them.
func TestEveryStoryDraws(t *testing.T) {
	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			g := New()
			var errs []string
			root := mode.of(g)
			hear := func(err error) { errs = append(errs, err.Error()) }
			switch v := root.(type) {
			case *surf.View:
				v.OnError = hear
			case *surf.CompiledView:
				v.OnError = hear
			default:
				t.Fatalf("the gallery is a %T", root)
			}
			app := surflive.New(root)
			kinds := map[string]int{}
			for _, s := range g.all {
				g.Pick(s.id)
				tree := app.Render()
				surfui.Resolve(tree, "r")
				if g.cur.err != "" || len(g.cur.errs) > 0 || len(errs) > 0 {
					t.Errorf("%s (%s): %q %q %q", s.id, g.Title(), g.cur.err, g.cur.errs, errs)
					errs = nil
				}
				if !strings.Contains(surfui.HTML(tree), `id="r.kpage.kmain.kmid.kc-`+s.id+`"`) {
					t.Errorf("%s: no canvas in\n%.400s", s.id, surfui.HTML(tree))
				}
				if g.HasCells() {
					g.FlipCells()
					tree := app.Render()
					surfui.Resolve(tree, "r")
					if !strings.Contains(surfui.HTML(tree), `kc-`+s.id+`.kstory.kcells`) {
						t.Errorf("%s: no cells drawn", s.id)
					}
					g.FlipCells()
					kinds["cells"]++
				}
				if g.HasStates() {
					g.FlipStates()
					tree := app.Render()
					surfui.Resolve(tree, "r")
					for _, v := range g.matrix {
						if v.d.err != "" || len(v.d.errs) > 0 {
							t.Errorf("%s in its state %s: %q %q", s.id, v.st.name, v.d.err, v.d.errs)
						}
					}
					if !strings.Contains(surfui.HTML(tree), `kc-`+s.id+`.kstory.kmatrix.km-focus`) {
						t.Errorf("%s: no states drawn", s.id)
					}
					g.FlipStates()
					kinds["states"]++
				}
				switch {
				case s.widget != nil:
					kinds["widget"]++
				case s.tree != nil:
					kinds["tree"]++
				case s.file.scene:
					kinds["scene"]++
				case s.file.screen():
					kinds["screen"]++
				default:
					kinds["composite"]++
				}
			}
			if kinds["widget"] != len(surf.Widgets) || kinds["tree"] == 0 || kinds["composite"] < 15 || kinds["scene"] == 0 || kinds["screen"] < 10 || kinds["cells"] < 6 ||
				kinds["states"] != kinds["widget"]+kinds["composite"] {
				t.Errorf("the stories drawn: %v", kinds)
			}
		})
	}
}

// modes are the gallery's two: its templates interpreted and compiled.
var modes = []struct {
	name string
	of   func(*Gallery) surflive.Component
}{{"interpreted", Interpreted}, {"compiled", Compiled}}

// hosts are the gallery's two hosts here: surfrun.Runtime, and surftest's host
// of its own (a surfrun.Surface), which sends the stylesheet and the
// resources the gallery sends through its runner as the Runtime does.
var hosts = []struct {
	name  string
	start starter
}{{"runtime", surftest.Start}, {"hosted", surftest.StartHosted}}

// starter starts a model on a test host: surftest.Start or StartHosted.
type starter = func(testing.TB, surfrun.Model, surfrun.Options, ...hottytest.Option) *surftest.App

// The gallery's elements, by their keys in gallery.surf.
const (
	barID    = "r.kpage.kbar"
	fitID    = barID + ".kfit.kwhy"
	sideID   = "r.kpage.kmain.kside"
	asideID  = "r.kpage.kmain.kaside"
	navID    = sideID + ".knav"
	filterID = navID + ".ksearch.kfilter"
	listID   = navID + ".klist"
	titleID  = "r.kpage.kmain.kmid.ktitle"
	logID    = "r.kpage.kmain.kmid.klog"
)

// barCtl is the toolbar's control key in a window wide enough for the
// toolbar to be one row (startOn's); barAt is it in the toolbar's row.
func barCtl(key string) string { return barAt(0, key) }
func barAt(row int, key string) string {
	group := map[string]string{"theme": "canvas", "density": "canvas", "width": "canvas",
		"cells": "views", "states": "views", "reference": "views", "stories": "panes", "controls": "panes"}[key]
	return barID + ".kr" + strconv.Itoa(row) + ".k" + group + ".k" + key
}

// canvasID is the canvas while story sel is picked, storyID the stack
// round its tree in it.
func canvasID(sel string) string { return "r.kpage.kmain.kmid.kc-" + sel }
func storyID(sel string) string  { return canvasID(sel) + ".kstory" }

// controlID is the control key of story sel's.
func controlID(sel, key string) string {
	return "r.kpage.kmain.kaside.kp-" + sel + ".k" + key + ".kcontrol"
}

// start runs a new gallery in mode on a test host, with story sel
// picked, and waits until its canvas shows it.
func start(t *testing.T, of func(*Gallery) surflive.Component, sel string) (*surftest.App, *Gallery) {
	t.Helper()
	return startOf(t, New(), of, sel)
}

// startOf runs g so, in a window wide enough for a canvas of 120 cells
// beside the stories and the controls.
func startOf(t *testing.T, g *Gallery, of func(*Gallery) surflive.Component, sel string) (*surftest.App, *Gallery) {
	t.Helper()
	return startOn(t, surftest.Start, g, of, sel)
}

// startOn is startOf on the host start runs.
func startOn(t *testing.T, start starter, g *Gallery, of func(*Gallery) surflive.Component, sel string) (*surftest.App, *Gallery) {
	t.Helper()
	g.Pick(sel)
	a := start(t, surflive.New(of(g)), Options(g), hottytest.Size(200, 60))
	a.Wait("the canvas of "+sel, func() bool { _, ok := a.Surface().Element(canvasID(sel)); return ok })
	return a, g
}

// logged is the log as the surface shows it, a line each.
func logged(a *surftest.App) []string {
	var out []string
	for i := 0; i < logLines; i++ {
		l := a.Surface().TextOf(logID + ".kl" + strconv.Itoa(i))
		if l == "" {
			break
		}
		out = append(out, l)
	}
	return out
}

// waitLog waits for a line of the log with each of want in it.
func waitLog(t *testing.T, a *surftest.App, want ...string) {
	t.Helper()
	a.Wait("a log line with "+strings.Join(want, " and "), func() bool {
		for _, l := range logged(a) {
			if all(l, want) {
				return true
			}
		}
		return false
	})
}

func all(s string, subs []string) bool {
	for _, w := range subs {
		if !strings.Contains(s, w) {
			return false
		}
	}
	return true
}

// idOf is the id of the element of surface html with tag whose text is
// text.
func idOf(t *testing.T, html, tag, text string) string {
	t.Helper()
	m := regexp.MustCompile(`<` + tag + ` id="([^"]+)"[^>]*>` + regexp.QuoteMeta(text) + `</` + tag + `>`).FindStringSubmatch(html)
	if m == nil {
		t.Fatalf("no <%s> %q in\n%s", tag, text, html)
	}
	return m[1]
}

// A story picked in the sidebar is drawn in the canvas; the filter
// narrows the list as it is typed, and a story it shows is picked by a
// click too.
func TestPickAndFilter(t *testing.T) {
	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			a, g := start(t, mode.of, "w-surface")
			// A group is folded until a click on it, or on its chevron,
			// unfolds it; the picked story's is open.
			if _, folded := a.Surface().Element(listID + ".kw-button"); folded {
				t.Fatal("the button's stories show before its group is opened")
			}
			if err := a.Host.Click(a.Name, listID+".kg-widgets-button"); err != nil {
				t.Fatal(err)
			}
			a.Wait("the button's group open", func() bool { _, ok := a.Surface().Element(listID + ".kw-button"); return ok })
			if err := a.Host.Click(a.Name, listID+".kw-button"); err != nil {
				t.Fatal(err)
			}
			a.Wait("the button's story", func() bool {
				return a.Surface().TextOf(titleID) == "button / attributes" && a.Surface().TextOf(storyID("w-button")) == "Restart"
			})
			if !a.Has(listID+".kw-button", "is-sel") {
				t.Errorf("the picked story is not the list's selection")
			}

			if err := a.Host.Fill(a.Name, filterID, "notice"); err != nil {
				t.Fatal(err)
			}
			a.Wait("the list narrowed to the notices'", func() bool {
				_, button := a.Surface().Element(listID + ".kw-button")
				_, read := a.Surface().Element(listID + ".kf-notice-row-read")
				_, screen := a.Surface().Element(listID + ".kf-notices-errors")
				return !button && read && screen
			})
			if err := a.Host.Click(a.Name, listID+".kf-notice-row-read"); err != nil {
				t.Fatal(err)
			}
			a.Wait("the notice row's story", func() bool {
				return a.Surface().TextOf(titleID) == "notice-row / read" &&
					strings.Contains(a.Surface().TextOf(storyID("f-notice-row-read")), "hotty-blitz 0.5.3 built")
			})
			// While the list is filtered every group is open, and counts
			// the stories the filter leaves.
			if got := a.Surface().TextOf(listID + ".kg-screens-notices"); !strings.Contains(got, "notices") || !strings.HasSuffix(got, "2") {
				t.Errorf("the notices' group: %q", got)
			}
			_ = g

			if err := a.Host.Fill(a.Name, filterID, "zzz"); err != nil {
				t.Fatal(err)
			}
			a.Wait("no story", func() bool {
				return a.Surface().TextOf(navID+".knone") == `No story has "zzz" in its name.`
			})
		})
	}
}

// A control changes the canvas at once: a widget's attribute, which its
// template is made again with, and a partial's prop, a literal and a
// struct's field, which its prop's method gives.
func TestControls(t *testing.T) {
	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			a, g := start(t, mode.of, "w-button")
			button := storyID("w-button") + ".0.0" // in its row
			if err := a.Host.Fill(a.Name, controlID("w-button", "text"), "Go"); err != nil {
				t.Fatal(err)
			}
			a.Wait("the button's new text", func() bool { return a.Surface().TextOf(button) == "Go" })
			if err := a.Host.Click(a.Name, controlID("w-button", "bold")); err != nil {
				t.Fatal(err)
			}
			a.Wait("the button bold", func() bool { return a.Has(button, "u-bold") })
			a.Pick(controlID("w-button", "fill"), "ok")
			a.Wait("the button filled", func() bool { return a.Has(button, "u-fill-ok") })
			a.Pick(controlID("w-button", "fill"), "none")
			a.Wait("the button with no fill", func() bool { return !a.Has(button, "u-fill-ok") })

			a.Call(func() { g.Pick("f-foot-hint-one-key") })
			sel := "f-foot-hint-one-key"
			a.Wait("the foot hint", func() bool { return strings.Contains(a.Surface().TextOf(storyID(sel)), "Run") })
			if v, _ := a.Surface().Value(controlID(sel, "keys")); v != "enter" {
				t.Errorf("the keys control holds %q, want the story's enter", v)
			}
			if err := a.Host.Fill(a.Name, controlID(sel, "title"), "Run it now"); err != nil {
				t.Fatal(err)
			}
			a.Wait("the hint's new title", func() bool { return strings.Contains(a.Surface().TextOf(storyID(sel)), "Run it now") })

			sel = "f-notice-row-read"
			a.Call(func() { g.Pick(sel) })
			a.Wait("the notice's text in its control", func() bool {
				v, _ := a.Surface().Value(controlID(sel, "notice-text"))
				return v == "hotty-blitz 0.5.3 built"
			})
			if err := a.Host.Fill(a.Name, controlID(sel, "notice-text"), "A notice of the gallery's"); err != nil {
				t.Fatal(err)
			}
			a.Wait("the notice's new text", func() bool {
				return strings.Contains(a.Surface().TextOf(storyID(sel)), "A notice of the gallery's")
			})
			if err := a.Host.Click(a.Name, controlID(sel, "notice-unread")); err != nil {
				t.Fatal(err)
			}
			a.Wait("the notice unread", func() bool { return strings.Contains(a.Surface().TextOf(storyID(sel)), "●") })
		})
	}
}

func TestFieldStoryStartsWithEditableText(t *testing.T) {
	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			a, _ := start(t, mode.of, "w-field")
			control := controlID("w-field", "value")
			field := storyID("w-field") + ".0"
			if got, _ := a.Surface().Value(field); got != "Sample text" {
				t.Fatalf("example field = %q", got)
			}
			if err := a.Host.Fill(a.Name, control, "Edited"); err != nil {
				t.Fatal(err)
			}
			a.Wait("the field's edited value", func() bool {
				v, _ := a.Surface().Value(field)
				return v == "Edited"
			})
		})
	}
}

func TestListStoryShowsASelection(t *testing.T) {
	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			a, _ := start(t, mode.of, "w-list")
			item := storyID("w-list") + ".0.0"
			if !a.Has(item, "is-sel") {
				t.Fatal("list example has no selected item")
			}
			if err := a.Host.Click(a.Name, controlID("w-list", "nav")); err != nil {
				t.Fatal(err)
			}
			a.Wait("selected navigation item", func() bool { return a.Has(item, "is-sel") && a.Has(storyID("w-list")+".0", "u-nav") })
		})
	}
}

func TestRowStoryUsesChildrenTheLayoutCanShow(t *testing.T) {
	for _, mode := range modes {
		for _, tc := range []struct{ control, childClass, selected string }{
			{"segmented", "u-btn", "is-sel"},
			{"grid", "u-btn", "is-sel"},
			{"strip", "u-row", ""},
			{"search", "u-icon", ""},
		} {
			t.Run(mode.name+"/"+tc.control, func(t *testing.T) {
				a, _ := start(t, mode.of, "w-row")
				if err := a.Host.Click(a.Name, controlID("w-row", tc.control)); err != nil {
					t.Fatal(err)
				}
				root := storyID("w-row") + ".0"
				a.Wait("layout's sample children", func() bool {
					return a.Has(root+".0", tc.childClass) && (tc.selected == "" || a.Has(root+".1", tc.selected))
				})
				if tc.control == "search" && !a.Has(root+".1", "u-field") {
					t.Fatal("search has no editable field")
				}
				if tc.control == "search" {
					if err := a.Host.Fill(a.Name, root+".1", "palette"); err != nil {
						t.Fatal(err)
					}
					a.Wait("search input", func() bool { return strings.Contains(a.Surface().TextOf(logID), "palette") })
				}
			})
		}
	}
}

func TestRevealExamplesHaveAnItemToPointAt(t *testing.T) {
	g := New()
	for _, w := range surf.Widgets {
		g.Pick("w-" + w.Tag)
		i := slices.IndexFunc(g.Controls, func(c *Control) bool { return c.Name == "reveal" })
		if w.Tag == "spacer" {
			if i >= 0 {
				t.Error("a spacer offers reveal despite having no visible content")
			}
			continue
		}
		if i < 0 {
			t.Errorf("%s has no reveal control", w.Tag)
			continue
		}
		g.Flip(i)
		if g.cur.err != "" {
			t.Errorf("%s with reveal: %s", w.Tag, g.cur.err)
			continue
		}
		root := g.cur.render(nil)
		if root == nil || g.cur.node(root) == root || !strings.Contains(surfui.HTML(root), "Hover or focus") {
			t.Errorf("%s has no reveal context and target", w.Tag)
		}
	}
}

func TestRevealContextInTheLiveGallery(t *testing.T) {
	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			a, _ := start(t, mode.of, "w-text")
			if err := a.Host.Click(a.Name, controlID("w-text", "reveal")); err != nil {
				t.Fatal(err)
			}
			root := storyID("w-text") + ".0"
			a.Wait("reveal's item and target", func() bool {
				return a.Has(root, "u-item") && a.Has(root+".1", "u-reveal") && strings.Contains(a.Surface().TextOf(root), "Hover or focus")
			})
		})
	}
}

func TestWidgetControlsFollowWhatTheExampleCanShow(t *testing.T) {
	g := New()
	has := func(name string) bool {
		return slices.ContainsFunc(g.Controls, func(c *Control) bool { return c.Name == name })
	}
	for _, tc := range []struct{ story, absent, present string }{
		{"w-text", "checked", "tone"},
		{"w-text", "invalid", "wrap"},
		{"w-field", "gap", "value"},
		{"w-field", "wrap", "value"},
		{"w-field", "scroll", "value"},
		{"w-field", "top", "bare"},
		{"w-row", "checked", "gap"},
		{"w-row", "bare", "top"},
		{"w-switch", "tone", "checked"},
		{"w-item", "invalid", "selected"},
		{"w-heading", "bold", "text"},
		{"w-heading", "semibold", "text"},
		{"w-heading", "small", "text"},
		{"w-text", "checked", "bold"},
		{"w-text", "checked", "semibold"},
		{"w-text", "checked", "small"},
	} {
		g.Pick(tc.story)
		if has(tc.absent) || !has(tc.present) {
			t.Errorf("%s: %s present = %v, %s present = %v", tc.story, tc.absent, has(tc.absent), tc.present, has(tc.present))
		}
	}
	g.Pick("w-row")
	search := slices.IndexFunc(g.Controls, func(c *Control) bool { return c.Name == "search" })
	if search < 0 || !has("children (count)") {
		t.Fatal("row lacks search or its normal child count")
	}
	g.Flip(search)
	if has("children (count)") {
		t.Fatal("search offers a child count that cannot affect its fixed sample")
	}
	search = slices.IndexFunc(g.Controls, func(c *Control) bool { return c.Name == "search" })
	g.Flip(search)
	if !has("children (count)") {
		t.Fatal("row did not restore its child count after search")
	}
	g.Pick("w-button")
	if has("raised") || has("current") || has("selected") {
		t.Fatal("non-pill button offers pill fills")
	}
	pill := slices.IndexFunc(g.Controls, func(c *Control) bool { return c.Name == "pill" })
	if pill < 0 {
		t.Fatal("button has no pill control")
	}
	g.Flip(pill)
	if !has("raised") || !has("current") || !has("selected") {
		t.Fatal("pill lacks its fill controls")
	}
	raised := slices.IndexFunc(g.Controls, func(c *Control) bool { return c.Name == "raised" })
	g.Flip(raised)
	variants := g.variants()
	if len(variants) < 2 || !slices.ContainsFunc(variants[1].d.controls, func(c *Control) bool { return c.Name == "raised" && c.Value == "true" }) {
		t.Fatal("states matrix lost the pill's raised setting")
	}
	pill = slices.IndexFunc(g.Controls, func(c *Control) bool { return c.Name == "pill" })
	g.Flip(pill)
	if has("raised") || has("current") || has("selected") {
		t.Fatal("pill fills remained on a non-pill")
	}
}

func TestPillControlsAppearInTheLiveGallery(t *testing.T) {
	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			a, _ := start(t, mode.of, "w-button")
			raised := controlID("w-button", "raised")
			if _, ok := a.Surface().Element(raised); ok {
				t.Fatal("raised is visible before pill")
			}
			if err := a.Host.Click(a.Name, controlID("w-button", "pill")); err != nil {
				t.Fatal(err)
			}
			a.Wait("pill controls", func() bool { _, ok := a.Surface().Element(raised); return ok })
		})
	}
}

func TestBadgeControlsFollowTheVisibleBadge(t *testing.T) {
	g := New()
	g.Pick("w-button")
	index := func(name string) int {
		return slices.IndexFunc(g.Controls, func(c *Control) bool { return c.Name == name })
	}
	if index("badge-tone") >= 0 {
		t.Fatal("tone offered with no badge")
	}
	g.Flip(index("badge-dot"))
	if index("badge-tone") < 0 {
		t.Fatal("dot badge lacks its tone")
	}
	g.Choose(index("badge"), "2")
	if index("badge-dot") >= 0 || index("badge-tone") < 0 {
		t.Fatal("count badge offers a dot or lacks its tone")
	}
	g.Choose(index("badge"), "0")
	if index("badge-dot") < 0 || index("badge-tone") < 0 {
		t.Fatal("dot setting was not retained")
	}
	g.Flip(index("badge-dot"))
	if index("badge-tone") >= 0 {
		t.Fatal("tone remained with no badge")
	}
}

// A handler the story calls is logged, with its arguments: a click's
// $event by its fields, a field's $value, a pick's, and a partial's call
// of its user's method with a struct, which the method still gets.
func TestLog(t *testing.T) {
	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			a, g := start(t, mode.of, "w-button")
			button := storyID("w-button") + ".0.0" // in its row
			if err := a.Host.Click(a.Name, button); err != nil {
				t.Fatal(err)
			}
			waitLog(t, a, "click(event{kind: click, target: "+button, "count: 1})")

			a.Call(func() { g.Pick("w-field") })
			field := storyID("w-field") + ".0"
			a.Wait("the field", func() bool { _, ok := a.Surface().Element(field); return ok })
			if err := a.Host.Fill(a.Name, field, "abc"); err != nil {
				t.Fatal(err)
			}
			waitLog(t, a, `input("abc")`)
			waitLog(t, a, "focus(event{kind: focus, target: "+field)
			a.Wait("the value control at abc", func() bool { v, _ := a.Surface().Value(controlID("w-field", "value")); return v == "abc" })

			a.Call(func() { g.Pick("w-select") })
			sel := storyID("w-select") + ".0.0"
			a.Wait("the select", func() bool { _, ok := a.Surface().Element(sel); return ok })
			a.Pick(sel, "gamma")
			waitLog(t, a, `change("gamma")`)

			a.Call(func() { g.Pick("f-notice-row-unread-error") })
			a.Wait("the notice", func() bool {
				return strings.Contains(a.Surface().TextOf(storyID("f-notice-row-unread-error")), "Dismiss")
			})
			if err := a.Host.Click(a.Name, idOf(t, a.Surface().HTML(), "button", "Dismiss")); err != nil {
				t.Fatal(err)
			}
			waitLog(t, a, `dismiss(Notice{id: "n1", level: "error"`)
			var rows int
			a.Call(func() { rows = len(g.cur.state.(*screens.Notices).Rows) })
			if rows != 7 {
				t.Errorf("the notices have %d rows after the dismiss, want 7", rows)
			}
			if n := len(logged(a)); n != 1 {
				t.Errorf("the log has %d lines, want the dismiss's only: %q", n, logged(a))
			}
		})
	}
}

// The toolbar: the canvas takes its theme, its density and its width,
// and a screen with a cells UI is drawn by it.
func TestToolbar(t *testing.T) {
	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			sel := "f-palette-empty-query"
			a, _ := start(t, mode.of, sel)
			c := canvasID(sel)
			a.Pick(barCtl("theme"), "paper")
			a.Wait("the canvas in paper", func() bool { return a.Has(c, "u-theme-paper") && a.Has("r", "u-theme-graphite") })
			a.Pick(barCtl("density"), "roomy")
			a.Wait("the canvas roomy", func() bool { return a.Has(c, "u-roomy") && a.Has("r", "u-dense") })
			if w, _ := a.Surface().Var(c, "--u-w"); w != "80" {
				t.Errorf("the canvas is %q cells wide, want the story's 80", w)
			}
			if err := a.Host.Click(a.Name, barCtl("width")+".2"); err != nil {
				t.Fatal(err)
			}
			a.Wait("the canvas wider", func() bool { w, _ := a.Surface().Var(c, "--u-w"); return w == "100" })
			if err := a.Host.Click(a.Name, barCtl("cells")); err != nil {
				t.Fatal(err)
			}
			a.Wait("the palette in cells", func() bool {
				_, ok := a.Surface().Element(storyID(sel) + ".kcells")
				return ok && a.Has(c, "u-dense")
			})
		})
	}
}

// The wheel scrolls the stories, the canvas and the controls: each is a
// scroll box, in a document the host lets scroll (hotty SPEC §5.3).
func TestScrolls(t *testing.T) {
	sel := "w-text"
	a, _ := start(t, Interpreted, sel)
	for _, id := range []string{listID, canvasID(sel), "r.kpage.kmain.kaside.kp-" + sel} {
		if !a.Has(id, "u-scroll") {
			t.Errorf("%s is no scroll box", id)
		}
	}
	scrolls := false
	for _, m := range a.Host.Commands() {
		if hotty.Get(m.Control, "a") == "doc" && hotty.Get(m.Control, "scroll") == "1" {
			scrolls = true
		}
	}
	if !scrolls {
		t.Error("the document was sent with no scroll")
	}
}

// The sidebar's arrows are its list's: Tab from the filter gives the
// list the keyboard, the list being the box that scrolls (a box around
// it would take Tab first, and its arrows); the list gives them to the
// program (data-keys) where the host would scroll with them, and they
// move the selection, which either host keeps in view, the box scrolled
// to the story picked. A story picked far down is revealed in the first
// document too.
func TestSidebarKeys(t *testing.T) {
	for _, mode := range modes {
		for _, h := range hosts {
			t.Run(mode.name+"/"+h.name, func(t *testing.T) {
				sidebarKeys(t, h.start, mode.of)
			})
		}
	}
}

func sidebarKeys(t *testing.T, start starter, of func(*Gallery) surflive.Component) {
	g := New()
	ids := g.Stories()
	last, prev := ids[len(ids)-1], ids[len(ids)-2]
	a, _ := startOn(t, start, g, of, last)
	a.Call(func() {}) // the document's reveals, written
	if got := reveals(a.Ops()); !slices.Contains(got, listID+".k"+last) {
		t.Errorf("the document revealed %v, not the last story", got)
	}
	keys, _ := a.Surface().Attr(listID, "data-keys")
	if !slices.Contains(strings.Fields(keys), "arrow_up=program") || !slices.Contains(strings.Fields(keys), "arrow_down=program") {
		t.Errorf("the list's data-keys are %q, which don't give it the arrows", keys)
	}

	a.Wait("the filter has the keyboard", func() bool { return a.Surface().Focused() == filterID })
	a.Host.Key("tab")
	a.Wait("the list has the keyboard", func() bool { return a.Surface().Focused() == listID })
	a.Call(func() {})
	before := len(a.Ops())
	a.Host.Key("arrow_up")
	a.Wait("the story above picked", func() bool { return a.Has(listID+".k"+prev, "is-sel") })
	a.Call(func() {})
	if got := reveals(a.Ops()[before:]); !slices.Contains(got, listID+".k"+prev) {
		t.Errorf("↑ revealed %v, not the story above", got)
	}
}

// reveals are the targets of the reveal ops in ops.
func reveals(ops []hotty.Op) []string {
	var out []string
	for _, o := range ops {
		if o.Op == "reveal" {
			out = append(out, o.Target)
		}
	}
	return out
}

// The States switch draws the story in each of its states, which its
// controls make: a field's, and a composite's, with a row its stories
// file names. Hover and focus are forced on the story's node, by rules
// the run's stylesheet has beside the real ones.
func TestStates(t *testing.T) {
	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			sel := "w-field"
			a, g := start(t, mode.of, sel)
			if _, css, _ := a.Host.Resource(surfui.StyleID); !strings.Contains(string(css), "[data-force~=hover]") {
				t.Error("the run's stylesheet has no forced rules")
			}
			if err := a.Host.Fill(a.Name, controlID(sel, "value"), "alpha"); err != nil {
				t.Fatal(err)
			}
			if err := a.Host.Click(a.Name, barCtl("states")); err != nil {
				t.Fatal(err)
			}
			row := func(sel, name string) string { return storyID(sel) + ".kmatrix.km-" + name }
			a.Wait("the field's states", func() bool {
				for _, n := range []string{"default", "hover", "focus", "disabled", "error", "empty", "long-text"} {
					if _, ok := a.Surface().Element(row(sel, n)); !ok {
						return false
					}
				}
				return true
			})
			field := func(name string) string { return row(sel, name) + ".kin.0" }
			if f, _ := a.Surface().Attr(field("hover"), "data-force"); f != "hover" {
				t.Errorf("the hovered field is forced %q", f)
			}
			if f, _ := a.Surface().Attr(field("focus"), "data-force"); f != "focus within" {
				t.Errorf("the focused field is forced %q", f)
			}
			if f, ok := a.Surface().Attr(field("default"), "data-force"); ok {
				t.Errorf("the default field is forced %q", f)
			}
			if _, ok := a.Surface().Attr(field("disabled"), "disabled"); !ok {
				t.Error("the disabled field is not disabled")
			}
			if !a.Has(field("error"), "is-invalid") {
				t.Error("the field in error is not invalid")
			}
			for name, want := range map[string]string{"default": "alpha", "hover": "alpha", "empty": ""} {
				if v, _ := a.Surface().Value(field(name)); v != want {
					t.Errorf("the %s field holds %q, want %q", name, v, want)
				}
			}
			if v, _ := a.Surface().Value(field("long-text")); len(v) < 100 {
				t.Errorf("the long field holds %q", v)
			}

			sel = "f-notice-row-read"
			if err := a.Host.Click(a.Name, listID+".kg-composites-notice-row"); err != nil {
				t.Fatal(err)
			}
			a.Wait("the notice row's group open", func() bool { _, ok := a.Surface().Element(listID + ".kf-notice-row-read"); return ok })
			if err := a.Host.Click(a.Name, listID+".kf-notice-row-read"); err != nil {
				t.Fatal(err)
			}
			a.Wait("the notice's states", func() bool {
				_, empty := a.Surface().Element(row(sel, "empty"))
				return empty && strings.Contains(a.Surface().TextOf(row(sel, "error")), "error")
			})
			if f, _ := a.Surface().Attr(row(sel, "hover")+".kin.0.0", "data-force"); f != "hover" {
				t.Errorf("the hovered notice's row is forced %q", f)
			}
			if !strings.Contains(a.Surface().TextOf(row(sel, "long-text")), "A long text that runs on") {
				t.Errorf("the long notice: %q", a.Surface().TextOf(row(sel, "long-text")))
			}

			// A tree has no controls, so no states, and no switch for them.
			a.Call(func() { g.Pick("s-button-states") })
			a.Wait("the tree, as it is", func() bool {
				_, button := a.Surface().Element(storyID("s-button-states") + ".0.0")
				_, states := a.Surface().Element(barCtl("states"))
				return button && !states
			})
		})
	}
}

// A story of stories.All has no handlers: what the host sends it is
// logged as it came, on either host.
func TestUnhandled(t *testing.T) {
	for _, h := range hosts {
		t.Run(h.name, func(t *testing.T) {
			sel := "s-button-states"
			a, _ := startOn(t, h.start, New(), Interpreted, sel)
			button := storyID(sel) + ".0.0"
			if err := a.Host.Click(a.Name, button); err != nil {
				t.Fatal(err)
			}
			waitLog(t, a, "(no handler) event{kind: click, target: "+button)
		})
	}
}

// The frame shares the window out round the canvas, which keeps its
// width: the side columns at their widths where they fit, narrowed
// toward their least where they don't, and then left out, the controls
// first unless they were the last turned on; the canvas past the edge of
// a window narrower than it.
func TestFrameOf(t *testing.T) {
	for _, c := range []struct {
		cols, width       int
		stories, controls bool
		keep              bool
		side, aside, cut  int
		sideOut, asideOut bool
	}{
		{0, 80, true, true, false, sideCols, asideCols, 0, false, false},
		{200, 80, true, true, false, sideCols, asideCols, 0, false, false},
		{150, 80, true, true, false, 27, 39, 0, false, false},
		{140, 80, true, true, false, sideMin, asideMin, 0, false, false},
		{139, 80, true, true, false, sideCols, 0, 0, false, true},
		{120, 80, true, true, false, sideCols, 0, 0, false, true},
		{120, 80, true, true, true, 0, 37, 0, true, false},
		{115, 80, true, true, true, sideCols, 0, 0, false, true},
		{95, 80, true, true, false, 0, 0, 0, true, true},
		{95, 120, true, true, false, 0, 0, 27, true, true},
		{95, 24, true, true, false, 24, 37, 0, false, false},
		{60, 24, true, true, false, 27, 0, 0, false, true},
		{200, 80, true, false, false, sideCols, 0, 0, false, false},
		{120, 80, false, true, false, 0, 38 - 1, 0, false, false},
	} {
		f := frameOf(c.cols, c.width, c.stories, c.controls, c.keep)
		want := frame{c.side, c.aside, c.sideOut, c.asideOut, c.cut}
		if f != want {
			t.Errorf("%d cells round a canvas of %d, stories %v, controls %v, controls kept %v: %+v, want %+v",
				c.cols, c.width, c.stories, c.controls, c.keep, f, want)
		}
	}
	// Whatever the window, what is shown fits in it beside the canvas
	// (or the middle column's least), and no column is narrower than its
	// least.
	for cols := 1; cols <= 260; cols++ {
		for _, width := range []int{24, 30, 40, 50, 60, 80, 100, 120} {
			for _, keep := range []bool{false, true} {
				f := frameOf(cols, width, true, true, keep)
				used := framePad + max(width, midMin)
				if f.side > 0 {
					used += f.side + 1
				}
				if f.aside > 0 {
					used += f.aside + 1
				}
				if f.side != 0 && f.side < sideMin || f.aside != 0 && f.aside < asideMin ||
					f.side > sideCols || f.aside > asideCols || (f.side > 0 || f.aside > 0) && used > cols {
					t.Errorf("%d cells round %d: %+v", cols, width, f)
				}
			}
		}
	}
}

// The toolbar takes as many rows as its groups need, in order: one in a
// wide window (from 149 cells), two in half the 1796-px window (95)
// and at 120, and none is wider than the window by the widths frame.go
// reckons, which the shots check against what hotty-blitz draws.
func TestBar(t *testing.T) {
	g := New()
	g.Pick("f-palette-empty-query")
	widths := map[string]int{}
	for _, gr := range g.barGroups() {
		var r BarRow
		gr.put(&r)
		widths[fmt.Sprintf("%v", r)] = gr.cells
	}
	// The palette's toolbar has the switches that act on it: Cells and
	// Stories. It is one row from the width its groups take, each with
	// its gap.
	one := framePad
	for _, gr := range g.barGroups() {
		one += gr.cells + 2
	}
	for cols, rows := range map[int]int{80: 2, one - 1: 2, one: 1, 200: 1} {
		g.Resize(cols, 50)
		bar := g.Bar()
		if len(bar) != rows {
			t.Errorf("%d cells: the toolbar is %d rows, want %d: %+v", cols, len(bar), rows, bar)
		}
		var order []string
		for _, r := range bar {
			used := framePad
			for _, gr := range g.barGroups() {
				var one BarRow
				gr.put(&one)
				if one.Name && r.Name || one.Canvas && r.Canvas || one.Views && r.Views || one.Panes && r.Panes {
					used += gr.cells + 2
					order = append(order, fmt.Sprintf("%v", one))
				}
			}
			if used > cols {
				t.Errorf("%d cells: a row of the toolbar takes %d: %+v", cols, used, r)
			}
		}
		if len(order) != 4 {
			t.Errorf("%d cells: the toolbar's groups %v", cols, order)
		}
	}
}

// A window narrower than the canvas and the columns beside it never
// narrows the canvas: the runtime tells the gallery the window's size as
// it changes (Gallery.Resize, surfrun.Sizer, as a SIGWINCH does), and the
// side columns are left out, the toolbar wraps, and the line under it
// says what is hidden; the story keeps its width. A width the user picks
// is drawn at that width, past the window's edge if it must be.
func TestNarrowWindow(t *testing.T) {
	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			sel := "f-palette-empty-query"
			a, g := start(t, mode.of, sel)
			c := canvasID(sel)
			width := func(id string) string { w, _ := a.Surface().Var(id, "--u-w"); return w }
			shown := func(id string) bool { _, ok := a.Surface().Element(id); return ok }
			storyCols := func() (cols int) {
				a.Call(func() { cols = g.cur.state.(*screens.Palette).Cols })
				return cols
			}
			// The palette has no controls: no column of them, and no switch.
			if width(c) != "80" || width(sideID) != "30" || shown(asideID) || shown(barCtl("controls")) || shown(fitID) {
				t.Fatalf("at 200 cells: the canvas %q, the stories %q, the controls shown %v, the line %q",
					width(c), width(sideID), shown(asideID), a.Surface().TextOf(fitID))
			}

			a.Resize(95, 50) // half the 1796-px window
			a.Wait("the canvas fitted beside the stories at 95", func() bool { return width(c) == "62" && shown(sideID) && !shown(asideID) })
			if width(sideID) != "30" || storyCols() != 62 || shown(fitID) {
				t.Errorf("at 95 cells the stories are %q cells, the story %d, the line %q", width(sideID), storyCols(), a.Surface().TextOf(fitID))
			}
			if !shown(barAt(0, "width")) || !shown(barAt(0, "cells")) || !shown(barAt(1, "stories")) {
				t.Errorf("at 95 cells the toolbar isn't two rows:\n%s", a.Surface().HTML())
			}
			if _, off := a.Surface().Attr(barAt(1, "stories"), "disabled"); off {
				t.Error("at 95 cells the Stories switch can't hide them")
			}

			a.Resize(120, 50)
			a.Wait("the story's own width at 120", func() bool { return width(c) == "80" && shown(sideID) && !shown(asideID) })
			if width(sideID) != "30" || shown(fitID) {
				t.Errorf("at 120 cells the stories are %q cells, and the line says %q", width(sideID), a.Surface().TextOf(fitID))
			}
			a.Call(func() { g.SetWidth("120") })
			a.Wait("the canvas at 120", func() bool { return width(c) == "120" && !shown(sideID) })
			if f := a.Surface().TextOf(fitID); !strings.Contains(f, "its last 2 are cut off") || storyCols() != 120 {
				t.Errorf("a canvas wider than the window: the line says %q, the story is %d", f, storyCols())
			}

			a.Resize(200, 60)
			a.Wait("the stories again at 200", func() bool { return width(sideID) == "30" && !shown(asideID) && !shown(fitID) })
			if width(c) != "120" || !shown(barCtl("stories")) {
				t.Errorf("at 200 cells the canvas is %q, and the toolbar:\n%s", width(c), a.Surface().HTML())
			}
		})
	}
}

// A story wider than the window beside the stories (a settings page of
// 180 cells in a 191-cell window, 1796px at 1.6) is drawn narrower, so
// that the stories stay: the controls give way to it, and nothing says
// the window is too narrow (the maintainer, 2026-11-05).
func TestAWideStoryKeepsTheStories(t *testing.T) {
	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			sel := "s-look-settings"
			a, g := start(t, mode.of, sel)
			c := canvasID(sel)
			width := func(id string) string { w, _ := a.Surface().Var(id, "--u-w"); return w }
			shown := func(id string) bool { _, ok := a.Surface().Element(id); return ok }
			a.Resize(191, 55)
			a.Wait("the stories beside the settings page", func() bool { return width(c) == "158" && width(sideID) == "30" })
			if shown(asideID) || shown(fitID) {
				t.Errorf("at 191 cells the controls are shown %v, the line says %q", shown(asideID), a.Surface().TextOf(fitID))
			}
			a.Call(func() { g.SetWidth("180") })
			a.Wait("the user's width kept", func() bool { return width(c) == "180" && shown(fitID) })
		})
	}
}

// A widget's story places it as the screens use it: a label or a control
// in a row, which hugs it, a spacer between two marks, a row's children
// apart and a box padded round a stack; the states force the widget, not
// its row. A story that draws nothing by design says so in its about.
func TestPlaced(t *testing.T) {
	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			a, g := start(t, mode.of, "w-chip")
			chip := storyID("w-chip") + ".0.0"
			a.Wait("the chip in its row", func() bool {
				return a.Has(storyID("w-chip")+".0", "u-row") && a.Has(chip, "u-chip")
			})

			a.Call(func() { g.Pick("w-spacer") })
			a.Wait("the spacer between its marks", func() bool {
				return a.Surface().TextOf(storyID("w-spacer")) == "startend" && a.Has(storyID("w-spacer")+".0.1", "u-grow")
			})
			a.Call(func() { g.Pick("w-row") })
			a.Wait("the row's children apart", func() bool { return a.Has(storyID("w-row")+".0", "u-gap2") })
			a.Call(func() { g.Pick("w-box") })
			a.Wait("the box padded round a stack", func() bool {
				box := storyID("w-box") + ".0"
				return a.Has(box, "u-px3") && a.Has(box, "u-py1") && a.Has(box+".0", "u-stack") &&
					a.Surface().TextOf(box) == "Child 1Child 2Child 3"
			})

			a.Call(func() { g.Pick("w-chip") })
			a.Wait("the chip again", func() bool { _, ok := a.Surface().Element(chip); return ok })
			if err := a.Host.Click(a.Name, barCtl("states")); err != nil {
				t.Fatal(err)
			}
			hover := storyID("w-chip") + ".kmatrix.km-hover.kin.0"
			a.Wait("the hovered chip", func() bool { f, _ := a.Surface().Attr(hover+".0", "data-force"); return f == "hover" })
			if f, ok := a.Surface().Attr(hover, "data-force"); ok {
				t.Errorf("the chip's row is forced %q", f)
			}

			a.Call(func() { g.Pick("f-key-caps-no-keys") })
			a.Wait("the story's about", func() bool {
				return strings.Contains(a.Surface().TextOf(storyID("f-key-caps-no-keys")), "No keys: the caps draw nothing.")
			})
		})
	}
}

// A stories file of a widget is a scene file: the gallery opens on its
// first story, under Components, with its name as the title, its tag
// and its sentence under it, and its children drawn as wide as the
// window leaves beside the stories. An edit of the file in a checkout
// is in the canvas, the story kept.
func TestScene(t *testing.T) {
	dir := checkout(t)
	g := New()
	if err := g.Watch(dir, 5*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	sel := "f-button-buttons-on-a-release-page"
	if g.Sel != sel || g.Shown()[0].Label != "Components" || g.Shown()[1].Label != "Button" {
		t.Fatalf("the gallery opens on %s, under %v", g.Sel, g.Shown()[:2])
	}
	a, _ := startOn(t, surftest.Start, g, Interpreted, sel)
	if v := a.Surface().TextOf(titleID); v != "Buttons on a release page" {
		t.Errorf("the title: %q", v)
	}
	if v := a.Surface().TextOf("r.kpage.kmain.kmid.kabout"); !strings.HasPrefix(v, "<button>  Every kind of button, with") {
		t.Errorf("the line under the title: %q", v)
	}
	if !strings.Contains(a.Surface().TextOf(storyID(sel)), "Deploy to production") {
		t.Errorf("the scene: %q", a.Surface().TextOf(storyID(sel)))
	}
	if want := 200 - framePad - sideCols - 1; g.Width != want {
		t.Errorf("the canvas is %d cells wide in a window of 200, not %d", g.Width, want)
	}
	a.Pick(barCtl("theme"), "paper")
	edit(t, dir, "internal/stories/button.surf", "Save as draft", "Keep as a draft")
	a.Wait("the scene's new label, in the story and the theme it had", func() bool {
		return strings.Contains(a.Surface().TextOf(storyID(sel)), "Keep as a draft") && a.Has(canvasID(sel), "u-theme-paper")
	})
}

// What can do nothing for the story is not drawn: a scene has no
// controls, no states and no cells, so the toolbar has none of their
// switches and there is no controls' column; a widget's story has its
// controls and its states, and no cells. Dense is not a density the
// canvas is offered.
func TestOnlyWhatActs(t *testing.T) {
	a, g := start(t, Interpreted, "f-button-buttons-on-a-release-page")
	shown := func(id string) bool { _, ok := a.Surface().Element(id); return ok }
	for _, key := range []string{"cells", "states", "reference", "controls"} {
		if shown(barCtl(key)) {
			t.Errorf("a scene's toolbar has %s", key)
		}
	}
	if shown(asideID) || !shown(barCtl("stories")) {
		t.Errorf("a scene: the controls' column shown %v, the Stories switch %v", shown(asideID), shown(barCtl("stories")))
	}
	a.Call(func() { g.Pick("w-button") })
	a.Wait("the button's controls and states", func() bool {
		return shown(asideID) && shown(barCtl("controls")) && shown(barCtl("states")) && !shown(barCtl("cells"))
	})
	if slices.Contains(g.Densities(), "dense") {
		t.Errorf("the canvas is offered %v", g.Densities())
	}
	a.Call(func() { g.SetDensity("dense") })
	if g.Density != "compact" {
		t.Errorf("the canvas took the density %s", g.Density)
	}
}
