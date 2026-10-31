# Questions

**Agent → human.** These are decisions the agent must not make on its own.
They are only for decisions that are expensive to unwind. The test is: if this
turns out wrong, is fixing it a rename or a migration? Examples are names that
agents write, catalog shapes, and anything the next few legs build on.

How something looks isn't a question. Build it, show it, and expect to be
corrected.

## Format

`Q-NNNN-short-slug.md`:

```markdown
# Q-0001 — The question

Status: open | answered
Blocks: the tickets that wait on it
Raised: YYYY-MM-DD

## What I need decided
## Why I cannot decide it
## Options
## My lean
```

To answer, edit the file: write the answer under `## Answer` and set
`Status: answered`. The next leg moves it into `knowledge/` and unblocks the
tickets.
