---
name: memory-architecture
type: masterfile
status: decided
---

## Problem

The current memory system stores observations, decisions, and session summaries as markdown files in `docs/memory/`. This works well for git-native team sharing, but has growing gaps:

1. **No cross-project search.** Agents cannot ask "how did I solve X in project Y?" Each repo's memory is siloed. Personal notes already live in `~/.cx/memory.db` (SQLite), but project memory cannot be queried alongside them.
2. **No session or agent run tracking.** There is no structured record of what agents did, how long they ran, or what they produced. Debugging and retrospective analysis is manual.
3. **FTS5 is derived, not authoritative.** The `.cx/.index.db` is a rebuild-on-demand cache — it's correct when fresh, but there's no single queryable store that includes both personal notes and project memory.
4. **Scalability ceiling.** Parsing hundreds of markdown files on every index rebuild is fine for small teams. For larger teams or long-lived projects, rebuild latency and search quality will degrade.
5. **Vector readiness is absent.** There is nowhere to store embeddings today. Adding vector search later would require a migration with no migration path defined.

The developer wants to move to an SQLite-based memory system that spans projects, supports agent run tracking, and is designed from the start to be vector-ready.

---

## Context

### What exists today

**File-based project memory** (`docs/memory/`):
- `observations/` — markdown files, committed to git, team-visible
- `decisions/` — markdown files, committed to git, team-visible
- `sessions/` — markdown files, committed to git, team-visible

Each file has YAML frontmatter (type, author, tags, change, etc.) and a markdown body. The binary parses these on demand and rebuilds `.cx/.index.db` (FTS5 cache) when files change.

**Personal notes** (`~/.cx/memory.db`):
- Already SQLite with FTS5 virtual table
- The one place where SQLite is already the source of truth, not a cache
- Supports upsert via `topic_key`, deletion — personal notes can evolve, unlike project memory

**Agent infrastructure** (`~/.cx/`):
- `GlobalCXDir()` in `internal/project/registry.go` already creates and manages `~/.cx/`
- `projects.json` tracks registered project paths — this is the registry for cross-project discovery
- `.cx/.index.db` inside each project is gitignored (all of `.cx/` is gitignored via `.cx/.gitignore`)

**Search** (`cx search`):
- Unified FTS5 search over all `docs/` content via `.cx/.index.db`
- Personal notes included via `--personal` flag
- No cross-project capability

**Memory spec** (`docs/specs/memory/spec.md`):
- Already specifies `docs/memories/` (note: code uses `docs/memory/`, spec says `docs/memories/` — there is a discrepancy to resolve separately; see Resolved section)
- Four entity types: observation, decision, session, personal note

### Constraints

- The cx CLI is a Go binary; no server process runs between commands
- Agents never touch the DB directly — all DB access goes through `cx` CLI commands
- Team sharing of project memory must remain git-native (markdown in `docs/`)
- The existing `personal_notes` SQLite schema in `~/.cx/memory.db` is a reference point
- `github.com/mattn/go-sqlite3` is the standard Go SQLite driver (requires cgo) — `modernc.org/sqlite` is a pure-Go alternative that avoids cgo
- Go 1.25.0 is in use
- **Agent spawning model**: The Master agent cannot spawn agents through the cx binary. It uses the AI tool's `Agent` tool within the chat session and injects prompts. When the agent finishes, it returns a summary to the Master. The cx binary has no visibility into agent spawning — agent run tracking is purely prompt-driven and explicit.

### Resolved decisions

