# Tab completes the palette (hotty 0.8, V6-02)

The palette already bound Tab to Complete, beside Ctrl+Space. The field
now gives that key to the program: a binding of Tab on a field or around
it writes `tab=program` into the field's `data-keys`. Buttons and lists
keep Tab's focus order. Shift+Tab still moves focus back, and Escape
still reaches the program.

The cells renderer follows the same rule and reports Tab as the
program's key from that field in `Renderer.Keys`. A field that does not
bind it still moves focus. The change is shared by interpreted and
compiled templates; no generated file changes.

The audit found one component that completes typed text:
`internal/screens/palette.surf`. The icon and emoji grid and the Select
picker pick options; they do not complete a text field.

The palette completion test now presses Tab on a hottytest 0.8 host in
both modes. The tree-cells palette test completes `zoom` to `pane.zoom`
before running it. `surfui` checks the emitted keymaps, and `pkg/cells`
checks completion, `Renderer.Keys`, Shift+Tab, Escape and Tab on the
button beside the field.

The race tests in `surfui`, `pkg/cells`, `internal/screens` and
`surfrun` pass under CAP with hotty-go 0.8.0.
