---
name: memory-architecture
type: proposal
---

## Problem

The current memory system stores observations, decisions, and session summaries as markdown files in `docs/memory/`. This git-native approach works for team sharing but has five growing gaps:

1. **No cross-project search.** Agents cannot query "how did I solve X in project Y?" Each repo's memory is siloed. Personal notes already live in `~/.cx/memory.db` (SQLite) as source of truth, but project memory cannot be queried alongside them.
2. **No structured agent run tracking.** There is no record of which agents ran, how long they ran, or what they produced. Debugging and retrospective analysis is manual.
3. **FTS5 index is a derived cache, not authoritative.** `.cx/.index.db` is rebuilt on demand — correct when fresh, stale otherwise. There is no single queryable store spanning personal notes and project memory.
4. **Scalability ceiling.** Parsing hundreds of markdown files on every index rebuild works for small teams. For larger teams or long-lived projects, rebuild latency and search quality will degrade.
5. **No vector readiness.** There is nowhere to store embeddings today. Adding semantic search later would require a migration with no defined path.

## Approach

Introduce SQLite as a structured companion to the existing markdown files — not a replacement. Markdown files remain the team-sharing transport layer (git handles sync). SQLite adds fast local querying, cross-project federation, and explicit agent run tracking.

The system adds:
- A **per-project local DB** at `.cx/memory.db` that mirrors parsed project memory and holds sessions and agent_runs tables
- A **global index DB** at `~/.cx/index.db` that tracks all registered cx projects (replaces `projects.json`) and enables cross-project discovery
- **FTS5 search** on both DBs, with an embedding column placeholder for future vector search
- **Explicit agent run tracking** via `cx agent-run log` calls from Master and subagents (prompt-driven, not automatic)
- **Team sync** via `cx memory push/pull` — export project-visible memories to `docs/memory/` markdown, import back on pull (warn on conflicts, skip them)
- **Visibility tiers** — `personal` (local only) vs `project` (exported to git), inferred by memory type with per-record override

All memory access goes through `cx` CLI commands. Agents never open the DB directly.

## Scope

**In scope:**
- New `internal/memory/` Go package: schema, migrations, CRUD, FTS5, export, import
- New CLI commands: `cx memory save/decide/session/note/search/list/link/push/pull` and `cx agent-run log/list`
- Global index DB at `~/.cx/index.db`, replacing `projects.json` as project registry (Phase 1)
- Per-project `.cx/memory.db` created by `cx init` and `cx index rebuild`
- Cross-project federated search via `--all-projects` flag (Phase 1 federation; Phase 2 materialized index)
- `cx doctor` check for memory sync conflicts (local DB vs `docs/memory/` divergence)
- Skill file updates for all agent types and workflow modes (cx-build, cx-continue, cx-plan, cx-review, cx-prime, cx-memory, cx-scout, cx-supervise)
- Subagent template updates (cx-primer.md, cx-planner.md, cx-reviewer.md, cx-scout.md)
- Schema versioning via embedded Go migration functions

**Out of scope:**
- Vector search (Phase 3, future — embedding column is placeholdered but not activated)
- Git hook automation (`cx init --hooks`, T2 tier) — documented but not implemented as part of this change
- Resolving canonical spec path discrepancy (`docs/memories/` in specs → `docs/memory/` in code): this change normalizes all paths to `docs/memory/` via delta specs; the canonical spec update happens at archive time
- `cx context` command (`--mode`, `--load` subcommands) — covered by the context-priming spec; this change adds the underlying memory DB layer that `cx context` will query

**Dependencies:** None. This is a new subsystem built alongside existing infrastructure.

## Affected Specs

- **memory** — major delta: new DB topology, new storage model for project memory, new visibility tiers, new CLI commands (push/pull/link), agent run tracking
- **orchestration** — delta: agent memory read/write contracts, session summary required fields, agent-run logging pattern
- **context-priming** — delta: Primer now queries DB via `cx memory search` instead of reading markdown files; `cx context --mode` queries the new memory DB
- **session-modes** — delta: each mode now has explicit memory touchpoints (what to read, what to write, when)
- **doctor** — delta: new health checks for `.cx/memory.db` existence, schema version currency, and memory sync conflicts
- **search** — minor delta: `cx memory search` now queries `.cx/memory.db` (FTS5) instead of `.cx/.index.db`; `--all-projects` federation added
