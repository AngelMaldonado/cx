# Spec: Change Lifecycle

A change is a unit of work that modifies the system. It tracks what's being changed, why, and what specs are affected. Changes live in `docs/changes/<name>/` and flow through a defined lifecycle from creation to archive.

---

## Entry Points

How a change gets created depends on the session mode:

| Session Mode | Path to Change |
|-------------|----------------|
| **PLAN** | Developer brainstorms → masterfile in `docs/masterfiles/` → `cx decompose` creates the change |
| **BUILD** | Developer describes new work → agent runs `cx change new <name>` directly |
| **CONTINUE** | Change already exists → agent picks up where it left off |

BUILD mode can also follow the brainstorm path if the agent judges the scope warrants it. The skill guides this decision.

---

## Change Directory Structure

Every change lives in `docs/changes/<name>/` and requires three files before implementation can begin:

```
docs/changes/add-ble-pairing/
├── proposal.md          # REQUIRED — WHY: intent, scope, approach
├── design.md            # REQUIRED — HOW: technical decisions, architecture
├── tasks.md             # REQUIRED — WHAT: Linear issue refs + implementation notes
└── specs/               # DELTA: what changes in existing specs
    └── <area>/
        └── delta.md
```

All three files must be present and non-empty before the agent begins implementation. `cx doctor` warns on changes with missing files. `cx change status` shows completion state per file.

---

## Commands

| Command | Purpose |
|---------|---------|
| `cx change new <name>` | Create `docs/changes/<name>/` with empty templates for all three files |
| `cx change status` | List all active changes with per-file completion state |
| `cx change archive <name>` | Validate, trigger agent-assisted merge, move to archive |

---

## proposal.md

The intent document. Answers **why** this change exists.

```markdown
# Proposal: <change-name>

## Problem
<what's wrong or missing — the pain that motivates this change>

## Approach
<high-level solution direction — what will be done, not how>

## Scope
<what's in scope and what's explicitly out of scope>

## Affected Specs
<which spec areas will have deltas — e.g., gas-monitoring, device-communication>
```

---

## design.md

The technical approach. Answers **how** this change will be implemented.

```markdown
# Design: <change-name>

## Architecture
<how the solution fits into the existing system — components, data flow, interfaces>

## Technical Decisions
<key choices made during design — libraries, patterns, approaches>

## Implementation Notes
<anything the implementing agent needs to know — gotchas, constraints, dependencies>
```

---

## tasks.md

The work breakdown. Answers **what** needs to be done.

```markdown
# Tasks: <change-name>

## Linear Issues
- PROJ-100: <task description>
- PROJ-101: <task description>
- PROJ-102: <task description>

## Implementation Notes
<technical notes relevant to implementation — ordering, dependencies between tasks>
```

`tasks.md` is a reference document that maps implementation units to Linear issues — not a to-do list with checkboxes. The agent creates Linear issues via MCP, then records the references here.

---

## Delta Specs

When a change modifies an existing spec area, the agent creates a delta file:

```
docs/changes/<name>/specs/<area>/delta.md
```

### Delta Format

```markdown
# Delta: <spec-area>

## ADDED
### Requirement: <name>
<requirement text>

#### Scenario: <name>
- GIVEN ...
- WHEN ...
- THEN ...

## MODIFIED
### Requirement: <name> (from spec.md line N)
<new text>
(Previously: <old text>)

## REMOVED
### Requirement: <name>
(Deprecated — <reason>)
```

The three sections (ADDED, MODIFIED, REMOVED) are all optional — include only the ones that apply. The binary parses these by heading name during archive.

---

## Archive

When all tasks are complete, the agent runs `cx change archive <name>`. This triggers a multi-step process:

### 1. Validation

The binary checks:
- All three files (proposal.md, design.md, tasks.md) exist and are non-empty
- The agent confirms via Linear MCP that all referenced issues are done
- All delta files parse correctly (valid ADDED/MODIFIED/REMOVED sections)

If validation fails, the command prints what's missing and exits non-zero.

### 2. Agent-Assisted Spec Merge

