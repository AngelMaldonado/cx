# Specs

This directory describes the current behavior of the CX system. Each subdirectory covers one functional area.

---

## Spec Areas

| Area | Description | Status |
|------|-------------|--------|
| [brainstorm](brainstorm/spec.md) | Masterfile creation, skill-guided refinement, and decompose into change structure | Active |
| [change-lifecycle](change-lifecycle/spec.md) | Change creation, delta specs, agent-assisted archive merge, spec area auto-creation | Active |
| [conflict-resolution](conflict-resolution/spec.md) | Semantic conflict detection after git pull, dedicated resolver subagent, developer interview | Active |
| [context-priming](context-priming/spec.md) | Primer subagent architecture, `cx context` command, mode-based loading and distillation | Active |
| [doctor](doctor/spec.md) | Project health validation: docs/ structure, memory files, index, git hooks, MCP, skills | Active |
| [init](init/spec.md) | Project bootstrapping: scaffolding, agent selection, DIRECTION.md setup, git hooks | Active |
| [memory](memory/spec.md) | Data model, storage, parsing, unified deprecation, and query behavior for all memory types | Active |
| [reverse-engineering](reverse-engineering/spec.md) | Subagent-based codebase exploration, binary as indexing helper, structured findings | Active |
| [scenarios](scenarios/spec.md) | Given/When/Then scenario format for optional `scenarios.md` files | Active |
| [search](search/spec.md) | Unified `cx search` with FTS5, scope filters, `--personal`, `cx context --load` | Active |
| [session-modes](session-modes/spec.md) | The three session modes (CONTINUE, BUILD, PLAN) and their classification rules | Active |
| [skills](skills/spec.md) | Skill protocol (format, validation) and [catalog](skills/catalog/) of all cx-* skill files | Active |

---

## Spec Format

Each spec area contains:
- **`spec.md`** — Requirements: WHAT the system does
- **`scenarios.md`** *(optional)* — Given/When/Then scenarios: HOW it behaves (see [scenarios spec](scenarios/spec.md) for format)

Active changes that modify a spec area create a `delta.md` in `docs/changes/<name>/specs/<area>/`. The delta is merged into the canonical spec on `cx change archive`.
