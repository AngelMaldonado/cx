# Masterfile: spec-system-evolution

## Problem

CX's spec system has two friction points that degrade agent output quality and system reliability:

**1. The spec template is a stub, but the existing 13 spec areas use a richer format.** New projects scaffold `# Spec: name / ## Overview / ## Requirements / ## Behavior` — three bare sections with no guidance on RFC 2119 keyword use, no scenario structure, and no Purpose statement. Yet the existing CX specs themselves already use Purpose-driven prose and Given/When/Then scenarios. There is a gap between what the template produces and what CX actually does. Agents creating new spec areas have no template signal to match the established format.

**2. The lifecycle has no project-level configuration and no structured verify step.** Agents lack persistent context about a project's tech stack, conventions, or per-artifact rules — they infer this each session from docs/ files. The archive gate validates file presence and parses deltas but does not verify that the implementation actually satisfies the spec requirements. There is no escape hatch for non-behavioral changes (CI, tooling, docs) that do not need spec coverage. And the dependency graph — proposal → specs + design → tasks → apply → verify → archive — exists only in agent prompt text, with no binary enforcement.

**3. Agents retrieving context for artifact creation make multiple round-trips.** There is no command that delivers "here is the template, here are your dependencies, here is what you unlock next" in a single call. Agents must piece together context from multiple reads, which wastes tokens and increases the chance of missing a dependency.

These three problems compound: a thin spec template leads to low-quality specs; low-quality specs make the verify step ambiguous; no project config means agents cannot tailor artifact generation to the project's conventions.

---

## Context

**Current spec template** (`internal/templates/docs/spec.md`):
```markdown
# Spec: {{name}}

## Overview

## Requirements

## Behavior
```
Three bare sections. No RFC 2119. No Purpose. No Scenarios. Has an H1 title that is redundant with frontmatter.

**Existing CX spec format** (observed across all 13 areas): Rich Purpose-driven prose, requirements written with MUST/SHOULD/MAY keywords (informal), Given/When/Then scenarios in separate `scenarios.md` files, structured section headings. The existing format is better than the template — the template needs to catch up.

**Archive validation today** (`internal/change/change.go`): Checks that proposal.md, design.md, tasks.md exist and are non-empty (not just the template text). Bootstraps missing canonical spec directories from delta ADDED sections. Then moves the change to `docs/archive/`. No verification that implementation satisfies spec requirements.

**Dependency enforcement today**: Enforced entirely by the Master agent via prompts ("do not create tasks.md before design.md is complete"). No binary-level gate. Any agent that ignores the prompt can skip steps.

**No project config file exists.** Projects have `docs/memories/DIRECTION.md` for memory guidance and `docs/overview.md` / `docs/architecture/index.md` for general context, but no structured config that agents can parse for tech stack, conventions, and per-artifact rules.

**No `cx instructions` command exists.** Agents must read the spec template, then read dependency files, then read the canonical spec index, then decide what to write — four round-trips per artifact.

**Delta spec format** (`internal/templates/docs/delta-spec.md`): `## ADDED Requirements / ## MODIFIED Requirements / ## REMOVED Requirements`. This format does not include scenarios. A change that adds Given/When/Then scenarios has no structured place to put them in the delta.

**`cx change archive` flow**: validates completeness → bootstraps new spec areas → agent-assisted merge → moves to archive → memory save. The Planner (in decompose mode) does the merge. The verify step between tasks and archive exists in the masterfile but has no binary command or structured output format.

**`cx.yaml` does not exist in any form.** The closest analog is `docs/memories/DIRECTION.md`, which guides memory saves but not artifact generation.

**Artifact files currently use H1 titles.** All templates (spec.md, proposal.md, design.md, tasks.md, delta-spec.md, masterfile.md) open with a `# Title: name` line. This is redundant with frontmatter metadata and creates noise for agents that parse the file content.

---

## Direction

Adopt 7 concepts (down from 8 — see Dropped section below), adapted for CX's agent-driven model, in five logical groups:

### Group 1: Frontmatter-Only Artifact Headers (All Templates)

All markdown artifact templates drop their H1 title line. Instead, each file opens with YAML frontmatter that carries the name and type. Content begins immediately after the frontmatter block.

**spec.md template** (before):
```markdown
# Spec: {{name}}

## Overview
```

