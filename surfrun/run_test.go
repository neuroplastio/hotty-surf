package surfrun_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-go/hottyterm"
	"neuroplast.io/hotty-go/hottytest"

	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

// keyed is a surface with one binding, which counts its presses.
type keyed struct {
	key     string
	pressed int
}

func (m *keyed) Render() *surfui.Node {
	return surfui.Surface(surfui.Text("keys")).Bind(m.key, "Press", "press")
}

func (m *keyed) Do(act surfui.Action, ev surfrun.Event) {
	if act == "press" {
		m.pressed++
	}
}

// pressed opens m on a host with caps, presses key, and reports whether
// m's binding heard it.
func pressed(t *testing.T, m *keyed, caps hotty.Caps, key string) bool {
	t.Helper()
	return typed(t, m, caps, func(h *hottytest.Host) { h.Key(key) })
}

// typed is pressed with the keys sent by send.
func typed(t *testing.T, m *keyed, caps hotty.Caps, send func(h *hottytest.Host)) bool {
	t.Helper()
	h := hottytest.New(t, hottytest.Caps(caps))
	term := hottyterm.New(h, h, "t", h.TermSize, nil)
	r := surfrun.New(term, m, surfrun.Options{Name: "app"})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- r.Loop(ctx) }()
	defer func() { cancel(); _ = h.Close(); <-done }()
	name := term.Surface("app")
	for end := time.Now().Add(30 * time.Second); ; time.Sleep(5 * time.Millisecond) {
		if s := h.Surface(name); s != nil && s.Placed() {
			break
		}
		if time.Now().After(end) {
			t.Fatal("never placed")
		}
	}
	send(h)
	for end := time.Now().Add(300 * time.Millisecond); time.Now().Before(end); time.Sleep(5 * time.Millisecond) {
		var n int
		r.Call(func() { n = m.pressed })
		if n > 0 {
			return true
		}
	}
	return false
}

// A binding's mod is the host's (SPEC §10.4): ctrl unless it says super.
func TestModIsTheHosts(t *testing.T) {
	if !pressed(t, &keyed{key: "mod+r"}, hotty.Caps{}, "ctrl+r") {
		t.Error("mod+r did not hear ctrl+r where mod is ctrl")
	}
	if pressed(t, &keyed{key: "mod+r"}, hotty.Caps{Mod: "super"}, "ctrl+r") {
		t.Error("mod+r heard ctrl+r where mod is super")
	}
	if !pressed(t, &keyed{key: "ctrl+r"}, hotty.Caps{Mod: "super"}, "ctrl+r") {
		t.Error("ctrl+r did not hear ctrl+r")
	}
}

// The run asks for the kitty keyboard protocol's disambiguation, so that
// mod+enter reaches it as itself and not as enter.
func TestModifiedEnter(t *testing.T) {
	ctrlEnter := func(h *hottytest.Host) { h.Type("\x1b[13;5u") }
	if !typed(t, &keyed{key: "mod+enter"}, hotty.Caps{}, ctrlEnter) {
		t.Error("mod+enter did not hear ctrl+enter")
	}
	if typed(t, &keyed{key: "enter"}, hotty.Caps{}, ctrlEnter) {
		t.Error("enter heard ctrl+enter")
	}
}

// later draws its list, the keyboard's home, only once loaded.
type later struct {
	loaded bool
	moved  int
}

func (m *later) Render() *surfui.Node {
	if !m.loaded {
		return surfui.Surface(surfui.Text("Loading…"))
	}
	return surfui.Surface(surfui.Text("Agents"), surfui.List(surfui.Item(surfui.Text("a"))).Focusable().Autofocus().
		Bind("arrow_down", "Next", "next"))
}

func (m *later) Do(act surfui.Action, ev surfrun.Event) {
	switch act {
	case "load":
		m.loaded = true
	case "next":
		m.moved++
	}
}

