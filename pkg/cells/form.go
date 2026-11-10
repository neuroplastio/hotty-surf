package cells

import (
	"neuroplast.io/hotty-go"
	"neuroplast.io/hotty-surf/surfui"
)

// submitField is Enter in the focused text field n: the field commits (its
// change), then the form it is in is submitted, as a host does it (SPEC
// V6-03, §9). A field of several lines never gets here: its Enter is a
// line break (hotty.Resolve).
func (r *Renderer) submitField(n *surfui.Node) bool {
	due := r.commit()
	if f := surfui.FormOf(n); f != nil && surfui.Tag(n) == "input" {
		due = r.submit(f, nil) || due
	}
	return due
}

// submit sends the form's submit, with its fields as a host sends them
// (SPEC §9): each named control's name and value, in tree order, and the
// submitter's own where it has a name.
func (r *Renderer) submit(form, by *surfui.Node) bool {
	e := hotty.Event{Kind: hotty.EventSubmit, Target: surfui.IDOf(form)}
	var walk func(n *surfui.Node)
	walk = func(n *surfui.Node) {
		if surfui.Tag(n) == "" {
			return
		}
		if name, ok := surfui.AttrOf(n, "name"); ok && name != "" {
			if v, ok := r.formValue(n, n == by); ok {
				e.Fields = append(e.Fields, hotty.FormField{Name: name, Value: v})
			}
		}
		for _, k := range surfui.Kids(n) {
			walk(k)
		}
	}
	walk(form)
	return r.send(e)
}

// formValue is the value a control gives a form: a text field's as the
// user left it, a checked box's value or "on", the submitter's value; none
// for what the form leaves out.
func (r *Renderer) formValue(n *surfui.Node, submitter bool) (string, bool) {
	switch surfui.Tag(n) {
	case "input", "textarea":
		if st := r.fields[surfui.IDOf(n)]; st != nil {
			return st.ed.Value, true
		}
		t, _ := surfui.AttrOf(n, "type")
		v, hasValue := surfui.AttrOf(n, "value")
		switch t {
		case "checkbox", "radio":
			if _, on := surfui.AttrOf(n, "checked"); !on {
				return "", false
			}
			if !hasValue {
				v = "on"
			}
			return v, true
		case "button", "reset", "image", "file":
			return "", false
		case "submit":
			return v, submitter
		}
		return v, true
	case "button":
		v, _ := surfui.AttrOf(n, "value")
		return v, submitter
	}
	return "", false
}
