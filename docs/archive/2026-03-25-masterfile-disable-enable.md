---
name: disable-enable
type: masterfile
---

## Problem

Developers sometimes want a raw Claude Code session without cx's orchestration overhead — no Master agent, no modes, no skills, no change lifecycle. Currently the only way to accomplish this is to manually delete or rename CLAUDE.md, which is error-prone, loses state, and involves modifying a checked-in file. There is no first-class mechanism for temporarily suspending cx.

## Context

### How CLAUDE.md works today

CLAUDE.md is the primary injection point for cx in Claude Code. It lives at the repository root and is checked into git. `cx init` and `cx sync` write it via `agents.WriteConfigFile()`, which reads from `internal/templates/agents/config.md` and substitutes agent name, skills table, and skills directory. The file is re-generated on every sync, overwriting any manual edits.

Claude Code reads CLAUDE.md automatically at session start. If the file is absent or empty, Claude Code operates as a plain LLM assistant without any cx framework instructions. This is the "disabled" state we want to achieve.

### Global state location

The `~/.cx/` directory already exists as the global cx home (`project.GlobalCXDir()`). It stores `preferences.json`, `credentials.json`, `index.db`, and `projects.json`. A global disable flag fits naturally here as `~/.cx/disabled` (a sentinel file whose presence means "cx is disabled").

### Command gating

The Cobra CLI in `cmd/root.go` currently has no `PersistentPreRun` on the root command. All subcommands are registered in `rootCmd`'s `init()` and run independently. A root-level `PersistentPreRun` would intercept every subcommand call, making it the correct place to gate commands when disabled.

### CLAUDE.md as a generated artifact

CLAUDE.md is written by `agents.WriteConfigFile()`, which uses `atomicWriteAgent()`. When `cx init` or `cx sync` writes it, the content is generated from the embedded template. The current design does NOT write CLAUDE.md on every command — only on `init` and `sync`. This means:

- We cannot rely on cx to "skip" writing CLAUDE.md at runtime (it only writes on explicit commands)
- The mechanism must directly clear/restore the file on `cx disable` / `cx enable`
- The file must be absent or empty for Claude Code not to use it

### Git implications

CLAUDE.md is tracked by git. If we empty it, `git status` will show a modification. This is an intentional side effect — the developer is choosing to step out of cx mode. We do NOT want to stash or commit the change automatically; those are destructive or permanent.

The cleanest approach: when disabling, write a minimal CLAUDE.md placeholder ("cx is disabled — run `cx enable` to restore") and back up the original content to `~/.cx/claude-md-backup/<project-id>.md`. On enable, restore from the backup.

Backup is stored globally in `~/.cx/` (outside the repo) to avoid polluting the git index with backup files. The backup is keyed by project path hash (same `projectID()` function already in `registry.go`).

The per-project CLAUDE.md backup is scoped by project, so multiple projects can be independently enabled/disabled.

## Direction

### State mechanism

A sentinel file `~/.cx/disabled` signals global disable. Its presence = disabled; absence = enabled. This is the simplest possible state — no JSON parsing, no config fields, just a file existence check. The file can optionally contain a timestamp and reason for auditing.

Global scope is intentional: `cx disable` affects all projects on the machine simultaneously. If per-project disable is needed in the future, it can be added as `~/.cx/disabled-<project-id>`.

### CLAUDE.md handling

On `cx disable`:
1. For each registered project in the registry that has a CLAUDE.md / GEMINI.md / AGENTS.md:
   - Read the current content
   - Write the content to `~/.cx/agent-backups/<project-id>-<config-file-name>` (e.g., `~/.cx/agent-backups/<id>-CLAUDE.md`)
   - Overwrite the config file with a minimal stub:
     ```
     <!-- cx disabled — run `cx enable` to restore -->
     ```
2. Create `~/.cx/disabled`

On `cx enable`:
1. Remove `~/.cx/disabled`
2. For each registered project that has a backup in `~/.cx/agent-backups/`:
   - Restore the original content from the backup
   - Delete the backup file

The stub content is a single HTML comment so it is invisible to Claude Code (Claude ignores comments-only CLAUDE.md files — or at minimum, provides no cx instructions).

Alternative considered: rename `CLAUDE.md` to `CLAUDE.md.cx-disabled`. Rejected because: (a) Claude Code might pick up other `.md` files in some configurations, (b) the rename is visible in `git status` as a deletion, which is more disruptive than a modification, (c) restoration requires a rename back, adding complexity.

Alternative considered: emptying CLAUDE.md entirely. Accepted as equivalent to the stub — but a comment stub is preferable for discoverability (developer opens the file and immediately knows what happened).

### Command gating

Add a `PersistentPreRun` to `rootCmd` in `cmd/root.go`. This function:
1. Checks if `~/.cx/disabled` exists
2. If yes, checks if the current subcommand is in the allowlist: `enable`, `disable`, `version`
3. If not in the allowlist, prints: `cx is disabled. Run 'cx enable' to restore.` and exits with code 1

The allowlist check reads the first positional argument from `os.Args` (the subcommand name) before Cobra parsing, or uses `cmd.Use` in the `PersistentPreRun` callback — the latter is cleaner since Cobra has already parsed by then.

The `PersistentPreRun` approach is correct because it runs before every subcommand's `RunE`, but Cobra calls it with the leaf command, so we check the command name or walk up to root.

### New commands

`cx disable` — sets the disabled flag, backs up and stubs all config files for registered projects.
`cx enable` — clears the flag, restores all config files from backups.

Both commands are registered on `rootCmd`. They do NOT require being inside a git repo (unlike most cx commands) since they operate on global state in `~/.cx/`.

