// Package code shows highlighted code and line numbers in the book.
//
// The token colours are a theme of their own, for .dark and .light: add
// CSS to the app's stylesheet (style.KitCSS + app CSS + code.CSS).
package code

//go:generate templ generate -path .

import (
	"embed"
	"path"
	"strings"
	"sync"

	"github.com/alecthomas/chroma/v2"
)

// CSS is the token colours (code.css), by theme.
//
//go:embed code.css
var CSS string

// The lexers code is highlighted with: chroma's definitions (MIT,
// lexers/COPYING) for templ, Go, and HTML with the CSS and JavaScript it
// embeds, from chroma v2.27.0; go.xml is hotty-a2ui's (neuroplastio), which writes
// chroma's Go lexer (Go code there) as XML. Linking chroma's lexers
// package instead adds 4 MB to the binary and ≈ 0.2 s to each dev build.
//
//go:embed lexers/*.xml
var lexerXML embed.FS

var lexerRegistry = sync.OnceValue(func() *chroma.LexerRegistry {
	r := chroma.NewLexerRegistry()
	ents, _ := lexerXML.ReadDir("lexers")
	for _, e := range ents {
		if l, err := chroma.NewXMLLexer(lexerXML, path.Join("lexers", e.Name())); err == nil {
			r.Register(l)
		}
	}
	return r
})

// Token is a run of code of one kind, within a line. The kinds are
// hotty-a2ui's highlight package's (Apache-2.0, neuroplastio), which a
// rendition colours by role: "" (plain), "comment", "keyword", "type",
// "function", "string", "literal", "meta", "invalid".
type Token struct {
	Text string
	Kind string
}

// Highlight is code as lines of tokens, lexed by chroma's lexer for lang
// ("go", "templ", "html"). A line holds no "\n", tabs are four spaces, and
// no token is only spaces: they join the token after them (an HTML
// rendition's runtime drops whitespace-only text beside elements).
func Highlight(code, lang string) [][]Token {
	key := lang + "\x00" + code
	hlMu.Lock()
	if ls, ok := hlCache[key]; ok {
		hlMu.Unlock()
		return ls
	}
	hlMu.Unlock()
	ls := lex(code, lang)
	hlMu.Lock()
	if len(hlCache) > 256 {
		clear(hlCache)
	}
	hlCache[key] = ls
	hlMu.Unlock()
	return ls
}

var (
	hlMu    sync.Mutex
	hlCache = map[string][][]Token{}
)

func lex(code, lang string) [][]Token {
	code = strings.ReplaceAll(strings.TrimRight(code, "\n"), "\t", "    ")
	var toks []chroma.Token
	if l := lexerRegistry().Get(lang); l != nil {
		if it, err := chroma.Coalesce(l).Tokenise(nil, code+"\n"); err == nil {
			toks = it.Tokens()
		}
	}
	if toks == nil {
		toks = []chroma.Token{{Type: chroma.Text, Value: code + "\n"}}
	}
	out := [][]Token{nil}
	pending := "" // spaces waiting for the next token on the line
	for _, t := range toks {
		kind := kindOf(t.Type)
		for i, part := range strings.Split(t.Value, "\n") {
			if i > 0 {
				end(&out, &pending)
				out = append(out, nil)
			}
			if part == "" {
				continue
			}
			if strings.TrimSpace(part) == "" {
				pending += part
				continue
			}
			part, pending = pending+part, ""
			last := &out[len(out)-1]
			if n := len(*last); n > 0 && (*last)[n-1].Kind == kind {
				(*last)[n-1].Text += part
				continue
			}
			*last = append(*last, Token{Text: part, Kind: kind})
		}
	}
	end(&out, &pending)
	if len(out) > 1 && len(out[len(out)-1]) == 0 {
		out = out[:len(out)-1]
	}
	return out
}

// end gives a line's trailing spaces to its last token.
func end(out *[][]Token, pending *string) {
	if *pending == "" {
		return
	}
	if last := &(*out)[len(*out)-1]; len(*last) > 0 {
		(*last)[len(*last)-1].Text += *pending
	}
	*pending = ""
}

func kindOf(t chroma.TokenType) string {
	switch {
	case t == chroma.Error || t == chroma.GenericError:
		return "invalid"
	case t.InSubCategory(chroma.CommentPreproc):
		return "meta"
	case t.InCategory(chroma.Comment):
		return "comment"
	case t == chroma.KeywordType:
		return "type"
	case t == chroma.KeywordConstant:
		return "literal"
	case t.InCategory(chroma.Keyword):
		return "keyword"
	case t.InSubCategory(chroma.NameBuiltin), t == chroma.NameClass, t == chroma.NameTag, t == chroma.NameException:
		return "type"
	case t.InSubCategory(chroma.NameFunction):
		return "function"
	case t == chroma.NameConstant:
		return "literal"
	case t == chroma.NameDecorator, t == chroma.NameAttribute:
		return "meta"
	case t.InSubCategory(chroma.LiteralString):
		return "string"
	case t.InSubCategory(chroma.LiteralNumber), t == chroma.LiteralDate:
		return "literal"
	}
	return ""
}

// TokenClass is the class View gives a token of kind ("tok-keyword"), which
// CSS colours; "" for plain text.
func TokenClass(kind string) string {
	if kind == "" {
		return ""
	}
	return "tok-" + kind
}
