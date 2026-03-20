# CX

AI-native project knowledge system that unifies spec-driven development, shared team memory, and multi-agent coordination.

See [docs/overview.md](docs/overview.md) for the problem statement.

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

Three SQLite databases:

- **`.cx/memory.db`** (per-project) — observations, decisions, sessions, agent runs. Created by `cx init`.
- **`~/.cx/index.db`** (global) — project registry + cross-project search index. Replaces `projects.json`.
- **`~/.cx/memory.db`** (personal) — private notes that don't sync with the team.

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
