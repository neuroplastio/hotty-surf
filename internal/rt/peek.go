package rt

import "golang.org/x/net/html"

// Peek is what a test rig and the reload child read of a surface that no
// exported method gives: package surf fills it in its init, so that the
// packages that read it need not import surf's insides (and surf need not
// export them).
var Peek struct {
	// Tree is the tree the host was last sent, under its #root; nil before
	// the first Document.
	Tree func(s any) *html.Node
	// Faults are the faults of the renders since the last call, which it
	// forgets.
	Faults func(s any) []string
}
