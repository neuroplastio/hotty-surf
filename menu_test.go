package surf

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"neuroplast.io/hotty-surf/pkg/cells"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

// menuRows is the host application's case: a row's context menu, declared
// by the view and drawn by the host, beside the same rows with none.
const menuRows = `
<surface>
  <stack :each="n in notes" :key="n.id" @click="open(n.id)">
    {{ n.text }}
    <menu>
      <action icon="bell-off-outline" key="m" do="open('mute ' + n.id)">Mute {{ n.id }}</action>
      <action icon="content-copy" do="open('copy ' + n.id)">Copy text</action>
      <sep/>
      <action :disabled="!n.unread" do="open('dismiss ' + n.id)">Dismiss</action>
      <action :if="n.id == 'n1'" :icon="n.text == 'first' ? 'star' : 'x'" do="open('first')">First</action>
      <action :else :key="size(notes)" do="open('not first')">Not first</action>
    </menu>
    <button key="b" @click="open('button')">B<menu><action do="open('inner ' + n.id)">Inner</action></menu></button>
  </stack>
  <field key="f"><menu><action do="open('field')">Field</action></menu></field>
</surface>`

const plainRows = `
<surface>
  <stack :each="n in notes" :key="n.id" @click="open(n.id)">
    {{ n.text }}
    <button key="b" @click="open('button')">B</button>
  </stack>
  <field key="f"/>
</surface>`

func menuSet(t *testing.T, src string) *Set {
	t.Helper()
	set, err := Load(fstest.MapFS{"board.surf": {Data: []byte(src)}}, []*Shape{boardShape, noteShape})
	if err != nil {
		t.Fatal(err)
	}
	return set
}

// renderOf is a model of the Set's board.
type renderOf struct {
	set  *Set
	b    *board
	errs []error
}

func (m *renderOf) Render() *surfui.Node {
	n, errs := m.set.Render(boardObj{m.b})
	m.errs = append(m.errs, errs...)
	return n
}

func (m *renderOf) Do(act surfui.Action, ev surfrun.Event) {
	switch f := act.(type) {
	case func():
		f()
	case func(surfrun.Event):
		f(ev)
	}
}

// A <menu> draws nothing: the document and the cells are those of the
// same template without it, but for the `secondary` its elements ask the
// host for in data-on (SPEC §9).
func TestMenuNotDrawn(t *testing.T) {
	draw := func(src string) (string, string) {
		m := &renderOf{set: menuSet(t, src), b: newBoard()}
		s := surfrun.NewSurface(m, nil)
		r := cells.New(s.Cells())
		r.Show()
		return surfui.Document(s.Tree()), r.Draw(40, 10).Golden()
	}
	hm, cm := draw(menuRows)
	hp, cp := draw(plainRows)
	hm = strings.NewReplacer(` data-on="view secondary"`, ``, ` view secondary"`, `"`).Replace(hm)
	if hm != hp {
		t.Errorf("HTML with a menu:\n%s\nwithout:\n%s", hm, hp)
	}
	if cm != cp {
		t.Errorf("cells with a menu:\n%s\nwithout:\n%s", cm, cp)
	}
}

// An element with a menu asks the host for the secondary press in its
// data-on (SPEC §9): a :each row whose own click is kept, a button, a
// field. Without a menu nothing asks. A menu with no item that can be
// picked asks for nothing.
func TestMenuAsksForSecondary(t *testing.T) {
	on := func(src string) map[string]string {
		m := &renderOf{set: menuSet(t, src), b: newBoard()}
		s := surfrun.NewSurface(m, nil)
		s.Show()
		out := map[string]string{}
		for id, n := range s.Index() {
			out[id] = surfui.DataOn(n)
		}
		return out
	}
	got := on(menuRows)
	for id, want := range map[string]string{
		"r.kn1": "click view secondary", "r.kn2": "click view secondary", "r.kn1.kb": "view secondary", "r.kf": "view secondary", "r": "",
	} {
		if got[id] != want {
			t.Errorf("%s data-on %q, want %q", id, got[id], want)
		}
	}
	for id, d := range on(plainRows) {
		if strings.Contains(d, "secondary") {
			t.Errorf("%s asks for the secondary press with no menu: %q", id, d)
		}
	}
	none := menuSet(t, `<surface><stack><menu><action :disabled="true" do="open('x')">X</action><heading>Only</heading><sep/></menu>x</stack></surface>`)
	m := &renderOf{set: none, b: newBoard()}
	s := surfrun.NewSurface(m, nil)
	s.Show()
	for id, n := range s.Index() {
		if strings.Contains(surfui.DataOn(n), "secondary") {
			t.Errorf("%s asks with nothing to pick: %q", id, surfui.DataOn(n))
		}
	}
}

