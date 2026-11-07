package surfui_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-surf/surfui"
)

func TestResolveGivesKeyPaths(t *testing.T) {
	list := surfui.List(surfui.Item(surfui.Text("a")).Key("a b"), surfui.Item(surfui.Text("b")).Key("b"))
	root := surfui.Surface(surfui.Heading("h"), list)
	ix := surfui.Resolve(root, "r")
	for _, id := range []string{"r", "r.0", "r.1", "r.1.ka_20b", "r.1.kb", "r.1.kb.0"} {
		if ix[id] == nil {
			t.Errorf("no %s", id)
		}
	}
	if surfui.Parent(ix["r.1.kb"]) != list {
		t.Error("a row's parent is not its list")
	}
}

func TestDiffSendsTheCheapOps(t *testing.T) {
	view := func(status string, sel bool, w int) *surfui.Node {
		n := surfui.Surface(surfui.Text(status), surfui.Item(surfui.Text("x")).Selected(sel), surfui.Box().Width(w))
		surfui.Resolve(n, "r")
		return n
	}
	ops := surfui.Diff(view("idle", false, 10), view("busy", true, 12))
	var got []string
	for _, op := range ops {
		got = append(got, op.Op+" "+op.Target)
	}
	want := []string{"text r.0", "attr r.1", "attr r.1", "var r.2"}
	if !slices.Equal(got, want) {
		t.Fatalf("ops %v, want %v", got, want)
	}
	if ops := surfui.Diff(view("idle", false, 10), view("idle", false, 10)); len(ops) != 0 {
		t.Fatalf("no change sent %v", ops)
	}
}

// Reveals brings into view each element that becomes selected, in
// document order, by as little as it can: none for a selection that
// stays, or goes.
func TestRevealsTheNewSelection(t *testing.T) {
	view := func(sel ...string) (*surfui.Node, surfui.Index) {
		l := surfui.List().Key("l")
		for _, id := range []string{"a", "b", "c", "d"} {
			l.Add(surfui.Item(surfui.Text(id)).Key(id).Selected(slices.Contains(sel, id)))
		}
		n := surfui.Surface(surfui.Stack(l).Scroll())
		return n, surfui.Resolve(n, "r")
	}
	reveals := func(old surfui.Index, next *surfui.Node) []string {
		var got []string
		for _, op := range surfui.Reveals(old, next) {
			if op.Op != "reveal" || op.Value != nil {
				t.Errorf("%+v is no reveal by as little as it can", op)
			}
			got = append(got, op.Target)
		}
		return got
	}
	b, ib := view("b")
	if got := reveals(nil, b); !slices.Equal(got, []string{"r.0.kl.kb"}) {
		t.Errorf("a new document reveals %v, want b", got)
	}
	c, ic := view("c")
	if got := reveals(ib, c); !slices.Equal(got, []string{"r.0.kl.kc"}) {
		t.Errorf("b to c reveals %v, want c", got)
	}
	if again, _ := view("c"); len(reveals(ic, again)) != 0 {
		t.Error("a selection that stays is revealed again")
	}
	if none, _ := view(); len(reveals(ic, none)) != 0 {
		t.Error("a selection that goes reveals something")
	}
	if two, _ := view("d", "a", "c"); !slices.Equal(reveals(ic, two), []string{"r.0.kl.ka", "r.0.kl.kd"}) {
		t.Errorf("c to a, c and d reveals %v, want a and d in order", reveals(ic, two))
	}
	if !surfui.IsSelected(ic["r.0.kl.kc"]) || surfui.IsSelected(ic["r.0.kl.ka"]) {
		t.Error("IsSelected is wrong")
	}
}

