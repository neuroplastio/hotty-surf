// Command stories writes the stories as documents that `hotty render`
// and Chromium open: each story's HTML with the stylesheet inline, its
// focused and hovered states forced (stories.Force).
//
//	stories -list
//	stories -out DIR [-theme neuroplastio] [-density dense]
//
// With -gallery it writes the gallery's stories instead, every story its
// catalog lists (each widget's own, stories.All's, and the stories
// files' composites and screens), each as the gallery draws it: the
// whole gallery with the story picked, its canvas at the story's width,
// in each of the gallery's modes, interpreted and compiled. The legible
// gate (scripts/gate/legible.cjs) measures them.
//
//	stories -gallery -out DIR [-theme neuroplastio] [-density compact] [-width N]
//
// writes DIR/<mode>/<story id>.html and DIR/index.tsv, a line for each:
// its mode, id, file, declared width, canvas width, window (cols and
// rows), section and title. -width draws every canvas N cells wide, not
// at its story's width: the legible gate's calibration.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/neuroplastio/hotty-surf/internal/gallery"
	"github.com/neuroplastio/hotty-surf/internal/stories"
	"github.com/neuroplastio/hotty-surf/surfkeys"
	"github.com/neuroplastio/hotty-surf/surflive"
	"github.com/neuroplastio/hotty-surf/surfui"
)

func main() {
	list := flag.Bool("list", false, "list the stories and their widths")
	out := flag.String("out", "", "write every story to DIR/<name>.html")
	theme := flag.String("theme", "neuroplastio", "the theme: neuroplastio, terminal, paper")
	density := flag.String("density", "dense", "the density: dense, roomy, compact (with -gallery, compact, the gallery's)")
	ofGallery := flag.Bool("gallery", false, "write the gallery's stories, each as the gallery draws it, in each of its modes")
	width := flag.Int("width", 0, "with -gallery, draw every canvas `cells` wide, not at its story's width")
	flag.Parse()
	if *ofGallery {
		if *list || *out == "" {
			flag.Usage()
			os.Exit(2)
		}
		d := "compact"
		flag.Visit(func(f *flag.Flag) {
			if f.Name == "density" {
				d = *density
			}
		})
		if err := writeGallery(*out, *theme, d, *width); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *list {
		for _, s := range stories.All {
			fmt.Printf("%s\t%d\n", s.Name, s.Cols)
		}
		return
	}
	if *out == "" {
		flag.Usage()
		os.Exit(2)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, s := range stories.All {
		root := stories.Surface(s, *theme, surfui.Density(*density))
		surfui.Resolve(root, "r")
		name := strings.ReplaceAll(s.Name, "/", "-")
		if err := os.WriteFile(filepath.Join(*out, name+".html"), []byte(stories.Document(root)), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

// The window the gallery is drawn in: 1796×1153 logical px at 1.6, in
// the gate's cells of 10×21 CSS px; wider where a story is wider than
// the canvas it leaves (gallery.Window).
const winCols, winRows = 179, 54

// writeGallery writes each of the gallery's stories, in each mode, into
// dir, as the gallery draws it in a window: the whole gallery, its
// stylesheet inline, the story picked at the theme and density.
func writeGallery(dir, theme, density string, width int) error {
	modes := []struct {
		name string
		of   func(*gallery.Gallery) surflive.Component
	}{{"interpreted", gallery.Interpreted}, {"compiled", gallery.Compiled}}
	var index strings.Builder
	for _, m := range modes {
		if err := os.MkdirAll(filepath.Join(dir, m.name), 0o755); err != nil {
			return err
		}
		g := gallery.New()
		app := surflive.New(m.of(g))
		css := gallery.Options(g).CSS
		section := ""
		for _, e := range g.Shown() {
			if e.Kind == "section" {
				section = e.Label
			}
			if e.Kind != "story" {
				continue
			}
			g.Pick(e.ID)
			g.SetTheme(theme)
			g.SetDensity(density)
			cols := g.Width
			if width > 0 {
				g.SetWidth(strconv.Itoa(width))
			}
			wcols := max(winCols, gallery.Window(g.Width))
			g.Resize(wcols, winRows)
			if fit := g.Fit(); fit != "" {
				return fmt.Errorf("%s: the canvas is narrowed in a window %d wide: %s", e.ID, wcols, fit)
			}
			tree := app.Render()
			surfkeys.Expand(tree, gallery.Options(g).KeyStyle, true)
			surfui.Resolve(tree, "r")
			doc := `<!doctype html><html><head><meta charset="utf-8"><style>` + css + `</style></head><body>` +
				surfui.HTML(tree) + `</body></html>`
			file := filepath.Join(m.name, e.ID+".html")
			if err := os.WriteFile(filepath.Join(dir, file), []byte(doc), 0o644); err != nil {
				return err
			}
			fmt.Fprintf(&index, "%s\t%s\t%s\t%d\t%d\t%d\t%d\t%s\t%s\n", m.name, e.ID, file, cols, g.Width, wcols, winRows, section, g.Title())
		}
	}
	return os.WriteFile(filepath.Join(dir, "index.tsv"), []byte(index.String()), 0o644)
}
