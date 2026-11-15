# A list gives Space back (hotty 0.8, V6-05)

The kit had both workarounds V6-05 describes: a button under a Space
binding claimed a scroll action to get its native press back, and a
widget with a field anywhere inside it gave no Space to the program.

A button now writes `space=default` and `shift+space=default` where an
ancestor binds those keys. A text field gives them back the same way,
so Space types. The outer widget keeps its binding with a field inside
it, and the walk that suppressed Space for the whole subtree is gone.
The open Select keeps its own Space and Enter bindings for picking.

The host-backed scroll-box test now has a field inside its list: Space
marks with the list focused, presses its row's button, and types in
its row's field. A cells test exercises the same three acts. The
widget tests check the default bindings and resolved insertion action.
`go test -race ./surfui ./surfrun ./pkg/cells` passes under CAP.

No stylesheet or rendering change. hotty-go remains protocol-only.
