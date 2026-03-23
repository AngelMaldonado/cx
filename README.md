# CX

AI-native project knowledge system that unifies spec-driven development, shared team memory, and multi-agent coordination.

See [docs/overview.md](docs/overview.md) for the problem statement.

## Installation

### Homebrew (recommended)

```bash
brew tap AngelMaldonado/cx https://github.com/AngelMaldonado/cx.git
brew install cx
```

### From source

```bash
go install github.com/AngelMaldonado/cx@latest
```

### Build from source

```bash
git clone https://github.com/AngelMaldonado/cx.git
cd cx
make build
```

## Requirements

- **Go 1.25+** (see [go.mod](go.mod))

## Build

```bash
go build -o cx .
```

## Install globally

Symlink the binary so `cx` runs from anywhere. Rebuilds update automatically.

```bash
go build -o cx .
ln -sf $(pwd)/cx /usr/local/bin/cx
```

## Quick start

```bash
cx init                          # one-time project setup
cx brainstorm new my-feature     # start planning
cx decompose my-feature          # turn plan into a structured change
cx change status                 # check progress
```

## Commands

### Project setup

| Command | Description |
|---------|-------------|
| `cx init` | Initialize CX in the current project — scaffolds docs/, .cx/, agent configs, skills, and .cx/cx.yaml |
| `cx sync` | Regenerate agent configs, skills, and MCP settings |
| `cx doctor` | Run diagnostics and report project health (validates docs/ structure, cx.yaml, memory files) |
| `cx projects` | List registered projects; `cx projects remove <path>` to unregister |

### Planning

| Command | Description |
|---------|-------------|
| `cx brainstorm new <name>` | Create a masterfile for ideation and planning |
| `cx brainstorm status` | List active masterfiles |
| `cx decompose <name>` | Transform a masterfile into a structured change (proposal, design, tasks, specs/) and archive the masterfile |

### Changes

| Command | Description |
|---------|-------------|
| `cx change new <name>` | Scaffold a new change with template files (proposal.md, design.md, tasks.md, specs/) |
| `cx change status` | Show all changes with completion state, verify status, and synced delta markers |
| `cx change verify <name>` | Generate a structured verification prompt and scaffold verify.md |
| `cx change spec-sync <name>` | Merge delta specs into canonical specs mid-change without archiving |
| `cx change archive <name>` | Validate completeness, bootstrap missing canonical specs, and move to archive |
| `cx change archive --skip-specs` | Archive without spec verification (for CI, tooling, or docs-only changes) |

### Agent support

| Command | Description |
|---------|-------------|
| `cx instructions <artifact>` | Get template, project context, rules, dependency graph, and spec index for an artifact |
| `cx completion <shell>` | Generate shell completion scripts (bash, zsh, fish, powershell) |

## Artifact lifecycle

```
brainstorm → decompose → proposal → design ──→ tasks → verify → archive
                                  ↘ specs/ ↗
```

Every change lives in `docs/changes/<name>/` with:

