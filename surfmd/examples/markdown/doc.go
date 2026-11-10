// Package markdown is a screen that draws Markdown from a template with no
// Go component, <surf-markdown>, in the interpreted and the compiled mode
// alike. Its generator is its own command, which links surfmd
// (cmd/surfgen).
package markdown

//go:generate go run ./cmd/surfgen

// Doc is the screen: a page in GitHub's Markdown and a comment whose
// newlines are breaks. Its template is doc.surf.
type Doc struct {
	Body    string
	Comment string
}
