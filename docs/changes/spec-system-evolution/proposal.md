# Proposal: spec-system-evolution

## Problem

CX's spec and change-lifecycle tooling has three compounding deficiencies that reduce agent output quality:

1. **The spec template is a thin stub.** The three-section `# Spec: name / ## Overview / ## Requirements / ## Behavior` template gives agents no signal to use RFC 2119 keywords, no Purpose paragraph, no REQ-NNN naming, and no scenario structure — yet the 13 existing CX spec areas already use a richer format. Agents creating new spec areas have no template guidance to match established conventions.

2. **The lifecycle lacks structured verify and project configuration.** There is no pre-archive verification step that checks implementation against spec requirements. There is no escape hatch for non-behavioral changes (CI, tooling, docs) that do not need spec coverage. There is no project-level config that agents can read for tech stack and artifact-generation rules — they infer this each session from multiple doc files.

3. **Agents make multiple round-trips to gather artifact context.** No single command delivers "here is the template, here are your dependencies, here is what you unlock next, here are your project rules." Agents must piece this together from separate reads.

These problems compound: a weak template produces low-quality specs; low-quality specs make verify ambiguous; no project config means agents cannot tailor output to project conventions.

## Approach

Introduce seven improvements grouped into five logical areas:

- **Frontmatter-only artifact headers**: All markdown templates replace their H1 title with YAML frontmatter. The spec template gains a Purpose paragraph, REQ-NNN named requirements, RFC 2119 keywords, and a Scenarios pointer. The delta-spec template gains ADDED/MODIFIED/REMOVED Scenarios sections.
- **Project config (cx.yaml)**: A new optional file at the project root. `cx init` writes a commented template. Agents do not read it directly — `cx instructions` delivers its content.
- **cx instructions command**: A new command that delivers, in one call: the artifact template, project context from `cx.yaml`, project rules for the target artifact, dependency requirements, and what this artifact unlocks.
- **Schema-driven dependency graph**: The artifact dependency chain (proposal → specs + design → tasks → verify → archive) is formalized as a static Go data structure, not just agent prompt text. `cx change status` shows computed per-artifact state (BLOCKED/READY/IN-PROGRESS/DONE). `cx change archive` gains a `--skip-specs` flag for non-behavioral changes.
- **Structured verify step**: New `cx change verify <name>` command scaffolds a structured prompt for the Reviewer agent to confirm implementation satisfies spec requirements. A `verify.md` file records the outcome. `cx change archive` gates on `verify.md` presence (unless `--skip-specs`).
- **Spec-sync command**: New `cx change spec-sync <name>` merges delta specs into canonical specs without archiving the change, enabling early spec stabilization in long-running changes.

## Scope

**In scope:**
- All six markdown artifact templates (spec, delta-spec, proposal, design, tasks, masterfile)
- New `verify.md` and `cx.yaml` templates
- New Go packages: `internal/config/`, `internal/instructions/`, `internal/verify/`
- New commands: `cx instructions <artifact>`, `cx change verify <name>`, `cx change spec-sync <name>`
- Modified commands: `cx change archive` (verify gate, `--skip-specs` flag), `cx change status` (enhanced state display)
- Modified archive logic: completeness check updated for frontmatter-first files
- `cx init`: creates `cx.yaml` from template
- `cx doctor`: validates `cx.yaml` structure if present

**Out of scope:**
- Migrating existing 13 spec areas to the new frontmatter format (done separately or left as-is)
- Binary enforcement of artifact fill order during implementation (only gated at archive)
- Progressive rigor modes (considered and dropped in favor of `--skip-specs` flag)
- Changing the agent-assisted spec merge logic (archive merge flow is unchanged)

## Affected Specs

- `change-lifecycle` — verify step, spec-sync command, `--skip-specs` flag, artifact state model, frontmatter completeness check, delta-spec template update
- `init` — `cx.yaml` creation step added to init sequence
- `doctor` — new check: validate `cx.yaml` structure if present
- `skills` — `cx-change` skill must be updated to mandate `cx instructions <artifact>` before filling any change document; new commands added to skill registry
