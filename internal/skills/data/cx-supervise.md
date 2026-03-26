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

## Worktree-Aware Dispatch

When coordinating implementation work across multiple independent subtasks:

1. **Create worktrees before dispatching:** Run `cx worktree create <change>-task-N` for each independent subtask. Each executor receives its worktree path as working directory.
2. **Dispatch executors in parallel:** Independent subtasks run concurrently, each isolated in their own worktree.
3. **Dispatch Merger after executors complete:** Pass the Merger agent the list of task branch names, dependency order, the change's proposal.md and design.md, and the target branch.
4. **Clean up after merge:** Run `cx worktree cleanup <change>` once the Merger returns successfully.

Use worktrees for 2 or more independent subtasks. For a single subtask or explicitly dependent tasks, execute sequentially on the main working tree.

## Rules
- Each subtask must have clear acceptance criteria
- Agents should work on independent, non-overlapping areas
- Report progress at meaningful milestones
- Escalate blockers to the developer promptly
- Always pass session_id to sub-agent dispatches for agent-run tracking continuity
- At session end: `cx memory session --goal "..." --accomplished "..." --next "..." --change <name> --discoveries "..."`
