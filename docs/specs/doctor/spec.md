# Spec: Doctor

`cx doctor` validates the health of a CX project. It checks four areas, reports issues, and offers to fix auto-fixable problems.

---

## Usage

```bash
cx doctor          # Report issues
cx doctor --fix    # Report issues, then auto-repair what's fixable
```

---

## Check Areas

### 1. docs/ Structure Integrity

Validates that required files and directories exist:

| Check | Severity | Auto-fixable |
|-------|----------|-------------|
| `docs/overview.md` exists and has an H1 | Error | No |
| `docs/architecture/index.md` exists | Warning | No |
| `docs/specs/index.md` exists | Warning | No |
| `docs/memories/DIRECTION.md` exists and is non-empty | Error | Yes (create default template) |
| `docs/memories/observations/` directory exists | Warning | Yes (mkdir) |
| `docs/memories/decisions/` directory exists | Warning | Yes (mkdir) |
| `docs/memories/sessions/` directory exists | Warning | Yes (mkdir) |
| Active changes have all three files (proposal, design, tasks) | Warning | No |
| No orphan delta specs (delta references a spec area that doesn't exist and isn't being created) | Warning | No |

### 2. Memory File Health

Validates all files in `docs/memories/{observations,decisions,sessions}/`:

| Check | Severity | Auto-fixable |
|-------|----------|-------------|
| File has valid YAML frontmatter | Error | No |
| File has an H1 title | Error | No |
| Required frontmatter fields present (`author`, `created`/`started`) | Error | No |
| Observation has valid `type` (bugfix, discovery, pattern, context) | Error | No |
| Decision has all five required sections (Context, Outcome, Alternatives, Rationale, Tradeoffs) | Warning | No |
| Decision `status` in file is valid (`active` or `cancelled` only — `superseded` is set by index, never written in files) | Error | No |
| `deprecates` slug references an existing file | Warning | No |
| `author` matches a known git committer in this repo | Warning | No |

### 3. Index Health

Validates the FTS5 index cache:

| Check | Severity | Auto-fixable |
|-------|----------|-------------|
| `.cx/.index.db` exists | Warning | Yes (`cx index rebuild`) |
| Index is not stale (all docs/ files have mtimes older than last index build) | Warning | Yes (`cx index rebuild`) |

### 4. Git Hooks

Validates that CX git hooks are installed:

| Check | Severity | Auto-fixable |
|-------|----------|-------------|
| `.git/hooks/post-merge` exists and contains `cx index rebuild` | Warning | Yes (install) |
| `.git/hooks/post-checkout` exists and contains `cx index rebuild` | Warning | Yes (install) |

When auto-fixing, the binary checks if an existing hook file is present. If so, it appends the CX hook logic rather than overwriting.

### 5. MCP Server Config

Validates external dependencies:

| Check | Severity | Auto-fixable |
|-------|----------|-------------|
| `.mcp.json` exists | Warning | No |
| Linear MCP server configured in `.mcp.json` | Warning | No |

MCP checks are warnings, not errors — a project can function without Linear integration.

### 6. Skill Files

Validates that agent skill files are up-to-date:

| Check | Severity | Auto-fixable |
|-------|----------|-------------|
| All expected cx-* skill files exist for each detected agent | Warning | Yes (`cx sync`) |
| Skill files match the binary's built-in versions | Warning | Yes (`cx sync`) |
| Each skill has all four required sections (Description, Triggers, Steps, Rules) | Warning | No |

---

## Output Format

```
cx doctor

  docs/ structure
    ✓ overview.md exists
    ✓ architecture/index.md exists
    ✓ specs/index.md exists
    ✓ memories/DIRECTION.md exists
    ✗ change "refactor-mqtt" missing design.md
    ✗ change "refactor-mqtt" missing tasks.md

  memory files
    ✓ 42 observations parsed
    ✓ 8 decisions parsed
    ✓ 15 sessions parsed
    ⚠ decision "ble-just-works" missing ## Tradeoffs section
    ⚠ observation deprecates slug "mqtt-drops" not found

  git hooks
    ⚠ post-merge hook not installed
    ⚠ post-checkout hook not installed

  mcp servers
    ✓ Linear MCP server configured

  ─────────────────────────────
  2 errors, 4 warnings

  Auto-fixable:
    [1] Install post-merge hook
    [2] Install post-checkout hook

  Fix these? [y/n]
```

---

## Severity Levels

| Level | Meaning | Effect |
|-------|---------|--------|
| **Error** | Something is broken — commands may fail | Exit code 1 |
| **Warning** | Something is suboptimal — commands still work | Exit code 0 |

If there are only warnings (no errors), `cx doctor` exits with code 0. Any error means exit code 1.

---

## Auto-Fix Flow

**Without `--fix`** (default): report-only mode. Lists all issues with their severity and whether they're auto-fixable. Does not modify anything.

**With `--fix`**:

1. Binary prints the full report first
2. Lists auto-fixable items with numbers
3. Asks `Fix these? [y/n]`
4. On `y`: applies all fixes using atomic writes (temp file + rename):
   - Creates missing directories
   - Creates default DIRECTION.md template
   - Installs/updates git hooks (appends if existing hooks are present)
   - Runs `cx index rebuild` for stale/missing index
   - Runs `cx sync` for outdated skill files
5. Re-runs the affected checks and prints updated status
6. On `n`: exits without changes

The binary never auto-fixes without asking. The developer always sees what will be fixed before confirming.

---

## When to Run

- After `cx init` — verify the scaffolding is complete
- Before `cx change archive` — the archive command runs doctor checks internally and refuses to proceed on errors
- After major refactors — check that docs/ structure is still valid
- When something feels off — general health check
- The `cx-doctor` skill teaches the agent to run it proactively
