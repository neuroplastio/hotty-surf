package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

//go:embed pack.html
var packHTML string

const module = "neuroplast.io/hotty-surf"

type multi []string

func (m *multi) String() string     { return strings.Join(*m, ",") }
func (m *multi) Set(s string) error { *m = append(*m, s); return nil }

type name struct {
	Key, Line, Reason string
	Added             bool
}

type pkgView struct {
	Path, Label string
	Names       []name
	Doc         string
	Examples    []file
}

type file struct{ Name, Text string }

type page struct {
	Title, Date, Commit string
	Added, Removed      int
	Missing             []string
	Notes               template.HTML
	Packages            []pkgView
	Files               []file
	Appendix            []template.HTML
}

// packCmd writes a review pack: one HTML page with, for each package, the
// names not yet approved and their reasons, its examples and its go doc, and
// the extra files and notes given. It fails, after writing the page, when a
// name has no reason.
func packCmd(args []string) error {
	fs := flag.NewFlagSet("pack", flag.ExitOnError)
	approvedPath := fs.String("approved", "", "the approved API")
	nextPath := fs.String("next", "", "the API now")
	reasonsPath := fs.String("reasons", "", "api/reasons.md")
	out := fs.String("o", "", "the page")
	title := fs.String("title", "API pack", "the page's title")
	notes := fs.String("notes", "", "markdown shown first")
	var appendix, files multi
	fs.Var(&appendix, "appendix", "markdown shown last (repeatable)")
	fs.Var(&files, "file", "a file shown whole, as an example, from the first dir (repeatable)")
	fs.Parse(args)
	dirs := fs.Args()

	approved, err := readLines(*approvedPath)
	if err != nil {
		return err
	}
	next, err := readLines(*nextPath)
	if err != nil {
		return err
	}
	reasons, err := readReasons(*reasonsPath)
	if err != nil {
		return err
	}
	p := page{Title: *title, Date: time.Now().Format("2006-01-02")}
	if c, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output(); err == nil {
		p.Commit = strings.TrimSpace(string(c))
	}
	byPkg := map[string]*pkgView{}
	var order []string
	for _, c := range diff(approved, next) {
		path, key := keyOf(c.line)
		v := byPkg[path]
		if v == nil {
			v = &pkgView{Path: path, Label: label(path)}
			byPkg[path] = v
			order = append(order, path)
		}
		r := reasons[key]
		if c.added {
			p.Added++
			if r == "" {
				p.Missing = append(p.Missing, key)
			}
		} else {
			p.Removed++
		}
		v.Names = append(v.Names, name{Key: key, Line: strings.TrimPrefix(c.line, "pkg "+path+", "), Reason: r, Added: c.added})
	}
	slices.SortFunc(order, func(a, b string) int { return strings.Compare(label(a), label(b)) })
	for _, path := range order {
		v := byPkg[path]
		v.Doc, v.Examples = docOf(path, dirs)
		p.Packages = append(p.Packages, *v)
	}
	for _, f := range files {
		b, err := os.ReadFile(filepath.Join(dirs[0], f)) // relative to the module's root
		if err != nil {
			return err
		}
		p.Files = append(p.Files, file{f, string(b)})
	}
	md := goldmark.New(goldmark.WithExtensions(extension.GFM))
	render := func(path string) (template.HTML, error) {
		b, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		// A docs page's front matter is the site's, not the pack's.
		if rest, ok := bytes.CutPrefix(b, []byte("---\n")); ok {
			if _, body, ok := bytes.Cut(rest, []byte("\n---\n")); ok {
				b = body
			}
		}
		var h bytes.Buffer
		if err := md.Convert(b, &h); err != nil {
			return "", err
		}
		return template.HTML(h.String()), nil
	}
	if *notes != "" {
		if p.Notes, err = render(*notes); err != nil {
			return err
		}
	}
	for _, a := range appendix {
		h, err := render(a)
		if err != nil {
			return err
		}
		p.Appendix = append(p.Appendix, h)
	}
	t := template.Must(template.New("pack").Parse(packHTML))
	var b bytes.Buffer
	if err := t.Execute(&b, p); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(*out, b.Bytes(), 0o644); err != nil {
		return err
	}
	if len(p.Missing) > 0 {
		return fmt.Errorf("%d names have no reason in %s: %s", len(p.Missing), *reasonsPath, strings.Join(p.Missing, ", "))
	}
	return nil
}

var lineRE = regexp.MustCompile(`^pkg ([^,]+), (func|method|type|const|var) (?:\((\*?)(\w+)\) )?(\w+)(.*)$`)
var memberRE = regexp.MustCompile(`^ ?\S*\s*(struct|interface), (\w+)`)

// keyOf is a line's package path and its name's key: label.Name, or
// label.Type.Member for a method or a field.
func keyOf(line string) (path, key string) {
	m := lineRE.FindStringSubmatch(line)
	if m == nil {
		return "?", line
	}
	path = m[1]
	l := label(path)
	switch m[2] {
	case "method":
		return path, l + "." + m[4] + "." + m[5]
	case "type":
		rest := m[6]
		if i := strings.Index(rest, "["); i == 0 { // type parameters
			rest = rest[strings.Index(rest, "]")+1:]
		}
		if mm := memberRE.FindStringSubmatch(rest); mm != nil {
			return path, l + "." + m[5] + "." + mm[2]
		}
	}
	return path, l + "." + m[5]
}

// label is how the pack and reasons.md name a package: surf for the root,
// its path in the module otherwise (ui/code).
func label(path string) string {
	if path == module {
		return "surf"
	}
	return strings.TrimPrefix(path, module+"/")
}

var reasonRE = regexp.MustCompile("^\\s*[-*] `([^`]+)`:\\s*(.+)$")

// readReasons reads reasons.md: a line "- `key`: why" for each name.
func readReasons(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, l := range strings.Split(string(b), "\n") {
		if m := reasonRE.FindStringSubmatch(l); m != nil {
			out[m[1]] = m[2]
		}
	}
	return out, nil
}

// docOf is go doc -all for path and its example files, from whichever of
// dirs holds it.
func docOf(path string, dirs []string) (string, []file) {
	for _, d := range dirs {
		dir, err := exec.Command("go", "-C", d, "list", "-f", "{{.Dir}}", path).Output()
		if err != nil {
			continue
		}
		doc, _ := exec.Command("go", "-C", d, "doc", "-all", path).Output()
		var exs []file
		ms, _ := filepath.Glob(filepath.Join(strings.TrimSpace(string(dir)), "*example*_test.go"))
		for _, m := range ms {
			b, _ := os.ReadFile(m)
			exs = append(exs, file{filepath.Base(m), string(b)})
		}
		return string(doc), exs
	}
	return "(go doc found nothing)", nil
}
