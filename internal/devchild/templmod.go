//go:build surfdev && surftemplmod

package devchild

import templruntime "github.com/a-h/templ/runtime"

// The reloader's dev builds carry mods/templ (internal/templmod) and its
// tag: templ then reads the strings again when asked, and render needn't
// wait out its recheck.
func init() { reread = templruntime.ReloadWatchedStrings }
