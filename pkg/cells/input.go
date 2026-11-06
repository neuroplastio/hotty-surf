package cells

import (
	"strings"
	"time"

	"github.com/neuroplastio/hotty-go"
	"github.com/neuroplastio/myelin/myelinkey"

	"github.com/neuroplastio/hotty-surf/surfui"
)

// Input is the terminal's, at the cells of the last Draw. Each call
// reports whether the renderer or the program took it, and whether the
// screen may have changed: Draw again (its Gen says whether it did).
//
// A key goes as SPEC §10.2 has a host take it while the surface has the
// keyboard. Tab and Shift+Tab move the focus through what takes it, in
// the tree's order, and past the last or the first leave the surface
// (blur). A focused text field edits with its keymap (hotty.Resolve, the
// data-keys from the root down to it), as hottyedit.Field does, and
// reports each edit as input; a key it doesn't take is the program's. On
// another element, a key its keymap (the data-keys from the root down)
// gives the program is the program's; then Space and Enter press a button;
// then a key its keymap binds to a scroll action scrolls, and a key a
// browser scrolls with scrolls the nearest Scroll box from it outward, or
// the page, that can move; and every other key is the program's
// (Program.Key). With nothing focused, each key but Tab scrolls the page
// where it can, or is the program's.

// Key takes a key, by myelin's name (SPEC §10.4: "a", "shift+tab",
// "ctrl+a", "arrow_down", "enter").
func (r *Renderer) Key(name string) (taken, redraw bool) {
	r.changed = false
	if r.menu != nil {
		// The menu has the keyboard: the program takes every key (the
		// Keys it lists are the menu's only), but ctrl+c, which closes
		// the menu and goes on. A key may move the lit row, so the row the
		// pointer is on is forgotten: its next motion, even within the
		// row, takes the lit row back, as a native menu's does.
		r.menuRow = nil
		if name == "ctrl+c" {
			r.closeMenu()
		} else if r.toProgram(name) {
			return true, true
		}
	}
	taken = r.key(name)
	return taken, r.changed || taken
}

func (r *Renderer) key(name string) bool {
	k, err := myelinkey.Parse(name)
	if err != nil {
		return false
	}
	mods, base, kind := myelinkey.Split(k)
	shiftOnly := mods&^myelinkey.Shift == 0
	if base == "tab" && kind == myelinkey.Physical && shiftOnly {
		r.settle(r.tab(mods != 0))
		r.changed = true
		return true
	}
	n := r.index[r.focused]
	if r.focused == "" || n == nil {
		if r.scrollKey(nil, name) {
			return true
		}
		return r.toProgram(name)
	}
	keymaps := r.keymaps(n)
	if surfui.IsTextField(n) {
		return r.fieldKey(n, name, keymaps)
	}
	km := hotty.ParseKeymap(r.Platform, strings.Join(keymaps, " "))
	if km.Program(name) {
		return r.toProgram(name)
	}
	if shiftOnly && (base == "space" || base == "enter") && presses(n) {
		bits := 0
		if mods&myelinkey.Shift != 0 {
			bits = hotty.ModShift
		}
		r.settle(r.clickOn(n, 1, bits))
		r.changed = true
		return true
	}
	if a := km.Scroll(name); a != "" && r.scrolls() {
		r.scrollBy(n, a)
		return true // used, whether or not anything moved (SPEC §10.2)
	}
	if r.scrollKey(n, name) {
		return true
	}
	return r.toProgram(name)
}

// route is what a call of the program's reported: an update follows one
// that is due.
func (r *Renderer) route(took, due bool) bool {
	r.settle(due)
	return took
}

// toProgram gives the program a key.
func (r *Renderer) toProgram(name string) bool {
	took, due := r.p.Key(name)
	if due {
		r.changed = true
	}
	return r.route(took, due)
}

