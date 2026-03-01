# Spec: Agent Orchestration

CX uses a hierarchical agent model where the developer communicates with a single **Master** agent that orchestrates all work through specialized agents. The Master never writes code itself — it classifies tasks, selects dispatch strategies, and routes work to the right agents.

All agent spawning uses the coding agent platform's native agent tool (e.g., Claude Code's `Agent` tool). No formal team structure or persistent agent state — each agent invocation is self-contained with its own context window.

---

## Agent Hierarchy

```
Developer
  └── Master (primary chat agent — pure orchestrator)
        │
        ├── Primer (context priming — standalone, disposable)
        │     └── Conflict-resolver (standalone, spawned by Primer if conflicts exist)
        │
        ├── Reviewer (post-implementation review — standalone)
        │
        ├── Direct dispatch (simple tasks)
        │     └── Any single agent: Scout, Contractor, etc.
        │
        └── Team dispatch (complex tasks)
              └── Supervisor (team lead for this task)
                    ├── Scout (read-only codebase exploration)
                    └── Contractor (foreman — spawns Workers)
                          └── Workers (ephemeral, focused implementation)
```

---

## Agent Roles

### Master

The developer's sole conversational partner. Pure orchestrator — it understands intent, classifies tasks, and dispatches the right agents to do the work.

**Responsibilities:**
- Receive the developer's message, understand what they want
- Spawn Primer at session start for context loading
- Classify tasks and choose dispatch strategy (direct or team)
- Pass primed context to dispatched agents
- Synthesize results from agents and present to developer
- Save session summaries via `cx memory session` at session end

**Constraints:**
- Never reads code files directly
- Never writes code or modifies implementation files
- Never calls `cx context` — that's the Primer's job
- All work delegated through agent spawning

**Skills used:** cx-memory, cx-brainstorm, cx-refine, cx-change, cx-linear, cx-doctor

---

### Primer (Standalone)

Spawned by Master at every session start. Loads, evaluates, and distills project knowledge so the Master starts with exactly the right context.

**Responsibilities:**
- Classify session mode: CONTINUE | BUILD | PLAN
- Call `cx context --mode <mode>` to get the context map
- Evaluate relevance, call `cx context --load` for specifics
- Check for `.cx/conflicts.json` — spawn Conflict-resolver if present
- Distill ~500-800 tokens of focused context
- Return structured context block to Master

**Constraints:**
- Disposable context window — thrown away after priming
- Only communicates with Master (returns context, nothing else)
- Never modifies files
- Never interacts with the developer directly

**Skills used:** cx-prime

See [context-priming spec](../context-priming/spec.md) for full design.

---

### Conflict-resolver (Standalone)

Spawned by Primer (not Master) when `.cx/conflicts.json` exists. Resolves semantic conflicts between team members' memories that arrived via `git pull`.

**Responsibilities:**
- Read `.cx/conflicts.json` and load conflicting entity pairs
- Classify each conflict: genuine conflict, complementary info, or duplicate
- Interview the developer (via AskUserQuestion) for genuine conflicts only
- Write resolution decisions via `cx memory decide`
- Return resolution summary to Primer

**Constraints:**
- Only spawned by Primer, never by Master or other agents
- Only communicates with Primer (returns summary)
- Can interact with developer only for conflict interviews
- Never modifies implementation files

**Skills used:** cx-conflict-resolve

See [conflict-resolution spec](../conflict-resolution/spec.md) for full design.

---

### Reviewer (Standalone)

Spawned by Master after implementation work completes. Reviews `git diff` against the change's spec/plan files, evaluates code quality, and verifies test coverage. Acts as a quality gate before work is presented to the developer as done.

**Responsibilities:**
- Read `git diff` to understand all changes made
- Compare changes against the change's proposal.md, design.md, and relevant canonical specs
- Evaluate code quality: does the code follow existing patterns and conventions?
- Verify tests exist for new/modified functionality and that they pass
- Produce a structured review report: alignment issues, quality concerns, test gaps
- Return the review to Master with a pass/fail recommendation