// An autofocus element that comes after the show gets the keyboard, and
// is the keys' home.
func TestAutofocusLater(t *testing.T) {
	h := hottytest.New(t)
	term := hottyterm.New(h, h, "t", h.TermSize, nil)
	m := &later{}
	r := surfrun.New(term, m, surfrun.Options{Name: "app", Tick: "load", TickEvery: 50 * time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- r.Loop(ctx) }()
	defer func() { cancel(); _ = h.Close(); <-done }()
	name := term.Surface("app")
	until := func(what string, ok func() bool) {
		t.Helper()
		for end := time.Now().Add(30 * time.Second); !ok(); time.Sleep(5 * time.Millisecond) {
			if time.Now().After(end) {
				t.Fatal(what)
			}
		}
	}
	until("the list never had the keyboard", func() bool {
		s := h.Surface(name)
		return s != nil && s.Focused() == "r.1"
	})
	h.Key("arrow_down")
	until("the list's key never ran", func() bool {
		var n int
		r.Call(func() { n = m.moved })
		return n > 0
	})
}

// sized keeps the sizes it is told.
type sized struct{ sizes [][2]int }

func (m *sized) Render() *surfui.Node                   { return surfui.Surface(surfui.Text("rows")) }
func (m *sized) Do(act surfui.Action, ev surfrun.Event) {}
func (m *sized) Resize(cols, rows int)                  { m.sizes = append(m.sizes, [2]int{cols, rows}) }

// A Sizer hears the surface's size before its first render, and again
// when the terminal's changes, the header's rows taken off.
func TestSizerHearsTheSize(t *testing.T) {
	h := hottytest.New(t)
	term := hottyterm.New(h, h, "t", h.TermSize, nil)
	m := &sized{}
	r := surfrun.New(term, m, surfrun.Options{Name: "app", Top: 1})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- r.Loop(ctx) }()
	defer func() { cancel(); _ = h.Close(); <-done }()
	last := func() (s [2]int) {
		r.Call(func() {
			if len(m.sizes) > 0 {
				s = m.sizes[len(m.sizes)-1]
			}
		})
		return s
	}
	for end := time.Now().Add(30 * time.Second); last() != [2]int{80, 23}; time.Sleep(5 * time.Millisecond) {
		if time.Now().After(end) {
			t.Fatalf("first size %v", last())
		}
	}
	term.Resized(hottyterm.Size{Cols: 100, Rows: 30})
	for end := time.Now().Add(30 * time.Second); last() != [2]int{100, 29}; time.Sleep(5 * time.Millisecond) {
		if time.Now().After(end) {
			t.Fatalf("resized %v", last())
		}
	}
}

// The flag is pushed on the alternate screen and popped before leaving it.
func TestKeyboardProtocolPushedAndPopped(t *testing.T) {
	h := hottytest.New(t)
	term := hottyterm.New(h, h, "t", h.TermSize, nil)
	var out strings.Builder
	r := surfrun.New(term, &keyed{key: "q"}, surfrun.Options{Name: "app", Stream: &out})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- r.Loop(ctx) }()
	name := term.Surface("app")
	for end := time.Now().Add(30 * time.Second); ; time.Sleep(5 * time.Millisecond) {
		if s := h.Surface(name); s != nil && s.Placed() {
			break
		}
		if time.Now().After(end) {
			t.Fatal("never placed")
		}
	}
	cancel()
	<-done
	_ = h.Close()
	s := out.String()
	alt, push, pop, main := strings.Index(s, "\x1b[?1049h"), strings.Index(s, "\x1b[>1u"), strings.LastIndex(s, "\x1b[<u"), strings.LastIndex(s, "\x1b[?1049l")
	if !(alt >= 0 && alt < push && push < pop && pop < main) {
		t.Fatalf("alternate screen at %d, push at %d, pop at %d, main screen at %d", alt, push, pop, main)
	}
}

// plain is a surface of one text, with a cells UI.
type plain struct{}

func (plain) Render() *surfui.Node              { return surfui.Surface(surfui.Text("plain")) }
func (plain) Do(surfui.Action, surfrun.Event)   {}
func (plain) DrawCells(cols, rows int) []string { return []string{"plain"} }
func (plain) CellsKey(string) bool              { return false }

// written runs a plain surface with opts on a test host, or with cells on one that
// is no hotty host, and returns the runtime, what it has written so far
// (read on its goroutine), and stop, which ends the run and returns all
// it wrote.
func written(t *testing.T, opts surfrun.Options, cells bool) (r *surfrun.Runtime, sofar, stop func() string) {
	t.Helper()
	h := hottytest.New(t)
	var known *hottyterm.Known
	if cells {
		known = &hottyterm.Known{}
	}
	term := hottyterm.New(h, h, "t", h.TermSize, known)
	var out strings.Builder
	opts.Name, opts.Stream = "app", &out
	r = surfrun.New(term, plain{}, opts)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- r.Loop(ctx) }()
	sofar = func() string {
		var s string
		r.Call(func() { s = out.String() })
		return s
	}
	stop = func() string {
		cancel()
		<-done
		_ = h.Close()
		return out.String()
	}
	return r, sofar, stop
}

