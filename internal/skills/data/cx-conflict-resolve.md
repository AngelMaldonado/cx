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