**Constraints:**
- Read-only: never modifies code or files
- Never interacts with the developer directly — returns review to Master
- Spawned by Master, not by Supervisor or Contractor
- Does not re-implement or suggest alternative implementations — only identifies issues
- If the review fails, Master decides whether to dispatch fixes (via Contractor) or present issues to the developer

**Skills used:** cx-review

---

### Scout (Read-Only Explorer)

Read-only codebase exploration agent. Can read files, search patterns, grep code, and use `cx search` — but never modifies anything. Replaces the former "reverse-engineering subagent" with a broader mandate.

**Responsibilities:**
- Explore codebase structure, map what exists
- Answer questions about code (how does X work, where is Y defined)
- Find patterns, conventions, and dependencies
- Build understanding that feeds into implementation planning
- Return structured findings to the requesting agent

**Constraints:**
- Read-only: can read files, glob, grep, use `cx search`, but never writes
- Never modifies files, never runs destructive commands
- Can be dispatched by Master (direct) or Supervisor (team)
- Returns findings — never makes implementation decisions

**Skills used:** cx-scout

See [scout spec](../scout/spec.md) for full design.

---

### Supervisor (Team Lead)

Spawned by Master for complex tasks that require coordination between exploration and implementation. Manages a team of Scout + Contractor for a single focused task.

**Responsibilities:**
- Receive task description and primed context from Master
- Plan team composition: does this need exploration? Implementation? Both?
- Spawn Scout for codebase research (if needed)
- Spawn Contractor for implementation (when ready)
- Coordinate dependencies between Scout findings and Contractor work
- Report results, observations, and completion status to Master

**Constraints:**
- Never writes code directly — coordinates agents who do
- Never interacts with the developer — all communication through Master
- Scoped to one task — doesn't persist across tasks
- If a Worker reports a blocker, escalates to Master

**Skills used:** cx-supervise

---

### Contractor (Foreman)

Spawned by Supervisor for implementation work. Decomposes tasks into discrete subtasks, spawns Workers for each, reviews their output. The Contractor is a foreman — it manages the work but doesn't do it.

**Responsibilities:**
- Receive implementation task and context from Supervisor
- Decompose into discrete subtasks (one per file or concern)
- Spawn Workers with specific instructions, files, and context
- Review Worker output for correctness
- Run checks (tests, linting) after Workers complete
- Save observations via `cx memory save` for non-obvious discoveries
- Report completion to Supervisor

**Constraints:**
- Never writes code directly — always delegates to Workers
- Never interacts with the developer — communicates only with Supervisor
- If decomposition reveals the task is larger than expected, reports to Supervisor before spawning more Workers
- Keeps Workers focused: one subtask per Worker

**Skills used:** cx-contract, cx-memory, cx-change, cx-linear

---

### Workers (Ephemeral)

Spawned by Contractor for specific implementation subtasks. The only agents in the hierarchy that actually write code and modify files. Each Worker has a focused scope and is discarded after completion.

**Responsibilities:**
- Receive a specific subtask: file(s) to modify, exact change to make, relevant context
- Implement the change
- Report what was done back to Contractor

**Constraints:**
- Focused scope — one subtask, one concern
- Receives inline instructions from Contractor (no dedicated skill)
- Disposable — spawned per task, discarded after completion
- No access to the full project context — only what Contractor provides

**Skills used:** None (receives inline instructions)

---

## Dispatch Strategies

### Direct Dispatch

Master spawns a single agent directly. No Supervisor, no team structure.

**When to use:**
- Task is clearly single-scope, no coordination needed
- Quick question about code → Scout
- Simple fix or small change → Contractor (spawns one Worker)
- Health check → Master runs `cx doctor` itself

**Flow:**
```
Master → single agent → result back to Master
```

**Examples:**
- "What does the auth middleware do?" → Master spawns Scout
- "Fix the typo on line 42 of server.go" → Master spawns Contractor
- "Run the tests" → Master spawns a single Worker

---

### Team Dispatch

Master spawns a Supervisor who builds and manages a team.

