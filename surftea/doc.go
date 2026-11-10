// Package surftea shows a hotty-surf model as a surface of a Bubble Tea
// program, placed with hottytea's Session at a rectangle the program picks.
// It is for a program that owns its screen and puts surfaces among its own
// cells (hotty-surf's guide, "Hosting in a Bubble Tea program"), as a
// multiplexer puts its panes; a program that is only a surface runs under
// surfrun.Run.
//
// A Host is one surface: a surfrun.Surface (the model's tree, the element
// with the keyboard, the fields, a Select's list and an element's menu)
// with the Session's calls around it. A program makes one Host per surface,
// each with a name of its own, and they share nothing but the Session's
// stylesheet, which the first of them sends and the rest find sent. Its
// life in the program's Update, after the Session's own (hottytea):
//
//   - hottytea.ReadyMsg: Ready, for each Host (how a binding's mod reads).
//   - hottytea.EventMsg: Event, until a Host takes it (the surface it names
//     is its own, or its list's or menu's).
//   - a key: Key, on the Host that has the keyboard of the program's own;
//     what it does not take is the program's. Lend gives the keyboard to
//     the program for keys of its own and takes it back.
//   - PostMsg: Posted, for a function the model posted from a goroutine of
//     its own (surfrun.Runner), back on the program's.
//   - hottytea.RelayoutMsg: Relayout.
//   - drawing the frame: Surfaces (or Surface) for each Host in the Layout's
//     list, then Layout, then AfterLayout for each Host, and Flush.
//
// Where there is no host (hottytea's Mode is Text) the same Host draws in
// cells instead, at the same rectangle: Lines, Caret, CellsKey, CellsPaste
// and CellsClick.
//
// A Select's open list and an element's menu are surfaces of their own,
// placed where the surface's frame says (surfrun.List, surfrun.MenuFrame)
// above the Host's surface, and taken off when they close: Surfaces lists
// them after the Host's own.
//
// Nothing here is on a wire of its own: a Host writes hotty commands
// through the Session, and what hotty carries is hotty's (SPEC §3, §6).
package surftea
