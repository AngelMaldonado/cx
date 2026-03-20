---
name: memory-architecture
type: tasks
---

## Tasks

### Phase 1 — Core DB

**TASK-01: Create `internal/memory/db.go` — DB open, WAL mode, migrate-on-open**

Files: `internal/memory/db.go`

Implement the DB lifecycle functions:
- `OpenProjectDB(projectPath string) (*sql.DB, error)` — opens `.cx/memory.db` for the given project root, creates file if absent, sets WAL mode, calls `Migrate()`
- `OpenGlobalIndexDB() (*sql.DB, error)` — opens `~/.cx/index.db`, creates file if absent, sets WAL mode, calls `Migrate()`
- `OpenPersonalDB() (*sql.DB, error)` — opens `~/.cx/memory.db`, adds `schema_migrations` table if absent (personal notes schema unchanged), calls `Migrate()`
- `Close(db *sql.DB) error`
- DB open function should be driver-agnostic (use `database/sql` interface; driver registered separately) to allow swapping from `modernc.org/sqlite` to `go-sqlite3` in Phase 3

Dependencies: none
Acceptance criteria: each Open function returns a usable `*sql.DB`; calling Open twice on the same file does not corrupt data; WAL mode pragma confirmed on each open

---

**TASK-02: Create `internal/memory/migrations.go` — embedded migrations**

Files: `internal/memory/migrations.go`

Implement:
- `Migration` struct: `{Version int, Description string, Up func(*sql.DB) error}`
- `var migrations []Migration` — ordered list starting at version 1
- `Migrate(db *sql.DB) error` — ensures `schema_migrations` table exists; finds current version; applies unapplied migrations in order; idempotent
- `v1Schema(db *sql.DB) error` — creates all tables for per-project DB: `memories`, `memories_fts`, `sessions`, `agent_runs`, `memory_links`, `schema_migrations`
- `v1IndexSchema(db *sql.DB) error` — creates all tables for `~/.cx/index.db`: `projects`, `memory_index`, `memory_index_fts`, `schema_migrations`

Use full `CREATE TABLE IF NOT EXISTS` and `CREATE VIRTUAL TABLE IF NOT EXISTS` statements as specified in design.md.

Dependencies: TASK-01
Acceptance criteria: fresh in-memory DB reaches current schema version after Migrate(); upgrading a v1 DB to v2 (add memory_links) leaves schema_migrations with two rows; Migrate() on already-current DB is a no-op

---

**TASK-03: Create `internal/memory/entities.go` — CRUD for all entity types**

Files: `internal/memory/entities.go`

Implement:
- `SaveMemory(db *sql.DB, m Memory) error` — insert into `memories`; auto-set `updated_at`; if `deprecates` field is set, mark the referenced entity `deprecated=1`; if both entities are decisions, also set `status='superseded'` on the old one
- `GetMemory(db *sql.DB, id string) (Memory, error)`
- `ListMemories(db *sql.DB, opts ListOpts) ([]Memory, error)` — opts: entity_type filter, change_id filter, recent duration, include_deprecated, limit; default excludes deprecated and archived rows
- `SaveSession(db *sql.DB, s Session) error`
- `GetLatestSession(db *sql.DB) (Session, error)` — most recent session by started_at
- `SaveAgentRun(db *sql.DB, run AgentRun) error`
- `ListAgentRuns(db *sql.DB, sessionID string) ([]AgentRun, error)`
- `SaveMemoryLink(db *sql.DB, link MemoryLink) error`
- Go structs: `Memory`, `Session`, `AgentRun`, `MemoryLink` matching the schema columns

Dependencies: TASK-01, TASK-02
Acceptance criteria: CRUD round-trips for all types; deprecation chain sets `deprecated=1` on referenced entity; `ListMemories` with default opts excludes deprecated rows; all required fields validated before insert (non-empty id, title, entity_type, author, created_at)

---

**TASK-04: Create `internal/memory/fts.go` — FTS5 search and cross-project federation**

Files: `internal/memory/fts.go`

