package storybook

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Source is a piece of real code the Source tab shows: the story's own
// (its Add call, its render) or a component's it calls.
type Source struct {
	Title string // "story", "render", "basecoat.Menu"
	Path  string // relative to the module's root
	Line  int    // the first line's number
	Lang  string // "go", "templ"
	Code  string
}

// root is the module's directory and its path, as this file was built:
// the sources are read from there at run time, so an edit shows on the
// next render. A binary run away from its source tree has none to show.
var root = func() string {
	_, f, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(f))))
}()

const module = "neuroplast.io/hotty-surf"

// Sources is the story's code, then the code of the components it calls:
// the Add call, the render (a templ component or a Go func), and each
// component the render calls in the module (a templ component, or a Go
// func or type), one level of the
// story's own helpers deep.
func (s *Story) Sources() []Source {
	if s.fn == 0 {
		return storySources(s.id)
	}
	var out []Source
	seen := map[string]bool{}
	if src, ok := callAt(s.file, s.line); ok {
		src.Title = "story"
		out = append(out, src)
	}
	fn := runtime.FuncForPC(s.fn)
	if fn == nil {
		return out
	}
	file, _ := fn.FileLine(s.fn)
	name := fn.Name()[strings.LastIndex(fn.Name(), ".")+1:]
	if strings.HasPrefix(name, "func") { // a func literal: it is in the Add call
		out = refs(out, out[len(out)-1:], filepath.Dir(file), seen, 1)
		return s.propTypes(out, out[len(out)-1], seen)
	}
	var render Source
	var ok bool
	if t, found := strings.CutSuffix(file, "_templ.go"); found {
		render, ok = templBlock(t+".templ", name)
	} else {
		render, ok = goDecl(file, name)
	}
	if !ok {
		return out
	}
	render.Title = "render: " + name
	seen[filepath.Dir(file)+"."+name] = true
	out = append(out, render)
	out = refs(out, out, filepath.Dir(file), seen, 1)
	return s.propTypes(out, render, seen)
}

// storySources finds a v0.7 story's registration and its templ render.
// Those registrations wrap their render functions in closures, so the
// closure's program counter alone cannot name the original component.
func storySources(id string) []Source {
	if demo := demoSources(id); len(demo) > 0 {
		return demo
	}
	dir := filepath.Join(root, "internal", "stories")
	entries, _ := os.ReadDir(dir)
	needle := `ID: "` + id + `"`
	render := regexp.MustCompile(`View:\s*(?:\w+\()?([a-zA-Z_]\w*)`)
	propsRender := regexp.MustCompile(`return\s+\w+\s*,\s*([a-zA-Z_]\w*)\(`)
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".templ") {
			continue
		}
		file := filepath.Join(dir, e.Name())
		text, ok := read(file)
		if !ok {
			continue
		}
		at := strings.Index(text, needle)
		if at < 0 {
			continue
		}
		start := strings.LastIndex(text[:at], "add(Story{")
		if start < 0 {
			continue
		}
		end := strings.Index(text[at:], "})")
		if end < 0 {
			continue
		}
		end += at + 2
		code := text[start:end]
		out := []Source{{Title: "story", Path: rel(file), Line: 1 + strings.Count(text[:start], "\n"), Lang: "go", Code: code}}
		m := render.FindStringSubmatch(code)
		if editable := propsRender.FindStringSubmatch(code); len(editable) == 2 {
			m = editable
		}
		if len(m) == 2 {
			if src, ok := templBlock(file, m[1]); ok {
				src.Title = "render: " + m[1]
				out = append(out, src)
				out = refs(out, out[1:], dir, map[string]bool{}, 1)
			}
		}
		return out
	}
	return nil
}

