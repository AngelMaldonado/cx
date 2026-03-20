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
1. Load project overview and specs
2. Load project config (.cx/cx.yaml) for context and rules
3. Load recent and relevant memories
4. Load active change documents if applicable
5. Present a summary of loaded context

## Rules
- Only load what is relevant — avoid context overload
- Prioritize recent decisions over old observations
- Always include project config context when available
- If docs/specs/ is empty or missing, signal empty state to the Master and recommend Scout → Planner bootstrapping
