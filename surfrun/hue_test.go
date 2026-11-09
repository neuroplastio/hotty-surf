package surfrun_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"neuroplast.io/hotty-go/hottyterm"
	"neuroplast.io/hotty-go/hottytest"

	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surftest"
	"neuroplast.io/hotty-surf/surfui"
)

// hued is a surface of a dot and a text in the hue its colour names; a key
// changes the colour, and the surface is drawn again.
type hued struct {
	rn     surfrun.Runner
	colour string
	next   []string
}

func (m *hued) Render() *surfui.Node {
	return surfui.Surface(surfui.Row(surfui.Dot().Hue(m.colour), surfui.Text("name").Hue(m.colour)).Gap(1)).
		Bind("n", "Next", "next")
}

func (m *hued) Do(act surfui.Action, ev surfrun.Event) {
	if act == "next" && len(m.next) > 0 {
		m.colour, m.next = m.next[0], m.next[1:]
	}
}

func (m *hued) Attach(r surfrun.Runner) { m.rn = r }

type palSender interface{ SetPalette(surfui.Palette) }

// warnings collects a run's warnings.
type warnings struct {
	mu   sync.Mutex
	msgs []string
}

func (w *warnings) add(msg string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.msgs = append(w.msgs, msg)
}

func (w *warnings) all() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]string(nil), w.msgs...)
}

var (
	coral = surfui.Hue{Dark: "#ff8e7a", Light: "#a42d1c", Cells: surfui.Cell(9)}
	teal  = surfui.Hue{Dark: "#4fd1c5", Light: "#0d645d"}
	// faint's dark value is 3.7:1 on the dark ground: a dot, not text.
	faint = surfui.Hue{Dark: "#7a8499", Light: "#a42d1c"}
)

// On either host, Options.Palette puts each hue's rules after the
// stylesheet, and SetPalette declares them again: the stylesheet is sent
// in place of the last, the hues it replaced gone, and the stylesheet's
// own kept.
func TestPaletteSentAndReplaced(t *testing.T) {
	for _, h := range hosts {
		t.Run(h.name, func(t *testing.T) {
			m := &hued{colour: "coral"}
			a := h.start(t, m, surfrun.Options{CSS: surfui.CSS + ".mark {}", Palette: surfui.Palette{"coral": coral}, Warn: func(string) {}})
			css := func() string { _, b, _ := a.Host.Resource(surfui.StyleID); return string(b) }
			for _, want := range []string{".mark {}", "--u-hue-coral: #ff8e7a;", ".u-hue-coral { color: var(--u-hue-coral); }"} {
				if !strings.Contains(css(), want) {
					t.Errorf("the first stylesheet lacks %q", want)
				}
			}
			if !strings.HasPrefix(css(), surfui.CSS) {
				t.Error("the palette's rules don't follow the stylesheet")
			}
			a.Call(func() { m.rn.(palSender).SetPalette(surfui.Palette{"teal": teal}) })
			a.Wait("the palette sent again", func() bool { return strings.Contains(css(), "--u-hue-teal: #4fd1c5;") })
			if strings.Contains(css(), "coral") || !strings.Contains(css(), ".mark {}") {
				t.Errorf("declared again, the set is replaced and the stylesheet kept:\n%s", css()[len(css())-300:])
			}
			a.Call(func() { m.rn.(palSender).SetPalette(nil) })
			a.Wait("the palette taken away", func() bool { return !strings.Contains(css(), "teal") })
			if css() != surfui.CSS+".mark {}" {
				t.Errorf("no hues, no rules")
			}
		})
	}
}

// A palette that is declared again is sent again, though it is the same.
func TestPaletteSentAgain(t *testing.T) {
	var out syncBuf
	var w warnings
	m := &hued{colour: "coral"}
	a := surftest.Start(t, m, surfrun.Options{Palette: surfui.Palette{"coral": coral}, Stream: &out, Warn: w.add})
	sends := func() int { return strings.Count(out.String(), surfui.StyleID) }
	n := sends()
	if n == 0 {
		t.Fatal("the first stylesheet is never sent")
	}
	for i := 1; i <= 2; i++ {
		a.Call(func() { a.Run.SetPalette(surfui.Palette{"coral": coral}) })
		if got := sends(); got != n+i {
			t.Errorf("declared the same %d times: the stylesheet was sent %d times in all, want %d", i, got, n+i)
		}
	}
	if got := w.all(); len(got) != 0 {
		t.Errorf("warnings %q", got)
	}
}

