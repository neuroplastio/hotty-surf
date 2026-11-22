package devcmd

import (
	"bytes"
	"go/format"
	"io/fs"
	"strings"
	"testing"
)

// Every Go file the scaffold writes is gofmt's already, as `hotty-surf new`
// renders it.
func TestScaffoldIsFormatted(t *testing.T) {
	n := 0
	err := fs.WalkDir(scaffold, "scaffold", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go.tmpl") {
			return err
		}
		n++
		b, err := fs.ReadFile(scaffold, p)
		if err != nil {
			return err
		}
		src := []byte(strings.ReplaceAll(string(b), "APPNAME", "demo"))
		out, err := format.Source(src)
		if err != nil {
			t.Errorf("%s: %v", p, err)
			return nil
		}
		if !bytes.Equal(out, src) {
			t.Errorf("%s isn't gofmt's once rendered", p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("no Go templates in the scaffold")
	}
}
