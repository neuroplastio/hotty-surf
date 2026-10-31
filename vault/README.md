# Vault

What an agent needs to move hotty-surf forward, and what the maintainer
needs to steer it. Start here, then read [`/AGENTS.md`](../AGENTS.md),
gov's NEIO-15 (the decision) and R-23 (the research behind it).

## Map

| Path | What it is |
| --- | --- |
| [`stories/README.md`](stories/README.md) | The stories: the yardstick every model is judged on |
| [`tasks/board.md`](tasks/board.md) | The live board: v0's milestones (NEIO-15), and R-23's record |
| [`feedback/`](feedback/) | **Human → agent.** Steers the work. Preempts the board. |
| [`questions/`](questions/) | **Agent → human.** Decisions the agent must not make. Blocks work. |
| [`knowledge/`](knowledge/) | What was learned: the survey, the candidates, the Blitz gaps, the CSS subset, the cells, the look against its references |
| [`journal/`](journal/) | One entry per leg: what changed, how to see it. |

## The loop

Read `feedback/`, then the board; take the first open item nothing blocks;
build it; check it (`make check`, and the screens rendered and looked at);
write a journal entry; tick the board. A leg that changes how something
looks stops there.