- **DB topology**: Per-Project DB + Global Index. Each project has `.cx/memory.db`; a global `~/.cx/index.db` enables cross-project discovery and federated queries.
- **Team sync**: T1 (git-tracked export) is the primary mechanism. T2 (git hook automation) is an opt-in overlay added via `cx init --hooks`.
- **Search**: FTS5 now, with an embedding column placeholder in the schema for future vector search.
- **Memory types**: All four — observations, decisions, session summaries, agent interactions.
- **Cross-project queries**: Yes, via federated queries across per-project DBs using the global index.
- **projects.json → index.db**: Phase 1. `projects.json` is replaced by `~/.cx/index.db` immediately in Phase 1. No backward-compat read of `projects.json` is needed for Phase 2.
- **Export format**: Clean break. `cx memory push` uses a new format optimized for DB round-tripping. No obligation to byte-match existing committed markdown files.
- **Session and agent_runs scope**: Per-project only. Both tables live in `.cx/memory.db`, NOT in `~/.cx/index.db`. Cross-project session history is not a goal.
- **Conflict resolution**: Not automatic. `cx doctor` detects conflicts (same memory ID, different content between local DB and `docs/memory/` files). The developer resolves manually, potentially through team communication. `cx memory pull` warns on conflicts but does NOT overwrite — it imports only non-conflicting memories and lists conflicts for the developer.
- **Push visibility defaults**: Auto-detect by type. Decisions default to `project` visibility (shared). Sessions and agent_runs default to `personal`. Developer can override per-memory with `--visibility`.
- **Canonical path (`docs/memory/` vs `docs/memories/`)**: DEFERRED. The discrepancy between the code and the spec will be resolved as a separate task. This design uses `docs/memory/` to match current code.

---

## Direction

The plan introduces SQLite as a structured companion to the existing markdown files, not a replacement. Markdown files remain the source of truth for team-shared project memory (git handles sync). SQLite adds:

1. **A global index DB** at `~/.cx/index.db` that spans all registered projects (replaces `projects.json` in Phase 1)
2. **Per-project local DB** at `.cx/memory.db` that mirrors parsed project memory and enables fast local queries
3. **Agent run tracking** (sessions + agent_runs tables) so every agent invocation can be explicitly recorded
4. **FTS5 now, vector-ready later** — embedding column is planned but not implemented in v1

### DB Topology: Per-Project DB + Global Index

Each project has `.cx/memory.db` (local, gitignored). A lightweight `~/.cx/index.db` tracks known projects and their last-indexed timestamps, enabling cross-project discovery. `projects.json` is replaced by `index.db` in Phase 1.

```
~/.cx/
├── memory.db         ← personal notes (existing, unchanged)
└── index.db          ← global project registry + cross-project query layer (replaces projects.json)

<project>/.cx/
├── .gitignore        ← "* " — everything inside is gitignored
├── cx.yaml           ← project config
├── memory.db         ← parsed project memory (structured local cache); also holds sessions + agent_runs
└── .index.db         ← existing: FTS5 doc index (rename → fts.db or keep)
```

**Cross-project queries**: The CLI opens `~/.cx/index.db` to get known project paths, then opens each project's `.cx/memory.db` to federate the query. Results are merged and ranked in-process. For N projects this is N sequential SQLite opens — fast enough for dozens of projects.

Data travels with the project (`.cx/memory.db` is local but mirrors the committed markdown). Deleting a project means deleting its `.cx/` directory — clean. No shared write contention between different projects.

---

## Schema Design

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
-- Populated by cx index rebuild --global; stale until rebuilt.
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

-- Schema versioning
CREATE TABLE IF NOT EXISTS schema_migrations (
    version     INTEGER PRIMARY KEY,
    applied_at  TEXT DEFAULT (datetime('now')),
    description TEXT
);
```

Note: `sessions` and `agent_runs` tables do NOT live in `~/.cx/index.db`. They are per-project and live in `.cx/memory.db`.

---

### `<project>/.cx/memory.db` — Per-Project DB

```sql
-- Mirrors parsed docs/memory/ markdown files. Rebuilt from markdown on demand.
CREATE TABLE IF NOT EXISTS memories (
    id          TEXT PRIMARY KEY,         -- filename slug (the entity ID)
    entity_type TEXT NOT NULL             -- observation | decision | session | agent_interaction
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
    shared_at   TEXT,                     -- ISO8601 datetime of last push for this row (NULL = never pushed)
    created_at  TEXT NOT NULL,
    updated_at  TEXT DEFAULT (datetime('now')),
    archived_at TEXT,                     -- non-null if soft-archived (excluded from context)

    -- Vector-ready: uncomment and add sqlite-vec in Phase 3
    -- embedding  BLOB,                   -- sqlite-vec float32[] for semantic search
    UNIQUE(id)
);

-- FTS5 over project memory for fast local search
CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
    title, content, tags, entity_type,
    content=memories, content_rowid=rowid
);