**spec.md template** (after):
```markdown
---
name: {{name}}
type: spec
---

<One paragraph stating the purpose of this spec area — what it governs and why it exists.>

---

## Requirements

### REQ-001: <Requirement Name>
<Agents MUST/SHOULD/MAY ...> <behavior description>

### REQ-002: <Requirement Name>
...

---

## Scenarios

*Optional. Add a `scenarios.md` file alongside this spec for Given/When/Then examples.*
```

**proposal.md template** (after):
```markdown
---
name: {{name}}
type: proposal
---

## Problem

## Approach

## Scope

## Affected Specs
```

**design.md template** (after):
```markdown
---
name: {{name}}
type: design
---

## Architecture

## Key Decisions

## External Dependencies
```

**tasks.md template** (after):
```markdown
---
name: {{name}}
type: tasks
---

## Tasks
```

**delta-spec.md template** (after):
```markdown
---
name: {{name}}
area: {{area}}
type: delta-spec
change: {{change}}
---

## ADDED Requirements

## MODIFIED Requirements

## REMOVED Requirements

## ADDED Scenarios

## MODIFIED Scenarios

## REMOVED Scenarios
```

**masterfile.md template** (after):
```markdown
---
name: {{name}}
type: masterfile
---

## Problem

## Context

## Direction

## Open Questions

## Files to Modify

## Risks

## Testing
```

This is a template-only change for most files — no Go parsing logic depends on H1 presence. The `internal/change/change.go` completeness checks look for non-empty content after stripping the template text; they will continue to work with frontmatter as the file header. Any code that displays the change name reads it from the filename or a config struct, not from the H1.

Key additions to the spec.md content (beyond removing H1):
- Purpose paragraph as the first section (not a section heading, just prose)
- Requirements as named subsections with RFC 2119 keywords (MUST/SHALL/SHOULD/MAY)
- Pointer to scenarios.md for behavioral examples
- Horizontal rule separators (matching existing spec style)
- REQ-NNN naming convention surfaced in the template

This surfaces REQ-NNN naming that helps MODIFIED deltas reference requirements by name instead of line number (fixing a current gap in the archive merge flow).

### Group 2: Project Config (cx.yaml)

New file at project root: `cx.yaml`. Created by `cx init` from a template; optional (absent = all defaults).

```yaml
schema: cx/v1
context: |
  Tech stack: Go 1.23, PostgreSQL 15, React 18
  Conventions: REST API, snake_case DB columns, trunk-based development
rules:
  proposal:
    - Include rollback plan for all data migrations
  specs:
    - Use REQ-NNN naming for requirements
    - Write Given/When/Then scenarios for non-obvious behaviors
  design:
    - Document all external service dependencies
  tasks:
    - Create Linear issues before writing tasks.md
```

The binary reads `cx.yaml` and injects `context` and relevant `rules` into the output of `cx instructions <artifact>`. Agents do not need to find and read this file themselves — `cx instructions` delivers it.

Implementation in Go: new `internal/config/` package. `config.Load(rootDir)` reads `cx.yaml` if it exists; returns zero-value struct if absent. No validation beyond YAML parse. `cx init` writes a commented-out template to guide the developer.

`cx doctor` adds a check: if `cx.yaml` exists, validate its top-level structure. Warning (not error) if malformed.

### Group 3: cx instructions Command

New command: `cx instructions <artifact>` where artifact is one of: `proposal`, `specs`, `design`, `tasks`.

Output format (stdout, for agent consumption):
```
=== ARTIFACT: specs ===

TEMPLATE:
<full content of the spec.md template>

PROJECT CONTEXT:
<content of cx.yaml .context field, or "(no cx.yaml found)">

PROJECT RULES FOR THIS ARTIFACT:
<list from cx.yaml .rules.specs, or "(none configured)">

DEPENDENCIES (read these before writing):
- docs/changes/<current-change>/proposal.md

WHAT THIS UNLOCKS:
- design.md (requires: proposal + specs)
- tasks.md (requires: specs + design)

EXISTING SPECS:
<list of existing spec areas from docs/specs/index.md>
```

The command does not take a change name — it operates on the current working context. Agents call this at the start of filling any artifact. This replaces the multi-file-read pattern agents currently use.

Implementation: new `cmd/instructions.go`, new `internal/instructions/` package. Reads `cx.yaml`, the appropriate template, `docs/specs/index.md`, and the static dependency graph (hardcoded in the binary per the schema in Group 4).

### Group 4: Schema-Driven Dependency Graph