- **proposal.md** — problem, approach, scope, affected specs
- **design.md** — architecture, technical decisions, implementation notes
- **tasks.md** — task breakdown with Linear issue references
- **specs/** — delta specs per affected area (ADDED/MODIFIED/REMOVED requirements + scenarios)
- **verify.md** — structured verification (completeness, correctness, coherence)

All artifacts use YAML frontmatter for metadata. Specs use RFC 2119 keywords (MUST/SHOULD/MAY) and REQ-NNN identifiers.

## Spec system

Canonical specs live in `docs/specs/<area>/spec.md`. Changes produce delta specs that describe what's changing:

```markdown
## ADDED Requirements
### REQ-005: New behavior
The system MUST ...

## MODIFIED Requirements
### REQ-002: Updated behavior
Previous: The system SHOULD ...
New: The system MUST ...

## REMOVED Requirements
### REQ-003: Deprecated behavior
```

On archive, delta specs merge into canonical specs via agent-assisted review. For greenfield projects, the archive workflow bootstraps new spec areas automatically.

## Project config

`cx init` creates `.cx/cx.yaml` with project context and per-artifact rules:

```yaml
schema: cx/v1

context: |
  Language: Go 1.25
  Framework: Cobra CLI
  Database: PostgreSQL

rules:
  specs:
    - Use Given/When/Then format for scenarios
  design:
    - Include sequence diagrams for cross-service flows
```

Agents receive this context when creating artifacts via `cx instructions`.

## Multi-agent orchestration

CX coordinates a Master agent with specialized subagents:

- **Primer** — loads and distills project context on demand (read-only)
- **Scout** — explores and maps codebases (read-only)
- **Planner** — designs solutions, writes masterfiles, fills change docs, merges specs on archive
- **Reviewer** — reviews code and docs for quality, correctness, and security (read-only)
- **Executor agents** — project-specific experts (e.g., go-expert, react-expert) defined by the developer

## Working modes

Every CX session runs in one of three modes. The Master agent classifies the developer's opening message and activates the corresponding workflow.

| Mode | When to use | Output |
|------|-------------|--------|
| **PLAN** | "how should we approach...", "let's design...", "brainstorm..." | A masterfile at `docs/masterfiles/` |
| **BUILD** | "implement...", "add...", "build...", "create..." | A completed change in `docs/changes/<name>/` |
| **CONTINUE** | "continue working on...", "pick up where we left off", "resume..." | Progress on an active change |

### PLAN

For high-level thinking and design — without writing a line of code. Context is intentionally minimal (project overview only) to encourage clean-slate thinking.

1. Primer loads project overview
2. Requirements gathered via conversation
3. Planner creates a masterfile at `docs/masterfiles/<name>.md`
4. Iterate on the masterfile until the design is solid
5. Transition to BUILD (only when the developer explicitly approves)

### BUILD

For creating something new — a feature, fix, or integration. Follows the full change lifecycle from planning through implementation and review.

1. Primer loads context (spec index, active decisions, recent observations)
2. Requirements gathered, plan approved
3. `cx decompose <name>` scaffolds `docs/changes/<name>/` with proposal.md, design.md, tasks.md
4. Planner fills in the change docs; executors work through the task list
5. Reviewer gates the work; on pass, `cx change archive <name>` merges delta specs into canonical specs

Dependency graph enforced by the Master:

```
proposal → specs + design → tasks → implement → verify → archive
```

### CONTINUE

For resuming existing work. State is recovered from change docs, memory, and session history — no need to re-explain context.

1. Primer loads the active change (proposal, design, tasks, last session summary, change-scoped memory)
2. `cx change status` shows what's done and what remains
3. Executor picks up where the last session left off, guided by the `--next` field from the prior session summary

The `--next` field written by `cx memory session` at the end of each session is the critical bridge — it tells the next session exactly what to do first.

## Memory system

SQLite-backed persistence for project knowledge. Memories survive across sessions and sync with the team via git.

### Commands

| Command | Description |
|---------|-------------|
| `cx memory save --type <T> --title "..." --content "..."` | Save an observation or agent interaction |
| `cx memory decide --title "..." --context "..." --outcome "..." --alternatives "..." --rationale "..."` | Record a technical decision |
| `cx memory session --goal "..." --accomplished "..." --next "..."` | Save session summary (--next is critical for CONTINUE mode) |
| `cx memory search "query"` | FTS5 search across memories |
| `cx memory list [--type T] [--recent 7d]` | List memories with filters |
| `cx memory link <id1> <id2> --relation <type>` | Link two memory entities |
| `cx memory push` | Export project memories to `docs/memory/` for team sharing |
| `cx memory pull` | Import teammates' memories from `docs/memory/` |
| `cx agent-run log --type <T> --session <id> --status <S>` | Track agent invocations |
| `cx agent-run list` | List agent runs |

### Architecture

Three SQLite databases, each with a distinct scope:

```
<project>/.cx/memory.db          ~/.cx/index.db          ~/.cx/memory.db
   per-project store                global registry           personal notes
 ──────────────────────         ──────────────────────    ──────────────────
  memories                       projects                  personal_notes
  memories_fts (FTS5)            memory_index              personal_notes_fts
  sessions                       memory_index_fts (FTS5)   schema_migrations
  agent_runs                     schema_migrations
  memory_links
  schema_migrations
         │                              │
         │   docs/memory/ (git-tracked) │
         └──────────────────────────────┘
              team transport layer
```

**`<project>/.cx/memory.db`** — Per-project memory store. Created by `cx init`.
- `memories` — observations, decisions, sessions, and agent interactions. Columns: `id`, `entity_type`, `subtype`, `title`, `content`, `author`, `source`, `change_id`, `file_refs`, `spec_refs`, `tags`, `visibility`, `shared_at`, `created_at`, `updated_at`, `archived_at`
- `memories_fts` — FTS5 virtual table (content mirror of `memories`) for full-text search
- `sessions` — build/continue/plan session records with `mode`, `change_name`, `goal`, `summary`
- `agent_runs` — per-session agent invocations with `agent_type`, `result_status`, `duration_ms`
- `memory_links` — typed links between memories (`related-to`, `caused-by`, `resolved-by`, `see-also`)
- `schema_migrations` — applied migration versions
- All three databases use WAL mode and foreign key enforcement. `.cx/memory.db` is gitignored — never committed.

**`~/.cx/index.db`** — Global project registry. Replaces the old `projects.json`.
- `projects` — registered project paths with `name`, `path`, `git_remote`, `last_synced`
- `memory_index` — lightweight title/tag index of each project's memories, synced on push/pull
- `memory_index_fts` — FTS5 virtual table for cross-project search
- `schema_migrations`

**`~/.cx/memory.db`** — Personal notes. Local-only, never synced with the team.
- `personal_notes` — columns: `id`, `topic_key`, `title`, `content`, `tags`, `projects`, `created_at`, `updated_at`
- `personal_notes_fts` — FTS5 virtual table
- `schema_migrations`

The `docs/memory/` directory is the team transport layer: `cx memory push` exports project memories as markdown files that get committed to git, and `cx memory pull` imports them into the local DB. SQLite is the query layer; markdown is the sync layer.

### Team sync

Memories sync via git using markdown as the interchange format:

```bash
cx memory push          # export to docs/memory/ (git-tracked)
git add docs/memory/ && git commit && git push
# teammate pulls...
cx memory pull          # import from docs/memory/ into local DB
```

Conflict detection: if a memory ID exists locally with different content, `cx memory pull` warns and skips. `cx doctor` also checks for sync conflicts.

### Visibility

| Type | Default | Shared via push? |
|------|---------|-----------------|
| Observation | `project` | Yes |
| Decision | `project` | Yes |
| Session | `personal` | No (unless overridden) |
| Agent run | `personal` | No |

## Dashboard

Launch the interactive TUI with `cx dashboard` (aliases: `dash`, `ui`). The dashboard polls all three SQLite databases every 5 seconds and displays live project memory.

### Navigation

| Key | Action |
|-----|--------|
| `h` / `←` | Previous tab |
| `l` / `→` | Next tab |
| `tab` / `shift+tab` | Next / previous tab |
| `1`–`8` | Jump directly to a tab |
| `j` / `↓` | Navigate down |
| `k` / `↑` | Navigate up |
| `g` / `G` | Top / bottom of list |
| `ctrl+d` / `ctrl+u` | Half-page scroll down / up |
| `ctrl+f` / `ctrl+b` | Full-page scroll down / up |
| `H` / `M` / `L` | Jump to visible top / middle / bottom |
| `/` | Activate search (Memories and Cross-Project tabs) |
| `esc` | Exit search or close overlay |
| `r` | Force data refresh |
| `?` | Toggle help overlay |
| `q` | Quit |

### Tabs

| # | Tab | Description |
|---|-----|-------------|
| 1 | Home | Overview stats and recent activity |
| 2 | Memories | Browse observations and decisions with FTS5 search, type filter, and detail overlay |
| 3 | Sessions | Timeline of build/plan/continue sessions |
| 4 | Runs | Agent execution history grouped by session (expand/collapse with `space`) |
| 5 | Sync | Push/pull status and pending memory exports (`p` push, `P` push --all, `u` pull) |
| 6 | Notes | Personal notes browser (reads `~/.cx/memory.db`) |
| 7 | Graph | Memory link relationship graph |
| 8 | Cross-Project | Federated FTS5 search across all registered projects via `~/.cx/index.db` |

## Project layout

```
docs/
├── overview.md              # project description
├── specs/                   # canonical specifications
│   ├── index.md
│   └── <area>/spec.md
├── changes/                 # active work in progress
│   └── <name>/
│       ├── proposal.md
│       ├── design.md
│       ├── tasks.md
│       ├── verify.md
│       └── specs/<area>/spec.md   # delta specs
├── archive/                 # completed changes (audit trail)
├── memories/                # observations, decisions, sessions
└── masterfiles/             # active brainstorm documents
.cx/
├── cx.yaml                  # project config
├── memory.db                # SQLite: observations, decisions, sessions, agent runs
└── .index.db                # FTS5 search index (gitignored)
```