// keymaps are the data-keys of the elements from the root down to n.
func (r *Renderer) keymaps(n *surfui.Node) []string {
	var out []string
	for q := n; q != nil; q = surfui.Parent(q) {
		if v := surfui.DataKeys(q); v != "" {
			out = append(out, v)
		}
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// presses reports whether Space and Enter click n: a button, a link, a
// summary.
func presses(n *surfui.Node) bool {
	switch surfui.Tag(n) {
	case "button", "summary":
		return true
	case "a":
		_, ok := surfui.AttrOf(n, "href")
		return ok
	}
	return false
}

// fieldKey is a key in the focused text field.
func (r *Renderer) fieldKey(n *surfui.Node, name string, keymaps []string) bool {
	st := r.fields[r.focused]
	if st == nil {
		return r.toProgram(name)
	}
	km := hotty.Resolve(r.Platform, surfui.Tag(n) == "textarea", keymaps...)
	if km.Prefix(name) {
		return r.toProgram(name)
	}
	a, changed := st.ed.Key(km, name)
	switch {
	case a == "":
		return r.toProgram(name)
	case a == hotty.Paste:
		// The clipboard is the program's to read: it calls Paste.
		return r.toProgram(name)
	case a == hotty.Copy, a == hotty.Cut:
		text, ok := selected(&st.ed)
		if !ok || a == hotty.Copy && r.Copy == nil {
			// Nothing to give, or no clipboard: the key is the program's.
			return r.toProgram(name)
		}
		if r.Copy != nil {
			r.Copy(text)
		}
		if a == hotty.Copy {
			return true
		}
		if st.ed.Do(hotty.DeleteCharBackward) {
			r.settle(r.edited())
		}
	case changed:
		r.settle(r.edited())
	}
	r.changed = true
	return true
}

// Paste types text into the focused text field, as a paste does there:
// a field of one line leaves its newlines out. Elsewhere nothing takes it.
func (r *Renderer) Paste(text string) (taken, redraw bool) {
	r.changed = false
	n := r.index[r.focused]
	st := r.fields[r.focused]
	if n == nil || st == nil {
		return false, false
	}
	if surfui.Tag(n) != "textarea" {
		text = strings.NewReplacer("\r\n", "", "\n", "", "\r", "").Replace(text)
	}
	if st.ed.Type(text) {
		r.settle(r.edited())
	}
	return true, true
}

// Focus is the terminal's focus coming (in) or going: going, the surface
// loses the keyboard, as a host's does (blur, and the field commits);
// coming back, what had it has it again.
func (r *Renderer) Focus(in bool) (taken, redraw bool) {
	r.changed = false
	switch {
	case !in && r.kb:
		r.away = r.focused
		r.settle(r.blur())
	case in && r.away != "" && r.index[r.away] != nil:
		r.settle(r.take(r.away, true))
		r.away = ""
	default:
		return false, false
	}
	return true, true
}

// tab moves the focus to the next element that takes it, or the
// previous, from the focused one, in tree order; past the last, or before
// the first, the surface loses the keyboard.
func (r *Renderer) tab(back bool) bool {
	var order []*surfui.Node
	at := -1
	todo := []*surfui.Node{r.tree}
	for len(todo) > 0 && r.tree != nil {
		n := todo[len(todo)-1]
		todo = todo[:len(todo)-1]
		if surfui.Tag(n) == "" {
			continue
		}
		if surfui.IDOf(n) == r.focused && r.focused != "" {
			at = len(order)
		}
		if focusable(n) && tabbable(n) {
			order = append(order, n)
		}
		kids := surfui.Kids(n)
		for i := len(kids) - 1; i >= 0; i-- {
			todo = append(todo, kids[i])
		}
	}
	in := at >= 0 && at < len(order) && surfui.IDOf(order[at]) == r.focused
	i := at
	switch {
	case at < 0 && back:
		i = len(order) - 1
	case at < 0:
		i = 0
	case back:
		i--
	case in:
		i++
	}
	if i < 0 || i >= len(order) {
		return r.blur()
	}
	return r.take(surfui.IDOf(order[i]), true)
}

// tabbable reports whether Tab reaches n: not a negative tabindex.
func tabbable(n *surfui.Node) bool {
	t, ok := surfui.AttrOf(n, "tabindex")
	return !ok || !strings.HasPrefix(strings.TrimSpace(t), "-")
}

// The mouse.

// Press is a press of the main button at x, y, with mods held (hotty's
// bits, hotty.ModShift…): what takes the keyboard there takes it, or the
// surface loses it; a press on the open list goes to the list. The click
// comes with the Release.
func (r *Renderer) Press(x, y, mods int) (taken, redraw bool) {
	r.changed = false
	r.press = nil
	if r.menuPress(x, y, true) {
		return true, true
	}
	h, ok := r.hitAt(x, y)
	if ok && h.list {
		_, due := r.p.ListEvent(hotty.Event{Kind: hotty.EventPress, Surface: surfui.ListID, Target: surfui.IDOf(h.n), Mods: mods})
		if r.kb {
			// The list's surface takes the keyboard from the main one.
			due = r.blur() || due
		}
		r.press = &pressed{n: h.n, list: true, mods: mods}
		r.settle(due)
		return true, true
	}
	if r.over != nil && r.overRect().has(x, y) {
		// The list's own room, which no option holds.
		_, due := r.p.ListEvent(hotty.Event{Kind: hotty.EventPress, Surface: surfui.ListID, Mods: mods})
		r.settle(due)
		return true, true
	}
	var n *surfui.Node
	if ok {
		n = h.n
	}
	r.drag = nil
	started, due := false, false
	if ok {
		// A press on a drag handle starts its drag first (SPEC §9.1).
		started, due = r.startDrag(n, x, y, mods)
	}
	if f := closest(n, focusable); f != nil {
		due = r.take(surfui.IDOf(f), true) || due
	} else if r.kb {
		// A press on what takes no focus gives the keyboard back.
		due = r.blur() || due
	}
	r.press = &pressed{n: closest(n, reportsClick), mods: mods}
	r.settle(due)
	return ok || due || started, true
}

// overRect is the open list's box.
func (r *Renderer) overRect() *rect {
	o := r.over
	return &rect{o.x, o.y, o.w, o.h}
}

// Release is the release of the button pressed: over what the press
// reports, its click (SPEC §9), counted in a run of clicks on it. During
// a drag, dragend first, and the click only where the drag ended on the
// handle it began on (drag.go).
func (r *Renderer) Release(x, y, mods int) (taken, redraw bool) {
	r.changed = false
	if r.menuRelease(x, y) {
		return true, true
	}
	p := r.press
	r.press = nil
	if r.drag != nil {
		clicks, due := r.endDrag(x, y, mods)
		r.settle(due)
		if !clicks {
			r.click.id = "" // a drag that ends elsewhere ends the run of clicks too
			return true, true
		}
		r.release(p, x, y, mods)
		return true, true
	}
	return r.release(p, x, y, mods)
}

// release is Release's click: p's, where it is over what p reports.
func (r *Renderer) release(p *pressed, x, y, mods int) (taken, redraw bool) {
	if p == nil || p.n == nil {
		return false, false
	}
	h, ok := r.hitAt(x, y)
	if !ok || h.list != p.list || !holds(surfui.IDOf(p.n), h.n) {
		return false, false
	}
	if p.list {
		_, due := r.p.ListEvent(hotty.Event{Kind: hotty.EventClick, Surface: surfui.ListID, Target: surfui.IDOf(p.n), Count: 1, Mods: mods})
		r.settle(due)
		return true, true
	}
	// The press may have drawn the tree anew: the element now.
	n := r.index[surfui.IDOf(p.n)]
	if n == nil {
		return false, false
	}
	r.settle(r.clickOn(n, r.count(n), mods))
	return true, true
}

// count is a click's place in its run of clicks on n: one more within
// half a second of the last.
func (r *Renderer) count(n *surfui.Node) int {
	now := time.Now
	if r.Now != nil {
		now = r.Now
	}
	t := now()
	id := surfui.IDOf(n)
	if r.click.id == id && t.Sub(r.click.at) < 500*time.Millisecond {
		r.click.count++
	} else {
		r.click.count = 1
	}
	r.click.id, r.click.at = id, t
	return r.click.count
}

// clickOn sends the click n reports, as a host does: its value, and the
// cells it takes.
func (r *Renderer) clickOn(n *surfui.Node, count, mods int) bool {
	e := hotty.Event{Kind: hotty.EventClick, Target: surfui.IDOf(n), Count: count, Mods: mods}
	if v, ok := surfui.AttrOf(n, "value"); ok {
		e.Value = v
	}
	if b := r.boxOf(surfui.IDOf(n)); b != nil {
		e.Area = &hotty.Area{Col: b.x, Row: b.y - r.offsets[""], W: b.w, H: b.h}
	}
	_, due := r.p.Event(e)
	return due
}

// hitAt is what a press at x, y acts on: the hit on top there.
func (r *Renderer) hitAt(x, y int) (hit, bool) {
	r.ensure()
	for i := len(r.hits) - 1; i >= 0; i-- {
		if r.hits[i].r.has(x, y) {
			return r.hits[i], true
		}
	}
	return hit{}, false
}

// ensure draws, if nothing has been drawn, so that input has cells.
func (r *Renderer) ensure() {
	if !r.drawn {
		r.Draw(max(r.cols, 80), max(r.rows, 24))
	}
}

// closest is n or the nearest element around it for which ok holds.
func closest(n *surfui.Node, ok func(*surfui.Node) bool) *surfui.Node {
	for ; n != nil; n = surfui.Parent(n) {
		if surfui.Tag(n) != "" && ok(n) {
			return n
		}
	}
	return nil
}

// holds reports whether n is the element with id outer or in it: by id,
// since the tree may be new since the element was found.
func holds(outer string, n *surfui.Node) bool {
	for ; n != nil; n = surfui.Parent(n) {
		if surfui.Tag(n) != "" && surfui.IDOf(n) == outer {
			return true
		}
	}
	return false
}

// Wheel is the wheel at x, y, by notches: down where positive, three
// rows a notch. It scrolls the innermost Scroll box there that can move
// that way, or one around it, or the page.
func (r *Renderer) Wheel(x, y, notches int) (taken, redraw bool) {
	r.changed = false
	if r.menu != nil {
		r.closeMenu() // and the wheel goes on
	}
	r.ensure()
	if !r.scrolls() || notches == 0 {
		return false, false
	}
	by := notches * 3
	if o := r.over; o != nil && r.overRect().has(x, y) {
		if o.f&fScroll != 0 {
			id := surfui.IDOf(o.n)
			v := o.view(0)
			off := r.clamp(r.offsets[id]+by, o.contentH, v.h)
			moved := off != r.offsets[id]
			r.offsets[id] = off
			return true, moved
		}
		return true, false
	}
	for i := len(r.ports) - 1; i >= 0; i-- {
		pt := r.ports[i]
		if pt.view.has(x, y) && r.move(pt, by) {
			return true, true
		}
	}
	return false, false
}

// move scrolls the view pt by rows, and reports whether it moved.
func (r *Renderer) move(pt port, by int) bool {
	content, view := r.extent(pt)
	off := r.clamp(r.offsets[pt.id]+by, content, view)
	if off == r.offsets[pt.id] {
		return false
	}
	r.offsets[pt.id] = off
	r.changed = true
	return true
}

// extent is what a view scrolls: its content's rows, and its own.
func (r *Renderer) extent(pt port) (content, view int) {
	if pt.id == "" {
		return r.root.h, r.rows
	}
	return pt.b.contentH, pt.b.view(0).h
}

// The keys a browser scrolls with, and what they do.
var scrollKeys = map[string]hotty.Action{
	"arrow_up": hotty.ScrollUp, "arrow_down": hotty.ScrollDown,
	"page_up": hotty.ScrollPageUp, "page_down": hotty.ScrollPageDown,
	"space": hotty.ScrollPageDown, "shift+space": hotty.ScrollPageUp,
	"home": hotty.ScrollStart, "end": hotty.ScrollEnd,
}

// scrollKey scrolls with a key a browser scrolls with, from n outward
// (the page with n nil), and reports whether anything moved: a key that
// moves nothing goes on to the program (SPEC §5.3).
func (r *Renderer) scrollKey(n *surfui.Node, name string) bool {
	a, ok := scrollKeys[name]
	if !ok || !r.scrolls() {
		return false
	}
	return r.scrollBy(n, a)
}

// scrollBy does the scroll action a to the nearest view from n outward
// that can move that way, and reports whether one did.
func (r *Renderer) scrollBy(n *surfui.Node, a hotty.Action) bool {
	r.ensure()
	for _, pt := range r.viewsFrom(n) {
		content, view := r.extent(pt)
		var by int
		switch a {
		case hotty.ScrollUp:
			by = -1
		case hotty.ScrollDown:
			by = 1
		case hotty.ScrollPageUp:
			by = -max(view*7/8, 1)
		case hotty.ScrollPageDown:
			by = max(view*7/8, 1)
		case hotty.ScrollHalfPageUp:
			by = -max(view/2, 1)
		case hotty.ScrollHalfPageDown:
			by = max(view/2, 1)
		case hotty.ScrollStart:
			by = -content
		case hotty.ScrollEnd:
			by = content
		default:
			return false
		}
		if r.move(pt, by) {
			return true
		}
	}
	return false
}

// viewsFrom are the views that scroll from n outward, the page last.
func (r *Renderer) viewsFrom(n *surfui.Node) []port {
	var out []port
	for q := n; q != nil; q = surfui.Parent(q) {
		if !hasClass(q, "u-scroll") {
			continue
		}
		for _, pt := range r.ports {
			if pt.id == surfui.IDOf(q) {
				out = append(out, pt)
				break
			}
		}
	}
	if r.root != nil && r.root.h > r.rows {
		out = append(out, port{id: "", view: rect{0, 0, r.cols, r.rows}, b: r.root})
	}
	return out
}

// At is the id of the innermost element drawn at x, y, whether or not
// it acts on a press: what a host's event at that cell would target, as
// a host that draws an element's context menu asks for it at the cell
// the user pressed (surfrun.Surface.Menu). Of siblings drawn over each
// other, the one drawn last is. It is "" where the surface draws
// nothing, or a Select's open list does, which is the runtime's.
func (r *Renderer) At(x, y int) string {
	r.ensure()
	if h, ok := r.hitAt(x, y); ok && h.list || r.over != nil && r.overRect().has(x, y) {
		return ""
	}
	if vis, ok := r.rects[r.tree]; r.tree == nil || !ok || !vis.has(x, y) {
		return ""
	}
	n := r.tree
	for {
		var in *surfui.Node
		for _, k := range surfui.Kids(n) {
			if vis, ok := r.rects[k]; ok && vis.has(x, y) {
				in = k
			}
		}
		if in == nil {
			return surfui.IDOf(n)
		}
		n = in
	}
}

// Where is a cell a press on reaches the element id: inside it, where
// what acts on a press there is what a press on it reaches (it, or the
// nearest element around it that acts). ok is false where it is not on
// the screen. For a test, or a host that acts by id.
func (r *Renderer) Where(id string) (x, y int, ok bool) {
	r.ensure()
	var n *surfui.Node
	if r.over != nil && strings.HasPrefix(id, surfui.ListID+".") {
		walk(r.over, func(k *box) bool {
			if surfui.Tag(k.n) != "" && surfui.IDOf(k.n) == id {
				n = k.n
				return false
			}
			return true
		})
	} else {
		n = r.index[id]
	}
	vis, seen := r.rects[n]
	if n == nil || !seen {
		return 0, 0, false
	}
	want := closest(n, acts)
	for yy := vis.y; yy < vis.y+vis.h; yy++ {
		for xx := vis.x; xx < vis.x+vis.w; xx++ {
			top, ok := r.hitAt(xx, yy)
			if ok && top.n == want || !ok && want == nil {
				return xx, yy, true
			}
		}
	}
	return 0, 0, false
}
