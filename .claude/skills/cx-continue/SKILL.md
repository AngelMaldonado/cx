---
name: cx-continue
description: CONTINUE mode workflow. Activate when the developer is resuming existing work on an active change.
---

# Skill: cx-continue

## Description
Workflow for picking up existing work. Loads the active change context and dispatches the right agent to continue.

## Triggers
- Developer says "let's continue", "where were we", "resume", "back to [feature]", "pick up on [change]"
- Developer references an existing change by name
- Developer mentions previous work

## Steps

### 1. Load context
- Dispatch **Primer** to load the active change context (proposal, design, tasks, last session, change-scoped memory)
- Primer loads last session summary via `cx memory list --type session --change <name>`
- The `next_steps` from the prior session drives what to do next
- If Primer signals empty state, redirect to BUILD mode
- If multiple active changes exist: use `AskUserQuestion` to ask which change to resume
- Review the Primer's summary to understand current state
- If Primer signals empty state (no specs, no changes), this is not a CONTINUE — redirect to BUILD or run `cx init` first.

### 2. Assess remaining work
- Check `cx change status` to see what's done and what's missing
- If the remaining work is straightforward: dispatch executor directly
- If the remaining work is complex or the design needs updating: dispatch **Planner** first to re-plan, then executor

### 3. Implement
- Dispatch **executor agent** with the change name and the Primer's context summary
- The executor picks up where work left off based on the change docs and last session

### 4. Review (if implementation was done)
- Dispatch **Reviewer** as a quality gate
- Present results to the developer

## Rules
- Always load context via Primer before dispatching — the executor needs to know what was done previously
- If the change docs are incomplete (missing proposal, design, or tasks), address that before implementation
- At session end: `cx memory session --goal "..." --accomplished "..." --next "..." --change <name>`
- The --next field is the critical bridge — without it, the next CONTINUE session starts blind
