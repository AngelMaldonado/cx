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

### 3. Decompose (mandatory before implementation)

- Run `cx decompose <name>` — scaffolds `docs/changes/<name>/` with empty templates and archives the masterfile
- Dispatch **Planner** in **decompose** mode — it reads the archived masterfile and fills in proposal.md and design.md
- Verify via `cx change status` that proposal and design are filled

### 4. Implement

- Dispatch **executor agent** with the change name — it reads the change docs and implements
- The executor fills in tasks.md with completed work and Linear issue references

### 5. Review

- Dispatch **Reviewer** as a quality gate
- If Reviewer finds issues: dispatch executor to fix, then re-review
- Present results to the developer

## Rules

- Never dispatch an executor without completing decompose first
- All three change files (proposal, design, tasks) must be non-empty before implementation starts
- The developer must approve the plan before decompose
- Save a session summary via `cx memory save --type session` at session end