-- Agent run tracking: one row per agent invocation (per-project only)
-- Sessions are explicitly logged by the Master after session end.
CREATE TABLE IF NOT EXISTS sessions (
    id          TEXT PRIMARY KEY,         -- UUID or timestamp-slug
    mode        TEXT NOT NULL             -- build | continue | plan
                CHECK(mode IN ('build', 'continue', 'plan')),
    change_name TEXT,                     -- active change during session
    goal        TEXT,                     -- session goal (from cx memory session --goal)
    started_at  TEXT NOT NULL DEFAULT (datetime('now')),
    ended_at    TEXT,
    summary     TEXT                      -- session summary body
);

-- One row per agent dispatch within a session.
-- Populated via explicit cx agent-run log calls — NOT automatically by the binary.
CREATE TABLE IF NOT EXISTS agent_runs (
    id              TEXT PRIMARY KEY,     -- UUID
    session_id      TEXT REFERENCES sessions(id) ON DELETE SET NULL,
    agent_type      TEXT NOT NULL,        -- scout | planner | reviewer | go-expert | etc.
    prompt_summary  TEXT,                 -- first 200 chars of the prompt (for debugging)
    result_status   TEXT                  -- success | blocked | needs-input
                    CHECK(result_status IN ('success', 'blocked', 'needs-input', NULL)),
    result_summary  TEXT,                 -- agent's summary field
    artifacts       TEXT,                 -- JSON array of file paths produced
    duration_ms     INTEGER,
    created_at      TEXT DEFAULT (datetime('now'))
);

-- Explicit links between memory entities (for "related to" or "caused by" relationships)
CREATE TABLE IF NOT EXISTS memory_links (
    from_id         TEXT NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    to_id           TEXT NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    relation_type   TEXT NOT NULL         -- related-to | caused-by | resolved-by | see-also
                    CHECK(relation_type IN ('related-to', 'caused-by', 'resolved-by', 'see-also')),
    created_at      TEXT DEFAULT (datetime('now')),
    PRIMARY KEY (from_id, to_id, relation_type)
);

-- Schema versioning table for embedded migrations
CREATE TABLE IF NOT EXISTS schema_migrations (
    version     INTEGER PRIMARY KEY,
    applied_at  TEXT DEFAULT (datetime('now')),
    description TEXT
);
```

---

### `~/.cx/memory.db` — Personal Notes (Existing, Extended)

The existing `personal_notes` table remains unchanged. A new `schema_migrations` table is added on first open:

```sql
-- existing (unchanged):
CREATE TABLE personal_notes (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    type        TEXT NOT NULL CHECK(type IN ('pattern', 'preference', 'tool_tip', 'reminder')),
    title       TEXT NOT NULL,
    content     TEXT NOT NULL,
    topic_key   TEXT UNIQUE,
    projects    TEXT,                  -- JSON array
    tags        TEXT,                  -- comma-separated
    created_at  TEXT DEFAULT (datetime('now')),
    updated_at  TEXT DEFAULT (datetime('now'))
);

CREATE VIRTUAL TABLE personal_notes_fts USING fts5(
    title, content, tags,
    content=personal_notes, content_rowid=id
);

