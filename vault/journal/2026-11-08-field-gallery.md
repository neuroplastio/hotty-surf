# 2026-11-08: field in the gallery

The field's generated attributes story started with an empty value. Its
text tone and weight were therefore invisible until someone typed into it.
It starts with editable sample text now; the host's accent and bold controls
visibly change it. This changes the gallery's example, not field styling.
The widget story has no separate cells view; cells drawing is unchanged.

The gallery test checks the example's initial value and that its value
control updates the field in both runtime modes. The host gallery was looked
at in Graphite and Stone at full and half widths, and its bold and accent
controls exercised in each theme.
