# Spec: cx init

`cx init` bootstraps a project for CX. It's the only setup command a developer runs. It creates the full `docs/` scaffolding, installs git hooks, generates agent skills, and interactively configures the project's memory direction.

---

## Usage

```bash
cx init
```

No flags. Fully interactive. Idempotent — safe to run again on an already-initialized project (only creates what's missing, never overwrites).

---

## What It Does

### 1. Verify Git Repository

The binary checks that the current directory is inside a git repository. If not, it exits with an error:

```
Error: not a git repository. CX requires git for memory sync and version history.
Run 'git init' first.
```

It also reads `git config user.name` to identify the author. If not set, it exits with an error:

```
Error: git user.name not configured.
Run 'git config user.name "Your Name"' first.
```

### 2. Create docs/ Scaffolding

Creates the full directory structure:

```
docs/
├── overview.md                    # Template with H1 + section headings
├── architecture/
│   └── index.md                   # Template with Tech Stack table
├── specs/
│   └── index.md                   # Empty spec registry
├── memories/
│   ├── DIRECTION.md               # Generated from interactive setup (step 5)
│   ├── observations/
│   ├── decisions/
│   └── sessions/
├── masterfiles/                   # Empty, ready for brainstorming
├── changes/                       # Empty, ready for active work
└── archive/                       # Empty, ready for completed changes
```

If any of these already exist, they are skipped (never overwritten).

### 3. Create .cx/ Cache Directory

```
.cx/
└── .gitignore                     # Contains: *
```

The `.gitignore` ignores everything in `.cx/` — it's purely local cache. The `.index.db` file is created on first `cx` command that needs it.

### 4. Agent Selection

The binary asks the developer which agents to generate skills for:

```
Which coding agents do you use?

  [1] Claude Code (.claude/)
  [2] Gemini (.gemini/)
  [3] Codex (.codex/)

Select (comma-separated, e.g., 1,2):
```

For each selected agent, the binary:
- Creates the agent directory (`.claude/`, `.gemini/`, `.codex/`)
- Writes the main config file (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md`)
- Copies static skill files to the skills directory

Skills are static files shipped with the binary — no project-specific generation.

### 5. Interactive DIRECTION.md Setup

The binary asks 2-3 questions to generate a tailored `docs/memories/DIRECTION.md`:

```
Let's configure what your agents should remember.

What type of project is this?
  [1] Web API / Backend
  [2] Frontend / UI
  [3] Firmware / Embedded
  [4] CLI / Developer Tool
  [5] Full-stack
  [6] Other

What matters most to your team? (pick up to 3)
  [1] Performance characteristics
  [2] External system constraints (APIs, hardware, protocols)
  [3] Security decisions and tradeoffs
  [4] User experience patterns
  [5] Data model and migration history
  [6] Infrastructure and deployment
  [7] Cross-team integration points
```

Based on answers, the binary generates a `DIRECTION.md` with project-relevant categories in the Always Save, Never Save, and Type Guidance sections. The developer can edit it further afterward.

### 6. Install Git Hooks

Installs `post-merge` and `post-checkout` hooks:

```
Installing git hooks...

  .git/hooks/post-merge    ✓ installed
  .git/hooks/post-checkout ✓ installed
```

The `post-merge` hook runs two commands:
1. `cx index rebuild` — refreshes the FTS5 search index with any new docs/ content from the merge
2. `cx conflicts detect` — scans for semantic conflicts between incoming and local memory entities, writing results to `.cx/conflicts.json` if any are found (see [conflict-resolution spec](../conflict-resolution/spec.md))

The `post-checkout` hook runs `cx index rebuild` only.

If hooks already exist, the binary asks before modifying:

```
  .git/hooks/post-merge already exists. Append CX hook? [y/n]
```

On `y`, it appends the CX hook logic. On `n`, it skips with a warning.

### 7. Register Project Globally

The binary adds the current project's absolute path to `~/.cx/projects.json`:

```json
{
  "projects": [
    "/Users/angel/dev/iot-platform",
    "/Users/angel/dev/mobile-app"
  ]
}
```

If this is the **first-ever `cx init`** (no `~/.cx/projects.json` exists), the binary also asks about auto-update:

```
Auto-update: cx can check for new versions weekly via Homebrew.
Enable auto-update? [y/n]
```

The preference is stored in `~/.cx/preferences.json`:

```json
{
  "auto_update_check": true,
  "last_update_check": "2026-02-26T10:00:00Z"
}
```

If enabled, `cx` checks `brew outdated cx` at most once per week on any command launch. It never updates automatically — just prints a notice if a new version is available.

### 8. Check MCP Dependencies

Checks `.mcp.json` for required MCP servers:

```
Checking MCP servers...

  ⚠ Linear MCP server not found in .mcp.json

  CX skills use the Linear MCP server for task tracking.
  Add the following to your .mcp.json:

  {
    "mcpServers": {
      "linear": {
        "command": "npx",
        "args": ["-y", "@linear/mcp-server"],
        "env": { "LINEAR_API_KEY": "<your-key>" }
      }
    }
  }
```

This is a warning, not a blocker. The project initializes even without Linear.

### 9. Summary

```
✓ CX initialized

  docs/           scaffolding created
  .cx/            cache directory created
  .claude/        skills generated (9 files)
  git hooks       installed
  DIRECTION.md    configured for: Firmware / Embedded
  project         registered in ~/.cx/projects.json
  auto-update     enabled (weekly check via Homebrew)

  Next steps:
    1. Fill docs/overview.md with your project description
    2. Fill docs/architecture/index.md with your tech stack
    3. Start a session — your agent will handle the rest

  Run 'cx doctor' anytime to check project health.
  Run 'cx upgrade' to update cx and sync all projects.
```

---

## Idempotency

`cx init` is safe to run multiple times:

| Resource | Already exists? | Action |
|----------|----------------|--------|
| docs/ files | Yes | Skip (never overwrite) |
| Agent directories | Yes | Skip, unless new skills are available |
| Skill files | Yes | Overwrite (skills are owned by the binary) |
| Git hooks | Yes | Ask before appending |
| .cx/ directory | Yes | Skip |
| DIRECTION.md | Yes | Skip (user-owned) |
| Project in ~/.cx/projects.json | Yes | Skip (already registered) |
| Auto-update preference | Yes | Skip (only asked on first-ever init) |

The only things that are always overwritten are skill files — they're static and owned by the binary.

---

## cx sync

Regenerates agent configs for the **current project**:

```bash
cx sync
```

This:
1. Re-copies all static skill files for all configured agents (overwrites existing)
2. Regenerates the main config files (CLAUDE.md, GEMINI.md, AGENTS.md)
3. Does NOT touch docs/, DIRECTION.md, or git hooks

---

## cx upgrade

Updates the binary and syncs all registered projects:

```bash
cx upgrade
```

This:
1. Runs `brew upgrade cx` to get the latest binary
2. Reads `~/.cx/projects.json` for all registered project paths
3. For each project: `cd <path> && cx sync`
4. Reports results:

```
cx upgrade

  Upgrading cx...
    cx 1.2.0 → 1.3.0 ✓

  Syncing projects...
    /Users/angel/dev/iot-platform     ✓ 9 skills updated
    /Users/angel/dev/mobile-app       ✓ 9 skills updated
    /Users/angel/dev/old-project      ⚠ directory not found (removed from registry)

  Done. 2 projects synced, 1 removed.
```

### Stale project handling

If a registered project path no longer exists (deleted, moved), `cx upgrade` removes it from `projects.json` and prints a warning. No error — just cleanup.

If a project path exists but is no longer a git repo or has no `docs/` directory, `cx upgrade` skips it with a warning.

---

## cx projects

Manage the global project registry:

```bash
cx projects                    # list all registered projects
cx projects remove <path>      # remove a project from the registry
```

Output:

```
cx projects

  Registered projects:
    /Users/angel/dev/iot-platform       cx v1.3.0   3 agents   last sync: 2h ago
    /Users/angel/dev/mobile-app         cx v1.3.0   1 agent    last sync: 2h ago
```

---

## cx disable

`cx disable` globally suspends cx across all registered projects.

**Behavior:**

- If `~/.cx/disabled` already exists: print `cx is already disabled.` and exit 0 (idempotent).
- Otherwise:
  1. For every project path in the global registry, detect which agent config files are installed (`CLAUDE.md`, `GEMINI.MD`, `AGENTS.md`).
  2. For each existing config file, back up its content to `~/.cx/agent-backups/<project-id>-<ConfigFile>`.
  3. Overwrite the config file with the minimal stub: `<!-- cx disabled — run 'cx enable' to restore. Do not edit this file while cx is disabled. -->`
  4. Create `~/.cx/disabled` with a JSON payload containing `disabled_at` (ISO timestamp) and `reason` (empty string in v1).
- Prints a summary of how many projects and config files were stubbed.
- Does NOT require a git repository (operates on global state).

---

## cx enable

`cx enable` restores cx across all registered projects.

**Behavior:**

- If `~/.cx/disabled` does not exist: print `cx is already enabled.` and exit 0 (idempotent).
- Otherwise:
  1. Remove `~/.cx/disabled`.
  2. Scan `~/.cx/agent-backups/` and restore each backup to its original config file path.
  3. Delete each backup file after successful restoration.
  4. If a backup's project path is no longer in the registry: print a warning to stderr (`warning: project path <path> not found; backup retained at <backup-path>`), retain the backup, and continue.
- Prints a summary of restored files and any warnings.
- Does NOT require a git repository.

---

## Command Gating While Disabled

When `~/.cx/disabled` exists, all `cx` subcommands except `enable`, `disable`, and `version` are blocked at the root command's `PersistentPreRun` hook:

```
cx is disabled. Run 'cx enable' to restore.
```

Exit code: 1.

The block applies to `cx sync` as well — a sync while disabled cannot accidentally overwrite stubbed config files with the full agent config.

---

## Global State Files

Two entries in `~/.cx/` support the disable/enable lifecycle:

| Path | Purpose |
|------|---------|
| `~/.cx/disabled` | Sentinel file; presence = disabled. JSON content: `{"disabled_at": "<ISO>", "reason": ""}` |
| `~/.cx/agent-backups/<id>-<ConfigFile>` | Per-project, per-agent config file backup. Keyed by `projectID(path)` (12-char SHA-256 hex prefix of absolute project path). |

---

## Effect on Agent Config Files (CLAUDE.md and Siblings)

When disabled, agent config files contain only the stub comment. The stub is intentionally visible in `git status` as a modification; developers should not commit it. The stub comment warns against editing the file while disabled.

When enabled, config files are restored to the content that was present at the time `cx disable` was run. Any edits made to a config file while cx was disabled are overwritten by the restore.