Implement:
- `SearchMemories(db *sql.DB, query string, opts SearchOpts) ([]MemoryResult, error)` — FTS5 query on `memories_fts`; opts: entity_type, change_id, author, include_deprecated, limit; default excludes deprecated rows; returns ranked results
- `SearchAllProjects(globalDB *sql.DB, query string, opts SearchOpts) ([]ProjectMemoryResult, error)` — opens `~/.cx/index.db` to get project paths; opens each project's `.cx/memory.db`; federates `SearchMemories` across all; merges and ranks results; attaches project name to each result
- `MemoryResult` struct: `{Memory, Rank float64}`
- `ProjectMemoryResult` struct: `{Memory, ProjectName string, ProjectPath string, Rank float64}`

Note: `SearchAllProjects` opens N SQLite connections sequentially. For N < 50 this is acceptable. Phase 2 will add a materialized index to avoid this for summary-level queries.

Dependencies: TASK-03
Acceptance criteria: FTS5 returns expected rows for a known query; deprecated rows excluded by default; `--include-deprecated` includes them at bottom; cross-project search returns results from two test project DBs with correct project attribution

---

**TASK-05: Create `internal/memory/export.go` — `cx memory push` implementation**

Files: `internal/memory/export.go`

Implement:
- `Push(db *sql.DB, docsDir string, all bool) (PushResult, error)` — export memories with `visibility='project'` and `shared_at IS NULL` (or all if `all=true`) to `docs/memory/{observations,decisions}/` as markdown files; set `shared_at` on each exported row; create subdirectories as needed
- `PushResult` struct: `{Exported int, Skipped int, Files []string}`
- Exported markdown format: YAML frontmatter with all DB fields + markdown body (as specified in design.md); one file per memory; filename: `<id>.md`
- Session rows are never exported (regardless of visibility field)
- `all=true` re-exports all project-visible memories (idempotent — overwrites existing files)

Dependencies: TASK-03
Acceptance criteria: exported files contain valid YAML frontmatter with all fields; `shared_at` updated in DB after export; re-running push with same data is idempotent; session rows never appear in `docs/memory/`

---

**TASK-06: Create `internal/memory/import.go` — `cx memory pull` and index rebuild implementation**

Files: `internal/memory/import.go`

Implement:
- `Pull(db *sql.DB, docsDir string) (PullResult, error)` — scan `docs/memory/{observations,decisions}/`; parse frontmatter + body; for each parsed memory: if `id` not in DB → insert; if `id` in DB with same content → skip; if `id` in DB with different content → record conflict, skip; set `shared_at` on newly imported rows
- `PullResult` struct: `{Imported int, Skipped int, Conflicts []ConflictItem}`
- `ConflictItem` struct: `{ID string, LocalContent string, ImportedContent string}`
- `RebuildFromMarkdown(db *sql.DB, docsDir string) error` — full re-ingest of all `docs/memory/` files into DB; used by `cx index rebuild`; overwrites existing rows (no conflict detection — this is a local rebuild, not a pull from teammates)
- Parsing: read YAML frontmatter between `---` delimiters; extract all fields; body is everything after second `---`

Dependencies: TASK-03
Acceptance criteria: fixture markdown files parse correctly into expected Memory structs; conflicting IDs are skipped and listed in PullResult.Conflicts; non-conflicting IDs are imported; RebuildFromMarkdown on a directory with N files produces N rows in DB

---

**TASK-07: Wire `cmd/memory.go` — all `cx memory` subcommands**

Files: `cmd/memory.go`, `cmd/root.go`

Implement cobra subcommands:
- `cx memory save --type <T> --title "..." --content "..." [--change C] [--files p1,p2] [--specs s1,s2] [--tags t1,t2] [--deprecates slug] [--source agent] [--visibility personal|project]` → calls `entities.SaveMemory()`; visibility defaults to `project` for observation; writes to DB; also writes markdown file to `docs/memory/` (Phase 1: write-both)
- `cx memory decide --title "..." --context "..." --outcome "..." --alternatives "..." --rationale "..." --tradeoffs "..." [--change C] [--specs s1,s2] [--tags t1,t2] [--deprecates slug] [--visibility personal|project]` → validates all five body sections present; writes decision memory; default visibility `project`
- `cx memory session --goal "..." --accomplished "..." --next "..." [--discoveries "..."] [--blockers "..."] [--files "..."] [--change C]` → writes session memory; default visibility `personal`
- `cx memory note --type <T> --title "..." --content "..." [--topic-key key] [--projects p1,p2] [--tags t1,t2]` → upserts personal note in `~/.cx/memory.db`
- `cx memory search "query" [--type T] [--author A] [--change C] [--all-projects] [--include-deprecated]` → calls `fts.SearchMemories()` or `fts.SearchAllProjects()`
- `cx memory list [--type T] [--change C] [--recent 7d] [--all-projects]` → calls `entities.ListMemories()`
- `cx memory link <id1> <id2> --relation <type>` → calls `entities.SaveMemoryLink()`
- `cx memory push [--all]` → calls `export.Push()`
- `cx memory pull` → calls `import.Pull()`; prints conflicts to stderr if any

