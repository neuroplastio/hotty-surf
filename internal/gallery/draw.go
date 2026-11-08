package gallery

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	surf "github.com/neuroplastio/hotty-surf"
	"github.com/neuroplastio/hotty-surf/surflive"
	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surfui"
)

// drawn is the picked story as the canvas draws it.
type drawn struct {
	g        *Gallery
	s        *story
	tree     func() *surfui.Node // a story of stories.All's
	view     *surf.View          // a template's, interpreted
	host     *owner              // a partial's story's owner, which gives its props
	state    any                 // a stories file's Go state
	controls []*Control
	at       int      // where its node is among its tree's root's children; -1: the root
	err      string   // why it draws nothing
	errs     []string // what its last render reported
}

// cellsRows are the rows a story's cells UI is drawn in.
const cellsRows = 30

// draw makes s the canvas's story.
func (g *Gallery) draw(s *story) *drawn {
	d := &drawn{g: g, s: s, at: -1}
	switch {
	case s.tree != nil:
		d.tree = s.tree
	case s.widget != nil:
		ws := newWidgetStory(d, s.widget)
		ws.load()
		d.controls = ws.shown()
	case s.file != nil:
		d.fileStory()
	}
	return d
}

// fileStory draws a stories file's story: a screen in its Go state, by
// its package's templates; a partial by a template of its own, its tag
// with its props, drawn in the file's Go state.
func (d *drawn) fileStory() {
	f := d.s.file
	if f.err != nil {
		d.fail(f.err)
		return
	}
	if d.s.at < 0 {
		d.fail(fmt.Errorf("%s.surf has no story", f.name))
		return
	}
	st := &f.stories[d.s.at]
	if err := f.pkg.load(); err != nil {
		d.fail(err)
		return
	}
	var base surf.Instance
	if mk, ok := states[f.name]; ok {
		d.state = mk(st.name)
		if base = f.pkg.Instance(d.state); base == nil {
			d.fail(fmt.Errorf("%s's Go state is no component of package %s", f.name, f.pkg.Name))
			return
		}
	}
	if f.screen() {
		if base == nil || base.Shape() != f.pkg.component(f.of) {
			d.fail(fmt.Errorf("%s.surf: a screen's stories need its Go state, a %s (states)", f.name, f.of))
			return
		}
		d.viewOf(f.pkg.set, newOwner(d, base, nil, false))
		return
	}
	declared, _ := f.pkg.set.Props(f.of)
	props, ctls, err := propsOf(declared, st, f.pkg.shapes)
	if err != nil {
		d.fail(fmt.Errorf("%s.surf: story %q: %v", f.name, st.name, err))
		return
	}
	d.host = newOwner(d, base, props, true)
	d.controls = ctls
	set, err := f.pkg.storySet(hostTemplate(f, st, props), d.host.shape)
	if err != nil {
		d.fail(fmt.Errorf("%s.surf: story %q: %v", f.name, st.name, err))
		return
	}
	d.viewOf(set, d.host)
}

// viewOf draws the story by set's template for inst, interpreted, its
// render's errors kept to show.
func (d *drawn) viewOf(set *surf.Set, inst surf.Instance) {
	v, ok := surf.Component(set, inst).(*surf.View)
	if !ok {
		d.fail(fmt.Errorf("%s draws no view", inst.Shape().Name))
		return
	}
	v.OnError = func(err error) {
		if msg := err.Error(); !slices.Contains(d.errs, msg) {
			d.errs = append(d.errs, msg)
		}
	}
	d.view = v
}

func (d *drawn) fail(err error) { d.err, d.view = err.Error(), nil }

// node is the story's own node in root, its tree: the root, or one of
// its children where something holds it, as a partial's stories file's
// wrapper does, and a widget's story's row (widgetStory.place).
func (d *drawn) node(root *surfui.Node) *surfui.Node {
	at := d.at
	if d.host != nil {
		at = 0
	}
	if ks := surfui.Kids(root); at >= 0 && at < len(ks) {
		return ks[at]
	}
	return root
}