// syncBuf is a buffer a run writes while a test reads.
type syncBuf struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// A hue whose text can't be read warns once, however often it is declared
// and the surface drawn; its text is the ink on the failing ground and its
// dot keeps the hue; a hue declared again with another colour warns anew.
func TestPaletteWarnsOnce(t *testing.T) {
	for _, h := range hosts {
		t.Run(h.name, func(t *testing.T) {
			var w warnings
			m := &hued{colour: "faint", next: []string{"faint", "faint"}}
			a := h.start(t, m, surfrun.Options{Palette: surfui.Palette{"faint": faint, "coral": coral}, Warn: w.add})
			css := func() string { _, b, _ := a.Host.Resource(surfui.StyleID); return string(b) }
			want := "hue faint's dark #7a8499 is 3.7:1 on the dark ground; text needs 4.5; its text draws in the ink there, its dots keep the hue"
			if got := w.all(); len(got) != 1 || got[0] != want {
				t.Fatalf("warnings %q, want %q", got, want)
			}
			for _, c := range []string{
				"  --u-hue-faint: #7a8499;\n  --u-hue-faint-ink: var(--u-ink);\n",
				"    --u-hue-faint: #a42d1c;\n    --u-hue-faint-ink: var(--u-hue-faint);\n",
				".u-hue-faint { color: var(--u-hue-faint-ink); }",
				".u-dot.u-hue-faint { --u-dw: 8px; color: var(--u-hue-faint); }",
				".u-hue-coral { color: var(--u-hue-coral); }",
			} {
				if !strings.Contains(css(), c) {
					t.Errorf("the stylesheet lacks %q", c)
				}
			}
			for range 2 {
				a.Host.Key("n")
				a.Call(func() { m.rn.(palSender).SetPalette(surfui.Palette{"faint": faint, "coral": coral}) })
			}
			if got := w.all(); len(got) != 1 {
				t.Errorf("declared and drawn again, warnings %q", got)
			}
			other := surfui.Hue{Dark: "#778099", Light: "#a42d1c"}
			a.Call(func() { m.rn.(palSender).SetPalette(surfui.Palette{"faint": other}) })
			if got := w.all(); len(got) != 2 || !strings.Contains(got[1], "faint's dark #778099") {
				t.Errorf("another colour, warnings %q", got)
			}
			// A hue that is no hue is warned of, and has no rule.
			a.Call(func() { m.rn.(palSender).SetPalette(surfui.Palette{"Bad": {Value: "red"}, "coral": coral}) })
			a.Wait("the stylesheet without it", func() bool {
				return strings.Contains(css(), ".u-hue-coral") && !strings.Contains(css(), "--u-hue-faint")
			})
			// The warnings can come after the stylesheet: wait for the
			// fourth, then look at them all.
			a.Wait("the hue's warnings", func() bool { return len(w.all()) >= 4 })
			if got := w.all(); len(got) != 4 || !strings.Contains(got[2], "lower-case identifier") || !strings.Contains(got[3], `"faint" is in no palette`) {
				t.Errorf("a hue that is none, warnings %q", got)
			}
			if strings.Contains(css(), "-hue-Bad") {
				t.Error("a hue that is none is in the stylesheet")
			}
		})
	}
}

// A name a render uses that no palette declares draws the ink, and is
// warned of once for each name, not for each render; the model's own
// palette counts as declared.
func TestUnknownHueWarnsOncePerName(t *testing.T) {
	for _, h := range hosts {
		t.Run(h.name, func(t *testing.T) {
			var w warnings
			m := &hued{colour: "mauve", next: []string{"mauve", "coral", "ochre", "mauve"}}
			a := h.start(t, m, surfrun.Options{Palette: surfui.Palette{"coral": coral}, Warn: w.add})
			for _, colour := range m.next {
				a.Host.Key("n")
				a.Wait("the colour "+colour, func() bool {
					var got string
					a.Call(func() { got = m.colour })
					return got == colour
				})
				a.Call(func() {}) // its update
			}
			want := []string{
				`hue "mauve" is in no palette: its text draws in the ink`,
				`hue "ochre" is in no palette: its text draws in the ink`,
			}
			if got := w.all(); strings.Join(got, "\n") != strings.Join(want, "\n") {
				t.Errorf("warnings %q, want %q", got, want)
			}
		})
	}
}

// In the cells a hue is a terminal colour of its palette's choosing, its
// own cells index or the nearest by angle, and declared again, the next
// frame draws the new one; a name no palette declares is the ink.
func TestHuesInCells(t *testing.T) {
	m := &hued{colour: "coral", next: []string{"teal", "mauve"}}
	var out syncBuf
	var w warnings
	h := hottytest.New(t, hottytest.Text(), hottytest.Size(40, 6))
	term := hottyterm.New(h, h, "t", h.TermSize, &hottyterm.Known{})
	r := surfrun.New(term, m, surfrun.Options{Name: "app", Stream: &out, Warn: w.add, Palette: surfui.Palette{"coral": coral, "teal": teal}})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- r.Loop(ctx) }()
	t.Cleanup(func() { cancel(); <-done; _ = h.Close() })
	waitScreen(t, h, "name")
	has := func(sgr string) bool { return strings.Contains(out.String(), sgr+"●") }
	if !has("\x1b[0;91m") {
		t.Errorf("coral (cells 9) isn't bright red: %q", out.String())
	}
	next := func(colour string) {
		h.Type("n")
		for end := time.Now().Add(30 * time.Second); time.Now().Before(end); time.Sleep(time.Millisecond) {
			var got string
			r.Call(func() { got = m.colour })
			if got == colour {
				break
			}
		}
		r.Call(func() {})
	}
	next("teal")
	if !has("\x1b[0;96m") {
		t.Errorf("teal (nearest by angle: cyan) isn't bright cyan: %q", out.String())
	}
	// Declared again: coral is now blue, and the colour of a hue that is
	// there changes with no other change on the screen.
	r.Call(func() {
		m.colour = "coral"
		r.SetPalette(surfui.Palette{"coral": {Dark: "#ff8e7a", Light: "#a42d1c", Cells: surfui.Cell(4)}})
	})
	r.Call(func() {})
	if !has("\x1b[0;34m") {
		t.Errorf("coral declared again with cells 4 isn't blue: %q", out.String())
	}
	// Unknown: the ink, and one warning.
	r.Call(func() { m.next = []string{"mauve"} })
	next("mauve")
	if got := w.all(); len(got) != 1 || !strings.Contains(got[0], `"mauve"`) {
		t.Errorf("warnings %q", got)
	}
}
