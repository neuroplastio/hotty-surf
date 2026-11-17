## G-cells-copy: a field's selection, for plx's copy and cut

In cells, plx copies a field's selection to the terminal's clipboard
(v0.6: `cr.Copy` callback → `Host.SetCopy`). v0.7's Screen has no
clipboard and leaves the copy, cut and paste keys to the program, but
exposes no field selection, so plx's copy and cut do nothing in cells
(`l4c-gaps.md`, G-cells-copy; it blocks slice 2, where copying in cells
is a tool feature).

### The shape: the program asks, on its own key

- `Selection() string`: the focused text field's selected text, "" for
  none (no field, nothing selected, a password).
- `Cut() string`: the same, deleted from the field as the user's edit
  (rev, input where it listens, change on commit). "" and no edit in a
  readonly or disabled field, or a password.
- The keys stay as they are: the Screen doesn't take copy or cut
  (`fieldKey`), so the program hears them and calls these. Paste is
  already `Paste(text)`. One direction each way: the program owns the
  clipboard.
- Implemented and tested in the branch (grapheme-correct, readonly,
  password).

### Open questions, with leans

1. **Methods, or a copy callback on New** (v0.6's `Copy func(string)`).
   **Lean: methods.** New's signature is approved, and a callback needs
   a second option. With methods, the program, which already decides what
   its copy key means (a pane's own selection vs the field's), asks only
   when it's the field's.
2. **Cut as its own method, or `Selection()` plus a `Delete()`.**
   **Lean: Cut.** It's one act. A separate Delete is a second name and a
   window where the two disagree.
3. **A password's selection.** **Lean: "".** Browsers refuse to copy from
   `type=password`, and a host does the same.
