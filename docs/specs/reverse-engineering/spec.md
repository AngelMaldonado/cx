# Spec: Reverse Engineering

Reverse engineering is how CX bootstraps documentation for an existing codebase. It uses a **subagent** (same pattern as the primer) to explore the codebase and docs, with the binary providing fast indexing and lookup to make navigation efficient.

---

## Architecture

```
Main Agent: "Help me understand this codebase"
    │
    ▼
Spawns reverse-engineering subagent
    │ passes: developer's question or goal
    ▼
Subagent (disposable context window)
    │
    ├── 1. Calls cx index rebuild to ensure index is current
    ├── 2. Calls cx search to find relevant files/docs
    ├── 3. Reads code files, package manifests, config files
    ├── 4. Builds understanding: architecture, patterns, flows
    ├── 5. Produces structured findings
    └── 6. Returns report to main agent
            │
            ▼
Main Agent uses findings (or writes to docs/)
```

The subagent's context window is disposable — it can load dozens of source files, reason about structure and patterns, and return a focused summary.

---

## Why a Subagent?

Understanding a codebase requires reading many files, which consumes significant context. The main agent shouldn't spend its context window on exploration when it needs that space for the actual work. The subagent:

- Reads broadly (10-50+ files) without impacting the main agent
- Reasons about patterns, structure, and relationships
- Returns only the relevant findings the main agent needs
- Can be spawned multiple times for different questions

---

## The Binary's Role — Indexing Helper

The binary does not do static analysis or code understanding. It provides fast lookup so the subagent can navigate efficiently:

### What the binary indexes

`cx index rebuild` indexes all of `docs/` (specs, memories, architecture, changes). For codebase navigation, the subagent uses the agent's native file tools (read, glob, grep) — the binary doesn't index source code.

### How the subagent uses the binary

```bash
# Find docs related to a topic
cx search "authentication"

# Load a specific spec
cx context --load spec user-management

# Check what's already documented
cx context --load architecture
```

The binary makes docs navigation instant. The subagent uses its own file tools for code navigation.

---

## The Skill — cx-reverse

The skill teaches the subagent how to explore systematically. It's not a step-by-step script — it's guidelines for thorough exploration.

### Exploration strategy

1. **Start with manifests** — `package.json`, `go.mod`, `Cargo.toml`, `pyproject.toml`. Identify languages, frameworks, dependencies.

2. **Map the directory structure** — understand the project layout, identify conventional directories (src/, lib/, cmd/, internal/, tests/).

3. **Find entry points** — main files, route definitions, CLI command registration, exported modules.

4. **Trace key flows** — follow a request from entry point through middleware, handlers, services, data layer.

5. **Identify patterns** — error handling conventions, logging approach, dependency injection, testing patterns.

6. **Read existing docs** — check docs/, README, architecture decisions, comments. Note gaps.

### Output structure

The subagent returns findings in a structured format:

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
- <file:line> — <description>

## Patterns & Conventions
- <pattern>: <description>

## Gaps
- <what's undocumented or unclear>
```

---

## Use Cases

### Bootstrap docs/ for a new CX project

When running `cx init` on an existing codebase, the agent can spawn the reverse-engineering subagent to pre-fill `docs/overview.md` and `docs/architecture/index.md` with what it discovers.

### Answer specific questions

The main agent delegates targeted questions:
- "How does authentication work in this codebase?"
- "What database queries does the /devices endpoint make?"
- "Where is the MQTT client configured?"

The subagent explores, finds the answer, and returns it — the main agent never reads those files.

### Onboard new team members

A new developer asks "how does this system work?" The agent spawns the subagent to produce a comprehensive overview, drawing from both the codebase and existing docs/.

---

## Commands

No dedicated binary commands for reverse engineering. The subagent uses existing commands:

| Command | How the subagent uses it |
|---------|------------------------|
| `cx search "query"` | Find relevant docs content |
| `cx context --load <resource>` | Load specific specs or architecture docs |
| `cx index rebuild` | Ensure the docs index is current |

All code navigation uses the agent's native file tools (read, glob, grep).
