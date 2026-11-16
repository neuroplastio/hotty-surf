#!/bin/sh
# coverage-md.sh writes docs/coverage.md from ui/coverage.tsv, so the page
# and the matrix never drift. Edit the tsv, then run this:
#
#	sh scripts/coverage-md.sh [out]
#
# out is docs/coverage.md unless given (a test can write elsewhere and
# compare).
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
tsv=$root/ui/coverage.tsv
out=${1:-$root/docs/coverage.md}

[ -r "$tsv" ] || { echo "coverage-md: no $tsv" >&2; exit 1; }

tmp=$out.tmp.$$
trap 'rm -f "$tmp"' EXIT INT TERM

{
	cat <<'EOF'
---
group: Guides
parent: hotty-surf
order: 60
nav: Basecoat coverage
path: hotty-surf/coverage
description: every Basecoat 1.1.0 component, how package ui makes it and tests it
---

# Basecoat coverage

Package `ui` gives every component of Basecoat 1.1.0 (shadcn's look in
plain CSS classes) a typed Go API for templ. This page says, for each of
Basecoat's 39 component files, how `ui` makes it, what it leans on in the
host, its names, its stories and its tests.

This page is written by `scripts/coverage-md.sh` from
[`ui/coverage.tsv`](../ui/coverage.tsv), which the coverage test reads.
Edit the tsv, then run the script; don't edit this page.

## The columns

- **component**: Basecoat's file name, `dist/components/<name>.css`.
- **markup**, how the HTML is made:
  - `jinja`: Basecoat ships a Jinja macro for it
    (`templates/jinja/*.html.jinja`), translated to typed templ;
  - `css`: Basecoat has CSS only; the typed wrapper is generated from a
    table of its classes and variants, read out of Basecoat's CSS;
  - `hand`: written by hand, because translation can't make it (the
    notes say why).
- **behaviour**: the script of Basecoat's `dist/js` translated to Go, `-`
  for none, or `hand` where there is no script to translate or it can't be
  translated (the notes say why).
- **browser**: a browser feature the component relies on instead of
  script, or `-`. The notes say what a hotty host (hottyterm on Blitz) is
  known to do with it.
- **api**: the exported names: the function, its Props, its State if it
  has one, its parts and its item types.
- **stories**: `<component>/<story>`, kebab case; the spike's storybook
  names where it has the story.
- **tests**, each a kind of test the component needs:
  - `render`: the generated markup equals Basecoat's Jinja render;
  - `spec`: every class and attribute value the wrapper writes is one
    Basecoat's CSS defines;
  - `differential`: the Go behaviour against Basecoat's script in
    Chromium, step by step;
  - `keys-cells`: the stories can be used by keyboard alone in the cells
    rendition.
- **status**: `done`, the spike has a working, tested equivalent; `port`,
  code exists to move (generated markup, a hand-written kit component, a
  story); `new`, nothing yet.
- **notes**: why a hand port, a Blitz gap, a hotty limit, variants that
  don't fit the shared enums.

## The convention

- A component is a function returning a `templ.Component`, named for
  Basecoat's file: `ui.Button`, `ui.DropdownMenu`, `ui.NativeSelect`.
- Its attributes are one struct, `ui.<Name>Props`; its zero value is
  Basecoat's default.
- Behaviour state the app shouldn't manage (open, the highlighted item, a
  filter) is in `ui.<Name>State`, which the app keeps and passes by
  pointer. The app's own data (the value, the items) stays in Props, with
  `On…` callbacks.
- Parts are `ui.<Name><Part>`; a part that is a plain element in
  Basecoat's markup (a card's `<header>`) stays a plain element. Items are
  plain data structs: `ui.SelectItem`, `ui.Tab`.
- Variants and sizes are shared: `ui.Variant`, `ui.Size`.

```templ
@ui.Select(&a.sizeMenu, ui.SelectProps{ID: "size", Items: sizes,
	Value: a.size, OnChange: a.setSize})
@ui.Button(ui.ButtonProps{ID: "save", OnClick: a.save}) { Save }
```

EOF

	LC_ALL=C awk -F'\t' '
	function code(s) { return "`" s "`" }
	# list renders a comma-separated cell as code spans, or "-".
	function list(s,   n, a, i, r) {
		if (s == "-" || s == "") return "-"
		n = split(s, a, ",")
		r = ""
		for (i = 1; i <= n; i++) {
			sub(/^ +/, "", a[i])
			r = r (i > 1 ? ", " : "") code(a[i])
		}
		return r
	}
	# prose keeps a note as text, with its tags as code and pipes escaped.
	function prose(s) {
		gsub(/\|/, "\\|", s)
		gsub(/<[^>]+>/, "`&`", s)
		return s
	}
	function plural(n, one, many) { return n " " (n == 1 ? one : many) }
	NR == 1 { next }
	{
		rows++
		markup[$2]++
		status[$8]++
		if ($3 == "-") none++
		else if ($3 == "hand") { hand++; handl = handl (hand > 1 ? ", " : "") $1 }
		else { js++; jsl = jsl (js > 1 ? ", " : "") code($3) }
		if ($4 != "-") browser++
		n = split($7, t, ",")
		for (i = 1; i <= n; i++) if (t[i] != "-") tests[t[i]]++
		line[rows] = "| " $1 " | " $2 " | " ($3 == "-" || $3 == "hand" ? $3 : code($3)) \
			" | " list($4) " | " list($5) " | " list($6) " | " list($7) \
			" | " $8 " | " prose($9) " |"
	}
	END {
		print "## The counts"
		print ""
		print "Of " rows " components:"
		print ""
		print "- **markup:** " markup["jinja"] " jinja, " markup["css"] " css, " \
			(markup["hand"] + 0) " hand."
		print "- **behaviour:** " plural(js, "script", "scripts") \
			" translated (" jsl "); " plural(hand, "component", "components") \
			" by hand (" handl "); " none " with none."
		print "- **browser:** " plural(browser, "component relies", "components rely") \
			" on a browser feature."
		print "- **status:** " (status["done"] + 0) " done, " (status["port"] + 0) \
			" port, " (status["new"] + 0) " new."
		print "- **tests:** render " (tests["render"] + 0) ", spec " (tests["spec"] + 0) \
			", differential " (tests["differential"] + 0) ", keys-cells " \
			(tests["keys-cells"] + 0) "."
		print ""
		print "## The components"
		print ""
		print "| component | markup | behaviour | browser | api | stories | tests | status | notes |"
		print "| --- | --- | --- | --- | --- | --- | --- | --- | --- |"
		for (i = 1; i <= rows; i++) print line[i]
	}' "$tsv"
} >"$tmp"

mv "$tmp" "$out"
trap - EXIT INT TERM
