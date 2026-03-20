---
name: memory-architecture
type: design
---

## Architecture

### DB Topology

Two SQLite databases span the system:

```
~/.cx/
├── memory.db         ← personal notes (existing, unchanged schema; add schema_migrations table)
└── index.db          ← global project registry + cross-project query layer (NEW; replaces projects.json)

<project>/.cx/
├── .gitignore        ← "* " — everything inside is gitignored
├── cx.yaml           ← project config
├── memory.db         ← per-project: parsed project memory + sessions + agent_runs (NEW)
└── .index.db         ← existing FTS5 doc index (unchanged; consider rename to fts.db later)
```

**Cross-project queries**: The CLI opens `~/.cx/index.db` to get registered project paths, then opens each project's `.cx/memory.db` to federate the query. Results are merged and ranked in-process. For N projects this is N sequential SQLite opens — fast enough for dozens of projects. Phase 2 adds a materialized `memory_index` table in `~/.cx/index.db` to offload summary-level queries without opening individual project DBs.

**Data ownership**: Per-project memory travels with the project (`.cx/memory.db` mirrors committed markdown). Deleting a project means deleting its `.cx/` directory — clean, no shared write contention.

### Component Map

```
cmd/memory.go          ← cobra command tree: save, decide, session, note, search, list, link, push, pull
cmd/agent_run.go       ← cobra command tree: log, list

internal/memory/
├── db.go              ← DB open/close, WAL mode, migrate-on-open lifecycle
├── migrations.go      ← embedded ordered migration functions keyed by integer version
├── entities.go        ← CRUD: memories, sessions, agent_runs, memory_links
├── fts.go             ← FTS5 query helpers; federated search across project DBs
├── export.go          ← cx memory push: DB rows → docs/memory/ markdown files
└── import.go          ← cx memory pull + cx index rebuild: markdown → DB rows

internal/project/
├── registry.go        ← modified: GlobalCXDir() bootstraps ~/.cx/index.db; replaces projects.json reads/writes
└── scaffold.go        ← modified: ScaffoldCXCache() creates .cx/memory.db with schema

internal/doctor/
└── checks.go          ← modified: CheckMemoryHealth() validates DB + schema version; add sync conflict check
```

### Memory Lifecycle

```
Agent calls cx memory save
    → cmd/memory.go validates required fields
    → internal/memory/entities.go writes row to .cx/memory.db
    → (Phase 1) row also written to docs/memory/ markdown file
    → (Team sync) cx memory push exports visibility=project rows to docs/memory/
    → teammates git pull → cx memory pull ingests docs/memory/ into their .cx/memory.db
```

```
Primer loads context
    → cx memory search "query" [--type T] [--change C] [--all-projects]
    → FTS5 query on .cx/memory.db
    → (--all-projects) open ~/.cx/index.db, federate to each project's .cx/memory.db
    → ranked results returned to Primer
```

---

## Schema

### `~/.cx/index.db` — Global Index

```sql
-- Tracks all known cx projects. Replaces projects.json.
CREATE TABLE IF NOT EXISTS projects (
    id          TEXT PRIMARY KEY,         -- stable slug: git remote URL hash, or abs path hash if no remote
    name        TEXT NOT NULL,            -- project name (from docs/overview.md H1 or directory name)
    path        TEXT NOT NULL UNIQUE,     -- absolute filesystem path
    git_remote  TEXT,                     -- git remote URL if present
    last_synced TEXT,                     -- ISO8601 datetime of last memory.db rebuild
    created_at  TEXT DEFAULT (datetime('now')),
    updated_at  TEXT DEFAULT (datetime('now'))
);

-- Lightweight summary rows for cross-project search without opening every project DB.
-- Phase 2: populated by cx index rebuild --global; stale until rebuilt.
CREATE TABLE IF NOT EXISTS memory_index (
    rowid       INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id  TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    local_id    TEXT NOT NULL,            -- entity ID in the project's memory.db
    entity_type TEXT NOT NULL,            -- observation | decision | session | agent_interaction
    title       TEXT NOT NULL,
    tags        TEXT,                     -- comma-separated
    created_at  TEXT NOT NULL,
    deprecated  INTEGER DEFAULT 0
);

CREATE VIRTUAL TABLE IF NOT EXISTS memory_index_fts USING fts5(
    title, tags,
    content=memory_index, content_rowid=rowid
);

CREATE TABLE IF NOT EXISTS schema_migrations (
    version     INTEGER PRIMARY KEY,
    applied_at  TEXT DEFAULT (datetime('now')),
    description TEXT
);
```

### `<project>/.cx/memory.db` — Per-Project DB

