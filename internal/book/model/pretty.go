package storybook

import (
	"fmt"
	"go/format"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// Pretty is v as a Go composite literal, gofmt'd: what the Props tab shows.
// Only exported fields are shown; a func is shown by its type.
func Pretty(v any) string {
	var b strings.Builder
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer && !rv.IsNil() {
		rv = rv.Elem()
	}
	lit(&b, rv, 0)
	src := "package p\n\nvar props = " + b.String() + "\n"
	out, err := format.Source([]byte(src))
	if err != nil {
		return b.String()
	}
	s := strings.TrimPrefix(string(out), "package p\n\nvar props = ")
	return strings.TrimSuffix(s, "\n")
}

func lit(b *strings.Builder, v reflect.Value, depth int) {
	if !v.IsValid() {
		b.WriteString("nil")
		return
	}
	if depth > 8 {
		b.WriteString("nil /* … */")
		return
	}
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			b.WriteString("nil")
			return
		}
		b.WriteByte('&')
		lit(b, v.Elem(), depth+1)
	case reflect.Interface:
		if v.IsNil() {
			b.WriteString("nil")
			return
		}
		lit(b, v.Elem(), depth+1)
	case reflect.Struct:
		t := v.Type()
		b.WriteString(typeName(t))
		b.WriteString("{")
		n := 0
		for i := range t.NumField() {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			if n == 0 {
				b.WriteByte('\n')
			}
			n++
			b.WriteString(f.Name)
			b.WriteString(": ")
			lit(b, v.Field(i), depth+1)
			b.WriteString(",\n")
		}
		b.WriteString("}")
	case reflect.Slice, reflect.Array:
		if v.Kind() == reflect.Slice && v.IsNil() {
			b.WriteString("nil")
			return
		}
		b.WriteString(typeName(v.Type()))
		b.WriteString("{")
		for i := range v.Len() {
			if i == 0 {
				b.WriteByte('\n')
			}
			lit(b, v.Index(i), depth+1)
			b.WriteString(",\n")
		}
		b.WriteString("}")
	case reflect.Map:
		if v.IsNil() {
			b.WriteString("nil")
			return
		}
		b.WriteString(typeName(v.Type()))
		b.WriteString("{")
		keys := v.MapKeys()
		sort.Slice(keys, func(i, j int) bool { return fmt.Sprint(keys[i]) < fmt.Sprint(keys[j]) })
		for i, k := range keys {
			if i == 0 {
				b.WriteByte('\n')
			}
			lit(b, k, depth+1)
			b.WriteString(": ")
			lit(b, v.MapIndex(k), depth+1)
			b.WriteString(",\n")
		}
		b.WriteString("}")
	case reflect.Func:
		if v.IsNil() {
			b.WriteString("nil")
			return
		}
		// A func value can't be written as a literal: a comment says
		// what it is, so that the literal still parses (and gofmt aligns
		// it).
		b.WriteString("nil /* " + typeName(v.Type()) + " */")
	case reflect.String:
		b.WriteString(strconv.Quote(v.String()))
	default:
		b.WriteString(fmt.Sprint(v))
	}
}

// typeName is t as a Go type, its package by its name (basecoat.Popup).
func typeName(t reflect.Type) string { return t.String() }

// flatten is v's leaves by path (Popup.Open: "true"), for telling what a
// handler changed. Only exported fields; funcs are left out.
func flatten(v any) map[string]string {
	m := map[string]string{}
	flat(m, "", reflect.ValueOf(v), 0)
	return m
}

func flat(m map[string]string, path string, v reflect.Value, depth int) {
	if !v.IsValid() || depth > 6 {
		return
	}
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			m[path] = "nil"
			return
		}
		flat(m, path, v.Elem(), depth+1)
	case reflect.Struct:
		t := v.Type()
		for i := range t.NumField() {
			if f := t.Field(i); f.IsExported() {
				flat(m, join(path, f.Name), v.Field(i), depth+1)
			}
		}
	case reflect.Slice, reflect.Array:
		m[join(path, "len")] = strconv.Itoa(v.Len())
		for i := range min(v.Len(), 32) {
			flat(m, path+"["+strconv.Itoa(i)+"]", v.Index(i), depth+1)
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			flat(m, path+"["+fmt.Sprint(k)+"]", v.MapIndex(k), depth+1)
		}
	case reflect.Func:
	case reflect.String:
		m[path] = strconv.Quote(v.String())
	default:
		m[path] = fmt.Sprint(v)
	}
}

func join(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}

// changes says what differs from a to b, one "path: old → new" a change,
// in path order.
func changes(a, b map[string]string) []string {
	var out []string
	for k, v := range b {
		if o, ok := a[k]; !ok {
			out = append(out, k+": "+v)
		} else if o != v {
			out = append(out, k+": "+o+" → "+v)
		}
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			out = append(out, k+": gone")
		}
	}
	sort.Strings(out)
	return out
}
