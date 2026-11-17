// Package lint is the kit's rules for an app's sources: what keeps an app
// right in a terminal, whose font, width and colours are not a browser's.
// `hotty-surf lint`, `hotty-surf run` after every save, surftest.Lint and a
// dev parent such as plx's run it.
//
// The rules: colour (a colour value outside the theme's files and tokens),
// px (a px size, where rem follows the terminal's font), root-font (a font
// size on html or :root), hidden-below (the page hidden below a
// breakpoint, where a pane is often narrower), templ-import (templ
// imported in a .templ file), class (a class no stylesheet defines: the
// kit's has the classes the kit uses, not all of Tailwind's, so an app's
// other classes go in its own CSS), button-type (a <button> in a <form>
// with no type, which submits it). colour and px read the app's .css files
// too, but for a theme file (theme*.css) and, for colour, a custom
// property (--name: value). Dir checks class only for an app with no
// Tailwind build of its own (style/build.sh).
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
// not the kit's, and not under node_modules, testdata, dist (a build's
// output) or a directory whose name starts with . or _. err is for files
// it couldn't read.
func Dir(dir string) ([]Finding, error) {
	return walk(dir)
}
