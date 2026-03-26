# CX Framework — Architecture & Agent Behavior Report

> Generated 2026-03-26 from the current codebase implementation.

---

## Table of Contents

1. [What CX Is](#what-cx-is)
2. [High-Level Architecture](#high-level-architecture)
3. [CLI Command Tree](#cli-command-tree)
4. [Agent Hierarchy & Dispatch](#agent-hierarchy--dispatch)
5. [Session Modes](#session-modes)
6. [Change Lifecycle](#change-lifecycle)
7. [Memory System](#memory-system)
8. [Spec Management](#spec-management)
9. [Brainstorm & Masterfile Flow](#brainstorm--masterfile-flow)
10. [Worktree-Based Parallel Execution](#worktree-based-parallel-execution)
11. [Skills System](#skills-system)
12. [TUI Dashboard](#tui-dashboard)
13. [Internal Package Map](#internal-package-map)
14. [Build & Distribution](#build--distribution)
15. [Key Design Decisions](#key-design-decisions)

---

## What CX Is

CX is an **AI-native project knowledge system** built in Go. It is a CLI tool that:

- Scaffolds structured documentation (`docs/`) as the single source of truth
- Manages a SQLite-backed memory store (observations, decisions, session summaries)
- Tracks work through a formal **change lifecycle** with dependency gates
- Installs and configures AI agent harness files for Claude Code, Gemini CLI, and Codex CLI
- Provides a Bubble Tea TUI dashboard for browsing memories, sessions, and agent runs

The core philosophy: **`docs/` is the canonical state, committed to git. SQLite is a local query cache rebuilt from those files.** AI agents orchestrate work through structured documents, not ad-hoc conversations.

---

## High-Level Architecture

```mermaid
graph TB
    subgraph Developer
        DEV[Developer]
    end

    subgraph "CX CLI (Go Binary)"
        CLI[cx binary<br/>Cobra CLI]
        TUI[Bubble Tea TUI<br/>8-tab dashboard]
        PROJ[project pkg<br/>Git detection, scaffolding]
        MEM[memory pkg<br/>3 SQLite DBs + FTS5]
        CHG[change pkg<br/>Lifecycle management]
        BRAIN[brainstorm pkg<br/>Masterfile management]
        AGENTS[agents pkg<br/>Config file generation]
        SKILLS[skills pkg<br/>Embedded skill files]
        TMPL[templates pkg<br/>Embedded doc templates]
        DOC[doctor pkg<br/>Health checks]
        CFG[config pkg<br/>cx.yaml parser]
        INSTR[instructions pkg<br/>Artifact prompt builder]
        VER[verify pkg<br/>Verification scaffold]
    end

    subgraph "File System"
        DOCS["docs/<br/>specs/ changes/ archive/<br/>masterfiles/ memory/"]
        CXDIR[".cx/<br/>cx.yaml, memory.db"]
        GLOBAL["~/.cx/<br/>index.db, memory.db,<br/>disabled sentinel"]
        AGENTCFG[".claude/ .gemini/ .codex/<br/>Agent config + skills"]
    end

    subgraph "AI Agent Layer"
        MASTER[Master Agent<br/>Orchestrator]
        PRIMER[Primer<br/>Context loader]
        SCOUT[Scout<br/>Code explorer]
        PLANNER[Planner<br/>Solution designer]
        REVIEWER[Reviewer<br/>Quality gate]
        EXEC[Executor Agents<br/>Code writers]
    end

    DEV -->|runs| CLI
    DEV -->|runs| TUI
    CLI --> PROJ & MEM & CHG & BRAIN & DOC & CFG & INSTR & VER
    TUI --> MEM
    AGENTS --> AGENTCFG
    SKILLS -.->|go:embed| CLI
    TMPL -.->|go:embed| CLI
    PROJ --> DOCS & CXDIR & GLOBAL
    MEM --> CXDIR & GLOBAL
    CHG --> DOCS

    DEV -->|opens AI session| MASTER
    MASTER -->|dispatches| PRIMER & SCOUT & PLANNER & REVIEWER & EXEC
    MASTER -->|runs directly| CLI
    PRIMER -.->|reads| DOCS & MEM
    SCOUT -.->|reads| DOCS
    PLANNER -->|writes| DOCS
    EXEC -->|writes code| DOCS
    REVIEWER -.->|reads| DOCS
```

---

## CLI Command Tree

Every command starts with `project.IsGitRepo()` to find the project root via `git rev-parse --show-toplevel`. A `PersistentPreRun` hook blocks all commands (except `enable`, `disable`, `version`) when `~/.cx/disabled` exists.

```mermaid
graph LR
    CX[cx]
    CX --> INIT[init]
    CX --> DOCTOR[doctor --fix]
    CX --> SYNC[sync]
    CX --> PROJ[projects]
    CX --> CHANGE[change]
    CX --> BRAINSTORM[brainstorm]
    CX --> DECOMPOSE[decompose]
    CX --> MEMORY[memory]
    CX --> AGENTRUN[agent-run]
    CX --> INSTR[instructions]
    CX --> DASH[dashboard]
    CX --> DISABLE[disable]
    CX --> ENABLE[enable]
    CX --> WORKTREE[worktree]

    WORKTREE --> WTCREATE[create]
    WORKTREE --> WTLIST[list]
    WORKTREE --> WTCLEAN[cleanup]

    PROJ --> PROJRM[remove]

    CHANGE --> CHNEW[new]
    CHANGE --> CHSTAT[status]
    CHANGE --> CHARCH[archive]
    CHANGE --> CHVER[verify]
    CHANGE --> CHSYNC[spec-sync]

    BRAINSTORM --> BRNEW[new]
    BRAINSTORM --> BRSTAT[status]

    MEMORY --> MSAVE[save]
    MEMORY --> MDECIDE[decide]
    MEMORY --> MSESSION[session]
    MEMORY --> MSEARCH[search]
    MEMORY --> MLIST[list]
    MEMORY --> MPUSH[push]
    MEMORY --> MPULL[pull]
    MEMORY --> MLINK[link]
    MEMORY --> MNOTE[note]
    MEMORY --> MFORGET[forget]
    MEMORY --> MDEPR[deprecate]

    AGENTRUN --> ARLOG[log]
    AGENTRUN --> ARLIST[list]
```

### Command Purposes

| Command | What It Does |
|---------|-------------|
| `cx init` | Bootstrap: scaffold `docs/`, `.cx/`, select agent integrations, install git hooks, write `.mcp.json` |
| `cx doctor [--fix]` | Validate project health across 7 check groups; `--fix` auto-repairs fixable issues |
| `cx sync` | Re-generate agent config files, skills, and MCP configs for all installed agents |
| `cx projects` / `remove` | List or remove projects from the global `~/.cx/index.db` registry |
| `cx change new/status/archive/verify/spec-sync` | Full change lifecycle management |
| `cx brainstorm new/status` | Create and list masterfiles for ideation |
| `cx decompose <name>` | Convert masterfile to change structure, archive masterfile |
| `cx memory save/decide/session/search/list/push/pull/link/note/forget/deprecate` | Full memory CRUD, search, and team sync |
| `cx agent-run log/list` | Track AI agent dispatches per session |
| `cx instructions <artifact>` | Generate prompt with template + project context + dependency graph for an artifact |
| `cx dashboard` | Launch the Bubble Tea TUI |
| `cx disable/enable` | Suspend/restore all agent configs (`disable` removes cx-managed files and restores pre-init snapshot; `enable` removes sentinel and runs full sync) |
| `cx worktree create <branch-name>` | Create a git worktree + branch under `.cx/worktrees/` for isolated task execution |
| `cx worktree list` | Show active worktrees: branch name, path, HEAD commit |
| `cx worktree cleanup <change-name>` | Remove all worktrees whose branch name is prefixed with `<change-name>` |

---

## Agent Hierarchy & Dispatch

The Master agent (defined in `CLAUDE.md`) is a pure orchestrator. It **never reads source code, never writes code, never analyzes code**. It classifies developer intent, dispatches specialized subagents, and enforces the change lifecycle.

### Agent Hierarchy

```mermaid
graph TB
    DEV[Developer] --> MASTER[Master Agent<br/>Pure Orchestrator]

    MASTER -->|"session start<br/>(disposable)"| PRIMER[Primer<br/>Context Loader]
    MASTER -->|"code questions"| SCOUT[Scout<br/>Code Explorer]
    MASTER -->|"design work"| PLANNER[Planner<br/>Solution Designer]
    MASTER -->|"post-implementation"| REVIEWER[Reviewer<br/>Quality Gate]
    MASTER -->|"complex tasks"| SUPERVISOR[Supervisor<br/>Multi-Agent Coord]
    MASTER -->|"simple impl"| EXECUTOR[Executor<br/>Code Writer]
    MASTER -->|"after parallel executors"| MERGER[Merger<br/>Branch Integrator]

    PRIMER -.->|"if conflicts.json"| CONFLICT[Conflict Resolver]
    SUPERVISOR --> SCOUT2[Scout]
    SUPERVISOR --> CONTRACTOR[Contractor<br/>Foreman]
    CONTRACTOR --> WORKER1[Worker 1]
    CONTRACTOR --> WORKER2[Worker 2]
    CONTRACTOR --> WORKERN[Worker N]

    style MASTER fill:#f9a825,color:#000
    style PRIMER fill:#4fc3f7,color:#000
    style SCOUT fill:#81c784,color:#000
    style PLANNER fill:#ce93d8,color:#000
    style REVIEWER fill:#ef5350,color:#fff
    style SUPERVISOR fill:#ffb74d,color:#000
    style EXECUTOR fill:#a1887f,color:#fff
    style MERGER fill:#80cbc4,color:#000
    style WORKER1 fill:#90a4ae,color:#000
    style WORKER2 fill:#90a4ae,color:#000
    style WORKERN fill:#90a4ae,color:#000
```

### Agent Capabilities Matrix

| Agent | Reads Code | Writes Code | Reads Memory | Writes Memory | Disposable |
|-------|-----------|-------------|-------------|--------------|------------|
| **Master** | Never | Never | Never directly | `cx memory session/decide/save`, `cx agent-run log` | No (always loaded) |
| **Primer** | Never | Never | `cx memory search/list` | Never | Yes |
| **Scout** | Yes (read-only) | Never | None | Never — returns findings to Master; Master saves non-trivial structural discoveries as observations | Yes |
| **Planner** | Never | Writes `docs/` only | Via Primer context | `cx memory decide` (architectural decisions), `cx memory save --type observation` (non-obvious constraints) | Yes |
| **Reviewer** | Yes (read-only) | Never | `cx memory search --change` | Never — returns findings to Master; Master saves recurring patterns or important lessons as observations | Yes |
| **Executor** | Yes | Yes | Receives primed context | `cx memory save --type observation --change <name>` (non-trivial implementation discoveries) | Yes |
| **Merger** | Yes | Yes (merges branches) | Receives context from Master | `cx memory save --type observation` (conflict patterns and resolution strategies) | Yes |
| **Workers** | Yes | Yes | Inline instructions only | None | Yes |

### CX Core Subagent Templates

CX ships **6 subagent template files** embedded in the binary and written to the agent config directory on `cx init` / `cx sync`:

| Template file | Agent role | Read/Write |
|---------------|-----------|------------|
| `cx-primer.md` | Context Loader — loads docs, specs, memory for session priming | Read-only |
| `cx-scout.md` | Code Explorer — maps codebases, traces code paths, answers structural questions | Read-only |
| `cx-planner.md` | Solution Designer — writes masterfiles and change docs (`docs/` only) | Writes `docs/` |
| `cx-reviewer.md` | Quality Gate — reviews code and docs for correctness, security, conventions | Read-only |
| `cx-executor.md` | Implementation Worker — writes code, runs tests, follows proposal/design/tasks | Writes code |
| `cx-merger.md` | Branch Integrator — merges executor worktree branches after parallel execution; resolves conflicts | Writes code |

`cx-executor.md` includes the context loading protocol, implementation rules, and explicit memory save instructions (`cx memory save --type observation --change <name>`). Project-specific executor agents (e.g., `go-expert`, `react-expert`) are defined separately by the developer and supplement the core executor.

### Dispatch Decision Flow

```mermaid
flowchart TD
    REQ[Developer Request] --> CLASSIFY{Classify Intent}

    CLASSIFY -->|"single file, obvious"| DIRECT_EXEC[Direct → Executor]
    CLASSIFY -->|"code question only"| DIRECT_SCOUT[Direct → Scout]
    CLASSIFY -->|"multi-file, needs exploration"| TEAM[Team Dispatch]
    CLASSIFY -->|"architectural / broad"| TEAM
    CLASSIFY -->|"ambiguous"| TEAM

    TEAM --> SUP[Supervisor]
    SUP --> S[Scout maps files]
    SUP --> C[Contractor plans work]
    C --> W[Workers implement]

    DIRECT_EXEC --> LOG1[cx agent-run log]
    DIRECT_SCOUT --> LOG2[cx agent-run log]
    W --> LOG3[cx agent-run log]
```

### Context Loading Protocol

Every subagent dispatch follows this loading order:

```mermaid
flowchart LR
    A[".cx/cx.yaml<br/>Project context"] --> B["docs/specs/index.md<br/>Spec area map"]
    B --> C["Relevant specs<br/>Only task-related areas"]
    C --> D["docs/changes/<br/>Active overlapping changes"]
    D --> E["Source code<br/>Only when needed"]
```

For executor dispatches, the Master also provides:
1. `proposal.md` and `design.md` from the change
2. The specific task description from `tasks.md`
3. Scout's file map of affected areas
4. Session ID for tracking

---

## Session Modes

The Master classifies every developer interaction into one of four modes. Each mode has different context loading, memory behavior, and workflow steps.

```mermaid
flowchart TD
    DEV[Developer Message] --> MODE{Classify Mode}

    MODE -->|"new feature, build, create"| BUILD
    MODE -->|"resume, continue, where were we"| CONTINUE
    MODE -->|"brainstorm, plan, what if"| PLAN
    MODE -->|"fix, patch, tweak, quick"| FIX

    subgraph "BUILD Mode"
        BUILD --> B1[Primer loads full context]
        B1 --> B2[Gather requirements<br/>3-5 AskUserQuestion rounds]
        B2 --> B3[Planner creates masterfile]
        B3 --> B4[cx decompose → change docs]
        B4 --> B5[Planner fills proposal + design]
        B5 --> B6[Planner generates tasks]
        B6 --> B7["Execute tasks: independent tasks run in<br/>parallel via cx worktree (one branch each);<br/>dependent tasks run sequentially on main tree"]
        B7 --> B7M[Merger integrates worktree branches]
        B7M --> B8[Reviewer quality gate]
        B8 --> B9[Ask about archiving]
        B9 --> B10[cx change archive + spec merge]
    end

    subgraph "CONTINUE Mode"
        CONTINUE --> C1[Primer loads last session<br/>--next field is critical]
        C1 --> C2[cx change status<br/>disambiguate if multiple]
        C2 --> C3[Resume from --next steps]
        C3 --> C4[Execute remaining tasks]
        C4 --> C5[Reviewer if impl done]
    end

    subgraph "PLAN Mode"
        PLAN --> P1[Minimal Primer<br/>clean-slate thinking]
        P1 --> P2[AskUserQuestion for goals]
        P2 --> P3[Planner creates masterfile]
        P3 --> P4[Iterate until satisfied]
        P4 --> P5{Transition to BUILD?}
        P5 -->|yes| B4
        P5 -->|no| P6[Save session memory]
    end

    subgraph "FIX Mode"
        FIX --> F1[Scout maps affected files<br/>NO Primer]
        F1 --> F2[Executor applies fix<br/>NO change docs]
        F2 --> F3{Review?}
        F3 -->|yes| F4[Reviewer]
        F3 -->|no| F5[Done]
    end
```

### Mode Context Comparison

| Aspect | BUILD | CONTINUE | PLAN | FIX |
|--------|-------|----------|------|-----|
| **Primer** | Full load | Session-focused | Minimal (clean slate) | Skipped entirely |
| **Memory loaded** | Decisions + observations (7d) + notes | Last session + change-scoped memory | Personal preferences only | Nothing |
| **Change docs** | Created fresh | Resumed | None until transition | Never created |
| **Token budget** | 500-800 tokens | 500-800 tokens | 300-500 tokens | 0 tokens |
| **Memory written** | Decisions, observations, session | Observations, session | Session (at transition) | `agent-run log` only |
| **Scope guard** | None | None | No implementation allowed | Redirect to BUILD if scope grows |

---

## Change Lifecycle

Changes are the **fundamental unit of work** in CX. Every piece of implementation is tracked as a change in `docs/changes/<name>/`.

### Change Directory Structure

```
docs/changes/<name>/
├── proposal.md      — Problem statement, goals, success criteria
├── design.md        — Technical approach, architecture decisions
├── tasks.md         — Ordered implementation tasks
├── verify.md        — Created by cx change verify (not by cx change new)
└── specs/
    └── <area>/
        └── spec.md  — Delta spec: ADDED / MODIFIED / REMOVED sections
```

### Lifecycle State Machine

```mermaid
stateDiagram-v2
    [*] --> Created: cx change new
    Created --> ProposalDone: Agent fills proposal.md

    ProposalDone --> DesignDone: Agent fills design.md
    ProposalDone --> SpecsDone: Agent creates delta specs

    DesignDone --> TasksDone: Agent fills tasks.md
    SpecsDone --> TasksDone: Agent fills tasks.md

    TasksDone --> Implementing: Executors work through tasks
    Implementing --> VerifyPending: cx change verify
    VerifyPending --> Verified: Reviewer writes PASS in verify.md

    Verified --> Archived: cx change archive
    Archived --> SpecsMerged: Planner merges delta → canonical

    note right of Created
        Scaffolds all files from
        embedded Go templates
    end note

    note right of TasksDone
        GATE: Both specs AND design
        must be done before tasks
    end note

    note right of Verified
        GATE: verify.md must contain
        "PASS" and no "CRITICAL" lines
    end note

    note right of Archived
        Moves to docs/archive/YYYY-MM-DD-name/
        Bootstraps missing canonical specs
    end note
```

### Dependency Graph (Enforced by Master)

```mermaid
graph LR
    PROPOSAL[proposal.md] --> SPECS[delta specs]
    PROPOSAL --> DESIGN[design.md]
    SPECS --> TASKS[tasks.md]
    DESIGN --> TASKS
    TASKS --> APPLY[Implementation]
    APPLY --> VERIFY[verify.md]
    VERIFY --> ARCHIVE[Archive]
```

The Master agent enforces these gates: no step proceeds until its dependencies are complete. `cx change archive` programmatically validates file completeness by comparing content against templates (stripping frontmatter).

### Archive Validation

`cx change archive <name>` performs these checks:

1. `proposal.md`, `design.md`, `tasks.md` must all differ from their templates (not empty/unchanged)
2. `verify.md` must exist and contain the string `PASS`
3. `verify.md` must not contain any line starting with `CRITICAL`
4. Exception: `--skip-specs` bypasses the verify gate for non-behavioral changes

On successful archive:
- Files move to `docs/archive/YYYY-MM-DD-<name>/`
- For each unsynced delta spec where no canonical spec exists, `deltaToCanonical()` bootstraps one from the ADDED Requirements section
- Returns `ArchiveResult{ArchivePath, BootstrappedSpecs[], DeltaSpecs[]}`

---

## Memory System

### Three-Database Architecture

```mermaid
graph TB
    subgraph "Project Scope"
        PDB[".cx/memory.db<br/>Project Memory DB"]
        PDB --> MEMORIES[memories table<br/>+ memories_fts FTS5]
        PDB --> SESSIONS[sessions table]
        PDB --> RUNS[agent_runs table]
        PDB --> LINKS[memory_links table]
    end

    subgraph "Global Scope"
        GDB["~/.cx/index.db<br/>Global Index DB"]
        GDB --> PROJECTS[projects table]
        GDB --> MIDX[memory_index<br/>+ memory_index_fts]
    end

    subgraph "Personal Scope"
        NDB["~/.cx/memory.db<br/>Personal Notes DB"]
        NDB --> NOTES[personal_notes table]
    end

    subgraph "Team Sync (Git)"
        DOCMEM["docs/memory/<br/>decisions/ observations/ sessions/"]
    end

    PDB -->|"cx memory push"| DOCMEM
    DOCMEM -->|"cx memory pull"| PDB
    PDB -->|"memory_index rows"| GDB
```

All databases use WAL mode and foreign keys. Schema migrations are version-tracked via a `schema_migrations` table and run idempotently on every open.

### Memory Entity Types

| Entity | Table | Stored In | Purpose |
|--------|-------|-----------|---------|
| **Observation** | `memories` | Project DB | Discoveries, patterns found during exploration |
| **Decision** | `memories` | Project DB | Technical decisions with context/rationale/alternatives |
| **Session** | `sessions` | Project DB | Session summaries with goal/accomplished/next |
| **Agent Interaction** | `memories` | Project DB | Logged via `cx agent-run log` |
| **Agent Run** | `agent_runs` | Project DB | Individual agent dispatch records linked to sessions |
| **Memory Link** | `memory_links` | Project DB | Relationships: `related-to`, `caused-by`, `resolved-by`, `see-also` |
| **Personal Note** | `personal_notes` | Personal DB | Developer notes that span projects, never synced |

### FTS5 Search Flow

```mermaid
flowchart LR
    QUERY["cx memory search 'query'"] --> FTS["memories_fts<br/>MATCH ?"]
    FTS --> JOIN["JOIN memories<br/>ON rowid"]
    JOIN --> FILTER["WHERE filters:<br/>entity_type, change_id,<br/>author, deprecated=0"]
    FILTER --> RANK["ORDER BY<br/>FTS5 rank"]
    RANK --> RESULT["MemoryResult{<br/>Memory, Rank float64}"]
```

Cross-project search (`--all-projects`) opens each registered project's `.cx/memory.db` via the global index and federates results with project attribution.

### Push/Pull Team Sync

```mermaid
sequenceDiagram
    participant DB as .cx/memory.db
    participant FS as docs/memory/
    participant GIT as Git (team)

    Note over DB,FS: Push (export)
    DB->>DB: SELECT visibility='project'<br/>AND shared_at IS NULL
    DB->>FS: Write <id>.md<br/>(YAML frontmatter + body)
    DB->>DB: UPDATE shared_at = NOW()
    FS->>GIT: git add + commit

    Note over DB,FS: Pull (import)
    GIT->>FS: git pull
    FS->>DB: Parse .md files
    DB->>DB: INSERT new rows
    Note over DB: Conflicts (same ID,<br/>different content)<br/>collected but not<br/>auto-resolved
```

### Deprecation Chain

When `SaveMemory()` is called with `Deprecates != ""`:
1. Sets `deprecated=1` on the referenced memory
2. Removes it from the FTS index
3. If both are decisions: sets `status='superseded'` on the old one

### Memory Ownership Model

Each agent class has a defined and exclusive role in memory writes:

| Agent | Memory Responsibility |
|-------|-----------------------|
| **Master** | Saves session summaries (`cx memory session`), decisions (`cx memory decide`), and observations on behalf of read-only agents (`cx memory save`) |
| **Planner** | Saves architectural decisions (`cx memory decide`) and non-obvious constraints (`cx memory save --type observation`) during design work |
| **Executor** | Saves non-trivial implementation discoveries (`cx memory save --type observation --change <name>`) |
| **Scout** | Read-only — returns findings to Master; Master decides what to save |
| **Reviewer** | Read-only — returns findings to Master; Master decides what to save |
| **Primer** | Read-only — returns distilled context; no memory writes |

### Post-Dispatch Memory Protocol

After each subagent returns, the Master follows this protocol:

| Agent returned | Master action |
|---------------|---------------|
| **Scout** | Evaluate findings; save non-trivial structural discoveries as observations via `cx memory save` |
| **Reviewer** | Save recurring patterns or important lessons as observations via `cx memory save` |
| **Planner** | No Master action needed — Planner saves its own decisions and observations during design work |
| **Executor** | Review summary for anything missed; Executor already saves its own observations |

---

## Spec Management

Specs live in `docs/specs/` with an `index.md` serving as the authoritative catalog.

### Spec Architecture

```mermaid
graph TB
    subgraph "Canonical Specs (docs/specs/)"
        IDX["index.md<br/>Spec area catalog"]
        SPEC1["brainstorm/spec.md"]
        SPEC2["change-lifecycle/spec.md"]
        SPEC3["memory/spec.md"]
        SPEC4["search/spec.md"]
        SPECN["... 13 total areas"]
    end

    subgraph "Delta Specs (per change)"
        DELTA["docs/changes/<name>/specs/<area>/spec.md"]
        ADDED["## ADDED Requirements"]
        MODIFIED["## MODIFIED Requirements"]
        REMOVED["## REMOVED Requirements"]
        DELTA --> ADDED & MODIFIED & REMOVED
    end

    subgraph "Archive Flow"
        ARCHIVE["cx change archive"]
        BOOTSTRAP["deltaToCanonical()<br/>ADDED → canonical spec"]
        MERGE["Planner merge prompt<br/>delta + canonical → merged"]
    end

    DELTA -->|"on archive"| ARCHIVE
    ARCHIVE -->|"if no canonical exists"| BOOTSTRAP
    ARCHIVE -->|"if canonical exists"| MERGE
    BOOTSTRAP --> SPEC1
    MERGE --> SPEC1
```

`cx change spec-sync <name>` generates a merge prompt (not automated) that assembles both the delta and canonical spec for an AI agent to produce the merged result. The `synced: true` flag is added to delta frontmatter after merge.

---

## Brainstorm & Masterfile Flow

```mermaid
flowchart LR
    NEW["cx brainstorm new <name>"] -->|creates| MF["docs/masterfiles/<name>.md"]
    MF -->|"agent/dev refines"| MF2["Filled masterfile:<br/>Problem, Vision,<br/>Open Questions,<br/>Constraints, Notes"]
    MF2 -->|"cx decompose <name>"| CHG["docs/changes/<name>/<br/>proposal.md, design.md,<br/>tasks.md, specs/"]
    MF2 -->|"archived to"| ARCH["docs/archive/<br/>YYYY-MM-DD-masterfile-<name>.md"]

    style MF fill:#fff3e0,color:#000
    style CHG fill:#e8f5e9,color:#000
    style ARCH fill:#e0e0e0,color:#000
```

Masterfile names must be kebab-case, max 40 characters. The `fileModified()` function detects whether a masterfile has been filled by comparing content against the template (stripping frontmatter).

---

## Worktree-Based Parallel Execution

### The Problem

In BUILD mode, independent tasks can be executed in parallel to reduce wall-clock time. However, without isolation, executors write to the same working tree and risk file-level conflicts, partial-state corruption, and interleaved test failures that are difficult to attribute to a single task.

### The Solution

Each independent task gets its own git worktree under `.cx/worktrees/`. Every executor operates on a dedicated branch in its own directory, with no shared file-system state during execution. When all parallel executors complete, a Merger agent integrates the branches back into the main tree and resolves any conflicts. A single cleanup command removes all worktrees created for a given change.

### Execution Flow

```mermaid
flowchart TD
    PLAN[Master identifies independent tasks]
    PLAN --> WT1["cx worktree create <change>-task-1"]
    PLAN --> WT2["cx worktree create <change>-task-2"]

    subgraph "Parallel Execution"
        WT1 --> E1["Executor 1<br/>(worktree 1 / branch <change>-task-1)"]
        WT2 --> E2["Executor 2<br/>(worktree 2 / branch <change>-task-2)"]
    end

    E1 --> MERGE
    E2 --> MERGE
    MERGE[Merger integrates branches into main tree]
    MERGE --> CLEAN["cx worktree cleanup <change>"]
    CLEAN --> REVIEW[Reviewer validates merged state]
```

### CLI Commands

| Command | What It Does |
|---------|-------------|
| `cx worktree create <branch-name>` | Creates a git worktree + branch under `.cx/worktrees/<branch-name>`; the branch is created off the current HEAD |
| `cx worktree list` | Lists all active worktrees: branch name, filesystem path, and HEAD commit |
| `cx worktree cleanup <change-name>` | Removes all worktrees and branches whose names are prefixed with `<change-name>` |

### Execution Paths

| Scenario | Strategy |
|----------|----------|
| **2 or more independent tasks** | Worktree-based parallel execution (default) |
| **Dependent tasks only** | Sequential execution on the main working tree (fallback) |
| **Mixed dependency graph** | Parallel groups execute via worktrees; each group's result is merged before the next dependent group begins |

The Master determines task independence from the dependency graph in `tasks.md`. A task is independent if it has no predecessor tasks that are not yet complete.

---

## Skills System

Skills are the workflow definitions that tell the Master agent how to handle each session mode. There are **16 skills** embedded in the Go binary via `//go:embed`.

### Skill Format

```markdown
# Skill: cx-<name>
## Description    — what & when
## Triggers       — activation patterns
## Steps          — numbered workflow
## Rules          — constraints & guardrails
```

### Skill Lifecycle

```mermaid
flowchart LR
    EMB["Go source<br/>internal/skills/data/*.md"] -->|"go:embed"| BIN["cx binary"]
    BIN -->|"cx sync / cx init"| DISK[".claude/skills/<name>/SKILL.md"]
    DISK -->|"Claude Code reads"| AGENT["Master Agent"]

    style EMB fill:#e3f2fd,color:#000
    style BIN fill:#f3e5f5,color:#000
    style DISK fill:#e8f5e9,color:#000
```

Skills are **not user-editable** on disk. `cx sync` always overwrites them. Customization goes in `CLAUDE.md`, not skill files.

### All 16 Skills

| Skill | Mode | Purpose | Key Behavior |
|-------|------|---------|-------------|
| `cx-build` | BUILD | Full new feature lifecycle | Requirements → plan → decompose → implement → review → archive |
| `cx-continue` | CONTINUE | Resume existing work | Loads last session's `--next` field; picks up where left off |
| `cx-plan` | PLAN | High-level brainstorming | Clean-slate thinking; no implementation; creates masterfiles |
| `cx-fix` | FIX | Quick localized changes | Bypasses entire change lifecycle; Scout → Executor → optional Review |
| `cx-brainstorm` | - | Masterfile creation | Free-form ideation; decomposes to change when ready |
| `cx-change` | - | Change CRUD & lifecycle | `cx instructions` before every artifact; enforces dependency graph |
| `cx-review` | - | Code/doc quality gate | Read-only; structured pass/fail report; blocks archive on CRITICAL |
| `cx-scout` | - | Codebase exploration | Read-only; returns findings to Master; Master evaluates and saves non-trivial discoveries as observations |
| `cx-prime` | - | Session context loading | Disposable; loads mode-specific memory; 500-800 token output |
| `cx-memory` | - | Memory CRUD & sync | Push/pull team sync; `--next` is critical session bridge |
| `cx-supervise` | - | Multi-agent coordination | Task distribution; progress tracking; result aggregation |
| `cx-refine` | - | Iterative doc improvement | Review cycles until satisfied |
| `cx-contract` | - | API contract management | Backward compat checks; also serves as Contractor (foreman) |
| `cx-conflict-resolve` | - | Memory conflict resolution | Spawned by Primer (not Master); requires developer input |
| `cx-doctor` | - | Project health checks | 7 check groups; `--fix` with developer approval |
| `cx-linear` | - | Linear issue tracking | Requires Linear MCP server |

### Skill Interaction Map

```mermaid
graph TB
    subgraph "BUILD Chain"
        BP[cx-prime] --> BB[cx-brainstorm]
        BB --> BC[cx-change<br/>decompose]
        BC --> BCD[cx-change<br/>design/tasks]
        BCD --> BS[cx-scout<br/>per-task mapping]
        BS --> BCE[Executor<br/>implementation]
        BCE --> BR[cx-review]
        BR --> BCA[cx-change<br/>archive]
    end

    subgraph "CONTINUE Chain"
        CP[cx-prime<br/>session recovery] --> CC[cx-change<br/>status check]
        CC --> CS[cx-scout<br/>if needed]
        CS --> CE[Executor]
        CE --> CR[cx-review]
    end

    subgraph "PLAN Chain"
        PB[cx-brainstorm] --> PR[cx-refine<br/>iterate]
        PR -->|"explicit approval"| BC
    end

    subgraph "FIX Chain"
        FS[cx-scout] --> FE[Executor]
        FE -->|optional| FR[cx-review]
    end

    subgraph "Cross-cutting"
        MEM[cx-memory<br/>session bridge]
        DOC[cx-doctor]
        LIN[cx-linear]
        CONF[cx-conflict-resolve]
    end

    BCA -.->|"session summary --next"| MEM
    MEM -.->|"CONTINUE recovery"| CP
    BP -.->|"if conflicts.json"| CONF
```

---

## TUI Dashboard

The TUI is a Bubble Tea application with 8 tabs, polling data every 5 seconds from all three SQLite databases.

### TUI Architecture

```mermaid
graph TB
    subgraph "AppModel (app.go)"
        TAB[Tab Router]
        POLL[5s Poll Cycle<br/>tea.Tick]
        LOADER[data.Loader<br/>wraps 3 DBs]
    end

    POLL --> LOADER
    LOADER --> MSG[DataLoadedMsg]
    MSG --> TAB

    TAB --> HOME[1. Home<br/>Overview + stats]
    TAB --> MEMS[2. Memories<br/>FTS5 search + filters]
    TAB --> SESS[3. Sessions<br/>Session summaries]
    TAB --> RUNZ[4. Runs<br/>Agent dispatch log]
    TAB --> SYNC2[5. Sync<br/>Push/pull controls]
    TAB --> NOTES2[6. Notes<br/>Personal notes]
    TAB --> GRAPH[7. Graph<br/>Memory link visualization]
    TAB --> CROSS[8. Cross-Project<br/>Federated search]

    MEMS -->|enter| DETAIL[Detail Overlay]
    GRAPH -->|enter| DETAIL

    style HOME fill:#fff9c4,color:#000
    style MEMS fill:#c8e6c9,color:#000
    style SESS fill:#b3e5fc,color:#000
    style RUNZ fill:#f8bbd0,color:#000
    style SYNC2 fill:#d1c4e9,color:#000
    style NOTES2 fill:#ffe0b2,color:#000
    style GRAPH fill:#b2dfdb,color:#000
    style CROSS fill:#f0f4c3,color:#000
```

### Layout

- **Wide mode** (>= 100 columns): two-pane layout (list + preview)
- **Narrow mode** (< 100 columns): single-pane
- Tab bar (1 row) + content area + status bar (1 row)
- Theming: auto-detects terminal background → Catppuccin Mocha (dark) or Latte (light)

### Key Interactions

| Key | Action |
|-----|--------|
| `tab` / `shift+tab` | Navigate between tabs |
| `j` / `k` or arrows | Navigate list items |
| `enter` | Open detail overlay or navigate to linked memory |
| `/` | Start FTS5 search (Memories tab) |
| `d` | Deprecate a memory |
| `p` / `l` | Push / Pull (Sync tab) |
| `q` / `ctrl+c` | Quit |

---

## Internal Package Map

```mermaid
graph LR
    subgraph "cmd/ (CLI Layer)"
        ROOT[root.go<br/>Cobra setup]
        CMD_CH[change.go]
        CMD_MEM[memory.go]
        CMD_BR[brainstorm.go]
        CMD_DOC[doctor.go]
        CMD_INIT[init.go]
        CMD_SYNC[sync.go]
        CMD_DASH[dashboard.go]
        CMD_AGT[agentrun.go]
        CMD_INSTR[instructions.go]
        CMD_DIS[disable.go]
    end

    subgraph "internal/ (Business Logic)"
        PKG_PROJ[project/<br/>Git, scaffolding, registry]
        PKG_MEM[memory/<br/>3 DBs, FTS5, CRUD]
        PKG_CHG[change/<br/>Lifecycle, templates]
        PKG_BR[brainstorm/<br/>Masterfiles]
        PKG_AGT[agents/<br/>Config generation]
        PKG_SKL[skills/<br/>Embedded .md files]
        PKG_TPL[templates/<br/>Embedded templates]
        PKG_CFG[config/<br/>cx.yaml parser]
        PKG_INS[instructions/<br/>Artifact prompts]
        PKG_DOC[doctor/<br/>Health checks]
        PKG_VER[verify/<br/>Verification]
        PKG_TUI[tui/<br/>Bubble Tea app]
        PKG_UI[ui/<br/>CLI output, theme]
        PKG_DATA[tui/data/<br/>DB loader for TUI]
    end

    CMD_CH --> PKG_CHG
    CMD_MEM --> PKG_MEM
    CMD_BR --> PKG_BR
    CMD_DOC --> PKG_DOC
    CMD_INIT --> PKG_PROJ & PKG_AGT
    CMD_SYNC --> PKG_AGT & PKG_SKL
    CMD_DASH --> PKG_TUI
    CMD_INSTR --> PKG_INS
    CMD_AGT --> PKG_MEM

    PKG_CHG --> PKG_TPL
    PKG_BR --> PKG_TPL & PKG_CHG
    PKG_INS --> PKG_CFG & PKG_TPL
    PKG_TUI --> PKG_DATA --> PKG_MEM
    PKG_TUI --> PKG_UI
    PKG_AGT --> PKG_SKL & PKG_TPL
    PKG_DOC --> PKG_PROJ & PKG_MEM & PKG_AGT & PKG_SKL
```

---

## Build & Distribution

```mermaid
flowchart LR
    SRC[Go Source] -->|"make build"| BIN["cx binary<br/>-ldflags Version=dev"]
    SRC -->|"goreleaser"| RELEASE["Releases:<br/>darwin/amd64<br/>darwin/arm64<br/>linux/amd64<br/>linux/arm64"]
    RELEASE --> TARBALL[".tar.gz archives"]
    RELEASE --> FORMULA["Formula/ PR<br/>Homebrew tap"]

    style RELEASE fill:#e8f5e9,color:#000
```

- **Module**: `github.com/AngelMaldonado/cx`
- **Go version**: 1.25
- **Key deps**: Cobra (CLI), Bubble Tea + Lip Gloss + Glamour (TUI), modernc SQLite (pure Go, no CGO)
- **Version injection**: `cmd.Version` set via `-ldflags` at build time
- **All file writes**: atomic via `.tmp` → `os.Rename()` pattern

---

## Key Design Decisions

### 1. Master Never Touches Code
The Master agent's context window is always loaded — it persists the entire conversation. Every token it consumes survives. By delegating all code reading/writing to disposable subagents, the Master stays lean and avoids context compaction.

### 2. `docs/` is the Source of Truth, SQLite is Cache
All canonical data lives in markdown files committed to git. SQLite databases are local query caches that can be rebuilt from markdown via `cx memory pull` and `RebuildFromMarkdown()`. This means team sync happens through git, not a separate sync protocol.

### 3. FIX Mode is an Intentional Bypass
FIX mode deliberately skips Primer, Planner, memory writes, and change docs. A scope guard redirects to BUILD if the fix grows beyond a single-file localized change. This prevents FIX from becoming an untracked refactor path.

### 4. The `--next` Field is the Only Session Bridge
The `next` field in `cx memory session` summaries is how CONTINUE mode recovers state across sessions. Without it, the next session has no bridge to prior work. Both `cx-continue` and `cx-memory` skills call this out as critical.

### 5. Skills are Compiled, Not Editable
Skill files are embedded in the Go binary via `//go:embed` and written to disk by `cx sync`. They are always overwritten on upgrade. Developer customization goes in `CLAUDE.md`, not skill files.

### 6. Conflict Resolution Happens During Priming
The Conflict Resolver is spawned by Primer (not Master) when `.cx/conflicts.json` exists. This means conflicts are surfaced during context loading, before the Master classifies the session mode.

### 7. Atomic Writes Everywhere
All file writes in the codebase use a `.tmp` → `os.Rename()` pattern to prevent partial writes. This is especially important for SQLite-adjacent markdown files that represent canonical state.

### 8. Disable/Enable as Safety Valve

The disable/enable mechanism preserves the user's original agent configuration across the cx lifecycle:

- **`cx init`**: before writing any agent files, snapshots the user's pre-existing agent directories (`.claude/`, `.gemini/`, `.codex/`) to `~/.cx/agent-backups/<project-id>/pre-init/`. Only the first init captures this baseline; subsequent inits do not overwrite it.
- **`cx disable`**: removes all cx-managed files (config file, skill directories, subagent files) via `RemoveCXManagedFiles()`, then restores the pre-init snapshot if one exists. This leaves the user in their original pre-cx state. Creates a `~/.cx/disabled` sentinel.
- **`cx enable`**: removes the sentinel, then runs a full sync (equivalent to `cx sync`) — calls `WriteConfigFile` + `WriteSkills` + `WriteSubagents` for each detected agent. No backup-restore involved.

All commands check the sentinel in `PersistentPreRun` and refuse to run while disabled (except `enable`, `disable`, `version`).