Register `memoryCmd` in `cmd/root.go`.

Dependencies: TASK-03, TASK-04, TASK-05, TASK-06
Acceptance criteria: each subcommand invokes the correct internal function; `cx memory save` exits non-zero on missing required fields; `cx memory pull` prints conflict warnings to stderr; all commands produce useful output on success

---

**TASK-08: Wire `cmd/agent_run.go` — `cx agent-run` subcommands**

Files: `cmd/agent_run.go`, `cmd/root.go`

Implement cobra subcommands:
- `cx agent-run log --type <agent_type> --session <session_id> --status <success|blocked|needs-input> --summary "..." [--artifacts p1,p2] [--duration-ms N] [--prompt-summary "..."]` → calls `entities.SaveAgentRun()` in current project's `.cx/memory.db`
- `cx agent-run list [--session <session_id>]` → calls `entities.ListAgentRuns()`; prints tabular output

Register `agentRunCmd` in `cmd/root.go`.

Dependencies: TASK-03
Acceptance criteria: `cx agent-run log` writes a row with correct fields; `cx agent-run list` returns the row; missing required flags exit non-zero with helpful message

---

**TASK-09: Update `internal/project/registry.go` — replace projects.json with index.db**

Files: `internal/project/registry.go`

Modify `GlobalCXDir()` and related functions:
- On first call: if `~/.cx/index.db` does not exist, create it via `memory.OpenGlobalIndexDB()` + `memory.Migrate()`; if `projects.json` exists, one-time import all paths into `projects` table
- Replace all `projects.json` reads/writes with `index.db` queries: `RegisterProject(path, name, gitRemote)`, `GetRegisteredProjects() ([]Project, error)`, `UnregisterProject(path)`
- Keep the function signatures compatible with existing call sites

Dependencies: TASK-01, TASK-02
Acceptance criteria: after migration, `GetRegisteredProjects()` returns all previously registered paths; new `cx init` registers the project in `index.db`; `projects.json` is not read or written after migration

---

**TASK-10: Update `internal/project/scaffold.go` — create `.cx/memory.db` on init**

Files: `internal/project/scaffold.go`

Current state: `ScaffoldCXCache()` already creates `.cx/` directory, `.cx/.gitignore`, and `.cx/cx.yaml` from the embedded template. The `DIRECTION.md` generation was removed from this file in a prior commit; there is no reference to DIRECTION.md in the current file.

Modify `ScaffoldCXCache()` to also:
- Create `.cx/memory.db` by calling `memory.OpenProjectDB(projectPath)` + `memory.Migrate()`
- Close the DB after migration
- Update `CXCacheResult` struct to include a `MemoryDBCreated bool` field

Dependencies: TASK-01, TASK-02
Acceptance criteria: `cx init` produces a `.cx/memory.db` with the correct schema version; running `cx init` a second time does not corrupt an existing DB

---

**TASK-11: Update `internal/doctor/checks.go` — memory health checks**

Files: `internal/doctor/checks.go`

Current state: `CheckDocsStructure()` already validates `.cx/cx.yaml` when present (warns on parse failure). The DIRECTION.md required-file check that existed in an earlier version of the spec has been removed from this file — there is no DIRECTION.md check in the current code. `CheckMemoryHealth()` currently counts `.md` files in `docs/memory/{observations,decisions,sessions}/`. `CheckIndexHealth()` checks for `.cx/.index.db`.

