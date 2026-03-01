---
name: cx-brainstorm
description: Create and decompose masterfiles for brainstorming. Activate when the developer wants to explore a new idea or turn a masterfile into a structured change.
---

# Skill: cx-brainstorm

## Description
Create and decompose masterfiles for brainstorming. A masterfile is a freeform document in `docs/masterfiles/` where ideas are explored before being structured into a formal change.

## Triggers
- Developer wants to explore a new idea
- Developer wants to turn a masterfile into a structured change
- Developer references an existing masterfile by name

## Steps
1. Run `cx brainstorm <name>` to create a masterfile
2. Work with the developer to fill in sections: Problem, Vision, Open Questions, Constraints, Notes
3. When ready, run `cx decompose <name>` to structure into a change

## Rules
- Masterfile names must be kebab-case, max 40 characters
- Never decompose without developer approval
- After decompose, the masterfile is archived
