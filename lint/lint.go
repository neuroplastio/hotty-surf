// Package lint is the kit's rules for an app's sources: what keeps an app
// right in a terminal, whose font, width and colours are not a browser's.
// `surf lint`, `surf run` after every save, surftest.Lint and a dev parent
// such as plx's run it.
//
// The rules: colour (a colour value outside the theme's files), px (a px
// size, where rem follows the terminal's font), root-font (a font size on
// html or :root), hidden-below (the page hidden below a breakpoint, where a
// pane is often narrower), templ-import (templ imported in a .templ file).
// An exception is written where it is made, with its reason, on the line
// before: // lint:allow px: <why>. An allow without a reason is a finding.
package lint

// A Finding is one place an app breaks a rule.
type Finding struct {
	File      string // relative to the directory checked
	Line, Col int
	Rule      string // the rule's name, as above
	Text      string // what is wrong, and what to write instead
}

// Dir checks the app's sources under dir: its .templ, .go and .css files,
// not the kit's. err is for files it couldn't read.
func Dir(dir string) ([]Finding, error) {
	panic("L0")
}
