package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/neuroplastio/hotty-surf/surflive"
	"github.com/neuroplastio/hotty-surf/surfrun"
	"github.com/neuroplastio/hotty-surf/surftest"
)

// The first screen: a typed name changes the heading, a click counts,
// and a click sends one op.
func TestHello(t *testing.T) {
	h := &Hello{Name: "world"}
	a := surftest.Start(t, surflive.New(surfInterpHello(h)),
		surfrun.Options{Name: "hello"})
	if got := a.Surface().TextOf("r.0.0"); got != "Hello, world!" {
		t.Fatalf("the heading is %q:\n%s", got, a.Surface().HTML())
	}

	if err := a.Host.Fill(a.Name, "r.0.1.kname", "Ada"); err != nil {
		t.Fatal(err)
	}
	a.Wait("the heading greets Ada", func() bool {
		return a.Surface().TextOf("r.0.0") == "Hello, Ada!"
	})

	before := len(a.Ops())
	if err := a.Host.Click(a.Name, "r.0.1.kadd"); err != nil {
		t.Fatal(err)
	}
	a.Wait("one click", func() bool {
		return a.Surface().TextOf("r.0.kcount") == "1 clicks"
	})
	ops := a.Ops()[before:]
	if len(ops) != 1 || ops[0].Op != "text" || ops[0].Target != "r.0.kcount" {
		t.Errorf("the click sent %v, want one text op on r.0.kcount", ops)
	}
	var n int
	a.Call(func() { n = h.Count }) // the state, read on the runtime's goroutine
	if n != 1 {
		t.Errorf("Count is %d, want 1", n)
	}
}

// With no host, the same state is drawn in cells, and back.
func TestCells(t *testing.T) {
	h := &Hello{Name: "world"}
	a := surftest.Start(t, surflive.New(surfInterpHello(h)),
		surfrun.Options{Name: "hello"})
	a.Call(a.Run.Fail) // as a host that fails
	a.Wait("the cells", func() bool { return strings.Contains(a.Host.Screen(), "Hello, world!") })
	a.Host.Key("s")
	a.Host.Key("enter")
	a.Wait("a letter and a count", func() bool {
		s := a.Host.Screen()
		return strings.Contains(s, "Hello, worlds!") && strings.Contains(s, "1 clicks")
	})
	a.Call(a.Run.Recover)
	a.Wait("the surface again", func() bool {
		s := a.Surface()
		return s != nil && s.TextOf("r.0.0") == "Hello, worlds!"
	})
}

// Hot reload: a template changed on disk is drawn at once, with the
// state as it was.
func TestHotReload(t *testing.T) {
	dir := t.TempDir()
	files, _ := filepath.Glob("*.surf")
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, f), src, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	h := &Hello{Name: "world", Count: 2}
	m, err := reloading(h, dir)
	if err != nil {
		t.Fatal(err)
	}
	a := surftest.Start(t, m, surfrun.Options{Name: "hello"})
	src, _ := os.ReadFile(filepath.Join(dir, "hello.surf"))
	src = []byte(strings.Replace(string(src), "Hello, {{ name }}!", "Hi, {{ name }}: {{ count }}", 1))
	if err := os.WriteFile(filepath.Join(dir, "hello.surf"), src, 0o644); err != nil {
		t.Fatal(err)
	}
	a.Wait("the new heading", func() bool { return a.Surface().TextOf("r.0.0") == "Hi, world: 2" })
}

// The guide's code is these files: a block in docs/guide.md whose first
// line names a file here (`// examples/hello/hello.go`), or of
// internal/stories/ or internal/gallery/ (Writing a story), is that file, or a part of it. So is a
// block of the README's that names one.
func TestGuide(t *testing.T) {
	block := regexp.MustCompile("(?s)```[a-z]*\n((?://|<!--) " +
		"((?:examples/hello|internal/stories|internal/gallery)/[^ \n]+)[^\n]*\n)(.*?)```")
	for _, doc := range []string{"docs/guide.md", "README.md"} {
		text, err := os.ReadFile("../../" + doc)
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, m := range block.FindAllStringSubmatch(string(text), -1) {
			name, body := m[2], m[3]
			if strings.HasPrefix(name, "examples/hello/") {
				n++
			}
			src, err := os.ReadFile("../../" + name)
			if err != nil {
				t.Errorf("%s quotes %s: %v", doc, name, err)
				continue
			}
			if !strings.Contains(string(src), body) {
				t.Errorf("%s's block from %s is not in the file:\n%s", doc, name, body)
			}
		}
		if n < 2 {
			t.Errorf("%s quotes %d blocks of examples/hello, want its first screen's", doc, n)
		}
	}
}
