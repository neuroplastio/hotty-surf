# 2026-11-08: gallery controls that apply

The widget gallery showed controls which could not affect the example:
checked on text, gap on a leaf, a switch's tone, and pill fills on an
ordinary button. It now hides those, plus top, invalid, bare, selected,
wrap and scroll where they cannot apply to the example. The raised,
current and selected controls appear when a button becomes a pill and
are hidden again when it stops being one. The states matrix retains a
pill's settings as it makes variants. The underlying vocabulary is
unchanged. The remaining controls still need an audit.

Tests cover the filtering, live updates in both runtimes and the states
matrix. The host gallery was viewed at both widths in Graphite and Stone,
and the pill switch exercised in each theme. The widget stories have no
separate cells view.
