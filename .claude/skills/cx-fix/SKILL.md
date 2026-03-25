---
name: cx-fix
description: FIX mode workflow. Activate when the developer wants a quick, localized code change that bypasses the full change lifecycle.
---

# Skill: cx-fix

## Description

Lightweight path for small, localized code changes. Skips the full change lifecycle (no change docs, no Planner, no archive). Use when the developer wants speed over documentation.

## Triggers

- Developer says "fix", "patch", "tweak", "quick fix", "one-liner"
- Developer requests a small, localized change without architectural implications
- Developer explicitly wants to skip the change lifecycle ("just change X to Y", "rename this", "update this value")

## Steps

### 1. Map the affected area

- Dispatch **Scout** to locate the relevant files and understand the code surrounding the fix
- Pass the developer's fix description to Scout so it knows where to look
- Wait for Scout to return a focused map of the affected area

### 2. Apply the fix

- Dispatch **executor agent** with:
  1. The developer's fix description (exact wording)
  2. Scout's map of affected files
- No change docs required — the executor works directly from the description and Scout context
- After executor returns, log the run: `cx agent-run log --type executor --session <session_id> --status <status> --summary "..."`

### 3. Offer review (optional)

- Use `AskUserQuestion` to ask the developer: "Fix applied. Want a review?"
- If yes: dispatch **Reviewer** with the executor's changes and the original fix description as acceptance criteria
- If no: session is complete

## Rules

- Never create change docs (no proposal.md, design.md, tasks.md, no cx decompose)
- Never dispatch Primer — FIX mode loads no project context intentionally
- Never dispatch Planner — FIX mode skips planning entirely
- Never save memory (no cx memory save, no cx memory session, no cx memory decide)
- Always dispatch Scout before the executor — the executor needs file locations
- If the fix description grows in scope (multiple unrelated files, architectural changes), stop and redirect to BUILD mode
- FIX mode is for localized changes only — one area, one concern