// demoSources finds the book-only demonstrations, whose registrations use
// loops to share the spike's variant data instead of a separate add call.
func demoSources(id string) []Source {
	dir := filepath.Join(root, "internal", "stories")
	var file, render string
	switch {
	case strings.HasPrefix(id, "code/"):
		file, render = "book_code.go", "codeDemo"
	case strings.HasPrefix(id, "separator/"):
		file, render = "book_misc.templ", "separatorDemo"
	case strings.HasPrefix(id, "icons/"):
		file, render = "book_misc.templ", "iconsDemo"
	case strings.HasPrefix(id, "tabs-demo/"):
		file, render = "book_parts.templ", "tabsDemo"
	case strings.HasPrefix(id, "nav-list/"):
		file, render = "book_parts.templ", "navDemo"
	case strings.HasPrefix(id, "controls/"):
		file, render = "book_parts.templ", "controlsDemo"
	case strings.HasPrefix(id, "event-log/"):
		file, render = "book_parts.templ", "logDemo"
	default:
		return nil
	}
	path := filepath.Join(dir, file)
	text, ok := read(path)
	if !ok {
		return nil
	}
	at := strings.Index(text, `"`+id+`"`)
	if at < 0 {
		return nil
	}
	start := strings.LastIndex(text[:at], "\n") + 1
	end := strings.Index(text[at:], "\n")
	if end < 0 {
		end = len(text)
	} else {
		end += at
	}
	out := []Source{{Title: "story", Path: rel(path), Line: 1 + strings.Count(text[:start], "\n"), Lang: "go", Code: strings.TrimSpace(text[start:end])}}
	var src Source
	if strings.HasSuffix(file, ".templ") {
		src, ok = templBlock(path, render)
	} else {
		src, ok = goDecl(path, render)
	}
	if ok {
		src.Title = "render: " + render
		out = append(out, src)
		out = refs(out, out[1:], dir, map[string]bool{}, 1)
	}
	return out
}

// propTypes adds the module's types the props hold (a field's type, through
// pointers and slices: ui.Tabs, ui.Popup), and the templ methods of each
// that the render calls (p.Tabs.Show(…) shows `templ (t *Tabs) Show`): a
// kit component kept in the app's state shows as its state and its views.
// The runtime (native), the storybook's own packages and internal ones are
// left out.
func (s *Story) propTypes(out []Source, render Source, seen map[string]bool) []Source {
	t := reflect.TypeOf(s.props)
	if t == nil {
		return out
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return out
	}
	calls := methodCall.FindAllStringSubmatch(render.Code, -1)
	for i := range t.NumField() {
		ft := t.Field(i).Type
		for ft.Kind() == reflect.Pointer || ft.Kind() == reflect.Slice {
			ft = ft.Elem()
		}
		pkg := ft.PkgPath()
		if ft.Name() == "" || !strings.HasPrefix(pkg, module+"/") || pkg == module+"/native" ||
			strings.HasPrefix(pkg, module+"/storybook") || strings.Contains(pkg, "/internal/") {
			continue
		}
		dir := filepath.Join(root, strings.TrimPrefix(pkg, module+"/"))
		short := pkg[strings.LastIndex(pkg, "/")+1:]
		if c, ok := find(dir, ft.Name(), seen); ok {
			c.Title = short + "." + ft.Name()
			out = append(out, c)
		}
		for _, m := range calls {
			if c, ok := findMethod(dir, ft.Name(), m[1], seen); ok {
				c.Title = short + "." + ft.Name() + "." + m[1]
				out = append(out, c)
			}
		}
	}
	return out
}

// findMethod is the templ method name of typ in the package in dir:
// `templ (x *typ) name(…) { … }`.
func findMethod(dir, typ, name string, seen map[string]bool) (Source, bool) {
	key := dir + "." + typ + "." + name
	if seen[key] {
		return Source{}, false
	}
	seen[key] = true
	head := regexp.MustCompile(`^templ \(\w+ \*?` + regexp.QuoteMeta(typ) + `\) ` + regexp.QuoteMeta(name) + `\(`)
	ents, _ := os.ReadDir(dir)
	for _, e := range ents {
		if strings.HasSuffix(e.Name(), ".templ") {
			if s, ok := block(filepath.Join(dir, e.Name()), head.MatchString); ok {
				return s, true
			}
		}
	}
	return Source{}, false
}

