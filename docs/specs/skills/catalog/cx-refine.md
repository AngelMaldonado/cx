# Skill: cx-refine

## Description
Teaches the Master how to iteratively improve a masterfile during brainstorming. Refinement is a skill-guided activity — there is no binary command. The agent reads the current masterfile, identifies gaps or weak areas, and edits it directly based on conversation with the developer.

## Triggers
- Developer asks to improve or iterate on a masterfile ("let's refine this", "what's missing?", "sharpen the vision")
- Developer adds new context that should be incorporated into the masterfile
- Agent identifies open questions in the masterfile that should be resolved

## Steps
1. Read the current masterfile from `docs/masterfiles/<name>.md`.
2. Assess the masterfile's completeness:
   - **Problem section**: Is the problem clearly defined? Are the pain points specific?
   - **Vision section**: Is the target state concrete enough to decompose into tasks?
   - **Open Questions**: Are there unresolved questions blocking progress?
   - **Constraints**: Are technical and business constraints captured?
3. Present your assessment to the developer. Highlight the weakest section and suggest what to improve.
4. Based on the developer's input, edit the masterfile directly. Keep sections organized and avoid duplicating content between sections.
5. After each edit, re-read the masterfile and check if the Open Questions list has changed. Resolved questions should be moved to the relevant section (e.g., a resolved constraint moves to Constraints).
6. When the masterfile feels complete — problem is clear, vision is actionable, no blocking open questions, constraints are known — suggest decomposing with `cx decompose <name>`.

## Rules
- Never edit a masterfile without the developer's input — refinement is collaborative, not autonomous
- Never remove content from a masterfile without explaining why
- Keep the masterfile under 500 lines — if it's growing beyond that, it's likely ready to decompose
- Do not create new binary commands or files — refinement happens entirely through direct file editing
- If the developer adds information that sounds like a memory (discovery, decision), suggest saving it separately with cx-memory in addition to adding it to the masterfile
