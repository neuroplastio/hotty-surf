package surfrun

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/neuroplastio/hotty-go"

	"github.com/neuroplastio/hotty-surf/surfui"
)

// A Select (surfui.Select) is no host control either. A host may show no list
// for a <select>, and hotty-blitz shows none, so a program that wants one
// draws it (SPEC §10.2): a button, and the list in a surface of its own
// at a higher z, placed by the area the button's click reports (§9), as
// plx's kit does. The surface does that for every Select: it hands its
// host the list's tree and where the Select is (List), and the host
// places it. The runtime places it under the Select (listPlacement).
//
// Closed, the Select's keys reach the program (its data-keys give those
// a host would scroll with): ↓ and ↑ open the list, and Home, End, the
// pages and a letter pick at once. A click, or Space or Enter (the host's
// click from the keyboard), opens the list under it; a key, which carries
// no area, opens it where the host's last view put the Select (its
// data-on asks for view, SPEC §9.5). Open,
// the Select keeps the keyboard and gives the program Space and Enter
// too: ↑ ↓, Home, End and a letter move, Space and Enter pick, Escape
// closes. A click on an option picks it; a click on the Select, Tab, or
// the keyboard going anywhere but the list closes it. Each pick runs the
// Select's OnChange, as a Cycle's step does.

// ListSurface is the name of the surface a Select's open list shows in:
// for a test that reads it (surftest's List).
func (r *Runtime) ListSurface() string { return r.listSurface }

// selectList is the open list of a Select.
type selectList struct {
	id      string // the Select's
	value   string
	options []string
	at      int          // the option the keys are on
	area    *hotty.Area  // the Select's cells, where its click said
	tree    *surfui.Node // the list's tree as last sent, nil before
}

// selectClick is a click on the Select n: it opens its list, or closes
// it if it is open.
func (s *Surface) selectClick(n *surfui.Node, e hotty.Event) {
	if s.list != nil && s.list.id == surfui.IDOf(n) {
		s.closeList(false)
		return
	}
	s.openList(n, e.Area)
}

// openList opens n's list, under area if a click said where n is, else
// where the last view did.
func (s *Surface) openList(n *surfui.Node, area *hotty.Area) {
	value, options, ok := surfui.SelectOf(n)
	if !ok || len(options) == 0 || !s.shown {
		return
	}
	if _, off := surfui.AttrOf(n, "disabled"); off {
		return
	}
	if a, ok := s.areas[surfui.IDOf(n)]; ok && area == nil {
		area = &a
	}
	s.closeList(false)
	s.list = &selectList{id: surfui.IDOf(n), value: value, options: options, at: max(slices.Index(options, value), 0), area: area}
	s.due = true
}

// closeList closes the open list, if there is one; refocus gives the
// Select the keyboard back, which a press on the list took.
func (s *Surface) closeList(refocus bool) {
	l := s.list
	if l == nil {
		return
	}
	s.list = nil
	s.due = true
	if l.tree != nil {
		s.listGone = true
	}
	if refocus && s.index[l.id] != nil {
		s.focus = l.id
		s.focused = l.id
	}
}

// DropList closes the open list at once, picking nothing, and reports
// whether its surface is up, which the host then deletes itself: for a
// host that takes the surface away (it ends, shows another model, or
// goes to the cells), whose surface moved (a resize), or that lost the
// list's surface (ENOENT). The next Update draws the Select closed.
func (s *Surface) DropList() (shown bool) {
	shown = s.listGone || s.list != nil && s.list.tree != nil
	if s.list != nil {
		s.due = true
	}
	s.list, s.listPress, s.listGone = nil, false, false
	return shown
}

// pick picks option i of the open list: it closes, the Select has the
// keyboard again, and its OnChange runs if the pick is another value.
func (s *Surface) pick(i int) {
	l := s.list
	if l == nil || i < 0 || i >= len(l.options) {
		return
	}
	s.closeList(true)
	s.listPress = false
	if n := s.index[l.id]; n != nil {
		s.selectTo(n, l.options[i])
	}
}

