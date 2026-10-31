package surfui

import (
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-go"
)

// keymap is the keymap a host reads for n while n is focused (SPEC
// §10.2): the data-keys of each element from the root down to n, n's
// last; a text field's on its default keymap.
func keymap(n *Node) *hotty.Keymap {
	var values []string
	for p := n; p != nil; p = p.parent {
		if k := dataKeys(p); k != "" {
			values = append([]string{k}, values...)
		}
	}
	if IsTextField(n) {
		return hotty.Resolve(hotty.Platform{}, false, values...)
	}
	return hotty.ParseKeymap(hotty.Platform{}, strings.Join(values, " "))
}

// A binding of a key a host scrolls with gives it to the program from the
// element it binds on, so that it holds in everything in that element:
// the binding hears it before any scroll. Other keys reach the program
// anyway, and a button keeps Space, which presses it.
func TestBindingsGiveScrollKeys(t *testing.T) {
	row := Item(Text("a")).Attr("tabindex", "0").Key("a")
	btn := Button("Open").OnClick("open")
	sw := Switch(false).OnClick("flip")
	sel := Select("b", "a", "b")
	list := List(row, Item(btn, sw, sel).Key("b")).Focusable().
		Bind("arrow_down", "Next", "next").Bind("arrow_left", "Close", "close").Bind("home", "First", "first").
		Bind("space", "Mark", "mark").Bind("shift+space", "Unmark", "unmark").
		Bind("enter", "Open", "open").Bind("j", "Next", "next").Bind("ctrl+n", "Next", "next")
	box := Stack(list).Scroll()
	root := Surface(box).Bind("page_down", "Page", "page").Bind("escape", "Quit", "quit")
	Resolve(root, "r")

	for _, c := range []struct {
		name string
		n    *Node
		want string
	}{
		{"the list", list, "arrow_down=program arrow_left=program home=program space=program shift+space=program"},
		{"the root", root, "page_down=program"},
		{"the scroll box", box, ""},
		{"a row", row, ""},
		{"a button", btn, "space=scroll-page-down shift+space=scroll-page-up"},
		{"a switch", sw, "space=scroll-page-down shift+space=scroll-page-up"},
		{"a select", sel, strings.Join(selectKeys, " ") + " space=scroll-page-down shift+space=scroll-page-up"},
	} {
		if got := dataKeys(c.n); got != c.want {
			t.Errorf("%s's data-keys are %q, want %q", c.name, got, c.want)
		}
	}
	for _, c := range []struct {
		name    string
		n       *Node
		program []string
		not     []string
	}{
		{"the list", list, []string{"arrow_down", "arrow_left", "home", "space", "shift+space", "page_down"}, []string{"arrow_up", "end", "enter", "j"}},
		{"a row", row, []string{"arrow_down", "space", "page_down"}, []string{"arrow_up"}},
		{"the scroll box", box, []string{"page_down"}, []string{"arrow_down", "space"}},
		{"a button", btn, []string{"arrow_down", "home", "page_down"}, []string{"space", "shift+space", "enter"}},
		{"a select", sel, []string{"arrow_down", "arrow_up", "arrow_left"}, []string{"space", "enter"}},
	} {
		km := keymap(c.n)
		for _, k := range c.program {
			if !km.Program(k) {
				t.Errorf("focused, %s doesn't give the program %s", c.name, k)
			}
		}
		for _, k := range c.not {
			if km.Program(k) {
				t.Errorf("focused, %s gives the program %s", c.name, k)
			}
		}
	}

	// Open, a Select gives Space and Enter, which pick in its list.
	OpenSelect(sel)
	if got, want := dataKeys(sel), strings.Join(selectOpenKeys, " "); got != want {
		t.Errorf("an open select's data-keys are %q, want %q", got, want)
	}
	// An element that binds none of those keys has no data-keys, as before.
	if h := HTML(Surface(Button("x")).Bind("escape", "Quit", "quit")); strings.Contains(h, "data-keys") {
		t.Errorf("escape alone: %s", h)
	}
}