func TestMenu(t *testing.T) {
	m := &renderOf{set: menuSet(t, menuRows), b: newBoard()}
	s := surfrun.NewSurface(m, nil)
	r := cells.New(s.Cells())
	r.Show()
	r.Draw(40, 10)
	if len(m.errs) > 0 {
		t.Fatal(m.errs)
	}
	text := func(at surfrun.MenuAt) string {
		mn, ok := s.Menu(at)
		if !ok {
			return "none"
		}
		var items []string
		for _, it := range mn.Items {
			if it.Sep {
				items = append(items, it.ID+":-")
				continue
			}
			items = append(items, fmt.Sprintf("%s:%s/%s/%s/%v", it.ID, it.Label, it.Icon, it.Key, it.Enabled))
		}
		return mn.ID + " " + strings.Join(items, " ")
	}
	n1 := "r.kn1 0:Mute n1/bell-off-outline/m/true 1:Copy text/content-copy//true 2:- 3:Dismiss///true 4:First/star//true"
	n2 := "r.kn2 0:Mute n2/bell-off-outline/m/true 1:Copy text/content-copy//true 2:- 3:Dismiss///false 5:Not first//2/true"
	for _, c := range []struct {
		what string
		at   surfrun.MenuAt
		want string
	}{
		{"a row", surfrun.MenuAt{ID: "r.kn1"}, n1},
		{"a row with :disabled and :else", surfrun.MenuAt{ID: "r.kn2"}, n2},
		{"the innermost: a button's in the row", surfrun.MenuAt{ID: "r.kn1.kb"}, "r.kn1.kb 0:Inner///true"},
		{"a field's", surfrun.MenuAt{ID: "r.kf"}, "r.kf 0:Field///true"},
		{"the root, which has none", surfrun.MenuAt{ID: "r"}, "none"},
		{"no element", surfrun.MenuAt{ID: "r.knope"}, "none"},
		{"nothing focused", surfrun.MenuAt{Focus: true}, "none"},
	} {
		if got := text(c.at); got != c.want {
			t.Errorf("%s:\n got %s\nwant %s", c.what, got, c.want)
		}
	}
	// By a cell: the row's text, and its button, inner.
	for id, want := range map[string]string{"r.kn2": n2, "r.kn2.kb": "r.kn2.kb 0:Inner///true"} {
		x, y, ok := r.Where(id)
		if !ok {
			t.Fatalf("%s is not on the screen:\n%s", id, r.Screen().Text())
		}
		if id == "r.kn2" {
			x = 0 // its text, before the button
		}
		if got := text(surfrun.MenuAt{ID: r.At(x, y)}); got != want {
			t.Errorf("at %d, %d (%s): got %s, want %s", x, y, r.At(x, y), got, want)
		}
	}
	// By focus: the button, focused by a press.
	x, y, _ := r.Where("r.kn1.kb")
	r.Press(x, y, 0)
	if got := text(surfrun.MenuAt{Focus: true}); got != "r.kn1.kb 0:Inner///true" {
		t.Errorf("focused %q: got %s", s.Focused(), got)
	}

	b := m.b
	pick := func(menu, item string) string {
		n := len(b.Log)
		took, _ := s.Pick(menu, item)
		return fmt.Sprintf("%v %s", took, strings.Join(b.Log[n:], ","))
	}
	for _, c := range [][3]string{
		{"r.kn1", "0", "true open mute n1"},
		{"r.kn2", "1", "true open copy n2"},
		{"r.kn1.kb", "0", "true open inner n1"},
		{"r.kn2", "3", "false "}, // disabled
		{"r.kn2", "4", "false "}, // left out by its :if
		{"r.kn2", "2", "false "}, // a separator
		{"r.kn9", "0", "false "}, // no such row
		{"r.kn2", "x", "false "}, // no such item
		{"r.kn1", "3", "true open dismiss n1"},
	} {
		if got := pick(c[0], c[1]); got != c[2] {
			t.Errorf("pick %s %s: %s, want %s", c[0], c[1], got, c[2])
		}
	}
	// A pick after the row is gone runs nothing.
	b.Notes = b.Notes[1:]
	s.Update()
	if got := pick("r.kn1", "0"); got != "false " {
		t.Errorf("pick of a row gone: %s", got)
	}
	if got := pick("r.kn2", "0"); got != "true open mute n2" {
		t.Errorf("pick of the row left: %s", got)
	}
}

// A bound icon that is no MDI name is left out at render, and is a
// render error; the item stays.
func TestMenuIconError(t *testing.T) {
	set := menuSet(t, `<surface><menu><action :icon="title" do="open('x')">X</action></menu></surface>`)
	b := newBoard()
	b.Title = "Not An Icon"
	n, errs := set.Render(boardObj{b})
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), `board.surf:1:16: an icon's name is one of MDI's, as cog-outline, not "Not An Icon"`) {
		t.Errorf("errors: %v", errs)
	}
	if items := surfui.MenuOf(n); len(items) != 1 || items[0].Icon != "" || items[0].Label != "X" {
		t.Errorf("items: %+v", items)
	}
}
