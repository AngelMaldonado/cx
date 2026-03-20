---
name: session-modes
type: delta-spec
area: session-modes
change: memory-architecture
---

## ADDED Requirements

### Explicit Memory Touchpoints Per Mode

Each session mode now has defined memory read and write checkpoints:

**BUILD mode touchpoints:**

| Step | Action | Command |
|------|--------|---------|
| Session start | Primer loads recent observations + active decisions | `cx memory list --type observation --recent 7d` |
| Requirements | Master saves significant constraints as decisions | `cx memory decide --change <name>` |
| Implementation | Executor saves per-task discoveries | `cx memory save --type observation --change <name>` |
| Each agent dispatch | Master logs agent run | `cx agent-run log --type <t> --session <id> ...` |
| Session end | Master saves session summary | `cx memory session --goal "..." --accomplished "..." --next "..."` |

**CONTINUE mode touchpoints:**

| Step | Action | Command |
|------|--------|---------|
| Session start | Primer loads last session (critical bridge) | `cx memory list --type session --change <name>` |
| Session start | Primer loads change-scoped memory | `cx memory search --change <name>` |
| Implementation | Same as BUILD implementation step | — |
| Session end | Master saves session summary with populated `next_steps` | `cx memory session --next "..."` |

**PLAN mode touchpoints:**

| Step | Action | Command |
|------|--------|---------|
| Session start | Minimal — personal notes only (clean-slate intent) | Personal notes via `cx context --mode plan` |
| Transition to BUILD | Master saves planning session summary | `cx memory session --goal "..." --accomplished "..."` |

### `next_steps` as Session Bridge

- In CONTINUE mode, the `next_steps` field of the previous session summary IS the session recovery mechanism
- Primer always includes `next_steps` from the last session in the primed context for CONTINUE mode
- Executors and Master are responsible for populating `next_steps` meaningfully at session end

## MODIFIED Requirements

### CONTINUE mode — Session Recovery Source

- Previous: Session recovery relied on loading the most recent session file from `docs/memories/sessions/` by mtime
- Modified: Session recovery queries `.cx/memory.db` via `cx memory list --type session --change <name>` sorted by `started_at` descending

### BUILD mode — Primer Dispatch

- Previous: BUILD mode in `cx-build` skill did not mandate a Primer dispatch at session start
- Modified: Primer is dispatched at the start of every BUILD session before requirements gathering; loads recent observations, active decisions, and personal notes

## REMOVED Requirements

None. All existing session mode classification rules and context budgets remain in force.
