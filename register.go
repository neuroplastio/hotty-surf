package surf

import (
	"errors"
	"fmt"
	"sync"

	"neuroplast.io/hotty-surf/surfui"
)

// A widget from outside (docs/surf.md, "Widgets from outside"): a package
// that isn't hotty-surf's adds a tag to the vocabulary with Register,
// and both modes and surflua read the entry alike, as they read the
// floor's.
//
//	func init() {
//		surf.MustRegister(surf.Widget{
//			Tag: "x-gauge", Doc: "a level in a bar",
//			Form: surf.FormNone,
//			Pkg:  "example.com/gauge", Func: "New", New: New,
//			Attrs: []surf.Attr{{Name: "level", Type: surf.TypeInt,
//				Func: "SetLevel", Apply: SetLevel}},
//			CSS: gaugeCSS,
//		})
//	}
//
// The interpreter calls New, and the compiled mode (surfgen) imports Pkg
// and calls Func, so Func is New's name in Pkg and the two are one
// function; an attribute's Func and Apply are one in the same way.

var reg sync.Mutex

// Register adds a widget to the vocabulary: its tag, its attributes, its
// events, its constructor and its stylesheet. It is for a package's init,
// before any Load. It refuses, and adds nothing, where:
//
//   - the tag has no dash (the floor's tags have none; a dash is what
//     tells a registered widget, or a component, from a floor widget's),
//     or isn't lowercase letters, digits and dashes;
//   - the tag is registered already;
//   - Form isn't FormKids, FormText or FormNone, or New isn't the func
//     for it (Widget.New), or Pkg or Func is empty;
//   - an attribute has no name or Func, is of a type but string, bool or
//     int, is a constructor's argument, has an Apply of the wrong type,
//     names an attribute every widget takes, or repeats one.
//
// A template of the same name, a component's or a partial's, is a load
// error (Load), as a template can't be both.
func Register(w Widget) error {
	if err := checkWidget(&w); err != nil {
		return fmt.Errorf("surf: register <%s>: %w", w.Tag, err)
	}
	reg.Lock()
	defer reg.Unlock()
	if _, dup := widget(w.Tag); dup {
		return fmt.Errorf("surf: register <%s>: the tag is a widget's already", w.Tag)
	}
	attrs := make([]Attr, len(w.Attrs))
	for i, a := range w.Attrs {
		a.Tags, a.Pkg = []string{w.Tag}, w.Pkg
		attrs[i] = a
	}
	w.Attrs = attrs
	w.Events = append([]string(nil), w.Events...)
	Widgets = append(Widgets, w)
	Attrs = append(Attrs, attrs...)
	surfui.AddCSS(w.CSS)
	return nil
}

// MustRegister is Register, for a package's init: it panics where
// Register refuses.
func MustRegister(w Widget) {
	if err := Register(w); err != nil {
		panic(err)
	}
}

// Registered are the widgets registered from outside, in the order they
// were.
func Registered() []Widget {
	reg.Lock()
	defer reg.Unlock()
	var out []Widget
	for _, w := range Widgets {
		if w.New != nil {
			out = append(out, w)
		}
	}
	return out
}

// checkTag says what is wrong with a registered widget's tag.
func checkTag(tag string) error {
	dash := false
	for i, r := range tag {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9', r == '-':
			if i == 0 {
				return errors.New("the tag starts with a letter")
			}
			dash = dash || r == '-'
		default:
			return fmt.Errorf("the tag is lowercase letters, digits and dashes, not %q", tag)
		}
	}
	if !dash {
		return errors.New("the tag has a dash (a floor widget's has none)")
	}
	if tag[len(tag)-1] == '-' {
		return errors.New("the tag doesn't end in a dash")
	}
	return nil
}

func checkWidget(w *Widget) error {
	if err := checkTag(w.Tag); err != nil {
		return err
	}
	if w.Pkg == "" || w.Func == "" {
		return errors.New("Pkg and Func say where the compiled mode finds the constructor")
	}
	var ok bool
	switch w.Form {
	case FormKids:
		_, ok = w.New.(func(...*surfui.Node) *surfui.Node)
	case FormText:
		_, ok = w.New.(func(string) *surfui.Node)
	case FormNone:
		_, ok = w.New.(func() *surfui.Node)
	default:
		return errors.New("Form is FormKids, FormText or FormNone")
	}
	if !ok {
		return fmt.Errorf("New is %T, not the constructor a form %d takes (Widget.New)", w.New, w.Form)
	}
	seen := map[string]bool{}
	for _, a := range w.Attrs {
		if err := checkAttr(a); err != nil {
			return fmt.Errorf("attribute %q: %w", a.Name, err)
		}
		if seen[a.Name] {
			return fmt.Errorf("attribute %q twice", a.Name)
		}
		seen[a.Name] = true
		for _, f := range Attrs {
			if f.Name == a.Name && len(f.Tags) == 0 {
				return fmt.Errorf("attribute %q is one every widget takes", a.Name)
			}
		}
	}
	for _, e := range w.Events {
		if e == "" {
			return errors.New("an event with no name")
		}
	}
	return nil
}

func checkAttr(a Attr) error {
	switch a.Name {
	case "", "if", "else-if", "else", "each", "slot", "key":
		return errors.New("the name is taken, or none")
	}
	if a.Arg {
		return errors.New("a registered widget's attribute is a modifier, not a constructor's argument")
	}
	if a.Func == "" {
		return errors.New("Func says where the compiled mode finds the modifier")
	}
	var ok bool
	switch a.Type {
	case TypeString:
		if a.Fails {
			_, ok = a.Apply.(func(*surfui.Node, string) error)
		} else {
			_, ok = a.Apply.(func(*surfui.Node, string))
		}
	case TypeBool:
		_, ok = a.Apply.(func(*surfui.Node, bool))
		if a.Fails {
			return errors.New("a bool attribute doesn't fail")
		}
	case TypeInt:
		_, ok = a.Apply.(func(*surfui.Node, int64))
		if a.Fails {
			return errors.New("an int attribute doesn't fail")
		}
	default:
		return errors.New("Type is TypeString, TypeBool or TypeInt")
	}
	if !ok {
		return fmt.Errorf("Apply is %T, not a func(*surfui.Node, T) for its type (Attr.Apply)", a.Apply)
	}
	if len(a.Values) > 0 && a.Type != TypeString {
		return errors.New("Values are a string attribute's")
	}
	return nil
}
