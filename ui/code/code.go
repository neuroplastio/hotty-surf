// Package code shows source code highlighted, in the kit's look: Chroma's
// lexers, coloured by the theme's tokens. It is a package of its own so that
// an app that shows no code doesn't link Chroma.
package code

import "github.com/a-h/templ"

// Block is src highlighted as lang (a Chroma lexer's name or alias: "go",
// "json", "sh"), in Basecoat's pre. first is the first line's number; 0
// shows no numbers. A lang Chroma doesn't know shows src plain.
func Block(src, lang string, first int) templ.Component {
	panic("L0")
}