Add new checks:
- `CheckMemoryDBExists(projectPath string)` — warn if `.cx/memory.db` is missing (Warning, auto-fixable via `cx index rebuild`)
- `CheckMemorySchemaVersion(projectPath string)` — warn if DB schema version is less than current expected version (Warning, auto-fixable via `cx index rebuild`)
- `CheckMemorySyncConflicts(projectPath, docsDir string)` — compare IDs in local `.cx/memory.db` against `docs/memory/{observations,decisions}/` files; warn for each ID where content differs ("memory sync conflict — local and shared versions of memory `<id>` differ") (Warning, not auto-fixable)

Integrate new checks into `RunAllChecks()` by adding a dedicated `CheckMemoryDBHealth(rootDir string) CheckGroup` function that calls the above three helpers.

No changes needed to `CheckMemoryHealth()` (file count checks remain valid) or `CheckIndexHealth()` (unchanged).

Dependencies: TASK-03, TASK-06
Acceptance criteria: `cx doctor` warns when `.cx/memory.db` is missing; warns when schema version is stale; warns on each sync conflict; does not block — prints warnings and exits 0

---

**TASK-12: Add `modernc.org/sqlite` to go.mod**

Files: `go.mod`, `go.sum`

Run `go get modernc.org/sqlite` to add the pure-Go SQLite driver. Register it as the `sqlite` driver name in an init function in `internal/memory/db.go` (or a separate `internal/memory/driver.go`).

Dependencies: none
Acceptance criteria: `go build ./...` succeeds; `go test ./internal/memory/...` uses the driver without cgo

---

**TASK-13: Tests for `internal/memory/` — unit tests**

Files: `internal/memory/migrations_test.go`, `internal/memory/entities_test.go`, `internal/memory/fts_test.go`, `internal/memory/import_test.go`

Write tests:
- `migrations_test.go`: apply migrations on a fresh in-memory DB; verify schema version; upgrade v1 → v2 produces two rows in schema_migrations; Migrate() on current DB is no-op
- `entities_test.go`: CRUD round-trips for Memory, Session, AgentRun, MemoryLink; deprecation chain marks referenced entity `deprecated=1`; ListMemories default excludes deprecated rows; `--include-deprecated` includes them
- `fts_test.go`: FTS5 query returns expected rows; deprecated rows excluded by default; FTS5 over two test project DBs (federated) returns results from both with correct project attribution
- `import_test.go`: parse fixture markdown files from `testdata/docs/memory/`; verify DB rows match expected entity fields; conflicting pull skips and records the conflict

Dependencies: TASK-01 through TASK-06
Acceptance criteria: `go test ./internal/memory/...` passes; test coverage includes happy path and error cases for each function

---

### Phase 2 — Team Sync Validation and Doctor

**TASK-14: Integration tests for push/pull and conflict detection**

Files: `internal/memory/export_test.go`, `internal/memory/import_test.go` (extended)

Write integration tests:
- `cx memory push` exports only `visibility=project` rows; personal and session rows remain local; exported files have valid YAML frontmatter; `shared_at` updated in DB
- `cx memory pull` imports non-conflicting rows; warns and skips conflicting rows; same-content rows are skipped silently
- `cx doctor` conflict check: local DB and `docs/memory/` disagree on a memory ID → warning emitted
- Deprecation: save entity A, save entity B with `--deprecates A`, verify A excluded from default search, included with `--include-deprecated`
- Visibility: session rows with `visibility=personal` never appear in `docs/memory/` after push

Dependencies: TASK-05, TASK-06, TASK-11
Acceptance criteria: all integration tests pass; `cx doctor` warnings are deterministic given known input state

---

### Phase 3 — Agent Integration (Skill and Template Updates)

**TASK-15: Update cx-build skill — add memory touchpoints**

Files: `internal/skills/data/cx-build.md` (canonical source), `.claude/skills/cx-build/SKILL.md` (installed copy — must also be updated)

Current state: `cx-build.md` already references `.cx/cx.yaml` in the executor context assembly step. It already has a `cx memory save --type session` rule at the end. No DIRECTION.md references. No agent-run logging. No Primer dispatch at session start.

