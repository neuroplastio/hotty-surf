package surfui

// A form is a stack whose fields submit it: Enter in a one-line text field
// commits the field and then submits (SPEC §9, V6-03), and a Submit button
// does too. The program hears one "submit" (OnSubmit), after the fields'
// own changes, so that what it checks is what the user typed. What a form
// checks is the program's: its handler looks at what the fields gave it,
// sets each field's Error, and refuses the submit by doing nothing else.
// There is no validation language.
//
// A field of several lines never submits on Enter: its Enter is a line
// break.

// Form is a Stack of kids whose text fields submit it on Enter.
func Form(kids ...*Node) *Node { return El("form", kids...).Class("u-stack") }

// OnSubmit runs act when the form n is submitted: by Enter in one of its
// fields of one line, or by a Submit button. The fields it holds have
// committed their values first (OnChange). The handler is the form's
// check: it refuses by setting the fields' Error and doing nothing else.
func (n *Node) OnSubmit(act Action) *Node { return n.On("submit", act) }

// Submit makes a Button submit its form, as a Button of a form that is
// none of these does not.
func (n *Node) Submit() *Node {
	if n.tag == "button" {
		n.Attr("type", "submit")
	}
	return n
}

// Submits reports whether n is a Submit button.
func Submits(n *Node) bool {
	if n.tag != "button" && n.tag != "input" {
		return false
	}
	t, _ := AttrOf(n, "type")
	return t == "submit"
}

// Password makes a Field's text a password's: a host masks it, and the
// cells draw each character as •. Its text is never copied.
func (n *Node) Password() *Node {
	if n.tag == "input" {
		n.Attr("type", "password")
	}
	return n
}

// FormOf is the form that n is in, nil for none.
func FormOf(n *Node) *Node {
	for p := n.parent; p != nil; p = p.parent {
		if p.tag == "form" {
			return p
		}
	}
	return nil
}

// Label puts a caption on a Field: a text above it, before it in a Dense
// surface; it is also the field's accessible label. On a Bare field
// (which sits in another control) and on any other element, it is only
// the accessible label. It must come before the field is added to its
// parent, which draws it.
func (n *Node) Label(s string) *Node {
	n.Attr("aria-label", s)
	if isField(n) {
		n.caption, n.captioned = s, true
	}
	return n
}

// AriaLabel sets only the accessible label of n, which Label of a Field
// would also draw.
func (n *Node) AriaLabel(s string) *Node { return n.Attr("aria-label", s) }

// Error puts a message under a Field, in the error tone, and makes the
// field Invalid; "" is no error. It must come before the field is added
// to its parent, which draws it.
func (n *Node) Error(msg string) *Node {
	if !isField(n) {
		return n
	}
	n.message = msg
	return n.Invalid(msg != "")
}

// ErrorOf is the message of a Field's Error, "" for none.
func ErrorOf(n *Node) string { return n.message }

func isField(n *Node) bool { return n.tag == "input" || n.tag == "textarea" }

// decorate gives a Field with a Label or an Error the stack that draws
// them, in place of the field in its parent: its label and the field
// (side by side in a Dense surface), then its message:
//
//	stack
//	  stack.u-labelled  label, field
//	  text.u-tone-error message
//
// The field keeps its key, or is "field", so that its id stays the same
// as a message comes and goes; the stack takes its key and its u-grow
// and u-keep. A Bare field is drawn by the control that holds it: its
// Label is only the accessible one, and it gets no stack unless it has
// an Error.
func decorate(k *Node) *Node {
	if !isField(k) {
		return k
	}
	label := k.captioned && k.caption != "" && !hasClassOf(k, "u-bare")
	if !label && k.message == "" {
		return k
	}
	if k.key == "" {
		k.key = "field"
	}
	line := Stack().Class("u-labelled")
	if label {
		line.Add(Text(k.caption).Muted().Class("u-label").Keep())
	}
	line.kids = append(line.kids, k) // not Add, which would draw it again
	out := Stack(line).Key(k.key)
	if k.message != "" {
		out.Add(Text(k.message).Class("u-tone-error", "u-error").Wrap().Key("error"))
	}
	for _, c := range []string{"u-grow", "u-keep"} {
		if hasClassOf(k, c) {
			out.Class(c)
		}
	}
	return out
}
