package surf

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	cellite "neuroplast.io/cel-lite"
	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

// A component and a data struct, with the instances surfgen would write.

type note struct {
	ID     string
	Text   string
	Unread bool
}

type board struct {
	Title string
	Notes []*note
	Sel   string
	Log   []string
}

func (b *board) Unread() int64 {
	n := int64(0)
	for _, x := range b.Notes {
		if x.Unread {
			n++
		}
	}
	return n
}

func (b *board) Open(id string) { b.Sel = id; b.Log = append(b.Log, "open "+id) }

func (b *board) Pick(e surfrun.Event) { b.Log = append(b.Log, "pick "+e.Value) }

var (
	noteShape = &Shape{Name: "note", Fields: []Field{
		{Name: "id", Type: cellite.String}, {Name: "text", Type: cellite.String}, {Name: "unread", Type: cellite.Bool},
	}}
	unreadT    = cellite.Int
	boardShape = &Shape{Name: "board", Component: true,
		Fields: []Field{
			{Name: "title", Type: cellite.String}, {Name: "notes", Type: cellite.ListOf(cellite.StructOf("note"))},
			{Name: "sel", Type: cellite.String}, {Name: "log", Type: cellite.ListOf(cellite.String)},
		},
		Methods: []Method{
			{Name: "unread", Result: &unreadT},
			{Name: "open", Args: []cellite.Type{cellite.String}},
			{Name: "pick", Args: []cellite.Type{EventType}},
		},
	}
)

type noteObj struct{ p *note }

func (o noteObj) Field(i int) cellite.Value {
	switch i {
	case 0:
		return cellite.StringValue(o.p.ID)
	case 1:
		return cellite.StringValue(o.p.Text)
	case 2:
		return cellite.BoolValue(o.p.Unread)
	}
	return cellite.NullValue()
}

type noteList []*note

func (l noteList) Len() int { return len(l) }
func (l noteList) At(i int) cellite.Value {
	if l[i] == nil {
		return cellite.NullValue()
	}
	return cellite.ObjectValue("note", noteObj{l[i]})
}

type strList []string

func (l strList) Len() int               { return len(l) }
func (l strList) At(i int) cellite.Value { return cellite.StringValue(l[i]) }

type boardObj struct{ p *board }

func (o boardObj) Shape() *Shape { return boardShape }
func (o boardObj) Field(i int) cellite.Value {
	switch i {
	case 0:
		return cellite.StringValue(o.p.Title)
	case 1:
		return cellite.ListValue(noteList(o.p.Notes))
	case 2:
		return cellite.StringValue(o.p.Sel)
	case 3:
		return cellite.ListValue(strList(o.p.Log))
	}
	return cellite.NullValue()
}
func (o boardObj) Call(m int, args []cellite.Value) cellite.Value {
	switch m {
	case 0:
		return cellite.IntValue(o.p.Unread())
	case 1:
		o.p.Open(args[0].String())
	case 2:
		o.p.Pick(EventOf(args[0]))
	}
	return cellite.NullValue()
}

var boardFS = fstest.MapFS{
	"board.surf": {Data: []byte(`
<surface>
  <row gap="1" key="head">
    <heading>{{ title }}</heading>
    <spacer/>
    <chip :if="unread() > 0" fill="accent">{{ unread() }} unread</chip>
    <chip :else fill="muted">All read</chip>
    <select :value="sel" :options="['a', 'b']" @change="pick($event)"/>
  </row>
  <list focusable key="list">
    <bind key="enter" label="Open" do="open(sel)"/>
    <bind :if="size(notes) > 5" key="x" do="open(sel)"/>
    <note-row :each="n in notes" :key="n.id" :note="n" :sel="n.id == sel"/>
    <item :if="size(notes) == 0" key="none"><text muted>Nothing.</text></item>
  </list>
</surface>`)},
	"note-row.surf": {Data: []byte(`
<props note="note" sel="bool"/>
<item :selected="sel" @click="open(note.id)">
  <glyph :tone="note.unread ? 'accent' : 'muted'">●</glyph>
  <text grow :bold="note.unread">{{ note.text }}</text>
</item>`)},
}

