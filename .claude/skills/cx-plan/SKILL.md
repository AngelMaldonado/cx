---
name: cx-plan
description: PLAN mode workflow. Activate when the developer wants to brainstorm, explore ideas, or design at a high level without implementing.
---

# Skill: cx-plan

## Description
Workflow for brainstorming and high-level planning. Creates and iterates masterfiles without implementation. Clean-slate thinking — minimal context loaded intentionally.

## Triggers
- Developer says "let's plan...", "brainstorm...", "what if we...", "how should we approach..."
- Developer mentions "architecture", "v2", "roadmap", "redesign"
- Developer wants to explore an idea without committing to implementation

## Steps

### 1. Gather the idea
- Use `AskUserQuestion` to understand what the developer wants to explore
- Ask about goals, constraints, and what success looks like
- Keep context light — planning benefits from a clean slate
- PLAN mode loads minimal context intentionally — project overview only, no observations or decisions
- This is a clean-slate creative mode

### 2. Plan
- Dispatch **Planner** in **create plan** mode
- Planner creates a masterfile at `docs/masterfiles/<name>.md` and returns a brief
- Present the brief to the developer via `AskUserQuestion`

### 3. Iterate
- If the developer has feedback: dispatch **Planner** in **iterate plan** mode with the feedback
- Planner refines the masterfile (never deletes content, moves resolved questions to Context)
- Present updated brief, repeat until the developer is satisfied
- Direction must narrow over time, Open Questions must shrink

### 4. Transition to BUILD (only if developer requests)
- When the developer says "let's do it", "go ahead", "build this":
- Before transitioning: `cx memory session --goal "..." --accomplished "planned <name>" --next "decompose and implement"`
- Run `cx decompose <name>` to scaffold the change and archive the masterfile
- Switch to the cx-build workflow from step 3 (decompose) onward

## Rules
- No implementation during PLAN mode — masterfile only
- Never decompose without explicit developer approval
- After decompose, the masterfile is archived and session switches to BUILD
- Masterfile names must be kebab-case, max 40 characters
- At session end: `cx memory session --goal "..." --accomplished "..." --next "..."`
