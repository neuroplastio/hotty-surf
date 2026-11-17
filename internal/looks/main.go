// Command looks is ui's looks pass: each story of the components named,
// drawn by hotty-blitz (hotty render, as a host draws a surface) and by
// headless Chromium from the same document, in a theme, and compared.
//
//	go run ./internal/looks -hotty <hotty> -fonts <hotty-blitz fonts dir> -out DIR [-theme dark|light] [-scale 2] [-cols 80] [-rows auto] <component>...
//	go run ./internal/looks … -files <file.html>...
//
// For each story it writes, under DIR:
//   - <id>.html, the document the surface sends, in the theme's class;
//   - <id>.blitz.png, hotty render's, as many rows as it takes;
//   - <id>.chromium.png, Chromium's at the same size, with the host's
//     stylesheet (hotty css) in a layer under the document and hotty-blitz's
//     own fonts, so both engines draw the same glyphs;
//   - <id>.sheet.png, the two side by side and a third panel marking in red
//     where they differ.
//
// and prints a line per story: its id, the share of pixels that differ and
// the box they cover, in device px. A pixel differs when a channel is more
// than -levels outside what the other image has within 1 device px of it,
// so antialiasing alone doesn't count (as the spike's look loop did).
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"neuroplast.io/hotty-surf"
	"neuroplast.io/hotty-surf/internal/stories"
	"neuroplast.io/hotty-surf/ui"
)

func main() {
	hotty := flag.String("hotty", "", "hotty-blitz's hotty binary")
	fonts := flag.String("fonts", "", "hotty-blitz's fonts directory (crates/hotty-blitz/fonts)")
	chromium := flag.String("chromium", "chromium", "the Chromium binary")
	out := flag.String("out", "", "the directory to write to")
	theme := flag.String("theme", "dark", "dark or light")
	scale := flag.Float64("scale", 2, "device pixels per CSS pixel")
	cols := flag.Int("cols", 80, "the surface's width in cells")
	levels := flag.Int("levels", 8, "how far a channel may be off before a pixel differs")
	only := flag.String("story", "", "only the story of this id")
	files := flag.Bool("files", false, "the arguments are HTML files to compare as they are, not components")
	rows := flag.String("rows", "auto", "the surface's height in cells, or auto: as many as it takes (a story whose overlay is fixed takes no rows of its own, so it needs a number)")
	flag.Parse()
	if *hotty == "" || *fonts == "" || *out == "" || flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		die(err)
	}
	// A cell 10×21 CSS px, a terminal's at a normal size.
	cw, ch := int(10**scale+0.5), int(21**scale+0.5)
	cell := fmt.Sprintf("%dx%d", cw, ch)
	host, err := exec.Command(*hotty, "css", "--cell", cell, "--scale", fmt.Sprint(*scale)).Output()
	if err != nil {
		die(fmt.Errorf("hotty css: %w", err))
	}
	l := looker{hotty: *hotty, fonts: *fonts, chromium: *chromium, out: *out, host: string(host),
		scale: *scale, cols: *cols, rows: *rows, cell: cell, levels: uint32(*levels)}
	if *files {
		// Documents as they are (a repro, or a story's cut down), each
		// compared as a story is.
		for _, f := range flag.Args() {
			doc, err := os.ReadFile(f)
			if err != nil {
				die(err)
			}
			l.look(strings.TrimSuffix(filepath.Base(f), ".html"), string(doc))
		}
		return
	}
	for _, comp := range flag.Args() {
		for _, st := range stories.Of(comp) {
			if *only != "" && st.ID != *only {
				continue
			}
			doc, err := surf.New("story", st.View(), ui.Stylesheet()).Document()
			if err != nil {
				die(fmt.Errorf("%s: %w", st.ID, err))
			}
			l.look(st.ID, themed(doc, *theme))
		}
	}
}

type looker struct {
	hotty, fonts, chromium, out, host string
	scale                             float64
	cols                              int
	rows, cell                        string
	levels                            uint32
}

// look draws doc in both engines, compares the two and prints the line.
func (l looker) look(id, doc string) {
	base := filepath.Join(l.out, strings.ReplaceAll(id, "/", "-"))
	if err := os.WriteFile(base+".html", []byte(doc), 0o644); err != nil {
		die(err)
	}
	b := base + ".blitz.png"
	if o, err := exec.Command(l.hotty, "render", base+".html", "-o", b, "--cols", fmt.Sprint(l.cols),
		"--rows", l.rows, "--cell", l.cell, "--scale", fmt.Sprint(l.scale)).CombinedOutput(); err != nil {
		die(fmt.Errorf("%s: hotty render: %v\n%s", id, err, o))
	}
	bi := read(b)
	page := base + ".chromium.html"
	if err := os.WriteFile(page, []byte(chromiumPage(doc, l.host, l.fonts)), 0o644); err != nil {
		die(err)
	}
	c := base + ".chromium.png"
	w, h := bi.Bounds().Dx(), bi.Bounds().Dy()
	cmd := exec.Command(l.chromium, "--headless", "--disable-gpu", "--hide-scrollbars", "--no-first-run",
		"--allow-file-access-from-files", "--virtual-time-budget=3000", "--font-render-hinting=none", "--force-color-profile=srgb",
		fmt.Sprintf("--force-device-scale-factor=%v", l.scale),
		fmt.Sprintf("--window-size=%d,%d", int(float64(w)/l.scale), int(float64(h)/l.scale)),
		"--screenshot="+c, "file://"+page)
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+filepath.Join(l.out, ".chromium"))
	if o, err := cmd.CombinedOutput(); err != nil {
		die(fmt.Errorf("%s: chromium: %v\n%s", id, err, o))
	}
	ci := read(c)
	share, box, mask := compare(bi, ci, l.levels)
	write(base+".sheet.png", sheet(bi, ci, mask))
	fmt.Printf("%s\t%.2f%%\t%v\n", id, share*100, box)
}

