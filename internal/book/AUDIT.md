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
- [ ] Complete the comparison across every story and populated Events.
  `audit-alert-source.png`, `alert-props-side-by-side.png`,
  `audit-alert-events-empty.png`, `audit-alert-html.png`, and
  `audit-button-filter.png` compare the same stories in both windows.
  Alert empty Events lacked the spike's italic style; it is restored.
  The Button preview had an extra count; it is removed, while Clicks remains
  editable and shared across renditions. State changes were not logged in
  Events; the spike's prop observation is restored. The model's filter also
  matched opaque IDs in addition to group and title; it now matches only
  group and title, like the spike.
- [x] `audit-icons.png` showed the gallery missing tile borders and icon
  sizing. The tile borders and size are restored. The kit has a different
  set of Lucide names, so each gallery shows the icons its kit ships.
- [ ] Finish porting all spike story props and live handlers. Alert, Button,
  Badge, Field, Area, Check, Card, Table, Tabs and Kbd have their spike
  fields and control kinds. Code, NavList, Controls, EventLog and Tabs kit
  demonstrations and variants have been added. Check's mixed visual state
  is an existing kit limitation and is recorded in coverage.
- [ ] The spike's shell demonstrations (List, icon buttons, thumbnails,
  busy badge and symbols) depend on components absent from v0.7's public
  kit; determine their port path without adding unreviewed public API.
- [ ] The dropdown and popover stories wait for B3's public API. If that API
  does not expose Open/Active, omit those fields and record that decision
  rather than accessing private state.
- [ ] Run the full gate after the audit fixes.
