package gallery

import (
	"fmt"
	"strconv"
	"strings"

	"neuroplast.io/hotty-surf/pkg/markup"
)

// storyFile is a stories file (stories.Files): a component's stories.
type storyFile struct {
	name    string // the file's, without .surf
	of      string // the component's template
	in      string // the widget each story is drawn in, if any
	cols    int
	stories []fileStory
	states  []state // its rows of the states matrix
	pkg     *pack   // the package whose template of is
	scene   bool    // of is a widget of the vocabulary: each story's children are its scene
	title   string  // a scene file's component as a person names it: Button
	err     error   // why its stories draw nothing
}

// fileStory is one <story> of a stories file.
type fileStory struct {
	name  string
	cols  int
	about string        // a muted line drawn under it, if any; a scene's sentence, under its title
	props []markup.Attr // its other attributes: the props it gives
	src   string        // its own lines of the file, as written, their common indent taken off
	line  int           // the file's line its first is
	kids  []*markup.Node
}

// screen says the file's component is a component of its package's, with
// its own state, not a partial: a screen.
func (f *storyFile) screen() bool { return f.pkg != nil && f.pkg.component(f.of) != nil }

// parseStoryFile reads a stories file, of a template of one of packs.
// Where it is wrong, f.err says where and why, and it has the stories
// read up to there.
func parseStoryFile(file, src string, packs []*pack) *storyFile {
	f := &storyFile{name: strings.TrimSuffix(file, ".surf")}
	fail := func(at markup.Pos, format string, args ...any) *storyFile {
		if f.err == nil {
			f.err = fmt.Errorf("%s:%d:%d: %s", file, at.Line, at.Col, fmt.Sprintf(format, args...))
		}
		return f
	}
	nodes, err := markup.Parse(file, src)
	if err != nil {
		f.err = err
		return f
	}
	root := elements(nodes)
	if len(root) != 1 || root[0].Tag != "stories" {
		return fail(markup.Pos{Line: 1, Col: 1}, "a stories file is one <stories of=\"component\">")
	}
	for _, a := range root[0].Attrs {
		switch a.Name {
		case "of":
			f.of = a.Value
		case "in":
			f.in = a.Value
		case "cols":
			n, err := strconv.Atoi(a.Value)
			if err != nil || n < 1 {
				return fail(a.ValuePos, "cols is a width in cells, not %q", a.Value)
			}
			f.cols = n
		case "title":
			f.title = a.Value
		default:
			return fail(a.Pos, "<stories> takes of, title, in and cols, not %s", a.Name)
		}
	}
	if f.of == "" {
		return fail(root[0].Pos, "<stories> has no of: the component's template")
	}
	if f.pkg = packOf(packs, f.of); f.pkg == nil {
		if _, ok := widgetOf(f.of); !ok {
			return fail(root[0].Pos, "no package has a template %s.surf, and the vocabulary has no <%s>", f.of, f.of)
		}
		f.scene = true
		if f.in != "" {
			return fail(root[0].Pos, "<%s> is a widget: its stories are scenes, drawn in nothing (in)", f.of)
		}
	}
	if f.title != "" && !f.scene {
		return fail(root[0].Pos, "title is a scene file's: %s is a template", f.of)
	}
	for _, n := range elements(root[0].Kids) {
		if n.Tag == "state" {
			st := state{}
			for _, a := range n.Attrs {
				if a.Name == "name" {
					st.name = a.Value
				} else {
					st.values = append(st.values, setting{a.Name, a.Value})
				}
			}
			if st.name == "" {
				return fail(n.Pos, "a <state> has a name")
			}
			f.states = append(f.states, st)
			continue
		}
		if n.Tag != "story" {
			return fail(n.Pos, "<stories> holds <story>s and <state>s, not <%s>", n.Tag)
		}
		st := fileStory{kids: n.Kids, line: n.Pos.Line}
		st.src = storySource(src, n)
		for _, a := range n.Attrs {
			switch a.Name {
			case "name":
				st.name = a.Value
			case "cols":
				c, err := strconv.Atoi(a.Value)
				if err != nil || c < 1 {
					return fail(a.ValuePos, "cols is a width in cells, not %q", a.Value)
				}
				st.cols = c
			case "about":
				st.about = a.Value
			default:
				st.props = append(st.props, a)
			}
		}
		if st.name == "" {
			return fail(n.Pos, "a <story> has a name")
		}
		if f.scene && (len(st.props) > 0 || len(elements(st.kids)) == 0) {
			return fail(n.Pos, "<%s> is a widget: a story of it is a scene, its children, and gives no props", f.of)
		}
		if f.screen() && (len(st.props) > 0 || len(elements(st.kids)) > 0 || st.about != "") {
			return fail(n.Pos, "%s is a component: a story of it gives no props, no children and no about, its Go state is its own", f.of)
		}
		f.stories = append(f.stories, st)
	}
	return f
}

