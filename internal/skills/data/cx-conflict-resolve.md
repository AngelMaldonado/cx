---
name: cx-conflict-resolve
description: Resolve conflicts between memory files, specs, and implementation. Activate when contradictions are detected or the developer notices conflicting guidance.
---

# Skill: cx-conflict-resolve

## Description
Resolve conflicts between memory files, specs, and implementation. Detects contradictions and guides the developer through resolution.

## Triggers
- Doctor reports conflicting memories
- Developer notices contradictory guidance
- Two specs disagree on an approach

## Steps
1. Identify the conflicting sources
2. Present both sides to the developer with context
3. Guide resolution: deprecate one, merge, or create a new decision
4. Update affected files to reflect the resolution

## Rules
- Never silently resolve conflicts — always involve the developer
- Preserve the deprecated version in archive
- Create a decision memory documenting the resolution
- Memory sync conflicts (same entity ID, different content in local DB vs docs/memory/) are NOT handled by this skill — use `cx memory pull` and `cx doctor` instead
- This skill handles semantic conflicts: contradictory specs, overlapping changes, conflicting design decisions
