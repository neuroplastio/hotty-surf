kit.css is Basecoat 1.1.0's theme and components (Copyright (c) 2025 Ronan
Berder, MIT licensed) with the Tailwind CSS 4.1.18 utilities (Copyright (c)
Tailwind Labs, Inc., MIT licensed) the kit's markup uses, built by
style/build.mjs from internal/basecoat, Basecoat's own files.

icons_gen.go is lucide-react 0.474.0's icons (Copyright (c) Lucide
Contributors 2022, ISC licensed), the subset the kit draws.

select.templ, menu.templ, dialog.templ, tabs.templ and toast.templ write
the markup of Basecoat 1.1.0's Jinja macros (templates/jinja/*.html.jinja;
Copyright (c) 2025 Ronan Berder, MIT licensed) in templ, and select.go,
menu.go, tabs.go and toast.go translate its scripts (dist/js/select.js,
dropdown-menu.js, tabs.js and toast.js, under the same license) to Go;
none of its JavaScript is used. toast.go's icons are the toast macro's
SVGs, and dialog.templ's close icon the dialog macro's.

code highlights with Chroma (github.com/alecthomas/chroma/v2, Copyright
(C) 2017 Alec Thomas, MIT licensed), its lexers' definitions included.

code.View uses the embedded XML lexers in internal/highlight/lexers. The
CSS, HTML, JavaScript and templ lexers are from Chroma (MIT; their license
is in that directory's COPYING). The Go lexer is from hotty-a2ui
(Apache-2.0, neuroplastio).

shapes_gen.go contains Material Symbols paths (Copyright Google, Apache
License 2.0), from @iconify-json/material-symbols 1.2.93.
