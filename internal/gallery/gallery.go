// Package gallery is a live browser of hotty-surf's stories: each widget
// of the vocabulary with a control for every attribute it takes, the
// stories of stories.All, and those of the stories files
// (internal/stories/*.surf), the composites' and the screens'. The picked story is
// drawn live in a canvas, at the toolbar's theme, density and width, and
// takes the keyboard and the pointer as it does in a program; the calls
// its handlers make are logged under it, and its props are controls
// beside it. The toolbar's States switch draws it in each of its states
// instead, one under another (matrix.go). A story's canvas is its width
// where the window has room for it beside the stories, and narrower
// where it hasn't, so that the stories stay; a width the user picks is
// kept whatever the window's, and the frame says what it leaves out
// (frame.go). Watching a checkout, the gallery reads its stories files,
// templates and stylesheet from there, and again when they change
// (source.go). gallery.surf draws it; cmd/surfgallery runs it.
package gallery

//go:generate go run ../../cmd/surfgen

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"neuroplast.io/hotty-go"

	"neuroplast.io/hotty-surf/internal/stories"
	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfrun"
	"neuroplast.io/hotty-surf/surfui"
)

// Gallery is the gallery's state.
type Gallery struct {
	Filter    string // what the sidebar's list is narrowed to
	Sel       string // the picked story's id
	Theme     string // the canvas's
	Density   string
	Width     int  // the canvas's width in cells
	Cells     bool // the story drawn by its cells UI
	States    bool // the story drawn in each of its states (matrix.go)
	Reference bool // the gate's renders of the story under it (reference.go)
	Controls  []*Control
	Log       []string // a line for each call the story's handlers made, oldest first

	src     *source
	all     []*story
	byID    map[string]*story
	cur     *drawn
	matrix  []*variant // the states matrix's rows, made again when a control changes
	drawing bool       // the story is rendering: its calls are its functions', not logged
	cols    int        // the window's width in cells, 0 until the run tells it: what the frame lays out in

	// The groups of the sidebar's tree a click opened (true) or closed
	// (false); one it says nothing of is open while it holds the picked
	// story.
	open map[string]bool

	// The side columns (frame.go): left out by the toolbar's switches,
	// and which keeps its room where the window has room for one.
	noStories, noControls bool
	keepControls          bool
	source                bool // the Source switch: the story's .surf beside the canvas
	// own: the canvas's width is the user's (SetWidth), not the frame's
	// fit of the story's (fitWidth).
	own bool

	// Watching a checkout (Watch):
	dir       string
	every     time.Duration
	rt        surfrun.Runner
	stop      chan struct{}
	reloadErr string            // why the last reload kept what the gallery had
	gate      string            // the checkout's .shots/gate/, the gate's renders
	refSent   map[string][]byte // the reference's renders, as they were sent
}

// Entry is a row of the sidebar's tree: a section's caption, a
// component's group, which folds, or a story under it.
type Entry struct {
	ID    string
	Label string
	Kind  string // section, group or story
	Open  bool   // a group whose stories are shown
	Icon  string // a group's icon, an MDI name: its section's
	Count string // a group's stories, counted
}

// Control is one of the story's controls: a prop, an attribute, a field
// of a struct prop, or a list's length.
type Control struct {
	Name    string
	Key     string // its row's key: its name's letters and digits
	Kind    string // switch, field, select or cycle
	Value   string
	On      bool
	Options []string
	Invalid bool // a field's value that its prop doesn't take

	set     func(v string) bool // gives the story v; false where it doesn't take it
	text    bool                // a string the story draws: the matrix empties it and fills it
	dur     bool                // an int that is a time.Duration, shown as one
	changed bool                // changed by its user: the matrix's states start from it
}

// logLines is how many of the log's last lines show.
const logLines = 8

// New is the gallery with its first story picked: its stories files,
// templates and stylesheet the ones it was built with.
func New() *Gallery {
	g := &Gallery{Theme: "graphite", Density: "compact", src: builtIn()}
	g.index(catalog(g.src))
	if len(g.all) > 0 {
		g.Pick(g.all[0].id)
	}
	return g
}