func newBoard() *board {
	return &board{Title: "Notes", Sel: "n2", Notes: []*note{
		{"n1", "first", true}, {"n2", "second", false},
	}}
}

// The interpreter draws what the same tree built in Go draws.
func TestRender(t *testing.T) {
	set, err := Load(boardFS, []*Shape{boardShape, noteShape})
	if err != nil {
		t.Fatal(err)
	}
	b := newBoard()
	got, errs := set.Render(boardObj{b})
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	row := func(n *note) *surfui.Node {
		tone := surfui.Muted
		if n.Unread {
			tone = surfui.Accent
		}
		// Modifiers apply in the table's order: bold before grow.
		text := surfui.Text(n.Text)
		if n.Unread {
			text.Bold()
		}
		text.Grow()
		return surfui.Item(surfui.Glyph("●").Tone(tone), text).Selected(n.ID == b.Sel).OnClick(func() {}).Key(n.ID)
	}
	want := NewSurface(
		surfui.Row(surfui.Heading("Notes"), surfui.Spacer(), surfui.Chip("1 unread").Fill(surfui.Accent),
			surfui.Select("n2", "a", "b").OnChange(func(surfrun.Event) {})).Gap(1).Key("head"),
		surfui.List(row(b.Notes[0]), row(b.Notes[1])).Focusable().Key("list").Bind("enter", "Open", func() {}),
	)
	if g, w := html(got), html(want); g != w {
		t.Errorf("got\n%s\nwant\n%s", g, w)
	}

	// A handler calls its method with what its arguments were at render.
	act, _ := surfui.Handler(surfui.Kids(surfui.Kids(got)[1])[0], "click")
	act.(func())()
	if b.Sel != "n1" || b.Log[0] != "open n1" {
		t.Errorf("click: %+v", b)
	}
	pick, _ := surfui.Handler(surfui.Kids(surfui.Kids(got)[0])[3], "change")
	pick.(func(surfrun.Event))(surfrun.Event{Kind: "change", Value: "b"})
	if b.Log[1] != "pick b" {
		t.Errorf("change: %v", b.Log)
	}
	keys := surfui.Bindings(surfui.Kids(got)[1])
	if len(keys) != 1 || keys[0].Key != "enter" || keys[0].Title != "Open" {
		t.Errorf("binds: %+v", keys)
	}

	b.Notes = nil
	got, _ = set.Render(boardObj{b})
	if h := html(got); !strings.Contains(h, "All read") || !strings.Contains(h, "Nothing.") {
		t.Errorf("empty:\n%s", h)
	}
}

func html(n *surfui.Node) string {
	surfui.Resolve(n, "r")
	return surfui.HTML(n)
}

