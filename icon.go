package surf

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"neuroplast.io/hotty-surf/pkg/markup"
	"neuroplast.io/hotty-surf/pkg/mdi"
	"neuroplast.io/hotty-surf/surfui"
)

// An <icon> is a glyph: its text (&#xF08BB;), or an MDI symbol's name
// (name="cog-outline"), which a literal resolves when the template
// compiles, so that neither mode's code holds MDI's names unless a name
// is bound. With svg= or path= a host draws an SVG in its place, filled
// in its colour at its size, and the cells draw its glyph: the element's
// name=, or its asset's.
//
// An svg="logo" is logo.svg, beside the templates, which the Set reads
// with them: a 24×24 box, one <path>, and the glyph the cells draw as an
// MDI name in data-glyph:
//
//	<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"
//	     data-glyph="hexagon-outline">
//	  <path d="M12 2 …Z"/>
//	</svg>

// SVGAsset is an .svg file beside the templates: its path's data, and
// the glyph the cells draw for it, "" where it names none.
type SVGAsset struct {
	Path  string
	Glyph string
}

// helpBox is what the cells draw for a name MDI has not: MDI's help-box.
const helpBox = string(mdi.HelpBox)

// IconGlyph is the glyph of MDI's symbol name, as an <icon>'s bound name
// draws it: help-box, and an error, for a name MDI has not.
func IconGlyph(name string) (string, error) {
	if g, ok := mdi.Glyph(name); ok {
		return string(g), nil
	}
	return helpBox, fmt.Errorf("an icon's name is one of MDI's, as cog-outline, not %q", name)
}

// IconNode is an <icon> as both modes build it: glyph, and where path is
// set, an SVG of it a host draws in the glyph's place. A path CheckPath
// refuses is left out, and is the error.
func IconNode(glyph, path string) (*surfui.Node, error) {
	if path == "" {
		return surfui.Icon(glyph), nil
	}
	if glyph == "" {
		glyph = helpBox
	}
	if err := CheckPath(path); err != nil {
		return surfui.Icon(glyph), err
	}
	return surfui.SVGIcon(glyph, path), nil
}

// IconAsset is an <icon>'s bound svg: the asset name, a found or not, as
// the Set's SVG gives it. The glyph is the element's where it has one,
// else the asset's, else help-box; an asset the Set has not is help-box,
// and the error.
func IconAsset(glyph, name string, a SVGAsset, ok bool) (*surfui.Node, error) {
	if !ok {
		return surfui.Icon(helpBox), fmt.Errorf("no svg icon %q: an svg is an .svg file beside the templates", name)
	}
	if glyph == "" {
		glyph = a.Glyph
	}
	return IconNode(glyph, a.Path)
}

// ButtonNode is a <button> as both modes build it: its label, and the
// icon among its children, before the label or after it.
func ButtonNode(label string, after bool, icons []*surfui.Node) *surfui.Node {
	if len(icons) == 0 {
		return surfui.Button(label)
	}
	return surfui.IconButton(icons[0], label, after)
}

// CheckPath checks an SVG path's data: its commands and numbers, and
// nothing that isn't.
func CheckPath(d string) error {
	for _, c := range []byte(d) {
		switch {
		case c >= '0' && c <= '9', c == '.', c == '-', c == '+', c == ',', c == 'e', c == 'E',
			c == ' ', c == '\t', c == '\n', c == '\r', strings.IndexByte("MmLlHhVvCcSsQqTtAaZz", c) >= 0:
		default:
			return fmt.Errorf("a path is an SVG path's data, as \"M12 2L2 22h20Z\", not %q", short(d))
		}
	}
	if strings.TrimSpace(d) == "" {
		return errors.New("a path is an SVG path's data, as \"M12 2L2 22h20Z\", not empty")
	}
	return nil
}

// short is s cut to a refusal's length.
func short(s string) string {
	if len(s) > 24 {
		return s[:24] + "…"
	}
	return s
}

// SVG is the Set's asset of that name: name.svg beside its templates.
func (s *Set) SVG(name string) (SVGAsset, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.svgs[name]
	return a, ok
}

