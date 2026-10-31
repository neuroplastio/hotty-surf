package conformance

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// UnmarshalJSON reads a Branch as encoding/json reads one, its fields by
// their Go names, but without recursing on the tree. A tree as deep as a
// render goes (cases/deep.txt) is 30 objects deep in its state.json, and
// encoding/json, which recurses on each, needs 1.6 KB of stack an object
// under TinyGo: the whole of a goroutine's 64 KB for that tree, where the
// render of it needs 16 KB at most (make stack). So the tests read the
// cases' trees here, with an explicit stack, from the decoder's tokens.
// It is the tests' alone, in a test file: surfgen doesn't see it.
func (b *Branch) UnmarshalJSON(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))
	if t, err := d.Token(); err != nil || t != json.Delim('{') {
		return fmt.Errorf("a Branch is an object: %v", err)
	}
	type open struct {
		b    *Branch
		kids bool // inside its Kids' array
	}
	stack := []open{{b: b}}
	for len(stack) > 0 {
		top := &stack[len(stack)-1]
		t, err := d.Token()
		if err != nil {
			return err
		}
		if top.kids {
			switch t {
			case json.Delim(']'):
				top.kids = false
			case json.Delim('{'):
				k := &Branch{}
				top.b.Kids = append(top.b.Kids, k)
				stack = append(stack, open{b: k})
			default:
				return fmt.Errorf("a Branch's Kids are Branches, not %v", t)
			}
			continue
		}
		if t == json.Delim('}') {
			stack = stack[:len(stack)-1]
			continue
		}
		key, _ := t.(string)
		switch key {
		case "ID", "Name", "Via":
			var s string
			if err := d.Decode(&s); err != nil {
				return fmt.Errorf("a Branch's %s: %v", key, err)
			}
			switch key {
			case "ID":
				top.b.ID = s
			case "Name":
				top.b.Name = s
			default:
				top.b.Via = s
			}
		case "Kids", "Next":
			v, err := d.Token()
			if err != nil {
				return err
			}
			switch {
			case v == nil:
			case key == "Kids" && v == json.Delim('['):
				top.kids = true
			case key == "Next" && v == json.Delim('{'):
				n := &Branch{}
				top.b.Next = n
				stack = append(stack, open{b: n})
			default:
				return fmt.Errorf("a Branch's %s is not %v", key, v)
			}
		default:
			return fmt.Errorf("a Branch has no field %q", key)
		}
	}
	return nil
}