// cells is the story's cells UI: a screen's that has one.
func (d *drawn) cells() surfrun.Cells {
	if d.s.file == nil || !d.s.file.screen() {
		return nil
	}
	c, _ := d.state.(surfrun.Cells)
	return c
}

// resize tells the story's Go state the canvas's size, where it lays out
// by it.
func (d *drawn) resize(cols int) {
	if s, ok := d.state.(surfrun.Sizer); ok && d.host == nil {
		s.Resize(cols, cellsRows)
	}
}

// setDensity tells the story's Go state the density the canvas draws it
// at, where it lays out by it: the toolbar's, whatever the screen's own.
func (d *drawn) setDensity(dn surfui.Density) {
	if s, ok := d.state.(surfrun.DensitySetter); ok && d.host == nil {
		s.SetDensity(dn)
	}
}

// render is the story's tree, drawn at ctx's depth.
func (d *drawn) render(ctx *surflive.Ctx) *surfui.Node {
	d.errs = d.errs[:0]
	switch {
	case d.err != "":
		return nil
	case d.tree != nil:
		return d.tree()
	case d.view != nil:
		d.g.drawing = true
		defer func() { d.g.drawing = false }()
		return d.view.Render(ctx)
	}
	return nil
}

// canvas is the picked story drawn live, in a surface of the toolbar's
// theme, density and width, which scrolls: its own tree, by its own
// handlers, which the gallery's root holds as its child.
type canvas struct{ g *Gallery }

func (c canvas) Render(ctx *surflive.Ctx) *surfui.Node {
	g, d := c.g, c.g.cur
	density := surfui.Density(g.Density)
	var root *surfui.Node
	switch {
	case d == nil:
		root = surfui.Text("No story.").Muted()
	case g.Cells && d.cells() != nil:
		density = surfui.Dense // cells are the terminal's grid
		d.setDensity(density)
		root = d.cellsTree()
	case g.States && g.HasStates():
		// The surface, the stack, the matrix, a row and its stack round
		// each story are five levels.
		root = g.matrixTree(ctx.At(ctx.Depth() + 5))
	default:
		// The surface and the stack round the story are two levels.
		d.setDensity(density)
		root = d.render(ctx.At(ctx.Depth() + 2))
	}
	var s *surfui.Node
	if root != nil && slices.Contains(surfui.Classes(root), "u-surface") {
		s = root // a screen's own surface
	} else {
		s = surfui.Surface(surfui.Stack(surfui.If(root != nil, root)).Pad(1, 0).Key("story"))
	}
	s.Theme(g.Theme).Density(density).Scroll().Width(g.Width).Label("Canvas")
	if g.reloadErr != "" {
		surfui.SetKids(s, append([]*surfui.Node{surfui.Text(g.reloadErr).Tone(surfui.Error).Wrap().Key("reload")}, surfui.Kids(s)...)...)
	}
	if g.States && g.HasStates() {
		return s // each row says what its render reported
	}
	if g.Reference && g.HasReference() {
		s.Add(g.reference())
	} else {
		g.unsendRefs()
	}
	if d != nil && d.err != "" {
		s.Add(surfui.Text(d.err).Tone(surfui.Error).Wrap().Key("err"))
	}
	for i, e := range d.renderErrs() {
		s.Add(surfui.Text(e).Tone(surfui.Error).Small().Wrap().Key("e" + strconv.Itoa(i)))
	}
	return s
}

func (d *drawn) renderErrs() []string {
	if d == nil {
		return nil
	}
	return d.errs
}

// cellsKeys are the keys the cells canvas takes, beside the story's
// bindings: what a cells UI moves and types with. Tab is left to the
// gallery, to leave the canvas by.
var cellsKeys = func() []string {
	ks := []string{"arrow_up", "arrow_down", "arrow_left", "arrow_right", "page_up", "page_down", "home", "end",
		"enter", "escape", "backspace", "space"}
	for r := 'a'; r <= 'z'; r++ {
		ks = append(ks, string(r))
	}
	for r := '0'; r <= '9'; r++ {
		ks = append(ks, string(r))
	}
	return append(ks, "-", ".", "/")
}()