-- NEW: add to ~/.cx/memory.db
CREATE TABLE IF NOT EXISTS schema_migrations (
    version     INTEGER PRIMARY KEY,
    applied_at  TEXT DEFAULT (datetime('now')),
    description TEXT
);
```

---

## Memory Visibility and Team Sync

### Memory Visibility Tiers

Each row in the `memories` table carries a `visibility` field. Defaults are inferred by memory type; the developer can override per-memory with `--visibility`.

| Memory type | Default visibility | Rationale |
|-------------|-------------------|-----------|
| `decision` | `project` | Architectural decisions are team-relevant |
| `observation` | `project` | Discoveries that inform future work are team-relevant |
| `session` | `personal` | Session summaries are personal state, not team artifacts |
| `agent_run` | `personal` | Agent run logs are personal debug data |

Both defaults can be overridden: `cx memory save --visibility personal` or `cx memory decide --visibility project`.

| Tier | Behavior |
|------|----------|
| `personal` | Stays in local `.cx/memory.db`; never exported or pushed |
| `project` | Exported to `docs/memory/` and shared via git |

### Primary: Git-tracked export (T1)

Project-visible memories are exported as structured markdown files in `docs/memory/`. These files are committed to git and pulled by teammates.

```
docs/memory/
├── observations/
│   └── <slugified-title>.md     ← one file per memory
├── decisions/
│   └── <slugified-title>.md
└── sessions/                    ← personal by default; not pushed here
```

Each file uses YAML frontmatter carrying the full DB fields, with the markdown body as content. The format is optimized for DB round-tripping — it is not byte-compatible with any prior format:

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

**Commands:**

```bash
cx memory push               # export all visibility=project rows not yet pushed (shared_at IS NULL or stale)
cx memory pull               # import docs/memory/ markdown into local .cx/memory.db (warns on conflicts, does NOT overwrite them)
cx memory push --all         # re-export all project-visible memories (idempotent, overwrites)
```

`cx memory push` sets `shared_at` on each exported row. `cx memory pull` imports only non-conflicting memories. A conflict is when the same `id` exists in both the local DB and `docs/memory/` with different `content`. Conflicting memories are listed and skipped — the developer must resolve them manually (and possibly coordinate with teammates). Session summaries and agent runs with `visibility=personal` are never written to `docs/memory/` by push.

The key design principle: **the markdown files in `docs/memory/` are the shared transport layer; the local `.cx/memory.db` is the source of truth for querying.** This preserves the existing file-based team convention while adding the structured DB layer on top. The `cx memory push / pull` commands make the sharing workflow explicit where previously it was implicit ("commit your `docs/memory/` files").

### Opt-in: Git hook automation (T2)

T2 is an opt-in overlay on top of T1 that requires no schema changes. The format is identical to T1; automation is wired via git hooks.

```bash
cx init --hooks    # install .git/hooks/pre-commit and post-merge wrappers
```

`cx memory push` runs as a pre-commit hook; `cx memory pull` runs after `git merge` or `git pull`. The export format minimizes conflicts: one file per memory record, append-only creation. Overwriting a file (e.g., updating `shared_at`) can only conflict if two teammates push an update to the same memory simultaneously.

T2 is documented and implemented as a feature flag. Teams that want seamless sync enable it; others rely on T1 manual push/pull.

---

## CLI Interface Design

All memory commands go through `cx`; agents never open the DB directly.

### Saving

```bash
cx memory save --type observation --title "..." --content "..."
    [--change <name>] [--files <p1,p2>] [--specs <s1,s2>]
    [--tags <t1,t2>] [--deprecates <slug>] [--source <agent-name>]
    [--visibility <personal|project>]

cx memory decide --title "..."
    --context "..." --outcome "..." --alternatives "..." --rationale "..." --tradeoffs "..."
    [--change <name>] [--specs <s1,s2>] [--tags <t1,t2>] [--deprecates <slug>]
    [--visibility <personal|project>]

cx memory session --goal "..."
    --accomplished "..." --discoveries "..." [--blockers "..."] --files "..." --next "..."
    [--change <name>]
    # defaults to visibility=personal; override with --visibility project

cx memory note --type preference --title "..." --content "..."
    [--topic-key <key>] [--projects <p1,p2>] [--tags <t1,t2>]
```

### Querying

```bash
cx memory search "query"                   # FTS5 in current project
cx memory search "query" --all-projects    # federate across all registered projects
cx memory search "query" --type decision   # filter by entity type
cx memory search "query" --author angel    # filter by author
cx memory list --type observation          # list without FTS (recent first)
cx memory list --all-projects --type decision  # cross-project list
```

### Linking

```bash
cx memory link <id1> <id2> --relation related-to
cx memory link <id1> <id2> --relation caused-by
```

### Portability

```bash
cx memory push               # export project-visible memories to docs/memory/
cx memory pull               # import docs/memory/ markdown into .cx/memory.db (warns on conflicts, skips them)
cx memory push --all         # re-export all project-visible memories (idempotent)
```

### Agent run tracking (prompt-driven, not automatic)

Agent run logging is explicit. The cx binary has no visibility into agent spawning. The Master calls `cx agent-run log` after each agent dispatch returns. Subagents can also call `cx agent-run log` to record their own work before returning.

```bash
# Log a completed agent run (single command; called after the agent returns)
cx agent-run log --type <agent_type> --session <session_id>
    --status <success|blocked|needs-input>
    --summary "..."
    [--artifacts <p1,p2>]
    [--duration-ms <ms>]
    [--prompt-summary "first 200 chars of the prompt"]

