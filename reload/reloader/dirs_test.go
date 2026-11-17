package reloader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fsnotify/fsnotify"
)

// TestStructural: what makes the next build run go list again, and what
// doesn't.
func TestStructural(t *testing.T) {
	root := t.TempDir()
	pkg := filepath.Join(root, "pkg")
	made := filepath.Join(root, "made")
	for _, d := range []string{pkg, made} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	l := &loop{dirs: []string{root, pkg}, modFile: filepath.Join(root, "go.mod"), workFile: filepath.Join(root, "go.work")}
	for _, c := range []struct {
		name string
		op   fsnotify.Op
		want bool
	}{
		{made, fsnotify.Create, true},
		{filepath.Join(root, "main.go"), fsnotify.Create, false},
		{filepath.Join(root, "main.go"), fsnotify.Write, false},
		{pkg, fsnotify.Remove, true},
		{pkg, fsnotify.Rename, true},
		{filepath.Join(pkg, "a.go"), fsnotify.Remove, false},
		{filepath.Join(root, "go.mod"), fsnotify.Write, true},
		{filepath.Join(root, "go.work"), fsnotify.Write, true},
		{filepath.Join(root, "go.sum"), fsnotify.Write, false},
	} {
		if got := l.structural(fsnotify.Event{Name: c.name, Op: c.op}); got != c.want {
			t.Errorf("%v %s: %v, want %v", c.op, c.name, got, c.want)
		}
	}
}

// TestImportsChanged: a Go file's imports are noted when first read, and
// a change of them, not of the rest of the file, is news.
func TestImportsChanged(t *testing.T) {
	f := filepath.Join(t.TempDir(), "a.go")
	l := &loop{imports: map[string]string{}}
	for i, c := range []struct {
		src  string
		want bool
	}{
		{"package a\n\nimport \"fmt\"\n\nvar _ = fmt.Sprint\n", false},
		{"package a\n\nimport \"fmt\"\n\nvar _ = fmt.Sprintln\n", false},
		{"package a\n\nimport (\n\t\"fmt\"\n\t\"example.com/app/lib\"\n)\n", true},
		{"package a\n\nimport (\n\t\"example.com/app/lib\"\n\t\"fmt\"\n)\n", false},
		{"package a\n", true},
	} {
		if err := os.WriteFile(f, []byte(c.src), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := l.importsChanged(f, nil); got != c.want {
			t.Errorf("%d: %v, want %v", i, got, c.want)
		}
	}
}
