package gallery

import (
	"cmp"
	"io/fs"
	"slices"
	"strings"

	surf "neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/stories"
	"neuroplast.io/hotty-surf/surfui"
)

// story is one of the gallery's stories, from one of three sources.
type story struct {
	id      string // its entry's
	section string // Widgets, Stories, Composites or Screens
	group   string // its component's: a widget's tag, a template's name
	name    string
	cols    int // the canvas's width, in cells
	// One of:
	tree   func() *surfui.Node // a story of stories.All, drawn as it is
	widget *surf.Widget        // a widget's own, made from its attributes
	file   *storyFile          // a story of a stories file
	at     int                 // its place in file
}

// The sections, in the sidebar's order.
const (
	widgets    = "Widgets"
	others     = "Stories" // stories.All's that are no widget's
	composites = "Composites"
	screensSec = "Screens"
)

// Composites and screens come in this order, then the rest by name.
var (
	compositeOrder = []string{"palette-row", "notice-row", "foot-hint", "key-caps", "setting-group", "setting-row"}
	screenOrder    = []string{"palette", "settings", "notices", "agents"}
)

// catalog is every story: each widget's own and its stories of
// stories.All, stories.All's other stories, then src's stories files'
// composites and screens.
func catalog(src *source) []*story {
	var out []*story
	byGroup := map[string][]stories.Story{}
	var groups []string
	for _, s := range stories.All {
		g, _, _ := strings.Cut(s.Name, "/")
		if _, ok := byGroup[g]; !ok {
			groups = append(groups, g)
		}
		byGroup[g] = append(byGroup[g], s)
	}
	all := func(section, g string) {
		for _, s := range byGroup[g] {
			_, name, _ := strings.Cut(s.Name, "/")
			out = append(out, &story{id: "s-" + slug(s.Name), section: section, group: g, name: name, cols: s.Cols, tree: s.Tree})
		}
	}
	for i := range surf.Widgets {
		w := &surf.Widgets[i]
		if w.Tag == "code" {
			w = surf.BlockCode() // the block, which has the attributes; the inline code is the prose story's
		}
		out = append(out, &story{id: "w-" + w.Tag, section: widgets, group: w.Tag, name: "attributes", cols: 50, widget: w})
		all(widgets, w.Tag)
	}
	for _, g := range groups {
		if _, ok := widgetOf(g); !ok {
			all(others, g)
		}
	}
	files := readStoryFiles(src.stories, src.packs)
	for _, sec := range []struct {
		name  string
		order []string
		of    bool // its files' components are screens
	}{{composites, compositeOrder, false}, {screensSec, screenOrder, true}} {
		var in []*storyFile
		for _, f := range files {
			if f.screen() == sec.of {
				in = append(in, f)
			}
		}
		slices.SortStableFunc(in, func(a, b *storyFile) int {
			ia, ib := orderOf(sec.order, a.of), orderOf(sec.order, b.of)
			return cmp.Or(cmp.Compare(ia, ib), cmp.Compare(a.of, b.of))
		})
		for _, f := range in {
			for i, st := range f.stories {
				out = append(out, &story{id: "f-" + slug(f.name+"-"+st.name), section: sec.name, group: f.of, name: st.name,
					cols: cmp.Or(st.cols, f.cols, 60), file: f, at: i})
			}
			if len(f.stories) == 0 {
				out = append(out, &story{id: "f-" + slug(f.name), section: sec.name, group: f.of, name: "no story", cols: 60, file: f, at: -1})
			}
		}
	}
	return out
}

// orderOf is name's place in order, after all of it where it isn't there.
func orderOf(order []string, name string) int {
	if i := slices.Index(order, name); i >= 0 {
		return i
	}
	return len(order)
}

// widgetOf is the widget of tag.
func widgetOf(tag string) (*surf.Widget, bool) {
	for i := range surf.Widgets {
		if surf.Widgets[i].Tag == tag {
			return &surf.Widgets[i], true
		}
	}
	return nil, false
}

// valuesOf are the values the attribute name takes.
func valuesOf(name string) []string {
	for _, a := range surf.Attrs {
		if a.Name == name && a.Values != nil {
			return a.Values
		}
	}
	return nil
}

// readStoryFiles reads every stories file of fsys's root, by name, each
// of a template of one of packs. One that doesn't read is still listed,
// with its error as its one story.
func readStoryFiles(fsys fs.FS, packs []*pack) []*storyFile {
	names, _ := fs.Glob(fsys, "*.surf")
	var out []*storyFile
	for _, n := range names {
		src, err := fs.ReadFile(fsys, n)
		f := parseStoryFile(n, string(src), packs)
		if err != nil {
			f.err = err
		}
		out = append(out, f)
	}
	return out
}
