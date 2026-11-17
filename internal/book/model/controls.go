package storybook

import (
	"reflect"
	"strconv"
	"strings"
)

// Control is a prop the user can change in the book: a bool, a string, an
// int, or a string with a few values (an enum, from the field's `enum`
// tag: `enum:"default,secondary,destructive"`).
type Control struct {
	Path    string // the field's path in the props: Popup.Open
	Kind    string // "bool", "string", "int", "enum"
	Value   string // as text: "true", "42", "ghost"
	Options []string
}

// Controls are the story's props that have a control: every exported
// field of those kinds, in structs and pointers to structs within the
// props, in field order. A field tagged `sb:"-"` has none.
func (s *Story) Controls() []Control {
	var out []Control
	controls(&out, "", reflect.ValueOf(s.props), "", 0)
	return out
}

func controls(out *[]Control, path string, v reflect.Value, enum string, depth int) {
	if depth > 8 || !v.IsValid() {
		return
	}
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			controls(out, path, v.Elem(), enum, depth+1)
		}
	case reflect.Struct:
		t := v.Type()
		for i := range t.NumField() {
			f := t.Field(i)
			if !f.IsExported() || f.Tag.Get("sb") == "-" {
				continue
			}
			controls(out, join(path, f.Name), v.Field(i), f.Tag.Get("enum"), depth+1)
		}
	case reflect.Bool:
		*out = append(*out, Control{Path: path, Kind: "bool", Value: strconv.FormatBool(v.Bool())})
	case reflect.String:
		if enum != "" {
			*out = append(*out, Control{Path: path, Kind: "enum", Value: v.String(), Options: strings.Split(enum, ",")})
		} else {
			*out = append(*out, Control{Path: path, Kind: "string", Value: v.String()})
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		*out = append(*out, Control{Path: path, Kind: "int", Value: strconv.FormatInt(v.Int(), 10)})
	}
}

// Set sets the control at path to value, as text. A value that doesn't
// parse, or a path with no control, changes nothing.
func (s *Story) Set(path, value string) bool {
	v := reflect.ValueOf(s.props)
	for _, name := range strings.Split(path, ".") {
		for v.Kind() == reflect.Pointer {
			if v.IsNil() {
				return false
			}
			v = v.Elem()
		}
		if v.Kind() != reflect.Struct {
			return false
		}
		f, ok := v.Type().FieldByName(name)
		if !ok || !f.IsExported() {
			return false
		}
		v = v.FieldByIndex(f.Index)
	}
	if !v.CanSet() {
		return false
	}
	switch v.Kind() {
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return false
		}
		v.SetBool(b)
	case reflect.String:
		v.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil || v.OverflowInt(n) {
			return false
		}
		v.SetInt(n)
	default:
		return false
	}
	return true
}
