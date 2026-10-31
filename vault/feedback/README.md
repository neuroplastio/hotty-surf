# Feedback inbox

**Human → agent.** Drop a markdown file here to steer hotty-surf: something that
looks wrong, a key that does the wrong thing, a priority change. The agent
reads this directory before every leg and drains it before touching the
board.

Name the file `YYYY-MM-DD-short-slug.md`. Free-form prose is fine. A
screenshot, the story's name and the rendition (cells, a hotty host, which
theme) help.

An item that has been addressed is deleted in the leg that addresses it. The
journal entry for that leg links back to it, and git keeps the exact words:

```bash
git log --diff-filter=D -- 'vault/feedback/*'
```
