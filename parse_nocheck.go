//go:build !surfparsecheck

package surf

import "golang.org/x/net/html"

const parseCheck = false

func checkParse(*html.Node, []byte) {}
