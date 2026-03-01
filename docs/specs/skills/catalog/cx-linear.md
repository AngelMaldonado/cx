# Skill: cx-linear

## Description
Teaches the Master and Contractor how to integrate with Linear for task tracking. CX does not call the Linear API directly — all Linear communication happens through the agent's MCP server. This skill defines the workflow for creating issues, updating status, and referencing them in tasks.md.

## Triggers
- Developer mentions a task that should be tracked ("create an issue for this", "add this to Linear")
- Agent is filling in tasks.md for a change and needs to create Linear issues
- Developer asks about task status ("what's the status of CUB-140?")

## Steps
1. Verify the Linear MCP server is available. If not, tell the developer: "Linear MCP server is not configured. Run `cx doctor` to check MCP dependencies."
2. To create an issue, use the Linear MCP server's create function with:
   - Title: concise task description
   - Description: link to the change proposal (e.g., "See docs/changes/<name>/proposal.md")
   - Team: inferred from project context or asked from developer
   - Labels: relevant tags
3. After creating the issue, add it to `docs/changes/<name>/tasks.md` in the format:
   ```
   ## Linear Issues
   - PROJ-NNN: <task description>
   ```
4. To check issue status, use the Linear MCP server's query function.
5. When a task is completed, update the Linear issue status via MCP. Do not modify tasks.md — it's a reference document, not a checklist.

## Rules
- Never bypass the MCP server to call Linear's API directly
- Always include a reference back to the change's proposal.md in issue descriptions
- If the Linear MCP server is unavailable, still create the tasks.md entries with placeholder refs (e.g., `- PENDING: <task description>`) and tell the developer to add Linear refs later
- Do not create Linear issues for trivial subtasks — only for meaningful work units that deserve tracking
- The agent does not manage Linear sprints, priorities, or assignments — only issue creation and status queries
