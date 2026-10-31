// Package demodata is the made-up data the demo screens draw: a terminal
// workspace's settings page, its command palette's commands, its panes,
// and the coding agents at work in them, with the acts that change them.
// Nothing in it is any real program's: the names, the values and the
// rules are invented, small enough to read whole, and shaped as such a
// workspace's would be.
//
// A Source is the one place a screen reads from and acts on. It lives in
// memory: an act changes only the Source, and each is recorded in order
// (Source.Acts), so that a test can check what a screen did as well as
// what it shows.
package demodata
