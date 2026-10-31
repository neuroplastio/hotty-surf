package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// An unexported struct with a template is a component, as one exported
// is: type notices struct and notices.surf make surfInterpNotices.
func TestUnexported(t *testing.T) {
	dir := t.TempDir()
	for name, src := range map[string]string{
		"go.mod":       "module example.com/notices\n\ngo 1.26\n",
		"notices.go":   "package notices\n\nimport \"time\"\n\ntype notices struct {\n\tSel  string\n\tWait time.Duration\n}\n\nfunc (n *notices) Pick(id string) { n.Sel = id }\n",
		"notices.surf": `<surface><button @click="pick('a')">{{ sel }}</button></surface>`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	out, err := generate(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"func surfInterpNotices(c *notices) surflive.Component", "func surfCompiledNotices(c *notices) surflive.Component", `surf.Shape{Name: "notices", Component: true`, `{Name: "wait", Type: cellite.Int, Go: "time.Duration"}`} {
		if !strings.Contains(string(out), want) {
			t.Errorf("surf_gen.go has no %s:\n%s", want, out)
		}
	}
	if strings.Contains(string(out), "SurfPackage") {
		t.Errorf("surf_gen.go exports SurfPackage without -export")
	}
	out, err = generate(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"func SurfPackage() surf.Package", `Name: "notices", Files: surfFiles`, "case *notices:"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("surf_gen.go with -export has no %s:\n%s", want, out)
		}
	}
}