// index makes all the gallery's stories, each with an id of its own.
func (g *Gallery) index(all []*story) {
	g.all = all
	g.byID = map[string]*story{}
	for _, s := range g.all {
		for id, n := s.id, 2; g.byID[s.id] != nil; n++ {
			s.id = id + "-" + strconv.Itoa(n) // two names with one slug
		}
		g.byID[s.id] = s
	}
}

// Options are the run's options for g. Its document scrolls, so the
// wheel scrolls its stories, its canvas and its controls, each a scroll
// box (hotty SPEC §5.3). Its stylesheet has the matrix's forced rules
// beside the real ones (stories.Beside). A story of stories.All, which
// has no handlers, logs what the host sends it.
func Options(g *Gallery) surfrun.Options {
	return surfrun.Options{Name: "gallery", Scroll: hotty.ScrollVertical, CSS: stories.Beside(g.src.css), Palette: stories.Palette, Unhandled: g.unhandled,
		KeyStyle: stories.KeyStyle}
}

// Interpreted is g drawn by its templates, interpreted.
func Interpreted(g *Gallery) surflive.Component { return surfInterpGallery(g) }

// Compiled is g drawn by its templates compiled into Go.
func Compiled(g *Gallery) surflive.Component { return surfCompiledGallery(g) }

// Shown is the sidebar's tree, flattened: every story whose component or
// name has the filter in it, under its section's caption and its
// component's group. A group's stories are shown while it is open: where
// a click opened it, where it holds the picked story and no click closed
// it, and, while the list is filtered, always.
func (g *Gallery) Shown() []Entry {
	var out []Entry
	section, group, open := "", "", false
	head := -1
	for _, s := range g.found() {
		if s.section != section {
			section, group = s.section, ""
			out = append(out, Entry{ID: "x-" + slug(section), Label: section, Kind: "section"})
		}
		if s.group != group {
			group = s.group
			id := groupID(s)
			open = g.isOpen(id)
			head = len(out)
			out = append(out, Entry{ID: id, Label: group, Kind: "group", Open: open, Icon: sectionIcons[section]})
		}
		n, _ := strconv.Atoi(out[head].Count)
		out[head].Count = strconv.Itoa(n + 1)
		if open {
			out = append(out, Entry{ID: s.id, Label: s.name, Kind: "story"})
		}
	}
	return out
}

// sectionIcons are the icons of a section's groups, MDI names.
var sectionIcons = map[string]string{
	scenes:     "shape-outline",
	widgets:    "code-tags",
	others:     "book-open-outline",
	composites: "puzzle-outline",
	screensSec: "monitor",
}

// groupID is the id of s's group's row.
func groupID(s *story) string { return "g-" + slug(s.section+"-"+s.group) }

// found are the stories the filter leaves, in the catalog's order.
func (g *Gallery) found() []*story {
	q := strings.ToLower(strings.TrimSpace(g.Filter))
	if q == "" {
		return g.all
	}
	var out []*story
	for _, s := range g.all {
		// A scene is found by its words, not as component/name.
		find := s.group + "/" + s.name
		if s.section == scenes {
			find = s.group + " " + s.name
		}
		if strings.Contains(strings.ToLower(find), q) {
			out = append(out, s)
		}
	}
	return out
}

// Stories are the ids of the stories the filter leaves, in the
// sidebar's order, whatever groups are folded.
func (g *Gallery) Stories() []string {
	var ids []string
	for _, s := range g.found() {
		ids = append(ids, s.id)
	}
	return ids
}

// isOpen says the group id shows its stories.
func (g *Gallery) isOpen(id string) bool {
	if strings.TrimSpace(g.Filter) != "" {
		return true
	}
	if open, said := g.open[id]; said {
		return open
	}
	return g.cur != nil && groupID(g.cur.s) == id
}

// Toggle folds the group id, or unfolds it.
func (g *Gallery) Toggle(id string) {
	if g.open == nil {
		g.open = map[string]bool{}
	}
	g.open[id] = !g.isOpen(id)
}

// Themes are the themes a surface takes.
func (g *Gallery) Themes() []string { return valuesOf("theme") }

// Densities are the densities the canvas takes: a surface's but dense,
// the cells' own, which no host draws a page in (the maintainer,
// 2026-11-10).
func (g *Gallery) Densities() []string {
	return slices.DeleteFunc(slices.Clone(valuesOf("density")), func(d string) bool { return d == string(surfui.Dense) })
}

