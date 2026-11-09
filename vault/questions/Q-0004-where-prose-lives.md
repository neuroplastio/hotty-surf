# Q-0004 — Where prose, the Markdown converter and the Bubble Tea adapter live

Status: open
Blocks: a2ui-exit L8–L12, L14
Raised: 2026-11-09

## What I need decided

1. Prose, link, table and tree (A1–A5): hotty-surf's vocabulary, or
   registered from the website's code through the extension points
   (B1–B3), offered to hotty-surf once settled.
2. The goldmark converter: a sub-module `neuroplast.io/hotty-surf/surfmd`
   that registers `<markdown>`, or one per program.
3. The Bubble Tea adapter (B9): a sub-module `neuroplast.io/hotty-surf/surftea`,
   or in hotty-go's hottytea.

## Why I cannot decide it

Module boundaries and the vocabulary are what programs import and pin;
moving them later is a migration.

## Options and my lean

1. In hotty-surf. Both the website and margin need them, so they are
   general; B1–B3 are still built, for what only one program draws.
2. A sub-module, so the core takes no goldmark and stays TinyGo-clean,
   and two programs don't write the same walk.
3. A hotty-surf sub-module: hotty-go stays protocol-only and depends on
   nothing of surf's, and the adapter is mostly surfrun.
