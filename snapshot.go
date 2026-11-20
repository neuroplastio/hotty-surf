package surf

//go:generate msgp -marshal=false -tests=false -unexported -file snapshot.go -o snapshot_gen.go

//msgp:newtime
//msgp:limit arrays:1048576 maps:1048576

import (
	"errors"
	"fmt"
	"strings"

	"github.com/tinylib/msgp/msgp"
	"golang.org/x/net/html"
)

// snapshot is a surface as the host has it, which a dev reload hands from
// one build to the next (package reload): the new build's first frame is
// a delta from it, so the host's document, its fields and the keyboard stay
// where they were.
type snapshot struct {
	Name    string            `msg:"name"`
	Tree    string            `msg:"tree"` // the last frame's #root, with its ids
	IDs     map[string]string `msg:"ids"`
	Next    int               `msg:"next"`
	Vals    map[string]string `msg:"vals"`
	Checked map[string]bool   `msg:"checked"`
	Revs    map[string]int    `msg:"revs"`
	Focused string            `msg:"focused"`
	Asked   string            `msg:"asked"`
}

// snap encodes the surface as the host has it. It is written once per
// handover, so its writer isn't pooled.
func (s *Surface) snap(w *msgp.Writer) error {
	if s.prev == nil {
		return errors.New("surf: no document to hand over")
	}
	var tree strings.Builder
	if err := html.Render(&tree, s.prev); err != nil {
		return err
	}
	sn := snapshot{Name: s.name, Tree: tree.String(), IDs: s.ids, Next: s.nextID, Vals: s.vals,
		Checked: s.checked, Revs: s.revs, Focused: s.focused, Asked: s.asked}
	return sn.EncodeMsg(w)
}

// adopt takes what the build before had sent the host as this surface's
// own: the next Update sends a delta from it.
func (s *Surface) adopt(r *msgp.Reader) error {
	var sn snapshot
	if err := sn.DecodeMsg(r); err != nil {
		return err
	}
	if sn.Name != s.name {
		return fmt.Errorf("surf: a snapshot of %q given to %q", sn.Name, s.name)
	}
	s.forgetInstances()
	kids, err := html.ParseFragment(strings.NewReader(sn.Tree), bodyCtx)
	if err != nil {
		return err
	}
	if len(kids) != 1 || attr(kids[0], "id") != "root" {
		return errors.New("surf: the snapshot's tree is no #root")
	}
	s.reset()
	s.prev, s.nextID, s.focused, s.asked = kids[0], sn.Next, sn.Focused, sn.Asked
	s.lastFocus = sn.Focused
	for k, v := range sn.IDs {
		s.ids[k] = v
	}
	for k, v := range sn.Vals {
		s.vals[k] = v
	}
	for k, v := range sn.Checked {
		s.checked[k] = v
	}
	for k, v := range sn.Revs {
		s.revs[k] = v
	}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if id := attr(n, "id"); id != "" {
				s.byID[id] = n
				if isField(n) {
					s.fields = append(s.fields, id)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(s.prev)
	s.focusIn = s.chain(s.focused)
	// Until the first Update renders, events have no handlers to reach.
	s.frame = nil
	s.wake()
	return nil
}

// restyle swaps the surface's stylesheet; the next Update sends it.
func (s *Surface) restyle(css string) error {
	if strings.Contains(strings.ToLower(css), "</style") {
		return errors.New("surf: the stylesheet holds </style")
	}
	s.css, s.cssDue = css, true
	s.wake()
	return nil
}
