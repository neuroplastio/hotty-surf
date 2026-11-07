package gallery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/neuroplastio/hotty-surf/internal/screens"
	"github.com/neuroplastio/hotty-surf/internal/stories"
	"github.com/neuroplastio/hotty-surf/surflive"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// checkout is a directory of the test's with the files a checkout has
// that the gallery reads: internal/stories/, internal/screens/ and
// surfui/surf.css.
func checkout(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.CopyFS(filepath.Join(dir, "internal", "stories"), stories.Files); err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(filepath.Join(dir, "internal", "screens"), screens.SurfPackage().Files); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "surfui"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "surfui", "surf.css"), []byte(surfui.CSS), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// edit replaces old with new in the checkout's file name.
func edit(t *testing.T, dir, name, old, new string) {
	t.Helper()
	path := filepath.Join(dir, name)
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), old) {
		t.Fatalf("%s has no %q", name, old)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(src), old, new, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Watching a checkout, an edit of a template, of a stories file or of
// the stylesheet reaches the canvas while the gallery runs, with the
// gallery's own state kept: the toolbar's theme, a control's value. An
// edit that doesn't compile keeps what the canvas had, and it says why.
// So on either host.
func TestReload(t *testing.T) {
	for _, mode := range modes {
		for _, h := range hosts {
			t.Run(mode.name+"/"+h.name, func(t *testing.T) { reload(t, h.start, mode.of) })
		}
	}
}

func reload(t *testing.T, start starter, of func(*Gallery) surflive.Component) {
	dir := checkout(t)
	g := New()
	if err := g.Watch(dir, 5*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	sel := "f-foot-hint-one-key"
	a, _ := startOn(t, start, g, of, sel)
	story, c := storyID(sel), canvasID(sel)
	a.Pick(barCtl("theme"), "paper")
	if err := a.Host.Fill(a.Name, controlID(sel, "title"), "Run it"); err != nil {
		t.Fatal(err)
	}
	a.Wait("the title typed", func() bool { return strings.Contains(a.Surface().TextOf(story), "Run it") })

	edit(t, dir, "internal/screens/foot-hint.surf", "<text>{{ title }}</text>", "<text>{{ title }}!</text>")
	a.Wait("the hint by its new template", func() bool { return strings.Contains(a.Surface().TextOf(story), "Run it!") })
	if !a.Has(c, "u-theme-paper") {
		t.Error("the reload lost the canvas's theme")
	}

	edit(t, dir, "internal/screens/foot-hint.surf", "<text>{{ title }}!</text>", "<text>{{ title( }}</text>")
	a.Wait("the reload's error", func() bool { return strings.Contains(a.Surface().TextOf(c+".kreload"), "foot-hint.surf") })
	if !strings.Contains(a.Surface().TextOf(story), "Run it!") {
		t.Errorf("the canvas lost its story: %q", a.Surface().TextOf(c))
	}
	edit(t, dir, "internal/screens/foot-hint.surf", "<text>{{ title( }}</text>", "<text>[{{ title }}]</text>")
	a.Wait("the hint mended", func() bool {
		_, failed := a.Surface().Element(c + ".kreload")
		return !failed && strings.Contains(a.Surface().TextOf(story), "[Run it]")
	})

	edit(t, dir, "internal/stories/foot-hint.surf", `title="Run" keys="enter"`, `title="Run" keys="space"`)
	a.Wait("the story's new keys", func() bool { v, _ := a.Surface().Value(controlID(sel, "keys")); return v == "space" })
	if v, _ := a.Surface().Value(controlID(sel, "title")); v != "Run it" {
		t.Errorf("the reload lost the title typed: %q", v)
	}

	css := filepath.Join(dir, "surfui", "surf.css")
	if err := os.WriteFile(css, []byte(surfui.CSS+"\n.u-gallery-mark:hover { color: red; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a.Wait("the new stylesheet, its forced rules beside", func() bool {
		_, src, _ := a.Host.Resource(surfui.StyleID)
		return strings.Contains(string(src), ".u-gallery-mark:hover") && strings.Contains(string(src), ".u-gallery-mark[data-force~=hover]")
	})
}

// The Reference switch shows the gate's renders of a story of
// stories.All under it, from the checkout's .shots/gate/: each sent as an
// image resource, shown as cid:, under the gate's verdict. Where the gate
// drew none, it says so. So on either host.
func TestReference(t *testing.T) {
	for _, h := range hosts {
		t.Run(h.name, func(t *testing.T) { reference(t, h.start) })
	}
}

func reference(t *testing.T, start starter) {
	dir := checkout(t)
	gate := filepath.Join(dir, ".shots", "gate")
	if err := os.MkdirAll(filepath.Join(gate, "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	id := "button-states-graphite-compact-2"
	report := "| [" + id + "](" + id + ".png) | 1/1 | 2.6% | 0.056% | 7.0 | pass |  |\n"
	if err := os.WriteFile(filepath.Join(gate, "report.md"), []byte(report), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"blitz", "chromium", "diff"} {
		if err := os.WriteFile(filepath.Join(gate, "work", id+"."+kind+".png"), []byte("\x89PNG "+kind), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	g := New()
	if err := g.Watch(dir, time.Hour); err != nil {
		t.Fatal(err)
	}
	sel := "s-button-states"
	a, _ := startOn(t, start, g, Interpreted, sel)
	if err := a.Host.Click(a.Name, barCtl("reference")); err != nil {
		t.Fatal(err)
	}
	ref := canvasID(sel) + ".kref"
	a.Wait("the renders", func() bool {
		for _, kind := range []string{"blitz", "chromium", "diff"} {
			_, data, _ := a.Host.Resource(refID(kind))
			src, _ := a.Surface().Attr(ref+".k"+kind+".1", "src")
			if string(data) != "\x89PNG "+kind || src != "cid:"+refID(kind) {
				return false
			}
		}
		return true
	})
	if v := a.Surface().TextOf(ref + ".khead"); !strings.Contains(v, "pass, 7.0 CSS px² differ") {
		t.Errorf("the verdict: %q", v)
	}
	a.Pick(barCtl("theme"), "paper")
	a.Wait("no renders in paper", func() bool {
		_, _, sent := a.Host.Resource(refID("blitz"))
		return !sent && strings.Contains(a.Surface().TextOf(ref+".knone"), "make gate")
	})
}