### `cx status` integration (optional, out of scope for this change)

If a `cx status` command is added in the future, it should show the enabled/disabled state. This is out of scope for the current change but worth noting.

## Open Questions

None. All design questions have been resolved above.

## Observations

- `project.GlobalCXDir()` already creates `~/.cx/` idempotently — use it for creating the backups subdirectory.
- `project.projectID()` is unexported but its logic (SHA-256 hash of path, first 12 hex chars) is simple enough to replicate or export.
- The `init/spec.md` spec covers CLAUDE.md creation. This change modifies the behavior of that file but does not change the init flow itself. A delta spec under the `init` area may be needed.
- `cmd/root.go`'s `rootCmd` has no `PersistentPreRun` today — adding one is safe as long as the allowlist is correct.
- The `agents.WriteConfigFile()` function always overwrites CLAUDE.md. If cx is disabled and the developer runs `cx sync`, the sync should be gated too (it's not in the allowlist), so CLAUDE.md won't be accidentally restored by sync while disabled. This is the correct behavior.

## Files to Modify

### `cmd/root.go`
- Add `PersistentPreRun` to `rootCmd` that checks for `~/.cx/disabled` and gates all commands except `enable`, `disable`, and `version`.
- Register `disableCmd` and `enableCmd`.

### `cmd/disable.go` (new file)
- Implement `cx disable`:
  - Call `project.GlobalCXDir()` to get `~/.cx/`
  - Iterate registered projects via `project.LoadRegistry()`
  - For each project, for each installed agent config file (CLAUDE.md, GEMINI.md, AGENTS.md):
    - Read current content
    - Write backup to `~/.cx/agent-backups/<project-id>-<config-file>`
    - Write stub content to the config file
  - Create `~/.cx/disabled` with timestamp
  - Print confirmation

### `cmd/enable.go` (new file)
- Implement `cx enable`:
  - Check `~/.cx/disabled` — if not present, print "cx is already enabled" and return
  - Remove `~/.cx/disabled`
  - For each backup file in `~/.cx/agent-backups/`:
    - Parse the project ID and config filename from the backup filename
    - Look up the project path from the registry
    - Restore the original content
    - Delete the backup file
  - Print confirmation

### `internal/project/disable.go` (new file)
- `IsDisabled() bool` — returns true if `~/.cx/disabled` exists
- `SetDisabled(reason string) error` — creates `~/.cx/disabled` with timestamp + reason
- `ClearDisabled() error` — removes `~/.cx/disabled`
- `BackupAgentConfig(projectPath, configFile, content string) error` — writes to `~/.cx/agent-backups/<id>-<configFile>`
- `RestoreAgentConfigs() ([]RestoredFile, error)` — reads all backups, restores files, deletes backups
- `ExportProjectID(path string) string` — exports the `projectID()` hash function (currently unexported in registry.go)

## Risks

1. **Multi-project backup/restore mismatch**: If a project is removed from the registry while cx is disabled, its config file won't be restored. Mitigation: the backup file still exists in `~/.cx/agent-backups/`; running `cx enable` logs a warning for any backup whose project path is no longer in the registry, and offers to restore by path anyway.

2. **CLAUDE.md changed externally while disabled**: If the developer modifies CLAUDE.md while cx is disabled, `cx enable` will overwrite those changes with the backup. Mitigation: document this clearly. The stub comment warns the developer not to edit the file while disabled.

3. **cx sync run while disabled**: gated by `PersistentPreRun` — will be blocked. No risk.

4. **Concurrent cx commands**: Two simultaneous `cx disable` calls could both read backups and overwrite each other. Mitigation: this is an extremely rare edge case (CLI tool, not a daemon); no file locking needed for v1.

5. **Home directory not writable**: `GlobalCXDir()` would fail. This is an existing constraint for all global cx operations; not new.

6. **Git noise**: The stub content in CLAUDE.md will show as a modification in `git status`. This is intentional and expected. The developer should not commit this change. We may want to add a note in the stub about not committing. Per-project `.gitignore` modification is out of scope.

## Testing

1. **Unit tests for `internal/project/disable.go`**:
   - `TestIsDisabled`: create/remove `~/.cx/disabled`, verify return values
   - `TestSetDisabled` / `TestClearDisabled`: round-trip
   - `TestBackupAgentConfig` / `TestRestoreAgentConfigs`: write backup, restore, verify content equality and file deletion

2. **Integration tests for `cx disable` / `cx enable`**:
   - Set up a temp git repo with a mock project registry entry
   - Run `cx disable`: verify `~/.cx/disabled` created, CLAUDE.md stubbed, backup exists
   - Run `cx enable`: verify `~/.cx/disabled` removed, CLAUDE.md restored to original, backup deleted
   - Run any other cx command while disabled: verify exit code 1 and "cx is disabled" message

3. **Edge cases**:
   - `cx enable` when already enabled: prints message, exits 0
   - `cx disable` when already disabled: prints message, exits 0 (idempotent)
   - Missing project path at restore time: warning printed, backup retained

4. **Manual verification**:
   - Open a Claude Code session after `cx disable` — verify no cx system prompt behavior
   - Run `cx enable` — re-open Claude Code session, verify cx behavior resumes

## References

- `internal/project/registry.go` — `GlobalCXDir()`, `LoadRegistry()`, `projectID()` patterns
- `internal/agents/agents.go` — `All()` for enumerating config files; `WriteConfigFile()` for the generation path being bypassed
- `cmd/root.go` — where `PersistentPreRun` will be added
- `docs/specs/init/spec.md` — covers CLAUDE.md creation lifecycle (delta spec may be needed)