// The setters change a tree in place, as a fine-grained model does, and
// the readers give copies.
func TestSettersAndReaders(t *testing.T) {
	n := surfui.Text("old").Class("x").Tone(surfui.Warn).Attr("title", "t").Var("w", "3")
	surfui.SetText(n, "new")
	if surfui.TextOf(n) != "new" {
		t.Errorf("text %q", surfui.TextOf(n))
	}
	n.Unclass("x").Unattr("title").Unvar("w")
	cs := surfui.Classes(n)
	if slices.Contains(cs, "x") || !slices.Contains(cs, "u-text") {
		t.Errorf("classes %v", cs)
	}
	cs[0] = "changed"
	if surfui.Classes(n)[0] == "changed" {
		t.Error("Classes is not a copy")
	}
	if len(surfui.Attrs(n)) != 0 || len(surfui.Vars(n)) != 0 {
		t.Errorf("attrs %v, vars %v", surfui.Attrs(n), surfui.Vars(n))
	}
	if a := surfui.Attrs(surfui.Field().Placeholder("p")); !slices.Contains(a, surfui.Attr{K: "placeholder", V: "p"}) {
		t.Errorf("field attrs %v", a)
	}

	row := surfui.Row(surfui.Text("a"))
	old := surfui.Kids(row)
	b := surfui.Text("b")
	surfui.SetKids(row, nil, b)
	if k := surfui.Kids(row); len(k) != 1 || k[0] != b || surfui.Parent(b) != row {
		t.Errorf("kids %v", k)
	}
	if surfui.TextOf(old[0]) != "a" {
		t.Error("SetKids changed the old children's slice")
	}

	raw := surfui.Raw("<svg></svg>")
	surfui.SetRaw(raw, "<svg><path/></svg>")
	if h := surfui.HTML(raw); h != "<span><svg><path/></svg></span>" {
		t.Errorf("raw %s", h)
	}
}

func TestCheckedIsSwitchsState(t *testing.T) {
	for _, on := range []bool{false, true} {
		a, b := surfui.Switch(on), surfui.Switch(!on).Unclass("is-on").Checked(on)
		if surfui.HTML(a) != surfui.HTML(b) {
			t.Errorf("on=%v: %s, flipped %s", on, surfui.HTML(a), surfui.HTML(b))
		}
	}
}

func TestGiveable(t *testing.T) {
	for key, want := range map[string]bool{
		"mod+enter": true, "alt+enter": true, "ctrl+n": true, "enter": true, "arrow_down": true,
		"shift+delete": true, "f2": true, "page_down": true,
		"a": false, "shift+a": false, "delete": false, "backspace": false, "arrow_left": false, "home": false,
	} {
		if got := surfui.Giveable(key); got != want {
			t.Errorf("Giveable(%q) = %v", key, got)
		}
	}
}

// The Compact looks' nodes: what each new primitive and modifier writes.
func TestCompactNodesHTML(t *testing.T) {
	for _, c := range []struct {
		name string
		n    *surfui.Node
		want string
	}{
		{"Key", surfui.Key("p"), `<kbd id="r" class="u-key">p</kbd>`},
		{"Field.Bare", surfui.Field().Bare(), `<input id="r" class="u-field u-bare" type="text">`},
		{"Button.Bare", surfui.Button("Reset").Bare(), `<button id="r" class="u-btn u-bare" type="button">Reset</button>`},
		{"List.Nav", surfui.List(surfui.Item(surfui.Text("a"))).Nav(),
			`<div id="r" class="u-list u-nav" role="listbox" tabindex="0">` +
				`<div id="r.0" class="u-item" role="option"><span id="r.0.0" class="u-text">a</span></div></div>`},
		// An Item added later is focusable too.
		{"List.Entries", surfui.List(surfui.Item(surfui.Text("a"))).Entries().Add(surfui.Item(surfui.Text("b"))),
			`<div id="r" class="u-list u-entries" role="listbox">` +
				`<div id="r.0" class="u-item" role="option" tabindex="0"><span id="r.0.0" class="u-text">a</span></div>` +
				`<div id="r.1" class="u-item" role="option" tabindex="0"><span id="r.1.0" class="u-text">b</span></div></div>`},
		{"Dialog", surfui.Dialog(surfui.Text("x")), `<div id="r" class="u-dialog" role="dialog"><span id="r.0" class="u-text">x</span></div>`},
		{"Scale", surfui.Box().Scale(80), `<div id="r" class="u-box u-scale" style="--u-zoom:0.8"></div>`},
		{"Scale(100)", surfui.Box().Scale(100), `<div id="r" class="u-box"></div>`},
		{"Scroll", surfui.Box().Scroll(), `<div id="r" class="u-box u-scroll" tabindex="0"></div>`},
		{"Down", surfui.Row().Down(7), `<div id="r" class="u-row u-down" style="--u-dn:7"></div>`},
		{"Down(0)", surfui.Row().Down(0), `<div id="r" class="u-row"></div>`},
		{"Height", surfui.Stack().Height(7), `<div id="r" class="u-stack u-height" style="--u-h:7"></div>`},
		{"Height(-1)", surfui.Stack().Height(-1), `<div id="r" class="u-stack u-height" style="--u-h:0"></div>`},
		{"Compact", surfui.Surface().Density(surfui.Compact), `<div id="r" class="u-surface u-theme-graphite u-compact"></div>`},
		{"Compact, then Roomy", surfui.Surface().Density(surfui.Compact).Density(surfui.Roomy),
			`<div id="r" class="u-surface u-theme-graphite u-roomy"></div>`},
	} {
		surfui.Resolve(c.n, "r")
		if got := surfui.HTML(c.n); got != c.want {
			t.Errorf("%s:\n got %s\nwant %s", c.name, got, c.want)
		}
	}
}

