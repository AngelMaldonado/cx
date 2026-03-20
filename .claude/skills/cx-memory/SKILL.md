---
name: cx-memory
description: Manage project memory files including observations, decisions, and session logs. Activate when the developer makes a significant decision, records an observation, or ends a work session.
---

# Skill: cx-memory

## Description

Manage project memory using the SQLite-backed cx memory commands. Memory is the persistence layer for observations, decisions, sessions, and agent interactions.

## Triggers

- Developer says "remember this", "save observation", "record decision", "session summary"
- End of a work session
- Significant discovery or technical decision during implementation

## Steps

1. Identify the memory type: observation, decision, session, or agent interaction
2. Use the appropriate command:
   - **Observation**: `cx memory save --type observation --title "..." --content "..." [--change C] [--tags t1,t2]`
   - **Decision**: `cx memory decide --title "..." --context "..." --outcome "..." --alternatives "..." --rationale "..." [--change C]`
   - **Session summary**: `cx memory session --goal "..." --accomplished "..." --next "..." [--change C] [--discoveries "..."]`
   - **Search**: `cx memory search "query" [--type T] [--change C] [--all-projects]`
   - **List**: `cx memory list [--type T] [--change C] [--recent 7d]`
   - **Team sync**: `cx memory push` (export to docs/memory/) / `cx memory pull` (import from docs/memory/)
   - **Link**: `cx memory link <id1> <id2> --relation related-to|caused-by|resolved-by|see-also`
3. For agent run tracking: `cx agent-run log --type <agent_type> --session <id> --status <s> --summary "..."`

## Rules

- Decisions default to visibility `project` (shared with team via push)
- Observations default to visibility `project`
- Sessions default to visibility `personal` (local only unless explicitly pushed)
- Always include `--change <name>` when working within an active change
- Session summaries MUST include `--next` — this is how CONTINUE mode recovers state
- Use `cx memory push` after significant observations or decisions to share with team
- Use `cx memory pull` after `git pull` to import teammates' memories
