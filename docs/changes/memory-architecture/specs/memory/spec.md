---
name: memory
type: delta-spec
area: memory
change: memory-architecture
---

## ADDED Requirements

### Per-Project SQLite DB (`.cx/memory.db`)

- `cx init` creates `.cx/memory.db` with the full schema via `ScaffoldCXCache()` + `Migrate()`
- `.cx/memory.db` is the primary queryable store for project memory — replaces file-scanning for search and context loading
- Schema includes: `memories`, `memories_fts`, `sessions`, `agent_runs`, `memory_links`, `schema_migrations` tables
- `cx index rebuild` populates `.cx/memory.db` from `docs/memory/` markdown files (full re-ingest, no conflict detection)
- DB opened with WAL mode (`PRAGMA journal_mode=WAL`) for concurrent reader safety

### Global Index DB (`~/.cx/index.db`)

- `cx init` creates or updates `~/.cx/index.db`, replacing `projects.json` as the project registry
- `~/.cx/index.db` schema: `projects`, `memory_index`, `memory_index_fts`, `schema_migrations` tables
- On first creation of `index.db`, all paths from `projects.json` are one-time imported into the `projects` table
- `projects.json` is no longer read or written after migration

### Agent Run Tracking

- New `agent_runs` table in `.cx/memory.db` — one row per agent dispatch within a session
- Populated explicitly via `cx agent-run log` — NOT automatically by the binary
- New `sessions` table in `.cx/memory.db` — one row per development session
- New CLI commands: `cx agent-run log --type <t> --session <id> --status <s> --summary "..."` and `cx agent-run list`

### Memory Visibility Tiers

- Every row in `memories` carries a `visibility` field: `personal` (local only) or `project` (exported to git)
- Default visibility by type: `observation` → `project`, `decision` → `project`, `session` → `personal`, `agent_run` → `personal`
- Developer overrides per-record with `--visibility personal|project`

### Team Sync Commands

- `cx memory push` — exports `visibility=project` rows with `shared_at IS NULL` to `docs/memory/{observations,decisions}/` as markdown; sets `shared_at` on exported rows
- `cx memory push --all` — re-exports all project-visible memories (idempotent)
- `cx memory pull` — ingests `docs/memory/` markdown into `.cx/memory.db`; imports non-conflicting rows; warns and skips conflicting rows (same `id`, different `content`)
- Session rows are never exported by `cx memory push` regardless of visibility field

### Cross-Project Federation

- `cx memory search --all-projects "query"` — opens `~/.cx/index.db` to get project paths, opens each project's `.cx/memory.db`, federates search, merges and ranks results with project attribution
- `cx memory list --all-projects` — cross-project listing without FTS

### New CLI Commands

- `cx memory link <id1> <id2> --relation <type>` — saves explicit link between memory entities; `relation_type` in (`related-to`, `caused-by`, `resolved-by`, `see-also`)
- `cx agent-run log` / `cx agent-run list` — agent run tracking commands

### Export Markdown Format

New format optimized for DB round-tripping (not byte-compatible with prior format):

```yaml
---
id: <slug>
entity_type: observation
visibility: project
author: angel
source: go-expert
change_id: memory-architecture
file_refs: ["internal/memory/db.go"]
spec_refs: ["memory"]
tags: sqlite, migration
created_at: 2026-03-20T14:00:00Z
shared_at: 2026-03-20T14:05:00Z
---
```

### Schema Versioning

- Embedded Go migrations keyed by integer version; `Migrate(db *sql.DB) error` called on every DB open
- Idempotent — no external migration tool needed
- `schema_migrations` table in each DB tracks applied versions

## MODIFIED Requirements

### Storage Model — `docs/memory/` path confirmed as canonical

- Previous spec used `docs/memories/` (plural) as the canonical path
- This change normalizes all references to `docs/memory/` (singular) to match the existing Go codebase
- All three subdirectories remain: `docs/memory/observations/`, `docs/memory/decisions/`, `docs/memory/sessions/`

### Memory Search — DB-backed FTS5

- Previous: `cx memory search` queried `.cx/.index.db` (the unified doc FTS5 cache)
- Modified: `cx memory search` now queries `.cx/memory.db` FTS5 (`memories_fts` virtual table); `.cx/.index.db` remains for spec/change/architecture doc search only
- Search results now exclude deprecated and archived rows by default; `--include-deprecated` includes them

### Context Priming Memory Input

- Previous: Primer loaded memory by reading markdown files in `docs/memory/` directly
- Modified: Primer loads memory via `cx memory search` and `cx memory list` commands against `.cx/memory.db`

### `cx memory save` / `cx memory decide` / `cx memory session` — Dual Write

- All three save commands write to BOTH `.cx/memory.db` AND `docs/memory/` markdown files (Phase 1 dual-write; markdown remains the team transport)
- `cx memory save` gains new flags: `--source <agent-name>`, `--visibility <personal|project>`

## REMOVED Requirements

### `docs/memories/DIRECTION.md` as Canonical Memory Policy File

- Previous spec required `docs/memories/DIRECTION.md` as a required project file read at save-time by agents
- This file is not part of the memory-architecture change scope; the path normalization to `docs/memory/` means this becomes `docs/memory/DIRECTION.md` if it exists
- The memory-architecture change does not add or remove the DIRECTION.md mechanism — it is out of scope for this change

### `cx memory search` as Alias for Full-Text Doc Search

- The old `cx memory search` queried the unified `.cx/.index.db` index covering all `docs/` content
- Modified behavior: `cx memory search` now queries only `.cx/memory.db` (memory entities only)
- Full `docs/` search remains available via `cx search` (unchanged)