// The terminal's alternate scroll mode (DECSET 1007) as the runtime saves,
// turns off and restores it.
const (
	altSave    = "\x1b[?1007s"
	altOff     = "\x1b[?1007l"
	altRestore = "\x1b[?1007r"
)

// Where the document scrolls on a host, the terminal's alternate scroll
// mode is saved and turned off on the alternate screen, so that a wheel
// past what scrolls types no arrow keys, and restored after it. Swap, and
// the cells while a host fails, keep it in step: off while a document
// scrolls, as saved while none does. With no host, or in a run that
// doesn't scroll, it is left alone.
func TestAlternateScroll(t *testing.T) {
	r, sofar, stop := written(t, surfrun.Options{Scroll: hotty.ScrollVertical}, false)
	if s := sofar(); !strings.Contains(s, "\x1b[>1u"+altSave+altOff) {
		t.Errorf("the alternate screen's start is %q, want the mode saved and off", s[:min(len(s), 40)])
	}
	count := func(what string) int { return strings.Count(sofar(), what) }
	r.Call(func() { r.Swap(plain{}, nil, 0, 0) })
	if n := count(altRestore); n != 1 {
		t.Errorf("a swap to a document that doesn't scroll restored the mode %d times", n)
	}
	r.Call(func() { r.Swap(plain{}, nil, 0, hotty.ScrollVertical) })
	if s, o := count(altSave), count(altOff); s != 1 || o != 2 {
		t.Errorf("a swap back saved the mode %d times and turned it off %d, want once and twice", s, o)
	}
	r.Call(r.Fail)
	if n := count(altRestore); n != 2 {
		t.Errorf("in the cells the mode was restored %d times in all, want 2", n)
	}
	r.Call(r.Recover)
	if n := count(altOff); n != 3 {
		t.Errorf("back on the host the mode was turned off %d times in all, want 3", n)
	}
	s := stop()
	main, restore := strings.LastIndex(s, "\x1b[?1049l"), strings.LastIndex(s, altRestore)
	if main < 0 || restore < main {
		t.Errorf("the main screen at %d, the mode restored at %d: want it restored after", main, restore)
	}

	for _, c := range []struct {
		name  string
		opts  surfrun.Options
		cells bool
	}{
		{"a run that doesn't scroll", surfrun.Options{}, false},
		{"no host", surfrun.Options{Scroll: hotty.ScrollVertical}, true},
	} {
		_, _, stop := written(t, c.opts, c.cells)
		if s := stop(); strings.Contains(s, "\x1b[?1007") {
			t.Errorf("%s: the mode is written", c.name)
		}
	}
}

// quitter ends the run from a handler.
type quitter struct{ r surfrun.Runner }

func (m *quitter) Render() *surfui.Node {
	return surfui.Surface(surfui.Text("q")).Bind("q", "Quit", func() { m.r.Quit() })
}
func (m *quitter) Do(act surfui.Action, ev surfrun.Event) { act.(func())() }
func (m *quitter) Attach(r surfrun.Runner)                { m.r = r }

func TestQuitFromAHandler(t *testing.T) {
	h := hottytest.New(t)
	term := hottyterm.New(h, h, "t", h.TermSize, nil)
	r := surfrun.New(term, &quitter{}, surfrun.Options{Name: "app"})
	done := make(chan error, 1)
	go func() { done <- r.Loop(context.Background()) }()
	defer func() { _ = h.Close() }()
	for end := time.Now().Add(30 * time.Second); ; time.Sleep(5 * time.Millisecond) {
		if s := h.Surface(term.Surface("app")); s != nil && s.Placed() {
			break
		}
		if time.Now().After(end) {
			t.Fatal("never placed")
		}
	}
	h.Key("q")
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("the run did not end")
	}
}

