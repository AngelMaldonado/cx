# CX

> A deterministic orchestration binary that unifies spec-driven development, shared team memory, and multi-agent coordination. No config files. The binary + docs/ is all you need.

---

## Problem Statement

Modern software teams use coding agents (Claude Code, Gemini, Codex) alongside issue trackers, specs, and documentation. But these tools live in separate worlds:

- Agents have no memory between sessions — every session starts cold
- Specifications drift from implementation — no mechanism keeps them in sync
- Team knowledge (decisions, discoveries, gotchas) lives in chat logs, Slack threads, and people's heads
- Every project has a different config format, different "where does stuff live" answer
- Multiple agents on the same team have no shared context

The result: agents repeat mistakes, teams re-debate settled questions, specs become outdated on arrival, and new team members spend days just figuring out why things are the way they are.

## Pain Points

- **No session continuity**: An agent that spent 3 hours on a problem starts fresh next session — no memory of what it tried, what failed, or what the next step was
- **Config proliferation**: Every tool wants its own config file (`cx.yaml`, `.cxrc`, `config.json`) that becomes the real source of truth and drifts from docs
- **Knowledge loss**: When a developer discovers that MQTT silently drops messages over 256KB, that insight lives in their head — the next person hits the same wall
- **Decision amnesia**: Technical decisions get made, but the *why* gets lost — future developers second-guess and re-debate settled choices
- **Spec rot**: Specs describe the intended system; the implemented system diverges; no one notices until something breaks
- **Agent fragmentation**: Teams using Claude Code and Gemini side-by-side have no shared context or conventions

## Target State

A single `cx` binary that any project can adopt:

1. **Developer talks to their coding agent** — the agent handles everything else
2. **`docs/` is the single source of truth** — specs, changes, architecture, all in plain markdown
3. **Memory persists** — observations, decisions, and session summaries accumulate in `docs/memories/`, committed to git, shared with the team
4. **Specs stay current** — changes track what they modify, archive merges deltas back to specs
5. **Agents are interchangeable** — Claude, Gemini, Codex all get the same skills, same context, same conventions
6. **No config files** — the binary infers everything from `docs/` content and environment variables
7. **Conflicts detected automatically** — after every `git pull`, semantic conflicts between team members' memories and decisions are surfaced and resolved before the next session
8. **One command updates everything** — `cx upgrade` updates the binary via Homebrew and syncs skills across all registered projects

## Solution

```
Developer → Master (orchestrator) → dispatches agents → read skills → call cx → read/write docs/
                │                        │
                │                        ├── Primer (context priming)
                │                        ├── Scout (codebase exploration)
                │                        └── Supervisor → Contractor → Workers (implementation)
                │
                └──→ MCP servers (Linear, etc.) → external APIs

Developer → cx dashboard (TUI) → read-only view of docs/ + memory
```

The binary has three entry points for developers:
- `cx init` — one-time project setup, generates agent configs and skills
- `cx upgrade` — updates the binary via Homebrew and syncs skills across all registered projects
- `cx dashboard` — TUI for visibility into specs, changes, memory, team sync

Everything else is agent-driven: brainstorming new features, decomposing work into changes, tracking progress, updating specs, and saving institutional knowledge.

## Success Criteria

- An agent starting a new session recovers full context within 500 tokens, without reading chat history
- Team members can search and find why any architectural decision was made
- Spec files stay synchronized with what's actually implemented
- Switching between agents (Claude → Gemini) requires no reconfiguration
- A developer can `cx init` on a new project and be productive in under 5 minutes
- A new team member can understand the system's history through `cx memory search` and `docs/`

## Constraints & Assumptions

- **No CX server** — the binary is fully local; team sync is just git (push/pull). Git hooks trigger index rebuild and conflict detection.
- **No agent lock-in** — works with any coding agent that can read files and run shell commands
- **Markdown only** — docs/ is pure markdown, no frontmatter schemas, no metadata files
- **Linear via MCP** — task tracking happens through the agent's MCP server, not through cx directly
- **Go binary** — single static binary, no runtime dependencies, cross-compiled for macOS/Linux/Windows
- **Git is the version system** — no separate versioning for specs; git history is the audit trail
