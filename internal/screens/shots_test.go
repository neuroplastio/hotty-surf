package screens

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/neuroplastio/hotty-surf/internal/screens/demodata"
	"github.com/neuroplastio/hotty-surf/surflive"
)

var shots = flag.String("shots", "", "write the screens' documents, in the states worth a look, to DIR")

// TestShots writes documents, on the mock, for scripts/render.sh:
//
//	go test ./screens -run TestShots -shots $PWD/.shots
//	scripts/render.sh .shots/surf-palette.html 64
func TestShots(t *testing.T) {
	if *shots == "" {
		t.Skip("no -shots")
	}
	write := func(name string, root surflive.Component) {
		doc := Document(surflive.New(root), "")
		if err := os.WriteFile(filepath.Join(*shots, "surf-"+name+".html"), []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	p := NewPalette(demodata.Mock())
	write("palette", surfInterpPalette(p))
	p.SetQuery("split")
	write("palette-query", surfInterpPalette(p))
	p.TogglePanel()
	write("palette-panel", surfInterpPalette(p))
	p.SetQuery("qqqq")
	write("palette-empty", surfInterpPalette(p))
	p.setLine("config.set palette.style.scale 30")
	p.Enter()
	write("palette-refused", surfInterpPalette(p))

	s := NewSettings(demodata.Mock())
	write("settings", surfInterpSettings(s))
	s.Pick(2)
	s.set("dock.controls", "false")
	write("settings-tools", surfInterpSettings(s))
	s.Search("palette")
	write("settings-search", surfInterpSettings(s))

	src := demodata.Mock()
	l := NewAgents(src)
	list, _ := src.Agents()
	l.Loaded = true
	l.answer(list, nil)
	write("agents", surfInterpAgents(l))
	l.Refused = "no agent gone-1 now"
	write("agents-refused", surfInterpAgents(l))
	l.answer(nil, nil)
	l.Refused = ""
	write("agents-empty", surfInterpAgents(l))
}