// A Cycle's parts are where its clicks land: CycleOf finds the Cycle,
// its value and options from any of them, and the step each takes.
func TestCycleOf(t *testing.T) {
	c := surfui.Cycle("b", "a", "b", "c")
	surfui.Resolve(c, "c")
	for i, step := range []int{-1, 1, 1} {
		got, value, options, s, ok := surfui.CycleOf(surfui.Kids(c)[i])
		if !ok || got != c || value != "b" || len(options) != 3 || s != step {
			t.Errorf("part %d: %v %q %v %d %v, want the Cycle, b, 3 options, step %d", i, got == c, value, options, s, ok, step)
		}
	}
	if _, _, _, s, ok := surfui.CycleOf(c); !ok || s != 0 {
		t.Errorf("the Cycle itself: step %d, %v", s, ok)
	}
	if _, _, _, _, ok := surfui.CycleOf(surfui.Text("x")); ok {
		t.Error("a Text is no Cycle")
	}
}

// A Field with an OnChange commits (surfui.Commits): it reports its edits and
// gives the program Enter, which commits it; one without doesn't.
func TestCommittingFieldGivesEnter(t *testing.T) {
	f := surfui.Field().OnChange("save")
	plain := surfui.Field()
	root := surfui.Surface(f, plain).Bind("ctrl+s", "Save", "save")
	surfui.Resolve(root, "r")
	if !surfui.Commits(f) || surfui.Commits(plain) || surfui.Commits(surfui.Switch(true).OnChange("x")) {
		t.Fatal("Commits is wrong")
	}
	h := surfui.HTML(f)
	for _, want := range []string{`data-on="input"`, `data-keys="enter=program ctrl+s=program"`} {
		if !strings.Contains(h, want) {
			t.Errorf("no %s in %s", want, h)
		}
	}
	if h := surfui.HTML(plain); strings.Contains(h, "enter") || strings.Contains(h, "data-on") {
		t.Errorf("a field without OnChange: %s", h)
	}
}

