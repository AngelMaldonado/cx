# Delta Spec: init

Change: spec-system-evolution

## ADDED Requirements

### REQ-NEW-001: cx.yaml Creation
`cx init` MUST create a `cx.yaml` file at the project root from the embedded template. This step MUST occur after DIRECTION.md setup (step 5 in the current sequence) and before git hook installation. The file MUST be created with commented template content explaining each field. `cx init` MUST skip this step if `cx.yaml` already exists (idempotent behavior, consistent with all other init steps).

### REQ-NEW-002: cx.yaml in Init Summary
The `cx init` summary output MUST include a `cx.yaml` line indicating the file was created, e.g., `cx.yaml    project config created`.

## MODIFIED Requirements

### Idempotency Table
Previously: the idempotency table listed docs/ files, agent directories, skill files, git hooks, `.cx/` directory, DIRECTION.md, project registry entry, and auto-update preference.
Now: the table MUST also include `cx.yaml` with the rule: if already exists, skip (never overwrite — user-owned after creation).

### Step Sequence
Previously: `cx init` had 8 steps (verify git → create docs/ → create .cx/ → agent selection → DIRECTION.md → git hooks → register globally → check MCP).
Now: `cx init` has 9 steps. A new step 6 "Create cx.yaml" is inserted between DIRECTION.md setup (step 5) and git hook installation (step 6, now step 7). All subsequent step numbers shift accordingly.

## REMOVED Requirements

None. All existing `cx init` behaviors are preserved.