// SVGs are the names of the Set's assets, in order.
func (s *Set) SVGs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	names := make([]string, 0, len(s.svgs))
	for n := range s.svgs {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// readSVGs reads every .svg file in fsys's root, with its stamp.
func readSVGs(fsys fs.FS, stamps map[string]stamp) (map[string]SVGAsset, error) {
	ents, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	svgs := map[string]SVGAsset{}
	var errs []error
	for _, e := range ents {
		if e.IsDir() || path.Ext(e.Name()) != ".svg" {
			continue
		}
		src, err := fs.ReadFile(fsys, e.Name())
		if err != nil {
			errs = append(errs, err)
			continue
		}
		stamps[e.Name()] = stampOf(fsys, e.Name(), string(src))
		a, err := parseSVG(e.Name(), string(src))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		svgs[strings.TrimSuffix(e.Name(), ".svg")] = a
	}
	return svgs, errors.Join(errs...)
}

// svgAttrs are the attributes an asset's <svg> takes, which the inline
// SVG a host draws doesn't keep (but data-glyph, the cells').
var svgAttrs = []string{"data-glyph", "fill", "height", "id", "version", "viewBox", "width", "xmlns", "xmlns:xlink"}

// parseSVG reads an asset: an <svg> of viewBox 0 0 24 24, which holds one
// <path d> (and a <title> or <desc>, which a host doesn't draw).
func parseSVG(name, src string) (SVGAsset, error) {
	if strings.HasPrefix(strings.TrimSpace(src), "<?xml") {
		if i := strings.Index(src, "?>"); i >= 0 {
			// Blanked, not cut, so that a place is the file's.
			src = blank(src[:i+2]) + src[i+2:]
		}
	}
	nodes, err := markup.Parse(name, src)
	if err != nil {
		return SVGAsset{}, err
	}
	fail := func(at markup.Pos, format string, args ...any) (SVGAsset, error) {
		return SVGAsset{}, &markup.Error{File: name, Pos: at, Msg: fmt.Sprintf(format, args...)}
	}
	if len(nodes) != 1 || nodes[0].Kind != markup.Element || nodes[0].Tag != "svg" {
		return fail(markup.Pos{Line: 1, Col: 1}, "an svg icon is one <svg> element")
	}
	root := nodes[0]
	var a SVGAsset
	box := false
	for _, at := range root.Attrs {
		switch at.Name {
		case "viewBox":
			if strings.Join(strings.Fields(at.Value), " ") != "0 0 24 24" {
				return fail(at.ValuePos, "an svg icon's viewBox is 0 0 24 24, not %q", at.Value)
			}
			box = true
		case "data-glyph":
			g, ok := mdi.Glyph(at.Value)
			if !ok {
				return fail(at.ValuePos, "data-glyph is one of MDI's names, as cog-outline, not %q", at.Value)
			}
			a.Glyph = string(g)
		default:
			if !contains(svgAttrs, at.Name) {
				return fail(at.Pos, "%s", takesNo("svg", at.Name, svgAttrs))
			}
		}
	}
	if !box {
		return fail(root.Pos, "an svg icon's box is 24×24: viewBox=\"0 0 24 24\"")
	}
	for _, k := range root.Kids {
		switch {
		case k.Kind == markup.Text:
			return fail(k.Pos, "an svg icon holds a <path>, not text")
		case k.Tag == "title" || k.Tag == "desc":
		case k.Tag != "path":
			return fail(k.Pos, "an svg icon holds one <path>, not <%s>: draw it as one path", k.Tag)
		case a.Path != "":
			return fail(k.Pos, "an svg icon holds one <path>; this is a second: join them into one")
		default:
			for _, at := range k.Attrs {
				if at.Name != "d" && at.Name != "id" {
					return fail(at.Pos, "%s", takesNo("path", at.Name, []string{"d", "id"}))
				}
			}
			d, ok := k.Attr("d")
			if !ok {
				return fail(k.Pos, "<path> has its data, d=\"M…\"")
			}
			if err := CheckPath(d.Value); err != nil {
				return fail(d.ValuePos, "%v", err)
			}
			a.Path = strings.Join(strings.Fields(d.Value), " ")
		}
	}
	if a.Path == "" {
		return fail(root.Pos, "an svg icon holds one <path d=\"M…\">")
	}
	return a, nil
}

// blank is s as spaces, its newlines kept.
func blank(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' {
			return r
		}
		return ' '
	}, s)
}
