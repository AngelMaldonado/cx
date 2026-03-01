# Skill: cx-conflict-resolve

## Description
Conflict resolution skill for the Conflict-resolver. Spawned by the Primer when `.cx/conflicts.json` exists at session start. The Conflict-resolver loads conflicting entity pairs, reasons about whether they represent genuine semantic conflicts, and interviews the developer to resolve each one.

## Triggers
- Primer detects `.cx/conflicts.json` exists during cx-prime steps
- Developer explicitly asks to resolve conflicts ("check for conflicts", "any conflicts from the pull?")

## Steps
1. Read `.cx/conflicts.json` to get the list of conflict candidates.
2. For each candidate pair:
   a. Load the full content of both entities using `cx context --load memory <id>`.
   b. Read both entities completely. Assess whether this is a genuine conflict:
      - **Genuine conflict**: The two entities make contradictory claims or decisions about the same topic. They cannot both be true or both be followed simultaneously.
      - **Not a conflict**: The entities cover related topics but don't contradict each other. They add complementary information. One is more specific than the other but consistent.
      - **Already resolved**: One entity already deprecates the other, or one is already deprecated.
   c. If not a genuine conflict, skip it silently — do not bother the developer with false positives.
   d. If a genuine conflict, present it to the developer.

3. **Interview the developer about how to resolve the conflict. Use the AskUserQuestion tool.** Present:
   - A clear summary of both conflicting entities (title, author, date, key content)
   - Why they conflict (your assessment)
   - Resolution options:
     - Keep entity A, deprecate entity B
     - Keep entity B, deprecate entity A
     - Both are valid in different contexts (dismiss — not a real conflict)
     - Write a new entity that reconciles both

4. Based on the developer's answer:
   - **Keep A, deprecate B**: Run `cx memory decide` (or `cx memory save` for observations) with `--deprecates <B-slug>` to create a new entity that replaces B.
   - **Keep B, deprecate A**: Run the appropriate command with `--deprecates <A-slug>`.
   - **Both valid**: Take no action for this pair.
   - **Reconcile**: Work with the developer to draft a new entity that captures the reconciled view, deprecating whichever old entities it replaces.

5. After all conflicts are processed, delete `.cx/conflicts.json`.
6. Return a summary to the Primer: how many conflicts were found, how many resolved, how many dismissed.

## Rules
- **Always interview the developer for genuine conflicts — never resolve autonomously.** The whole point is human judgment on semantic disagreements.
- Use the AskUserQuestion tool for every genuine conflict — do not use regular prompts
- Skip false positives silently. Only present conflicts where the entities genuinely contradict each other.
- If a conflict involves a deprecated entity, skip it — deprecated entities are already excluded from context
- After resolution, always delete .cx/conflicts.json — even if all conflicts were dismissed
- Do not modify existing entity files. Resolution always works by creating NEW entities with `deprecates` pointing to the old ones
- If the developer dismisses all conflicts ("none of these are real conflicts"), respect that and clean up