Changes:
- Step 1 (before requirements): Add mandatory Primer dispatch before requirements gathering; Primer loads recent observations, active decisions, personal notes; currently this step is skipped entirely in cx-build
- Requirements step: Add note that Master saves significant constraints and choices as decisions via `cx memory decide --change <name>`
- Implementation step: Add `cx agent-run log` call after each Agent dispatch returns; include `session_id` in executor dispatch prompt so executors can include it in their own `cx agent-run log` calls
- Session end: Replace `cx memory save --type session` (old file-based command) with `cx memory session --goal "..." --accomplished "..." --next "..."` with all required fields: goal, accomplished, change_id, discoveries, blockers, next_steps, files_touched

Note: After updating `internal/skills/data/cx-build.md`, run `cx sync` (or manually copy) to update `.claude/skills/cx-build/SKILL.md`.

Dependencies: TASK-07, TASK-08
Acceptance criteria: skill file reflects all four workflow memory touchpoints from design.md Section 2; required session summary fields match the table in design.md Section 3; `cx doctor --fix` would sync both copies

---

**TASK-16: Update cx-continue skill — session recovery and memory touchpoints**

Files: `internal/skills/data/cx-continue.md` (canonical source), `.claude/skills/cx-continue/SKILL.md` (installed copy — must also be updated)

Current state: `cx-continue.md` already references `.cx/cx.yaml` indirectly (Primer loads it). It already dispatches Primer in Step 1. It has `cx memory save --type session` in the Rules section (old file-based command). No explicit `next_steps` guidance.

Changes:
- Step 1 (load context): Add explicit note that Primer loads last session summary (`cx memory list --type session --change <name>`); add that `next_steps` from the prior session summary drives what to do next; Primer blocks on disambiguation if multiple active changes
- Session end: Replace `cx memory save --type session` rule with `cx memory session --goal "..." --accomplished "..." --next "..."` with all required fields; emphasize `next_steps` as the critical bridge — without it, the next CONTINUE session cannot recover state

Dependencies: TASK-07
Acceptance criteria: CONTINUE mode explicitly loads session context via DB; next_steps field called out as critical for continuity

---

**TASK-17: Update cx-plan skill — session summary at BUILD transition**

Files: `internal/skills/data/cx-plan.md` (canonical source), `.claude/skills/cx-plan/SKILL.md` (installed copy — must also be updated)

Current state: `cx-plan.md` already references `.cx/cx.yaml`. No session summary step. No explicit clean-slate context guidance.

Changes:
- Session start: Add explicit note that PLAN mode loads minimal context (project overview only via `cx context --mode plan`); intentionally clean-slate for creative work — do not load observations, decisions, or change history
- Transition to BUILD step: Add `cx memory session --goal "..." --accomplished "..."` call capturing what was planned and the decomposed change name before exiting PLAN mode

Dependencies: TASK-07
Acceptance criteria: PLAN mode saves a session summary when transitioning to BUILD; skill documents that PLAN context loading is intentionally minimal

---

**TASK-18: Update cx-review skill — pre-review memory loading**

Files: `internal/skills/data/cx-review.md` (canonical source), `.claude/skills/cx-review/SKILL.md` (installed copy — must also be updated)

Current state: `cx-review.md` already references `.cx/cx.yaml` in Step 2. No memory loading step. No explicit read-only constraint documented.

Changes:
- Step 1 (before reading changes): Add `cx memory search --change <name>` to load change-scoped observations and decisions; these inform the review (prior constraints, design decisions made during implementation)
- Rules section: Add explicit read-only constraint — Reviewer never writes memory; significant recurring patterns returned to Master in the review report; Master decides whether to save as observations

Dependencies: TASK-07
Acceptance criteria: Reviewer loads change-scoped memories before reviewing; read-only constraint documented in Rules

---

**TASK-19: Update cx-prime skill — DB-backed memory loading**

Files: `internal/skills/data/cx-prime.md` (canonical source), `.claude/skills/cx-prime/SKILL.md` (installed copy — must also be updated)

Current state: `cx-prime.md` already references `.cx/cx.yaml` in Step 2. Step 3 says "load recent and relevant memories" — generic, no specific commands. No mode classification. No explicit DB commands.