# List agent runs for a session
cx agent-run list [--session <session_id>]
```

There is no `cx agent-run start` or `cx agent-run end` — the run is recorded atomically after it completes. The `duration_ms` field is populated by the caller (Master or subagent) based on wall time it measured externally.

---

## Migration Strategy

### Phase 1: DB alongside markdown (v1) — includes global index

1. `cx init` creates `.cx/memory.db` with the new schema (migrations v1)
2. `cx init` also creates or updates `~/.cx/index.db`, replacing `projects.json` as the project registry
3. `~/.cx/index.db` is bootstrapped from `projects.json` on first creation (one-time import)
4. `cx index rebuild` also populates `.cx/memory.db` from markdown files
5. All `cx memory save/decide/session` commands write to both markdown (for git) and `.cx/memory.db`
6. `cx memory search` queries `.cx/memory.db` instead of `.cx/.index.db` for memory entities; `.index.db` remains for spec/change/architecture docs
7. Existing markdown files are migrated on first `cx index rebuild` after upgrade
8. `cx memory search --all-projects` opens `~/.cx/index.db`, federates to project DBs

### Phase 2: Federation refinement (v2)

1. Materialized `memory_index` table in `~/.cx/index.db` is populated to offload summary-level cross-project queries (avoids opening every project DB for lightweight searches)
2. `cx index rebuild --global` refreshes the materialized index

### Phase 3: Vector search (v3, future)

1. Integrate `sqlite-vec` extension (Go wrapper available)
2. Uncomment `embedding BLOB` column in `memories` (already noted in schema comment)
3. `cx memory search --semantic "..."` uses cosine similarity on embeddings
4. Embeddings generated by a configurable provider (local model or API)
5. Switch SQLite driver from `modernc.org/sqlite` to `go-sqlite3` (cgo) to support `sqlite-vec` native extensions

### Schema versioning approach

Migrations are embedded in the Go binary as ordered functions keyed by integer version:

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

`Migrate()` is called on every DB open. Idempotent. No external migration tool needed.

---

## Files to Modify

**New files:**
- `internal/memory/db.go` — DB open, migrate, connection management
- `internal/memory/migrations.go` — embedded migration functions
- `internal/memory/entities.go` — CRUD for memories, sessions, agent_runs, memory_links
- `internal/memory/fts.go` — FTS5 query helpers, federation across project DBs
- `internal/memory/export.go` — markdown export from DB (implements `cx memory push`)
- `internal/memory/import.go` — markdown parse + DB ingest (implements `cx memory pull`; replaces/extends current index rebuild)
- `cmd/memory.go` — cobra command subtree: save, decide, session, note, search, list, link, push, pull
- `cmd/agent_run.go` — agent-run log and list commands

**Modified files:**
- `internal/project/scaffold.go` — `ScaffoldCXCache()` also creates `.cx/memory.db` with schema
- `internal/project/registry.go` — `GlobalCXDir()` also bootstraps `~/.cx/index.db`; replace `projects.json` reads/writes with `index.db` queries
- `internal/doctor/checks.go` — `CheckMemoryHealth()` validates DB exists and schema version is current; `CheckIndexHealth()` updated; add memory sync conflict check (see Risks)
- `go.mod` — add `modernc.org/sqlite` (pure-Go, no cgo) for Phase 1 and 2
- `cmd/root.go` — register new memory and agent-run commands

**Possibly absorbed:**
- `.cx/.index.db` naming — consider renaming to `.cx/fts.db` to distinguish from `.cx/memory.db`

---

## Risks

**1. SQLite driver choice (cgo vs pure-Go)**
- `modernc.org/sqlite` is used in Phase 1 and 2 — pure-Go, no C compiler required, simpler build
- Phase 3 vector search via `sqlite-vec` requires native extensions and will need a switch to `go-sqlite3` (cgo)
- Mitigation: Design the DB open function behind an interface so the driver can be swapped without touching entity or query code

**2. DB and markdown divergence**
- If an agent writes markdown directly (bypassing `cx memory save`) the DB is stale
- Mitigation: `cx index rebuild` always re-ingests from markdown; lazy rebuild on mtime check (existing pattern) applies to `memory.db` too

**3. Concurrent writes**
- Multiple terminal sessions in the same project could write to `.cx/memory.db` simultaneously
- Mitigation: Open DB with `_journal_mode=WAL` (SQLite WAL mode supports concurrent readers, serialized writers); this is standard practice

**4. docs/memory/ path discrepancy**
- Current code uses `docs/memory/`; the memory spec says `docs/memories/`
- This is DEFERRED as a separate task. This implementation uses `docs/memory/` to match current code.

**5. Federation performance**
- Cross-project search opens N SQLite connections serially
- Mitigation: For reasonable N (< 50 projects), this is <500ms. For larger N, the materialized `memory_index` table in `~/.cx/index.db` (Phase 2) offloads summary-level queries without opening individual project DBs

**6. Global index.db bootstrapping**
- `~/.cx/index.db` needs to be populated from existing `projects.json` on first run
- Mitigation: On first creation of `index.db`, one-time import all paths from `projects.json` into the projects table

**7. Memory sync conflicts**
- Two teammates can push different versions of the same memory (same `id`, different `content`)
- `cx memory pull` does NOT auto-resolve conflicts. It warns the developer and skips conflicting memories, importing only non-conflicting ones. The developer must decide which version to keep, potentially requiring team communication.
- `cx doctor` includes a check: "memory sync conflict — local and shared versions of memory X differ" to surface conflicts proactively.

---

## Agent Memory Integration

This section defines exactly where and how every agent type and workflow mode should interact with memory — what to read, what to write, and when.

### 1. Memory Read/Write Contract Per Agent

**Master Agent**:
- READ: Never reads memory directly. Dispatches Primer for all context loading.
- WRITE: Saves session summaries at session end (`cx memory session`). Records decisions made during requirements gathering (`cx memory decide`). Persists Scout discoveries that Scout itself cannot save (Scout is read-only).
- AGENT RUN LOGGING: Calls `cx agent-run log` AFTER each Agent dispatch returns. Subagents may also call `cx agent-run log` before returning. There is no automatic tracking — both the Master and subagents must call this explicitly.

**Primer (cx-primer)**:
- READ: Calls `cx memory search` or `cx context` to load relevant memories based on session mode.
  - CONTINUE mode: loads last session summary (critical for pickup), change-scoped observations and decisions.
  - BUILD mode: loads recent observations, active decisions.
  - PLAN mode: minimal — project overview and personal preferences only (intentionally clean-slate).
- WRITE: Never writes memory. Primer is read-only and disposable.

**Scout (cx-scout)**:
- READ: None. Scout explores code, not memory.
- WRITE: Returns discoveries to Master. Scout is read-only — it cannot call `cx memory save`. The Master must save significant Scout findings as observations after receiving Scout's response.

**Planner (cx-planner)**:
- READ: Receives primed context from the Master's dispatch prompt. Does not query memory independently.
- WRITE: Saves architectural decisions made during design with `cx memory decide --change <name>`. Saves non-obvious constraints discovered during planning with `cx memory save --type observation`.

**Reviewer (cx-reviewer)**:
- READ: Loads change-scoped memories before reviewing with `cx memory search --change <name>` to verify that prior constraints and decisions were respected.
- WRITE: Returns findings to Master. Reviewer is read-only — the Master saves review lessons as observations when they represent recurring patterns worth remembering.

**Executor agents**:
- READ: Receive primed context and task context in the dispatch prompt.
- WRITE: Save per-task discoveries using `cx memory save --type observation --change <name>`:
  - Bugs found and fixed
  - Undocumented constraints encountered
  - Reusable patterns applied

---

### 2. Workflow Memory Touchpoints

**BUILD Mode**:

```
Step 1: Session Start
  → Master dispatches Primer (currently skipped in cx-build — THIS MUST BE ADDED)
  → Primer loads: recent observations, active decisions, personal notes