// OnQuit decides whether the model's quit ends the run; ctrl+c ends it
// regardless.
func TestOnQuitKeepsTheRun(t *testing.T) {
	h := hottytest.New(t)
	term := hottyterm.New(h, h, "t", h.TermSize, nil)
	asked := 0
	r := surfrun.New(term, &quitter{}, surfrun.Options{Name: "app", OnQuit: func() bool { asked++; return false }})
	done := make(chan error, 1)
	go func() { done <- r.Loop(context.Background()) }()
	defer func() { _ = h.Close() }()
	for end := time.Now().Add(30 * time.Second); ; time.Sleep(5 * time.Millisecond) {
		if s := h.Surface(term.Surface("app")); s != nil && s.Placed() {
			break
		}
		if time.Now().After(end) {
			t.Fatal("never placed")
		}
	}
	h.Key("q")
	for end := time.Now().Add(30 * time.Second); ; time.Sleep(5 * time.Millisecond) {
		var n int
		r.Call(func() { n = asked })
		if n == 1 {
			break
		}
		if time.Now().After(end) {
			t.Fatal("OnQuit was not asked")
		}
	}
	select {
	case <-done:
		t.Fatal("the run ended though OnQuit said no")
	case <-time.After(100 * time.Millisecond):
	}
	h.Key("ctrl+c")
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("ctrl+c did not end the run")
	}
}

// picker is a <select> of the host's (surfui.El: surfui.Select is the
// runtime's), whose value the model also sets: drawn with it, or (bound)
// left as drawn first with its value set as a signal binding sets it.
type picker struct {
	value string
	bound bool
}

// hostSelect is a <select> as it is, value picked.
func hostSelect(value string, options ...string) *surfui.Node {
	s := surfui.El("select")
	for _, o := range options {
		opt := surfui.El("option", surfui.Str(o)).Attr("value", o)
		if o == value {
			opt.Attr("selected", "")
		}
		s.Add(opt)
	}
	return s
}

func (m *picker) Render() *surfui.Node {
	if m.bound {
		return surfui.Surface(hostSelect("one", "one", "two", "three").Attr("value", m.value).OnChange("pick"))
	}
	return surfui.Surface(hostSelect(m.value, "one", "two", "three").OnChange("pick"))
}

func (m *picker) Do(act surfui.Action, ev surfrun.Event) {
	if act == "pick" {
		m.value = ev.Value
	}
}

// A select's value the model changes goes as a=field, after the user's
// pick as before it (SPEC §6.4): a select the user has picked in no
// longer follows its options' selected.
func TestSelectValueFromTheModel(t *testing.T) {
	for _, bound := range []bool{false, true} {
		selectValue(t, &picker{value: "one", bound: bound}, true)
	}
}

// The model's first change goes too, before the user has picked or typed
// in the control: the value the document gave it is the one sent.
func TestControlValueFromTheModelFirst(t *testing.T) {
	for _, bound := range []bool{false, true} {
		selectValue(t, &picker{value: "one", bound: bound}, false)
	}
}

func selectValue(t *testing.T, m *picker, pick bool) {
	h := hottytest.New(t)
	term := hottyterm.New(h, h, "t", h.TermSize, nil)
	r := surfrun.New(term, m, surfrun.Options{Name: "app"})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- r.Loop(ctx) }()
	defer func() { cancel(); _ = h.Close(); <-done }()
	name := term.Surface("app")
	wait := func(what string, ok func() bool) {
		t.Helper()
		for end := time.Now().Add(30 * time.Second); !ok(); time.Sleep(5 * time.Millisecond) {
			if time.Now().After(end) {
				t.Fatal(what)
			}
		}
	}
	wait("never placed", func() bool { s := h.Surface(name); return s != nil && s.Placed() })
	const id = "r.0"
	if pick {
		if err := h.Choose(name, id, "two"); err != nil {
			t.Fatal(err)
		}
		wait("the pick never reached the model", func() bool {
			var v string
			r.Call(func() { v = m.value })
			return v == "two"
		})
	}
	r.Post(func() { m.value = "three" })
	wait("the model's value never reached the select", func() bool {
		v, _ := h.Surface(name).Value(id)
		return v == "three"
	})
	var fields int
	for _, c := range h.Commands() {
		if hotty.Get(c.Control, "a") == "field" && hotty.Get(c.Control, "t") == id {
			fields++
		}
	}
	if fields != 1 {
		t.Errorf("bound %v: %d a=field for the select, want 1", m.bound, fields)
	}
}

// compactRoot is a model whose surface is Compact.
type compactRoot struct{}

func (compactRoot) Render() *surfui.Node {
	return surfui.Surface(surfui.Text("rows")).Density(surfui.Compact)
}

func (compactRoot) Do(surfui.Action, surfrun.Event) {}