// A text field keeps the keys it edits with: the arrows, Home and End an
// element around it gives the program it binds back, and an element with
// a field in it gives no Space, which the field types.
func TestFieldKeepsItsKeys(t *testing.T) {
	f := Field()
	list := List(Item(f).Key("a")).Focusable().
		Bind("arrow_left", "Close", "close").Bind("arrow_right", "Open", "open").
		Bind("home", "First", "first").Bind("end", "Last", "last").
		Bind("arrow_down", "Next", "next").Bind("space", "Mark", "mark")
	Resolve(Surface(list), "r")
	if got, want := dataKeys(list), "arrow_left=program arrow_right=program home=program end=program arrow_down=program"; got != want {
		t.Errorf("the list's data-keys are %q, want %q", got, want)
	}
	if got, want := dataKeys(f), "arrow_left=char-backward arrow_right=char-forward home=line-start end=line-end arrow_down=program"; got != want {
		t.Errorf("the field's data-keys are %q, want %q", got, want)
	}
	km := keymap(f)
	for key, want := range map[string]hotty.Action{
		"arrow_left": hotty.CharBackward, "arrow_right": hotty.CharForward,
		"home": hotty.LineStart, "end": hotty.LineEnd,
	} {
		if got := km.Lookup(key); got != want {
			t.Errorf("in the field %s is %q, want %q", key, got, want)
		}
	}
	if !km.Program("arrow_down") || km.Program("space") {
		t.Error("the field gives the program the wrong keys of the list's")
	}
	if !keymap(list).Program("arrow_left") || keymap(list).Program("space") {
		t.Error("focused, the list gives the wrong keys")
	}

	// Its own bindings it gives as before: only those a field can do
	// without.
	own := Field().Bind("arrow_left", "x", "x").Bind("arrow_up", "y", "y")
	Resolve(Surface(own), "r")
	if got := dataKeys(own); got != "arrow_up=program" {
		t.Errorf("a field's own ← and ↑: %q", got)
	}
}

func TestScrollKey(t *testing.T) {
	for _, k := range []string{"arrow_up", "arrow_down", "arrow_left", "arrow_right", "page_up", "page_down", "home", "end", "space", "shift+space"} {
		if !ScrollKey(k) {
			t.Errorf("%s is a key a host scrolls with", k)
		}
	}
	for _, k := range []string{"enter", "j", "ctrl+home", "shift+arrow_down", "alt+space", "tab", "escape"} {
		if ScrollKey(k) {
			t.Errorf("%s is no key a host scrolls with", k)
		}
	}
}

// A binding to a scroll action gives the key to the host, to scroll with:
// a button in it keeps Space with no stand-in, and a field in it types
// the key.
func TestBindScroll(t *testing.T) {
	btn := Button("Open").OnClick("open")
	f := Field()
	pane := Stack(btn, f).Scroll().
		BindScroll("j", "Down", hotty.ScrollDown).BindScroll("space", "Page", hotty.ScrollPageDown).
		Bind("arrow_down", "Next", "next")
	root := Surface(pane).Bind("j", "Jump", "jump")
	Resolve(root, "r")
	if got, want := dataKeys(pane), "j=scroll-down space=scroll-page-down arrow_down=program"; got != want {
		t.Errorf("the pane's data-keys are %q, want %q", got, want)
	}
	if got := dataKeys(btn); got != "" {
		t.Errorf("a button under a scroll's Space: %q", got)
	}
	if got := dataKeys(root); got != "" {
		t.Errorf("the root's j, no key a host scrolls with: %q", got)
	}
	if km := keymap(btn); km.Scroll("j") != hotty.ScrollDown || km.Scroll("space") != hotty.ScrollPageDown {
		t.Errorf("focused, the button doesn't scroll with j and Space")
	}
	if km := keymap(f); km.Lookup("j") != hotty.Insert || km.Lookup("space") != hotty.Insert || km.Scroll("j") != "" {
		t.Errorf("the field doesn't type j and Space: %q, %q", km.Lookup("j"), km.Lookup("space"))
	}
	if _, ok := ScrollAction("sideways"); ok {
		t.Error("sideways is a scroll action")
	}
	if a, ok := ScrollAction("half-page-down"); !ok || a != hotty.ScrollHalfPageDown {
		t.Errorf("half-page-down is %q", a)
	}
}
