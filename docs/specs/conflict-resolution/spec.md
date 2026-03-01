# Spec: Conflict Resolution

When teammates push memory files that semantically conflict — contradictory decisions, observations that disagree, or overlapping coverage of the same topic — the system detects and surfaces these for human resolution. Resolution is agent-assisted: a dedicated subagent identifies potential conflicts and interviews the developer to resolve them.

---

## Trigger

Conflict detection runs as part of the `post-merge` git hook flow:

```
git pull
    → post-merge hook fires
    → cx index rebuild (updates FTS5 index with new files)
    → cx conflicts detect (compares new entities against existing index)
    → if conflicts found: writes .cx/conflicts.json
    → next agent session: primer reads conflicts file, spawns conflict-resolver subagent
```

The binary handles detection (fast, deterministic). The subagent handles resolution (reasoning, interviewing).

---

## Detection — `cx conflicts detect`

The binary identifies **new entities** (files that appeared since the last index build) and compares them against the existing index. It does not use a hardcoded similarity threshold — it produces a candidate set that the subagent evaluates.

### What the binary provides

For each new entity, the binary outputs:

```json
{
  "new_entity": {
    "id": "2026-02-25T14-00-00-maria-mqtt-qos-level",
    "type": "decision",
    "title": "Use MQTT QoS 1 for telemetry",
    "tags": ["mqtt", "telemetry"],
    "specs": ["device-communication"],
    "status": "active"
  },
  "candidates": [
    {
      "id": "2026-02-20T10-00-00-angel-mqtt-qos-zero",
      "type": "decision",
      "title": "Use MQTT QoS 0 for telemetry",
      "tags": ["mqtt", "telemetry"],
      "specs": ["device-communication"],
      "status": "active",
      "match_reason": "same_tags_and_specs"
    }
  ]
}
```

### Candidate matching rules

The binary finds candidates using index queries, not semantic reasoning:

| Signal | How it's matched |
|--------|-----------------|
| Same tags | 2+ shared tags between new and existing entity |
| Same spec refs | Same `specs` frontmatter values |
| Same change | Both reference the same `change` |
| FTS5 proximity | High FTS5 rank when using the new entity's title as a query against existing entities |
| Both active decisions | Two decisions with `status: active` sharing tags or spec refs |

The binary casts a wide net. False positives are expected — the subagent filters them.

### Output

Results are written to `.cx/conflicts.json`:

```json
{
  "detected_at": "2026-02-25T14:30:00Z",
  "trigger": "post-merge",
  "conflicts": [
    {
      "new_entity": { ... },
      "candidates": [ ... ]
    }
  ]
}
```

If no candidates are found for any new entity, the file is not written (or an existing one is deleted).

---

## Resolution — Conflict-Resolver Subagent

When the primer subagent detects `.cx/conflicts.json` at session start, it spawns a **dedicated conflict-resolver subagent** before proceeding with normal context priming.

### Subagent behavior

The conflict-resolver subagent:

1. **Reads** `.cx/conflicts.json` to get the conflict candidates
2. **Loads** the full content of each entity pair using `cx context --load memory <id>`
3. **Reasons** about whether each candidate pair is a genuine semantic conflict, using its own judgment — no hardcoded rules
4. **Filters out** false positives (e.g., two observations about the same topic that don't contradict each other)
5. **For each genuine conflict**, interviews the developer using AskUserQuestion

### Interview format

The skill explicitly instructs the subagent to interview the developer. Example interaction:

```
Conflict detected between two active decisions:

1. "Use MQTT QoS 0 for telemetry" (by angel, Feb 20)
   → QoS 0 (fire-and-forget) is faster, acceptable packet loss for telemetry

2. "Use MQTT QoS 1 for telemetry" (by maria, Feb 25)
   → QoS 1 (at-least-once) ensures delivery, needed for billing-critical readings

How should this be resolved?

Options:
- Keep decision 1 (deprecate decision 2)
- Keep decision 2 (deprecate decision 1)
- Both are valid for different contexts (no conflict — keep both active)
- Write a new decision that reconciles both
```

The subagent must use the AskUserQuestion tool — it never resolves conflicts autonomously.

### Resolution actions

Based on the developer's answer, the subagent calls the appropriate `cx` command:

| Developer choice | Action |
|-----------------|--------|
| Keep A, deprecate B | `cx memory decide` with `deprecates: <B-slug>` |
| Keep B, deprecate A | `cx memory decide` with `deprecates: <A-slug>` |
| Both valid | No action — subagent removes the pair from conflicts.json |
| New reconciling decision | `cx memory decide` with `deprecates: <A-slug>` (new decision deprecates the old one it replaces) |

### Cleanup

After all conflicts are resolved (or dismissed), the subagent deletes `.cx/conflicts.json`. The primer then proceeds with normal context priming.

If the developer dismisses all conflicts without resolving them, the file is still deleted — the same conflicts won't be re-raised. They'll only resurface if new entities arrive that match the same candidates.

---

## What counts as a conflict

The subagent — not the binary — decides what's a genuine conflict. The skill provides these heuristics:

**Genuine conflicts (should surface):**
- Two `active` decisions on the same topic reaching different conclusions
- An observation that directly contradicts an active decision
- Two observations about the same system behavior reporting different facts

**Not conflicts (should dismiss):**
- Two observations about the same topic that add complementary information
- A decision and an observation that are consistent (the observation just adds context)
- Entities with overlapping tags but unrelated content
- An old entity that's already deprecated

The subagent reads the full content of both entities to make this judgment — tag/spec overlap alone is not sufficient to declare a conflict.

---

## Edge cases

### Multiple conflicts in one pull

If a pull brings 5 new files and 3 have conflicts, the subagent presents them one at a time. The developer resolves each before seeing the next.

### Conflict with a deprecated entity

If the candidate is already deprecated (`deprecated = 1` in the index), the binary doesn't include it in candidates. Only active, non-deprecated entities are compared.

### No agent session after pull

If the developer pulls but doesn't start an agent session, `.cx/conflicts.json` sits there until the next session. It doesn't block anything — the conflicts are just deferred.

### Hooks not installed

If post-merge hooks aren't installed, `cx conflicts detect` never runs automatically. However, `cx index rebuild` (which runs lazily on the next command) also triggers conflict detection. The conflicts are caught, just not immediately.

---

## Command Reference

| Command | Purpose | Called by |
|---------|---------|----------|
| `cx conflicts detect` | Compare new entities against index, write .cx/conflicts.json | post-merge hook (after cx index rebuild) |
| `cx conflicts list` | Print current unresolved conflicts (reads .cx/conflicts.json) | Developer or agent |
| `cx conflicts clear` | Delete .cx/conflicts.json without resolving | Developer (manual override) |
