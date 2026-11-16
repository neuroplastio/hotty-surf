popup.templ and popup.go are the hotty-pot spike's translation of Basecoat
1.1.0's components (Copyright (c) 2025 Ronan Berder, MIT licensed) and of
shadcn/ui's (Copyright (c) 2023 shadcn, MIT licensed): their markup and
classes, so Basecoat's CSS styles them. The popups' behaviour that
Basecoat's JavaScript has (dropdown-menu.js, popover.js, command.js) is
written again in Go (popup.go); none of its JavaScript is used.

The app-chrome components (symbol.templ, busy.templ, listitem.templ,
layout.templ, chrome.go) were written for the spike. shapes_gen.go's paths
are Material Symbols' (Copyright Google, Apache License 2.0), taken from
@iconify-json/material-symbols 1.2.93.
