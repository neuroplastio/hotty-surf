package surfui

import (
	"slices"
	"strconv"
)

// A tree is a List whose Items say how deep they are (Depth), whether
// they are a branch (Branch) and whether it is open (Open): the program
// flattens its tree, draws the rows that show and none of a closed
// branch's children, and owns what is open and what is selected. The
// runtime folds nothing. An Item that says any of the three has the
// disclosure in front of its label: a chevron where it is a branch, and
// as much room where it isn't, so that the labels of a level line up.

// chevronRight is Material's chevron right, as chevronDown is its down.
const chevronRight = "\U000F0142"

// treeSpec is what a tree Item says of itself.
type treeSpec struct {
	depth        int
	branch, open bool
}

// treeMax bounds a depth: two cells a level must not overflow a wasm int.
const treeMax = 1 << 20

// TreeMove is the action of a binding a tree List makes for a key (its
// arrows, Home, End and Enter): the action of the Item's own handler the
// key stands for, which the runtime runs as it would the event. Kind is
// "click" (the key selects the Item: its Target), "toggle" or
// "activate", with the Item's key as Value.
type TreeMove struct {
	Act    Action
	Kind   string
	Target string
	Value  string
}

// OnToggle runs act when the user asks for a branch to open or to
// close: a click on its chevron, or ← or → on it in a tree. The
// program flips what it keeps; nothing else changes. The event's value
// is the Item's key.
func (n *Node) OnToggle(act Action) *Node { return n.On("toggle", act) }

// Depth is how deep an Item is in a tree, from 0: its label is
// indented by two cells a level, as padding in front, so that its fill
// and its selection's bar keep the left edge of the row.
func (n *Node) Depth(depth int) *Node {
	t := n.treeItem()
	t.depth = min(max(depth, 0), treeMax)
	n.Var("u-depth", strconv.Itoa(t.depth))
	n.Attr("aria-level", strconv.Itoa(t.depth+1))
	return n
}

// Branch says an Item has children, drawn or not: a folder with nothing
// in it is a folder still.
func (n *Node) Branch(branch bool) *Node {
	n.treeItem().branch = branch
	return n.treeMark()
}

// Open says a branch's children are shown. The program draws them: a
// closed branch's are none, and the cells and the host draw what they
// are given.
func (n *Node) Open(open bool) *Node {
	n.treeItem().open = open
	return n.treeMark()
}

// TreeKeys says whether a List answers the keys of a tree itself (the
// default, "tree") or leaves every key to the program ("none"), which
// binds them as it likes.
func (n *Node) TreeKeys(own bool) *Node {
	n.treeOff = !own
	return n
}

// treeItem is n's tree spec, made when it is first asked for, with the
// disclosure in front of its label.
func (n *Node) treeItem() *treeSpec {
	if n.tree == nil {
		n.tree = &treeSpec{}
		d := Icon(" ").Width(2).Keep().Key("disclosure").Class("u-tree-disclosure").Attr("aria-hidden", "true")
		d.disclosure = true
		n.kids = append([]*Node{d}, n.kids...)
		n.Class("u-tree-item").Attr("role", "treeitem")
	}
	return n.tree
}

// treeMark draws the disclosure to say what the spec does.
func (n *Node) treeMark() *Node {
	t, d := n.tree, n.kids[0]
	n.Unattr("aria-expanded")
	mark, cells := " ", " "
	if t.branch {
		n.Attr("aria-expanded", strconv.FormatBool(t.open))
		mark, cells = chevronRight, "▸"
		if t.open {
			mark, cells = chevronDown, "▾"
		}
	}
	SetText(d, mark)
	d.cells = cells + " "
	return n
}

// DisclosureOf is the tree Item whose chevron n is, which a click on
// asks to toggle: nil for any other node, and for a chevron that has no
// handler to toggle with, or no branch to toggle.
func DisclosureOf(n *Node) *Node {
	if n == nil || !n.disclosure || n.parent == nil || n.parent.tree == nil || !n.parent.tree.branch {
		return nil
	}
	if _, ok := Handler(n.parent, "toggle"); !ok {
		return nil
	}
	return n.parent
}

// TreeOpen is whether the branch item is open.
func TreeOpen(item *Node) bool { return item.tree != nil && item.tree.open }

// TreeDepth is how deep item is in its tree, 0 for an Item that says
// nothing of it.
func TreeDepth(item *Node) int {
	if item.tree == nil {
		return 0
	}
	return item.tree.depth
}

// treeKeys are the keys a tree List binds.
var treeKeys = []string{"arrow_up", "arrow_down", "arrow_left", "arrow_right", "home", "end", "enter"}

// resolveTree gives a tree List its keys: they are made again at each
// resolve, from the Items it has then, so that they follow the model's
// selection and what is open.
func resolveTree(n *Node) {
	n.keys = slices.DeleteFunc(n.keys, func(b Binding) bool { _, own := b.Act.(TreeMove); return own })
	if !slices.Contains(n.class, "u-list") {
		return
	}
	var rows []*Node
	at, tree := -1, false
	for _, k := range n.kids {
		if !slices.Contains(k.class, "u-item") {
			continue // a caption between the rows, which the selection skips
		}
		if _, off := AttrOf(k, "disabled"); off {
			continue
		}
		tree = tree || k.tree != nil
		if IsSelected(k) {
			at = len(rows)
		}
		rows = append(rows, k)
	}
	if !tree {
		return
	}
	n.Attr("role", "tree")
	if at >= 0 {
		n.Attr("aria-activedescendant", rows[at].id)
	}
	if n.treeOff {
		return
	}
	bind := func(key, title string, m TreeMove) {
		if m.Act != nil {
			n.keys = append(n.keys, Binding{Key: key, Title: title, Act: m})
		}
	}
	pick := func(key, title string, i int) {
		if i < 0 || i >= len(rows) || i == at {
			return
		}
		act, _ := Handler(rows[i], "click")
		bind(key, title, TreeMove{Act: act, Kind: "click", Target: rows[i].id})
	}
	toggle := func(key, title string) bool {
		act, ok := Handler(rows[at], "toggle")
		if ok {
			bind(key, title, TreeMove{Act: act, Kind: "toggle", Target: rows[at].id, Value: rows[at].key})
		}
		return ok
	}
	if len(rows) == 0 {
		return
	}
	n.Focusable()
	if at < 0 {
		pick("arrow_down", "Down", 0)
		pick("home", "First", 0)
		pick("end", "Last", len(rows)-1)
		return
	}
	pick("arrow_up", "Up", at-1)
	pick("arrow_down", "Down", at+1)
	pick("home", "First", 0)
	pick("end", "Last", len(rows)-1)
	cur := rows[at]
	if act, ok := Handler(cur, "activate"); ok {
		bind("enter", "Open", TreeMove{Act: act, Kind: "activate", Target: cur.id, Value: cur.key})
	}
	t := cur.tree
	if t == nil {
		return
	}
	// ←: a branch that is open closes; any other row goes to its parent,
	// the nearest row above it that is shallower.
	if t.branch && t.open {
		toggle("arrow_left", "Close")
	} else {
		for i := at - 1; i >= 0; i-- {
			if TreeDepth(rows[i]) < t.depth {
				pick("arrow_left", "Parent", i)
				break
			}
		}
	}
	// →: a branch that is closed opens; one that is open goes to its
	// first child, the row after it where it is deeper.
	if t.branch && !t.open {
		toggle("arrow_right", "Open")
	} else if t.branch && at+1 < len(rows) && TreeDepth(rows[at+1]) > t.depth {
		pick("arrow_right", "Child", at+1)
	}
}
