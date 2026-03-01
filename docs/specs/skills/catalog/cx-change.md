# Skill: cx-change

## Description
Teaches the Master and Contractor how to manage the full change lifecycle — creating changes, tracking status, and archiving completed work. A change is the unit of structured work in CX: it has a proposal (why), a design (how), tasks (what), and optional delta specs (what changes in existing specs).

## Triggers
- Developer wants to start new work without brainstorming ("let's build X", "create a change for Y")
- Developer asks about change status ("what's in progress?", "what changes are active?")
- Developer wants to archive a completed change ("this is done", "archive this change")

## Steps

### Creating a change
1. Run: `cx change new <name>` where `<name>` is a kebab-case slug.
2. The binary creates `docs/changes/<name>/` with three template files:
   - `proposal.md` — WHY: intent, scope, approach
   - `design.md` — HOW: technical decisions, architecture
   - `tasks.md` — WHAT: Linear issue references + implementation notes
3. Work with the developer to fill in proposal.md first. A change without a clear proposal will drift.
4. Fill in design.md with the technical approach. Reference relevant specs and active decisions.
5. For tasks.md, create Linear issues via the MCP server (using cx-linear skill) and reference them.

### Checking status
6. Run: `cx change status` to see all active changes with their completion state.
7. The output shows which files exist and which are empty, plus any delta specs.

### Creating delta specs
8. When the change modifies existing system behavior, create delta specs:
   - Create `docs/changes/<name>/specs/<area>/delta.md`
   - Use the delta format: ## ADDED, ## MODIFIED, ## REMOVED sections
   - Each section references specific requirements in the canonical spec
9. Delta specs are how the system tracks what a change will modify in existing specs. They're merged during archive.

### Archiving a change
10. Run: `cx change archive <name>`. The binary validates:
    - All three required files exist and are non-empty
    - All delta specs reference valid spec areas
11. For each delta spec, the binary initiates an agent-assisted merge:
    - The binary reads the delta and the canonical spec
    - The binary presents both to you (the agent) as structured content
    - Produce a merged version of the canonical spec incorporating the delta
    - Present the merged result to the developer inline for review
    - If the developer approves, the binary writes the merged spec
    - If the developer requests changes, adjust and re-present
    - If `cx doctor` finds issues in the merged spec, fix and re-present
12. For ADDED sections referencing a new spec area that doesn't exist, the binary auto-creates `docs/specs/<new-area>/spec.md` with the added content.
13. After all merges complete, the binary moves the change to `docs/archive/<date>-<name>/`.
14. Save a session memory summarizing what was archived.

## Rules
- Change names must be kebab-case, max 40 characters
- Never archive a change without the developer's explicit approval
- All three files (proposal.md, design.md, tasks.md) must be non-empty before archiving
- During merge: always present the merged spec to the developer before writing. Never silently overwrite a canonical spec
- If a merge conflict can't be resolved (e.g., the canonical spec changed significantly since the change was created), flag it to the developer and suggest manual resolution
- Delta specs should reference specific requirements by name — vague deltas ("update the API section") are not acceptable
