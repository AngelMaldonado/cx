---
name: cx-build
description: BUILD mode workflow. Activate when the developer wants to create something new — a feature, fix, or integration that doesn't exist yet.
---

# Skill: cx-build

## Description

Full workflow for building something new. Covers requirements gathering, planning, decomposition, implementation, and review.

## Triggers

- Developer says "I want to add...", "let's build...", "new feature:", "we need to create...", "implement..."
- Developer describes something that doesn't exist yet

## Steps

### 1. Prime context

- Dispatch **Primer** to load project context, recent observations, active decisions, and personal notes
- Wait for Primer to return before gathering requirements

### 2. Gather requirements

- Ask the developer about scope, constraints, preferences, and tech choices
- Keep asking until there are no open questions (3-5 rounds is normal)
- Do NOT dispatch the Planner until requirements are clear

### 3. Plan

- Dispatch **Planner** in **create plan** mode with all gathered requirements
- Planner creates a masterfile at `docs/masterfiles/<name>.md` and returns a brief
- Present the brief to the developer and ask for approval
- If changes needed: dispatch **Planner** in **iterate plan** mode with feedback, repeat until approved

### 4. Decompose (MANDATORY — you must do this yourself)

**YOU run this command directly via Bash. Do NOT skip it. Do NOT delegate it.**

```
cx decompose <name>
```

This scaffolds `docs/changes/<name>/` with empty templates and archives the masterfile. If you skip this, there are no change docs for anyone to work with.

After running `cx decompose`:
- Dispatch **Planner** in **decompose** mode with the change name and archived masterfile path — it fills in proposal.md and design.md
- Verify via `cx change status` that proposal and design are filled

### 5. Design task breakdown

- Dispatch **Planner** in **task design** mode — it reads the change docs (proposal.md, design.md), analyzes the work, and produces a task breakdown in tasks.md
- The task breakdown assigns work to specific executor agents based on the project's tech stack
- Present the task breakdown to the developer and ask for approval

### 6. Implement (orchestrate the full task list)

You are responsible for working through the entire task list, not just spawning one agent.

**First, build the visible task board using `TodoWrite`:**
- Read tasks.md and create a `TodoWrite` entry for every task, all set to `pending`
- This gives the developer a visible checklist of all work before any execution starts

**Context package for each executor (always include all of these):**
1. Project context from `.cx/cx.yaml`
2. The proposal.md and design.md from the change
3. The specific task description from tasks.md
4. Relevant spec areas (from delta specs in the change)
5. A Scout map of the files the task will modify (dispatch Scout first if needed)

Do NOT dispatch an executor with just a task name — always pass the full context package.

**For independent tasks (2 or more with no cross-dependencies) — default path:**
1. For each independent task: run `cx worktree create <change>-task-N` to create an isolated worktree
2. Mark each task `in_progress` via `TodoWrite`
3. Dispatch all independent executors in parallel, each with:
   - The full context package above
   - Their assigned worktree path as the working directory
4. Wait for all parallel executors to complete; log each run via `cx agent-run log`
5. Mark each task `completed` (or surface blockers to the developer)
6. Dispatch **Merger** agent with: list of task branch names, dependency order, proposal.md and design.md, and the target branch
7. Run `cx worktree cleanup <change>` after the Merger returns successfully

**For dependent tasks (explicit ordering required):**
- Execute sequentially on the main working tree (no worktrees needed)
- For each task in dependency order:
  1. Update to `in_progress` via `TodoWrite`
  2. Dispatch the assigned executor with the full context package
  3. Wait for the executor to return; log the run
  4. Update to `completed` via `TodoWrite`
  5. If blocked or failed: surface the issue to the developer and decide whether to retry, skip, or adjust

**Mixed (some independent, some dependent):**
- Execute independent groups in parallel via worktrees first
- After Merger completes, execute dependent tasks sequentially on the merged result

After all tasks complete, update tasks.md with completion status.

Do NOT dispatch a single executor and stop. You must drive every task to completion.

### 7. Review & Archive

- Dispatch **Reviewer** as a quality gate over all completed work
- If Reviewer finds issues: dispatch executor to fix, then re-review
- **When review passes — DO NOT wrap up. DO NOT commit. You are not done yet.**
- Use `AskUserQuestion` to ask the developer: "Review passed. Ready to archive this change and merge specs?"
- If approved:
  1. Run `cx change archive <name>` — validates completeness, bootstraps any missing canonical specs, moves change to `docs/archive/`
  2. Dispatch **Planner** in **archive** mode with the archived change path — Planner reads each delta spec, merges into canonical specs, presents for approval
- If declined: leave the change active for further work
- Only AFTER the developer responds to the archive question is the BUILD workflow complete

## Rules

- After a successful review, always ask about archiving — the BUILD workflow is not complete until the developer answers
- Worktree-based parallel execution is the default for independent tasks. Only fall back to sequential execution when tasks have explicit dependencies or when there are fewer than 2 independent tasks.
- Never dispatch an executor without completing decompose first
- All three change files (proposal, design, tasks) must be non-empty before implementation starts
- The developer must approve the plan before decompose
- At session end, save a session summary: `cx memory session --goal "..." --accomplished "..." --next "..." --change <name> --discoveries "..." --files "..."`
- The --next field is critical — without it, the next CONTINUE session cannot recover state

