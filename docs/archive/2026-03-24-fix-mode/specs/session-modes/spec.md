# Delta Spec: session-modes

Change: fix-mode

## ADDED Requirements

### Mode: FIX

**When**: Developer wants a quick, localized code change that bypasses the full change lifecycle.

**Signal phrases**: "fix", "patch", "tweak", "quick fix", "one-liner", "just change X to Y", "rename this", "update this value", or any request for a small localized change with no architectural implications.

### 3-step flow

```
1. Scout     → always dispatched; receives the developer's fix description;
               returns a focused map of the affected code area
2. Executor  → receives fix description + Scout's map;
               applies the change; no change docs required
3. Reviewer  → optional; Master uses AskUserQuestion("Fix applied. Want a review?")
```

### Memory touchpoints

FIX mode intentionally creates no memory artifacts.

| Step | Action |
|------|--------|
| Each agent dispatch | Master logs agent run via `cx agent-run log` |
| Session end | No `cx memory session` — git log is the only artifact |

### What the Master skips in FIX mode

- Primer dispatch (no project context load)
- Requirements gathering rounds
- Planner dispatch
- `cx decompose`, `cx change new`
- proposal.md, design.md, tasks.md
- Archive flow and spec merging
- `cx memory save`, `cx memory decide`, `cx memory session`

### Scope guard

If the fix description grows in scope (multiple unrelated files, architectural implications), the Master stops FIX mode and redirects the developer to BUILD mode.

### Classification additions

The Mode Classification decision tree gains a FIX branch. FIX is checked before the BUILD/PLAN check:

```
Developer's opening message
    │
    ├── References an existing change by name?
    │   └── YES → CONTINUE
    │
    ├── Mentions "continue", "resume", "pick up", "where were we"?
    │   └── YES → CONTINUE
    │
    ├── Mentions "plan", "brainstorm", "think about", "redesign", "architecture"?
    │   └── YES → PLAN
    │
    ├── Small, localized fix? ("fix", "patch", "tweak", "quick fix", "one-liner",
    │   "just change X to Y", "rename this", "update this value")
    │   └── YES → FIX
    │
    ├── Describes something new to build, add, create, or implement?
    │   └── YES → BUILD
    │
    └── Unclear → BUILD (safest default)
```

### CLAUDE.md dispatch table addition

The dispatch table adds a 4th row:

| Mode | Skill | When |
|------|-------|------|
| **FIX** | `cx-fix` | Developer wants a quick, localized code change |

## MODIFIED Requirements

### The Three Modes section header

The existing "## The Three Modes" section title and introductory text (which says "three modes") must be updated to reflect that there are now four modes. The FIX mode entry is added to the flow diagram:

```
        ├── "just rename this variable"
        │         ▼
        │    FIX — Minimal: Scout maps area, Executor applies change,
        │          Reviewer optional, no change docs
```

## REMOVED Requirements

None.
