package surf

import "github.com/neuroplastio/hotty-surf/surfui"

// Palette is a set of named colours, hues, declared at run time: a colour
// from a user's configuration, which a template's <palette> can't know.
// surfrun.Runtime.SetPalette (and Options.Palette) declare it, and
// declaring again replaces the set and sends the stylesheet again.
//
//	rt.SetPalette(surf.Palette{
//		"coral": {Dark: "#ff8e7a", Light: "#a42d1c", Cells: surf.Cell(9)},
//	})
//
// A hue is a Dark and a Light value, or one Value for both grounds, in hex
// (#rgb or #rrggbb), and the terminal colour Cells names, 0 to 15, the
// nearest chromatic one by hue angle where nil. Text in it needs 4.5:1 on
// the dark ground and on Stone's, and a dot 3:1: a hue that falls short
// warns once, its text is drawn in the ink on the failing ground, and its
// dots keep it (docs/surf.md, "Colours from data").
type Palette = surfui.Palette

// Hue is one of a Palette's.
type Hue = surfui.Hue

// Cell is a pointer to n, for a Hue's Cells.
func Cell(n int) *int { return surfui.Cell(n) }
