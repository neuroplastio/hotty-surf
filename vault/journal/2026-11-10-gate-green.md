# 2026-11-10: the gate on main (leg a2ui/gate)

`make gate` said "111 to look at" and `make legible` failed on two
documents (s-text-clickable, both modes) on main.

## Cause

A commit, not the machine. Bisected with `--only look-agents`, one run at
a time under the lock: 0369667 passes ("as known"); 450e529 (surfui:
enlarge base surface text, three font sizes in surf.css) fails with the
same 6 cases as main. The Blitz binary (b7cbf17, sha256 742e3492…), its
fonts, Chromium 153.0.8010.12 and Playwright 1.63 are the baseline's own;
nothing the gate loads points into a hotty-blitz checkout. So the new
look is an intended change: the gate's cases moved on the known gaps
(`round.rs` hairlines, half-leading, thin borders), named in
`vault/knowledge/blitz-gaps.md` ("Larger base text").

## What changed

- `make gate-baseline`: 29 new, 20 worse, 62 off the list.
- `text/clickable` is 54 cells wide, not 50: its "#4182" was cut to
  "#41" in Roomy, a real overflow of the larger text.

## Worktrees and node_modules

The Makefile already installs `scripts/gate/node_modules` (`npm ci`) when
it is missing. A symlink to another checkout's directory is older than
package.json, so make reinstalls into the shared one: remove the symlink
and let make install, or touch the target.