// cellsTree is the story's cells UI as the canvas shows it: its lines,
// each in its colours, and its keys bound on them, in a surface of its
// own as a screen's, so that its lines have the canvas's Width whole:
// the canvas's padding round them took two columns, which every line's
// runs then gave up, each with its own ….
func (d *drawn) cellsTree() *surfui.Node {
	lines := d.cells().DrawCells(d.g.Width, cellsRows)
	st := surfui.Stack().Focusable().Label("Cells").Key("cells")
	for i, l := range lines {
		st.Add(sgrLine(l).Key("l" + strconv.Itoa(i)))
	}
	keys := cellsKeys
	for _, b := range d.bindings() {
		if !slices.Contains(keys, b.Key) {
			keys = append(slices.Clip(keys), b.Key)
		}
	}
	for _, k := range keys {
		st.Bind(k, "", func() { d.cellsKey(k) })
	}
	return surfui.Surface(surfui.Stack(st).Key("story"))
}

// bindings are the keys the story's tree binds, from its keyboard's home
// up, as the runtime finds them with nothing focused: what the cells
// take as the host's do.
func (d *drawn) bindings() []surfui.Binding {
	tree := d.render(nil)
	if tree == nil {
		return nil
	}
	surfui.Resolve(tree, "r")
	n := home(tree)
	if n == nil {
		n = tree
	}
	var out []surfui.Binding
	for ; n != nil; n = surfui.Parent(n) {
		out = append(out, surfui.Bindings(n)...)
	}
	return out
}

// home is the node that takes the keyboard when the surface is shown.
func home(n *surfui.Node) *surfui.Node {
	if surfui.Focused(n) {
		return n
	}
	for _, k := range surfui.Kids(n) {
		if h := home(k); h != nil {
			return h
		}
	}
	return nil
}

// cellsKey is a key on the cells canvas: the story's binding of it, or
// its cells UI's.
func (d *drawn) cellsKey(key string) {
	for _, b := range d.bindings() {
		if b.Key != key {
			continue
		}
		switch act := b.Act.(type) {
		case func():
			act()
		case func(surfrun.Event):
			act(surfrun.Event{Kind: "key", Key: key})
		}
		return
	}
	if c := d.cells(); c != nil {
		c.CellsKey(key)
	}
}

// sgrLine is a cells line as text: its runs, each in the tone, the bold
// and the reverse video its ANSI codes give it, as one Line, cut once at
// its end where it draws wider than its cells: the terminal's font's
// advance in a surface isn't its cell (hottyterm at 1.6 draws 15.35
// device pixels for a 15-pixel cell), which hotty doesn't promise yet.
func sgrLine(line string) *surfui.Node {
	row := surfui.Line()
	var tone surfui.Tone
	bold, rev := false, false
	flush := func(text string) {
		if text == "" {
			return
		}
		t := surfui.Text(text)
		if tone != "" {
			t.Tone(tone)
		}
		if bold {
			t.Bold()
		}
		if rev {
			t.Fill(surfui.Accent)
		}
		row.Add(t)
	}
	for line != "" {
		i := strings.Index(line, "\x1b[")
		if i < 0 {
			flush(line)
			break
		}
		flush(line[:i])
		j := strings.IndexByte(line[i:], 'm')
		if j < 0 {
			break
		}
		for _, code := range strings.Split(line[i+2:i+j], ";") {
			switch code {
			case "0", "":
				tone, bold, rev = "", false, false
			case "1":
				bold = true
			case "22":
				bold = false
			case "7":
				rev = true
			case "27":
				rev = false
			case "39":
				tone = ""
			default:
				tone = sgrTones[code]
			}
		}
		line = line[i+j+1:]
	}
	if len(surfui.Kids(row)) == 0 {
		row.Add(surfui.Text(" "))
	}
	return row
}

// sgrTones are the tones of the cells' ANSI colours (internal/screens/cells.go).
var sgrTones = map[string]surfui.Tone{"35": surfui.Accent, "32": surfui.OK, "33": surfui.Warn, "31": surfui.Error, "36": surfui.Info, "90": surfui.Muted}
