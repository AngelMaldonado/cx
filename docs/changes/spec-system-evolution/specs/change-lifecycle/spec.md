# Delta Spec: change-lifecycle

Change: spec-system-evolution

## ADDED Requirements

### REQ-NEW-001: Artifact Frontmatter Headers
All change artifact templates (proposal.md, design.md, tasks.md, delta-spec.md, masterfile.md) MUST open with a YAML frontmatter block containing at minimum `name` and `type` fields. The H1 title line is removed. Content begins after the closing `---` of the frontmatter block.

### REQ-NEW-002: Spec Template — Purpose and REQ-NNN Format
The spec.md template MUST include: (1) a Purpose paragraph as the opening prose after frontmatter, (2) requirements structured as `### REQ-NNN: <Name>` subsections with RFC 2119 keywords (MUST/SHOULD/MAY), and (3) a Scenarios section that points to an optional `scenarios.md` file.

### REQ-NEW-003: Delta Spec — Scenarios Sections
The delta-spec.md template MUST include three additional sections: `## ADDED Scenarios`, `## MODIFIED Scenarios`, `## REMOVED Scenarios`. These sections are optional to fill but MUST appear in the template as designated locations for scenario changes.

### REQ-NEW-004: Project Config (cx.yaml)
A `cx.yaml` file at the project root MAY be present. When present, it MUST conform to the `cx/v1` schema with top-level fields: `schema`, `context`, and `rules`. The `rules` field is a map of artifact names to lists of rule strings. The binary MUST read `cx.yaml` via `internal/config.Load()` and return zero-value defaults when the file is absent.

### REQ-NEW-005: cx instructions Command
A new `cx instructions <artifact>` command MUST exist where artifact is one of: `proposal`, `specs`, `design`, `tasks`. The command MUST output to stdout: the artifact template, project context from `cx.yaml`, project rules for the artifact, dependency requirements from the static artifact graph, what the artifact unlocks, and the list of existing spec areas from `docs/specs/index.md`.

### REQ-NEW-006: Schema-Driven Artifact Dependency Graph
The artifact dependency graph MUST be defined as a static data structure in Go (`internal/instructions/graph.go`). The graph defines: proposal has no dependencies; specs and design both require proposal; tasks requires specs and design; verify requires tasks. This graph MUST be the authoritative source for `cx instructions` dependency output and `cx change status` state display.

### REQ-NEW-007: cx change verify Command
A new `cx change verify <name>` subcommand MUST exist. It MUST print to stdout: all delta specs for the change with REQ-NNN requirements extracted into a COMPLETENESS checklist, the change's proposal.md and design.md for intent context, and a structured template with COMPLETENESS, CORRECTNESS, and COHERENCE check dimensions. After agent review, the agent writes a `verify.md` to `docs/changes/<name>/verify.md`.

### REQ-NEW-008: Verify Gate on Archive
`cx change archive <name>` MUST check that `verify.md` exists and contains "PASS" with no CRITICAL issues before proceeding, unless `--skip-specs` is passed. If `verify.md` is absent or contains FAIL/CRITICAL, the archive MUST exit non-zero with a clear error message.

### REQ-NEW-009: --skip-specs Flag on Archive
`cx change archive <name>` MUST accept a `--skip-specs` flag. When passed: (1) delta spec validation is skipped entirely, (2) the verify gate is skipped, (3) the change archives immediately after proposal/design/tasks completeness check, and (4) the skip MUST be logged in the archive memory save observation.

### REQ-NEW-010: cx change spec-sync Command
A new `cx change spec-sync <name>` subcommand MUST exist. It MUST: (1) require only proposal.md to be filled (not design or tasks), (2) run the same agent-assisted merge flow as archive step 2 for each delta, (3) after the merge, set `synced: true` in each merged delta file's YAML frontmatter, and (4) NOT move the change to archive. `cx change archive` MUST skip deltas that have `synced: true` in their frontmatter.

### REQ-NEW-011: Enhanced cx change status State Display
`cx change status` MUST show a computed state for each artifact: BLOCKED (dependency not met), READY (deps complete, file at template-only state), IN-PROGRESS (file has content but artifact not verified/complete), DONE. It MUST also show the verify state per change (PENDING/PASS/FAIL) and indicate which delta specs are SYNCED vs. pending.

### REQ-NEW-012: Frontmatter-Aware Completeness Check
The completeness check in `internal/change/change.go` MUST strip the YAML frontmatter block (everything between the opening `---` and the closing `---` on its own line) before testing whether the remaining content equals the template stub. A file containing only frontmatter and whitespace MUST be treated as empty (template-only state).

### REQ-NEW-013: verify.md Template
A new `verify.md` template MUST exist at `internal/templates/docs/verify.md` with YAML frontmatter (`name`, `type: verify`) and sections: `## Result`, `## Completeness`, `## Correctness`, `## Coherence`, `## Issues`.

### REQ-NEW-014: cx.yaml Template
A new `cx.yaml` template MUST exist at `internal/templates/docs/cx.yaml` with commented explanations for each field. `cx init` MUST write this template to the project root if `cx.yaml` does not already exist.

## MODIFIED Requirements

### Change Directory Structure (proposal.md format)
Previously: proposal.md opened with `# Proposal: <change-name>`.
Now: proposal.md opens with YAML frontmatter (`name`, `type: proposal`) and no H1 title.

### Change Directory Structure (design.md format)
Previously: design.md opened with `# Design: <change-name>`.
Now: design.md opens with YAML frontmatter (`name`, `type: design`) and no H1 title.

### Change Directory Structure (tasks.md format)
Previously: tasks.md opened with `# Tasks: <change-name>`.
Now: tasks.md opens with YAML frontmatter (`name`, `type: tasks`) and no H1 title.

### Delta Spec Format
Previously: delta files opened with `# Delta: <spec-area>` and a `Change: <name>` line.
Now: delta files open with YAML frontmatter containing `name`, `area`, `type: delta-spec`, `change` fields. The `# Delta:` H1 and `Change:` line are removed. The three existing requirement sections (ADDED/MODIFIED/REMOVED) are retained. Three scenario sections are added (ADDED/MODIFIED/REMOVED Scenarios).

### Archive — Validation Step
Previously: validation checked proposal.md, design.md, tasks.md exist and are non-empty; verified Linear issues are done; parsed delta files for valid sections.
Now: same as before, PLUS checks `verify.md` exists and has PASS state (unless `--skip-specs` flag is set). The completeness check is frontmatter-aware.

### Archive — Lifecycle Diagram
Previously: lifecycle ended at `cx change archive` which validated → merged specs → moved to archive → saved memory.
Now: lifecycle adds a verify step between tasks completion and archive: `cx change verify <name>` → agent writes `verify.md` → `cx change archive` gates on `verify.md`.

## REMOVED Requirements

None. All existing change lifecycle behaviors are preserved. The archive merge flow (agent-assisted, developer-approved) is unchanged.
