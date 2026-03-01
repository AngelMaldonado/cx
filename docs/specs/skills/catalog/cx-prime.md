# Skill: cx-prime

## Description
Context priming skill for the Primer. Spawned by the Master at the start of every session, this skill classifies the session mode (CONTINUE, BUILD, or PLAN), loads relevant context from the project, and distills it into a tight context block for the Master. The Primer's context window is disposable — it can load broadly and reason about relevance without polluting the Master's context.

## Triggers
- Every new agent session (always — this skill runs before any other)
- Master spawns the Primer and passes the developer's opening message

## Steps
1. Check for unresolved conflicts: read `.cx/conflicts.json`. If it exists and contains entries, spawn the Conflict-resolver (using cx-conflict-resolve skill) BEFORE proceeding. Wait for resolution to complete.
2. Read the developer's opening message and classify the session mode:
   - **CONTINUE**: Message references ongoing work, a specific change, or picking up where they left off. Look for: change names, "continue", "pick up", "where I left off", file names from recent sessions.
   - **BUILD**: Message describes a specific task to implement. Look for: "add", "fix", "implement", "build", "create", "update".
   - **PLAN**: Message is exploratory or architectural. Look for: "should we", "how would", "what if", "design", "think about", "brainstorm".
   - If ambiguous between CONTINUE and BUILD, check if there's an active change matching the message. If yes → CONTINUE. If no → BUILD.
   - If ambiguous between BUILD and PLAN, default to BUILD.
3. Run `cx context --mode <mode>` to get the context map. This returns a structured list of available resources grouped by category.
4. Evaluate each resource in the map against the developer's intent. For each resource, decide: **load** (relevant) or **skip** (noise).
5. For each resource marked "load", run `cx context --load <resource-type> <name>` to get the full content.
6. Distill all loaded content into a context block of 500-800 tokens. The output format must be:

```
## Session Context

**Mode**: CONTINUE | BUILD | PLAN
**Active change**: <name> (or "none")

### Relevant context
- <key fact or decision, one line each>
- <key fact or decision, one line each>
- ...

### Session recovery (CONTINUE only)
**Last session goal**: <goal>
**Accomplished**: <what was done>
**Next steps**: <exact pickup point>

### Active decisions
- <decision title>: <outcome, one line>
- ...

### Relevant specs
- <spec area>: <one-line summary of what's relevant>
- ...
```

7. Return this context block to the Master. The Master uses it as its starting context, then decides on dispatch strategy (direct or team). See [orchestration spec](../../orchestration/spec.md).

## Rules
- Never make changes to any files — the primer is read-only
- Never interact with the developer directly — the Primer only communicates with the Master
- Always run conflict detection (step 1) before context loading
- If `cx context --mode` returns an empty map (new project, no docs), return a minimal context block with just the mode and "New project — no existing context"
- The distilled output must be under 800 tokens — ruthlessly cut anything that isn't directly relevant to the developer's opening message
- For CONTINUE mode: always include session recovery (last session's goal, accomplished, next steps)
- For BUILD mode: always include relevant active decisions and the spec area being modified
- For PLAN mode: keep it minimal — personal preference notes and high-level project overview only
