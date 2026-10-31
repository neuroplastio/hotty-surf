package screens

import "github.com/neuroplastio/hotty-surf/surfkeys"

// The key caps (S6, H5): a key sequence in myelin's names ("ctrl+b p
// shift+x"), drawn one cap a chord in the settings' style by package
// keys. key-caps.surf holds the sequence, a <key>, which the surface
// draws in the palette's style (Palette.KeyStyle); the palette's cells,
// which write their own lines, have the caps as text (capsText). In the
// demo's style, ctrl+b is ^b, shift+x ⇧x, mod+enter ^enter on Linux,
// alt+enter alt+enter and escape esc.

// capsText is seq's caps in st as a cells line has them (surfkeys.Text).
func capsText(st surfkeys.Style, seq string) string { return surfkeys.Text(st, false, seq) }