func TestLoadErrors(t *testing.T) {
	// A :if chain whose condition failed once made Load loop, growing
	// without end: end the test binary rather than the machine.
	stop := time.AfterFunc(20*time.Second, func() { panic("Load doesn't end") })
	defer stop.Stop()
	for _, c := range []struct{ src, want string }{
		{`<surface><text>{{ titel }}</text></surface>`, `board.surf:1:19: undeclared reference to 'titel'`},
		{`<surface><text :bold="title">x</text></surface>`, `board.surf:1:23: :bold is a bool, not string`},
		{`<surface><row gap="x"/></surface>`, `board.surf:1:20: gap is an int, not "x"`},
		// The refusal names what the widget takes, its own and every
		// widget's, each once, in alphabetical order.
		{`<surface><row frob/></surface>`, `board.surf:1:15: <row> takes no frob; it takes autofocus, bare, bold, checked, chrome, disabled, down, drag, drop, fill, fit, fit-start, focusable, gap, grid, grow, height, indent, invalid, keep, key, label, muted, name, pad, reveal, scale, scroll, search, segmented, selected, semibold, small, strip, title, tone, top, width or wrap`},
		{`<surface><text w="3">x</text></surface>`, `board.surf:1:16: <text> takes no w; it takes autofocus, bare, bold, checked, chrome, disabled, down, drag, drop, fill, focusable, gap, grow, height, hue, indent, invalid, keep, key, label, muted, name, pad, reveal, scale, scroll, selected, semibold, small, title, tone, top, width or wrap`},
		{`<surface><select value="a" size="3"/></surface>`, `board.surf:1:28: <select> takes no size; it takes autofocus, bare, bold, checked, chrome, disabled, down, drag, drop, fill, focusable, gap, grow, height, indent, invalid, keep, key, label, muted, name, options, pad, reveal, scale, scroll, selected, semibold, small, title, tone, top, value, width or wrap`},
		// A dot's modifier is refused elsewhere, the same way.
		{`<surface><text pulse>x</text></surface>`, `board.surf:1:16: <text> takes no pulse; it takes autofocus, bare, bold, checked, chrome, disabled, down, drag, drop, fill, focusable, gap, grow, height, hue, indent, invalid, keep, key, label, muted, name, pad, reveal, scale, scroll, selected, semibold, small, title, tone, top, width or wrap`},
		{`<surface><blink/></surface>`, `board.surf:1:10: <blink> is no widget; a component's tag has a dash`},
		{`<surface><button @click="close()">x</button></surface>`, `board.surf:1:26: board has no method close`},
		{`<surface><button @click="open()">x</button></surface>`, `board.surf:1:26: open takes 1 arguments, not 0`},
		{`<surface><text tone="loud">x</text></surface>`, `board.surf:1:22: a tone is accent, ok, warn, error, info, muted or plain, not "loud"`},
		{`<surface><face icon="Git Branch"/></surface>`, `board.surf:1:22: an icon's name is one of MDI's, as cog-outline, not "Git Branch"`},
		{`<surface><button badge="2" badge-tone="ok">x</button></surface>`, `board.surf:1:40: a badge's tone is accent, warn, error or muted, not "ok"`},
		{`<surface><text :else>x</text></surface>`, `board.surf:1:16: :else with no :if before it`},
		{`<surface><text :if="nope">a</text></surface>`, `board.surf:1:21: undeclared reference to 'nope'`},
		{`<surface><text :if="nope">a</text><text :else>b</text></surface>`, `board.surf:1:21: undeclared reference to 'nope'`},
		{`<surface><text :if="true">a</text><text :else-if="nope">b</text><text :else>c</text></surface>`, `board.surf:1:51: undeclared reference to 'nope'`},
		{`<surface><x-y/></surface>`, `board.surf:1:10: no template x-y.surf for <x-y>`},
		// A dyn :state is the tag's component's, which there must be.
		{`<surface><x-bare :state="dyn(title)"/></surface>`, `board.surf:1:26: <x-bare>'s :state is dyn, and x-bare.surf is no component's template`},
		{`<surface/><stack/>`, `board.surf:1:11: a template has one root element; <stack> is a second`},
		// Directives that can't go together.
		{`<surface><text :if="true" :else>a</text></surface>`, `board.surf:1:27: :if and :else on one element`},
		{`<surface><text :if="true">a</text><text :else-if="true" :each="n in notes">b</text></surface>`, `board.surf:1:41: :each and :else-if on one element`},
		{`<surface><text :each="n in notes" :else>a</text></surface>`, `board.surf:1:35: :each and :else on one element`},
		// {{ }} of a value with no text.
		{`<surface><text>{{ notes }}</text></surface>`, `board.surf:1:19: {{ }} of a list(note), which has no text`},
		{`<surface><text>{{ notes[0] }}</text></surface>`, `board.surf:1:19: {{ }} of a note, which has no text`},
		{`<surface><text>{{ null }}</text></surface>`, `board.surf:1:19: {{ }} of a null_type, which has no text`},
		// What would be dropped without a word.
		{`<bind key="enter" do="open('x')"/>`, `board.surf:1:1: <bind> has no widget around it to bind on`},
		{`<surface><x-box><text>a</text></x-box></surface>`, `board.surf:1:17: <x-box> has no <slot/> for these children`},
		{`<surface><x-box><text slot="foot">a</text></x-box></surface>`, `board.surf:1:23: <x-box> has no slot foot`},
		{`<surface><x-bare><bind key="enter" do="open('x')"/></x-bare></surface>`, `board.surf:1:18: <bind> has no widget around it to bind on`},
		// A <bind> runs a do or scrolls, one of them.
		{`<surface><list><bind key="j" scroll="sideways"/></list></surface>`, `board.surf:1:30: scroll is up, down, page-up, page-down, half-page-up, half-page-down, start, end, left or right, not sideways`},
		{`<surface><list><bind key="j" :scroll="title"/></list></surface>`, `board.surf:1:30: scroll is up, down, page-up, page-down, half-page-up, half-page-down, start, end, left or right, not title`},
		{`<surface><list><bind key="j" scroll="down" do="open('x')"/></list></surface>`, `board.surf:1:16: <bind> has a key, and a do or a scroll`},
		{`<surface><list><bind key="j"/></list></surface>`, `board.surf:1:16: <bind> has a key, and a do or a scroll`},
		// A do that doesn't compile is its own error, not a missing do.
		{`<surface><list><bind key="x" do="close()"/></list></surface>`, `board.surf:1:34: board has no method close`},
		// The event's names: an @ handler's arguments only, each whole,
		// to a parameter of its type.
		{`<surface><text>{{ $value }}</text></surface>`, `board.surf:1:19: $value is an @ handler's argument only`},
		{`<surface><text :bold="$event">x</text></surface>`, `board.surf:1:23: $event is an @ handler's argument only`},
		{`<surface><list><bind key="x" do="open($value)"/></list></surface>`, `board.surf:1:39: $value is an @ handler's argument only`},
		{`<surface><button @click="open($value + 'x')">x</button></surface>`, `board.surf:1:31: $value is a whole argument, not part of an expression`},
		{`<surface><button @click="open(($value))">x</button></surface>`, `board.surf:1:32: $value is a whole argument, not part of an expression`},
		{`<surface><button @click="open($evnt)">x</button></surface>`, `board.surf:1:31: $evnt is neither $value nor $event`},
		{`<surface><button @click="open($event)">x</button></surface>`, `board.surf:1:31: open's argument 1 is string, not $event, which goes to a surfrun.Event parameter only`},
		{`<surface><button @click="pick">x</button></surface>`, `board.surf:1:26: pick takes 1 arguments, not 0: its argument 1 is a surfrun.Event, which only $event gives`},
		{`<surface><list><bind key="x" do="pick"/></list></surface>`, `board.surf:1:34: pick takes 1 arguments, not 0: its argument 1 is a surfrun.Event, which only $event gives, in an @ handler`},
		{`<surface><button @click="pick($value)">x</button></surface>`, `board.surf:1:31: pick's argument 1 is a surfrun.Event, which only $event gives`},
		{`<surface><button @click="pick(null)">x</button></surface>`, `board.surf:1:31: pick's argument 1 is a surfrun.Event, which only $event gives`},
		{`<surface><text>{{ unread($event) }}</text></surface>`, `board.surf:1:26: $event is an @ handler's argument only`},
		// A context menu: one <menu> in a widget, of <action>s and <sep/>s.
		{`<surface><action do="open('x')">A</action></surface>`, `board.surf:1:10: <action> goes in a <menu>`},
		{`<surface><row><sep/></row></surface>`, `board.surf:1:15: <sep> goes in a <menu>`},
		{`<menu><action do="open('x')">A</action></menu>`, `board.surf:1:1: <menu> goes in a widget, whose context menu it is`},
		{`<surface><x-bare><menu><action do="open('x')">A</action></menu></x-bare></surface>`, `board.surf:1:18: <menu> goes in a widget, whose context menu it is`},
		{`<surface><row><menu><action do="open('x')">A</action></menu><menu><sep/></menu></row></surface>`, `board.surf:1:61: <row> has one <menu>; this is a second`},
		{`<surface><row><menu><text>A</text></menu></row></surface>`, `board.surf:1:21: <menu> holds <action>s, <heading>s and <sep/>s, not <text>`},
		{`<surface><row><menu>Open</menu></row></surface>`, `board.surf:1:21: <menu> holds <action>s, <heading>s and <sep/>s, not text`},
		{`<surface><row><menu><heading key="h">A</heading><action do="open('x')">B</action></menu></row></surface>`, `board.surf:1:30: a menu's <heading> takes nothing, not key`},
		{`<surface><row><menu><heading><text>A</text></heading><action do="open('x')">B</action></menu></row></surface>`, `board.surf:1:30: a menu's <heading> is text, not <text>`},
		{`<surface><row><menu><heading/><action do="open('x')">B</action></menu></row></surface>`, `board.surf:1:21: a menu's <heading> has its text: <heading>Move to</heading>`},
		{`<surface><row><menu/></row></surface>`, `board.surf:1:15: <menu> has an <action>`},
		{`<surface><row><menu :if="true"><sep/></menu></row></surface>`, `board.surf:1:21: <menu> takes nothing, not :if`},
		{`<surface><row><menu><action>Open</action></menu></row></surface>`, `board.surf:1:21: <action> has a do, the handler a pick of it runs: do="method(args)"`},
		{`<surface><row><menu><action do="open('x')"/></menu></row></surface>`, `board.surf:1:21: <action> has a label, its text: <action do="…">Label</action>`},
		{`<surface><row><menu><action do="open('x')"><text>A</text></action></menu></row></surface>`, `board.surf:1:44: an <action>'s label is text, not <text>`},
		{`<surface><row><menu><action :do="open('x')">A</action></menu></row></surface>`, `board.surf:1:29: do is a handler: do="method(args)"`},
		{`<surface><row><menu><action do="close()">A</action></menu></row></surface>`, `board.surf:1:33: board has no method close`},
		{`<surface><row><menu><action do="open($value)">A</action></menu></row></surface>`, `board.surf:1:38: $value is an @ handler's argument only`},
		{`<surface><row><menu><action do="open('x')" tone="ok">A</action></menu></row></surface>`, `board.surf:1:44: <action> takes no tone; it takes danger, disabled, do, icon or key`},
		{`<surface><row><menu><action do="open('x')" icon="Content Copy">A</action></menu></row></surface>`, `board.surf:1:50: an icon's name is one of MDI's, as cog-outline, not "Content Copy"`},
		{`<surface><row><menu><action do="open('x')" :icon="unread()">A</action></menu></row></surface>`, `board.surf:1:51: :icon is a string, not int`},
		{`<surface><row><menu><action do="open('x')" :disabled="title">A</action></menu></row></surface>`, `board.surf:1:55: :disabled is a bool, not string`},
		{`<surface><row><menu><action do="open('x')" disabled="maybe">A</action></menu></row></surface>`, `board.surf:1:54: disabled is bare, true or false, not "maybe"`},
		{`<surface><row><menu><sep key="x"/></menu></row></surface>`, `board.surf:1:26: <sep> takes nothing, not key`},
		{`<surface><row><menu><sep>x</sep></menu></row></surface>`, `board.surf:1:26: <sep> has no children`},
		{`<surface><row><menu><action :each="n in notes" do="open(n.id)">A</action></menu></row></surface>`, `board.surf:1:29: <action> takes no :each: each of a menu's items is one of its template's, whose place is its id`},
		{`<surface><row><menu><action do="open('x')">{{ notes }}</action></menu></row></surface>`, `board.surf:1:47: {{ }} of a list(note), which has no text`},
	} {
		fs := fstest.MapFS{"board.surf": {Data: []byte(c.src)},
			"x-box.surf":  {Data: []byte(`<box><slot name="head"/></box>`)},
			"x-bare.surf": {Data: []byte(`<slot/>`)}}
		_, err := Load(fs, []*Shape{boardShape, noteShape})
		if err == nil || err.Error() != c.want {
			t.Errorf("%s:\n got %v\nwant %s", c.src, err, c.want)
		}
	}
}