// Widths are the canvas's widths to step through: the story's own among
// them.
func (g *Gallery) Widths() []string {
	ws := []int{24, 30, 40, 50, 60, 80, 100, 120}
	if g.cur != nil && !slices.Contains(ws, g.cur.s.cols) {
		ws = append(ws, g.cur.s.cols)
		slices.Sort(ws)
	}
	if !slices.Contains(ws, g.Width) {
		ws = append(ws, g.Width)
		slices.Sort(ws)
	}
	out := make([]string, len(ws))
	for i, w := range ws {
		out[i] = strconv.Itoa(w)
	}
	return out
}

// Title is the picked story's name, its component's first; a scene's
// is its own, as a person reads it.
func (g *Gallery) Title() string {
	switch {
	case g.cur == nil:
		return "No story"
	case g.cur.s.section == scenes:
		return g.cur.s.name
	}
	return g.cur.s.group + " / " + g.cur.s.name
}

// Section is the picked story's section in the sidebar.
func (g *Gallery) Section() string {
	if g.cur == nil {
		return ""
	}
	return g.cur.s.section
}

// About is a scene's line under its title: its component's tag, and
// the sentence that says what the scene shows; "" for any other story.
func (g *Gallery) About() string {
	if g.cur == nil || g.cur.s.section != scenes {
		return ""
	}
	tag := "<" + g.cur.s.file.of + ">"
	if g.cur.s.about == "" {
		return tag
	}
	return tag + "  " + g.cur.s.about
}

// HasControls says the story has controls: the toolbar's Controls
// switch and the controls' column are shown only where it has.
func (g *Gallery) HasControls() bool { return len(g.Controls) > 0 }

// SourcePart is a piece of a story's .surf source: where it is from,
// its text, and the file's line its first line is.
type SourcePart struct {
	Title string
	Text  string
	From  int
}

// Source is the picked story's .surf source, as its author wrote it: a
// stories file's story is its own lines of the file, and under them the
// template it draws, where it draws one (a screen, a partial); a
// widget's own story is the template its controls make. A story of
// stories.All is a Go tree, and has none.
func (g *Gallery) Source() []SourcePart {
	if g.cur == nil {
		return nil
	}
	s := g.cur.s
	switch {
	case s.widget != nil && g.cur.src != "":
		return []SourcePart{{Title: "<" + s.widget.Tag + ">, as the controls set it", Text: g.cur.src, From: 1}}
	case s.file != nil && s.at >= 0 && s.at < len(s.file.stories):
		f := s.file
		st := &f.stories[s.at]
		out := []SourcePart{{Title: storiesDir + "/" + f.name + ".surf", Text: st.src, From: st.line}}
		if f.pkg != nil && f.pkg.load() == nil {
			name := f.of + ".surf"
			if tf, ok := f.pkg.files[name]; ok {
				out = append(out, SourcePart{Title: name + ", the template it draws", Text: strings.TrimRight(string(tf.Data), "\n"), From: 1})
			}
		}
		return out
	}
	return nil
}

// HasSource says the story has .surf source to show: the toolbar's
// Source switch is drawn only where it has.
func (g *Gallery) HasSource() bool { return len(g.Source()) > 0 }

// HasCells says the story's component has a cells UI, which the
// toolbar's Cells switch draws it by.
func (g *Gallery) HasCells() bool { return g.cur != nil && g.cur.cells() != nil }

// Canvas is the picked story, drawn live.
func (g *Gallery) Canvas() surflive.Component { return canvas{g} }

// Tail is the log's last lines.
func (g *Gallery) Tail() []string { return g.Log[max(len(g.Log)-logLines, 0):] }

// Quiet is what the log says while it is empty.
func (g *Gallery) Quiet() string {
	if g.cur != nil && g.cur.tree != nil {
		return "A story of stories.All has no handlers: a click or a focus on it is logged as the host sends it."
	}
	if g.cur != nil && g.cur.s.section == scenes {
		return "A scene has no handlers: a click or a focus on it is logged as the host sends it."
	}
	return "No calls yet: click, type or pick in the story."
}