// storySource is the lines of src that are the <story> n, from its
// opening tag to its closing one, with the indent they share taken off:
// what its author wrote. A story holds no story, so its end is the
// first </story> from its start, or, where it has no children, the end
// of its tag.
func storySource(src string, n *markup.Node) string {
	lines := strings.Split(src, "\n")
	at := n.Pos.Line - 1
	if at < 0 || at >= len(lines) {
		return ""
	}
	end := len(lines) - 1
	for i := at; i < len(lines); i++ {
		if strings.Contains(lines[i], "</story>") || len(n.Kids) == 0 && strings.Contains(lines[i], "/>") {
			end = i
			break
		}
	}
	lines = lines[at : end+1]
	indent := -1
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		if in := len(l) - len(strings.TrimLeft(l, " \t")); indent < 0 || in < indent {
			indent = in
		}
	}
	for i, l := range lines {
		if len(l) >= indent && indent > 0 {
			lines[i] = l[indent:]
		}
		lines[i] = strings.TrimRight(lines[i], " \t\r")
	}
	return strings.Join(lines, "\n")
}

// elements are ns' elements: their texts are space between them.
func elements(ns []*markup.Node) []*markup.Node {
	var out []*markup.Node
	for _, n := range ns {
		if n.Kind == markup.Element {
			out = append(out, n)
		}
	}
	return out
}

// hostTemplate is the template a partial's story is drawn by: the
// partial's tag, in the file's widget (a stack where it names none, so
// that a partial that draws nothing is no error), each prop given by its
// method (storyProp0, …), with the story's argument where the story
// binds it, and the story's children; then the story's about, a muted
// line, where it has one.
func hostTemplate(f *storyFile, st *fileStory, props []*prop) string {
	in := f.in
	if in == "" {
		in = "stack"
	}
	var b strings.Builder
	b.WriteString("<" + in + ">")
	b.WriteString("<" + f.of)
	for i, p := range props {
		fmt.Fprintf(&b, ` :%s="%s(%s)"`, p.name, propMethod(i), escape(p.expr, true))
	}
	if len(st.kids) == 0 {
		b.WriteString("/>")
	} else {
		b.WriteString(">")
		for _, k := range st.kids {
			writeNode(&b, k)
		}
		b.WriteString("</" + f.of + ">")
	}
	if st.about != "" {
		b.WriteString(`<text small muted wrap key="about">` + escape(st.about, false) + "</text>")
	}
	b.WriteString("</" + in + ">")
	return b.String()
}

// propMethod is the name of the host's method that gives prop i.
func propMethod(i int) string { return "storyProp" + strconv.Itoa(i) }

// writeNode writes n as a template's source.
func writeNode(b *strings.Builder, n *markup.Node) {
	if n.Kind == markup.Text {
		for _, p := range n.Parts {
			if p.Expr {
				b.WriteString("{{ " + p.Src + " }}")
			} else {
				b.WriteString(escape(p.Src, false))
			}
		}
		return
	}
	b.WriteString("<" + n.Tag)
	for _, a := range n.Attrs {
		b.WriteString(" " + a.Name)
		if !a.Bare {
			b.WriteString(`="` + escape(a.Value, true) + `"`)
		}
	}
	if len(n.Kids) == 0 {
		b.WriteString("/>")
		return
	}
	b.WriteString(">")
	for _, k := range n.Kids {
		writeNode(b, k)
	}
	b.WriteString("</" + n.Tag + ">")
}

// escape is s as a template's literal text, or an attribute's value in
// "". A {{ in a text stays one: no entity writes it.
func escape(s string, attr bool) string {
	if attr {
		return attrEscaper.Replace(s)
	}
	return textEscaper.Replace(s)
}

var (
	textEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;")
	attrEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", `"`, "&quot;")
)

// sceneTemplate is the template a scene is drawn by: the story's
// children, one under another.
func sceneTemplate(st *fileStory) string {
	var b strings.Builder
	b.WriteString("<stack>")
	for _, k := range st.kids {
		writeNode(&b, k)
	}
	b.WriteString("</stack>")
	return b.String()
}