// themed puts a story in the kit's theme, on its background, as an app's
// root does (examples/tasks: kit-surface dark), without kit-surface's
// 100vh, which a render as many rows as it takes can't have.
func themed(doc, theme string) string {
	i := strings.Index(doc, `<div id="root">`)
	if i < 0 {
		return doc
	}
	j := i + len(`<div id="root">`)
	k := strings.LastIndex(doc, "</div>")
	return doc[:j] + `<div class="` + theme + `" style="background: var(--background); color: var(--foreground); font-family: var(--font-sans);">` +
		doc[j:k] + `</div>` + doc[k:]
}

// chromiumPage is doc as a host draws it: the host's stylesheet in a layer
// under the document's, and hotty-blitz's fonts under the names it gives
// them.
func chromiumPage(doc, host, fonts string) string {
	var b strings.Builder
	b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\"><style>\n")
	for _, f := range []struct {
		family, file string
		weight       int
	}{
		{"IBM Plex Sans", "IBMPlexSans-Regular", 400}, {"IBM Plex Sans", "IBMPlexSans-Medium", 500},
		{"IBM Plex Sans", "IBMPlexSans-SemiBold", 600},
		{"AdwaitaMono Nerd Font Propo", "AdwaitaMonoNerdFontPropo-Regular", 400},
		{"AdwaitaMono Nerd Font Propo", "AdwaitaMonoNerdFontPropo-Bold", 700},
	} {
		fmt.Fprintf(&b, "@font-face { font-family: %q; src: url(\"file://%s\") format(\"woff2\"); font-weight: %d; }\n",
			f.family, filepath.Join(fonts, f.file+".woff2"), f.weight)
	}
	b.WriteString("@layer hotty-host {\n" + host + "\n}\nhtml, body { margin: 0; }\n</style></head><body>")
	b.WriteString(doc)
	b.WriteString("</body></html>")
	return b.String()
}

func read(f string) image.Image {
	r, err := os.ReadFile(f)
	if err != nil {
		die(err)
	}
	img, err := png.Decode(bytes.NewReader(r))
	if err != nil {
		die(fmt.Errorf("%s: %w", f, err))
	}
	return img
}

func write(f string, img image.Image) {
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		die(err)
	}
	if err := os.WriteFile(f, b.Bytes(), 0o644); err != nil {
		die(err)
	}
}

// compare is the share of a's pixels that differ from b's, the box they
// cover, and a mask of them.
func compare(a, b image.Image, levels uint32) (float64, image.Rectangle, *image.Alpha) {
	r := a.Bounds().Intersect(b.Bounds())
	if a.Bounds() != b.Bounds() {
		fmt.Fprintf(os.Stderr, "sizes differ: blitz %v, chromium %v\n", a.Bounds().Size(), b.Bounds().Size())
	}
	mask := image.NewAlpha(a.Bounds())
	var box image.Rectangle
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if near(a, b, x, y, r, levels) && near(b, a, x, y, r, levels) {
				continue
			}
			n++
			mask.SetAlpha(x, y, color.Alpha{255})
			box = box.Union(image.Rect(x, y, x+1, y+1))
		}
	}
	return float64(n) / float64(r.Dx()*r.Dy()), box, mask
}

// near says whether p's pixel at x, y is within levels of one of q's
// within 1 px of it.
func near(p, q image.Image, x, y int, r image.Rectangle, levels uint32) bool {
	pr, pg, pb, _ := p.At(x, y).RGBA()
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if !(image.Point{x + dx, y + dy}).In(r) {
				continue
			}
			qr, qg, qb, _ := q.At(x+dx, y+dy).RGBA()
			if off(pr, qr) <= levels && off(pg, qg) <= levels && off(pb, qb) <= levels {
				return true
			}
		}
	}
	return false
}

func off(a, b uint32) uint32 {
	a, b = a>>8, b>>8
	if a > b {
		return a - b
	}
	return b - a
}

// sheet is Blitz's image, Chromium's, and the first marked in red where
// they differ, side by side.
func sheet(a, b image.Image, mask *image.Alpha) image.Image {
	w, h := a.Bounds().Dx(), max(a.Bounds().Dy(), b.Bounds().Dy())
	const gap = 8
	s := image.NewRGBA(image.Rect(0, 0, 3*w+2*gap, h))
	draw.Draw(s, s.Bounds(), image.NewUniform(color.RGBA{255, 0, 255, 255}), image.Point{}, draw.Src)
	draw.Draw(s, a.Bounds(), a, image.Point{}, draw.Src)
	draw.Draw(s, b.Bounds().Add(image.Pt(w+gap, 0)), b, image.Point{}, draw.Src)
	third := a.Bounds().Add(image.Pt(2*(w+gap), 0))
	draw.Draw(s, third, a, image.Point{}, draw.Src)
	draw.Draw(s, third, image.NewUniform(color.RGBA{0, 0, 0, 160}), image.Point{}, draw.Over)
	draw.DrawMask(s, third, image.NewUniform(color.RGBA{255, 40, 40, 255}), image.Point{}, mask, image.Point{}, draw.Over)
	return s
}

func die(err error) {
	fmt.Fprintln(os.Stderr, "looks:", err)
	os.Exit(1)
}
