# Spec: Scout

Read-only codebase exploration agent. Replaces the former "reverse-engineering subagent" with a broader mandate: any task requiring reading and understanding code without modifying it.

See [orchestration spec](../orchestration/spec.md) for how Scout fits into the agent hierarchy.

---

## Role

- Read-only codebase exploration agent
- Replaces the former "reverse-engineering subagent"
- Broader mandate: any task requiring reading and understanding code without modifying it
- Can be dispatched directly by Master or as part of a team under a Supervisor

---

## Capabilities

### Exploration Strategy

1. **Start with manifests** -- `package.json`, `go.mod`, `Cargo.toml`, `pyproject.toml`. Identify languages, frameworks, dependencies.

2. **Map the directory structure** -- understand the project layout, identify conventional directories (src/, lib/, cmd/, internal/, tests/).

3. **Find entry points** -- main files, route definitions, CLI command registration, exported modules.

4. **Trace key flows** -- follow a request from entry point through middleware, handlers, services, data layer.

5. **Identify patterns** -- error handling conventions, logging approach, dependency injection, testing patterns.

6. **Read existing docs** -- check docs/, README, architecture decisions, comments. Note gaps.

### Binary-Assisted Navigation

The binary does not do static analysis or code understanding. It provides fast lookup so Scout can navigate efficiently:

- `cx index rebuild` indexes all of `docs/` (specs, memories, architecture, changes)
- `cx search` provides FTS5 full-text search across indexed docs
- `cx context --load` loads specific specs, architecture docs, or memory entities

The binary makes docs navigation instant. Scout uses its own file tools for code navigation.

---

## Dispatch Modes

### Direct Dispatch

```
Master --> Scout --> findings back to Master
```

Used for:
- Targeted questions ("How does authentication work?")
- Understanding a specific flow ("What happens when a device connects?")
- Pre-implementation research ("What patterns does the codebase use for error handling?")
- Answering "where is X defined?" questions

### Team Dispatch

```
Supervisor --> Scout --> findings to Supervisor (shared with Contractor)
```

Used for:
- Broad exploration that feeds into implementation work
- Understanding existing code before refactoring
- Mapping dependencies before making changes
- Building context that the Contractor needs for safe implementation

---

## Output Format

Scout returns findings in a structured format:

```markdown
## Tech Stack
- Languages: <detected>
- Frameworks: <detected>
- Databases: <detected>
- Key dependencies: <notable libraries>

## Architecture
- Pattern: <monolith/microservices/serverless/etc.>
- Key components: <list with brief descriptions>
- Data flow: <how data moves through the system>

## Key Entry Points
- <file:line> -- <description>

## Patterns & Conventions
- <pattern>: <description>

## Gaps
- <what's undocumented or unclear>
```

For targeted questions, Scout returns a focused answer rather than the full template -- the format adapts to the question asked.

---

## Commands Used

| Command | How Scout uses it |
|---------|-------------------|
| `cx search "query"` | FTS5 search across docs/ to find relevant documented content |
| `cx context --load <resource>` | Load specific specs, architecture docs, or memory entities |
| `cx index rebuild` | Ensure the docs index is current before searching |

All code navigation uses native file tools:
- **Read** -- read file contents
- **Glob** -- find files by pattern
- **Grep** -- search for patterns in code

Scout never modifies files. All tools are used in read-only mode.

---

## Constraints

- Read-only: can read files, glob, grep, use `cx search`, but never writes
- Never modifies files, never runs destructive commands
- Returns findings -- never makes implementation decisions
- Does not interact with the developer directly -- returns results to the dispatching agent (Master or Supervisor)

---

## Skill

**Skills used:** cx-scout

The cx-scout skill teaches Scout how to explore systematically. It is not a step-by-step script -- it provides guidelines for thorough exploration adapted to the question being asked.
