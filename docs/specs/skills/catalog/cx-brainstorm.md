# Skill: cx-brainstorm

## Description
Teaches the Master how to create and decompose masterfiles for brainstorming. A masterfile is a freeform document in `docs/masterfiles/` where ideas are explored before being structured into a formal change. This skill covers creation (`cx brainstorm`) and transformation into a change (`cx decompose`).

## Triggers
- Developer wants to explore a new idea ("let's brainstorm", "I have an idea", "what if we...")
- Developer wants to turn a masterfile into a structured change ("let's turn this into a real change", "decompose this")
- Developer references an existing masterfile by name

## Steps

### Creating a masterfile
1. Run: `cx brainstorm <name>` where `<name>` is a kebab-case slug (e.g., `realtime-notifications`).
2. The binary creates `docs/masterfiles/<name>.md` with a template containing sections: Problem, Vision, Open Questions, Constraints, and Notes.
3. Work with the developer to fill in the sections. Don't rush — brainstorming is exploratory. Ask clarifying questions. Challenge assumptions.
4. The masterfile is a living document. Edit it directly as the conversation progresses. There is no binary command for refinement — use the cx-refine skill for iterative improvement.

### Decomposing a masterfile into a change
5. When the developer is ready to structure the idea, run: `cx decompose <name>`
6. The binary:
   - Reads `docs/masterfiles/<name>.md`
   - Creates `docs/changes/<name>/` with proposal.md, design.md, and tasks.md
   - Pre-fills proposal.md from the masterfile's Problem and Vision sections
   - Archives the masterfile to `docs/archive/<date>-masterfile-<name>.md`
7. Review the generated proposal.md with the developer. It will need refinement — the auto-fill is a starting point.
8. Fill in design.md (technical approach) and tasks.md (Linear issue references) using the cx-change skill.

## Rules
- Masterfile names must be kebab-case, max 40 characters, no special characters
- Never decompose a masterfile without the developer's explicit approval — brainstorming should feel open-ended, not rushed
- After decompose, the masterfile is archived. Do not edit archived masterfiles
- If a masterfile with the same name already exists, `cx brainstorm` will fail — choose a different name or work with the existing one
- Masterfiles are NOT indexed by `cx search` or included in context priming — they are working documents, not knowledge
