---
name: cx-change
description: Create and manage structured changes tracked in docs/changes/. Activate when the developer wants to start a new feature, reference an existing change, or update change status.
---

# Skill: cx-change

## Description
Create and manage structured changes. A change is a set of related modifications tracked in `docs/changes/` with proposal, design, and task documents.

## Triggers
- Developer wants to start a new feature or change
- Developer references an existing change by name
- Developer wants to update change status

## Steps

1. Before filling any change document, run `cx instructions <artifact>` to receive the template, project context, dependency state, and spec index.
2. Run `cx change new <name>` to scaffold a new change with template files and an empty specs/ directory.
3. Fill proposal.md, then design.md, following the dependency order shown by `cx instructions`.
4. During decompose, identify affected spec areas and create delta specs under `specs/<area>/spec.md` with ADDED/MODIFIED/REMOVED structure.
5. Run `cx change status` to see completion state, verify status, and synced deltas.
6. Run `cx change verify <name>` once implementation is complete. Review the output and fill verify.md (or dispatch the Reviewer agent).
7. Run `cx change archive <name>` to archive the completed change.
   - For non-behavioral changes (CI, docs, tooling): `cx change archive <name> --skip-specs`
8. For long-running changes needing early spec stabilization, run `cx change spec-sync <name>` before archiving.

## Rules

- Always call `cx instructions <artifact>` before writing proposal.md, specs, design.md, or tasks.md
- Change names must be kebab-case, max 40 characters
- All three core files (proposal, design, tasks) must be non-empty before archiving
- verify.md must have PASS status before archiving (unless --skip-specs)

## Commands

| Command | Purpose |
|---------|---------|
| `cx change new <name>` | Scaffold change directory with templates |
| `cx change status` | Show all changes with completion and verify state |
| `cx change verify <name>` | Generate verification prompt and scaffold verify.md |
| `cx change spec-sync <name>` | Merge delta specs into canonical specs mid-change |
| `cx change archive <name>` | Validate, bootstrap specs, and move to archive |
| `cx instructions <artifact>` | Get template + context + deps for an artifact |
