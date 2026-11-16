//go:build surfparsecheck

package surf

import (
	"fmt"

	"golang.org/x/net/html"
)

// parseCheck parses every render the fast parser took by x/net/html too,
// and panics where the trees differ.
const parseCheck = true

func checkParse(fast *html.Node, src []byte) {
	slow := newRoot()
	if err := parseSlow(slow, src); err != nil {
		panic(err)
	}
	if path := diffNodes(fast, slow, "#root"); path != "" {
		panic(fmt.Sprintf("surf: the fast parser differs from x/net/html at %s, in:\n%s", path, src))
	}
}
