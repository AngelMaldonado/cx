---
name: cx-supervise
description: Coordinate multi-agent workflows with task distribution and progress tracking. Activate when a complex task requires multiple agents or the developer wants to parallelize work.
---

# Skill: cx-supervise

## Description
Coordinate multi-agent workflows. Manages task distribution, progress tracking, and result aggregation across multiple AI agents.

## Triggers
- Complex task requires multiple agents
- Developer wants to parallelize work
- Multi-step workflow needs coordination

## Steps
1. Break the task into independent subtasks
2. Assign subtasks to appropriate agents
   - After each sub-agent returns, log the run: `cx agent-run log --type <agent_type> --session <session_id> --status <status> --summary "..."`
   - Pass `session_id` to each sub-agent dispatch prompt
3. Monitor progress and handle blockers
4. Aggregate results and report to developer

## Rules
- Each subtask must have clear acceptance criteria
- Agents should work on independent, non-overlapping areas
- Report progress at meaningful milestones
- Escalate blockers to the developer promptly
- Always pass session_id to sub-agent dispatches for agent-run tracking continuity