// selectTo runs the Select n's OnChange with value, unless it is n's
// value already or n is disabled.
func (s *Surface) selectTo(n *surfui.Node, value string) {
	now, _, ok := surfui.SelectOf(n)
	if !ok || now == value {
		return
	}
	if _, off := surfui.AttrOf(n, "disabled"); off {
		return
	}
	if act, ok := surfui.Handler(n, "change"); ok {
		s.do(act, Event{Kind: "change", Target: surfui.IDOf(n), Value: value})
	}
}

// ListEvent routes an event of the open list's surface: a press, which
// takes the keyboard from the surface (the blur it causes there is no
// leaving), and closes the list where it is on no option; and a click on
// an option, which picks it. It reports whether the surface took the
// event, and whether an Update is due.
func (s *Surface) ListEvent(e hotty.Event) (took, redraw bool) {
	s.fresh()
	l := s.list
	if l == nil {
		return false, s.due
	}
	switch e.Kind {
	case hotty.EventPress:
		s.listPress = true
		if l.option(e.Target) < 0 {
			s.closeList(true)
		}
		return true, s.due
	case hotty.EventClick:
		if i := l.option(e.Target); i >= 0 {
			s.pick(i)
			return true, s.due
		}
	}
	return false, s.due
}

// option is the index of the option id is, or is in; -1 if none.
func (l *selectList) option(id string) int {
	for i := range l.options {
		if o := surfui.OptionID(i); id == o || strings.HasPrefix(id, o+".") {
			return i
		}
	}
	return -1
}

// selectKey is key for a Select: the open list's, whatever has the
// keyboard, or the focused Select's. It reports whether the Select took
// it.
func (s *Surface) selectKey(key string) bool {
	if !s.shown {
		return false
	}
	if l := s.list; l != nil {
		last := len(l.options) - 1
		switch key {
		case "arrow_up":
			l.at = max(l.at-1, 0)
		case "arrow_down":
			l.at = min(l.at+1, last)
		case "home", "page_up":
			l.at = 0
		case "end", "page_down":
			l.at = last
		case "enter", "space":
			s.pick(l.at)
		case "escape":
			s.closeList(true)
		default:
			ch, ok := character(key)
			if !ok {
				return false
			}
			l.at = byLetter(l.options, l.at, ch)
		}
		s.due = true
		return true
	}
	n := s.focusedSelect()
	if n == nil {
		return false
	}
	value, options, _ := surfui.SelectOf(n)
	at, last := slices.Index(options, value), len(options)-1
	switch key {
	case "arrow_down", "arrow_up":
		// As a select-only combobox's arrows (WAI-ARIA APG): they open
		// the list, on the picked option.
		s.openList(n, nil)
		return true
	case "home", "page_up":
		at = 0
	case "end", "page_down":
		at = last
	case "enter", "space":
		// A host gives these to the program only while the list is open;
		// one that gives them anyway opens it.
		s.openList(n, nil)
		return true
	default:
		ch, ok := character(key)
		if !ok {
			return false
		}
		at = byLetter(options, at, ch)
	}
	if at >= 0 {
		s.selectTo(n, options[at])
	}
	return true
}

// focusedSelect is the Select with the keyboard, where it takes keys: it
// has options, and is not disabled; nil for none.
func (s *Surface) focusedSelect() *surfui.Node {
	if !s.shown || s.focused == "" {
		return nil
	}
	n := s.index[s.focused]
	_, options, ok := surfui.SelectOf(n)
	if !ok || len(options) == 0 {
		return nil
	}
	if _, off := surfui.AttrOf(n, "disabled"); off {
		return nil
	}
	return n
}

// listKeys are the keys an open list takes besides a character, and
// selectKeys a focused Select's (selectKey).
var (
	listKeys   = []string{"arrow_up", "arrow_down", "home", "page_up", "end", "page_down", "enter", "space", "escape"}
	selectKeys = listKeys[:len(listKeys)-1]
)

// byLetter is the next option after at that starts with ch, ignoring
// case, round to the first; at itself where none does.
func byLetter(options []string, at int, ch rune) int {
	for i := 1; i <= len(options); i++ {
		j := (at + i + len(options)) % len(options)
		if first, _ := utf8.DecodeRuneInString(options[j]); unicode.ToLower(first) == ch {
			return j
		}
	}
	return at
}

