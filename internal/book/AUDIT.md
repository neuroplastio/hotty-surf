# Storybook spike port audit

Comparison: the spike and v0.7 book on the same headless hottyterm display,
Alert selected in both. The screenshot names below are in the audit capture
directory. This is a working list; unchecked items remain to be verified or
fixed before the audit is complete.

## Requested differences

- [x] Rendition tabs: v0.7 adds hotty/cells and `r` as requested.
- [x] List: v0.7 uses a real scrolling surface and reveal, as requested.
- [x] Group names: v0.7 groups by component use instead of the spike's
  package names, as requested. The v0.7 inventory also includes its own kit
  stories.
- [x] HTML used to show the whole surface document and stylesheet. It now
  renders only the selected component with the spike formatter and note.
  `audit-alert-html.png` compares both on Alert. v0.7's kit emits its own
  component attributes, which differ from the old kit's markup.
- [x] The inspector tabs used to scroll away with the panel. The selected
  panel is now the scrolling box, with a fixed tab list; a cells regression
  checks that content moves without moving the tab row. On hottyterm, wheel
  scrolling moved source content while the tab row stayed fixed; see
  `inspector-before.png` and `inspector-after.png`.
- [x] Alert Props used a private, reduced type. It now shows `stories.Alert`
  with Variant, Title, Text and Icon. The Alert component and editable
  defaults match the spike. `alert-props-side-by-side.png` shows the two
  with the duplicate select arrow noted below.
- [x] The inspector labels and fields lacked the spike's compact styling and
  gap before the code. Restored monospace small labels, field sizes, spacing,
  and the spike's checkbox appearance.

## In progress

- [ ] The host draws a native select arrow despite Basecoat `appearance:none`,
  in addition to Basecoat's background chevron. The CSS retains the spike
  markup and Basecoat chevron; the host issue is under investigation.
  `alert-props-arrow-fixed.png` shows the double arrow on hottyterm 0.8.7.
- [ ] Compare Source, Props, Events, HTML, header, list, navigation keys and
  empty states on the same stories in both windows; record each difference.
  Alert empty Events had non-italic text; the italic style is restored.
- [ ] Finish porting all spike story props and live handlers, including the
  spike's story variants and kit-component demonstrations.
- [ ] The dropdown and popover stories wait for B3's public API. If that API
  does not expose Open/Active, omit those fields and record that decision
  rather than accessing private state.
- [ ] Run the full gate after the audit fixes.