// Uncontrolled is what the controls' panel says where the story has none.
func (g *Gallery) Uncontrolled() string {
	switch {
	case g.cur == nil:
		return ""
	case g.cur.tree != nil:
		return "A story of stories.All is a Go tree: it has no props."
	case g.cur.s.section == scenes:
		return "A scene has no controls: edit its stories file, and it draws again."
	case g.cur.s.file != nil && g.cur.s.file.screen():
		return "A screen takes no props: its stories are its Go state's."
	}
	return "Its component declares no props."
}

// SetFilter narrows the list to the stories with q in their names.
func (g *Gallery) SetFilter(q string) { g.Filter = q }

// Pick picks the story id.
func (g *Gallery) Pick(id string) {
	s, ok := g.byID[id]
	if !ok {
		return
	}
	delete(g.open, groupID(s)) // its group shows it, whatever a click said
	g.Sel, g.Cells, g.own = s.id, false, false
	g.Log, g.matrix = nil, nil
	g.cur = g.draw(s)
	g.Controls = g.cur.controls
	for _, c := range g.Controls {
		c.Key = slug(c.Name)
	}
	g.Width = fitWidth(g.cols, s.cols, g.reserve())
	g.cur.resize(g.Width)
}

// Move picks the story by stories down the list from the picked one (up
// for less than 0), through the groups, folded or not: the one it comes
// to unfolds.
func (g *Gallery) Move(by int) {
	ids := g.Stories()
	if len(ids) == 0 {
		return
	}
	at := slices.Index(ids, g.Sel)
	at = min(max(at+by, 0), len(ids)-1)
	if ids[at] != g.Sel {
		g.Pick(ids[at])
	}
}

// SetTheme is the canvas's theme.
func (g *Gallery) SetTheme(t string) { g.Theme = t }

// SetDensity is the canvas's density, one of Densities.
func (g *Gallery) SetDensity(d string) {
	if slices.Contains(g.Densities(), d) {
		g.Density = d
	}
}

// SetWidth is the canvas's width in cells, which the story is drawn at
// whatever the window's.
func (g *Gallery) SetWidth(w string) {
	if n, err := strconv.Atoi(w); err == nil && n > 0 {
		g.Width, g.own = n, true
		if g.cur != nil {
			g.cur.resize(g.Width)
		}
		for _, v := range g.matrix {
			v.d.resize(g.Width)
		}
	}
}

// FlipCells draws the story by its cells UI, or by its templates again.
func (g *Gallery) FlipCells() { g.Cells = !g.Cells && g.HasCells() }

// HasStates says the story has a states matrix: a widget's and a
// composite's, whose states its controls make.
func (g *Gallery) HasStates() bool {
	return g.cur != nil && (g.cur.s.widget != nil || g.cur.host != nil)
}

// FlipStates draws the story in each of its states, or as it is again.
func (g *Gallery) FlipStates() {
	g.States = !g.States && g.HasStates()
	g.matrix = nil
}

// ClearLog empties the log.
func (g *Gallery) ClearLog() { g.Log = nil }

// Flip turns control i, a switch, on or off.
func (g *Gallery) Flip(i int) {
	if i < 0 || i >= len(g.Controls) {
		return
	}
	c := g.Controls[i]
	c.On = !c.On
	c.Value = strconv.FormatBool(c.On)
	c.Invalid = !c.set(c.Value)
	c.changed, g.matrix = true, nil
}

// Choose gives control i the value v.
func (g *Gallery) Choose(i int, v string) {
	if i < 0 || i >= len(g.Controls) {
		return
	}
	c := g.Controls[i]
	c.Value = v
	c.Invalid = !c.set(v)
	c.changed, g.matrix = true, nil
}

// logf adds a line to the log, unless the story is rendering: a call
// then is a function's, which a render makes, not an act's.
func (g *Gallery) logf(line string) {
	if g.drawing {
		return
	}
	g.Log = append(g.Log, line)
}

// unhandled logs an event the host sent on a story of stories.All,
// which has no handlers, as it came: a click, a focus.
func (g *Gallery) unhandled(e surfrun.Event) {
	if g.cur == nil || g.cur.tree == nil && g.cur.s.section != scenes || !strings.Contains(e.Target+".", ".kc-"+g.Sel+".") {
		return
	}
	g.logf("(no handler) " + eventText(e))
}

// slug is s as an element's key: its letters and digits, and - for the
// rest.
func slug(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}