// The event's names are found outside strings and comments, and are
// whole arguments only inside the outermost call's parentheses.
func TestDollars(t *testing.T) {
	for _, c := range []struct {
		src  string
		want []dollar
	}{
		{`f($value)`, []dollar{{2, "value", true}}},
		{`f(a, $event , b)`, []dollar{{5, "event", true}}},
		{`f('$value', $event)`, []dollar{{12, "event", true}}},
		{`f("$value", b'$value', r"$value", """$value""", '''a ' $value''')`, nil},
		{`f(r'a\', $value)`, []dollar{{9, "value", true}}},
		{`f('a\', $value)`, nil}, // the string doesn't end: cel-lite says so
		{"f(x) // $value", nil},
		{"f(x, // c\n $value // d\n)", []dollar{{11, "value", true}}},
		{`f(($value))`, []dollar{{3, "value", false}}},
		{`f([$value])`, []dollar{{3, "value", false}}},
		{`f(g($value))`, []dollar{{4, "value", false}}},
		{`f($value + 'x')`, []dollar{{2, "value", false}}},
		{`$value`, []dollar{{0, "value", false}}},
		{`f($ev, $)`, []dollar{{2, "ev", true}, {7, "", true}}},
	} {
		got := dollars(c.src)
		if fmt.Sprint(got) != fmt.Sprint(c.want) {
			t.Errorf("%s: %v, want %v", c.src, got, c.want)
		}
	}
}