Step 2: Requirements Gathering
  → WRITE: Master saves significant constraints and choices as decisions

Steps 3–5: Planning and Design
  → WRITE: Planner saves architectural decisions with --change <name>

Step 6: Implementation (per task)
  → WRITE: Executor saves per-task discoveries as observations with --change <name>
  → EXPLICIT LOG: Master calls cx agent-run log after each agent dispatch returns

Step 7: Review
  → READ: Reviewer loads change-scoped memories before reviewing
  → WRITE: Master saves review lessons if recurring patterns found

Step 8: Session End
  → WRITE: Master saves session summary with all required fields (see Section 3)
```

**CONTINUE Mode**:

```
Step 1: Context Recovery
  → READ: Primer loads last session summary (critical bridge), change-scoped observations and decisions
  → This is the most memory-intensive step — session recovery depends entirely on the last session summary

Step 2: Assess and Implement
  → Same as BUILD steps 6–7

Step 3: Session End
  → WRITE: Session summary with populated next_steps (this IS the bridge to the next CONTINUE session)
```

**PLAN Mode**:

```
Step 1: Gather Idea
  → READ: Minimal — project overview only (intentionally clean-slate for creative work)

Steps 2–3: Ideate and Iterate
  → No memory reads or writes — the masterfile is the persistence mechanism during this phase

