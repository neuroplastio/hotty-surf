# 2026-11-08: design iteration plan

The kit's Graphite and Stone roles, static chrome components and row motion
are in place. The gallery and host surfaces need a fresh visual audit against
the design at full and half window widths, in both themes and in cells.
The latest chrome contract takes precedence where the earlier board differs.

1. Audit the gallery's command icons and their tooltips, pane and card
   clipping, light theme, and resizing. Fix the first reproducible issue;
   these are the maintainer's most visible reported problems.
2. Compare the most visible chrome surfaces and strips with the current
   mockup, then take the remaining migration items in board order. Check
   current main before reusing any earlier implementation.
3. Review kit stories against the component boards, concentrating on
   shared components used by those surfaces. Record renderer differences
   in the Blitz gap list and keep each visual change independently reviewable.
4. Revisit syntax colours when the code widget is available. The remaining
   widgets are being developed separately.

## Open questions

- Are all gallery complaints fixed in the current build? Lean: treat each
  as unverified until exercised in both themes and at both widths.
- Does an earlier migration implementation still apply? Lean: compare it
  with current main and port only missing behaviour, since subsequent work
  has changed these surfaces.
- Does a mismatch need a new role or vocabulary? Lean: use established
  roles and composition where they settle it; ask the maintainer before
  adding either.
