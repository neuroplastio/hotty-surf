# Q-0003 — A code block: its own highlighter, or the caller's spans

Status: open
Blocks: a2ui-exit L2 (code and its roles); fenced code in the docs and in
margin's comments; a diff later
Raised: 2026-11-09

## What I need decided

The branch's `<code language="go">` tokenizes Go and surf itself. The
website and margin highlight with chroma, for every language, and ask
for `<pre lang :spans wrap/>` taking `{text, role}` spans, with line
numbers (`numbers`, `:from`) for margin.

## Why I cannot decide it

It is the widget's shape and its name, and a diff (A13) builds on it.

## Options

1. Only the built-in highlighter (the branch). Callers lose their
   languages.
2. Only spans (the list). The tour's own Go and surf need a tokenizer
   from somewhere.
3. One widget, `code`, taking `language` or `:spans` in the same syntax
   roles, plus `wrap`, `numbers` and `from`; tabs to four spaces.

## My lean

3, keeping the branch's name and approved syntax tokens. No chroma
dependency in hotty-surf.
