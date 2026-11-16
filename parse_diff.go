package surf

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

// diffNodes is where trees a and b first differ, or "": the fast parser's
// check (surfparsecheck) and its tests compare it with x/net/html's by it.
func diffNodes(a, b *html.Node, path string) string {
	if a.Type != b.Type || a.Data != b.Data || a.DataAtom != b.DataAtom || a.Namespace != b.Namespace {
		return fmt.Sprintf("%s: %d %q %q / %d %q %q", path, a.Type, a.Data, a.Namespace, b.Type, b.Data, b.Namespace)
	}
	if len(a.Attr) != len(b.Attr) {
		return fmt.Sprintf("%s: attributes %v / %v", path, a.Attr, b.Attr)
	}
	for i := range a.Attr {
		if a.Attr[i] != b.Attr[i] {
			return fmt.Sprintf("%s: attribute %v / %v", path, a.Attr[i], b.Attr[i])
		}
	}
	ac, bc := a.FirstChild, b.FirstChild
	for i := 0; ac != nil || bc != nil; i++ {
		if ac == nil || bc == nil {
			return fmt.Sprintf("%s: child %d is missing on one side", path, i)
		}
		if d := diffNodes(ac, bc, fmt.Sprintf("%s/%s[%d]", path, strings.TrimSpace(ac.Data), i)); d != "" {
			return d
		}
		ac, bc = ac.NextSibling, bc.NextSibling
	}
	return ""
}