Changes:
- Step 1: Classify session mode: CONTINUE | BUILD | PLAN based on developer's opening message
- Step 2: Use `cx context --mode <mode> [--change <name>]` as the primary entry point to get the context map
- Step 3: Replace generic memory loading with mode-specific DB commands:
  - BUILD: `cx memory list --type decision` + `cx memory list --type observation --recent 7d` + personal notes
  - CONTINUE: `cx memory list --type session --change <name>` (last session first) + `cx memory search --change <name>` (change-scoped memory)
  - PLAN: personal preference notes only — no project memory loaded
- Add mode-specific loading table documenting these behaviors
- Rules: add that Primer is read-only and disposable — never writes memory, never modifies files

Dependencies: TASK-07
Acceptance criteria: Primer uses DB commands, not direct file reads; mode-specific loading strategy documented with explicit commands for each mode

---

**TASK-20: Update cx-memory skill — DB-backed command reference**

Files: `internal/skills/data/cx-memory.md` (canonical source), `.claude/skills/cx-memory/SKILL.md` (installed copy — must also be updated)

Current state: `cx-memory.md` exists with basic memory skill content. Inspect the current file before rewriting to preserve any still-valid content.

Rewrite to document DB-backed commands. Replace file-based save/search instructions with:
- Full command reference for project memory: `cx memory save`, `cx memory decide`, `cx memory session`, `cx memory search`, `cx memory list`, `cx memory link`, `cx memory push`, `cx memory pull`
- Local memory: `cx memory note`, `cx memory forget`
- Agent run tracking: `cx agent-run log`, `cx agent-run list`
- Visibility tier table: decision/observation → `project` by default; session/agent_run → `personal` by default; override with `--visibility`
- Team sync guidance: when to use `cx memory push` vs `cx memory pull`; conflict warning behavior

Dependencies: TASK-07, TASK-08
Acceptance criteria: skill file matches the CLI interface implemented in TASK-07 and TASK-08; all command flags documented

---

**TASK-21: Update cx-scout, cx-supervise, and cx-conflict-resolve skills**

Files:
- `internal/skills/data/cx-scout.md` + `.claude/skills/cx-scout/SKILL.md`
- `internal/skills/data/cx-supervise.md` + `.claude/skills/cx-supervise/SKILL.md`
- `internal/skills/data/cx-conflict-resolve.md` + `.claude/skills/cx-conflict-resolve/SKILL.md`

Current state: All three files exist; none reference DIRECTION.md. The cx-supervise skill may already reference `.cx/cx.yaml`. Read each before modifying.

Changes:
- `cx-scout`: Add explicit read-only contract — Scout cannot call `cx memory save`; all discoveries must be returned to Master in the summary; Master is responsible for deciding whether to save as observations via `cx memory save --type observation`
- `cx-supervise`: Add agent-run logging guidance — Supervisor passes `session_id` to each sub-agent dispatch; calls `cx agent-run log` after each Contractor/Scout return with the sub-agent's type, status, and summary
- `cx-conflict-resolve`: Add note distinguishing memory sync conflicts (same entity ID, different content in local DB vs `docs/memory/`) from semantic code conflicts; memory sync conflicts are resolved separately via `cx memory pull` and `cx doctor` — not through the conflict resolution workflow

Dependencies: TASK-07, TASK-08
Acceptance criteria: each skill file accurately reflects the agent's memory read/write contract from the agent integration contracts table in design.md

---

**TASK-22: Update subagent templates**

Files: subagent template files embedded in Go source. Confirmed location: `internal/agents/subagents.go` provides `SubagentSlugs()` and `WriteSubagents()`. The actual template content is in embedded files — run `grep -r "cx-primer\|cx-planner\|cx-reviewer\|cx-scout" internal/agents/` to find exact paths. The template files are likely in `internal/agents/data/` or `internal/templates/`.

Before implementing: dispatch Scout to confirm exact file paths for subagent templates in the codebase.

Changes:
- `cx-primer template`: Add `cx context --mode <mode>` as primary entry point; add `cx memory search` and `cx memory list` instructions; add mode-based loading table (BUILD/CONTINUE/PLAN); clarify read-only and disposable
- `cx-planner template`: Add `cx memory decide --change <name>` instruction for architectural decisions made during design; add `cx memory save --type observation` for non-obvious constraints discovered during planning
- `cx-reviewer template`: Add `cx memory search --change <name>` instruction before reviewing; clarify read-only status — findings returned to Master, never saved by Reviewer
- `cx-scout template`: Add explicit read-only memory contract — discoveries returned to Master; cannot call `cx memory save`