Step 4: Transition to BUILD
  → WRITE: Session summary capturing what was planned and the decomposed change name
```

---

### 3. Required Session Summary Fields

The `cx memory session` command must capture all mandatory fields:

| Field | Required | Purpose |
|-------|----------|---------|
| goal | Yes | What the session aimed to accomplish |
| accomplished | Yes | What was actually done |
| change_id | If applicable | Links session to a specific change |
| discoveries | No | Key observations worth noting |
| blockers | If any | Where progress stopped and why |
| next_steps | Yes | The exact pickup point for the next CONTINUE session |
| files_touched | No | Files modified during the session |

The `next_steps` field is the most critical for CONTINUE mode. Without it, the next Primer cannot load meaningful recovery context.

---

### 4. Agent Run Logging — Prompt-Driven Model

The `agent_runs` table is populated explicitly by the Master (or by subagents themselves before returning). The cx binary has no visibility into agent spawning and does NOT automatically populate this table.

**Pattern for the Master:**

```bash
# After the Agent tool call returns — log the completed run
cx agent-run log \
    --type scout \
    --session <session_id> \
    --status success \
    --summary "Explored internal/memory package structure, found no existing DB layer" \
    --artifacts "" \
    --duration-ms 4500
```

**Pattern for subagents (before returning to Master):**

```bash
# A subagent logs its own work before returning its summary
cx agent-run log \
    --type go-expert \
    --session <session_id> \
    --status success \
    --summary "Implemented internal/memory/db.go with WAL mode and migration runner" \
    --artifacts "internal/memory/db.go,internal/memory/migrations.go"
```

Both patterns write to `.cx/memory.db` in the current project. The `session_id` is passed in the dispatch prompt by the Master.

Uses of the agent_runs table:
- Debugging failed or blocked sessions
- Generating accurate session summaries (what agents ran, what they produced)
- Understanding session cost and duration
- Auditing what happened in a given change's history

---

### 5. CLI Commands for Agent Integration

Complete reference of commands used in the agent/workflow integration layer:

```bash
# Save memory (agents and Master)
cx memory save --type observation --title "..." --content "..." [--change <name>] [--source <agent>]
cx memory decide --title "..." --context "..." --outcome "..." [--change <name>]
cx memory session --goal "..." --accomplished "..." --next "..." [--change <name>]

# Query memory (Primer and Reviewer)
cx memory search "query" [--type <type>] [--change <name>] [--all-projects]
cx memory list [--type <type>] [--recent <duration>] [--change <name>]

# Visibility / team sync
cx memory push               # export project-visible memories to docs/memory/
cx memory pull               # import docs/memory/ into local .cx/memory.db (warns on conflicts, skips them)

# Agent run tracking (Master calls after dispatch; subagents call before returning)
cx agent-run log --type <agent_type> --session <session_id> --status <status> --summary "..."
    [--artifacts <p1,p2>] [--duration-ms <ms>] [--prompt-summary "..."]
cx agent-run list [--session <session_id>]

