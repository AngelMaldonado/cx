# Skill: cx-memory

## Description
Manage project memory files. Memories are structured observations, decisions, and session logs stored in `docs/memory/`.

## Triggers
- Developer makes a significant decision
- Important observation about the codebase
- End of a work session
- Developer asks to remember something

## Steps
1. Determine memory type: observation, decision, or session
2. Create the memory file with proper frontmatter
3. Include required sections for the memory type
4. Cross-reference related memories if applicable

## Rules
- Memories must have valid YAML frontmatter
- Decisions require: context, options considered, decision, rationale
- Observations require: what was observed, implications
- Session logs require: goals, progress, blockers, next steps