**When to use:**
- Task requires both understanding code and modifying it
- Multiple files or modules affected
- Task scope is unclear and needs exploration first
- Coordination between exploration and implementation matters

**Flow:**
```
Master → Supervisor → Scout (explore) + Contractor (implement) → Workers
       ↑                                                          │
       └──────────────── results cascade back ────────────────────┘
```

**Examples:**
- "Refactor the auth module" → Supervisor coordinates Scout + Contractor
- "Add BLE pairing support" → Scout explores existing code, Contractor implements
- "Fix the race condition in the worker pool" → Scout finds the issue, Contractor fixes it

---

### Decision Criteria

Master classifies each task using these heuristics:

```
Developer's task
    │
    ├── Single file, obvious change?
    │   └── YES → Direct dispatch (Contractor)
    │
    ├── Question about code, no changes needed?
    │   └── YES → Direct dispatch (Scout)
    │
    ├── Multi-file change, needs exploration first?
    │   └── YES → Team dispatch
    │
    ├── Architectural work, broad impact?
    │   └── YES → Team dispatch
    │
    └── Ambiguous → Team dispatch (safer default)
```

---

## Spawning Mechanism

All agent spawning uses the coding agent platform's native agent tool. Each agent is passed:

1. **Role description** — from the agent's skill file or inline
2. **Primed context** — relevant subset of what the Primer returned
3. **Task description** — the specific task or question
4. **Tool restrictions** — Scout gets read-only tools; Workers get full tools

Agents do not share context windows. Each agent starts fresh with only what it's given. Results flow back through the hierarchy: Workers → Contractor → Supervisor → Master → Developer.

---

## Skill Mapping

| Skill | Agent(s) | When |
|-------|----------|------|
| cx-prime | Primer | Every session start |
| cx-conflict-resolve | Conflict-resolver | When `.cx/conflicts.json` exists |
| cx-review | Reviewer | Post-implementation quality gate |
| cx-scout | Scout | Codebase exploration (direct or team) |
| cx-supervise | Supervisor | Team dispatch from Master |
| cx-contract | Contractor | Implementation coordination from Supervisor |
| cx-memory | Master, Contractor | During work (observations) or session end (summaries) |
| cx-brainstorm | Master | New idea exploration (PLAN mode) |
| cx-refine | Master | Masterfile iteration |
| cx-change | Master, Contractor | Change lifecycle management |
| cx-linear | Master, Contractor | Linear integration via MCP |
| cx-doctor | Master | Project health checks |

Workers receive inline instructions from Contractor — no dedicated skill file.

---

## Context Flow

```
Developer writes opening message
    │
    ▼
Master receives it
    │
    ├── Spawns Primer (passes developer's message)
    │       │
    │       ├── Checks .cx/conflicts.json
    │       │   └── If exists → spawns Conflict-resolver → resolves → returns
    │       │
    │       ├── Classifies mode: CONTINUE | BUILD | PLAN
    │       ├── Calls cx context --mode <mode>
    │       ├── Evaluates relevance, calls cx context --load selectively
    │       ├── Distills ~500-800 tokens of focused context
    │       └── Returns structured context block to Master
    │
    ▼
Master receives primed context
    │
    ├── Classifies task → direct or team dispatch
    │
    ├── Direct dispatch:
    │   └── Spawns agent with task + relevant context → receives result
    │
    └── Team dispatch:
        └── Spawns Supervisor with task + relevant context
                │
                ├── Supervisor spawns Scout (if exploration needed)
                │   └── Scout returns findings
                │
                └── Supervisor spawns Contractor (with Scout findings)
                        │
                        ├── Contractor decomposes into subtasks
                        ├── Spawns Workers for each subtask
                        ├── Workers implement and return results
                        └── Contractor reports to Supervisor
                                │
                                └── Supervisor reports to Master
    │
    ▼
Master spawns Reviewer (post-implementation quality gate)
    │
    ├── Reviewer reads git diff
    ├── Compares against specs/plan files
    ├── Checks code quality and test coverage
    └── Returns review report to Master
            │
            ├── Pass → Master presents results to Developer
            └── Fail → Master dispatches fixes or presents issues
```
