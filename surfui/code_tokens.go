package surfui

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// codeToken is source text and its lexical category. Presentation is
// separate: a token's spelling is never interpreted as markup.
type codeToken struct {
	text string
	kind string
}

// codeTokens is a deliberately small highlighter, not a parser. Even
// incomplete or invalid source is retained, byte for byte. Each scan
// consumes input, so a malformed snippet cannot stall a render.
func codeTokens(source, language string) []codeToken {
	if source == "" {
		return nil
	}
	if language != "go" && language != "surf" {
		return []codeToken{{source, "plain"}}
	}
	var out []codeToken
	whole := source
	inTag, tagName, expression := false, false, false
	for len(source) > 0 {
		n, kind := 1, "plain"
		switch {
		case language == "go":
			n, kind = codeLex(source)
		case expression:
			if strings.HasPrefix(source, "}}") {
				n, expression = 2, false
			} else {
				n, kind = codeLex(source)
			}
		case strings.HasPrefix(source, "<!--"):
			n, kind = codeEnd(source, 4, "-->"), "comment"
		case !inTag && strings.HasPrefix(source, "{{"):
			n, expression = 2, true
		case !inTag && source[0] == '<':
			inTag, tagName = true, true
			if strings.HasPrefix(source, "</") {
				n = 2
			}
		case inTag && source[0] == '>':
			inTag = false
		case inTag && (source[0] == '\'' || source[0] == '"'):
			n, kind = codeQuote(source), "string"
		case inTag:
			n = codeName(source, true)
			if n > 0 {
				kind = "attribute"
				if tagName {
					kind, tagName = "tag", false
				}
			} else {
				n = 1
			}
		default:
			// Plain template text, including quotes and numbers, is
			// not a programming language until an expression opens.
			n = 1
			for n < len(source) && source[n] != '<' && !strings.HasPrefix(source[n:], "{{") {
				n++
			}
		}
		if last := len(out) - 1; last >= 0 && kind == "plain" && out[last].kind == kind {
			end := len(whole) - len(source) + n
			out[last].text = whole[end-n-len(out[last].text) : end]
		} else {
			out = append(out, codeToken{source[:n], kind})
		}
		source = source[n:]
	}
	return out
}

func codeLex(s string) (int, string) {
	switch {
	case strings.HasPrefix(s, "//"):
		if end := strings.IndexByte(s, '\n'); end >= 0 {
			return end, "comment"
		}
		return len(s), "comment"
	case strings.HasPrefix(s, "/*"):
		return codeEnd(s, 2, "*/"), "comment"
	case s[0] == '"' || s[0] == '\'' || s[0] == '`':
		return codeQuote(s), "string"
	case s[0] >= '0' && s[0] <= '9':
		n := 1
		for n < len(s) {
			c := s[n]
			if c >= '0' && c <= '9' || c >= 'a' && c <= 'f' ||
				c >= 'A' && c <= 'F' || strings.ContainsRune("._xXoOpPi", rune(c)) ||
				(c == '+' || c == '-') && strings.ContainsRune("eEpP", rune(s[n-1])) {
				n++
				continue
			}
			break
		}
		return n, "number"
	}
	if n := codeName(s, false); n > 0 {
		if strings.Contains(" bool byte complex64 complex128 error float32 float64 int int8 int16 int32 int64 rune string uint uint8 uint16 uint32 uint64 uintptr any comparable ", " "+s[:n]+" ") {
			return n, "type"
		}
		if strings.Contains(" break case chan const continue default defer else fallthrough for func go goto if import interface map package range return select struct switch type var true false nil null in ", " "+s[:n]+" ") {
			return n, "keyword"
		}
		return n, "plain"
	}
	_, n := utf8.DecodeRuneInString(s)
	return n, "plain"
}

func codeName(s string, attribute bool) int {
	n := 0
	for n < len(s) {
		r, w := utf8.DecodeRuneInString(s[n:])
		if r == '_' || unicode.IsLetter(r) || n > 0 && unicode.IsDigit(r) ||
			attribute && strings.ContainsRune(":@-", r) {
			n += w
		} else {
			break
		}
	}
	return n
}

func codeQuote(s string) int {
	quote := s[0]
	for n := 1; n < len(s); n++ {
		if s[n] == quote {
			return n + 1
		}
		if s[n] == '\\' && quote != '`' && n+1 < len(s) {
			n++
		}
	}
	return len(s)
}

func codeEnd(s string, start int, end string) int {
	if n := strings.Index(s[start:], end); n >= 0 {
		return start + n + len(end)
	}
	return len(s)
}
