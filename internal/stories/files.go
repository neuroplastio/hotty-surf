package stories

import "embed"

// Files are the stories files, internal/stories/*.surf: a component's stories as
// its tag with its props, which the gallery draws live (cmd/surfgallery). A
// file is
//
//	<stories of="foot-hint" cols="40">
//	  <story name="one key" title="Run" keys="enter"/>
//	</stories>
//
// of names the component's template, cols the canvas's width in cells
// (a story may give its own), and in a widget each story is drawn in
// (in="list" for a list's rows). Each <story> gives its props as
// attributes, literal or bound (:row="rows[0]", in its file's Go state),
// and its children are what the component's slots take; its about, if
// any, is a muted line under it, which says what a story that draws
// nothing by design is (about="No keys: no caps."). A story of a
// component (a screen) gives none of these: its Go state is its own.
//
// A file of a widget of the vocabulary (of="button") is a scene file: a
// component as a person judges it. Each story's children are the scene
// itself, drawn at the window's width with no Go state; its name is its
// title, its about the sentence under it, and the file's title the
// component's name in the list (title="Button").
//
// The gallery's states matrix draws a story in each of its states, which
// it makes from the story's controls (default, hover, focus, disabled,
// error, empty, long text). A <state> names a row of the file's own, by
// the values it gives the controls, in place of the state of its name
// or after them:
//
//	<state name="error" notice.level="error"/>
//
//go:embed *.surf
var Files embed.FS
