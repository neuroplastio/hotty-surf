package gallery

import (
	"io/fs"
	"maps"
	"strconv"
	"sync"
	"testing/fstest"

	surf "github.com/neuroplastio/hotty-surf"
	"github.com/neuroplastio/hotty-surf/internal/screens"
)

// pack is a package whose components the stories files draw: its
// templates, shapes and instances, as surfgen -export gives them.
type pack struct {
	surf.Package

	once   sync.Once
	files  fstest.MapFS // its templates, read once
	shapes map[string]*surf.Shape
	set    *surf.Set            // its own, for its components' stories
	sets   map[string]*surf.Set // its partials' stories', by their host templates and shapes
	err    error
}

// packages are the packages the stories files may name a template of,
// each with its directory in a checkout, whose templates a watch reads.
var packages = []struct {
	pkg func() surf.Package
	dir string
}{{screens.SurfPackage, "internal/screens"}}

// Where a checkout keeps the stories files and the stylesheet.
const (
	storiesDir = "internal/stories"
	cssFile    = "surfui/surf.css"
)

// packOf is the package of packs with the template name.
func packOf(packs []*pack, name string) *pack {
	for _, p := range packs {
		if _, err := fs.Stat(p.Files, name+".surf"); err == nil {
			return p
		}
	}
	return nil
}

// load reads the package's templates and loads its own Set, once.
func (p *pack) load() error {
	p.once.Do(func() {
		p.files = fstest.MapFS{}
		p.shapes = map[string]*surf.Shape{}
		for _, sh := range p.Shapes {
			p.shapes[sh.Name] = sh
		}
		names, err := fs.Glob(p.Files, "*.surf")
		if err != nil {
			p.err = err
			return
		}
		for _, n := range names {
			src, err := fs.ReadFile(p.Files, n)
			if err != nil {
				p.err = err
				return
			}
			p.files[n] = &fstest.MapFile{Data: src}
		}
		p.set, p.err = surf.Load(p.Files, p.Shapes)
	})
	return p.err
}

// component is the shape of the component whose template is name: nil
// for a partial's.
func (p *pack) component(name string) *surf.Shape {
	for _, sh := range p.Shapes {
		if sh.Component && surf.TemplateName(sh.Name) == name {
			return sh
		}
	}
	return nil
}

// storySet is a Set of the package's templates and host, a story's own
// template, drawn with hostShape. The package's components are its data
// structs here: no template of theirs is compiled, but what the host
// reads of them is declared. It is made once for each host: the states
// matrix draws a story in many.
func (p *pack) storySet(host string, hostShape *surf.Shape) (*surf.Set, error) {
	if err := p.load(); err != nil {
		return nil, err
	}
	key := host
	for _, f := range hostShape.Fields {
		key += "\x00" + f.Name
	}
	for _, m := range hostShape.Methods {
		key += "\x00" + m.Name + "(" + strconv.Itoa(len(m.Args)) + ")"
	}
	if s, ok := p.sets[key]; ok {
		return s, nil
	}
	files := maps.Clone(p.files)
	files[surf.TemplateName(hostShape.Name)+".surf"] = &fstest.MapFile{Data: []byte(host)}
	shapes := []*surf.Shape{hostShape}
	for _, sh := range p.Shapes {
		c := *sh
		c.Component = false
		shapes = append(shapes, &c)
	}
	s, err := surf.Load(files, shapes)
	if err == nil {
		if p.sets == nil {
			p.sets = map[string]*surf.Set{}
		}
		p.sets[key] = s
	}
	return s, err
}
