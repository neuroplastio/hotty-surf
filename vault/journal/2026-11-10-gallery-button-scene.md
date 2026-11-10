# 2026-11-10: the gallery, one slice: the button as a scene

The maintainer can't judge the look from the gallery: its stories are
the pixel gate's fixtures. The scope, by the maintainer through hotty-pm
the same day, is one slice and then a stop: the loop as far as the slice
needs, and one component, the button, shown as a person can judge it.
Branch `gallery-human`, not landed.

**What changed**

- `make gallery` builds `cmd/surfgallery` from the working tree, under
  CAP, and opens it in a hottyterm window (`scripts/gallery.sh`, which
  built the HEAD commit by `git archive` before). It runs with `-dev`,
  as before: an edit of a stories file, a template or `surfui/surf.css`
  is in the window within the poll (250 ms), the story kept.
- A stories file of a widget is a scene file (`internal/stories/files.go`
  says how): each story's children are the scene, drawn with no Go
  state; the file's `title` names the component, the story's `name` is
  its title and its `about` the sentence under it.
- `internal/stories/button.surf` is the one scene: a release's page,
  with the main actions, the same disabled, the tones, the icons, small
  buttons in a list's rows, pills on a bar, and a segmented row and
  chips.
- The frame's smallest changes, for the one story to be shown properly:
  the scenes are a section of their own, Components, first in the
  sidebar, so the gallery opens on the button; a scene's canvas is as
  wide as the window leaves beside the stories (it names no width); and
  a line under the title has the component's tag and the sentence.
  Nothing else of the sidebar, the toolbar or the chrome changed.

**Left as it was:** every other story, the Go fixtures, the gates, the
default theme (graphite), the chrome's own look.

**Not done, and what it would take**

- **Surfaces beside cells.** Not cheap enough for the slice. The canvas
  draws cells only for a screen that has a cells UI of its own
  (`surfrun.Cells`), from its SGR lines. A scene's cells would come
  from `pkg/cells`: a renderer on the scene's tree, drawn at the
  canvas's width, and its Screen's runs (role colours, grounds and
  attributes) drawn as a surface's lines beside the story, with the
  keyboard and the pointer passed on to it for it to be live. The
  codex-vt worktree has uncommitted work on the same seam (24-bit SGR in
  the canvas's lines). Meanwhile the whole gallery in a terminal that
  doesn't speak hotty is the cells.
- **The Go loop** (a Go edit rebuilds and re-executes in the same
  window) and the reload of the gallery's own page were begun before
  the scope changed. They are not in this branch's commit.
- What a stories file can't show is in
  [`knowledge/surf-gaps.md`](../knowledge/surf-gaps.md): a forced hover
  or focus, a scene's state, a scene's states matrix.

**Seen in the scene, not touched** (the look is the maintainer's):

- A disabled button is hard to tell from an enabled one: its label is a
  little fainter and nothing else.
- A bare button has no edge at rest and gains one disabled, so the
  disabled Cancel looks more like a button than the live one.
- An icon-only pill (`flask`, `bell`, `cog`) is much smaller than the
  pills with a label beside it.

**How to see it:** `make gallery`. Edit `internal/stories/button.surf`
or `surfui/surf.css` and save. Renders: `.shots/button-story.png` and
`.shots/button-reloaded.png` (hottyterm on the private headless display,
1.6).
