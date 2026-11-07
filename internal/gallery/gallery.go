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

	"github.com/neuroplastio/hotty-go"

	"github.com/neuroplastio/hotty-surf/internal/stories"
	"github.com/neuroplastio/hotty-surf/surflive"
	"github.com/neuroplastio/hotty-surf/surfrun"
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

	// The side columns (frame.go): left out by the toolbar's switches,
	// and which keeps its room where the window has room for one.
	noStories, noControls bool
	keepControls          bool
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

// Entry is a line of the sidebar's list: a section's head, a component's,
// or a story.
type Entry struct {
	ID    string
	Label string
	Kind  string // section, group or story
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

// Shown is the sidebar's list: every story whose component or name has
// the filter in it, under its section's and its component's heads.
func (g *Gallery) Shown() []Entry {
	q := strings.ToLower(strings.TrimSpace(g.Filter))
	var out []Entry
	section, group := "", ""
	for _, s := range g.all {
		if q != "" && !strings.Contains(strings.ToLower(s.group+"/"+s.name), q) {
			continue
		}
		if s.section != section {
			section, group = s.section, ""
			out = append(out, Entry{ID: "x-" + slug(section), Label: section, Kind: "section"})
		}
		if s.group != group {
			group = s.group
			out = append(out, Entry{ID: "g-" + slug(section+"-"+group), Label: group, Kind: "group"})
		}
		out = append(out, Entry{ID: s.id, Label: s.name, Kind: "story"})
	}
	return out
}

// Themes are the themes a surface takes.
func (g *Gallery) Themes() []string { return valuesOf("theme") }

// Densities are the densities a surface takes.
func (g *Gallery) Densities() []string { return valuesOf("density") }

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

// Title is the picked story's name, its component's first.
func (g *Gallery) Title() string {
	if g.cur == nil {
		return "No story"
	}
	return g.cur.s.group + " / " + g.cur.s.name
}

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
	return "No calls yet: click, type or pick in the story."
}

// Uncontrolled is what the controls' panel says where the story has none.
func (g *Gallery) Uncontrolled() string {
	switch {
	case g.cur == nil:
		return ""
	case g.cur.tree != nil:
		return "A story of stories.All is a Go tree: it has no props."
	case g.cur.s.file != nil && g.cur.s.file.screen():
		return "A screen takes no props: its stories are its Go state's."
	}
	return "Its component declares no props."
}

// SetFilter narrows the list to the stories with q in their names.
func (g *Gallery) SetFilter(q string) { g.Filter = q }

// Pick picks the story id; a head picks its first story.
func (g *Gallery) Pick(id string) {
	s, ok := g.byID[id]
	if !ok {
		shown := g.Shown()
		at := slices.IndexFunc(shown, func(e Entry) bool { return e.ID == id })
		if at < 0 {
			return
		}
		for _, e := range shown[at:] {
			if e.Kind == "story" {
				s = g.byID[e.ID]
				break
			}
		}
		if s == nil {
			return
		}
	}
	g.Sel, g.Width, g.Cells, g.own = s.id, fitWidth(g.cols, s.cols), false, false
	g.Log, g.matrix = nil, nil
	g.cur = g.draw(s)
	g.Controls = g.cur.controls
	for _, c := range g.Controls {
		c.Key = slug(c.Name)
	}
	g.cur.resize(g.Width)
}

// Move picks the story by stories down the list from the picked one (up
// for less than 0).
func (g *Gallery) Move(by int) {
	var ids []string
	for _, e := range g.Shown() {
		if e.Kind == "story" {
			ids = append(ids, e.ID)
		}
	}
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

// SetDensity is the canvas's density.
func (g *Gallery) SetDensity(d string) { g.Density = d }

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
	if g.cur == nil || g.cur.tree == nil || !strings.Contains(e.Target+".", ".kc-"+g.Sel+".") {
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
