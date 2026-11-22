// Package code shows source code highlighted, in the kit's look: Chroma's
// lexers, coloured by the theme's tokens. It is a package of its own so that
// an app that shows no code doesn't link Chroma. Block and View have
// distinct markup and lexer sets.
package code

import (
	"context"
	"html"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/a-h/templ"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"

	bookhighlight "neuroplast.io/hotty-surf/internal/core/highlight"
)

// Stylesheet is the light and dark palette and layout for View. Append
// it to the surface's stylesheet after the kit stylesheet.
func Stylesheet() string { return bookhighlight.CSS }

// View shows source with wrapping lines, optional line numbers starting
// at first, and the spike palette. first <= 0 hides the numbers. lang
// selects Go, templ, HTML, CSS or JavaScript; an unknown language is
// shown plain. Append Stylesheet() to the surface's stylesheet.
func View(src, lang string, first int) templ.Component {
	return bookhighlight.View(src, lang, first)
}

// TokenClass is the CSS class used by View for kind. An empty kind
// returns ""; another kind returns "tok-" followed by kind. The
// supplied palette colors comment, keyword, type, function, string,
// literal, meta and invalid. Append Stylesheet to use those colors.
func TokenClass(kind string) string { return bookhighlight.TokenClass(kind) }

// Block is src highlighted as lang (a Chroma lexer's name or alias: "go",
// "json", "sh"), in Basecoat's pre. first is the first line's number; 0
// shows no numbers. A lang Chroma doesn't know shows src plain. Block
// uses the kit stylesheet and Chroma's broad lexer set; View wraps its
// lines and uses a separate palette and a smaller lexer set.
func Block(src, lang string, first int) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, block(highlight(src, lang), lang, first))
		return err
	})
}

// block is Basecoat's pre (its base styles give it the mono face) holding
// the lines, a block each, so that each line is a box of its own, its
// number the box's first text. An empty line holds a space, to keep its
// height. It is written in Go, not templ, because every byte in a pre is
// shown.
func block(lines [][]token, lang string, first int) string {
	var b strings.Builder
	b.WriteString(`<pre data-slot="code"`)
	if lang != "" {
		b.WriteString(` data-lang="` + html.EscapeString(lang) + `"`)
	}
	b.WriteString(`><code>`)
	for i, line := range lines {
		b.WriteString(`<div>`)
		if first > 0 {
			b.WriteString(`<span class="text-muted-foreground" aria-hidden="true">` + number(first, i, len(lines)) + `</span>`)
		}
		for _, t := range line {
			text := html.EscapeString(t.text)
			if c := class(t.kind); c != "" {
				b.WriteString(`<span class="` + c + `">` + text + `</span>`)
			} else {
				b.WriteString(text)
			}
		}
		if len(line) == 0 && first <= 0 {
			b.WriteString(" ")
		}
		b.WriteString(`</div>`)
	}
	b.WriteString(`</code></pre>`)
	return b.String()
}

// A token is a run of a line's text of one kind: "" (plain), "comment",
// "keyword", "type", "function", "string", "literal", "meta" or
// "invalid", the roles the spike's code view coloured.
type token struct {
	text string
	kind string
}

// class is the class a token of kind is drawn with: the theme's, as the
// kit's stylesheet has them. The kinds it has no colour for are plain.
func class(kind string) string {
	switch kind {
	case "comment":
		return "text-muted-foreground"
	case "keyword":
		return "font-semibold"
	case "type", "function":
		return "font-medium"
	case "invalid":
		return "text-destructive"
	}
	return ""
}

// number is line i's number, padded on the left to the widest's width,
// and the gap after it.
func number(first, i, lines int) string {
	n := strconv.Itoa(first + i)
	w := len(strconv.Itoa(first + lines - 1))
	return strings.Repeat(" ", w-len(n)) + n + "  "
}

// highlight is src as lines of tokens, lexed by Chroma's lexer for lang. A
// line holds no "\n" and no "\r" (a CRLF or a CR ends a line, as Chroma
// reads it), a tab is four spaces, and no token is only spaces: they join
// the token after them, as the spike's code view had them. It keeps the
// last 256 it lexed, since a view renders a block on every frame.
func highlight(src, lang string) [][]token {
	key := lang + "\x00" + src
	mu.Lock()
	ls, ok := cache[key]
	mu.Unlock()
	if ok {
		return ls
	}
	ls = lex(src, lang)
	mu.Lock()
	if len(cache) >= 256 {
		clear(cache)
	}
	cache[key] = ls
	mu.Unlock()
	return ls
}

var (
	mu    sync.Mutex
	cache = map[string][][]token{}
	// lineEnds makes every line end "\n", for the plain path as Chroma
	// does for a lexer's.
	lineEnds = strings.NewReplacer("\r\n", "\n", "\r", "\n")
)

func lex(src, lang string) [][]token {
	src = lineEnds.Replace(src)
	src = strings.ReplaceAll(strings.TrimRight(src, "\n"), "\t", "    ")
	var toks []chroma.Token
	if l := lexers.Get(lang); l != nil && lang != "" {
		if it, err := chroma.Coalesce(l).Tokenise(nil, src+"\n"); err == nil {
			toks = it.Tokens()
		}
	}
	if toks == nil {
		toks = []chroma.Token{{Type: chroma.Text, Value: src + "\n"}}
	}
	out := [][]token{nil}
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
			if n := len(*last); n > 0 && (*last)[n-1].kind == kind {
				(*last)[n-1].text += part
				continue
			}
			*last = append(*last, token{text: part, kind: kind})
		}
	}
	end(&out, &pending)
	if len(out) > 1 && len(out[len(out)-1]) == 0 {
		out = out[:len(out)-1]
	}
	return out
}

// end gives a line's trailing spaces to its last token.
func end(out *[][]token, pending *string) {
	if *pending == "" {
		return
	}
	if last := &(*out)[len(*out)-1]; len(*last) > 0 {
		(*last)[len(*last)-1].text += *pending
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
