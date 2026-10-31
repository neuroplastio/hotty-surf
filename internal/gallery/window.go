package gallery

// Window is the width in cells of the narrowest window the gallery draws
// a canvas cols wide in with both side columns whole: the canvas (or the
// middle column's least), the stories and the controls at their widths,
// a gap beside each, and the main row's pads (frameOf). internal/cmd/stories
// -gallery opens each story in one at least so wide, for the legible
// gate (scripts/gate/legible.cjs).
func Window(cols int) int { return max(cols, midMin) + sideCols + asideCols + 2 + framePad }
