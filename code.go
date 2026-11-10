package surf

import (
	"fmt"

	cellite "neuroplast.io/cel-lite"
	"neuroplast.io/hotty-surf/surfui"
)

// ModLanguage selects a code block's lexer in both template modes.
func ModLanguage(n *surfui.Node, language string) error {
	switch language {
	case "plain", "go", "surf":
		n.Language(language)
		return nil
	default:
		return fmt.Errorf("language is plain, go or surf, not %q", language)
	}
}

// ModSpans gives a code block its caller's spans in place of a lexer's,
// in both template modes. A role that is none of surfui.SpanRoles is an
// error, and the block keeps what it had.
func ModSpans(n *surfui.Node, spans []surfui.Span) error {
	if err := surfui.CheckSpans(spans); err != nil {
		return err
	}
	n.Spans(spans)
	return nil
}

// ModNumbers shows a code block's line numbers.
func ModNumbers(n *surfui.Node, v bool) { n.Numbers(v) }

// ModFrom is the number of a code block's first line.
func ModFrom(n *surfui.Node, v int64) { n.From(int(v)) }

// SpansOf is the spans a list holds, as :spans reads it: a list of
// surfui.Span (the struct a Go field of []surfui.Span gives a template),
// or of maps with the keys text and role (a Lua table's {text, role}, a
// literal's {'text': …, 'role': …}); role may be left out. It is the one
// reading of both template modes. A list of anything else, and a span
// whose text or role isn't a string, is an error.
func SpansOf(v cellite.Value) ([]surfui.Span, error) {
	if v.Kind() != cellite.ListKind {
		return nil, fmt.Errorf("spans are a list of {text, role}, not %s", kindName(v))
	}
	l := v.List()
	out := make([]surfui.Span, 0, l.Len())
	for i := range l.Len() {
		s, err := spanOf(l.At(i))
		if err != nil {
			return nil, fmt.Errorf("span %d: %w", i, err)
		}
		out = append(out, s)
	}
	return out, nil
}

func spanOf(v cellite.Value) (surfui.Span, error) {
	switch v.Kind() {
	case cellite.MapKind:
		var s surfui.Span
		for _, f := range []struct {
			key string
			to  *string
			req bool
		}{{"text", &s.Text, true}, {"role", &s.Role, false}} {
			x, ok := v.Map().Get(cellite.StringValue(f.key))
			if !ok || x.Kind() == cellite.NullKind {
				if f.req {
					return s, fmt.Errorf("no text")
				}
				continue
			}
			if x.Kind() != cellite.StringKind {
				return s, fmt.Errorf("%s is a string, not %s", f.key, kindName(x))
			}
			*f.to = x.String()
		}
		return s, nil
	case cellite.StructKind:
		if u, ok := v.Object().(interface{ Unwrap() any }); ok {
			switch p := u.Unwrap().(type) {
			case *surfui.Span:
				return *p, nil
			case surfui.Span:
				return p, nil
			}
		}
	}
	return surfui.Span{}, fmt.Errorf("a span is a surfui.Span or a map with text and role, not %s", kindName(v))
}