// A Select is a button the runtime drives, never a host's <select>: its
// value and a chevron, the value as wide as the widest option, the keys a
// host would scroll with the program's, and Space and Enter too while it
// is drawn open.
func TestSelectIsAButton(t *testing.T) {
	s := surfui.Select("top", "top", "center", "bottom").OnChange("pick")
	surfui.Resolve(surfui.Surface(s), "r")
	if v, o, ok := surfui.SelectOf(s); !ok || v != "top" || !slices.Equal(o, []string{"top", "center", "bottom"}) {
		t.Fatalf("SelectOf: %q %q %v", v, o, ok)
	}
	if _, _, ok := surfui.SelectOf(surfui.Button("x")); ok {
		t.Error("a Button is no Select")
	}
	h := surfui.HTML(s)
	for _, want := range []string{`<button`, `type="button"`, `aria-haspopup="listbox"`, `aria-expanded="false"`,
		`--u-chars:6`, `>top</span>`, "\U000F0140", `data-keys="arrow_up=program`} {
		if !strings.Contains(h, want) {
			t.Errorf("no %s in %s", want, h)
		}
	}
	if strings.Contains(h, "<select") || strings.Contains(h, "enter=program") {
		t.Errorf("closed: %s", h)
	}
	surfui.OpenSelect(s)
	h = surfui.HTML(s)
	for _, want := range []string{`aria-expanded="true"`, "\U000F0143", `enter=program space=program`} {
		if !strings.Contains(h, want) {
			t.Errorf("open: no %s in %s", want, h)
		}
	}
}

// A document links surf's stylesheet, then the host's own in their
// order, so that the host's win; with none, surf's alone.
func TestDocumentSheets(t *testing.T) {
	root := surfui.Surface(surfui.Text("a"))
	surfui.Resolve(root, "r")
	const head = `<!doctype html><html><head><meta charset="utf-8"><link rel="stylesheet" href="cid:` + surfui.StyleID + `">`
	body := `</head><body>` + surfui.HTML(root) + `</body></html>`
	if got := surfui.Document(root); got != head+body {
		t.Errorf("no sheets:\n got %s\nwant %s", got, head+body)
	}
	want := head + `<link rel="stylesheet" href="cid:plx-theme"><link rel="stylesheet" href="cid:a&amp;b">` + body
	if got := surfui.Document(root, "cid:plx-theme", "cid:a&b"); got != want {
		t.Errorf("two sheets:\n got %s\nwant %s", got, want)
	}
}

// A Select's list takes the density and theme nearest the Select: a
// compact canvas's in a dense program (the gallery), its own surface's
// over the root's.
func TestSelectListNearest(t *testing.T) {
	sel := surfui.Select("a", "a", "b").Key("sel")
	canvas := surfui.Surface(sel).Density(surfui.Compact).Theme("paper")
	root := surfui.Surface(canvas).Density(surfui.Dense).Theme("nord")
	ix := surfui.Resolve(root, "r")
	var n *surfui.Node
	for _, m := range ix {
		if m == sel {
			n = m
		}
	}
	if n == nil {
		t.Fatal("the select isn't in the tree")
	}
	cl := surfui.Classes(surfui.SelectList(n, "a", []string{"a", "b"}, 0))
	for _, c := range []string{"u-compact", "u-theme-paper"} {
		if !slices.Contains(cl, c) {
			t.Errorf("the list's root has no %s: %q", c, cl)
		}
	}
	for _, c := range []string{"u-dense", "u-theme-nord"} {
		if slices.Contains(cl, c) {
			t.Errorf("the list's root has the root's %s: %q", c, cl)
		}
	}
}

// A Select's list is a surface's root in the theme and density of the
// Select's, a box of its options, the picked one and the one the keys are
// on marked, each a click the runtime hears.
func TestSelectList(t *testing.T) {
	root := surfui.Surface().Density(surfui.Compact).Theme("nord")
	l := surfui.SelectList(root, "center", []string{"top", "center", "bottom"}, 2)
	ix := surfui.Resolve(l, surfui.ListID)
	for _, c := range []string{"u-surface", "u-compact", "u-theme-nord"} {
		if !slices.Contains(surfui.Classes(l), c) {
			t.Errorf("the list's root has no %s: %q", c, surfui.Classes(l))
		}
	}
	for i, want := range []struct {
		text, class string
	}{{"top", ""}, {"center", "is-picked"}, {"bottom", "is-hi"}} {
		o := ix[surfui.OptionID(i)]
		if o == nil {
			t.Fatalf("no option %d (%s)", i, surfui.OptionID(i))
		}
		if got := surfui.TextOf(o); got != want.text {
			t.Errorf("option %d is %q, want %q", i, got, want.text)
		}
		cl := surfui.Classes(o)
		if want.class != "" && !slices.Contains(cl, want.class) || want.class == "" && slices.ContainsFunc(cl, func(c string) bool { return strings.HasPrefix(c, "is-") }) {
			t.Errorf("option %d's classes %q, want %q", i, cl, want.class)
		}
		if h := surfui.HTML(o); !strings.Contains(h, `data-on="click"`) {
			t.Errorf("option %d hears no click: %s", i, h)
		}
	}
	if v, _ := surfui.AttrOf(ix[surfui.OptionID(1)], "aria-selected"); v != "true" {
		t.Error("the picked option is not aria-selected")
	}
}