# Context loading shortcut (used by Primer)
cx context --mode <build|continue|plan> [--change <name>]
cx context --load <resource_type> <slug>
```

---

### 6. Skill File Changes Required

The following skill files and subagent templates need updates to implement this contract:

**Skill files** (`skills/`):

| File | Change needed |
|------|---------------|
| `cx-build/SKILL.md` | Add Primer dispatch at Step 1. Add decision saves at requirements step. Add mandatory session summary fields at session end. Add `cx agent-run log` call after each agent dispatch. |
| `cx-continue/SKILL.md` | Specify mandatory session summary fields. Add review memory loading step. Clarify that `next_steps` in the previous session summary drives recovery. |
| `cx-plan/SKILL.md` | Add session summary at transition to BUILD step. |
| `cx-review/SKILL.md` | Add memory loading step: load change-scoped observations and decisions before reviewing. |
| `cx-scout/SKILL.md` | Clarify that Scout is read-only and that the Master must save significant Scout findings. |
| `cx-prime/SKILL.md` | Update to use `cx context` and `cx memory search` commands against the DB instead of reading files directly. |
| `cx-memory/SKILL.md` | Rewrite to document DB-backed commands instead of file-based memory. |
| `cx-supervise/SKILL.md` | Add agent-run logging instructions for multi-agent coordination. |

**Subagent templates** (embedded agent instruction templates):

| Template | Change needed |
|----------|---------------|
| `cx-primer.md` | Add `cx context --mode <mode>` and `cx memory search` instructions. |
| `cx-planner.md` | Add `cx memory decide --change <name>` instruction for design decisions. |
| `cx-reviewer.md` | Add `cx memory search --change <name>` instruction before reviewing. |
| `cx-scout.md` | Clarify read-only status and that discoveries should be returned to Master for memory saving. |

---

## Open Questions

6. **docs/memory/ rebuild trigger.** Should `.cx/memory.db` store its own `last_indexed_at` per-directory, or reuse the existing mechanism in `.index.db`? This is a low-stakes implementation detail but should be decided before writing `internal/memory/import.go`.

---

## Testing

**Unit tests:**
- `internal/memory/migrations_test.go` — apply migrations on a fresh in-memory SQLite DB; verify schema version
- `internal/memory/entities_test.go` — CRUD round-trips for memories, sessions, agent_runs, memory_links
- `internal/memory/fts_test.go` — FTS5 queries return expected rows; deprecated rows excluded by default; `--include-deprecated` includes them
- `internal/memory/import_test.go` — parse fixture markdown files, verify DB rows match expected entity fields

**Integration tests:**
- `cx memory save` → verify markdown file written + DB row inserted
- `cx memory search "query"` → returns expected result from FTS5
- `cx memory search --all-projects "query"` → federates across two test project DBs
- `cx index rebuild` on a directory with existing markdown → populates memory.db correctly
- Deprecation: save entity A, save entity B with `--deprecates A`, verify A is excluded from default search
- Visibility: `cx memory push` exports only `visibility=project` rows; session rows remain local
- Conflict detection: `cx memory pull` when local DB and docs/memory/ differ on same ID → warns and skips the conflicting entry
- `cx doctor` detects memory sync conflict between local DB and docs/memory/

**Migration tests:**
- Start with v1 DB, run migration to v2 → schema_migrations has two rows, memory_links table exists
- Fresh `cx init` → `~/.cx/index.db` created with projects table populated from `projects.json`

**Doctor tests:**
- `cx doctor` warns when `.cx/memory.db` is missing
- `cx doctor` warns when schema version is stale (v1 < current)
- `cx doctor` warns when memory sync conflict detected (local DB and docs/memory/ disagree on a memory ID)

---

## References

- `docs/specs/memory/spec.md` — current memory entity spec (authoritative source)
- `docs/specs/search/spec.md` — unified search spec (FTS5 index schema, `cx search` interface)
- `internal/project/registry.go` — `GlobalCXDir()`, `projects.json` registry
- `internal/doctor/checks.go` — `CheckMemoryHealth()`, `CheckIndexHealth()`
- `internal/project/scaffold.go` — `.cx/` directory setup
- SQLite WAL mode: https://www.sqlite.org/wal.html
- FTS5: https://www.sqlite.org/fts5.html
- `modernc.org/sqlite`: https://pkg.go.dev/modernc.org/sqlite
- `sqlite-vec` (future vector search): https://github.com/asg017/sqlite-vec