// $value gives a handler the event's value and $event the event, each at
// its parameter's place; a '$value' literal is itself.
func TestEventArgs(t *testing.T) {
	fsys := fstest.MapFS{"board.surf": {Data: []byte(`<surface>
  <button key="lit" @click="open('$value')">x</button>
  <field key="val" @change="open($value)"/>
  <select key="ev" :options="['a']" @change="pick($event)"/>
</surface>`)}}
	set, err := Load(fsys, []*Shape{boardShape, noteShape})
	if err != nil {
		t.Fatal(err)
	}
	b := newBoard()
	root, errs := set.Render(boardObj{b})
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	ids := surfui.Resolve(root, "r")
	click, _ := surfui.Handler(ids["r.klit"], "click")
	click.(func())()
	change, _ := surfui.Handler(ids["r.kval"], "change")
	change.(func(surfrun.Event))(surfrun.Event{Kind: "change", Value: "typed"})
	pick, _ := surfui.Handler(ids["r.kev"], "change")
	pick.(func(surfrun.Event))(surfrun.Event{Kind: "change", Value: "a"})
	if got := strings.Join(b.Log, ", "); got != "open $value, open typed, pick a" {
		t.Errorf("log: %s", got)
	}
}

func TestNames(t *testing.T) {
	for _, c := range [][3]string{
		{"AgentRow", "agent-row", "agentRow"}, {"notices", "notices", "notices"},
		{"ID", "id", "id"}, {"IDs", "i-ds", "ids"}, {"HTTPServer", "http-server", "httpServer"},
	} {
		if got := TemplateName(c[0]); got != c[1] {
			t.Errorf("TemplateName(%s) = %s, want %s", c[0], got, c[1])
		}
		if got := LowerCamel(c[0]); got != c[2] {
			t.Errorf("LowerCamel(%s) = %s, want %s", c[0], got, c[2])
		}
	}
}

