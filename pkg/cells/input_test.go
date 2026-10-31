package cells_test

import (
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/neuroplastio/hotty-go"

	"github.com/neuroplastio/hotty-surf/pkg/cells"
	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// form is a screen of controls: a field that commits, a button, a
// Select, a Cycle and a List the keys move.
type form struct {
	model
	name, level, theme string
	sel                int
}

func newForm() *form {
	f := &form{name: "bob", level: "info", theme: "dark"}
	f.render = func() *surfui.Node {
		items := []*surfui.Node{}
		for i, s := range []string{"one", "two", "three"} {
			items = append(items, surfui.Item(surfui.Text(s)).Key(s).Selected(i == f.sel))
		}
		return surfui.Surface(
			surfui.Row(surfui.Text("Name"), surfui.Field().Value(f.name).OnChange(func(e surfrun.Event) { f.name = e.Value }).Grow()).Gap(1),
			surfui.Row(surfui.Button("Save").OnClick(func() {}),
				surfui.Select(f.level, "debug", "info", "warn").OnChange(func(e surfrun.Event) { f.level = e.Value }),
				surfui.Cycle(f.theme, "dark", "light").OnChange(func(e surfrun.Event) { f.theme = e.Value })).Gap(1),
			surfui.List(items...).Focusable().
				Bind("arrow_down", "next", func() { f.sel = min(f.sel+1, 2) }).
				Bind("arrow_up", "previous", func() { f.sel = max(f.sel-1, 0) }),
		)
	}
	return f
}

// press clicks the element id where the screen shows it.
func press(t *testing.T, r *cells.Renderer, id string) {
	t.Helper()
	x, y, ok := r.Where(id)
	if !ok {
		t.Fatalf("%s is not on the screen:\n%s", id, r.Draw(40, 10).Golden())
	}
	r.Press(x, y, 0)
	r.Release(x, y, 0)
	r.Draw(40, 10)
}

func keys(r *cells.Renderer, names ...string) {
	for _, k := range names {
		r.Key(k)
		r.Draw(40, 10)
	}
}

// Tab walks what takes the keyboard in tree order and leaves the surface
// past the last; Shift+Tab walks back.
func TestTab(t *testing.T) {
	f := newForm()
	r := show(f)
	r.Draw(40, 10)
	var order []string
	for range 6 {
		keys(r, "tab")
		order = append(order, r.Focused())
	}
	want := []string{"r.0.1", "r.1.0", "r.1.1", "r.1.2", "r.2", ""}
	if !slices.Equal(order, want) {
		t.Errorf("tab: %q, want %q", order, want)
	}
	keys(r, "shift+tab")
	if r.Focused() != "r.2" {
		t.Errorf("shift+tab from nothing: %q", r.Focused())
	}
}

// A field edits as a host's: typing reports input, Enter commits, and
// leaving the field commits what was typed since, once.
func TestField(t *testing.T) {
	f := newForm()
	r := show(f)
	r.Draw(40, 10)
	press(t, r, "r.0.1")
	keys(r, "backspace", "backspace", "backspace", "a", "shift+l", "space", "x", "arrow_left", "delete", "enter")
	if f.name != "aL " {
		t.Errorf("enter committed %q, want %q (log %q)", f.name, "aL ", f.log)
	}
	s := r.Draw(40, 10)
	if !s.Caret.Visible || s.Caret.Y != 0 || s.Caret.X != 6+3 { // after "Name", a gap, the field's padding
		t.Errorf("the caret: %+v", s.Caret)
	}
	keys(r, "ctrl+a")
	r.Paste("pasted\nline")
	keys(r, "tab")
	if f.name != "pastedline" {
		t.Errorf("leaving committed %q (log %q)", f.name, f.log)
	}
	n := len(f.log)
	keys(r, "shift+tab", "tab")
	if len(f.log) != n {
		t.Errorf("leaving again committed again: %q", f.log[n:])
	}
	if s := r.Draw(40, 10); s.Caret.Visible {
		t.Errorf("a caret off the field: %+v", s.Caret)
	}
}

// A press focuses what takes the keyboard, and its release clicks what
// reports clicks; a press elsewhere takes the keyboard from the surface.
func TestClick(t *testing.T) {
	f := newForm()
	r := show(f)
	r.Draw(40, 10)
	press(t, r, "r.1.0")
	if r.Focused() != "r.1.0" || !slices.Contains(f.log, "click r.1.0") {
		t.Errorf("a click on Save: focus %q, log %q", r.Focused(), f.log)
	}
	press(t, r, "r.1.2.2")
	if f.theme != "light" || r.Focused() != "r.1.2" {
		t.Errorf("a click on the cycle's ›: %q, focus %q", f.theme, r.Focused())
	}
	press(t, r, "r.0.0") // the label, which takes nothing
	if r.Focused() != "" {
		t.Errorf("a press on a label leaves focus on %q", r.Focused())
	}
}

// A release clicks what was pressed, though the press drew the tree
// anew: the element is the one with its id.
func TestClickDrawn(t *testing.T) {
	f := newForm()
	r := show(f)
	r.Draw(40, 10)
	x, y, _ := r.Where("r.1.0")
	r.Press(x, y, 0)
	f.name = "ann" // the model changes, and the tree is drawn anew
	r.Update()
	r.Draw(40, 10)
	r.Release(x, y, 0)
	if !slices.Contains(f.log, "click r.1.0") {
		t.Errorf("no click after a new tree: log %q", f.log)
	}
}

// A Select's list opens over the screen under it, and a press on an
// option picks it; the keys move it while it is open.
func TestSelect(t *testing.T) {
	f := newForm()
	r := show(f)
	r.Draw(40, 10)
	press(t, r, "r.1.1")
	s := r.Draw(40, 10)
	if !strings.Contains(s.Text(), "debug") || !strings.Contains(s.Text(), "╭") {
		t.Fatalf("no list:\n%s", s.Golden())
	}
	press(t, r, surfui.OptionID(2))
	if f.level != "warn" || r.Focused() != "r.1.1" {
		t.Errorf("picked %q, focus %q", f.level, r.Focused())
	}
	if s := r.Draw(40, 10); strings.Contains(s.Text(), "debug") {
		t.Errorf("the list stayed open:\n%s", s.Text())
	}
	keys(r, "space", "arrow_up", "arrow_up", "enter")
	if f.level != "debug" {
		t.Errorf("the keys picked %q", f.level)
	}
}

// A key the focused element binds is the program's; one that nothing
// binds and nothing scrolls with is too.
func TestBound(t *testing.T) {
	f := newForm()
	r := show(f)
	r.Draw(40, 10)
	press(t, r, "r.2.ktwo")
	keys(r, "arrow_down")
	if f.sel != 1 {
		t.Errorf("arrow_down on the list: sel %d (log %q)", f.sel, f.log)
	}
	if s := r.Draw(40, 10); !strings.Contains(s.Golden(), "fg/sel") {
		t.Errorf("the selection of the list with the keyboard is not on sel:\n%s", s.Golden())
	}
	f.log = nil
	keys(r, "x")
	if len(f.log) != 0 {
		t.Errorf("an unbound key ran %q", f.log)
	}
	if took, _ := r.Key("x"); took {
		t.Error("an unbound key was taken")
	}
}

// Where the document scrolls, the wheel and the scrolling keys move a
// Scroll box, and what takes the keyboard comes into view.
func TestScroll(t *testing.T) {
	m := &model{}
	m.render = func() *surfui.Node {
		var rows []*surfui.Node
		for i := range 20 {
			rows = append(rows, surfui.Button("b"+strconv.Itoa(i)).OnClick(func() {}))
		}
		return surfui.Surface(surfui.Text("top"), surfui.Stack(rows...).Scroll(), surfui.Text("foot"))
	}
	r := cells.New(surfrun.NewSurface(m, nil).Cells())
	r.Scroll = hotty.ScrollVertical
	r.Show()
	first := func() string { return strings.Split(r.Draw(20, 8).Text(), "\n")[1] }
	if got := first(); !strings.Contains(got, "b0") {
		t.Fatalf("first row %q", got)
	}
	r.Wheel(5, 3, 1)
	if got := first(); !strings.Contains(got, "b3") {
		t.Errorf("the wheel down a notch: first row %q", got)
	}
	r.Wheel(5, 3, -5)
	if got := first(); !strings.Contains(got, "b0") {
		t.Errorf("the wheel back: first row %q", got)
	}
	for range 9 { // the box, which takes the keyboard too, then b0 to b7
		r.Key("tab")
		r.Draw(20, 8)
	}
	if got := r.Draw(20, 8).Text(); !strings.Contains(got, "b7") {
		t.Errorf("tab to b7: not in view\n%s", got)
	}
	r.Key("end")
	if got := r.Draw(20, 8).Text(); !strings.Contains(got, "b19") {
		t.Errorf("end, on a button: not at the end\n%s", got)
	}
}

// Keys lists what Key takes: in each state of the form, a key Key takes
// is listed, the renderer's own or the program's, or is a character the
// focused element takes; and a key it doesn't take is in neither. Each
// key listed is tried.
func TestKeys(t *testing.T) {
	states := []struct {
		name string
		to   []string
	}{
		{"nothing focused", nil},
		{"the field", []string{"tab"}},
		{"the field, all selected", []string{"tab", "ctrl+a"}},
		{"the button", []string{"tab", "tab"}},
		{"the Select", []string{"tab", "tab", "tab"}},
		{"the Cycle", []string{"tab", "tab", "tab", "tab"}},
		{"the list", []string{"tab", "tab", "tab", "tab", "tab"}},
	}
	names := []string{"tab", "shift+tab", "space", "shift+space", "enter", "shift+enter", "escape",
		"arrow_left", "arrow_right", "arrow_up", "arrow_down", "shift+arrow_left", "shift+arrow_down",
		"home", "end", "page_down", "backspace", "delete", "ctrl+a", "ctrl+c", "ctrl+x", "ctrl+v", "ctrl+z", "x", "shift+x"}
	for _, st := range states {
		start := func() *cells.Renderer {
			r := show(newForm())
			r.Draw(40, 10)
			keys(r, st.to...)
			return r
		}
		list, chars := start().Keys()
		all := slices.Clone(names)
		for _, u := range list {
			all = append(all, u.Name)
		}
		for _, k := range all {
			took, _ := start().Key(k)
			want := slices.ContainsFunc(list, func(u cells.KeyUse) bool { return u.Name == k }) ||
				typesChar(k) && (chars == cells.CharsText || chars == cells.CharsPick)
			if took != want {
				t.Errorf("%s: Key(%q) took %v; Keys %s, %v", st.name, k, took, listed(list), chars)
			}
		}
	}
}

// Keys in order: the renderer's own first, each key once, then the
// program's.
func TestKeysInOrder(t *testing.T) {
	for _, c := range []struct {
		name string
		to   []string
		want string
	}{
		{"the button", []string{"tab", "tab"}, "tab* shift+tab* space* enter* shift+space* shift+enter*"},
		{"the field", []string{"tab"}, "tab* shift+tab* arrow_left* arrow_right*"},
	} {
		r := show(newForm())
		r.Draw(40, 10)
		keys(r, c.to...)
		got, _ := r.Keys()
		if s := listed(got); !strings.HasPrefix(s, c.want) {
			t.Errorf("%s: %s, want it to start %s", c.name, s, c.want)
		}
		seen := map[string]bool{}
		for _, u := range got {
			if seen[u.Name] {
				t.Errorf("%s: %s twice in %s", c.name, u.Name, listed(got))
			}
			seen[u.Name] = true
		}
	}
	r := show(newForm())
	r.Draw(40, 10)
	keys(r, "tab")
	got, chars := r.Keys()
	s := " " + listed(got) + " "
	for _, k := range []string{"shift+arrow_left*", "ctrl+a*", "ctrl+z*", "space*", "shift+space*"} {
		if !strings.Contains(s, " "+k+" ") {
			t.Errorf("the field: no %s in %s", k, s)
		}
	}
	for _, k := range []string{"ctrl+c", "ctrl+v", "ctrl+x", "enter"} {
		if strings.Contains(s, " "+k+"* ") {
			t.Errorf("the field: %s is the program's, but the field's in %s", k, s)
		}
	}
	if chars != cells.CharsText {
		t.Errorf("the field: chars %v", chars)
	}
	keys(r, "ctrl+a")
	if got, _ := r.Keys(); !strings.Contains(" "+listed(got)+" ", " ctrl+x* ") {
		t.Errorf("the field, all selected: no ctrl+x* in %s", listed(got))
	}
}

// The scrolling keys are the renderer's while a view from the focused
// element outward has more than it shows, and only where the document
// scrolls.
func TestKeysScroll(t *testing.T) {
	m := &model{}
	m.render = func() *surfui.Node {
		var rows []*surfui.Node
		for i := range 20 {
			rows = append(rows, surfui.Button("b"+strconv.Itoa(i)).OnClick(func() {}))
		}
		return surfui.Surface(surfui.Text("top"), surfui.Stack(rows...).Scroll(), surfui.Text("foot"))
	}
	for _, scroll := range []hotty.Axes{0, hotty.ScrollVertical} {
		r := cells.New(surfrun.NewSurface(m, nil).Cells())
		r.Scroll = scroll
		r.Show()
		r.Draw(20, 8)
		r.Key("tab")
		r.Key("tab") // the box, then b0
		r.Draw(20, 8)
		got, _ := r.Keys()
		s := " " + listed(got) + " "
		for _, k := range []string{"arrow_down*", "page_down*", "home*", "end*"} {
			if has := strings.Contains(s, " "+k+" "); has != (scroll != 0) {
				t.Errorf("scroll %v: %s listed %v in %s", scroll, k, has, s)
			}
		}
	}
}

// typesChar reports whether key types a character, with Shift at most,
// as Chars has it: "a", "shift+a", ">", never a space.
func typesChar(key string) bool {
	key = strings.TrimPrefix(key, "shift+")
	r, n := utf8.DecodeRuneInString(key)
	return n > 0 && n == len(key) && unicode.IsPrint(r) && !unicode.IsSpace(r)
}

// listed writes keys as "name" for a binding, "name:Title" for one with a
// title, and "name*" for an Own key.
func listed(keys []cells.KeyUse) string {
	var out []string
	for _, u := range keys {
		switch {
		case u.Own:
			out = append(out, u.Name+"*")
		case u.Title != "":
			out = append(out, u.Name+":"+u.Title)
		default:
			out = append(out, u.Name)
		}
	}
	return strings.Join(out, " ")
}

// Copy and cut give a field's selected text to Copy, by its characters
// (grapheme clusters), as a host's field puts it on the clipboard, cut
// then deleting it. With nothing selected, in a password field, or with
// no Copy, the keys are the program's as before, and cut with no Copy
// still deletes.
func TestCopy(t *testing.T) {
	text, pw := "añ👍🏽 b", "hunter2"
	m := &model{}
	m.render = func() *surfui.Node {
		return surfui.Surface(
			surfui.Field().Value(text).Key("text").Autofocus().OnChange(func(e surfrun.Event) { text = e.Value }),
			surfui.Field().Attr("type", "password").Value(pw).Key("pw").OnChange(func(e surfrun.Event) { pw = e.Value }),
		)
	}
	var clip []string
	r := show(m)
	r.Copy = func(s string) { clip = append(clip, s) }
	r.Draw(40, 10)
	if r.Focused() != "r.ktext" {
		t.Fatalf("focused %q, want the text field", r.Focused())
	}
	if taken, _ := r.Key("ctrl+c"); taken || len(clip) != 0 {
		t.Errorf("ctrl+c with nothing selected: taken %v, copied %q; want the program's", taken, clip)
	}
	keys(r, "arrow_left", "shift+arrow_left", "shift+arrow_left", "shift+arrow_left")
	if got, _ := r.Keys(); !strings.Contains(" "+listed(got)+" ", " ctrl+c* ") {
		t.Errorf("a selection and Copy: no ctrl+c* in %s", listed(got))
	}
	if taken, _ := r.Key("ctrl+c"); !taken || !slices.Equal(clip, []string{"ñ👍🏽 "}) {
		t.Errorf("ctrl+c: taken %v, copied %q; want ñ👍🏽 and a space", taken, clip)
	}
	if taken, _ := r.Key("ctrl+x"); !taken || !slices.Equal(clip, []string{"ñ👍🏽 ", "ñ👍🏽 "}) {
		t.Errorf("ctrl+x: taken %v, copied %q; want the same again", taken, clip)
	}
	keys(r, "enter")
	if text != "ab" {
		t.Errorf("after the cut the field committed %q, want ab", text)
	}

	keys(r, "tab", "ctrl+a")
	if r.Focused() != "r.kpw" {
		t.Fatalf("focused %q, want the password field", r.Focused())
	}
	for _, k := range []string{"ctrl+c", "ctrl+x"} {
		if taken, _ := r.Key(k); taken {
			t.Errorf("%s in a password field: taken, want the program's", k)
		}
	}
	keys(r, "enter")
	if len(clip) != 2 || pw != "hunter2" {
		t.Errorf("a password field gave %q and has %q; want nothing, and hunter2", clip[2:], pw)
	}

	text = "bob"
	r = show(m)
	r.Draw(40, 10)
	keys(r, "ctrl+a")
	if taken, _ := r.Key("ctrl+c"); taken {
		t.Error("ctrl+c with no Copy: taken, want the program's")
	}
	if taken, _ := r.Key("ctrl+x"); !taken {
		t.Error("ctrl+x with no Copy: not taken, want the selection deleted")
	}
	keys(r, "enter")
	if text != "" {
		t.Errorf("ctrl+x with no Copy committed %q, want nothing", text)
	}
}
