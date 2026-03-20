---
name: orchestration
type: delta-spec
area: orchestration
change: memory-architecture
---

## ADDED Requirements

### Agent Run Logging Pattern

- Master calls `cx agent-run log` AFTER each Agent dispatch returns, atomically recording the completed run
- Subagents MAY also call `cx agent-run log` before returning their summary to record their own work
- Both patterns write to `.cx/memory.db` in the current project
- The `session_id` is passed by Master in every subagent dispatch prompt; subagents include it in their `cx agent-run log` call
- Required fields for `cx agent-run log`: `--type <agent_type>`, `--session <session_id>`, `--status <success|blocked|needs-input>`, `--summary "..."`
- Optional fields: `--artifacts <p1,p2>`, `--duration-ms <ms>`, `--prompt-summary "first 200 chars"`

### Session Tracking

- Master records a session start implicitly by initiating a session (session_id generated at start)
- Master saves session summary at session end via `cx memory session` with all required fields: `goal`, `accomplished`, `next_steps`; optional: `change_id`, `discoveries`, `blockers`, `files_touched`
- The `next_steps` field is critical for CONTINUE mode — it is the exact pickup point for the next session

### Agent Memory Contracts (formalized)

| Agent | Memory reads | Memory writes |
|-------|-------------|---------------|
| Master | Never directly — dispatches Primer | `cx memory session` at end, `cx memory decide` for requirements decisions, `cx agent-run log` after each dispatch |
| Primer | `cx memory search`, `cx memory list` | Never |
| Scout | None | Never (discoveries returned to Master; Master saves) |
| Planner | Receives primed context in prompt | `cx memory decide --change <name>`, `cx memory save --type observation` |
| Reviewer | `cx memory search --change <name>` | Never (Master saves review lessons) |
| Executor | Receives primed context in prompt | `cx memory save --type observation --change <name>` for per-task discoveries |

## MODIFIED Requirements

### Master — Memory Write Responsibility

- Previous: Master was described as saving session summaries
- Modified: Master saves session summaries AND decision records for requirements decisions AND persists Scout findings (since Scout is read-only and cannot call `cx memory save`)
- Master calls `cx agent-run log` after EVERY agent dispatch, not just selectively

### Contractor — Memory Write

- Previous spec listed `cx memory save` as a Contractor responsibility without specifying the `--change <name>` flag requirement
- Modified: Contractor (executor agent) must include `--change <name>` in all memory saves to link discoveries to the active change

## REMOVED Requirements

None. All existing orchestration requirements remain in force.
