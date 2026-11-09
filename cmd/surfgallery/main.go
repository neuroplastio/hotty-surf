// Command surfgallery is hotty-surf's gallery of stories (package gallery),
// in the terminal it runs in, which must speak hotty:
//
//	go run ./cmd/surfgallery              the gallery, by its templates interpreted
//	go run ./cmd/surfgallery -compiled    by its templates compiled into Go
//	go run ./cmd/surfgallery -story key-caps  the first story whose name has key-caps in it
//	go run ./cmd/surfgallery -story s-button-states   the story of that id
//	go run ./cmd/surfgallery -list        every story's id, in the sidebar's order
//	go run ./cmd/surfgallery -theme paper -density roomy   the canvas so
//	go run ./cmd/surfgallery -states      the story in each of its states
//	go run ./cmd/surfgallery -dev .       the checkout's stories, templates and stylesheet, reloaded as they change
//	go run ./cmd/surfgallery -dev . -reference -story button/states   with make gate's renders of it
//
// Its document scrolls, so the wheel scrolls its stories, its canvas and
// its controls, each a scroll box (hotty SPEC §5.3). scripts/gallery.sh
// runs it in a window of its own, or in a browser.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"slices"
	"strconv"
	"time"

	"neuroplast.io/hotty-surf/internal/gallery"
	"neuroplast.io/hotty-surf/surflive"
	"neuroplast.io/hotty-surf/surfrun"
)

func main() {
	compiled := flag.Bool("compiled", false, "draw the gallery by its templates compiled into Go")
	story := flag.String("story", "", "open the story whose id is `text`, or else the first whose component or name has it in it")
	list := flag.Bool("list", false, "print every story's id, one a line, in the sidebar's order, and exit")
	theme := flag.String("theme", "graphite", "the canvas's theme")
	density := flag.String("density", "compact", "the canvas's density")
	width := flag.Int("width", 0, "the canvas's width in `cells` (default the story's)")
	states := flag.Bool("states", false, "draw the story in each of its states")
	cells := flag.Bool("cells", false, "draw a screen by its cells UI")
	reference := flag.Bool("reference", false, "show the gate's renders of a story of stories.All under it (with -dev)")
	bare := flag.Bool("bare", false, "hide the stories' and the controls' columns")
	dev := flag.String("dev", "", "draw the checkout at `dir`'s stories files, templates and stylesheet, and reload them as they change")
	flag.Parse()
	g := gallery.New()
	if *dev != "" {
		if err := g.Watch(*dev, 250*time.Millisecond); err != nil {
			fmt.Fprintln(os.Stderr, "gallery:", err)
			os.Exit(1)
		}
	}
	if *list {
		for _, id := range storyIDs(g) {
			fmt.Println(id)
		}
		return
	}
	if *story != "" {
		pick(g, *story)
	}
	g.SetTheme(*theme)
	g.SetDensity(*density)
	if *width > 0 {
		g.SetWidth(strconv.Itoa(*width))
	}
	if *states {
		g.FlipStates()
	}
	if *cells {
		g.FlipCells()
	}
	if *reference {
		g.FlipReference()
	}
	if *bare {
		g.Bare()
	}
	c := gallery.Interpreted(g)
	if *compiled {
		c = gallery.Compiled(g)
	}
	if err := surfrun.Run(context.Background(), surflive.New(c), gallery.Options(g)); err != nil {
		fmt.Fprintln(os.Stderr, "gallery:", err)
		os.Exit(1)
	}
}

// pick opens the story whose id is text; or else the list is narrowed to
// the stories whose component or name has text in it, and the first of
// them opens.
func pick(g *gallery.Gallery, text string) {
	if slices.Contains(storyIDs(g), text) {
		g.Pick(text)
		return
	}
	g.SetFilter(text)
	if ids := storyIDs(g); len(ids) > 0 {
		g.Pick(ids[0])
	}
}

// storyIDs are the ids of the stories the list shows, in its order.
func storyIDs(g *gallery.Gallery) []string {
	var ids []string
	for _, e := range g.Shown() {
		if e.Kind == "story" {
			ids = append(ids, e.ID)
		}
	}
	return ids
}