// markList draws the open list's Select open in the tree ix indexes, and
// takes its value and options from there; a Select that is gone, or
// disabled, closes its list.
func (s *Surface) markList(ix surfui.Index) {
	l := s.list
	if l == nil {
		return
	}
	n := ix[l.id]
	value, options, ok := surfui.SelectOf(n)
	if !ok || len(options) == 0 {
		s.closeList(false)
		return
	}
	if _, off := surfui.AttrOf(n, "disabled"); off {
		s.closeList(false)
		return
	}
	surfui.OpenSelect(n)
	l.value, l.options = value, options
	l.at = min(l.at, len(options)-1)
}

// showList is what brings the list's surface up to the open list: a new
// list when it opens, its ops after; nil for nothing.
func (s *Surface) showList() *List {
	gone := s.listGone
	s.listGone = false
	l := s.list
	if l == nil {
		if gone {
			return &List{Closed: true}
		}
		return nil
	}
	from := s.index[l.id]
	if from == nil {
		from = s.prev
	}
	tree := surfui.SelectList(from, l.value, l.options, l.at)
	surfui.Resolve(tree, surfui.ListID)
	widest := 0
	for _, o := range l.options {
		widest = max(widest, utf8.RuneCountInString(o))
	}
	out := &List{Closed: gone, Tree: tree, New: l.tree == nil, Anchor: l.area,
		Cols: widest + 6, Rows: len(l.options) + 1, At: l.at}
	if !out.New {
		out.Ops = surfui.Diff(l.tree, tree)
	}
	l.tree = tree
	if !out.Closed && !out.New && len(out.Ops) == 0 {
		return nil
	}
	return out
}

// list writes what changed of the open list to b: its surface deleted,
// its document and its placement, or its delta.
func (r *Runtime) list(b *strings.Builder, l *List) {
	if l == nil {
		return
	}
	if l.Closed {
		b.WriteString(hotty.Del(r.listSurface, hotty.Q(hotty.NoReply)))
	}
	switch {
	case l.Tree == nil:
	case l.New:
		x, y, p := r.listPlacement(l)
		r.listScrolls = p.Rows < l.Rows
		if r.listScrolls {
			b.WriteString(hotty.Doc(r.listSurface, surfui.Document(l.Tree), hotty.Scroll(hotty.ScrollVertical)))
		} else {
			b.WriteString(hotty.Doc(r.listSurface, surfui.Document(l.Tree)))
		}
		b.WriteString(hotty.PlaceAt(r.listSurface, x, y, p))
	case len(l.Ops) > 0:
		ops := l.Ops
		if r.listScrolls {
			ops = append(ops, hotty.OpReveal(surfui.OptionID(l.At), ""))
		}
		b.WriteString(hotty.Delta(r.listSurface, ops))
	}
}

// dropList takes the open list off at once: the surface is going, or
// being sent again, or it moved.
func (r *Runtime) dropList() {
	if r.s.DropList() {
		r.write(hotty.Del(r.listSurface, hotty.Q(hotty.NoReply)))
	}
}

// listPlacement is where the list goes: under its Select, as wide as it
// at least, or above it where there is more room there; in the middle of
// the surface where no click said where the Select is. It is the rows it
// wants, and scrolls where that doesn't fit.
func (r *Runtime) listPlacement(l *List) (x, y int, p hotty.Placement) {
	cols, rows := l.Cols, l.Rows
	top, h := r.opts.Top, r.surfaceRows()
	if a := l.Anchor; a != nil {
		cols = max(cols, a.W)
		x = a.Col
		below := r.rows - (top + a.Row + a.H)
		above := top + a.Row
		if rows <= below || below >= above {
			y, rows = top+a.Row+a.H, min(rows, below)
		} else {
			rows = min(rows, above)
			y = top + a.Row - rows
		}
	} else {
		rows = min(rows, h)
		x, y = (r.cols-cols)/2, top+(h-rows)/2
	}
	cols = max(min(cols, r.cols), 1)
	x = max(min(x, r.cols-cols), 0)
	rows = max(rows, 1)
	y = max(min(y, r.rows-rows), 0)
	return x, y, hotty.Placement{Cols: cols, Rows: rows, Z: 1, Press: hotty.On}
}