// TestDensityOption checks that Options.Density draws the root at its
// density, whatever the model's (S11's pair).
func TestDensityOption(t *testing.T) {
	for _, c := range []struct {
		opt       surfui.Density
		want, not string
	}{
		{"", "u-compact", "u-dense"},
		{surfui.Dense, "u-dense", "u-compact"},
	} {
		h := hottytest.New(t)
		term := hottyterm.New(h, h, "t", h.TermSize, nil)
		r := surfrun.New(term, compactRoot{}, surfrun.Options{Name: "app", Density: c.opt})
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- r.Loop(ctx) }()
		name := term.Surface("app")
		for end := time.Now().Add(30 * time.Second); ; time.Sleep(5 * time.Millisecond) {
			if s := h.Surface(name); s != nil && s.Placed() {
				break
			}
			if time.Now().After(end) {
				t.Fatal("never placed")
			}
		}
		html := h.Surface(name).HTML()
		cancel()
		_ = h.Close()
		<-done
		if !strings.Contains(html, c.want) || strings.Contains(html, c.not) {
			t.Errorf("Density %q: want %s and no %s in %s", c.opt, c.want, c.not, html)
		}
	}
}

// twoButtons has a button with a handler and one with none. It keeps its
// runner, as a tool does that sends a stylesheet or a resource through
// it.
type twoButtons struct {
	clicks int
	rn     surfrun.Runner
}

func (m *twoButtons) Render() *surfui.Node {
	return surfui.Surface(surfui.Button("Bare").Key("bare"), surfui.Button("Counts").Key("counts").OnClick("count"))
}

func (m *twoButtons) Do(act surfui.Action, ev surfrun.Event) { m.clicks++ }

func (m *twoButtons) Attach(r surfrun.Runner) { m.rn = r }

// sender is a runner that sends a stylesheet and resources: the runtime,
// and surftest's host of its own.
type sender interface {
	SetCSS(css string)
	SetResource(id, mime string, data []byte)
}

// On either host, Options.CSS is the stylesheet sent, and SetCSS sends
// another in its place; Options.Unhandled hears an event on an element
// with no handler, and not one with.
func TestStylesheetAndUnhandled(t *testing.T) {
	for _, h := range hosts {
		t.Run(h.name, func(t *testing.T) {
			m := &twoButtons{}
			var heard []surfrun.Event
			a := h.start(t, m, surfrun.Options{CSS: surfui.CSS + ".mark {}", Unhandled: func(ev surfrun.Event) { heard = append(heard, ev) }})
			css := func() string { _, b, _ := a.Host.Resource(surfui.StyleID); return string(b) }
			if !strings.HasSuffix(css(), ".mark {}") {
				t.Errorf("the stylesheet sent is not Options.CSS: %.80q…", css())
			}
			a.Call(func() { m.rn.(sender).SetCSS(".other {}") })
			a.Wait("the stylesheet sent again", func() bool { return css() == ".other {}" })

			if err := a.Host.Click(a.Name, "r.kcounts"); err != nil {
				t.Fatal(err)
			}
			if err := a.Host.Click(a.Name, "r.kbare"); err != nil {
				t.Fatal(err)
			}
			// Each button's focus is unhandled too: neither has a focus handler.
			var clicked []surfrun.Event
			a.Wait("the bare button's click heard", func() bool {
				a.Call(func() {
					clicked = clicked[:0]
					for _, ev := range heard {
						if ev.Kind == "click" {
							clicked = append(clicked, ev)
						}
					}
				})
				return len(clicked) > 0
			})
			var clicks int
			a.Call(func() { clicks = m.clicks })
			if len(clicked) != 1 || clicked[0].Target != "r.kbare" || clicks != 1 {
				t.Errorf("unhandled clicks %+v, handled %d: want the bare button's unhandled, the other's handled", clicked, clicks)
			}
		})
	}
}

// On either host, a resource the program sends is the host's at once;
// nil deletes it.
func TestResource(t *testing.T) {
	for _, h := range hosts {
		t.Run(h.name, func(t *testing.T) {
			m := &twoButtons{}
			a := h.start(t, m, surfrun.Options{})
			pic := func() string { mime, b, ok := a.Host.Resource("pic"); return fmt.Sprint(mime, " ", string(b), " ", ok) }
			send := func(data []byte) { a.Call(func() { m.rn.(sender).SetResource("pic", "image/png", data) }) }
			send([]byte("one"))
			a.Wait("the resource", func() bool { return pic() == "image/png one true" })
			send([]byte("two"))
			a.Wait("the resource again", func() bool { return pic() == "image/png two true" })
			send(nil)
			a.Wait("the resource deleted", func() bool { return pic() == "  false" })
		})
	}
}
