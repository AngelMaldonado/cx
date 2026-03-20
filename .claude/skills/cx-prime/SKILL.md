---
name: cx-prime
description: Prime the AI agent context with relevant project knowledge. Activate at the start of a new conversation, when switching project areas, or when the agent needs background on a topic.
---

# Skill: cx-prime

## Description
Prime the AI agent context with relevant project knowledge. Loads key documents, recent memories, and active changes to establish working context.

## Triggers
- Start of a new conversation or session
- Developer switches to a different area of the project
- Agent needs background on a specific topic

## Steps

1. Classify session mode from the developer's opening message: CONTINUE, BUILD, or PLAN
2. Load project config from `.cx/cx.yaml` for context and rules
3. Load mode-specific memory:
   - **BUILD**: `cx memory list --type decision` + `cx memory list --type observation --recent 7d` + personal notes
   - **CONTINUE**: `cx memory list --type session --change <name>` (last session first) + `cx memory search --change <name>`
   - **PLAN**: personal preference notes only — no project memory loaded (clean-slate creative mode)
4. Load active change docs if applicable (`cx change status`)
5. Distill and present summary to the Master

## Rules
- Primer is read-only and disposable — never writes memory, never modifies files
- Always include project config context when available
- If docs/specs/ is empty or missing, signal empty state to the Master and recommend Scout → Planner bootstrapping
- Relevance filtering: prioritize recent decisions and observations over old ones