```sql
-- Mirrors parsed docs/memory/ markdown files. Rebuilt from markdown on demand.
CREATE TABLE IF NOT EXISTS memories (
    id          TEXT PRIMARY KEY,         -- filename slug (the entity ID)
    entity_type TEXT NOT NULL
                CHECK(entity_type IN ('observation', 'decision', 'session', 'agent_interaction')),
    subtype     TEXT,                     -- bugfix | discovery | pattern | context (observations only)
    title       TEXT NOT NULL,
    content     TEXT NOT NULL,            -- full markdown body (after frontmatter)
    author      TEXT NOT NULL,
    source      TEXT,                     -- agent name or "user" — who created it
    change_id   TEXT,                     -- active change when this was saved
    file_refs   TEXT,                     -- JSON array of file paths
    spec_refs   TEXT,                     -- JSON array of spec area slugs
    tags        TEXT,                     -- comma-separated
    deprecates  TEXT,                     -- slug of entity this one replaces
    deprecated  INTEGER DEFAULT 0,        -- 1 if another entity deprecates this one
    status      TEXT,                     -- active | superseded | cancelled (decisions only)
    visibility  TEXT NOT NULL DEFAULT 'personal'
                CHECK(visibility IN ('personal', 'project')),
    shared_at   TEXT,                     -- ISO8601 datetime of last push (NULL = never pushed)
    created_at  TEXT NOT NULL,
    updated_at  TEXT DEFAULT (datetime('now')),
    archived_at TEXT,                     -- non-null if soft-archived (excluded from context)

    -- Vector-ready: uncomment and add sqlite-vec in Phase 3
    -- embedding  BLOB,                   -- sqlite-vec float32[] for semantic search
    UNIQUE(id)
);

CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
    title, content, tags, entity_type,
    content=memories, content_rowid=rowid
);

-- Agent run tracking: one row per agent invocation (per-project only)
CREATE TABLE IF NOT EXISTS sessions (
    id          TEXT PRIMARY KEY,         -- UUID or timestamp-slug
    mode        TEXT NOT NULL
                CHECK(mode IN ('build', 'continue', 'plan')),
    change_name TEXT,                     -- active change during session
    goal        TEXT,                     -- session goal
    started_at  TEXT NOT NULL DEFAULT (datetime('now')),
    ended_at    TEXT,
    summary     TEXT                      -- session summary body
);

-- One row per agent dispatch within a session. Populated via explicit cx agent-run log calls.
CREATE TABLE IF NOT EXISTS agent_runs (
    id              TEXT PRIMARY KEY,     -- UUID
    session_id      TEXT REFERENCES sessions(id) ON DELETE SET NULL,
    agent_type      TEXT NOT NULL,        -- scout | planner | reviewer | go-expert | etc.
    prompt_summary  TEXT,                 -- first 200 chars of the prompt
    result_status   TEXT
                    CHECK(result_status IN ('success', 'blocked', 'needs-input', NULL)),
    result_summary  TEXT,                 -- agent's summary field
    artifacts       TEXT,                 -- JSON array of file paths produced
    duration_ms     INTEGER,
    created_at      TEXT DEFAULT (datetime('now'))
);

-- Explicit links between memory entities
CREATE TABLE IF NOT EXISTS memory_links (
    from_id         TEXT NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    to_id           TEXT NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    relation_type   TEXT NOT NULL
                    CHECK(relation_type IN ('related-to', 'caused-by', 'resolved-by', 'see-also')),
    created_at      TEXT DEFAULT (datetime('now')),
    PRIMARY KEY (from_id, to_id, relation_type)
);

CREATE TABLE IF NOT EXISTS schema_migrations (
    version     INTEGER PRIMARY KEY,
    applied_at  TEXT DEFAULT (datetime('now')),
    description TEXT
);
```

### `~/.cx/memory.db` — Personal Notes (Existing, Extended)

The existing `personal_notes` and `personal_notes_fts` tables are unchanged. A new `schema_migrations` table is added on first open after upgrade:

```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
    version     INTEGER PRIMARY KEY,
    applied_at  TEXT DEFAULT (datetime('now')),
    description TEXT
);
```

---

## Technical Decisions

### SQLite driver: `modernc.org/sqlite` for Phases 1–2

Pure-Go, no C compiler required, simpler build. `go-sqlite3` (cgo) would be required if `sqlite-vec` native extensions are integrated in Phase 3 for vector search. The DB open function is written behind an interface so the driver can be swapped without touching entity or query code. Go 1.25.0 is in use.

### WAL mode for concurrent writes

All DBs opened with `PRAGMA journal_mode=WAL`. SQLite WAL mode supports concurrent readers with serialized writers — handles multiple terminal sessions in the same project without corruption.

### Migrate-on-open

