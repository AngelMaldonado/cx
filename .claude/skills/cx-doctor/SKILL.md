---
name: cx-doctor
description: Run project health checks and fix common issues. Activate when the developer asks about project health, after cx init, or when another cx command encounters an error.
---

# Skill: cx-doctor

## Description
Run project health checks and fix common issues. Validates docs/ structure, memory files, git hooks, and MCP configuration.

## Triggers
- Developer asks about project health
- Agent encounters an error from another cx command
- After cx init to verify setup
- Periodically during long work sessions

## Steps
1. Run `cx doctor` to check project health
2. Review output organized by section: docs/, memory, git hooks, MCP config
3. Each check shows pass, warning, or error
4. For fixable issues, suggest `cx doctor --fix`
5. For non-fixable issues, explain manual steps

## Rules
- Always present results before suggesting fixes
- Never run --fix without developer approval
- Doctor is read-only by default
