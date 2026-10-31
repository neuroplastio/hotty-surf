//go:build !tinygo

package conformance

import (
	"encoding/json"
	"io/fs"
	"testing"

	"golang.org/x/tools/txtar"
)

// The cases' trees read as encoding/json reads them, with its own
// recursion (a Branch's fields copied into a type with no
// UnmarshalJSON). Not under TinyGo, where that recursion is what
// Branch's UnmarshalJSON is for.
func TestBranchJSON(t *testing.T) {
	type plain struct {
		ID, Name, Via string
		Kids          []*plain
		Next          *plain
	}
	var same func(b *Branch, p *plain) bool
	same = func(b *Branch, p *plain) bool {
		if b == nil || p == nil {
			return b == nil && p == nil
		}
		if b.ID != p.ID || b.Name != p.Name || b.Via != p.Via || len(b.Kids) != len(p.Kids) || !same(b.Next, p.Next) {
			return false
		}
		for i := range b.Kids {
			if !same(b.Kids[i], p.Kids[i]) {
				return false
			}
		}
		return true
	}
	files, _ := fs.Glob(Cases, "cases/*.txt")
	n := 0
	for _, file := range files {
		src, _ := fs.ReadFile(Cases, file)
		for _, f := range txtar.Parse(src).Files {
			if f.Name != "state.json" {
				continue
			}
			var b struct{ Trees []*Branch }
			var p struct{ Trees []*plain }
			if err := json.Unmarshal(f.Data, &b); err != nil {
				t.Fatalf("%s: %v", file, err)
			}
			if err := json.Unmarshal(f.Data, &p); err != nil {
				t.Fatalf("%s: %v", file, err)
			}
			if len(b.Trees) != len(p.Trees) {
				t.Fatalf("%s: %d trees, not %d", file, len(b.Trees), len(p.Trees))
			}
			for i := range b.Trees {
				if !same(b.Trees[i], p.Trees[i]) {
					t.Errorf("%s: tree %d reads otherwise", file, i)
				}
				n++
			}
		}
	}
	if n < 4 {
		t.Errorf("%d trees in the cases, not deep.txt's and toodeep.txt's 4", n)
	}
}