`Migrate(db *sql.DB)` is called every time a DB is opened. Idempotent. No external migration tool needed. Migrations are embedded Go functions keyed by integer version:

```go
// internal/memory/migrations.go
var migrations = []Migration{
    {Version: 1, Description: "initial schema", Up: v1Schema},
    {Version: 2, Description: "add memory_links", Up: v2AddLinks},
}

func Migrate(db *sql.DB) error {
    // ensure schema_migrations table exists
    // find current version
    // apply unapplied migrations in order
}
```

### projects.json → index.db (Phase 1, clean break)

`~/.cx/index.db` replaces `projects.json` entirely in Phase 1. On first creation of `index.db`, a one-time import reads all paths from `projects.json` into the `projects` table. No backward-compat read of `projects.json` is needed after migration.

### Export format: clean break

`cx memory push` writes a new markdown format optimized for DB round-tripping. No obligation to byte-match any existing committed markdown files. The frontmatter carries all DB fields; the body is the full markdown content.

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

Full markdown body of the memory here.
```

### Visibility defaults: auto-detect by type

| Memory type | Default visibility | Rationale |
|-------------|-------------------|-----------|
| `decision` | `project` | Architectural decisions are team-relevant |
| `observation` | `project` | Discoveries that inform future work are team-relevant |
| `session` | `personal` | Session summaries are personal state |
| `agent_run` | `personal` | Agent run logs are personal debug data |

Override per-record with `--visibility`.

### Agent run tracking: prompt-driven only

The cx binary has no visibility into agent spawning. The `agent_runs` table is populated explicitly via `cx agent-run log` calls — by the Master after each Agent dispatch returns, or by subagents before returning their summary. There is no `cx agent-run start/end` pair — runs are recorded atomically after completion.

### Conflict resolution: warn-and-skip

`cx memory pull` does NOT auto-resolve conflicts. A conflict is when the same `id` exists in both local DB and `docs/memory/` with different `content`. Conflicting rows are listed and skipped — only non-conflicting memories are imported. `cx doctor` surfaces the same conflicts proactively. Developers resolve manually.

### Session summaries: per-project only

Both `sessions` and `agent_runs` tables live in `.cx/memory.db` only, not in `~/.cx/index.db`. Cross-project session history is not a goal.

---

## Implementation Notes

### Phase 1 — Core DB (primary deliverable)

1. `cx init` creates `.cx/memory.db` with schema via `ScaffoldCXCache()`
2. `cx init` creates or updates `~/.cx/index.db`, bootstrapping from `projects.json` on first run
3. `cx index rebuild` populates `.cx/memory.db` from docs/memory/ markdown files
4. All `cx memory save/decide/session` commands write to `.cx/memory.db` (and continue writing markdown for team sync)
5. `cx memory search` queries `.cx/memory.db` FTS5; `.index.db` remains for spec/change/doc search
6. `cx memory search --all-projects` opens `~/.cx/index.db`, federates to each project's `.cx/memory.db`

### Phase 2 — Materialized cross-project index

Populate `memory_index` table in `~/.cx/index.db` via `cx index rebuild --global` to offload summary-level cross-project queries without opening every project DB.

### Phase 3 — Vector search (future)

Uncomment `embedding BLOB` column in `memories`, integrate `sqlite-vec`, switch driver to `go-sqlite3` (cgo), add `cx memory search --semantic`.

### rebuild trigger open question

Before writing `internal/memory/import.go`, decide: does `.cx/memory.db` store its own `last_indexed_at` per-directory, or reuse the existing mtime mechanism in `.index.db`? Low-stakes but affects the import.go implementation. The recommended approach is to add a `last_indexed_at` column to the `projects` table in `index.db` and use it for both the global index and the per-project memory DB rebuild check.

### File path for exported memories

`cx memory push` writes to `docs/memory/` (singular). This change normalizes all paths to `docs/memory/` to match the existing Go codebase. The canonical specs reference `docs/memories/` (plural) — the delta specs for this change update all affected spec areas to use `docs/memory/` (singular).

### Agent integration contracts (summary)

| Agent | Memory reads | Memory writes |
|-------|-------------|---------------|
| Master | Never directly — dispatches Primer | `cx memory session` at end, `cx memory decide` for requirements decisions, `cx agent-run log` after each dispatch |
| Primer | `cx memory search`, `cx memory list` | Never |
| Scout | None | Never (read-only; Master saves Scout findings) |
| Planner | Receives primed context in prompt | `cx memory decide --change <name>`, `cx memory save --type observation` |
| Reviewer | `cx memory search --change <name>` | Never (Master saves review lessons) |
| Executor | Receives primed context in prompt | `cx memory save --type observation --change <name>` for per-task discoveries |