Formalize the artifact dependency graph in the binary, not just in agent prompts. The graph is defined as a static data structure in Go (not in `cx.yaml` — this is CX's own invariant, not a project concern):

```go
var ArtifactGraph = []Artifact{
    {ID: "proposal",  File: "proposal.md",       Requires: []string{}},
    {ID: "specs",     File: "specs/**/*.md",      Requires: []string{"proposal"}},
    {ID: "design",    File: "design.md",          Requires: []string{"proposal"}},
    {ID: "tasks",     File: "tasks.md",           Requires: []string{"specs", "design"}},
    {ID: "verify",    File: "(agent-confirmed)",  Requires: []string{"tasks"}},
}
```

**Binary enforcement** on `cx change archive`:
1. Check proposal.md is filled (today: already done)
2. Check design.md is filled (today: already done)
3. Check tasks.md is filled (today: already done)
4. NEW: check that verify has been run (see Group 5)
5. If `--skip-specs` flag: skip delta spec check (see below)

**State display in `cx change status`**: Add a computed state per artifact — BLOCKED (dependency missing), READY (deps complete, file empty), IN-PROGRESS (file non-empty but not verified), DONE. This gives agents and developers a single command for lifecycle position.

**`--skip-specs` flag on `cx change archive`**: Escape hatch for non-behavioral changes (CI, tooling, docs, dependency bumps). When passed, the archive skips delta spec validation entirely and skips the verify step. The change is archived immediately after proposal/design/tasks completeness check. The flag is logged in the archive memory save so the skip is visible in history.

### Group 5: Structured Verify Step

New command: `cx change verify <name>`

The verify step is a pre-archive gate where the Planner (or a dedicated Verifier agent) checks the implementation against the specs. The binary does not do the verification itself — it orchestrates the agent review.

**What the binary does:**
1. Prints all delta specs and canonical spec files relevant to the change (like the merge output today, but pre-archive)
2. Prints the change's proposal.md and design.md for intent context
3. Prints three check dimensions:
   - COMPLETENESS: Are all ADDED requirements covered by implementation?
   - CORRECTNESS: Does the implementation match the spec intent?
   - COHERENCE: Are the design decisions reflected in the code?
4. Outputs a structured template for the agent to fill in:

```
=== VERIFY: <name> ===

COMPLETENESS
[ ] REQ-001: <requirement text>
[ ] REQ-002: <requirement text>
...

CORRECTNESS
<key behaviors to check against>

COHERENCE
<design decisions to verify>

AGENT: Review git diff and fill in the above. Output PASS or FAIL with severities:
CRITICAL (blocks archive) | WARNING (noted but allows archive) | SUGGESTION (informational)
```

5. After agent produces the review, the agent writes a `verify.md` to `docs/changes/<name>/verify.md`
6. `cx change archive` checks for `verify.md` existence (and that it contains PASS or no CRITICAL issues) before proceeding

**Severity semantics:**
- CRITICAL — implementation is missing or contradicts a MUST requirement. Blocks archive.
- WARNING — SHOULD requirement not met, or minor gap. Noted; archive proceeds.
- SUGGESTION — MAY requirement or stylistic issue. Informational only.

**What counts as PASS**: no CRITICAL issues, any number of WARNING/SUGGESTION.

**Integration with `cx change status`**: shows verify state as PENDING / PASS / FAIL per change.

**verify.md template** (after, no H1):
```markdown
---
name: {{name}}
type: verify
---

## Result

PASS | FAIL

## Completeness

## Correctness

## Coherence

## Issues
```

### Group 6: Delta Spec — Add Scenarios Support

The delta-spec template already shown in Group 1 includes Scenarios sections. This is covered there; no additional implementation beyond the template update.

The archive merge flow (agent-assisted) already handles prose; agents can include scenario changes in the Scenarios sections and the merge produces the correct `scenarios.md` update alongside `spec.md`. This is a template-only change — no new binary parsing needed.

### Group 7: Spec Sync Before Archive (Spec-Sync Command)

New command: `cx change spec-sync <name>`

Enables merging delta specs into canonical specs without archiving the change. Useful when a change spans multiple sessions and the team wants to stabilize the spec before implementation is complete.

Behavior: identical to the archive's spec merge step (step 2 of archive), but:
- Does NOT move the change to archive
- Does NOT require tasks.md to be complete
- Requires only proposal.md to be filled
- After merge, marks each delta as SYNCED (adds `synced: true` to the delta file's frontmatter)
- `cx change archive` skips already-SYNCED deltas during its own merge

This allows progressive spec evolution: write the spec early, sync it, continue implementation, then archive clean.

**`cx change status` update**: shows SYNCED deltas distinctly (e.g., `synced` vs `pending`).

---

## Dropped

**Progressive rigor modes (lite/full) — considered and dropped.**

The plan initially included two modes: Lite (default, verify optional, delta specs optional) and Full (verify required, delta specs required, rules enforced). This was dropped because:

1. Two modes add configuration surface and decision overhead without proportional value. Most teams will either always want verification or always skip it — a per-command `--skip-specs` flag covers the "skip" case without mode machinery.
2. Mode-based behavior makes it harder to reason about what `cx change archive` will do on a given project. One consistent behavior (verify required, `--skip-specs` to opt out) is simpler.
3. The `cx.yaml` rules field still allows project-specific guidance; the binary just does not enforce rules differently based on mode.

The `--skip-specs` flag on `cx change archive` replaces the practical escape hatch that Lite mode provided. Doctor checks remain mode-independent.

---

## Open Questions

None. All 7 concepts have been mapped to concrete implementation choices. Key design decisions are documented below in Observations.

---

## Observations

**Template vs. existing spec gap was intentional (now resolved)**: The existing 13 specs were written by humans who used a richer format than the generated template. The template was intentionally minimal to not overwhelm new users. The new template strikes a balance: structured enough to signal RFC 2119 usage and REQ-NNN naming, minimal enough not to be intimidating.

**H1 removal is safe for existing completeness checks**: The `internal/change/change.go` completeness check looks for non-empty content that is not equal to the template text. Since frontmatter is a different format than the old H1, the check needs to be updated to strip frontmatter before comparing against a "template-only" state. Concretely: a file with only frontmatter and no section content should still be considered empty (template-only). A file with frontmatter plus filled sections is non-empty.

**`cx.yaml` is not a schema constraint, it's agent configuration**: Unlike OpenSpec's schema field which enforces the dependency graph, CX's dependency graph is hardcoded in the Go binary. `cx.yaml` is purely for project context and artifact-generation rules. This matches CX's agent-driven model where the binary enforces structure and agents use config to tune output quality.

**`cx instructions` replaces ad-hoc context gathering**: Today, the Planner agent reads the spec template, reads existing specs, reads the proposal to understand what the spec should cover. `cx instructions specs` delivers all of this in one call. The command is also how `cx.yaml` rules reach the agent — no need to know where `cx.yaml` lives.

**Verify step is agent-driven, binary-scaffolded**: The binary cannot verify implementation correctness — it doesn't know the project language or semantics. The binary's job is to structure the verification prompt (extract requirements from delta specs, format the check template) and record the result. The agent (Planner or Reviewer) fills in the actual assessment.

**`--skip-specs` prevents lifecycle rigidity**: Some changes genuinely have no behavioral impact (update a dependency version, add a CI job, fix a typo in docs). Forcing verify and spec coverage on these wastes time and creates artificial spec entries. The flag is logged so the team can see it and audit if needed. This replaces what the dropped "lite mode" was solving.

**Spec-sync is useful for long-running changes**: If a change takes two weeks and the spec is finalized after day 3, the team benefits from having the canonical spec updated before the change is archived. Other changes can reference the updated spec immediately. The SYNCED marker in frontmatter prevents double-merge on archive.

**REQ-NNN naming in templates enables better delta merge**: The current archive merge identifies requirements by line number or text match, which breaks when lines shift. Named requirements (REQ-001: Auth Token Validation) give the Planner a stable reference for MODIFIED deltas. This is a convention, not a binary rule — the template signals it, agents adopt it.

**Delta scenarios are agent-merged**: The binary does not parse ADDED/MODIFIED/REMOVED Scenarios differently from requirements. The agent merge handles all sections as prose. The template addition is solely to give agents a designated place to put scenario changes, preventing them from being conflated with requirement changes.

**Frontmatter SYNCED marker replaces ad-hoc header approach**: The original plan noted adding a `synced: true` header to delta files. With frontmatter now standard across all artifacts, the SYNCED marker goes in the frontmatter block naturally: `synced: true`. The binary can parse this with the same YAML logic used for `cx.yaml`.

---

## Files to Modify

### Templates (pure content changes, no Go logic except completeness-check update)

- `internal/templates/docs/spec.md` — Replace H1 + 3-section stub with frontmatter + Purpose + named Requirements + Scenarios pointer
- `internal/templates/docs/delta-spec.md` — Replace H1 + Change line with frontmatter; add ADDED/MODIFIED/REMOVED Scenarios sections
- `internal/templates/docs/proposal.md` — Replace H1 with frontmatter
- `internal/templates/docs/design.md` — Replace H1 with frontmatter
- `internal/templates/docs/tasks.md` — Replace H1 with frontmatter
- `internal/templates/docs/masterfile.md` — Replace H1 with frontmatter
- `internal/templates/docs/cx.yaml` — New file: default template with comments explaining each field
- `internal/templates/docs/verify.md` — New file: verify output template with frontmatter (agent fills content)

### New Go packages

- `internal/config/` — New package
  - `config.go`: `Config` struct (Schema, Context, Rules map), `Load(rootDir string) (*Config, error)`, returns zero-value if no `cx.yaml`
  - Note: Mode field removed (progressive rigor dropped)

- `internal/instructions/` — New package
  - `instructions.go`: `Build(rootDir string, artifact string) (string, error)` — reads config, template, index, dependency graph; returns formatted output string
  - `graph.go`: `ArtifactGraph` static data structure and `DependenciesOf(artifact string) []string`, `UnlocksOf(artifact string) []string`

- `internal/verify/` — New package
  - `verify.go`: `BuildPrompt(rootDir, changeName string) (string, error)` — reads delta specs, canonical specs, proposal, design; formats verification prompt
  - `Record(rootDir, changeName, result string) error` — writes verify.md

### Modified Go packages

- `internal/change/change.go` — Modify `Archive()`:
  - Check `verify.md` exists and contains PASS / no CRITICAL (required, not mode-gated)
  - Add `--skip-specs` flag handling (new `ArchiveOptions` struct parameter)
  - Update `ChangeInfo` struct: add `HasVerify bool`, `VerifyStatus string`, `DeltasSynced []string`
  - Update completeness check: strip frontmatter before comparing against empty-template state

- `internal/change/templates.go` — `SpecTemplate()` now uses the updated `spec.md` template (no logic change, template file change propagates automatically)

### New commands

- `cmd/instructions.go` — `cx instructions <artifact>` command
  - Calls `internal/instructions.Build(rootDir, artifact)`
  - Prints to stdout (agent consumption)
  - Tab-completion for artifact names: proposal, specs, design, tasks

- `cmd/verify.go` — `cx change verify <name>` command (subcommand of `cx change`)
  - Calls `internal/verify.BuildPrompt()` and prints to stdout
  - Agent reads output, produces review, agent writes verify.md via its own file tools

- `cmd/specsync.go` (or add to `cmd/change.go`) — `cx change spec-sync <name>` command
  - Validates proposal.md exists and is filled
  - Runs same merge flow as archive step 2 (prints the merge prompt to stdout)
  - After agent completes merge: binary sets `synced: true` in delta file frontmatter
  - New subcommand of `changeCmd`

### Modified commands

- `cmd/change.go` — `runChangeArchive`:
  - Add `--skip-specs` flag
  - Add verify gate (required, not mode-conditional)
  - Update status output: show verify state and synced delta markers

- `cmd/change.go` — `runChangeStatus`:
  - Add verify state column
  - Add synced/pending indicator for delta specs
  - Add BLOCKED/READY/IN-PROGRESS/DONE computed state per artifact

### Init command

- `cmd/init.go` / relevant init package — Add `cx.yaml` creation step
  - After DIRECTION.md setup, create `cx.yaml` from template
  - No interactive rigor-level question (dropped with progressive rigor)
  - Skip if `cx.yaml` already exists (idempotent)

### Specs to create or update

- `docs/specs/init/spec.md` — Add `cx.yaml` creation step; update summary section
- `docs/specs/change-lifecycle/spec.md` — Add verify step, spec-sync command, --skip-specs flag, schema-driven state
- `docs/specs/doctor/spec.md` — Add cx.yaml validation check
- `docs/specs/skills/spec.md` or new catalog entry — Update cx-change skill to include new commands

---

## Risks

**Risk 1: Template update breaks existing workflows.**
New spec template uses REQ-NNN naming and RFC 2119 keywords, which agents trained on the old template may not follow initially. Mitigation: the template is guidance, not enforcement. Agents that produce looser requirements still produce valid specs — they just don't get the naming benefits for delta merge.

**Risk 2: `cx instructions` output is too long for agent context windows.**
If the project has 50 spec areas, listing them all in `cx instructions specs` could be verbose. Mitigation: list only the spec area names (not full content) and note total count. The agent can read individual specs if needed.

**Risk 3: Verify step becomes a bureaucratic checkbox.**
If agents fill in verify.md mechanically without genuinely checking, the gate becomes theater. Mitigation: the Reviewer agent (existing) is the right agent for verify — it already does post-implementation review against specs. Wire `cx change verify` into the Reviewer's workflow via the cx-review skill update.

**Risk 4: `cx.yaml` rules are ignored by agents.**
If agents don't call `cx instructions` before writing artifacts, they miss the rules. Mitigation: update the cx-change skill to mandate calling `cx instructions <artifact>` before filling any change document. The skill is the enforcement mechanism in CX's agent-driven model.

**Risk 5: Spec-sync creates merge conflicts at archive time.**
If a change runs spec-sync, then another change modifies the same spec area and archives first, the second archive merge could conflict. Mitigation: archive merge already handles this (agent flags divergence and asks developer). The SYNCED marker tells the merge agent to produce a new diff-from-synced rather than diff-from-original.

**Risk 6: Binary enforcement of dependency graph may be too strict.**
Agents sometimes write design.md and proposal.md simultaneously, then fill in gaps. Binary blocking on state checks could interrupt flow. Mitigation: binary enforces only at archive time, not during filling. The state display in `cx change status` is advisory. Agents can fill files in any order — binary only gates the final archive and verify step.

**Risk 7: Frontmatter-only files confuse completeness checks.**
A file with only frontmatter and no section content looks non-empty to a naive byte-count check. Mitigation: the completeness check must be updated to strip frontmatter before evaluating content. A file where all content after the closing `---` is whitespace is treated as empty (template-only state).

---

## Testing

### Unit tests

- `internal/config/config_test.go` — Load with/without cx.yaml; zero-value defaults; YAML parse error handling
- `internal/instructions/instructions_test.go` — Output for each artifact; cx.yaml rules injection; missing cx.yaml fallback; unknown artifact returns error
- `internal/verify/verify_test.go` — Prompt construction from delta specs; verify.md record write; PASS/FAIL detection in existing verify.md
- `internal/change/change_test.go` — Completeness check correctly treats frontmatter-only file as empty; treats frontmatter + content as non-empty

### Integration tests (if CX has an integration test harness)

- `cx instructions proposal` with and without cx.yaml
- `cx change archive <name> --skip-specs` skips delta validation and verify
- `cx change archive <name>` requires verify.md (no mode flag needed)
- `cx change spec-sync <name>` sets `synced: true` in delta frontmatter; subsequent archive skips them
- `cx change status` shows correct BLOCKED/READY/IN-PROGRESS/DONE state

### Manual verification

1. `cx init` on a new repo → `cx.yaml` created with commented template (no rigor-level prompt)
2. `cx brainstorm new test-feature` → `cx decompose test-feature` → `cx change new test-feature`
3. Open generated proposal.md → verify it has frontmatter, no H1 title
4. `cx instructions proposal` → verify output includes template, project context, dependencies, unlocks
5. Fill proposal.md → `cx instructions specs` → verify specs shows proposal as dependency satisfied
6. Write delta spec → verify delta file has frontmatter with `area:` and `change:` fields, no H1
7. `cx change spec-sync test-feature` → verify delta frontmatter now has `synced: true`, canonical spec updated
8. Fill design.md, tasks.md → `cx change verify test-feature` → verify prompt output includes REQ-NNN from delta
9. Agent fills verify.md with PASS → `cx change archive test-feature` → succeeds
10. `cx change archive test-feature` without verify.md → blocked with clear error
11. `cx change archive test-feature --skip-specs` → archives without delta check or verify
12. `cx doctor` → warns on malformed cx.yaml

### Spec alignment check

The updated `spec.md` template should match the format of the existing 13 spec areas (minus the H1 title — existing specs will be migrated separately or left as-is). Manually compare the generated spec for a new area against `docs/specs/change-lifecycle/spec.md` and `docs/specs/orchestration/spec.md` — the new template should produce a structure consistent with these (frontmatter replaces the H1 title; sections match).

---

## References

- OpenSpec repository: https://github.com/Fission-AI/OpenSpec
- RFC 2119 (MUST/SHALL/SHOULD/MAY keywords): https://datatracker.ietf.org/doc/html/rfc2119
- Existing CX specs (richer format reference): `docs/specs/change-lifecycle/spec.md`, `docs/specs/orchestration/spec.md`
- Existing scenario format spec: `docs/specs/scenarios/spec.md`
- Archive merge flow (current): `docs/specs/change-lifecycle/spec.md` → ## Archive
- Delta spec template (current): `internal/templates/docs/delta-spec.md`
- Spec template (current): `internal/templates/docs/spec.md`
