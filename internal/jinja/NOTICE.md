Package jinja is the oracle of ui's render tests: Basecoat 1.1.0's Jinja
macros (internal/basecoat/jinja; Copyright (c) 2025 Ronan Berder, MIT
licensed) translated to templ by machine, untyped, with Jinja's value
semantics in rt.go. ui's components are typed templ written by hand to
render what the macros render; each render test calls a macro here
(Call) with the arguments a component's Props stand for, and compares.

Everything here but jinja_test.go and testdata/cases.json is written by
gen/jinja2templ.py (python3 with jinja2), which `make oracle` runs:

    python3 -I gen/jinja2templ.py ../basecoat/jinja .
    templ fmt . && templ generate -path .
    python3 -I gen/jinja2templ.py --testdata ../basecoat/jinja .

TestJinja holds the translation to jinja2's own render of each case of
testdata/cases.json. On a Basecoat bump, `make oracle`, then the render
tests that fail name the components whose markup changed.

What differs from the macros on purpose (the translator's choices):
- Values the caller passes (attribute dicts, item URLs, toast onclick) are
  marked trusted (templ.SafeURL, templ.JSUnsafeFuncCall, templ.SafeCSS), as
  the macros print them unsanitised.
- Dialog takes its caller as a parameter, since it tests `caller is
  defined` and a templ component can't tell whether it was given children.
- Go maps have no order, so a dict iterated as a list (combobox's
  `selected` given as a dict) yields its keys sorted.