var (
	qualified = regexp.MustCompile(`([a-zA-Z_]\w*)\.([A-Z]\w*)\b`)
	localCall = regexp.MustCompile(`@([a-zA-Z_]\w*)\(`)
	// methodCall is a call of an exported method: .Show( in @p.Tabs.Show(.
	methodCall = regexp.MustCompile(`\.([A-Z]\w*)\(`)
)

// refs adds the components the code in from calls: pkg.Name in any of
// them, and @name (a templ component of the same package) in templ.
func refs(out, from []Source, dir string, seen map[string]bool, depth int) []Source {
	var next []Source
	for _, src := range from {
		imports := importsOf(filepath.Join(root, src.Path))
		for _, m := range qualified.FindAllStringSubmatch(src.Code, -1) {
			path, ok := imports[m[1]]
			if !ok || !strings.HasPrefix(path, module+"/") || path == module+"/native" || path == module+"/storybook" {
				continue
			}
			pdir := filepath.Join(root, strings.TrimPrefix(path, module+"/"))
			if c, ok := find(pdir, m[2], seen); ok {
				c.Title = m[1] + "." + m[2]
				out = append(out, c)
			}
		}
		if src.Lang != "templ" {
			continue
		}
		for _, m := range localCall.FindAllStringSubmatch(src.Code, -1) {
			if c, ok := find(filepath.Dir(filepath.Join(root, src.Path)), m[1], seen); ok {
				c.Title = m[1]
				out = append(out, c)
				next = append(next, c)
			}
		}
	}
	if depth > 0 && len(next) > 0 {
		return refs(out, next, dir, seen, depth-1)
	}
	return out
}

// find is name in the package in dir: a templ component, a func or a type.
func find(dir, name string, seen map[string]bool) (Source, bool) {
	key := dir + "." + name
	if seen[key] {
		return Source{}, false
	}
	seen[key] = true
	ents, _ := os.ReadDir(dir)
	for _, e := range ents {
		if strings.HasSuffix(e.Name(), ".templ") {
			if s, ok := templBlock(filepath.Join(dir, e.Name()), name); ok {
				return s, true
			}
		}
	}
	for _, e := range ents {
		n := e.Name()
		if strings.HasSuffix(n, ".go") && !strings.HasSuffix(n, "_templ.go") && !strings.HasSuffix(n, "_test.go") {
			if s, ok := goDecl(filepath.Join(dir, n), name); ok {
				return s, true
			}
		}
	}
	return Source{}, false
}

// templBlock is `templ name(…) { … }` in the .templ file, with the comment
// above it: to the first line that is only "}".
func templBlock(file, name string) (Source, bool) {
	return block(file, func(l string) bool { return strings.HasPrefix(l, "templ "+name+"(") })
}

// block is the templ block whose first line is the one head matches.
func block(file string, head func(string) bool) (Source, bool) {
	text, ok := read(file)
	if !ok {
		return Source{}, false
	}
	lines := strings.Split(text, "\n")
	start := -1
	for i, l := range lines {
		if head(l) {
			start = i
			break
		}
	}
	if start < 0 {
		return Source{}, false
	}
	end := start
	for end < len(lines) && strings.TrimRight(lines[end], " \t") != "}" {
		end++
	}
	for start > 0 && strings.HasPrefix(lines[start-1], "//") {
		start--
	}
	return Source{Path: rel(file), Line: start + 1, Lang: "templ",
		Code: strings.Join(lines[start:min(end+1, len(lines))], "\n")}, true
}

// goDecl is the top-level func or type name in the Go file, with its doc.
func goDecl(file, name string) (Source, bool) {
	f, fset, text, ok := parse(file)
	if !ok {
		return Source{}, false
	}
	for _, d := range f.Decls {
		var from, to token.Pos
		switch d := d.(type) {
		case *ast.FuncDecl:
			if d.Name.Name != name || d.Recv != nil {
				continue
			}
			from, to = d.Pos(), d.End()
			if d.Doc != nil {
				from = d.Doc.Pos()
			}
		case *ast.GenDecl:
			for _, sp := range d.Specs {
				if ts, ok := sp.(*ast.TypeSpec); ok && ts.Name.Name == name {
					from, to = d.Pos(), d.End()
					if d.Doc != nil {
						from = d.Doc.Pos()
					}
				}
			}
		}
		if from.IsValid() {
			return Source{Path: rel(file), Line: fset.Position(from).Line, Lang: "go",
				Code: text[fset.Position(from).Offset:fset.Position(to).Offset]}, true
		}
	}
	return Source{}, false
}