// A Dot's shape is a class for the sheet and its glyph for the cells;
// on any other node Ring, Target and Pulse do nothing.
func TestDotShapes(t *testing.T) {
	for _, c := range []struct {
		n            *surfui.Node
		glyph, class string
	}{
		{surfui.Dot(), "●", ""},
		{surfui.Dot().Ring(), "○", "u-ring"},
		{surfui.Dot().Tone(surfui.Error).Target(), "◉", "u-target"},
		{surfui.Dot().Tone(surfui.Info).Pulse(), "●", "u-pulse"}, // the cells' dot is the plain one
	} {
		cl := surfui.Classes(c.n)
		if got := surfui.TextOf(c.n); got != c.glyph || !slices.Contains(cl, "u-dot") || c.class != "" && !slices.Contains(cl, c.class) {
			t.Errorf("%q %q, want %q with u-dot and %q", got, cl, c.glyph, c.class)
		}
	}
	if n := surfui.Glyph("●").Ring().Target(); surfui.TextOf(n) != "●" || slices.Contains(surfui.Classes(n), "u-ring") || slices.Contains(surfui.Classes(n), "u-target") {
		t.Errorf("a Glyph took a dot's shape: %q %q", surfui.TextOf(n), surfui.Classes(n))
	}
	if n := surfui.Text("x").Pulse(); slices.Contains(surfui.Classes(n), "u-pulse") {
		t.Errorf("a Text pulses: %q", surfui.Classes(n))
	}
}

// An element with a menu asks for view (where it is, for a key that opens
// the menu) and secondary.
func TestMenuAsksForViewAndSecondary(t *testing.T) {
	n := surfui.Row(surfui.Text("x")).Menu(surfui.MenuItem{ID: "a", Label: "A", Act: func() {}})
	surfui.Resolve(surfui.Surface(n), "r")
	if got := surfui.DataOn(n); got != "view secondary" {
		t.Errorf("DataOn = %q", got)
	}
	if h := surfui.HTML(n); strings.Count(h, "data-on=") != 1 || !strings.Contains(h, `data-on="view secondary"`) {
		t.Errorf("html: %s", h)
	}
}

// An attribute an app sets and surf computes is written once, merged:
// data-on's tokens, data-keys' bindings (the app's own key wins), a
// style's vars then the app's declarations, and the app's data-steps.
func TestAppAttrsMergeWithComputed(t *testing.T) {
	n := surfui.Row(surfui.Text("x")).
		Menu(surfui.MenuItem{ID: "a", Label: "A", Act: func() {}}).
		Var("p", "1").
		Attr("data-on", "drag drop").Attr("data-steps", "0 1").
		Attr("style", "opacity:.5;").Attr("class", "k").Class("c")
	surfui.Resolve(surfui.Surface(n), "r")
	want := surfui.DataOn(n)
	if want != "view secondary" {
		t.Fatalf("DataOn changed: %q", want)
	}
	h := surfui.HTML(n)
	for _, w := range []string{`data-on="drag drop view secondary"`, `style="--p:1;opacity:.5"`, `class="k u-row c"`, `data-steps="0 1"`} {
		if strings.Count(h, w) != 1 {
			t.Errorf("want one %s in %s", w, h)
		}
	}
	for _, k := range []string{"data-on=", "style=", "data-steps="} {
		if strings.Count(h, k) != 1 {
			t.Errorf("%s written %d times in %s", k, strings.Count(h, k), h)
		}
	}
}
