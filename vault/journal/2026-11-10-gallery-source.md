# 2026-11-10: the story's .surf source in the gallery

The maintainer wants to read a screen's `.surf` layout beside what it
draws: a source view, no pointing. Through hotty-pm. Branch
`gallery-human`, not landed.

**What it is**

- A Source switch on the toolbar, drawn only where the story has
  `.surf` source. A story of `stories.All` is a Go tree: no switch.
- On, the source is in the column at the right, under the controls
  where the story has some, a scroll box of its own: it scrolls
  without the canvas.
- It shows what a person wrote:
  - a stories file's story: its own lines of the file, their indent
    taken off, numbered as the file numbers them;
  - under them, for a screen's or a partial's story, the template it
    draws (`settings.surf`, `notice-row.surf`), whole;
  - a widget's own story: the one-line template its controls make,
    which changes as they do.
- It is the kit's `<code language="surf" numbers wrap>`, so tags,
  attributes and strings have their roles.
- It follows the file as the canvas does (`-dev`).

**Beside, not under**

A scene is a column of sections, so its height is what is scarce; a
source under it would show a dozen lines of a hundred. Beside it, a
window's height of lines is read against what they draw. The cost is
the canvas's width, so a story's canvas gives the source room: half
the room beside the stories, between 44 cells and 86 (a line of 80
and its numbers). A width the user picked is kept, and the frame says
what it left out, as for the controls.

**Where it is**

`internal/gallery`: `file.go` (`storySource`, a story's lines),
`gallery.go` (`Source`, `HasSource`), `frame.go` (`reserve`, `refit`,
the column's width by what it holds, `FlipSource`), `gallery.surf`.
`surfgallery -source` opens with it on. Tests: `TestSource`,
`TestStorySource`.

**To see it**

`make gallery`, pick a story, Source. Shots: `.shots/slice4/`.