For each delta file, the binary does **not** merge automatically. The merge is an interactive collaboration between the binary, the agent, and the developer.

**Merge flow per delta:**

1. Binary reads the delta (`docs/changes/<name>/specs/<area>/delta.md`) and the canonical spec (`docs/specs/<area>/spec.md`)
2. Binary outputs both files as structured content to stdout, clearly delimited:

```
=== CANONICAL SPEC: <area>/spec.md ===
<full content of canonical spec>

=== DELTA: <area>/delta.md ===
<full content of delta>

=== MERGE INSTRUCTIONS ===
Produce a merged spec.md that:
- Incorporates all ADDED sections as new requirements
- Applies all MODIFIED sections (replacing old text)
- Removes all REMOVED sections
- Maintains the spec's existing structure and voice
```

3. The agent reads this output and produces a merged version of spec.md
4. The agent presents the merged result to the developer **inline in the conversation** for review:
   - Shows the key changes in a readable diff-like format
   - Explains what was added, modified, and removed
   - Asks the developer to approve or request changes
5. **If the developer approves**: the agent writes the merged spec to `docs/specs/<area>/spec.md`
6. **If the developer requests changes**: the agent adjusts the merged spec and re-presents
7. After writing, the binary runs `cx doctor` validation on the merged spec. If issues are found (missing required sections, malformed structure), the agent fixes them and re-presents to the developer

This keeps the agent in the loop for merge quality — deltas can be nuanced and a mechanical merge might produce an incoherent spec. The developer always sees and approves the result before it's written.

**Edge cases:**
- If the canonical spec has changed significantly since the change was created (e.g., another change was archived first), the agent flags the divergence to the developer and suggests manual review of the conflicts
- If a delta references requirements by line number and the lines have shifted, the agent uses requirement names to find the correct locations
- If the merge would result in contradictory requirements, the agent surfaces the contradiction and asks the developer to resolve

### 3. New Spec Area Auto-Creation

If a delta targets a spec area that doesn't exist yet (e.g., `docs/specs/sms-alerting/` doesn't exist):

1. Binary creates the new spec directory: `docs/specs/<area>/`
2. Binary creates `spec.md` from the ADDED sections in the delta
3. Binary updates `docs/specs/index.md` to include the new area
4. Agent reviews the generated spec before committing

### 4. Move to Archive

After merge is complete:

```
docs/changes/add-ble-pairing/
    → docs/archive/<date>-add-ble-pairing/
```

The archived directory retains all original files (proposal, design, tasks, deltas) as a historical record. Git history shows the full evolution.

### 5. Memory Save

The binary saves an observation summarizing the completed change:

```
docs/memories/observations/<date>-<author>-archived-<name>.md
```

This ensures the change shows up in future `cx context` output for team awareness.

---

## Status Display

`cx change status` output:

```
Active changes:

  add-ble-pairing (angel, 3 days)
    ✓ proposal  ✓ design  ✓ tasks
    Linear: CUB-140..CUB-143
    Delta specs: device-communication
    → Ready to archive

  fix-gas-threshold (carlos, 1 day)
    ✓ proposal  ✓ design  ○ tasks
    Linear: not created yet
    Delta specs: gas-monitoring
    → Missing: tasks.md

  refactor-mqtt-client (angel, 5 hours)
    ✓ proposal  ○ design  ○ tasks
    Linear: not created yet
    Delta specs: (none)
    → Missing: design.md, tasks.md
```

---

## Lifecycle Diagram

```
cx change new <name>
    │
    ▼
docs/changes/<name>/
    proposal.md (template)
    design.md (template)
    tasks.md (template)
    │
    ▼
Agent fills all three files
Agent creates Linear issues via MCP
Agent writes issue refs to tasks.md
Agent creates delta specs if needed
    │
    ▼
Implementation
Agent writes code
Agent saves observations (cx memory save)
Agent updates Linear via MCP
    │
    ▼
cx change archive <name>
    ├── Validates completeness
    ├── Agent-assisted spec merge
    ├── Auto-creates new spec areas from delta ADDED
    ├── Moves to docs/archive/<date>-<name>/
    └── Saves archive observation to memory
```