// dynObj is a dynamic instance whose fields are dyn, as a host with one
// number type makes, or not.
type dynObj struct {
	shape *Shape
	vals  []cellite.Value
}

func (o dynObj) Shape() *Shape                           { return o.shape }
func (o dynObj) Field(i int) cellite.Value               { return o.vals[i] }
func (o dynObj) Call(int, []cellite.Value) cellite.Value { return cellite.NullValue() }

// A host with one number type gives a number to an int or a double prop
// as the prop's type, and refuses another by the prop's name; a Go host
// gives it as it is, and the prop refuses a number of the other kind.
func TestOneNumber(t *testing.T) {
	fsys := fstest.MapFS{
		"nums.surf": {Data: []byte(`<surface><as-d key="a" :d="w"/><as-i key="b" :n="h"/><as-i key="c" :n="x"/></surface>`)},
		"as-d.surf": {Data: []byte(`<props d="double"/><text>{{ d + 0.5 }}</text>`)},
		"as-i.surf": {Data: []byte(`<props n="int"/><text>{{ n + 1 }}</text>`)},
	}
	for _, c := range []struct {
		one        bool
		text, errs string
	}{
		{true, "1.5 3 1", "nums.surf:1:54: prop n: 2.5 is not an int"},
		{false, "0.5 1 1", "nums.surf:1:10: a prop of type double, not an int\nnums.surf:1:32: a prop of type int, not a double\nnums.surf:1:54: a prop of type int, not a double"},
	} {
		dyn := cellite.Dyn
		shape := &Shape{Name: "nums", Component: true, OneNumber: c.one,
			Fields: []Field{{Name: "h", Type: dyn}, {Name: "w", Type: dyn}, {Name: "x", Type: dyn}}}
		set, err := Load(fsys, []*Shape{shape})
		if err != nil {
			t.Fatal(err)
		}
		root, errs := set.Render(dynObj{shape, []cellite.Value{cellite.DoubleValue(2), cellite.IntValue(1), cellite.DoubleValue(2.5)}})
		var texts, msgs []string
		for _, k := range []string{"r.ka", "r.kb", "r.kc"} {
			texts = append(texts, surfui.TextOf(surfui.Resolve(root, "r")[k]))
		}
		for _, err := range errs {
			msgs = append(msgs, err.Error())
		}
		if got := strings.Join(texts, " "); got != c.text {
			t.Errorf("one number %v: %q, want %q", c.one, got, c.text)
		}
		if got := strings.Join(msgs, "\n"); got != c.errs {
			t.Errorf("one number %v: errors\n%s\nwant\n%s", c.one, got, c.errs)
		}
	}
}

