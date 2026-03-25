---
name: disable-enable
type: proposal
---

## Problem

Developers sometimes want a raw Claude Code session without cx's orchestration overhead — no Master agent, no modes, no skills, no change lifecycle. Currently the only way to accomplish this is to manually delete or rename `CLAUDE.md` (or the equivalent for other agents), which is error-prone, loses the original content, and modifies a checked-in file. There is no first-class mechanism for temporarily suspending cx across all registered projects.

## Approach

Introduce two new top-level commands, `cx disable` and `cx enable`, that toggle a global disabled state using a sentinel file at `~/.cx/disabled`. When disabled:

1. All agent config files (`CLAUDE.md`, `GEMINI.md`, `AGENTS.md`) across every registered project are replaced with a minimal stub that tells the developer cx is disabled. The originals are backed up to `~/.cx/agent-backups/`.
2. Every cx command except `enable`, `disable`, and `version` is blocked by a `PersistentPreRun` hook on the root Cobra command.

`cx enable` reverses the process: removes the sentinel, restores all config files from backups, and deletes the backups.

## Scope

**In scope:**
- `cmd/disable.go` — new `cx disable` command
- `cmd/enable.go` — new `cx enable` command
- `cmd/root.go` — add `PersistentPreRun` gating; register the two new commands
- `internal/project/disable.go` — helper functions: `IsDisabled`, `SetDisabled`, `ClearDisabled`, `BackupAgentConfig`, `RestoreAgentConfigs`, `ExportProjectID`
- Global scope only: one `~/.cx/disabled` sentinel affects all projects on the machine

**Out of scope:**
- Per-project disable (deferred; can be added later as `~/.cx/disabled-<project-id>`)
- `cx status` command showing enabled/disabled state (future work)
- Automatic git stash or commit of the stub CLAUDE.md
- `.gitignore` modifications

## Affected Specs

- **init** — the lifecycle of `CLAUDE.md` (and sibling config files) is covered in the init spec. This change adds a new disable/enable lifecycle that suspends and restores those files. A delta spec for the `init` area is required to document the disable/enable behavior.
