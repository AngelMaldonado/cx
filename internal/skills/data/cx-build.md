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

### 1. Gather requirements

- Use `AskUserQuestion` to clarify scope, constraints, preferences, and tech choices
- Keep asking until there are no open questions (3-5 rounds is normal)
- Do NOT dispatch the Planner until requirements are clear

### 2. Plan

- Dispatch **Planner** in **create plan** mode with all gathered requirements
- Planner creates a masterfile at `docs/masterfiles/<name>.md` and returns a brief
- Present the brief to the developer via `AskUserQuestion` — approve or request changes?
- If changes needed: dispatch **Planner** in **iterate plan** mode with feedback, repeat until approved

### 3. Decompose (MANDATORY — you must do this yourself)

**YOU run this command directly via Bash. Do NOT skip it. Do NOT delegate it.**

```
cx decompose <name>
```

This scaffolds `docs/changes/<name>/` with empty templates and archives the masterfile. If you skip this, there are no change docs for anyone to work with.

After running `cx decompose`:
- Dispatch **Planner** in **decompose** mode with the change name and archived masterfile path — it fills in proposal.md and design.md
- Verify via `cx change status` that proposal and design are filled

### 4. Design task breakdown

- Dispatch **Planner** in **task design** mode — it reads the change docs (proposal.md, design.md), analyzes the work, and produces a task breakdown in tasks.md
- The task breakdown assigns work to specific executor agents based on the project's tech stack
- Present the task breakdown to the developer via `AskUserQuestion` for approval

### 5. Implement (orchestrate the full task list)

You are responsible for working through the entire task list, not just spawning one agent.

**First, build the visible task board using `TodoWrite`:**
- Read tasks.md and create a `TodoWrite` entry for every task, all set to `pending`
- This gives the developer a visible checklist of all work before any execution starts

**Then, work through each task:**
- For each task in dependency order:
  1. Update the task to `in_progress` via `TodoWrite`
  2. Dispatch the assigned **executor agent** with the task description, relevant change docs, and any context from previously completed tasks
  3. Wait for the executor to return
  4. Update the task to `completed` via `TodoWrite`
  5. If blocked or failed: present the issue to the developer via `AskUserQuestion` and decide whether to retry, skip, or adjust
- For independent tasks with no cross-dependencies: dispatch multiple executors in parallel
- After all tasks complete, update tasks.md with completion status

Do NOT dispatch a single executor and stop. You must drive every task to completion.

### 6. Review

- Dispatch **Reviewer** as a quality gate over all completed work
- If Reviewer finds issues: dispatch executor to fix, then re-review
- Present final results to the developer

## Rules

- Never dispatch an executor without completing decompose first
- All three change files (proposal, design, tasks) must be non-empty before implementation starts
- The developer must approve the plan before decompose
- Save a session summary via `cx memory save --type session` at session end