// callAt is the statement or declaration at line in the Go file: the
// smallest one that holds it, as the story's Add call.
func callAt(file string, line int) (Source, bool) {
	f, fset, text, ok := parse(file)
	if !ok {
		return Source{}, false
	}
	var best ast.Node
	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		a, b := fset.Position(n.Pos()).Line, fset.Position(n.End()).Line
		if line < a || line > b {
			return false
		}
		switch n.(type) {
		case ast.Stmt, *ast.ValueSpec:
			if _, block := n.(*ast.BlockStmt); !block {
				best = n
			}
		}
		return true
	})
	if best == nil {
		return Source{}, false
	}
	from, to := fset.Position(best.Pos()), fset.Position(best.End())
	// Back to the start of the line, so the first line is indented as the
	// rest, and then out of it.
	off := strings.LastIndexByte(text[:from.Offset], '\n') + 1
	return Source{Path: rel(file), Line: from.Line, Lang: "go", Code: dedent(text[off:to.Offset])}, true
}

func dedent(s string) string {
	lines := strings.Split(s, "\n")
	cut := -1
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		n := len(l) - len(strings.TrimLeft(l, "\t "))
		if cut < 0 || n < cut {
			cut = n
		}
	}
	for i, l := range lines {
		if len(l) >= cut && cut > 0 {
			lines[i] = l[cut:]
		}
	}
	return strings.Join(lines, "\n")
}

// importsOf is the file's imports, by the name they are used by.
func importsOf(file string) map[string]string {
	m := map[string]string{}
	text, ok := read(file)
	if !ok {
		return m
	}
	if strings.HasSuffix(file, ".templ") {
		// templ's imports are Go's: parse them as a Go file's head.
		if i := strings.Index(text, "\ntempl "); i >= 0 {
			text = text[:i]
		}
	}
	f, err := parser.ParseFile(token.NewFileSet(), "", text, parser.ImportsOnly)
	if err != nil {
		return m
	}
	for _, im := range f.Imports {
		p := strings.Trim(im.Path.Value, `"`)
		n := p[strings.LastIndex(p, "/")+1:]
		if im.Name != nil {
			n = im.Name.Name
		}
		m[n] = p
	}
	return m
}

func rel(file string) string {
	if r, err := filepath.Rel(root, file); err == nil {
		return r
	}
	return file
}

// read is a file's text, read again when it changes (its time or size).
func read(file string) (string, bool) {
	st, err := os.Stat(file)
	if err != nil {
		return "", false
	}
	filesMu.Lock()
	defer filesMu.Unlock()
	if c, ok := files[file]; ok && c.mod.Equal(st.ModTime()) && c.size == st.Size() {
		return c.text, true
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return "", false
	}
	files[file] = &cached{mod: st.ModTime(), size: st.Size(), text: string(b)}
	return string(b), true
}

// parse is a Go file parsed, again when it changes.
func parse(file string) (*ast.File, *token.FileSet, string, bool) {
	text, ok := read(file)
	if !ok {
		return nil, nil, "", false
	}
	filesMu.Lock()
	c := files[file]
	filesMu.Unlock()
	if c.ast == nil {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, text, parser.ParseComments)
		if err != nil {
			return nil, nil, "", false
		}
		c.ast, c.fset = f, fset
	}
	return c.ast, c.fset, text, true
}

type cached struct {
	mod  time.Time
	size int64
	text string
	ast  *ast.File
	fset *token.FileSet
}

var (
	filesMu sync.Mutex
	files   = map[string]*cached{}
)