// An attribute's Values are what it takes: a literal of each loads, and
// a word that is none of them doesn't.
func TestAttrValues(t *testing.T) {
	n := 0
	for i := range Attrs {
		a := &Attrs[i]
		if a.Values == nil {
			continue
		}
		n++
		if a.Type != TypeString || !a.Fails {
			t.Errorf("%s has values, but is no string a modifier refuses", a.Name)
		}
		for _, v := range a.Values {
			if err := checkLiteral(a, v); err != nil {
				t.Errorf("%s=%q: %v", a.Name, v, err)
			}
		}
		if err := checkLiteral(a, "frob"); err == nil {
			t.Errorf("%s takes frob, which is none of its values", a.Name)
		}
	}
	if n != 6 {
		t.Errorf("%d attributes have values, not 6 (tone, fill, theme, density, badge-tone, drop)", n)
	}
}

// Props lists a template's declared props, in their order, a partial's
// that no template uses too.
func TestProps(t *testing.T) {
	set, err := Load(fstest.MapFS{
		"board.surf":    {Data: []byte(`<surface><note-row :n="notes[0]" sel/></surface>`)},
		"note-row.surf": {Data: []byte(`<props n="note" sel="bool"/><text>{{ n.text }}</text>`)},
		"unused.surf":   {Data: []byte(`<props keys="list(string)" count="int"/><text>{{ count }}</text>`)},
	}, []*Shape{boardShape, noteShape})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name string
		want string
	}{{"note-row", "n note, sel bool"}, {"unused", "keys list(string), count int"}, {"board", ""}} {
		props, ok := set.Props(c.name)
		if !ok {
			t.Errorf("%s: no template", c.name)
			continue
		}
		var got []string
		for _, p := range props {
			got = append(got, p.Name+" "+cellite.FormatType(p.Type))
		}
		if s := strings.Join(got, ", "); s != c.want {
			t.Errorf("%s's props are %q, want %q", c.name, s, c.want)
		}
	}
	if _, ok := set.Props("nothing"); ok {
		t.Error("Props found a template the Set has not")
	}
}

// A frame holds a component chosen as it runs, which <embed> draws.
type frameObj struct {
	shape *Shape
	inset surflive.Component
}

func (o frameObj) Shape() *Shape { return o.shape }
func (o frameObj) Field(i int) cellite.Value {
	if i == 0 {
		return ComponentValue(o.inset)
	}
	return cellite.StringValue("x")
}
func (o frameObj) Call(int, []cellite.Value) cellite.Value { return ComponentValue(o.inset) }