Dependencies: TASK-07, TASK-08
Acceptance criteria: each template accurately describes the agent's memory behavior per the agent integration contracts table in design.md; `cx doctor --fix` would sync installed copies with embedded templates

---

**TASK-23: Update Master agent config — CLAUDE.md**

Files: `CLAUDE.md` (the Master agent instruction file at project root)

Current state: `CLAUDE.md` already references `.cx/cx.yaml` in the Context Loading Protocol. No DIRECTION.md references anywhere in the file. No `cx agent-run log` instructions. Session summary is mentioned as `cx memory save` (old command) in the cx-build and cx-continue skill Rules.

The `CLAUDE.md` file is the Master agent's primary instruction set. It is NOT an embedded template managed by the binary — it is committed directly to the repository.

Changes:
- Add `cx agent-run log` call pattern to the "What You Do" section: after each Agent tool dispatch returns, log the completed run with `cx agent-run log --type <t> --session <id> --status <s> --summary "..."`
- Add session management guidance: at session start, generate or assign a session_id; pass it to all subagent dispatch prompts; use it as the `--session` flag in all `cx agent-run log` calls
- Add session end pattern: save session summary via `cx memory session --goal "..." --accomplished "..." --next "..."` at session end or before context compaction
- Update the memory command references in any skill Rule sections that still reference `cx memory save --type session` to use `cx memory session` instead

Dependencies: TASK-07, TASK-08
Acceptance criteria: CLAUDE.md instructs agent-run logging after every dispatch; session summary saved at session end with all required fields; session_id passed to all subagent dispatches

---

## Implementation Notes

**Dependency order**: Tasks within Phase 1 must be implemented in numerical order (01 → 13). TASK-12 (go.mod) can be done in parallel with TASK-01. Phase 2 (TASK-14) can start after TASK-05, TASK-06, TASK-11. Phase 3 tasks (TASK-15 through TASK-23) are independent of each other once Phase 1 is complete, and can be distributed across sessions.

**Phase 1 is the critical path.** Phases 2–3 add important behavioral correctness and agent integration but do not block basic memory DB functionality.

**Driver registration**: `modernc.org/sqlite` must be registered once in an `init()` function before any DB open. Place this in `internal/memory/db.go` or a dedicated `internal/memory/driver.go` file to keep it isolated.

**Rebuild trigger decision** (open question from masterfile): Before implementing TASK-06 (`import.go`), decide whether `.cx/memory.db` uses its own `last_indexed_at` or reuses the `.index.db` mtime mechanism. Recommended: add `last_memory_indexed_at` to `projects` table in `index.db` (TASK-02) and check it in `import.go` (TASK-06).

**Phase 2 materialized index**: The `memory_index` table in `~/.cx/index.db` is scaffolded in TASK-02 but populated only in a future `cx index rebuild --global` command. Phase 1 cross-project search (TASK-04) opens individual project DBs directly. This is acceptable for N < 50 projects.

**Skill file paths**: The canonical skill sources are `internal/skills/data/<skill>.md`. When updated, the installed copies at `.claude/skills/<skill>/SKILL.md` must also be updated (either manually or via `cx sync`). TASK-22 references subagent templates whose exact embedded file paths should be confirmed by Scout before implementation begins — look in `internal/agents/` for the template source files.

**CLAUDE.md is not an embedded template**: TASK-23 modifies `CLAUDE.md` directly. It is not managed by the `cx sync` command — changes must be committed directly. No DIRECTION.md references exist in the current `CLAUDE.md` — no cleanup needed there.

**scaffold.go current state**: `ScaffoldCXCache()` currently handles `.cx/` directory, `.cx/.gitignore`, and `.cx/cx.yaml`. DIRECTION.md is not generated by scaffold — that was removed in a prior commit. TASK-10 only needs to add `.cx/memory.db` creation.

**checks.go current state**: `CheckDocsStructure()` already validates `cx.yaml` when present. The DIRECTION.md required-file check is not in the current code. TASK-11 only needs to add the new memory DB health check group.
