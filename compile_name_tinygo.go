//go:build tinygo

package surf

import "context"

// funcName is "" under TinyGo, which a release build is: the order of
// Compile's calls names each (compiledName), and no reload needs more.
func funcName(any) string { return "" }

// hasChildren is false under TinyGo: With's fault for a children block is
// a test's and a dev build's, which run on Go.
func hasChildren(context.Context) bool { return false }