var frameShape = &Shape{Name: "frame", Component: true,
	Fields:  []Field{{Name: "inset", Type: ComponentType}, {Name: "title", Type: cellite.String}},
	Methods: []Method{{Name: "pick", Result: TypeOf(ComponentType)}}}

// goInset is a component of Go's: a row, with the depth it was handed.
type goInset struct{ depth int }

func (g *goInset) Render(c *surflive.Ctx) *surfui.Node {
	g.depth = c.Depth()
	return surfui.Row(surfui.Text("inset"))
}

// <embed> draws a component the owner holds by its own Render, keyed;
// null draws nothing; and a template drawn there goes on from the
// embed's level, so that the whole nests 32 levels at most.
func TestEmbed(t *testing.T) {
	fs := fstest.MapFS{"frame.surf": {Data: []byte(`<surface>
  <embed :of="inset" key="in"/>
  <stack key="s"><embed :of="pick()" :key="title"/></stack>
</surface>`)}}
	set, err := Load(fs, []*Shape{frameShape})
	if err != nil {
		t.Fatal(err)
	}
	g := &goInset{}
	n, errs := set.Render(frameObj{frameShape, g})
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	surfui.Resolve(n, "r")
	want := `<div id="r" class="u-surface u-theme-graphite u-compact"><div id="r.kin" class="u-row"><span id="r.kin.0" class="u-text">inset</span></div>` +
		`<div id="r.ks" class="u-stack"><div id="r.ks.kx" class="u-row"><span id="r.ks.kx.0" class="u-text">inset</span></div></div></div>`
	if got := surfui.HTML(n); got != want {
		t.Errorf("drew\n%s\nwant\n%s", got, want)
	}
	if g.depth != 3 { // the surface, the stack, the embed
		t.Errorf("the inset in the stack was handed depth %d, want 3", g.depth)
	}
	if n, errs = set.Render(frameObj{frameShape, nil}); len(errs) > 0 || len(surfui.Kids(n)) != 1 {
		t.Errorf("a null :of drew %d nodes, errors %v; want only the stack", len(surfui.Kids(n)), errs)
	}

	// A view embedded at level 2 has 30 levels left: its template's
	// 31st nested element is past the limit.
	deep := strings.Repeat("<stack>", 31) + strings.Repeat("</stack>", 31)
	inner, err := Load(fstest.MapFS{"spec-case.surf": {Data: []byte(deep)}}, []*Shape{specShape})
	if err != nil {
		t.Fatal(err)
	}
	var heard []error
	view := Component(inner, specCase{}).(*View)
	view.OnError = func(err error) { heard = append(heard, err) }
	outer, err := Load(fstest.MapFS{"frame.surf": {Data: []byte(`<surface><embed :of="inset"/></surface>`)}}, []*Shape{frameShape})
	if err != nil {
		t.Fatal(err)
	}
	if _, errs := outer.Render(frameObj{frameShape, view}); len(errs) > 0 {
		t.Fatal(errs)
	}
	if len(heard) != 1 || !strings.Contains(heard[0].Error(), "spec-case.surf:1:211: nested past 32 levels") {
		t.Errorf("the embedded view heard %v, want its 31st stack past the limit", heard)
	}

	for _, c := range []struct{ src, want string }{
		{`<surface><embed of="inset"/></surface>`, `frame.surf:1:17: of is bound: :of="expr", a surflive.Component`},
		{`<surface><embed :of="title"/></surface>`, `frame.surf:1:22: <embed>'s :of is a surflive.Component, not string`},
		{`<surface><embed/></surface>`, `frame.surf:1:10: <embed> has an :of, the surflive.Component it draws`},
		{`<surface><embed :of="inset" bold/></surface>`, `frame.surf:1:29: <embed> takes :of and key, not bold`},
		{`<surface><embed :of="inset"><text>x</text></embed></surface>`, `frame.surf:1:29: <embed> has no children`},
	} {
		_, err := Load(fstest.MapFS{"frame.surf": {Data: []byte(c.src)}}, []*Shape{frameShape})
		if err == nil || err.Error() != c.want {
			t.Errorf("%s:\n got %v\nwant %s", c.src, err, c.want)
		}
	}
}
