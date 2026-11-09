package surfgen

import (
	"io/fs"
	"path"
	"strings"
	"testing"

	"golang.org/x/tools/txtar"
	"neuroplast.io/hotty-surf/pkg/conformance"
)

// The compiled mode loads its templates when surfgen runs: given a
// conformance case's templates that don't load (its load file), with the
// fixtures' Go types, surfgen fails with the case's errors, as the
// interpreted mode's Load does (conformance's TestCases).
func TestConformanceLoad(t *testing.T) {
	files, _ := fs.Glob(conformance.Cases, "cases/*.txt")
	ran := 0
	for _, file := range files {
		src, err := fs.ReadFile(conformance.Cases, file)
		if err != nil {
			t.Fatal(err)
		}
		ar := txtar.Parse(src)
		want, ok := conformance.File(ar, "load")
		if !ok {
			continue
		}
		ran++
		t.Run(strings.TrimSuffix(path.Base(file), ".txt"), func(t *testing.T) {
			_, err := generateFS("../conformance", conformance.Overlay(ar), false)
			got := "<nil>\n"
			if err != nil {
				got = err.Error() + "\n"
			}
			if got != string(want) {
				t.Errorf("surfgen:\n got %s\nwant %s", got, want)
			}
		})
	}
	if ran == 0 {
		t.Fatal("no case has a load file")
	}
}
