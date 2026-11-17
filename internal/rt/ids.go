package rt

import "strings"

// IDPrefix starts every id the runtime gives an element, so it can't be
// one an app writes: an app's id with it is a fault, as are root and
// hotty-css, the document's own.
const IDPrefix = "_"

// RuntimeID says whether id is one the runtime owns: generated, or the
// document's own. surf faults an app's id that is one, and cells' At
// skips them for the app's.
func RuntimeID(id string) bool {
	if id == "root" || id == "hotty-css" {
		return true
	}
	if !strings.HasPrefix(id, IDPrefix) || len(id) == len(IDPrefix) {
		return false
	}
	for _, c := range id[len(IDPrefix):] {
		if !('0' <= c && c <= '9' || 'a' <= c && c <= 'z') {
			return false
		}
	}
	return true
}
